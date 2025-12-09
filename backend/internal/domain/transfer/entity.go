// backend\internal\domain\transfer\entity.go
package transfer

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// TransferStatus ...
type TransferStatus string

const (
	StatusFulfilled TransferStatus = "fulfilled"
	StatusRequested TransferStatus = "requested"
	StatusError     TransferStatus = "error"
)

func IsValidStatus(s TransferStatus) bool {
	switch s {
	case StatusFulfilled, StatusRequested, StatusError:
		return true
	default:
		return false
	}
}

// TransferErrorType ...
type TransferErrorType string

const (
	ErrTypeInsufficientBalance TransferErrorType = "insufficient_balance"
	ErrTypeInvalidAddress      TransferErrorType = "invalid_address"
	ErrTypeNetworkError        TransferErrorType = "network_error"
	ErrTypeTimeout             TransferErrorType = "timeout"
	ErrTypeUnknown             TransferErrorType = "unknown"
)

func IsValidErrorType(t TransferErrorType) bool {
	switch t {
	case ErrTypeInsufficientBalance, ErrTypeInvalidAddress, ErrTypeNetworkError, ErrTypeTimeout, ErrTypeUnknown:
		return true
	default:
		return false
	}
}

// Transfer
type Transfer struct {
	ID            string
	MintAddress   string
	FromAddress   string
	ToAddress     string
	RequestedAt   time.Time
	TransferredAt *time.Time
	Status        TransferStatus
	ErrorType     *TransferErrorType
}

// Errors
var (
	ErrInvalidID            = errors.New("transfer: invalid id")
	ErrInvalidMintAddress   = errors.New("transfer: invalid mintAddress")
	ErrInvalidFromAddress   = errors.New("transfer: invalid fromAddress")
	ErrInvalidToAddress     = errors.New("transfer: invalid toAddress")
	ErrInvalidRequestedAt   = errors.New("transfer: invalid requestedAt")
	ErrInvalidTransferredAt = errors.New("transfer: invalid transferredAt")
	ErrInvalidStatus        = errors.New("transfer: invalid status")
	ErrInvalidErrorType     = errors.New("transfer: invalid errorType")
	ErrInvalidTransition    = errors.New("transfer: invalid status transition")
	ErrIncoherentState      = errors.New("transfer: incoherent fields for status")
)

var (
	Base58MinLen       = 32
	Base58MaxLen       = 44
	base58Alphabet     = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	allowedTransitions = map[TransferStatus]map[TransferStatus]struct{}{
		StatusRequested: {StatusFulfilled: {}, StatusError: {}},
		StatusError:     {StatusRequested: {}},
		StatusFulfilled: {},
	}
)

// Constructors

func New(
	id, mintAddress, fromAddress, toAddress string,
	requestedAt time.Time,
	transferredAt *time.Time,
	status TransferStatus,
	errorType *TransferErrorType,
) (Transfer, error) {
	if status == "" {
		status = StatusRequested
	}
	tr := Transfer{
		ID:            strings.TrimSpace(id),
		MintAddress:   strings.TrimSpace(mintAddress),
		FromAddress:   strings.TrimSpace(fromAddress),
		ToAddress:     strings.TrimSpace(toAddress),
		RequestedAt:   requestedAt.UTC(),
		TransferredAt: normalizeTimePtr(transferredAt),
		Status:        status,
		ErrorType:     normalizeErrorTypePtr(errorType),
	}
	if err := tr.validate(); err != nil {
		return Transfer{}, err
	}
	return tr, nil
}

func NewFromStringTimes(
	id, mintAddress, fromAddress, toAddress string,
	requestedAtStr string,
	transferredAtStr *string,
	status TransferStatus,
	errorType *TransferErrorType,
) (Transfer, error) {
	req, err := parseTime(requestedAtStr, ErrInvalidRequestedAt)
	if err != nil {
		return Transfer{}, err
	}
	var trAt *time.Time
	if transferredAtStr != nil && strings.TrimSpace(*transferredAtStr) != "" {
		t, err := parseTime(*transferredAtStr, ErrInvalidTransferredAt)
		if err != nil {
			return Transfer{}, err
		}
		trAt = &t
	}
	return New(id, mintAddress, fromAddress, toAddress, req, trAt, status, errorType)
}

