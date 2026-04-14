package storage

import (
	"errors"
	"sync"

	"github.com/onetap/salary-advance-loan-service/internal/models"
)

var (
	ErrCustomerNotFound      = errors.New("customer not found")
	ErrCustomerAlreadyExists = errors.New("customer already exists")
)

// CustomerStore defines the interface for customer data operations
type CustomerStore interface {
	GetCustomer(accountNo string) (*models.Customer, error)
	CreateCustomer(customer *models.Customer) error
	UpdateCustomer(customer *models.Customer) error
	ListCustomers() ([]*models.Customer, error)
	GetCanonicalCustomers() (map[string]*models.Customer, error)
}

// InMemoryCustomerStore implements CustomerStore with thread-safe in-memory storage
type InMemoryCustomerStore struct {
	mu        sync.RWMutex
	customers map[string]*models.Customer // keyed by account number
}

// NewInMemoryCustomerStore creates a new in-memory customer store
func NewInMemoryCustomerStore() *InMemoryCustomerStore {
	return &InMemoryCustomerStore{
		customers: make(map[string]*models.Customer),
	}
}

// GetCustomer retrieves a customer by account number
func (s *InMemoryCustomerStore) GetCustomer(accountNo string) (*models.Customer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	customer, exists := s.customers[accountNo]
	if !exists {
		return nil, ErrCustomerNotFound
	}

	return customer, nil
}

// CreateCustomer adds a new customer to the store
func (s *InMemoryCustomerStore) CreateCustomer(customer *models.Customer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.customers[customer.AccountNo]; exists {
		return ErrCustomerAlreadyExists
	}

	s.customers[customer.AccountNo] = customer
	return nil
}

// UpdateCustomer updates an existing customer in the store
func (s *InMemoryCustomerStore) UpdateCustomer(customer *models.Customer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.customers[customer.AccountNo]; !exists {
		return ErrCustomerNotFound
	}

	s.customers[customer.AccountNo] = customer
	return nil
}

// ListCustomers returns all customers in the store
func (s *InMemoryCustomerStore) ListCustomers() ([]*models.Customer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	customers := make([]*models.Customer, 0, len(s.customers))
	for _, customer := range s.customers {
		customers = append(customers, customer)
	}

	return customers, nil
}

// GetCanonicalCustomers returns a map of all customers keyed by account number
func (s *InMemoryCustomerStore) GetCanonicalCustomers() (map[string]*models.Customer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a copy of the map to prevent external modifications
	canonical := make(map[string]*models.Customer, len(s.customers))
	for accountNo, customer := range s.customers {
		canonical[accountNo] = customer
	}

	return canonical, nil
}
