// backend/internal/domain/bankPayout/entity.go
package bankPayout

import (
	"errors"
	"strings"
	"time"

	payoutdom "narratives/internal/domain/payoutAccount"
)

// Status represents the lifecycle of one bank payout for one resale
// SalesReceivable.
//
// pending:
//   - the matching SalesReceivable is payout-eligible
//   - the immutable bank destination has been snapshotted
//   - no payout execution is currently in progress
//
// processing:
//   - a worker has claimed the payout and is executing the bank payout gateway
//
// paid:
//   - the payout gateway has completed successfully
//
// failed_retryable:
//   - payout execution failed but may be retried
//
// failed:
//   - payout execution failed permanently
type Status string

const (
	StatusPending         Status = "pending"
	StatusProcessing      Status = "processing"
	StatusPaid            Status = "paid"
	StatusFailedRetryable Status = "failed_retryable"
	StatusFailed          Status = "failed"
)

var AllowedStatuses = map[Status]struct{}{
	StatusPending:         {},
	StatusProcessing:      {},
	StatusPaid:            {},
	StatusFailedRetryable: {},
	StatusFailed:          {},
}

const CurrencyJPY = "JPY"

var DefaultStatus = StatusPending

var (
	ErrInvalidID                      = errors.New("bankPayout: invalid id")
	ErrInvalidSalesReceivableID       = errors.New("bankPayout: invalid salesReceivableId")
	ErrInvalidOrderID                 = errors.New("bankPayout: invalid orderId")
	ErrInvalidPaymentID               = errors.New("bankPayout: invalid paymentId")
	ErrPaymentOrderMismatch           = errors.New("bankPayout: paymentId does not match orderId")
	ErrInvalidOrderItemIndex          = errors.New("bankPayout: invalid orderItemIndex")
	ErrInvalidResaleID                = errors.New("bankPayout: invalid resaleId")
	ErrInvalidSellerUserID            = errors.New("bankPayout: invalid sellerUserId")
	ErrInvalidPayoutAccountID         = errors.New("bankPayout: invalid payoutAccountId")
	ErrInvalidAmount                  = errors.New("bankPayout: invalid amount")
	ErrInvalidCurrency                = errors.New("bankPayout: invalid currency")
	ErrInvalidStatus                  = errors.New("bankPayout: invalid status")
	ErrInvalidStatusTransition        = errors.New("bankPayout: invalid status transition")
	ErrInvalidProviderPayoutID        = errors.New("bankPayout: invalid providerPayoutId")
	ErrInvalidBankCode                = errors.New("bankPayout: invalid bankCode")
	ErrInvalidBankName                = errors.New("bankPayout: invalid bankName")
	ErrInvalidBranchCode              = errors.New("bankPayout: invalid branchCode")
	ErrInvalidBranchName              = errors.New("bankPayout: invalid branchName")
	ErrInvalidBankAccountType         = errors.New("bankPayout: invalid bank account type")
	ErrInvalidAccountNumberCiphertext = errors.New("bankPayout: invalid accountNumberCiphertext")
	ErrInvalidBankLast4               = errors.New("bankPayout: invalid bankLast4")
	ErrInvalidAccountHolderName       = errors.New("bankPayout: invalid accountHolderName")
	ErrInvalidErrorType               = errors.New("bankPayout: invalid errorType")
	ErrInvalidErrorCode               = errors.New("bankPayout: invalid errorCode")
	ErrInvalidErrorMsg                = errors.New("bankPayout: invalid errorMsg")
	ErrFailureReasonRequired          = errors.New("bankPayout: failure reason is required")
	ErrInvalidCreatedAt               = errors.New("bankPayout: invalid createdAt")
	ErrInvalidUpdatedAt               = errors.New("bankPayout: invalid updatedAt")
	ErrInvalidProcessingAt            = errors.New("bankPayout: invalid processingAt")
	ErrInvalidPaidAt                  = errors.New("bankPayout: invalid paidAt")
)

var (
	MaxIDLength                      = 512
	MaxSellerUserIDLength            = 128
	MaxPayoutAccountIDLength         = 128
	MaxProviderPayoutIDLength        = 512
	MaxBankNameLength                = 100
	MaxBranchNameLength              = 100
	MaxAccountNumberCiphertextLength = 8192
	MaxAccountHolderNameLength       = 128
	MaxErrorTypeLength               = 128
	MaxErrorCodeLength               = 256
	MaxErrorMsgLength                = 2000
)

