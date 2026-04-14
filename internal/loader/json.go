package loader

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/onetap/salary-advance-loan-service/internal/models"
)

// rawCustomerJSON represents the JSON structure from the customers file
type rawCustomerJSON struct {
	AccountNo       string  `json:"accountNo"`
	AccountNumber   string  `json:"account_number"`
	CustomerName    string  `json:"customerName"`
	Name            string  `json:"name"`
	ProductName     string  `json:"productName"`
	AccountType     string  `json:"account_type"`
	CustomerBalance float64 `json:"customerBalance"`
	Balance         float64 `json:"balance"`
}

// rawSampleCustomerJSON supports both snake_case and camelCase fields.
type rawSampleCustomerJSON struct {
	AccountNo          string   `json:"accountNo"`
	AccountNoSnake     string   `json:"account_no"`
	AccountNumber      string   `json:"account_number"`
	CustomerName       string   `json:"customerName"`
	Name               string   `json:"name"`
	ProductName        string   `json:"productName"`
	AccountType        string   `json:"account_type"`
	CustomerBalance    *float64 `json:"customerBalance"`
	CustomerBalanceAlt *float64 `json:"balance"`
}

// rawTransactionJSON represents the JSON structure from the transactions file
type rawTransactionJSON struct {
	ID                  string  `json:"id"`
	FromAccount         string  `json:"fromAccount"`
	ToAccount           *string `json:"toAccount"`
	Amount              string  `json:"amount"`
	TransactionType     string  `json:"transactionType"`
	TransactionDate     string  `json:"transactionDate"`
	Remark              *string `json:"remark"`
	Reference           string  `json:"reference"`
	ThirdPartyReference *string `json:"thirdPartyReference"`
	InstitutionID       *string `json:"institutionId"`
	ClearedBalance      *string `json:"clearedBalance"`
	BillerID            *string `json:"billerId"`
}

// LoadCustomersFromJSON loads customer records from a JSON file
func LoadCustomersFromJSON(filepath string) ([]models.CustomerRecord, error) {
	// Read file
	data, err := os.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", filepath)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse JSON
	var rawCustomers []rawCustomerJSON
	if err := json.Unmarshal(data, &rawCustomers); err != nil {
		return nil, fmt.Errorf("malformed JSON: %w", err)
	}

	// Validate and convert to CustomerRecord
	customers := make([]models.CustomerRecord, 0, len(rawCustomers))
	var missingFields []string

	for i, raw := range rawCustomers {
		accountNo := strings.TrimSpace(raw.AccountNo)
		if accountNo == "" {
			accountNo = strings.TrimSpace(raw.AccountNumber)
		}
		name := strings.TrimSpace(raw.CustomerName)
		if name == "" {
			name = strings.TrimSpace(raw.Name)
		}
		accountType := strings.TrimSpace(raw.ProductName)
		if accountType == "" {
			accountType = strings.TrimSpace(raw.AccountType)
		}
		balance := raw.CustomerBalance
		if balance == 0 {
			balance = raw.Balance
		}

		// Validate required fields
		if accountNo == "" {
			missingFields = append(missingFields, fmt.Sprintf("record %d: missing account_no", i))
		}
		if name == "" {
			missingFields = append(missingFields, fmt.Sprintf("record %d: missing name", i))
		}

		// Convert to CustomerRecord
		customer := models.CustomerRecord{
			AccountNo:   accountNo,
			Name:        name,
			AccountType: accountType,
			Balance:     balance,
		}
		customers = append(customers, customer)
	}

	// Return validation errors if any required fields are missing
	if len(missingFields) > 0 {
		return nil, fmt.Errorf("validation error - missing required fields: %s", strings.Join(missingFields, "; "))
	}

	return customers, nil
}

