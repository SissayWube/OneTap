package rating

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/onetap/salary-advance-loan-service/internal/models"
	"github.com/onetap/salary-advance-loan-service/internal/storage"
	"github.com/onetap/salary-advance-loan-service/internal/transaction"
)

var ErrCustomerNotFound = errors.New("customer not found")

// RatingService calculates a customer's creditworthiness on a 1–10 scale.
type RatingService interface {
	CalculateRating(ctx context.Context, accountNo string) (models.CustomerRating, error)
	GetRatingBreakdown(ctx context.Context, accountNo string) (models.RatingBreakdown, error)
}

type Service struct {
	transactionService transaction.TransactionService
	customerStore      storage.CustomerStore
}

func NewService(transactionService transaction.TransactionService, customerStore storage.CustomerStore) *Service {
	return &Service{
		transactionService: transactionService,
		customerStore:      customerStore,
	}
}

// The four component score functions below normalize each metric to a 0-10 scale
// before the weighted sum is applied in CalculateRating. Each function represents
// a different aspect of creditworthiness.

func CalculateTransactionCountScore(count int) float64 {
	return math.Min(10.0, (float64(count)/50.0)*10.0)
}

// CalculateTransactionVolumeScore measures financial capacity based on total transaction amount.
// Higher volumes indicate greater financial resources and borrowing capacity.
func CalculateTransactionVolumeScore(volume float64) float64 {
	return math.Min(10.0, (volume/100000.0)*10.0)
}

// CalculateDurationScore measures the length of transaction history.
// Longer histories provide more data points and indicate established financial behavior.

func CalculateDurationScore(days int) float64 {
	return math.Min(10.0, (float64(days)/365.0)*10.0)
}

// CalculateStabilityScore measures cash flow consistency based on balance variance.
// Lower variance indicates more stable, predictable financial behavior.
//
func CalculateStabilityScore(variance float64) float64 {
	return 10.0 - math.Min(10.0, (variance/10000.0)*10.0)
}

// CalculateRating computes a customer's creditworthiness score on a 1-10 scale.
// The rating is calculated using a weighted multi-factor algorithm that considers
// transaction activity, financial capacity, history length, and cash flow stability.
func (s *Service) CalculateRating(ctx context.Context, accountNo string) (models.CustomerRating, error) {
	// Verify customer exists in the system
	if _, err := s.customerStore.GetCustomer(accountNo); err != nil {
		return models.CustomerRating{}, ErrCustomerNotFound
	}

	// Retrieve transaction statistics (includes synthetic transactions if needed)
	stats, err := s.transactionService.CalculateTransactionStats(ctx, accountNo)
	if err != nil {
		return models.CustomerRating{}, err
	}

	// Calculate individual component scores (each on 0-10 scale)
	countScore := CalculateTransactionCountScore(stats.TotalCount)    // 30% weight
	volumeScore := CalculateTransactionVolumeScore(stats.TotalVolume) // 30% weight
	durationScore := CalculateDurationScore(stats.DateRangeDays)      // 25% weight
	stabilityScore := CalculateStabilityScore(stats.BalanceVariance)  // 15% weight

	// Apply weighted sum to get total score
	// Formula: (0.30 × Count) + (0.30 × Volume) + (0.25 × Duration) + (0.15 × Stability)
	total := (0.30 * countScore) + (0.30 * volumeScore) + (0.25 * durationScore) + (0.15 * stabilityScore)

	// Enforce minimum rating of 1.0 (prevents zero or negative ratings)
	if total < 1.0 {
		total = 1.0
	}

	// Apply cap for insufficient transaction data
	// Customers with fewer than 3 transactions are capped at 5.0 to prevent
	// unreliable ratings from limited data
	isCapped := false
	capReason := ""
	if stats.TotalCount < 3 && total > 5.0 {
		total = 5.0
		isCapped = true
		capReason = "fewer than 3 transactions"
	}

	// Round final rating to 1 decimal place for consistency
	rating := math.Round(total*10) / 10

	// Return complete rating with detailed breakdown
	return models.CustomerRating{
		AccountNo: accountNo,
		Rating:    rating,
		Breakdown: models.RatingBreakdown{
			TransactionCountScore:  math.Round(countScore*10) / 10,     // Rounded for display
			TransactionVolumeScore: math.Round(volumeScore*10) / 10,    // Rounded for display
			DurationScore:          math.Round(durationScore*10) / 10,  // Rounded for display
			StabilityScore:         math.Round(stabilityScore*10) / 10, // Rounded for display
			TotalScore:             rating,
			IsCapped:               isCapped,
			CapReason:              capReason,
		},
		CalculatedAt: time.Now(),
	}, nil
}

// GetRatingBreakdown retrieves just the rating breakdown without the full rating object.
// This is a convenience method that wraps CalculateRating and extracts the breakdown.
//
// Parameters:
//   - ctx: Request context for cancellation and timeout handling
//   - accountNo: Customer account number
//
// Returns:
//   - RatingBreakdown: Detailed score breakdown with component scores
//   - error: ErrCustomerNotFound if customer doesn't exist, or other errors
func (s *Service) GetRatingBreakdown(ctx context.Context, accountNo string) (models.RatingBreakdown, error) {
	r, err := s.CalculateRating(ctx, accountNo)
	if err != nil {
		return models.RatingBreakdown{}, err
	}
	return r.Breakdown, nil
}
