package models

import "time"

// User represents a system user with authentication credentials
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // Never serialized to JSON
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Role represents user role type
type Role string

const (
	RoleAdmin    Role = "admin"
	RoleUploader Role = "uploader"
)
