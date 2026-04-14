package storage

import (
	"sync"

	"github.com/onetap/salary-advance-loan-service/internal/models"
)

// ValidationStore defines the interface for validation log data operations
type ValidationStore interface {
	CreateValidationLog(log *models.ValidationLog) error
	GetValidationLogs(filters models.ValidationFilters) ([]models.ValidationLog, error)
	GetValidationLogsByBatch(batchNo int) ([]models.ValidationLog, error)
	CountValidationLogs(filters models.ValidationFilters) (int, error)
	SaveProcessedRecord(record *models.ProcessedRecord) error
	ListProcessedRecords() ([]models.ProcessedRecord, error)
}

// InMemoryValidationStore implements ValidationStore with thread-safe in-memory storage
type InMemoryValidationStore struct {
	mu               sync.RWMutex
	logs             []models.ValidationLog   // all validation logs
	processedRecords []models.ProcessedRecord // all processed sample records
}

// NewInMemoryValidationStore creates a new in-memory validation store
func NewInMemoryValidationStore() *InMemoryValidationStore {
	return &InMemoryValidationStore{
		logs:             make([]models.ValidationLog, 0),
		processedRecords: make([]models.ProcessedRecord, 0),
	}
}

// CreateValidationLog adds a new validation log to the store
func (s *InMemoryValidationStore) CreateValidationLog(log *models.ValidationLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logs = append(s.logs, *log)
	return nil
}

// GetValidationLogs retrieves validation logs with optional filtering
func (s *InMemoryValidationStore) GetValidationLogs(filters models.ValidationFilters) ([]models.ValidationLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Filter logs based on criteria
	var filtered []models.ValidationLog
	for _, log := range s.logs {
		// Apply batch number filter if specified
		if filters.BatchNumber != nil && log.BatchNumber != *filters.BatchNumber {
			continue
		}

		// Apply verified filter if specified
		if filters.Verified != nil && log.Verified != *filters.Verified {
			continue
		}

		filtered = append(filtered, log)
	}

	// Apply offset
	if filters.Offset > 0 {
		if filters.Offset >= len(filtered) {
			return []models.ValidationLog{}, nil
		}
		filtered = filtered[filters.Offset:]
	}

	// Apply limit
	if filters.Limit > 0 && filters.Limit < len(filtered) {
		filtered = filtered[:filters.Limit]
	}

	return filtered, nil
}

// GetValidationLogsByBatch retrieves all validation logs for a specific batch number
func (s *InMemoryValidationStore) GetValidationLogsByBatch(batchNo int) ([]models.ValidationLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []models.ValidationLog
	for _, log := range s.logs {
		if log.BatchNumber == batchNo {
			result = append(result, log)
		}
	}

	return result, nil
}

// CountValidationLogs returns the total number of logs matching filters (before pagination).
func (s *InMemoryValidationStore) CountValidationLogs(filters models.ValidationFilters) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, log := range s.logs {
		if filters.BatchNumber != nil && log.BatchNumber != *filters.BatchNumber {
			continue
		}
		if filters.Verified != nil && log.Verified != *filters.Verified {
			continue
		}
		count++
	}

	return count, nil
}

// SaveProcessedRecord stores every processed record (including unverified ones).
func (s *InMemoryValidationStore) SaveProcessedRecord(record *models.ProcessedRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.processedRecords = append(s.processedRecords, *record)
	return nil
}

// ListProcessedRecords returns all processed records.
func (s *InMemoryValidationStore) ListProcessedRecords() ([]models.ProcessedRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]models.ProcessedRecord, len(s.processedRecords))
	copy(result, s.processedRecords)
	return result, nil
}
