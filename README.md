# Salary Advance Loan Service

A secure, high-performance Go-based REST API for processing salary advance loan applications. The service provides JWT authentication, customer data validation, transaction processing, and creditworthiness rating calculation.

## Features

- **JWT-based Authentication**: Secure user authentication with role-based access control (admin/uploader roles)
- **Rate-Limited Login**: Protection against brute force attacks (5 attempts per 15 minutes)
- **Customer Data Validation**: Validates customer records against canonical data with detailed error logging
- **Batch Processing**: Processes customer records in batches with intentional error injection for QA demonstration
- **Transaction Mapping**: Associates transaction history with customer accounts
- **Synthetic Transaction Generation**: Creates realistic transaction data for customers without history
- **Rating Calculation**: Computes creditworthiness scores (1-10 scale) using weighted multi-factor algorithm
- **Thread-Safe Storage**: In-memory data structures with RWMutex for concurrent access
- **Comprehensive Security**: Input validation, security headers, request timeouts, and audit logging

## Technology Stack

- **Language**: Go 1.21+
- **Web Framework**: Gin
- **Authentication**: JWT (golang-jwt/jwt)
- **Password Hashing**: bcrypt
- **Testing**: gopter (property-based testing), Go standard testing
- **Logging**: Structured JSON logging

## Prerequisites

- Go 1.21 or higher
- Git

## Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd salary-advance-loan-service
```

2. Install dependencies:
```bash
go mod download
```

3. Set up environment variables:
```bash
cp .env.example .env
# Edit .env and set JWT_SECRET to a secure random string
```

4. Prepare data files:
```bash
mkdir -p data
# Place customers.json, transactions.json in the data/ directory
```

## Configuration

All configuration is done through environment variables. See `.env.example` for all available options:

### Server Configuration
- `SERVER_PORT`: HTTP server port (default: 8080)
- `SERVER_READ_TIMEOUT`: Read timeout duration (default: 30s)
- `SERVER_WRITE_TIMEOUT`: Write timeout duration (default: 30s)
- `SERVER_MAX_REQUEST_SIZE`: Maximum request body size in bytes (default: 10MB)

### Authentication Configuration
- `JWT_SECRET`: Secret key for JWT signing (required, no default)
- `JWT_EXPIRATION`: JWT token expiration duration (default: 24h)
- `BCRYPT_COST`: Bcrypt hashing cost factor (default: 10, minimum: 10)

### Rate Limiting Configuration
- `RATE_LIMIT_MAX_ATTEMPTS`: Maximum failed login attempts (default: 5)
- `RATE_LIMIT_TIME_WINDOW`: Time window for rate limiting (default: 15m)
- `RATE_LIMIT_BLOCK_DURATION`: Block duration after exceeding limit (default: 15m)

### Logging Configuration
- `LOG_LEVEL`: Logging level (default: info, options: debug, info, warn, error)
- `LOG_FORMAT`: Log format (default: json)

### File Paths Configuration
- `CUSTOMERS_FILE`: Path to canonical customers JSON file (default: data/customers.json)
- `TRANSACTIONS_FILE`: Path to transactions JSON file (default: data/transactions.json)
- `SAMPLE_CUSTOMERS_FILE`: Path to sample customers JSON file (default: data/sample_customers.json)

## Running the Service

### Development
```bash
# Set JWT_SECRET environment variable
export JWT_SECRET="your-secret-key-here"

# Run the server
go run cmd/server/main.go
```

### Production
```bash
# Build the binary
go build -o server cmd/server/main.go

# Run the server
JWT_SECRET="your-production-secret" ./server
```

The server will start on port 8080 (or the port specified in `SERVER_PORT`).

## API Documentation

### Authentication

#### POST /auth/login
Authenticates a user and returns a JWT token.

**Request:**
```json
{
  "username": "admin",
  "password": "admin123"
}
```

**Response (200 OK):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "admin-001",
    "username": "admin",
    "role": "admin"
  },
  "expires_at": "2024-01-02T15:04:05Z"
}
```

**Error Responses:**
- `400 Bad Request`: Invalid request body
- `401 Unauthorized`: Invalid credentials
- `429 Too Many Requests`: Rate limit exceeded

**Default Test Users:**
- Admin: `admin` / `admin123`
- Uploader: `uploader` / `uploader123`

### Customer Validation

#### POST /customers/validate
Validates records from the configured sample file (`SAMPLE_CUSTOMERS_FILE`) in batches.

**Authentication**: Required (admin or uploader role)

**Request:** No request body required.

**Response (200 OK):**
```json
{
  "total_records": 50,
  "total_batches": 5,
  "passed_records": 48,
  "failed_records": 2,
  "batch_results": [...],
  "validation_results": [...]
}
```

### Transaction Endpoints

#### GET /customers/:accountNo/transactions
Retrieves all transactions for a customer account.

**Authentication**: Required (admin or uploader role)

**Response (200 OK):**
```json
{
  "account_no": "ACC-12345",
  "transactions": [...],
  "stats": {
    "total_count": 15,
    "total_volume": 45000.00,
    "date_range_days": 120,
    "balance_variance": 2500.50,
    "has_synthetic": false
  }
}
```

#### GET /customers/:accountNo/rating
Calculates and returns the creditworthiness rating for a customer.

**Authentication**: Required (admin or uploader role)