// BankDestinationSnapshot is the immutable seller bank destination captured
// when a BankPayout is created.
//
// The plaintext account number must never be stored. AccountNumberCiphertext is
// copied from the selected PayoutAccount and remains encrypted at rest.
//
// This snapshot intentionally prevents a later PayoutAccount update from
// changing the destination of an already-created BankPayout.
type BankDestinationSnapshot struct {
	BankCode   string `json:"bankCode" firestore:"bankCode"`
	BankName   string `json:"bankName" firestore:"bankName"`
	BranchCode string `json:"branchCode" firestore:"branchCode"`
	BranchName string `json:"branchName" firestore:"branchName"`

	AccountType             payoutdom.BankAccountType `json:"accountType" firestore:"accountType"`
	AccountNumberCiphertext string                    `json:"-" firestore:"accountNumberCiphertext"`
	BankLast4               string                    `json:"bankLast4" firestore:"bankLast4"`
	AccountHolderName       string                    `json:"accountHolderName" firestore:"accountHolderName"`
}

func (b BankDestinationSnapshot) Validate() error {
	if !isFixedDigits(b.BankCode, 4) {
		return ErrInvalidBankCode
	}
	if b.BankName == "" ||
		(MaxBankNameLength > 0 && len([]rune(b.BankName)) > MaxBankNameLength) {
		return ErrInvalidBankName
	}
	if !isFixedDigits(b.BranchCode, 3) {
		return ErrInvalidBranchCode
	}
	if b.BranchName == "" ||
		(MaxBranchNameLength > 0 && len([]rune(b.BranchName)) > MaxBranchNameLength) {
		return ErrInvalidBranchName
	}

	switch b.AccountType {
	case payoutdom.BankAccountTypeOrdinary, payoutdom.BankAccountTypeCurrent:
	default:
		return ErrInvalidBankAccountType
	}

	if b.AccountNumberCiphertext == "" ||
		(MaxAccountNumberCiphertextLength > 0 && len(b.AccountNumberCiphertext) > MaxAccountNumberCiphertextLength) {
		return ErrInvalidAccountNumberCiphertext
	}
	if !isFixedDigits(b.BankLast4, 4) {
		return ErrInvalidBankLast4
	}
	if b.AccountHolderName == "" ||
		(MaxAccountHolderNameLength > 0 && len([]rune(b.AccountHolderName)) > MaxAccountHolderNameLength) {
		return ErrInvalidAccountHolderName
	}

	return nil
}

// BankPayout represents the payout instruction created after a resale
// fulfillment has completed and the matching SalesReceivable has become
// available.
//
// Current development policy:
//   - one SalesReceivable creates at most one BankPayout
//   - BankPayout ID is deterministic from SalesReceivableID
//   - one BankPayout pays exactly one resale Order item
//   - bank destination data is snapshotted at BankPayout creation
//   - plaintext bank account numbers are never persisted
//   - the application layer coordinates SalesReceivable available -> reserved
//     before payout execution and reserved -> paid after payout success
//
// ProviderPayoutID is gateway-specific. In development the fake gateway may
// return a deterministic fake payout ID. A future real bank gateway may store
// its transfer/instruction identifier in the same field.
type BankPayout struct {
	ID string `json:"id" firestore:"id"`

	SalesReceivableID string `json:"salesReceivableId" firestore:"salesReceivableId"`
	OrderID           string `json:"orderId" firestore:"orderId"`
	PaymentID         string `json:"paymentId" firestore:"paymentId"`
	OrderItemIndex    int    `json:"orderItemIndex" firestore:"orderItemIndex"`
	ResaleID          string `json:"resaleId" firestore:"resaleId"`

	SellerUserID    string `json:"sellerUserId" firestore:"sellerUserId"`
	PayoutAccountID string `json:"payoutAccountId" firestore:"payoutAccountId"`

	BankDestination BankDestinationSnapshot `json:"bankDestination" firestore:"bankDestination"`

	Amount   int    `json:"amount" firestore:"amount"`
	Currency string `json:"currency" firestore:"currency"`
	Status   Status `json:"status" firestore:"status"`

	ProviderPayoutID string `json:"providerPayoutId,omitempty" firestore:"providerPayoutId,omitempty"`

	ErrorType *string `json:"errorType,omitempty" firestore:"errorType,omitempty"`
	ErrorCode *string `json:"errorCode,omitempty" firestore:"errorCode,omitempty"`
	ErrorMsg  *string `json:"errorMsg,omitempty" firestore:"errorMsg,omitempty"`

	CreatedAt time.Time `json:"createdAt" firestore:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" firestore:"updatedAt"`

	ProcessingAt *time.Time `json:"processingAt,omitempty" firestore:"processingAt,omitempty"`
	PaidAt       *time.Time `json:"paidAt,omitempty" firestore:"paidAt,omitempty"`
}

