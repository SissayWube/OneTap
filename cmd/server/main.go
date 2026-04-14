package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/onetap/salary-advance-loan-service/internal/api"
	"github.com/onetap/salary-advance-loan-service/internal/auth"
	"github.com/onetap/salary-advance-loan-service/internal/config"
	"github.com/onetap/salary-advance-loan-service/internal/loader"
	"github.com/onetap/salary-advance-loan-service/internal/models"
	"github.com/onetap/salary-advance-loan-service/internal/ratelimit"
	"github.com/onetap/salary-advance-loan-service/internal/rating"
	"github.com/onetap/salary-advance-loan-service/internal/storage"
	"github.com/onetap/salary-advance-loan-service/internal/transaction"
	"github.com/onetap/salary-advance-loan-service/internal/validation"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	userStore := storage.NewInMemoryUserStore()
	customerStore := storage.NewInMemoryCustomerStore()
	transactionStore := storage.NewInMemoryTransactionStore()
	validationStore := storage.NewInMemoryValidationStore()

	if err := loadDataFiles(cfg, customerStore, transactionStore, userStore); err != nil {
		log.Fatalf("Failed to load data files: %v", err)
	}

	authService := auth.NewService(userStore, cfg.Auth.JWTSecret, cfg.Auth.JWTExpiration, cfg.Auth.BcryptCost)
	rateLimiter := ratelimit.NewLoginAttemptTracker(cfg.RateLimit.MaxAttempts, cfg.RateLimit.TimeWindow, cfg.RateLimit.BlockDuration)
	validationService := validation.NewValidationService(customerStore, validationStore, cfg.Files.SampleCustomersPath)

	transactionService, err := transaction.NewService(transactionStore, customerStore)
	if err != nil {
		log.Fatalf("Failed to create transaction service: %v", err)
	}

	ratingService := rating.NewService(transactionService, customerStore)

	router := api.SetupRouter(api.RouterConfig{
		AuthService:        authService,
		RateLimiter:        rateLimiter,
		ValidationService:  validationService,
		TransactionService: transactionService,
		RatingService:      ratingService,
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		log.Printf("Server listening on port %d", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
}

func loadDataFiles(cfg *config.Config, customerStore storage.CustomerStore, transactionStore storage.TransactionStore, userStore storage.UserStore) error {
	customers, err := loader.LoadCustomersFromJSON(cfg.Files.CustomersPath)
	if err != nil {
		return fmt.Errorf("failed to load customers: %w", err)
	}
	for _, r := range customers {
		if err := customerStore.CreateCustomer(&models.Customer{
			AccountNo:   r.AccountNo,
			Name:        r.Name,
			AccountType: r.AccountType,
			Balance:     r.Balance,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}); err != nil {
			return fmt.Errorf("failed to store customer: %w", err)
		}
	}
	log.Printf("Loaded %d customers", len(customers))

	txs, err := loader.LoadTransactionsFromJSON(cfg.Files.TransactionsPath)
	if err != nil {
		return fmt.Errorf("failed to load transactions: %w", err)
	}
	for i := range txs {
		if err := transactionStore.CreateTransaction(&txs[i]); err != nil {
			return fmt.Errorf("failed to store transaction: %w", err)
		}
	}
	log.Printf("Loaded %d transactions", len(txs))

	enableDefaultUsers, err := getEnvAsBool("ENABLE_DEFAULT_USERS", true)
	if err != nil {
		return fmt.Errorf("invalid ENABLE_DEFAULT_USERS value: %w", err)
	}

	if enableDefaultUsers {
		if err := createDefaultUsers(cfg, userStore); err != nil {
			return err
		}
	}

	return nil
}

func createDefaultUsers(cfg *config.Config, userStore storage.UserStore) error {
	type defaultUser struct {
		id       string
		username string
		password string
		role     string
	}

	users := []defaultUser{
		{"admin-001", "admin", "admin123", "admin"},
		{"uploader-001", "uploader", "uploader123", "uploader"},
	}

	for _, u := range users {
		hash, err := auth.HashPassword(u.password, cfg.Auth.BcryptCost)
		if err != nil {
			return fmt.Errorf("failed to hash password for %s: %w", u.username, err)
		}
		if err := userStore.CreateUser(&models.User{
			ID:           u.id,
			Username:     u.username,
			PasswordHash: hash,
			Role:         u.role,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}); err != nil {
			return fmt.Errorf("failed to create user %s: %w", u.username, err)
		}
	}

	log.Println("Default users created (admin, uploader)")
	return nil
}

func getEnvAsBool(key string, defaultValue bool) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}
	return strconv.ParseBool(value)
}
