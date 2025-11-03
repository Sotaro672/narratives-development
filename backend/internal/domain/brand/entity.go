package brand

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// TS: Brand を正として定義（status は持たず isActive のみ）
type Brand struct {
	ID            string     `json:"id"`
	CompanyID     string     `json:"companyId"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	URL           string     `json:"websiteUrl,omitempty"` // TS: websiteUrl（内部フィールドは URL のまま）
	IsActive      bool       `json:"isActive"`
	ManagerID     *string    `json:"manager,omitempty"` // optional
	WalletAddress string     `json:"walletAddress"`
	CreatedAt     time.Time  `json:"createdAt"`
	CreatedBy     *string    `json:"createdBy,omitempty"`
	UpdatedAt     *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy     *string    `json:"updatedBy,omitempty"`
	DeletedAt     *time.Time `json:"deletedAt,omitempty"`
	DeletedBy     *string    `json:"deletedBy,omitempty"`
}

// Errors
var (
	ErrInvalidID            = errors.New("brand: invalid id")
	ErrInvalidCompanyID     = errors.New("brand: invalid companyId")
	ErrInvalidName          = errors.New("brand: invalid name")
	ErrInvalidDescription   = errors.New("brand: invalid description")
	ErrInvalidURL           = errors.New("brand: invalid url")
	ErrInvalidWalletAddress = errors.New("brand: invalid walletAddress")
	ErrInvalidCreatedAt     = errors.New("brand: invalid createdAt")
	ErrInvalidUpdatedAt     = errors.New("brand: invalid updatedAt")
)

// New constructs a Brand aligned to TS Brand.
// websiteUrl/manager/createdBy は任意。updated/ deleted は呼び出し側で必要に応じて設定してください。
func New(
	id, companyID, name, description, walletAddress string,
	websiteURL string,
	isActive bool,
	managerID, createdBy *string,
	createdAt time.Time,
) (Brand, error) {
	createdAt = createdAt.UTC()
	b := Brand{
		ID:            strings.TrimSpace(id),
		CompanyID:     strings.TrimSpace(companyID),
		Name:          strings.TrimSpace(name),
		Description:   strings.TrimSpace(description),
		URL:           strings.TrimSpace(websiteURL),
		IsActive:      isActive,
		ManagerID:     normalizePtr(managerID),
		WalletAddress: strings.TrimSpace(walletAddress),
		CreatedAt:     createdAt,
		CreatedBy:     normalizePtr(createdBy),
		UpdatedAt:     nil,
		UpdatedBy:     nil,
		DeletedAt:     nil,
		DeletedBy:     nil,
	}
	if err := b.validate(); err != nil {
		return Brand{}, err
	}
	return b, nil
}

// NewMinimal constructs a Brand with required fields only.
// isActive はデフォルト true。任意フィールドは空/未設定。
func NewMinimal(
	id, companyID, name, description, walletAddress string,
	createdAt time.Time,
) (Brand, error) {
	return New(id, companyID, name, description, walletAddress, "", true, nil, nil, createdAt)
}

// NewFromStringTime allows createdAt as string (ISO8601 variants).
func NewFromStringTime(
	id, companyID, name, description, walletAddress, createdAt string,
) (Brand, error) {
	t, err := parseTime(createdAt)
	if err != nil {
		return Brand{}, fmt.Errorf("%w: %v", ErrInvalidCreatedAt, err)
	}
	return NewMinimal(id, companyID, name, description, walletAddress, t)
}

func (b Brand) validate() error {
	if b.ID == "" {
		return ErrInvalidID
	}
	if b.CompanyID == "" {
		return ErrInvalidCompanyID
	}
	if b.Name == "" {
		return ErrInvalidName
	}
	if b.Description == "" {
		return ErrInvalidDescription
	}
	if b.URL != "" && !isValidURL(b.URL) {
		return ErrInvalidURL
	}
	if b.WalletAddress == "" {
		return ErrInvalidWalletAddress
	}
	if b.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if b.UpdatedAt != nil && b.UpdatedAt.Before(b.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	if b.DeletedAt != nil && b.UpdatedAt != nil && b.DeletedAt.Before(*b.UpdatedAt) {
		// deletedAt can be after createdAt but should not be before updatedAt if both exist
		return ErrInvalidUpdatedAt
	}
	return nil
}

func isValidURL(s string) bool {
	u, err := url.ParseRequestURI(s)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// State transitions (status は持たないため isActive のみを更新)

func (b *Brand) Activate(now time.Time, by *string) {
	now = now.UTC()
	b.IsActive = true
	b.UpdatedAt = &now
	b.UpdatedBy = normalizePtr(by)
}

func (b *Brand) Deactivate(now time.Time, by *string) {
	now = now.UTC()
	b.IsActive = false
	b.UpdatedAt = &now
	b.UpdatedBy = normalizePtr(by)
}

// Soft delete / restore

func (b *Brand) Delete(now time.Time, by *string) {
	now = now.UTC()
	b.DeletedAt = &now
	b.DeletedBy = normalizePtr(by)
	b.UpdatedAt = &now
	b.UpdatedBy = normalizePtr(by)
	// 任意: 論理削除時に無効化
	b.IsActive = false
}

func (b *Brand) Restore(now time.Time, by *string) {
	now = now.UTC()
	b.DeletedAt = nil
	b.DeletedBy = nil
	b.UpdatedAt = &now
	b.UpdatedBy = normalizePtr(by)
}

// Updates

func (b *Brand) UpdateName(name string, now time.Time, by *string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidName
	}
	now = now.UTC()
	b.Name = name
	b.UpdatedAt = &now
	b.UpdatedBy = normalizePtr(by)
	return nil
}

// websiteUrl（内部的には URL）更新
func (b *Brand) UpdateURL(urlStr string, now time.Time, by *string) error {
	urlStr = strings.TrimSpace(urlStr)
	if urlStr != "" && !isValidURL(urlStr) {
		return ErrInvalidURL
	}
	now = now.UTC()
	b.URL = urlStr
	b.UpdatedAt = &now
	b.UpdatedBy = normalizePtr(by)
	return nil
}

func (b *Brand) UpdateManager(managerID string, now time.Time, by *string) {
	now = now.UTC()
	managerID = strings.TrimSpace(managerID)
	if managerID == "" {
		b.ManagerID = nil
	} else {
		b.ManagerID = &managerID
	}
	b.UpdatedAt = &now
	b.UpdatedBy = normalizePtr(by)
}

func (b *Brand) UpdateWalletAddress(addr string, now time.Time, by *string) error {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ErrInvalidWalletAddress
	}
	now = now.UTC()
	b.WalletAddress = addr
	b.UpdatedAt = &now
	b.UpdatedBy = normalizePtr(by)
	return nil
}

// Helpers

func parseTime(s string) (time.Time, error) {
	if strings.TrimSpace(s) == "" {
		return time.Time{}, errors.New("empty time")
	}
	// Try common ISO8601 formats
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

func normalizePtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
}

// BrandPatch: リポジトリが部分更新時に参照（TS準拠のフィールドのみ）
type BrandPatch struct {
	CompanyID     *string
	Name          *string
	Description   *string
	URL           *string // websiteUrl
	IsActive      *bool
	ManagerID     *string
	WalletAddress *string
	UpdatedAt     *time.Time
	UpdatedBy     *string
	DeletedAt     *time.Time
	DeletedBy     *string
	CreatedBy     *string // ほぼ更新しないがスキーマに存在するため保持
}

// DDL reference (used by backend/cmd/ddlgen)
const BrandsTableDDL = `
CREATE TABLE IF NOT EXISTS brands (
    id UUID PRIMARY KEY,
    company_id TEXT NOT NULL,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(1000) NOT NULL,
    website_url TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    manager_id TEXT,
    wallet_address TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NULL,
    updated_at TIMESTAMPTZ NULL,
    updated_by TEXT NULL,
    deleted_at TIMESTAMPTZ NULL,
    deleted_by TEXT NULL,
    CONSTRAINT fk_brands_company
        FOREIGN KEY (company_id)
        REFERENCES companies(id)
        ON UPDATE CASCADE
        ON DELETE RESTRICT,
    CONSTRAINT fk_brands_manager
        FOREIGN KEY (manager_id)
        REFERENCES members(id)
        ON UPDATE CASCADE
        ON DELETE SET NULL,
    CONSTRAINT uq_brands_company_name UNIQUE (company_id, name),
    CHECK (updated_at IS NULL OR updated_at >= created_at),
    CHECK (deleted_at IS NULL OR deleted_at >= created_at)
);
CREATE INDEX IF NOT EXISTS idx_brands_company_id ON brands(company_id);
CREATE INDEX IF NOT EXISTS idx_brands_manager_id ON brands(manager_id);
CREATE INDEX IF NOT EXISTS idx_brands_is_active ON brands(is_active);
CREATE INDEX IF NOT EXISTS idx_brands_deleted_at ON brands(deleted_at);
`
