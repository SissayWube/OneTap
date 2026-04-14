package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidPassword = errors.New("password cannot be empty")
	ErrInvalidCost     = errors.New("bcrypt cost must be at least 10")
)

// HashPassword hashes a plaintext password using bcrypt. Minimum cost of 10 is enforced.
func HashPassword(password string, cost int) (string, error) {
	if password == "" {
		return "", ErrInvalidPassword
	}
	if cost < 10 {
		return "", ErrInvalidCost
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashed), nil
}

// ComparePassword checks a plaintext password against a bcrypt hash.
func ComparePassword(hashedPassword, password string) error {
	if hashedPassword == "" || password == "" {
		return ErrInvalidPassword
	}
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return fmt.Errorf("password does not match: %w", err)
		}
		return fmt.Errorf("failed to compare password: %w", err)
	}
	return nil
}
