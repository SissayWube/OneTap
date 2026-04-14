package models

import "time"

// Transaction represents a financial transaction
type Transaction struct {
	ID          string    `json:"id"`
	FromAccount string    `json:"from_account"`
	ToAccount   string    `json:"to_account"`
	Amount      float64   `json:"amount"`
	Type        string    `json:"type"`
	Date        time.Time `json:"date"`
	Description string    `json:"description"`
	IsSynthetic bool      `json:"is_synthetic"`
	CreatedAt   time.Time `json:"created_at"`
}

// TransactionType represents the type of transaction
type TransactionType string

const (
	TransactionTypeDebit  TransactionType = "debit"
	TransactionTypeCredit TransactionType = "credit"
)
