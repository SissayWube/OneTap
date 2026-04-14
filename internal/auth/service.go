package auth

import (
	"context"
	"errors"
	"time"

	"github.com/onetap/salary-advance-loan-service/internal/storage"
)

var ErrInvalidCredentials = errors.New("invalid username or password")

// AuthService handles login and token validation.
type AuthService interface {
	Login(ctx context.Context, username, password string) (string, error)
	ValidateToken(ctx context.Context, token string) (*UserClaims, error)
}

type Service struct {
	userStore     storage.UserStore
	jwtSecret     string
	jwtExpiration time.Duration
	bcryptCost    int
}

func NewService(userStore storage.UserStore, jwtSecret string, jwtExpiration time.Duration, bcryptCost int) *Service {
	return &Service{
		userStore:     userStore,
		jwtSecret:     jwtSecret,
		jwtExpiration: jwtExpiration,
		bcryptCost:    bcryptCost,
	}
}

func (s *Service) Login(ctx context.Context, username, password string) (string, error) {
	user, err := s.userStore.GetUser(username)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return "", ErrInvalidCredentials
		}
		return "", err
	}

	if err = ComparePassword(user.PasswordHash, password); err != nil {
		return "", ErrInvalidCredentials
	}

	return GenerateToken(user.ID, user.Username, user.Role, s.jwtSecret, s.jwtExpiration)
}

func (s *Service) ValidateToken(ctx context.Context, token string) (*UserClaims, error) {
	return ValidateToken(token, s.jwtSecret)
}
