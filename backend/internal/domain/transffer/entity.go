package transffer

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// TransfferStatus mirrors TS: "fulfilled" | "requested" | "error"
type TransfferStatus string

const (
	StatusFulfilled TransfferStatus = "fulfilled"
	StatusRequested TransfferStatus = "requested"
	StatusError     TransfferStatus = "error"
)

func IsValidStatus(s TransfferStatus) bool {
	switch s {
	case StatusFulfilled, StatusRequested, StatusError:
		return true
	default:
		return false
	}
}

// TransfferErrorType mirrors TS:
// "insufficient_balance" | "invalid_address" | "network_error" | "timeout" | "unknown" | null
type TransfferErrorType string

const (
	ErrTypeInsufficientBalance TransfferErrorType = "insufficient_balance"
	ErrTypeInvalidAddress      TransfferErrorType = "invalid_address"
	ErrTypeNetworkError        TransfferErrorType = "network_error"
	ErrTypeTimeout             TransfferErrorType = "timeout"
	ErrTypeUnknown             TransfferErrorType = "unknown"
)

func IsValidErrorType(t TransfferErrorType) bool {
	switch t {
	case ErrTypeInsufficientBalance, ErrTypeInvalidAddress, ErrTypeNetworkError, ErrTypeTimeout, ErrTypeUnknown:
		return true
	default:
		return false
	}
}

// Transffer mirrors web-app/src/shared/types/transffer.ts
type Transffer struct {
	ID            string
	MintAddress   string
	FromAddress   string
	ToAddress     string
	RequestedAt   time.Time
	TransfferedAt *time.Time
	Status        TransfferStatus
	ErrorType     *TransfferErrorType
}

// Errors
var (
	ErrInvalidID            = errors.New("transffer: invalid id")
	ErrInvalidMintAddress   = errors.New("transffer: invalid mintAddress")
	ErrInvalidFromAddress   = errors.New("transffer: invalid fromAddress")
	ErrInvalidToAddress     = errors.New("transffer: invalid toAddress")
	ErrInvalidRequestedAt   = errors.New("transffer: invalid requestedAt")
	ErrInvalidTransfferedAt = errors.New("transffer: invalid transfferedAt")
	ErrInvalidStatus        = errors.New("transffer: invalid status")
	ErrInvalidErrorType     = errors.New("transffer: invalid errorType")
	ErrInvalidTransition    = errors.New("transffer: invalid status transition")
	ErrIncoherentState      = errors.New("transffer: incoherent fields for status")
)

// Policy (sync with transfferConstants.ts if defined)
var (
	// Base58 constraints for Solana addresses (same as token domain)
	Base58MinLen   = 32
	Base58MaxLen   = 44
	base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

	// Allowed transitions:
	// requested -> fulfilled | error
	// error -> requested (retry)
	// fulfilled -> (terminal)
	allowedTransitions = map[TransfferStatus]map[TransfferStatus]struct{}{
		StatusRequested: {StatusFulfilled: {}, StatusError: {}},
		StatusError:     {StatusRequested: {}},
		StatusFulfilled: {},
	}
)

// Constructors

func New(
	id, mintAddress, fromAddress, toAddress string,
	requestedAt time.Time,
	transfferedAt *time.Time,
	status TransfferStatus,
	errorType *TransfferErrorType,
) (Transffer, error) {
	if status == "" {
		status = StatusRequested
	}
	tr := Transffer{
		ID:            strings.TrimSpace(id),
		MintAddress:   strings.TrimSpace(mintAddress),
		FromAddress:   strings.TrimSpace(fromAddress),
		ToAddress:     strings.TrimSpace(toAddress),
		RequestedAt:   requestedAt.UTC(),
		TransfferedAt: normalizeTimePtr(transfferedAt),
		Status:        status,
		ErrorType:     normalizeErrorTypePtr(errorType),
	}
	if err := tr.validate(); err != nil {
		return Transffer{}, err
	}
	return tr, nil
}

func NewFromStringTimes(
	id, mintAddress, fromAddress, toAddress string,
	requestedAtStr string,
	transfferedAtStr *string, // nil or empty -> null
	status TransfferStatus,
	errorType *TransfferErrorType,
) (Transffer, error) {
	req, err := parseTime(requestedAtStr, ErrInvalidRequestedAt)
	if err != nil {
		return Transffer{}, err
	}
	var trAt *time.Time
	if transfferedAtStr != nil && strings.TrimSpace(*transfferedAtStr) != "" {
		t, err := parseTime(*transfferedAtStr, ErrInvalidTransfferedAt)
		if err != nil {
			return Transffer{}, err
		}
		trAt = &t
	}
	return New(id, mintAddress, fromAddress, toAddress, req, trAt, status, errorType)
}

// Behavior

