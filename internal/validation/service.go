package validation

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/onetap/salary-advance-loan-service/internal/loader"
	"github.com/onetap/salary-advance-loan-service/internal/models"
	"github.com/onetap/salary-advance-loan-service/internal/storage"
)

type ValidationService struct {
	customerStore   storage.CustomerStore
	validationStore storage.ValidationStore
	sampleFilePath  string
}

func NewValidationService(customerStore storage.CustomerStore, validationStore storage.ValidationStore, sampleFilePath string) *ValidationService {
	return &ValidationService{
		customerStore:   customerStore,
		validationStore: validationStore,
		sampleFilePath:  sampleFilePath,
	}
}

type ValidationResult struct {
	RecordIndex      int                    `json:"record_index"`
	Verified         bool                   `json:"verified"`
	Errors           []string               `json:"errors"`
	ErrorDetails     []string               `json:"error_details,omitempty"`
	NormalizedRecord *models.CustomerRecord `json:"normalized_record,omitempty"`
}

type BatchResult struct {
	BatchNumber int            `json:"batch_number"`
	RecordCount int            `json:"record_count"`
	PassCount   int            `json:"pass_count"`
	FailCount   int            `json:"fail_count"`
	ErrorTypes  map[string]int `json:"error_types"`
}

type BatchProcessingResult struct {
	TotalRecords      int                `json:"total_records"`
	TotalBatches      int                `json:"total_batches"`
	PassedRecords     int                `json:"passed_records"`
	FailedRecords     int                `json:"failed_records"`
	BatchResults      []BatchResult      `json:"batch_results"`
	ValidationResults []ValidationResult `json:"validation_results"`
}

func (s *ValidationService) ValidateCustomer(ctx context.Context, record models.CustomerRecord) ValidationResult {
	var errs []string
	verified := true

	normalized := NormalizeCustomerRecord(record)

	if !ValidateAccountNumberFormat(record.AccountNo) {
		errs = append(errs, "invalid_account_format")
		verified = false
	}

	canonical, err := s.customerStore.GetCanonicalCustomers()
	if err != nil {
		errs = append(errs, "failed_to_load_canonical_data")
		return ValidationResult{Verified: false, Errors: errs, ErrorDetails: errorDetails(errs)}
	}

	if canon, exists := canonical[record.AccountNo]; !exists {
		errs = append(errs, "account_not_found")
		verified = false
	} else if !ValidateNameMatch(record.Name, canon.Name) {
		errs = append(errs, "name_mismatch")
		verified = false
	}

	result := ValidationResult{
		Verified:     verified,
		Errors:       errs,
		ErrorDetails: errorDetails(errs),
	}
	if verified {
		result.NormalizedRecord = &normalized
	}
	return result
}

func (s *ValidationService) ValidateBatch(ctx context.Context, records []models.CustomerRecord) []ValidationResult {
	results := make([]ValidationResult, len(records))
	for i, record := range records {
		r := s.ValidateCustomer(ctx, record)
		r.RecordIndex = i
		results[i] = r
	}
	return results
}

// injectIntentionalErrors corrupts two records to demonstrate error detection:
// index 12 gets an invalid account format (batch 2), index 35 gets a wrong name (batch 4).
func injectIntentionalErrors(records []models.CustomerRecord) []models.CustomerRecord {
	out := make([]models.CustomerRecord, len(records))
	copy(out, records)
	if len(out) > 12 {
		out[12].AccountNo = "INVALID-ACCT"
	}
	if len(out) > 35 {
		out[35].Name = "WRONG NAME INJECTED"
	}
	return out
}