// Behavior

func (t *Transfer) SetStatus(next TransferStatus, _ time.Time) error {
	if !IsValidStatus(next) {
		return ErrInvalidStatus
	}
	if !transitionAllowed(t.Status, next) {
		return ErrInvalidTransition
	}
	switch next {
	case StatusRequested:
		t.ErrorType = nil
		t.TransferredAt = nil
	case StatusFulfilled:
		if t.TransferredAt == nil || t.TransferredAt.IsZero() {
			return ErrInvalidTransferredAt
		}
		t.ErrorType = nil
	case StatusError:
		if t.ErrorType == nil || !IsValidErrorType(*t.ErrorType) {
			return ErrInvalidErrorType
		}
		t.TransferredAt = nil
	}
	t.Status = next
	return nil
}

func (t *Transfer) MarkFulfilled(at time.Time) error {
	if at.IsZero() {
		return ErrInvalidTransferredAt
	}
	utc := at.UTC()
	t.TransferredAt = &utc
	return t.SetStatus(StatusFulfilled, utc)
}

func (t *Transfer) MarkError(errType TransferErrorType) error {
	if !IsValidErrorType(errType) {
		return ErrInvalidErrorType
	}
	t.ErrorType = &errType
	return t.SetStatus(StatusError, time.Now().UTC())
}

func (t *Transfer) Retry() error {
	t.ErrorType = nil
	t.TransferredAt = nil
	return t.SetStatus(StatusRequested, time.Now().UTC())
}

// Validation

func (t Transfer) validate() error {
	if t.ID == "" {
		return ErrInvalidID
	}
	if !isValidBase58(t.MintAddress) {
		return ErrInvalidMintAddress
	}
	if !isValidBase58(t.FromAddress) {
		return ErrInvalidFromAddress
	}
	if !isValidBase58(t.ToAddress) {
		return ErrInvalidToAddress
	}
	if t.RequestedAt.IsZero() {
		return ErrInvalidRequestedAt
	}
	if !IsValidStatus(t.Status) {
		return ErrInvalidStatus
	}
	switch t.Status {
	case StatusRequested:
		if t.ErrorType != nil || t.TransferredAt != nil {
			return ErrIncoherentState
		}
	case StatusFulfilled:
		if t.ErrorType != nil {
			return ErrIncoherentState
		}
		if t.TransferredAt == nil || t.TransferredAt.IsZero() || t.TransferredAt.Before(t.RequestedAt) {
			return ErrInvalidTransferredAt
		}
	case StatusError:
		if t.ErrorType == nil || !IsValidErrorType(*t.ErrorType) {
			return ErrInvalidErrorType
		}
		if t.TransferredAt != nil {
			return ErrIncoherentState
		}
	}
	return nil
}

// Helpers (unchanged)

func transitionAllowed(from, to TransferStatus) bool {
	if m, ok := allowedTransitions[from]; ok {
		_, ok := m[to]
		return ok
	}
	return false
}

func isValidBase58(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if len(s) < Base58MinLen || (Base58MaxLen > 0 && len(s) > Base58MaxLen) {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !strings.ContainsRune(base58Alphabet, rune(s[i])) {
			return false
		}
	}
	return true
}

func normalizeTimePtr(p *time.Time) *time.Time {
	if p == nil || p.IsZero() {
		return nil
	}
	u := p.UTC()
	return &u
}

func normalizeErrorTypePtr(p *TransferErrorType) *TransferErrorType {
	if p == nil {
		return nil
	}
	if !IsValidErrorType(*p) {
		return nil
	}
	v := *p
	return &v
}

func parseTime(s string, classify error) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, classify
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	for _, l := range []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	} {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("%w: cannot parse %q", classify, s)
}
