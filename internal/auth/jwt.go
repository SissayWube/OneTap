package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken    = errors.New("invalid token")
	ErrTokenExpired    = errors.New("token has expired")
	ErrInvalidSecret   = errors.New("secret cannot be empty")
	ErrInvalidUserData = errors.New("user data cannot be empty")
)

// UserClaims holds the JWT payload fields alongside the standard registered claims.
type UserClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken signs a new HS256 JWT for the given user. Defaults to 24h expiry if expiration is zero.
func GenerateToken(userID, username, role, secret string, expiration time.Duration) (string, error) {
	if userID == "" || username == "" || role == "" {
		return "", ErrInvalidUserData
	}
	if secret == "" {
		return "", ErrInvalidSecret
	}
	if expiration == 0 {
		expiration = 24 * time.Hour
	}

	now := time.Now()
	claims := UserClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return signed, nil
}

// ValidateToken parses and verifies a JWT, returning the embedded claims.
// Returns ErrTokenExpired specifically so callers can distinguish expiry from other failures.
func ValidateToken(tokenString, secret string) (*UserClaims, error) {
	if strings.TrimSpace(tokenString) == "" {
		return nil, ErrInvalidToken
	}
	if secret == "" {
		return nil, ErrInvalidSecret
	}

	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
