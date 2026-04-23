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

// The four component score functions below normalise each metric to a 0–10 scale
// before the weighted sum is applied in CalculateRating.

func CalculateTransactionCountScore(count int) float64 {
	return math.Min(10.0, (float64(count)/50.0)*10.0)
}

func CalculateTransactionVolumeScore(volume float64) float64 {
	return math.Min(10.0, (volume/100000.0)*10.0)
}

func CalculateDurationScore(days int) float64 {
	return math.Min(10.0, (float64(days)/365.0)*10.0)
}

func CalculateStabilityScore(variance float64) float64 {
	return 10.0 - math.Min(10.0, (variance/10000.0)*10.0)
}

func (s *Service) CalculateRating(ctx context.Context, accountNo string) (models.CustomerRating, error) {
	// Verify the customer exists before attempting to build a rating.
	if _, err := s.customerStore.GetCustomer(accountNo); err != nil {
		return models.CustomerRating{}, ErrCustomerNotFound
	}

	// Fetch transaction metrics used to derive the rating components.
	stats, err := s.transactionService.CalculateTransactionStats(ctx, accountNo)
	if err != nil {
		return models.CustomerRating{}, err
	}

	// Normalize each transaction metric to a 0–10 score.
	countScore := CalculateTransactionCountScore(stats.TotalCount)
	volumeScore := CalculateTransactionVolumeScore(stats.TotalVolume)
	durationScore := CalculateDurationScore(stats.DateRangeDays)
	stabilityScore := CalculateStabilityScore(stats.BalanceVariance)

	// Weighted sum of all component scores produces the overall rating.
	total := (0.30 * countScore) + (0.30 * volumeScore) + (0.25 * durationScore) + (0.15 * stabilityScore)
	if total < 1.0 {
		// Ensure the rating floor does not fall below 1.
		total = 1.0
	}

	isCapped := false
	capReason := ""
	// Apply a cap for customers with too few transactions.
	if stats.TotalCount < 3 && total > 5.0 {
		total = 5.0
		isCapped = true
		capReason = "fewer than 3 transactions"
	}

	// Round the final rating to one decimal place for presentation.
	rating := math.Round(total*10) / 10

	return models.CustomerRating{
		AccountNo: accountNo,
		Rating:    rating,
		Breakdown: models.RatingBreakdown{
			TransactionCountScore:  math.Round(countScore*10) / 10,
			TransactionVolumeScore: math.Round(volumeScore*10) / 10,
			DurationScore:          math.Round(durationScore*10) / 10,
			StabilityScore:         math.Round(stabilityScore*10) / 10,
			TotalScore:             rating,
			IsCapped:               isCapped,
			CapReason:              capReason,
		},
		CalculatedAt: time.Now(),
	}, nil
}

func (s *Service) GetRatingBreakdown(ctx context.Context, accountNo string) (models.RatingBreakdown, error) {
	r, err := s.CalculateRating(ctx, accountNo)
	if err != nil {
		return models.RatingBreakdown{}, err
	}
	return r.Breakdown, nil
}