**Response (200 OK):**
```json
{
  "account_no": "ACC-12345",
  "rating": 7.8,
  "breakdown": {
    "transaction_count_score": 8.5,
    "transaction_volume_score": 7.2,
    "duration_score": 8.0,
    "stability_score": 7.5,
    "total_score": 7.8,
    "is_capped": false
  },
  "calculated_at": "2024-01-20T14:30:00Z"
}
```

### Validation Logs

#### GET /validation/logs
Retrieves validation logs with optional filtering.

**Authentication**: Required (admin role only)

**Query Parameters:**
- `batch_number` (optional): Filter by batch number
- `verified` (optional): Filter by verification status (true/false)
- `limit` (optional, default: 50): Maximum number of results
- `offset` (optional, default: 0): Pagination offset

**Response (200 OK):**
```json
{
  "logs": [...],
  "total": 50,
  "limit": 50,
  "offset": 0
}
```

### Health Check

#### GET /health
Returns the health status of the service.

**Authentication**: Not required

**Response (200 OK):**
```json
{
  "status": "healthy",
  "time": "2024-01-20T14:30:00Z"
}
```

## Validation Logic

The validation service validates customer records against canonical data:

### Validation Rules
1. **Account Number Format**: Must match the canonical format (e.g., "ACC-XXXXX")
2. **Name Matching**: Case-insensitive, whitespace-trimmed comparison against canonical data

### Error Types
- `invalid_account_format`: Account number doesn't match expected format
- `name_mismatch`: Customer name doesn't match canonical name for account
- `account_not_found`: Account number not in canonical list

### Batch Processing
- Records are processed in batches of 10
- Intentional errors are injected for QA demonstration:
  - Batch 2: Invalid account format at record index 12
  - Batch 4: Name mismatch at record index 35

## Rating Formula

Customer creditworthiness ratings are calculated on a 1-10 scale using a weighted algorithm:

### Component Scores (0-10 scale)
- **Transaction Count Score** (30% weight): `min(10, (count / 50) × 10)`
- **Transaction Volume Score** (30% weight): `min(10, (volume / 100000) × 10)`
- **Duration Score** (25% weight): `min(10, (days / 365) × 10)`
- **Stability Score** (15% weight): `10 - min(10, (variance / 10000) × 10)`

### Final Rating
```
Rating = (0.30 × CountScore) + (0.30 × VolumeScore) + 
         (0.25 × DurationScore) + (0.15 × StabilityScore)
```

### Special Rules
- Minimum rating: 1.0
- Maximum rating: 10.0
- Ratings are rounded to one decimal place
- Customers with fewer than 3 transactions are capped at 5.0

## Security Measures

### Authentication & Authorization
- JWT-based authentication with 24-hour token expiration
- Role-based access control (admin, uploader)
- Bcrypt password hashing with minimum cost factor of 10

### Rate Limiting
- 5 failed login attempts per 15 minutes per user
- Automatic account blocking with retry-after information

### Security Headers
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Strict-Transport-Security: max-age=31536000`
- `Content-Security-Policy: default-src 'self'`

### Input Validation
- Request body size limited to 10MB
- 30-second request timeout
- Input sanitization and validation

### Audit Logging
- All API requests logged with timestamp, method, path, user, and response status
- All authentication attempts logged
- All authorization failures logged
- Structured JSON logging format with correlation IDs

## Testing

### Run All Tests
```bash
go test ./...
```

### Run Tests with Coverage
```bash
go test ./... -cover
```

### Run Specific Test Package
```bash
# Unit tests
go test ./internal/auth/...
go test ./internal/rating/...

# Property-based tests
go test ./test/property/...
```

### Test Coverage
- Overall coverage: 80%+
- Unit tests: 47+ test functions
- Property-based tests: 17+ properties with 100 iterations each
- Integration tests: API endpoint testing

### Property-Based Tests
The service includes comprehensive property-based tests using gopter:
- Password hashing round-trip
- JWT token round-trip with role preservation
- Rate limiting blocks after threshold
- Customer validation correctness
- Batch division correctness
- Transaction mapping correctness
- Synthetic transaction constraints
- Balance invariant after synthetic transactions
- Rating range invariant
- Rating monotonicity
- Rating breakdown completeness
- Rating cap for insufficient data
- Authorization invariants

## Performance

### Performance Targets
- Validation of 50 records: < 2 seconds
- Rating calculation: < 500 milliseconds
- Transaction retrieval: < 300 milliseconds
- Concurrent requests: 100+ simultaneous requests
- Memory usage: < 512MB under normal load

### Benchmarks
```bash
go test -bench=. ./...
```

## Project Structure

```
salary-advance-loan-service/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
├── internal/
│   ├── api/
│   │   ├── errors/                 # Error types
│   │   ├── handlers/               # HTTP handlers
│   │   ├── middleware/             # Middleware
│   │   └── router.go               # Route definitions
│   ├── auth/                       # Authentication service
│   ├── config/                     # Configuration management
│   ├── loader/                     # File loaders (JSON/CSV)
│   ├── models/                     # Data models
│   ├── ratelimit/                  # Rate limiting
│   ├── rating/                     # Rating calculation
│   ├── storage/                    # In-memory storage
│   ├── transaction/                # Transaction processing
│   └── validation/                 # Customer validation
├── test/
│   └── property/                   # Property-based tests
├── data/                           # Data files
├── .env.example                    # Environment variables template
├── go.mod                          # Go module definition
└── README.md                       # This file
```

## Development

### Code Style
- Follow idiomatic Go conventions
- Use `gofmt` for code formatting
- Run `go vet` for static analysis