func (t *Transffer) SetStatus(next TransfferStatus, now time.Time) error {
	if !IsValidStatus(next) {
		return ErrInvalidStatus
	}
	if !transitionAllowed(t.Status, next) {
		return ErrInvalidTransition
	}
	switch next {
	case StatusRequested:
		// reset for retry
		t.ErrorType = nil
		t.TransfferedAt = nil
	case StatusFulfilled:
		if t.TransfferedAt == nil || t.TransfferedAt.IsZero() {
			return ErrInvalidTransfferedAt
		}
		t.ErrorType = nil
	case StatusError:
		if t.ErrorType == nil || !IsValidErrorType(*t.ErrorType) {
			return ErrInvalidErrorType
		}
		// must not have transfferedAt
		t.TransfferedAt = nil
	}
	// requestedAt should never be zero; no time mutation required here
	t.Status = next
	return nil
}

func (t *Transffer) MarkFulfilled(at time.Time) error {
	if at.IsZero() {
		return ErrInvalidTransfferedAt
	}
	utc := at.UTC()
	t.TransfferedAt = &utc
	return t.SetStatus(StatusFulfilled, utc)
}

func (t *Transffer) MarkError(errType TransfferErrorType) error {
	if !IsValidErrorType(errType) {
		return ErrInvalidErrorType
	}
	t.ErrorType = &errType
	return t.SetStatus(StatusError, time.Now().UTC())
}

func (t *Transffer) Retry() error {
	// Clear error and set back to requested
	t.ErrorType = nil
	t.TransfferedAt = nil
	return t.SetStatus(StatusRequested, time.Now().UTC())
}

// Validation

func (t Transffer) validate() error {
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
	// Coherence checks by status
	switch t.Status {
	case StatusRequested:
		if t.ErrorType != nil || t.TransfferedAt != nil {
			return ErrIncoherentState
		}
	case StatusFulfilled:
		if t.ErrorType != nil {
			return ErrIncoherentState
		}
		if t.TransfferedAt == nil || t.TransfferedAt.IsZero() || t.TransfferedAt.Before(t.RequestedAt) {
			return ErrInvalidTransfferedAt
		}
	case StatusError:
		if t.ErrorType == nil || !IsValidErrorType(*t.ErrorType) {
			return ErrInvalidErrorType
		}
		if t.TransfferedAt != nil {
			return ErrIncoherentState
		}
	}
	return nil
}

// Helpers

func transitionAllowed(from, to TransfferStatus) bool {
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

func normalizeErrorTypePtr(p *TransfferErrorType) *TransfferErrorType {
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
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("%w: cannot parse %q", classify, s)
}

// TransffersTableDDL defines the SQL for the transffers table migration.
const TransffersTableDDL = `
-- Migration: Initialize Transffer domain
-- Mirrors backend/internal/domain/transffer/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS transffers (
  id              TEXT        PRIMARY KEY,
  mint_address    TEXT        NOT NULL,
  from_address    TEXT        NOT NULL,
  to_address      TEXT        NOT NULL,
  requested_at    TIMESTAMPTZ NOT NULL,
  transffered_at  TIMESTAMPTZ,
  status          TEXT        NOT NULL CHECK (status IN ('fulfilled','requested','error')),
  error_type      TEXT,

  -- Non-empty checks
  CONSTRAINT chk_transffers_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(mint_address)) > 0
    AND char_length(trim(from_address)) > 0
    AND char_length(trim(to_address)) > 0
  ),

  -- error_type whitelist (nullable)
  CONSTRAINT chk_transffers_error_type CHECK (
    error_type IS NULL OR error_type IN ('insufficient_balance','invalid_address','network_error','timeout','unknown')
  ),

  -- time order
  CONSTRAINT chk_transffers_time_order CHECK (
    transffered_at IS NULL OR transffered_at >= requested_at
  ),

  -- state coherence
  CONSTRAINT chk_transffers_state_coherence CHECK (
    (status = 'requested' AND transffered_at IS NULL AND error_type IS NULL)
    OR (status = 'fulfilled' AND transffered_at IS NOT NULL AND error_type IS NULL)
    OR (status = 'error'     AND transffered_at IS NULL AND error_type IS NOT NULL)
  )
);

-- Useful indexes
CREATE INDEX IF NOT EXISTS idx_transffers_mint_address    ON transffers(mint_address);
CREATE INDEX IF NOT EXISTS idx_transffers_from_address    ON transffers(from_address);
CREATE INDEX IF NOT EXISTS idx_transffers_to_address      ON transffers(to_address);
CREATE INDEX IF NOT EXISTS idx_transffers_status          ON transffers(status);
CREATE INDEX IF NOT EXISTS idx_transffers_requested_at    ON transffers(requested_at);
CREATE INDEX IF NOT EXISTS idx_transffers_transffered_at  ON transffers(transffered_at);

COMMIT;
`