// ProcessSampleCustomers validates a list of customer records in batches and returns comprehensive results.
// This function is the core of the batch validation workflow, processing records in groups of 10 for
// better organization and progress tracking.
//
// Process Flow:
// 1. Inject intentional errors for QA demonstration (indices 12 and 35)
// 2. Divide records into batches of 10
// 3. Validate each batch against canonical customer data
// 4. Store validation logs and processed records for audit trail
// 5. Aggregate results across all batches
//
// Parameters:
//   - ctx: Request context for cancellation and timeout handling
//   - records: Slice of customer records to validate
//
// Returns:
//   - BatchProcessingResult containing:
//   - Summary statistics (total, passed, failed counts)
//   - Per-batch results with error type breakdown
//   - Individual validation results for each record
func (s *ValidationService) ProcessSampleCustomers(ctx context.Context, records []models.CustomerRecord) BatchProcessingResult {
	// Define batch size for processing records in groups
	const batchSize = 10
	total := len(records)

	// Calculate number of batches needed (rounds up for partial batches)
	// Example: 50 records / 10 = 5 batches, 55 records / 10 = 6 batches
	numBatches := (total + batchSize - 1) / batchSize

	// Inject intentional errors for QA demonstration:
	// - Index 12 (batch 2): Invalid account format
	// - Index 35 (batch 4): Name mismatch
	records = injectIntentionalErrors(records)

	// Initialize result collectors
	var allResults []ValidationResult // Individual validation results for each record
	var batchResults []BatchResult    // Summary results for each batch
	passed, failed := 0, 0            // Running totals across all batches

	// Process each batch sequentially
	for b := 0; b < numBatches; b++ {
		// Calculate batch boundaries
		start := b * batchSize
		end := start + batchSize

		// Handle last batch which may be smaller than batchSize
		if end > total {
			end = total
		}

		// Extract current batch slice
		batch := records[start:end]

		// Validate all records in the current batch
		results := s.ValidateBatch(ctx, batch)

		// Update record indices to reflect position in overall dataset
		// (ValidateBatch returns indices relative to batch, we need global indices)
		for i := range results {
			results[i].RecordIndex = start + i
		}

		// Store validation results in persistent storage for audit trail
		for i, r := range results {
			// Create detailed validation log entry
			s.validationStore.CreateValidationLog(&models.ValidationLog{
				ID:               uuid.New().String(),
				RecordIndex:      r.RecordIndex,
				BatchNumber:      b + 1, // Batch numbers are 1-indexed for user display
				Verified:         r.Verified,
				Errors:           r.Errors,
				ErrorDetails:     r.ErrorDetails,
				OriginalRecord:   batch[i],
				NormalizedRecord: r.NormalizedRecord,
				CreatedAt:        time.Now(),
			})

			// Create simplified processed record entry
			s.validationStore.SaveProcessedRecord(&models.ProcessedRecord{
				ID:          uuid.New().String(),
				RecordIndex: r.RecordIndex,
				BatchNumber: b + 1,
				Record:      batch[i],
				Verified:    r.Verified,
				Errors:      r.Errors,
				CreatedAt:   time.Now(),
			})
		}

		// Generate batch summary with error type breakdown
		br := s.batchResult(b+1, results)
		batchResults = append(batchResults, br)

		// Accumulate all individual results
		allResults = append(allResults, results...)

		// Update running totals
		passed += br.PassCount
		failed += br.FailCount
	}

	// Return comprehensive results including:
	// - Overall statistics
	// - Per-batch summaries
	// - Individual validation results
	return BatchProcessingResult{
		TotalRecords:      total,
		TotalBatches:      numBatches,
		PassedRecords:     passed,
		FailedRecords:     failed,
		BatchResults:      batchResults,
		ValidationResults: allResults,
	}
}

func (s *ValidationService) batchResult(batchNumber int, results []ValidationResult) BatchResult {
	errTypes := make(map[string]int)
	pass, fail := 0, 0
	for _, r := range results {
		if r.Verified {
			pass++
		} else {
			fail++
			for _, e := range r.Errors {
				errTypes[e]++
			}
		}
	}
	return BatchResult{
		BatchNumber: batchNumber,
		RecordCount: len(results),
		PassCount:   pass,
		FailCount:   fail,
		ErrorTypes:  errTypes,
	}
}

func (s *ValidationService) GetValidationLogs(ctx context.Context, filters models.ValidationFilters) ([]models.ValidationLog, error) {
	return s.validationStore.GetValidationLogs(filters)
}

func (s *ValidationService) CountValidationLogs(ctx context.Context, filters models.ValidationFilters) (int, error) {
	return s.validationStore.CountValidationLogs(filters)
}

func (s *ValidationService) GetProcessedRecords(ctx context.Context) ([]models.ProcessedRecord, error) {
	return s.validationStore.ListProcessedRecords()
}

func (s *ValidationService) ProcessConfiguredSampleCustomers(ctx context.Context) (BatchProcessingResult, error) {
	if s.sampleFilePath == "" {
		return BatchProcessingResult{}, fmt.Errorf("sample customers file path is not configured")
	}

	var records []models.CustomerRecord
	var err error

	switch filepath.Ext(s.sampleFilePath) {
	case ".json":
		records, err = loader.LoadSampleCustomersFromJSON(s.sampleFilePath)
	default:
		records, err = loader.LoadLegacySampleCustomersFromCSV(s.sampleFilePath)
	}
	if err != nil {
		return BatchProcessingResult{}, err
	}
	if len(records) != 50 {
		return BatchProcessingResult{}, fmt.Errorf("sample file must contain exactly 50 records, got %d", len(records))
	}

	return s.ProcessSampleCustomers(ctx, records), nil
}

func errorDetails(codes []string) []string {
	if len(codes) == 0 {
		return nil
	}
	details := make([]string, 0, len(codes))
	for _, code := range codes {
		switch code {
		case "invalid_account_format":
			details = append(details, "account number format is invalid")
		case "name_mismatch":
			details = append(details, "name does not match canonical record")
		case "account_not_found":
			details = append(details, "account number not found in canonical list")
		case "failed_to_load_canonical_data":
			details = append(details, "failed to load canonical customer list")
		default:
			details = append(details, code)
		}
	}
	return details
}
