// backend/internal/adapters/out/payout/mock_provider.go

package payout

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"

	applicationport "narratives/internal/application/port"
	payoutdom "narratives/internal/domain/payoutAccount"
)

const (
	mockProviderName            = "mock"
	mockProviderAccountIDPrefix = "mock_payout_"
)

var (
	ErrInvalidUserID            = errors.New("payout mock provider: invalid userId")
	ErrInvalidBankCode          = errors.New("payout mock provider: invalid bankCode")
	ErrInvalidBankName          = errors.New("payout mock provider: invalid bankName")
	ErrInvalidBranchCode        = errors.New("payout mock provider: invalid branchCode")
	ErrInvalidBranchName        = errors.New("payout mock provider: invalid branchName")
	ErrInvalidAccountType       = errors.New("payout mock provider: invalid accountType")
	ErrInvalidAccountNumber     = errors.New("payout mock provider: invalid accountNumber")
	ErrInvalidAccountHolderName = errors.New("payout mock provider: invalid accountHolderName")
	ErrInvalidProviderAccountID = errors.New("payout mock provider: invalid providerAccountId")
	ErrGenerateProviderAccount  = errors.New("payout mock provider: failed to generate provider account id")
)

// MockPayoutAccountProvider is a development-only PayoutAccountProvider.
//
// It validates registration input and returns a mock provider account ID, but
// never performs an actual payout-provider registration or money movement.
//
// Security policy:
//   - the full AccountNumber is used only during Register
//   - the full AccountNumber is never logged, persisted, or returned
//   - only BankLast4 leaves the provider adapter
//
// Availability policy:
//   - successful registration returns StatusRegistered
//   - PayoutReady is always false because this provider cannot receive payouts
type MockPayoutAccountProvider struct{}

var _ applicationport.PayoutAccountProvider = (*MockPayoutAccountProvider)(nil)

func NewMockPayoutAccountProvider() *MockPayoutAccountProvider {
	return &MockPayoutAccountProvider{}
}

func (p *MockPayoutAccountProvider) Name() string {
	return mockProviderName
}

func (p *MockPayoutAccountProvider) Register(
	ctx context.Context,
	in applicationport.RegisterPayoutAccountInput,
) (*applicationport.RegisterPayoutAccountResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	userID := strings.TrimSpace(in.UserID)
	bankCode := strings.TrimSpace(in.BankCode)
	bankName := strings.TrimSpace(in.BankName)
	branchCode := strings.TrimSpace(in.BranchCode)
	branchName := strings.TrimSpace(in.BranchName)
	accountNumber := strings.TrimSpace(in.AccountNumber)
	accountHolderName := strings.TrimSpace(in.AccountHolderName)

	if userID == "" {
		return nil, ErrInvalidUserID
	}
	if !isFixedDigits(bankCode, 4) {
		return nil, ErrInvalidBankCode
	}
	if bankName == "" {
		return nil, ErrInvalidBankName
	}
	if !isFixedDigits(branchCode, 3) {
		return nil, ErrInvalidBranchCode
	}
	if branchName == "" {
		return nil, ErrInvalidBranchName
	}
	if !isValidAccountType(in.AccountType) {
		return nil, ErrInvalidAccountType
	}
	if !isFixedDigits(accountNumber, 7) {
		return nil, ErrInvalidAccountNumber
	}
	if accountHolderName == "" {
		return nil, ErrInvalidAccountHolderName
	}

	providerAccountID, err := newMockProviderAccountID()
	if err != nil {
		return nil, err
	}

	return &applicationport.RegisterPayoutAccountResult{
		ProviderAccountID: providerAccountID,
		Status:            payoutdom.StatusRegistered,
		PayoutReady:       false,
		BankLast4:         accountNumber[len(accountNumber)-4:],
	}, nil
}

func (p *MockPayoutAccountProvider) Get(
	ctx context.Context,
	providerAccountID string,
) (*applicationport.PayoutAccountProviderState, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	providerAccountID = strings.TrimSpace(providerAccountID)
	if providerAccountID == "" || !strings.HasPrefix(providerAccountID, mockProviderAccountIDPrefix) {
		return nil, ErrInvalidProviderAccountID
	}

	return &applicationport.PayoutAccountProviderState{
		Status:      payoutdom.StatusRegistered,
		PayoutReady: false,
	}, nil
}

func newMockProviderAccountID() (string, error) {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", ErrGenerateProviderAccount
	}

	return mockProviderAccountIDPrefix + hex.EncodeToString(randomBytes), nil
}

func isValidAccountType(accountType payoutdom.BankAccountType) bool {
	switch accountType {
	case payoutdom.BankAccountTypeOrdinary, payoutdom.BankAccountTypeCurrent:
		return true
	default:
		return false
	}
}

func isFixedDigits(value string, length int) bool {
	if len(value) != length {
		return false
	}

	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}
