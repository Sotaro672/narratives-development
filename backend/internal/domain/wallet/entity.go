// backend\internal\domain\wallet\entity.go
package wallet

import (
	"errors"
	"regexp"
	"time"
)

// Domain errors
var (
	ErrInvalidWalletAddress = errors.New("wallet: invalid walletAddress")
	ErrInvalidAssetID       = errors.New("wallet: invalid assetId")
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

// Solana-like base58 wallet address / assetId format (approximation).
var base58Re = regexp.MustCompile(`^[1-9A-HJ-NP-Za-km-z]{32,44}$`)

func isValidWallet(s string) bool {
	return base58Re.MatchString(s)
}

func isValidAssetID(s string) bool {
	return base58Re.MatchString(s)
}

// Wallet mirrors web-app/src/shared/types/wallet.ts (updated)
//
//	interface Wallet {
//	  walletAddress: string;
//	  assetIds: string[];
//	  lastUpdatedAt: string;
//	  status: 'active' | 'inactive';
//	}
type Wallet struct {
	WalletAddress string
	AssetIDs      []string
	LastUpdatedAt time.Time
	Status        WalletStatus
}

// New constructs a Wallet.
// It sets LastUpdatedAt to updatedAt, and Status to 'active'.
func New(addr string, assetIDs []string, updatedAt time.Time) (Wallet, error) {
	w := Wallet{
		WalletAddress: addr,
		AssetIDs:      nil,
		LastUpdatedAt: updatedAt.UTC(),
		Status:        StatusActive,
	}

	if err := w.setAssetIDs(assetIDs); err != nil {
		return Wallet{}, err
	}

	if err := w.validate(); err != nil {
		return Wallet{}, err
	}

	return w, nil
}

// NewFull constructs a Wallet with all fields explicitly provided.
func NewFull(addr string, assetIDs []string, lastUpdatedAt time.Time, status WalletStatus) (Wallet, error) {
	w := Wallet{
		WalletAddress: addr,
		AssetIDs:      nil,
		LastUpdatedAt: lastUpdatedAt.UTC(),
		Status:        status,
	}

	if err := w.setAssetIDs(assetIDs); err != nil {
		return Wallet{}, err
	}

	if err := w.validate(); err != nil {
		return Wallet{}, err
	}

	return w, nil
}

// NewNow constructs Wallet using current time for LastUpdatedAt.
func NewNow(addr string, assetIDs []string, status WalletStatus) (Wallet, error) {
	now := time.Now().UTC()

	return NewFull(addr, assetIDs, now, status)
}

// Behavior

// AddAssetID appends an assetId if not present and updates LastUpdatedAt.
func (w *Wallet) AddAssetID(assetID string, now time.Time) error {
	if !isValidAssetID(assetID) {
		return ErrInvalidAssetID
	}

	if !contains(w.AssetIDs, assetID) {
		w.AssetIDs = append(w.AssetIDs, assetID)
		w.touch(now)
	}

	return nil
}

// RemoveAssetID removes an assetId if present and updates LastUpdatedAt when changed.
func (w *Wallet) RemoveAssetID(assetID string, now time.Time) bool {
	if assetID == "" {
		return false
	}

	before := len(w.AssetIDs)

	w.AssetIDs = remove(w.AssetIDs, assetID)

	changed := len(w.AssetIDs) != before
	if changed {
		w.touch(now)
	}

	return changed
}

// ReplaceAssetIDs replaces and validates the assetId set, deduplicated.
func (w *Wallet) ReplaceAssetIDs(assetIDs []string, now time.Time) error {
	if err := w.setAssetIDs(assetIDs); err != nil {
		return err
	}

	w.touch(now)

	return nil
}

func (w *Wallet) HasAssetID(assetID string) bool {
	return contains(w.AssetIDs, assetID)
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
	if !isValidWallet(w.WalletAddress) {
		return ErrInvalidWalletAddress
	}

	if w.LastUpdatedAt.IsZero() {
		return ErrInvalidLastUpdatedAt
	}

	if !isValidStatus(w.Status) {
		return ErrInvalidStatus
	}

	for _, assetID := range w.AssetIDs {
		if !isValidAssetID(assetID) {
			return ErrInvalidAssetID
		}
	}

	return nil
}

func (w *Wallet) setAssetIDs(assetIDs []string) error {
	d := dedup(assetIDs)

	for _, assetID := range d {
		if !isValidAssetID(assetID) {
			return ErrInvalidAssetID
		}
	}

	w.AssetIDs = d

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
