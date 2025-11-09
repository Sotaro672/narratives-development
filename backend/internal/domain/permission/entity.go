// backend\internal\domain\permission\entity.go
package permission

import (
	"errors"
	"fmt"
	"regexp"
)

// PermissionCategory mirrors web-app/src/shared/types/permission.ts
type PermissionCategory string

const (
	CategoryWallet       PermissionCategory = "wallet"
	CategoryInquiry      PermissionCategory = "inquiry"
	CategoryOrganization PermissionCategory = "organization"
	CategoryBrand        PermissionCategory = "brand"
	CategoryMember       PermissionCategory = "member"
	CategoryOrder        PermissionCategory = "order"
	CategoryProduct      PermissionCategory = "product"
	CategoryCampaign     PermissionCategory = "campaign"
	CategoryToken        PermissionCategory = "token"
	CategoryInventory    PermissionCategory = "inventory"
	CategoryProduction   PermissionCategory = "production"
	CategoryAnalytics    PermissionCategory = "analytics"
	CategorySystem       PermissionCategory = "system"
)

// CategoryValues returns all allowed categories.
func CategoryValues() []PermissionCategory {
	return []PermissionCategory{
		CategoryWallet,
		CategoryInquiry,
		CategoryOrganization,
		CategoryBrand,
		CategoryMember,
		CategoryOrder,
		CategoryProduct,
		CategoryCampaign,
		CategoryToken,
		CategoryInventory,
		CategoryProduction,
		CategoryAnalytics,
		CategorySystem,
	}
}

// IsValidCategory checks if c is within the allowed categories.
func IsValidCategory(c PermissionCategory) bool {
	switch c {
	case CategoryWallet,
		CategoryInquiry,
		CategoryOrganization,
		CategoryBrand,
		CategoryMember,
		CategoryOrder,
		CategoryProduct,
		CategoryCampaign,
		CategoryToken,
		CategoryInventory,
		CategoryProduction,
		CategoryAnalytics,
		CategorySystem:
		return true
	default:
		return false
	}
}

// Permission mirrors web-app/src/shared/types/permission.ts
// interface Permission { id: string; name: string; description: string; category: PermissionCategory; }
type Permission struct {
	ID          string
	Name        string
	Description string
	Category    PermissionCategory
}

var (
	ErrInvalidID       = errors.New("permission: invalid id")
	ErrInvalidName     = errors.New("permission: invalid name")
	ErrInvalidCategory = errors.New("permission: invalid category")
)

var nameRe = regexp.MustCompile(`^[a-z][a-z0-9.-]*\.[a-z][a-z0-9.-]*$`)

// New creates a Permission with validation.
func New(id, name, description string, category PermissionCategory) (Permission, error) {
	p := Permission{
		ID:          id,
		Name:        name,
		Description: description,
		Category:    category,
	}
	if err := p.validate(); err != nil {
		return Permission{}, err
	}
	return p, nil
}

// UpdateDescription updates the description (no extra validation).
func (p *Permission) UpdateDescription(desc string) {
	p.Description = desc
}

// validate performs basic checks aligned with TS types and typical naming like "brand.create".
func (p Permission) validate() error {
	if p.ID == "" {
		return ErrInvalidID
	}
	if p.Name == "" || !nameRe.MatchString(p.Name) {
		return ErrInvalidName
	}
	if !IsValidCategory(p.Category) {
		return ErrInvalidCategory
	}
	return nil
}

// MustNew panics on validation error (useful for seeding static permissions).
func MustNew(id, name, description string, category PermissionCategory) Permission {
	p, err := New(id, name, description, category)
	if err != nil {
		panic(fmt.Errorf("permission.MustNew: %w", err))
	}
	return p
}

// DDL reference (used by backend/cmd/ddlgen)
const PermissionsTableDDL = `
-- ==========================================
-- Permissions Table Initialization
-- ==========================================

CREATE TABLE IF NOT EXISTS permissions (
  id TEXT PRIMARY KEY,             -- 例: 'perm_001'
  name TEXT NOT NULL UNIQUE,       -- 例: 'brand.create'
  description TEXT NOT NULL,       -- 権限の説明
  category TEXT NOT NULL CHECK (
    category IN (
      'wallet',
      'inquiry',
      'organization',
      'brand',
      'member',
      'order',
      'product',
      'campaign',
      'token',
      'inventory',
      'production',
      'analytics',
      'system'
    )
  ),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ==========================================
-- Indexes
-- ==========================================

-- カテゴリでの絞り込み用
CREATE INDEX IF NOT EXISTS idx_permissions_category ON permissions(category);

-- 名前での検索用
CREATE INDEX IF NOT EXISTS idx_permissions_name ON permissions(name);

-- ==========================================
-- Comments
-- ==========================================

COMMENT ON TABLE permissions IS 'Permission master table defining system permissions';
COMMENT ON COLUMN permissions.id IS 'Primary key, e.g. perm_001';
COMMENT ON COLUMN permissions.name IS 'Unique permission key, e.g. brand.create';
COMMENT ON COLUMN permissions.category IS 'Permission category (wallet, brand, token, etc.)';
`
