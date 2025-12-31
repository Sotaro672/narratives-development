// backend/internal/domain/wallet/entity.go
package wallet

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Domain errors
var (
	ErrInvalidAvatarID      = errors.New("wallet: invalid avatarId")
	ErrInvalidWalletAddress = errors.New("wallet: invalid walletAddress")
	ErrInvalidMintAddress   = errors.New("wallet: invalid mintAddress")
	ErrInvalidLastUpdatedAt = errors.New("wallet: invalid lastUpdatedAt")
	ErrInvalidStatus        = errors.New("wallet: invalid status")
	// 追加: NotFound 用のドメインエラー
	ErrNotFound = errors.New("wallet: not found")
)

// WalletStatus mirrors TS: 'active' | 'inactive'
type WalletStatus string

const (
	StatusActive   WalletStatus = "active"
	StatusInactive WalletStatus = "inactive"
)

func isValidStatus(s WalletStatus) bool {
	return s == StatusActive || s == StatusInactive
}

// Solana-like base58 address/mint format (approximation).
var base58Re = regexp.MustCompile(`^[1-9A-HJ-NP-Za-km-z]{32,44}$`)

func isValidWallet(s string) bool {
	return base58Re.MatchString(s)
}

func isValidMint(s string) bool {
	return base58Re.MatchString(s)
}

// Wallet mirrors web-app/src/shared/types/wallet.ts (updated)
//
//	interface Wallet {
//	  avatarId: string;
//	  walletAddress: string;
//	  tokens: string[];
//	  lastUpdatedAt: string;
//	  status: 'active' | 'inactive';
//	}
type Wallet struct {
	AvatarID      string
	WalletAddress string
	Tokens        []string
	LastUpdatedAt time.Time
	Status        WalletStatus
}

// New constructs a Wallet.
// It sets LastUpdatedAt to updatedAt, and Status to 'active'.
func New(avatarID, addr string, tokens []string, updatedAt time.Time) (Wallet, error) {
	w := Wallet{
		AvatarID:      strings.TrimSpace(avatarID),
		WalletAddress: addr,
		Tokens:        nil,
		LastUpdatedAt: updatedAt.UTC(),
		Status:        StatusActive,
	}
	if err := w.setTokens(tokens); err != nil {
		return Wallet{}, err
	}
	if err := w.validate(); err != nil {
		return Wallet{}, err
	}
	return w, nil
}

// NewFull constructs a Wallet with all fields explicitly provided.
func NewFull(avatarID, addr string, tokens []string, lastUpdatedAt time.Time, status WalletStatus) (Wallet, error) {
	w := Wallet{
		AvatarID:      strings.TrimSpace(avatarID),
		WalletAddress: addr,
		Tokens:        nil,
		LastUpdatedAt: lastUpdatedAt.UTC(),
		Status:        status,
	}
	if err := w.setTokens(tokens); err != nil {
		return Wallet{}, err
	}
	if err := w.validate(); err != nil {
		return Wallet{}, err
	}
	return w, nil
}

// NewNow constructs Wallet using current time for LastUpdatedAt.
func NewNow(avatarID, addr string, tokens []string, status WalletStatus) (Wallet, error) {
	now := time.Now().UTC()
	return NewFull(avatarID, addr, tokens, now, status)
}

// NewFromStringTime accepts lastUpdatedAt as string (ISO8601). Status becomes 'active'.
func NewFromStringTime(avatarID, addr string, tokens []string, lastUpdatedAt string) (Wallet, error) {
	t, err := parseTime(lastUpdatedAt)
	if err != nil {
		return Wallet{}, fmt.Errorf("%w: %v", ErrInvalidLastUpdatedAt, err)
	}
	return New(avatarID, addr, tokens, t)
}

// NewFromStringTimes accepts ISO8601 string for lastUpdated and status.
func NewFromStringTimes(avatarID, addr string, tokens []string, lastUpdatedAt, status string) (Wallet, error) {
	lut, err := parseTime(lastUpdatedAt)
	if err != nil {
		return Wallet{}, fmt.Errorf("%w: %v", ErrInvalidLastUpdatedAt, err)
	}
	ws := WalletStatus(status)
	if !isValidStatus(ws) {
		return Wallet{}, ErrInvalidStatus
	}
	return NewFull(avatarID, addr, tokens, lut, ws)
}

// Behavior

// AddToken appends a mint if not present and updates LastUpdatedAt.
func (w *Wallet) AddToken(mint string, now time.Time) error {
	if !isValidMint(mint) {
		return ErrInvalidMintAddress
	}
	if !contains(w.Tokens, mint) {
		w.Tokens = append(w.Tokens, mint)
		w.touch(now)
	}
	return nil
}

// RemoveToken removes a mint if present and updates LastUpdatedAt when changed.
func (w *Wallet) RemoveToken(mint string, now time.Time) bool {
	if mint == "" {
		return false
	}
	before := len(w.Tokens)
	w.Tokens = remove(w.Tokens, mint)
	changed := len(w.Tokens) != before
	if changed {
		w.touch(now)
	}
	return changed
}

// ReplaceTokens replaces and validates the token set, deduplicated.
func (w *Wallet) ReplaceTokens(tokens []string, now time.Time) error {
	if err := w.setTokens(tokens); err != nil {
		return err
	}
	w.touch(now)
	return nil
}

func (w *Wallet) HasToken(mint string) bool {
	return contains(w.Tokens, mint)
}

// SetStatus updates status and LastUpdatedAt.
func (w *Wallet) SetStatus(s WalletStatus, now time.Time) error {
	if !isValidStatus(s) {
		return ErrInvalidStatus
	}
	w.Status = s
	w.touch(now)
	return nil
}

// Validation and helpers

func (w Wallet) validate() error {
	if strings.TrimSpace(w.AvatarID) == "" {
		return ErrInvalidAvatarID
	}
	if !isValidWallet(w.WalletAddress) {
		return ErrInvalidWalletAddress
	}
	if w.LastUpdatedAt.IsZero() {
		return ErrInvalidLastUpdatedAt
	}
	if !isValidStatus(w.Status) {
		return ErrInvalidStatus
	}
	for _, t := range w.Tokens {
		if !isValidMint(t) {
			return ErrInvalidMintAddress
		}
	}
	return nil
}

func (w *Wallet) setTokens(tokens []string) error {
	d := dedup(tokens)
	for _, t := range d {
		if !isValidMint(t) {
			return ErrInvalidMintAddress
		}
	}
	w.Tokens = d
	return nil
}

func (w *Wallet) touch(now time.Time) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	w.LastUpdatedAt = now.UTC()
}

func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func remove(xs []string, v string) []string {
	out := xs[:0]
	for _, x := range xs {
		if x != v {
			out = append(out, x)
		}
	}
	return out
}

func dedup(xs []string) []string {
	seen := make(map[string]struct{}, len(xs))
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		if x == "" {
			continue
		}
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	return out
}

func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, ErrInvalidLastUpdatedAt
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
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time: %q", s)
}
