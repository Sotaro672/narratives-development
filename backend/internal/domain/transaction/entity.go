// backend\internal\domain\transaction\entity.go
package transaction

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// TransactionType (TS: "receive" | "send")
type TransactionType string

const (
	TypeReceive TransactionType = "receive"
	TypeSend    TransactionType = "send"
)

// Entity (mirror of web-app/src/shared/types/transaction.ts)
type Transaction struct {
	ID          string
	AccountID   string
	BrandName   string
	Type        TransactionType
	Amount      int
	Currency    string
	FromAccount string
	ToAccount   string
	Timestamp   time.Time
	Description string
}

// Errors
var (
	ErrInvalidID          = errors.New("transaction: invalid id")
	ErrInvalidAccountID   = errors.New("transaction: invalid accountId")
	ErrInvalidBrandName   = errors.New("transaction: invalid brandName")
	ErrInvalidType        = errors.New("transaction: invalid type")
	ErrInvalidAmount      = errors.New("transaction: invalid amount")
	ErrInvalidCurrency    = errors.New("transaction: invalid currency")
	ErrInvalidFromAccount = errors.New("transaction: invalid fromAccount")
	ErrInvalidToAccount   = errors.New("transaction: invalid toAccount")
	ErrInvalidTimestamp   = errors.New("transaction: invalid timestamp")
)

// Policy (align with TS; keep minimal)
var (
	MinAmount  = 0 // >= 0
	MaxAmount  = 0 // 0 disables upper-bound check
	CurrencyRe = regexp.MustCompile(`^[A-Z]{3}$`)
	// Optional allow-list (empty map = allow all that match CurrencyRe)
	AllowedCurrencies = map[string]struct{}{}
)

// Constructors

func New(
	id string,
	accountID string,
	brandName string,
	typ TransactionType,
	amount int,
	currency string,
	fromAccount string,
	toAccount string,
	timestamp time.Time,
	description string,
) (Transaction, error) {
	t := Transaction{
		ID:          strings.TrimSpace(id),
		AccountID:   strings.TrimSpace(accountID),
		BrandName:   strings.TrimSpace(brandName),
		Type:        typ,
		Amount:      amount,
		Currency:    strings.ToUpper(strings.TrimSpace(currency)),
		FromAccount: strings.TrimSpace(fromAccount),
		ToAccount:   strings.TrimSpace(toAccount),
		Timestamp:   timestamp.UTC(),
		Description: description, // allow empty; TS does not mandate non-empty
	}
	if err := t.validate(); err != nil {
		return Transaction{}, err
	}
	return t, nil
}

func NewFromStringTimestamp(
	id string,
	accountID string,
	brandName string,
	typ TransactionType,
	amount int,
	currency string,
	fromAccount string,
	toAccount string,
	timestamp string, // ISO8601/RFC3339
	description string,
) (Transaction, error) {
	ts, err := parseTime(timestamp)
	if err != nil {
		return Transaction{}, fmt.Errorf("%w: %v", ErrInvalidTimestamp, err)
	}
	return New(id, accountID, brandName, typ, amount, currency, fromAccount, toAccount, ts, description)
}

// Behavior (minimal)

func (t *Transaction) Touch(newTime time.Time) error {
	if newTime.IsZero() {
		return ErrInvalidTimestamp
	}
	t.Timestamp = newTime.UTC()
	return nil
}

func (t *Transaction) SetDescription(desc string) {
	t.Description = desc // free-form
}

func (t *Transaction) SetAmount(v int, at time.Time) error {
	if v < MinAmount || (MaxAmount > 0 && v > MaxAmount) {
		return ErrInvalidAmount
	}
	t.Amount = v
	return t.Touch(at)
}

func (t *Transaction) SetType(tp TransactionType, at time.Time) error {
	if err := validateType(tp); err != nil {
		return err
	}
	t.Type = tp
	return t.Touch(at)
}

// Validation

func (t Transaction) validate() error {
	if t.ID == "" {
		return ErrInvalidID
	}
	if t.AccountID == "" {
		return ErrInvalidAccountID
	}
	if t.BrandName == "" {
		return ErrInvalidBrandName
	}
	if err := validateType(t.Type); err != nil {
		return err
	}
	if t.Amount < MinAmount || (MaxAmount > 0 && t.Amount > MaxAmount) {
		return ErrInvalidAmount
	}
	if t.Currency == "" || (CurrencyRe != nil && !CurrencyRe.MatchString(t.Currency)) {
		return ErrInvalidCurrency
	}
	if len(AllowedCurrencies) > 0 {
		if _, ok := AllowedCurrencies[t.Currency]; !ok {
			return ErrInvalidCurrency
		}
	}
	if t.FromAccount == "" {
		return ErrInvalidFromAccount
	}
	if t.ToAccount == "" {
		return ErrInvalidToAccount
	}
	if t.Timestamp.IsZero() {
		return ErrInvalidTimestamp
	}
	return nil
}

func validateType(tp TransactionType) error {
	switch tp {
	case TypeReceive, TypeSend:
		return nil
	default:
		return ErrInvalidType
	}
}

// Helpers

func parseTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty time")
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
	return time.Time{}, fmt.Errorf("cannot parse time: %q", s)
}

// TransactionsTableDDL defines the SQL for the transactions table migration.
const TransactionsTableDDL = `
-- Migration: Initialize Transaction domain
-- Mirrors backend/internal/domain/transaction/entity.go and web-app/src/shared/types/transaction.ts

BEGIN;

CREATE TABLE IF NOT EXISTS transactions (
  id            TEXT        PRIMARY KEY,
  account_id    TEXT        NOT NULL,
  brand_name    TEXT        NOT NULL,
  type          TEXT        NOT NULL CHECK (type IN ('receive','send')),
  amount        BIGINT      NOT NULL CHECK (amount >= 0),
  currency      TEXT        NOT NULL,
  from_account  TEXT        NOT NULL,
  to_account    TEXT        NOT NULL,
  timestamp     TIMESTAMPTZ NOT NULL,
  description   TEXT        NOT NULL DEFAULT '',

  -- Non-empty checks
  CONSTRAINT chk_tx_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(account_id)) > 0
    AND char_length(trim(brand_name)) > 0
    AND char_length(trim(currency)) > 0
    AND char_length(trim(from_account)) > 0
    AND char_length(trim(to_account)) > 0
  ),

  -- currency format (ISO 4217-like: 3 uppercase letters)
  CONSTRAINT chk_tx_currency_format CHECK (currency ~ '^[A-Z]{3}$')
);

-- Useful indexes
CREATE INDEX IF NOT EXISTS idx_tx_account_id ON transactions(account_id);
CREATE INDEX IF NOT EXISTS idx_tx_brand_name ON transactions(brand_name);
CREATE INDEX IF NOT EXISTS idx_tx_type       ON transactions(type);
CREATE INDEX IF NOT EXISTS idx_tx_currency   ON transactions(currency);
CREATE INDEX IF NOT EXISTS idx_tx_timestamp  ON transactions(timestamp);

COMMIT;
`
