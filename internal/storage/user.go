package storage

import (
	"errors"
	"sync"

	"github.com/onetap/salary-advance-loan-service/internal/models"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
)

// UserStore defines the interface for user data operations
type UserStore interface {
	GetUser(username string) (*models.User, error)
	CreateUser(user *models.User) error
	ListUsers() ([]*models.User, error)
}

// InMemoryUserStore implements UserStore with thread-safe in-memory storage
type InMemoryUserStore struct {
	mu    sync.RWMutex
	users map[string]*models.User // keyed by username
}

// NewInMemoryUserStore creates a new in-memory user store
func NewInMemoryUserStore() *InMemoryUserStore {
	return &InMemoryUserStore{
		users: make(map[string]*models.User),
	}
}

// GetUser retrieves a user by username
func (s *InMemoryUserStore) GetUser(username string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[username]
	if !exists {
		return nil, ErrUserNotFound
	}

	return user, nil
}

// CreateUser adds a new user to the store
func (s *InMemoryUserStore) CreateUser(user *models.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[user.Username]; exists {
		return ErrUserAlreadyExists
	}

	s.users[user.Username] = user
	return nil
}

// ListUsers returns all users in the store
func (s *InMemoryUserStore) ListUsers() ([]*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]*models.User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}

	return users, nil
}
