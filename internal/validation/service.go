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

func (s *ValidationService) ProcessSampleCustomers(ctx context.Context, records []models.CustomerRecord) BatchProcessingResult {
	const batchSize = 10
	total := len(records)
	numBatches := (total + batchSize - 1) / batchSize

	records = injectIntentionalErrors(records)

	var allResults []ValidationResult
	var batchResults []BatchResult
	passed, failed := 0, 0

	for b := 0; b < numBatches; b++ {
		start := b * batchSize
		end := start + batchSize
		if end > total {
			end = total
		}
		batch := records[start:end]

		results := s.ValidateBatch(ctx, batch)
		for i := range results {
			results[i].RecordIndex = start + i
		}

		for i, r := range results {
			s.validationStore.CreateValidationLog(&models.ValidationLog{
				ID:               uuid.New().String(),
				RecordIndex:      r.RecordIndex,
				BatchNumber:      b + 1,
				Verified:         r.Verified,
				Errors:           r.Errors,
				ErrorDetails:     r.ErrorDetails,
				OriginalRecord:   batch[i],
				NormalizedRecord: r.NormalizedRecord,
				CreatedAt:        time.Now(),
			})
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

		br := s.batchResult(b+1, results)
		batchResults = append(batchResults, br)
		allResults = append(allResults, results...)
		passed += br.PassCount
		failed += br.FailCount
	}

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