// NewID creates the deterministic BankPayout document ID.
//
// A SalesReceivable may create at most one BankPayout, so the receivable ID is
// the complete idempotency scope for payout creation.
func NewID(salesReceivableID string) (string, error) {
	if !isValidIdentifier(salesReceivableID, MaxIDLength) {
		return "", ErrInvalidSalesReceivableID
	}

	return salesReceivableID + "_bank_payout", nil
}

func IsValidStatus(status Status) bool {
	if status == "" {
		return false
	}

	_, ok := AllowedStatuses[status]
	return ok
}

// New creates a pending BankPayout after the matching SalesReceivable has
// crossed the fulfillment boundary and become payout-eligible.
func New(
	id string,
	salesReceivableID string,
	orderID string,
	paymentID string,
	orderItemIndex int,
	resaleID string,
	sellerUserID string,
	payoutAccountID string,
	bankDestination BankDestinationSnapshot,
	amount int,
	currency string,
	createdAt time.Time,
) (BankPayout, error) {
	if err := bankDestination.Validate(); err != nil {
		return BankPayout{}, err
	}

	createdAt = createdAt.UTC()

	payout := BankPayout{
		ID: id,

		SalesReceivableID: salesReceivableID,
		OrderID:           orderID,
		PaymentID:         paymentID,
		OrderItemIndex:    orderItemIndex,
		ResaleID:          resaleID,

		SellerUserID:    sellerUserID,
		PayoutAccountID: payoutAccountID,

		BankDestination: bankDestination,

		Amount:   amount,
		Currency: currency,
		Status:   DefaultStatus,

		ProviderPayoutID: "",

		ErrorType: nil,
		ErrorCode: nil,
		ErrorMsg:  nil,

		CreatedAt: createdAt,
		UpdatedAt: createdAt,

		ProcessingAt: nil,
		PaidAt:       nil,
	}

	if err := payout.Validate(); err != nil {
		return BankPayout{}, err
	}

	return payout, nil
}

// StartProcessing claims the BankPayout for execution.
//
// A retryable failure may be claimed again. The application/repository layer is
// responsible for ensuring only one worker owns the active processing lease.
func (p *BankPayout) StartProcessing(now time.Time) error {
	if p == nil {
		return ErrInvalidStatusTransition
	}

	switch p.Status {
	case StatusPending, StatusFailedRetryable:
	default:
		return ErrInvalidStatusTransition
	}

	if now.IsZero() {
		return ErrInvalidProcessingAt
	}

	processingAt := now.UTC()
	if processingAt.Before(p.CreatedAt) {
		return ErrInvalidProcessingAt
	}

	p.Status = StatusProcessing
	p.ProviderPayoutID = ""
	p.ErrorType = nil
	p.ErrorCode = nil
	p.ErrorMsg = nil
	p.ProcessingAt = &processingAt
	p.UpdatedAt = processingAt

	return p.Validate()
}

// ReclaimProcessing refreshes a stale processing claim.
//
// The repository/application layer decides whether the existing claim is stale.
func (p *BankPayout) ReclaimProcessing(now time.Time) error {
	if p == nil || p.Status != StatusProcessing {
		return ErrInvalidStatusTransition
	}
	if now.IsZero() {
		return ErrInvalidProcessingAt
	}

	processingAt := now.UTC()
	if !processingAt.After(p.UpdatedAt) {
		return ErrInvalidProcessingAt
	}

	p.ErrorType = nil
	p.ErrorCode = nil
	p.ErrorMsg = nil
	p.ProcessingAt = &processingAt
	p.UpdatedAt = processingAt

	return p.Validate()
}

