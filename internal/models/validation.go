package models

import "time"

// ValidationLog represents a validation log entry for a customer record
type ValidationLog struct {
	ID               string          `json:"id"`
	RecordIndex      int             `json:"record_index"`
	BatchNumber      int             `json:"batch_number"`
	Verified         bool            `json:"verified"`
	Errors           []string        `json:"errors"`
	ErrorDetails     []string        `json:"error_details,omitempty"`
	OriginalRecord   CustomerRecord  `json:"original_record"`
	NormalizedRecord *CustomerRecord `json:"normalized_record,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
}

// ProcessedRecord stores every processed sample record, including failed ones.
type ProcessedRecord struct {
	ID          string         `json:"id"`
	RecordIndex int            `json:"record_index"`
	BatchNumber int            `json:"batch_number"`
	Record      CustomerRecord `json:"record"`
	Verified    bool           `json:"verified"`
	Errors      []string       `json:"errors"`
	CreatedAt   time.Time      `json:"created_at"`
}

// ValidationFilters represents filters for querying validation logs
type ValidationFilters struct {
	BatchNumber *int
	Verified    *bool
	Limit       int
	Offset      int
}
