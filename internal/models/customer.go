package models

import "time"

// Customer represents a customer account in the system
type Customer struct {
	AccountNo   string    `json:"account_no"`
	Name        string    `json:"name"`
	AccountType string    `json:"account_type"`
	Balance     float64   `json:"balance"`
	Verified    bool      `json:"verified"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CustomerRecord represents a customer record from CSV/JSON input
type CustomerRecord struct {
	AccountNo   string  `json:"account_no" csv:"account_no"`
	Name        string  `json:"name" csv:"name"`
	AccountType string  `json:"account_type" csv:"account_type"`
	Balance     float64 `json:"balance" csv:"balance"`
}
