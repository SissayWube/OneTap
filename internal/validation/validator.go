package validation

import (
	"regexp"
	"strings"

	"github.com/onetap/salary-advance-loan-service/internal/models"
)

// accountNumberPattern matches numeric account numbers (9–16 digits).
// The actual data uses 13-digit identifiers but the range accommodates variation.
var accountNumberPattern = regexp.MustCompile(`^\d{9,16}$`)

func ValidateAccountNumberFormat(accountNo string) bool {
	return accountNumberPattern.MatchString(accountNo)
}

func NormalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func ValidateNameMatch(name1, name2 string) bool {
	return NormalizeName(name1) == NormalizeName(name2)
}

func NormalizeCustomerRecord(record models.CustomerRecord) models.CustomerRecord {
	return models.CustomerRecord{
		AccountNo:   strings.TrimSpace(record.AccountNo),
		Name:        NormalizeName(record.Name),
		AccountType: strings.TrimSpace(record.AccountType),
		Balance:     record.Balance,
	}
}
