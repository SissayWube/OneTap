package transaction

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/onetap/salary-advance-loan-service/internal/models"
	"github.com/onetap/salary-advance-loan-service/internal/storage"
)

var ErrCustomerNotFound = errors.New("customer not found")

// TransactionService retrieves and computes statistics for customer transactions.
// Synthetic transactions are generated automatically for customers with no history.
type TransactionService interface {
	GetCustomerTransactions(ctx context.Context, accountNo string) ([]models.Transaction, error)
	CalculateTransactionStats(ctx context.Context, accountNo string) (TransactionStats, error)
}

type TransactionStats struct {
	TotalCount       int       `json:"total_count"`
	TotalVolume      float64   `json:"total_volume"`
	DateRangeDays    int       `json:"date_range_days"`
	FirstTransaction time.Time `json:"first_transaction"`
	LastTransaction  time.Time `json:"last_transaction"`
	BalanceVariance  float64   `json:"balance_variance"`
	HasSynthetic     bool      `json:"has_synthetic"`
}

type Service struct {
	transactionStore storage.TransactionStore
	customerStore    storage.CustomerStore
	mu               sync.RWMutex
	accountIndex     map[string][]models.Transaction
}

func NewService(transactionStore storage.TransactionStore, customerStore storage.CustomerStore) (*Service, error) {
	s := &Service{
		transactionStore: transactionStore,
		customerStore:    customerStore,
		accountIndex:     make(map[string][]models.Transaction),
	}
	if err := s.indexTransactions(); err != nil {
		return nil, fmt.Errorf("failed to index transactions: %w", err)
	}
	return s, nil
}

func (s *Service) indexTransactions() error {
	txs, err := s.transactionStore.ListTransactions()
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, tx := range txs {
		s.accountIndex[tx.FromAccount] = append(s.accountIndex[tx.FromAccount], tx)
	}
	for acc := range s.accountIndex {
		sort.Slice(s.accountIndex[acc], func(i, j int) bool {
			return s.accountIndex[acc][i].Date.Before(s.accountIndex[acc][j].Date)
		})
	}
	return nil
}

// GetCustomerTransactions returns a customer's transactions, generating synthetic ones if none exist.
func (s *Service) GetCustomerTransactions(ctx context.Context, accountNo string) ([]models.Transaction, error) {
	customer, err := s.customerStore.GetCustomer(accountNo)
	if err != nil {
		return nil, ErrCustomerNotFound
	}

	s.mu.RLock()
	txs, exists := s.accountIndex[accountNo]
	s.mu.RUnlock()

	if exists && len(txs) > 0 {
		result := make([]models.Transaction, len(txs))
		copy(result, txs)
		return result, nil
	}

	return s.generateSyntheticTransactions(*customer)
}

func (s *Service) CalculateTransactionStats(ctx context.Context, accountNo string) (TransactionStats, error) {
	txs, err := s.GetCustomerTransactions(ctx, accountNo)
	if err != nil {
		return TransactionStats{}, err
	}
	if len(txs) == 0 {
		return TransactionStats{}, nil
	}

	stats := TransactionStats{
		TotalCount:       len(txs),
		FirstTransaction: txs[0].Date,
		LastTransaction:  txs[len(txs)-1].Date,
	}
	for _, tx := range txs {
		stats.TotalVolume += tx.Amount
		if tx.IsSynthetic {
			stats.HasSynthetic = true
		}
	}
	stats.DateRangeDays = int(stats.LastTransaction.Sub(stats.FirstTransaction).Hours() / 24)
	stats.BalanceVariance = balanceVariance(txs)
	return stats, nil
}

// balanceVariance computes the population variance of the running balance across all transactions.
// A lower variance means more stable cash flow, which feeds into the stability score.
func balanceVariance(txs []models.Transaction) float64 {
	if len(txs) == 0 {
		return 0
	}
	balances := make([]float64, len(txs))
	running := 0.0
	for i, tx := range txs {
		if tx.Type == string(models.TransactionTypeDebit) {
			running -= tx.Amount
		} else {
			running += tx.Amount
		}
		balances[i] = running
	}
	sum := 0.0
	for _, b := range balances {
		sum += b
	}
	mean := sum / float64(len(balances))
	variance := 0.0
	for _, b := range balances {
		d := b - mean
		variance += d * d
	}
	return variance / float64(len(balances))
}

// generateSyntheticTransactions creates 3–10 plausible transactions for customers
// with no real history so the rating service always has data to work with.
// Amounts stay between 100–5000 and the running balance is kept non-negative.
func (s *Service) generateSyntheticTransactions(customer models.Customer) ([]models.Transaction, error) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	n := rng.Intn(8) + 3 // 3–10

	now := time.Now()
	dates := make([]time.Time, n)
	for i := range dates {
		dates[i] = now.AddDate(0, 0, -(rng.Intn(180) + 1))
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })

	txs := make([]models.Transaction, 0, n)
	balance := customer.Balance

	for i := 0; i < n; i++ {
		amount := float64(rng.Intn(4901) + 100)
		var txType string

		if i < n/2 || balance < amount {
			txType = string(models.TransactionTypeCredit)
			balance += amount
		} else {
			if balance-amount < 0 {
				amount = balance * 0.8
				if amount < 100 {
					txType = string(models.TransactionTypeCredit)
					amount = float64(rng.Intn(4901) + 100)
					balance += amount
				} else {
					txType = string(models.TransactionTypeDebit)
					balance -= amount
				}
			} else {
				txType = string(models.TransactionTypeDebit)
				balance -= amount
			}
		}

		txs = append(txs, models.Transaction{
			ID:          uuid.New().String(),
			FromAccount: customer.AccountNo,
			ToAccount:   fmt.Sprintf("SYNTHETIC-%s", uuid.New().String()[:8]),
			Amount:      math.Round(amount*100) / 100,
			Type:        txType,
			Date:        dates[i],
			Description: "Synthetic transaction",
			IsSynthetic: true,
			CreatedAt:   now,
		})
	}

	if balance < 0 {
		return nil, errors.New("synthetic transactions would create negative balance")
	}

	for i := range txs {
		if err := s.transactionStore.CreateTransaction(&txs[i]); err != nil {
			return nil, fmt.Errorf("failed to store synthetic transaction: %w", err)
		}
	}

	s.mu.Lock()
	s.accountIndex[customer.AccountNo] = txs
	s.mu.Unlock()

	return txs, nil
}
