package storage

import (
	"errors"
	"sync"

	"github.com/onetap/salary-advance-loan-service/internal/models"
)

var (
	ErrTransactionNotFound = errors.New("transaction not found")
)

// TransactionStore defines the interface for transaction data operations
type TransactionStore interface {
	GetTransactionsByAccount(accountNo string) ([]models.Transaction, error)
	CreateTransaction(tx *models.Transaction) error
	ListTransactions() ([]models.Transaction, error)
}

// InMemoryTransactionStore implements TransactionStore with thread-safe in-memory storage
type InMemoryTransactionStore struct {
	mu           sync.RWMutex
	transactions []models.Transaction // all transactions
	accountIndex map[string][]int     // maps account number to transaction indices
}

// NewInMemoryTransactionStore creates a new in-memory transaction store
func NewInMemoryTransactionStore() *InMemoryTransactionStore {
	return &InMemoryTransactionStore{
		transactions: make([]models.Transaction, 0),
		accountIndex: make(map[string][]int),
	}
}

// GetTransactionsByAccount retrieves all transactions for a specific account
func (s *InMemoryTransactionStore) GetTransactionsByAccount(accountNo string) ([]models.Transaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	indices, exists := s.accountIndex[accountNo]
	if !exists {
		// Return empty slice instead of error for accounts with no transactions
		return []models.Transaction{}, nil
	}

	// Create a copy of transactions to prevent external modifications
	result := make([]models.Transaction, 0, len(indices))
	for _, idx := range indices {
		if idx < len(s.transactions) {
			result = append(result, s.transactions[idx])
		}
	}

	return result, nil
}

// CreateTransaction adds a new transaction to the store
func (s *InMemoryTransactionStore) CreateTransaction(tx *models.Transaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Add transaction to the slice
	idx := len(s.transactions)
	s.transactions = append(s.transactions, *tx)

	// Update the account index for fromAccount
	s.accountIndex[tx.FromAccount] = append(s.accountIndex[tx.FromAccount], idx)

	return nil
}

// ListTransactions returns all transactions in the store
func (s *InMemoryTransactionStore) ListTransactions() ([]models.Transaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a copy of all transactions to prevent external modifications
	result := make([]models.Transaction, len(s.transactions))
	copy(result, s.transactions)

	return result, nil
}