// MarkPaid records successful payout execution.
func (p *BankPayout) MarkPaid(providerPayoutID string, now time.Time) error {
	if p == nil || p.Status != StatusProcessing {
		return ErrInvalidStatusTransition
	}
	if !isValidIdentifier(providerPayoutID, MaxProviderPayoutIDLength) {
		return ErrInvalidProviderPayoutID
	}
	if now.IsZero() {
		return ErrInvalidPaidAt
	}

	paidAt := now.UTC()
	if p.ProcessingAt == nil || paidAt.Before(*p.ProcessingAt) {
		return ErrInvalidPaidAt
	}

	p.Status = StatusPaid
	p.ProviderPayoutID = providerPayoutID
	p.ErrorType = nil
	p.ErrorCode = nil
	p.ErrorMsg = nil
	p.PaidAt = &paidAt
	p.UpdatedAt = paidAt

	return p.Validate()
}

// MarkFailedRetryable records an execution failure that may be retried.
func (p *BankPayout) MarkFailedRetryable(
	errorType *string,
	errorCode *string,
	errorMsg *string,
	now time.Time,
) error {
	if p == nil || p.Status != StatusProcessing {
		return ErrInvalidStatusTransition
	}
	if err := validateFailureReason(errorType, errorCode, errorMsg); err != nil {
		return err
	}
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}

	updatedAt := now.UTC()
	if p.ProcessingAt == nil || updatedAt.Before(*p.ProcessingAt) {
		return ErrInvalidUpdatedAt
	}

	p.Status = StatusFailedRetryable
	p.ProviderPayoutID = ""
	p.ErrorType = cloneOptionalString(errorType)
	p.ErrorCode = cloneOptionalString(errorCode)
	p.ErrorMsg = cloneOptionalString(errorMsg)
	p.UpdatedAt = updatedAt

	return p.Validate()
}

// MarkFailed records a terminal execution failure.
func (p *BankPayout) MarkFailed(
	errorType *string,
	errorCode *string,
	errorMsg *string,
	now time.Time,
) error {
	if p == nil || p.Status != StatusProcessing {
		return ErrInvalidStatusTransition
	}
	if err := validateFailureReason(errorType, errorCode, errorMsg); err != nil {
		return err
	}
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}

	updatedAt := now.UTC()
	if p.ProcessingAt == nil || updatedAt.Before(*p.ProcessingAt) {
		return ErrInvalidUpdatedAt
	}

	p.Status = StatusFailed
	p.ProviderPayoutID = ""
	p.ErrorType = cloneOptionalString(errorType)
	p.ErrorCode = cloneOptionalString(errorCode)
	p.ErrorMsg = cloneOptionalString(errorMsg)
	p.UpdatedAt = updatedAt

	return p.Validate()
}

