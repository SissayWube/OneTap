package models

import "time"

// CustomerRating represents a customer's creditworthiness rating
type CustomerRating struct {
	AccountNo    string          `json:"account_no"`
	Rating       float64         `json:"rating"`
	Breakdown    RatingBreakdown `json:"breakdown"`
	CalculatedAt time.Time       `json:"calculated_at"`
}

// RatingBreakdown provides detailed breakdown of rating calculation
type RatingBreakdown struct {
	TransactionCountScore  float64 `json:"transaction_count_score"`
	TransactionVolumeScore float64 `json:"transaction_volume_score"`
	DurationScore          float64 `json:"duration_score"`
	StabilityScore         float64 `json:"stability_score"`
	TotalScore             float64 `json:"total_score"`
	IsCapped               bool    `json:"is_capped"`
	CapReason              string  `json:"cap_reason,omitempty"`
}