// LoadTransactionsFromJSON loads transaction records from a JSON file
func LoadTransactionsFromJSON(filepath string) ([]models.Transaction, error) {
	// Read file
	data, err := os.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", filepath)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse JSON
	var rawTransactions []rawTransactionJSON
	if err := json.Unmarshal(data, &rawTransactions); err != nil {
		return nil, fmt.Errorf("malformed JSON: %w", err)
	}

	// Validate and convert to Transaction
	transactions := make([]models.Transaction, 0, len(rawTransactions))
	var missingFields []string

	for i, raw := range rawTransactions {
		// Validate required fields
		if strings.TrimSpace(raw.ID) == "" {
			missingFields = append(missingFields, fmt.Sprintf("record %d: missing id", i))
		}
		if strings.TrimSpace(raw.FromAccount) == "" {
			missingFields = append(missingFields, fmt.Sprintf("record %d: missing from_account", i))
		}
		if strings.TrimSpace(raw.Amount) == "" {
			missingFields = append(missingFields, fmt.Sprintf("record %d: missing amount", i))
		}
		if strings.TrimSpace(raw.TransactionType) == "" {
			missingFields = append(missingFields, fmt.Sprintf("record %d: missing type", i))
		}
		if strings.TrimSpace(raw.TransactionDate) == "" {
			missingFields = append(missingFields, fmt.Sprintf("record %d: missing date", i))
		}

		// Parse amount (it's a string in the JSON)
		var amount float64
		if _, err := fmt.Sscanf(raw.Amount, "%f", &amount); err != nil {
			return nil, fmt.Errorf("record %d: invalid amount format: %s", i, raw.Amount)
		}

		// Parse date (Unix timestamp in milliseconds as string)
		var timestampMs int64
		if _, err := fmt.Sscanf(raw.TransactionDate, "%d", &timestampMs); err != nil {
			return nil, fmt.Errorf("record %d: invalid date format: %s", i, raw.TransactionDate)
		}

		// Convert to Transaction
		transaction := models.Transaction{
			ID:          raw.ID,
			FromAccount: raw.FromAccount,
			Amount:      amount,
			Type:        raw.TransactionType,
			Date:        timeFromMillis(timestampMs),
			IsSynthetic: false,
		}

		// Set optional fields
		if raw.ToAccount != nil {
			transaction.ToAccount = *raw.ToAccount
		}
		if raw.Remark != nil {
			transaction.Description = *raw.Remark
		}

		transactions = append(transactions, transaction)
	}

	// Return validation errors if any required fields are missing
	if len(missingFields) > 0 {
		return nil, fmt.Errorf("validation error - missing required fields: %s", strings.Join(missingFields, "; "))
	}

	return transactions, nil
}

// LoadSampleCustomersFromJSON loads sample customer records from a JSON file.
func LoadSampleCustomersFromJSON(filepath string) ([]models.CustomerRecord, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", filepath)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var rawCustomers []rawSampleCustomerJSON
	if err := json.Unmarshal(data, &rawCustomers); err != nil {
		return nil, fmt.Errorf("malformed JSON: %w", err)
	}

	customers := make([]models.CustomerRecord, 0, len(rawCustomers))
	var missingFields []string

	for i, raw := range rawCustomers {
		accountNo := strings.TrimSpace(raw.AccountNo)
		if accountNo == "" {
			accountNo = strings.TrimSpace(raw.AccountNoSnake)
		}
		if accountNo == "" {
			accountNo = strings.TrimSpace(raw.AccountNumber)
		}

		name := strings.TrimSpace(raw.CustomerName)
		if name == "" {
			name = strings.TrimSpace(raw.Name)
		}

		accountType := strings.TrimSpace(raw.ProductName)
		if accountType == "" {
			accountType = strings.TrimSpace(raw.AccountType)
		}

		balance := 0.0
		if raw.CustomerBalance != nil {
			balance = *raw.CustomerBalance
		} else if raw.CustomerBalanceAlt != nil {
			balance = *raw.CustomerBalanceAlt
		}

		if accountNo == "" {
			missingFields = append(missingFields, fmt.Sprintf("record %d: missing account number", i))
		}
		if name == "" {
			missingFields = append(missingFields, fmt.Sprintf("record %d: missing name", i))
		}

		customers = append(customers, models.CustomerRecord{
			AccountNo:   accountNo,
			Name:        name,
			AccountType: accountType,
			Balance:     balance,
		})
	}

	if len(missingFields) > 0 {
		return nil, fmt.Errorf("validation error - missing required fields: %s", strings.Join(missingFields, "; "))
	}

	return customers, nil
}

// timeFromMillis converts Unix timestamp in milliseconds to time.Time
func timeFromMillis(ms int64) time.Time {
	return time.Unix(ms/1000, (ms%1000)*1000000)
}
