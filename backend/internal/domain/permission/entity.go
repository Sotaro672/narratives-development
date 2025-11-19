// backend/internal/domain/permission/entity.go
package permission

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// PermissionCategory mirrors frontend shared types
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

// Permission mirrors frontend Permission model:
//
//   - ID: arbitrary unique identifier
//   - Name: "<category>.<action>" 例: "brand.read"
//   - Description: human readable
//   - Category: one of PermissionCategory
//
// 【前提】ユーザーは「基本閲覧のみ」
// Name の action は read-only に限定（read / list / view / export）
// ※ 書き込み系（create / update / delete / patch / approve など）は禁止
type Permission struct {
	ID          string
	Name        string
	Description string
	Category    PermissionCategory
}

var (
	ErrInvalidID        = errors.New("permission: invalid id")
	ErrInvalidName      = errors.New("permission: invalid name format (expected <category>.<action>)")
	ErrInvalidCategory  = errors.New("permission: invalid category")
	ErrForbiddenAction  = errors.New("permission: action is not allowed for read-only users")
	ErrCategoryMismatch = errors.New("permission: name prefix does not match category")
)

// "<category>.<action>" の基本形だけ先に絞る（英小文字と数字、.-）
var nameShapeRe = regexp.MustCompile(`^[a-z][a-z0-9.-]*\.[a-z][a-z0-9.-]*$`)

// 読み取り専用で許可するアクション
var readOnlyActions = map[string]struct{}{
	"read":   {},
	"list":   {},
	"view":   {},
	"export": {},
}

// New creates a Permission with validation (read-only enforced).
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

// UpdateDescription updates only the description.
func (p *Permission) UpdateDescription(desc string) {
	p.Description = desc
}

// validate performs checks aligned with read-only policy.
func (p Permission) validate() error {
	if strings.TrimSpace(p.ID) == "" {
		return ErrInvalidID
	}
	if strings.TrimSpace(p.Name) == "" || !nameShapeRe.MatchString(p.Name) {
		return ErrInvalidName
	}
	if !IsValidCategory(p.Category) {
		return ErrInvalidCategory
	}

	// "<category>.<action>" に分解
	i := strings.LastIndexByte(p.Name, '.')
	if i <= 0 || i == len(p.Name)-1 {
		return ErrInvalidName
	}
	prefix := p.Name[:i]
	action := p.Name[i+1:]

	// カテゴリ一致（例: CategoryBrand → "brand.*"）
	expectedPrefix := string(p.Category)
	// prefix は "brand" or "brand.something" を許可（サブ領域を使いたい場合に対応）
	if prefix != expectedPrefix && !strings.HasPrefix(prefix, expectedPrefix+".") {
		return ErrCategoryMismatch
	}

	// 読み取り専用アクションのみ許可
	if _, ok := readOnlyActions[action]; !ok {
		return ErrForbiddenAction
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

// IsReadOnlyAction reports whether the action part is read-only.
func IsReadOnlyAction(name string) bool {
	i := strings.LastIndexByte(name, '.')
	if i <= 0 || i == len(name)-1 {
		return false
	}
	action := name[i+1:]
	_, ok := readOnlyActions[action]
	return ok
}
