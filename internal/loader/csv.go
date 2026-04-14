package loader

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/onetap/salary-advance-loan-service/internal/models"
)

// LoadCustomersFromCSV loads customer records from a CSV file
func LoadCustomersFromCSV(filepath string) ([]models.CustomerRecord, error) {
	// Open file
	file, err := os.Open(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", filepath)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // Don't validate field count automatically

	// Read header
	header, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("empty CSV file")
		}
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	// Validate header
	expectedHeader := []string{"account_no", "name", "account_type", "balance"}
	if !reflect.DeepEqual(header, expectedHeader) {
		return nil, fmt.Errorf("invalid header: expected %v, got %v", expectedHeader, header)
	}

	var records []models.CustomerRecord
	lineNum := 1 // Line 1 is the header, data starts at line 2

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading line %d: %w", lineNum+1, err)
		}

		lineNum++

		// Validate field count
		if len(row) != 4 {
			return nil, fmt.Errorf("line %d: expected 4 fields, got %d", lineNum, len(row))
		}

		// Parse balance field as float64
		balance, err := strconv.ParseFloat(row[3], 64)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid balance format: %w", lineNum, err)
		}

		// Create customer record
		record := models.CustomerRecord{
			AccountNo:   row[0],
			Name:        row[1],
			AccountType: row[2],
			Balance:     balance,
		}

		records = append(records, record)
	}

	return records, nil
}

// LoadLegacySampleCustomersFromCSV reads sample files with headers customerName,accountNo.
func LoadLegacySampleCustomersFromCSV(filepath string) ([]models.CustomerRecord, error) {
	file, err := os.Open(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", filepath)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	header, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("empty CSV file")
		}
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	if len(header) != 2 {
		return nil, fmt.Errorf("invalid sample CSV header")
	}
	if strings.TrimSpace(header[0]) != "customerName" || strings.TrimSpace(header[1]) != "accountNo" {
		return nil, fmt.Errorf("invalid sample CSV header: expected customerName,accountNo")
	}

	records := make([]models.CustomerRecord, 0, 50)
	lineNum := 1
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading line %d: %w", lineNum+1, err)
		}
		lineNum++
		if len(row) != 2 {
			return nil, fmt.Errorf("line %d: expected 2 fields, got %d", lineNum, len(row))
		}

		records = append(records, models.CustomerRecord{
			Name:      strings.TrimSpace(row[0]),
			AccountNo: strings.TrimSpace(row[1]),
		})
	}

	return records, nil
}