// Validate validates the persisted BankPayout state.
func (p BankPayout) Validate() error {
	if !isValidIdentifier(p.ID, MaxIDLength) {
		return ErrInvalidID
	}
	if !isValidIdentifier(p.SalesReceivableID, MaxIDLength) {
		return ErrInvalidSalesReceivableID
	}

	expectedID, err := NewID(p.SalesReceivableID)
	if err != nil {
		return err
	}
	if p.ID != expectedID {
		return ErrInvalidID
	}

	if !isValidIdentifier(p.OrderID, MaxIDLength) {
		return ErrInvalidOrderID
	}
	if !isValidIdentifier(p.PaymentID, MaxIDLength) {
		return ErrInvalidPaymentID
	}
	if p.PaymentID != p.OrderID {
		return ErrPaymentOrderMismatch
	}
	if p.OrderItemIndex < 0 {
		return ErrInvalidOrderItemIndex
	}
	if !isValidIdentifier(p.ResaleID, MaxIDLength) {
		return ErrInvalidResaleID
	}
	if !isValidIdentifier(p.SellerUserID, MaxSellerUserIDLength) {
		return ErrInvalidSellerUserID
	}
	if !isValidIdentifier(p.PayoutAccountID, MaxPayoutAccountIDLength) {
		return ErrInvalidPayoutAccountID
	}
	if err := p.BankDestination.Validate(); err != nil {
		return err
	}
	if p.Amount <= 0 {
		return ErrInvalidAmount
	}
	if p.Currency != CurrencyJPY {
		return ErrInvalidCurrency
	}
	if !IsValidStatus(p.Status) {
		return ErrInvalidStatus
	}
	if p.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if p.UpdatedAt.IsZero() || p.UpdatedAt.Before(p.CreatedAt) {
		return ErrInvalidUpdatedAt
	}

	if err := validateOptionalString(p.ErrorType, MaxErrorTypeLength, ErrInvalidErrorType); err != nil {
		return err
	}
	if err := validateOptionalString(p.ErrorCode, MaxErrorCodeLength, ErrInvalidErrorCode); err != nil {
		return err
	}
	if err := validateOptionalString(p.ErrorMsg, MaxErrorMsgLength, ErrInvalidErrorMsg); err != nil {
		return err
	}

	switch p.Status {
	case StatusPending:
		if p.ProviderPayoutID != "" ||
			p.ProcessingAt != nil ||
			p.PaidAt != nil ||
			hasFailureReason(p.ErrorType, p.ErrorCode, p.ErrorMsg) {
			return ErrInvalidStatus
		}

	case StatusProcessing:
		if p.ProviderPayoutID != "" ||
			p.ProcessingAt == nil ||
			p.ProcessingAt.IsZero() ||
			p.ProcessingAt.Before(p.CreatedAt) ||
			p.PaidAt != nil ||
			hasFailureReason(p.ErrorType, p.ErrorCode, p.ErrorMsg) {
			return ErrInvalidStatus
		}

	case StatusPaid:
		if !isValidIdentifier(p.ProviderPayoutID, MaxProviderPayoutIDLength) ||
			p.ProcessingAt == nil ||
			p.ProcessingAt.IsZero() ||
			p.PaidAt == nil ||
			p.PaidAt.IsZero() ||
			p.PaidAt.Before(*p.ProcessingAt) ||
			hasFailureReason(p.ErrorType, p.ErrorCode, p.ErrorMsg) {
			return ErrInvalidStatus
		}

	case StatusFailedRetryable, StatusFailed:
		if p.ProviderPayoutID != "" ||
			p.ProcessingAt == nil ||
			p.ProcessingAt.IsZero() ||
			p.PaidAt != nil ||
			!hasFailureReason(p.ErrorType, p.ErrorCode, p.ErrorMsg) {
			return ErrInvalidStatus
		}
	}

	return nil
}

func validateFailureReason(errorType *string, errorCode *string, errorMsg *string) error {
	if err := validateOptionalString(errorType, MaxErrorTypeLength, ErrInvalidErrorType); err != nil {
		return err
	}
	if err := validateOptionalString(errorCode, MaxErrorCodeLength, ErrInvalidErrorCode); err != nil {
		return err
	}
	if err := validateOptionalString(errorMsg, MaxErrorMsgLength, ErrInvalidErrorMsg); err != nil {
		return err
	}
	if !hasFailureReason(errorType, errorCode, errorMsg) {
		return ErrFailureReasonRequired
	}

	return nil
}

func validateOptionalString(value *string, maxLength int, invalidErr error) error {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return invalidErr
	}
	if maxLength > 0 && len([]rune(trimmed)) > maxLength {
		return invalidErr
	}

	return nil
}

func hasFailureReason(errorType *string, errorCode *string, errorMsg *string) bool {
	return optionalStringHasValue(errorType) ||
		optionalStringHasValue(errorCode) ||
		optionalStringHasValue(errorMsg)
}

func optionalStringHasValue(value *string) bool {
	return value != nil && strings.TrimSpace(*value) != ""
}

func cloneOptionalString(value *string) *string {
	if value == nil {
		return nil
	}

	cloned := strings.TrimSpace(*value)
	return &cloned
}

func isValidIdentifier(value string, maxLength int) bool {
	if value == "" || strings.Contains(value, "/") {
		return false
	}
	if maxLength > 0 && len([]rune(value)) > maxLength {
		return false
	}

	return true
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
