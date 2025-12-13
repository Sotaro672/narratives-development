package company

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// ---------------------------
// 正規表現
// ---------------------------
//
// 会社名：漢字/ひらがな/カタカナ/英数字(半角/全角)/長音/スペース
// - 追加: 全角数字 ０-９ を許可
var companyNameRe = regexp.MustCompile(`^[\p{Han}\p{Hiragana}\p{Katakana}A-Za-z0-9０-９ー\s]+$`)

// ---------------------------
// Domain errors
// ---------------------------

var (
	ErrInvalidID        = errors.New("company: invalid id")
	ErrInvalidName      = errors.New("company: invalid name")
	ErrInvalidAdmin     = errors.New("company: invalid admin")
	ErrInvalidCreatedAt = errors.New("company: invalid createdAt")
	ErrInvalidUpdatedAt = errors.New("company: invalid updatedAt")
	ErrInvalidDeletedAt = errors.New("company: invalid deletedAt")
	ErrInvalidCreatedBy = errors.New("company: invalid createdBy")
	ErrInvalidUpdatedBy = errors.New("company: invalid updatedBy")
	ErrInvalidDeletedBy = errors.New("company: invalid deletedBy")
)

// ----------------------------------------
// Company entity
// ----------------------------------------

type Company struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Admin    string `json:"admin"` // root権限を持ったmemberId
	IsActive bool   `json:"isActive"`

	CreatedAt time.Time  `json:"createdAt"`
	CreatedBy string     `json:"createdBy"`
	UpdatedAt time.Time  `json:"updatedAt"`
	UpdatedBy string     `json:"updatedBy"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
	DeletedBy *string    `json:"deletedBy,omitempty"`
}

// ----------------------------------------
// Constructor
// ----------------------------------------

func NewCompany(
	id, name, admin, createdBy, updatedBy string,
	createdAt, updatedAt time.Time,
	isActive bool,
	deletedAt *time.Time,
	deletedBy *string,
) (Company, error) {
	c := Company{
		ID:        strings.TrimSpace(id),
		Name:      strings.TrimSpace(name),
		Admin:     strings.TrimSpace(admin),
		IsActive:  isActive,
		CreatedAt: createdAt.UTC(),
		CreatedBy: strings.TrimSpace(createdBy),
		UpdatedAt: updatedAt.UTC(),
		UpdatedBy: strings.TrimSpace(updatedBy),
		DeletedAt: normalizeTimePtr(deletedAt),
		DeletedBy: normalizeStrPtr(deletedBy),
	}

	if err := c.validate(); err != nil {
		return Company{}, err
	}
	return c, nil
}

func NewCompanyWithNow(
	id, name, admin, createdBy, updatedBy string,
	isActive bool,
	now time.Time,
) (Company, error) {
	now = now.UTC()
	return NewCompany(id, name, admin, createdBy, updatedBy, now, now, isActive, nil, nil)
}

// ----------------------------------------
// Behavior
// ----------------------------------------

func (c *Company) Activate(now time.Time, updatedBy string) error {
	c.IsActive = true
	c.UpdatedAt = now.UTC()
	c.UpdatedBy = strings.TrimSpace(updatedBy)
	return c.validateUpdateOnly()
}

func (c *Company) Deactivate(now time.Time, updatedBy string) error {
	c.IsActive = false
	c.UpdatedAt = now.UTC()
	c.UpdatedBy = strings.TrimSpace(updatedBy)
	return c.validateUpdateOnly()
}

func (c *Company) UpdateName(name string, now time.Time, updatedBy string) error {
	name = strings.TrimSpace(name)
	if err := validateCompanyName(name); err != nil {
		return err
	}
	c.Name = name
	c.UpdatedAt = now.UTC()
	c.UpdatedBy = strings.TrimSpace(updatedBy)
	return c.validateUpdateOnly()
}

func (c *Company) UpdateAdmin(admin string, now time.Time, updatedBy string) error {
	admin = strings.TrimSpace(admin)
	if admin == "" {
		return ErrInvalidAdmin
	}
	c.Admin = admin
	c.UpdatedAt = now.UTC()
	c.UpdatedBy = strings.TrimSpace(updatedBy)
	return c.validateUpdateOnly()
}

func (c *Company) SetDeleted(at *time.Time, by *string) error {
	at = normalizeTimePtr(at)
	by = normalizeStrPtr(by)

	if at == nil {
		c.DeletedAt = nil
		c.DeletedBy = nil
		return nil
	}

	if c.UpdatedAt.After(*at) {
		return ErrInvalidDeletedAt
	}

	c.DeletedAt = at
	c.DeletedBy = by

	if c.DeletedBy != nil && strings.TrimSpace(*c.DeletedBy) == "" {
		return ErrInvalidDeletedBy
	}
	return nil
}

// ----------------------------------------
// Validation
// ----------------------------------------

func (c Company) validate() error {
	if strings.TrimSpace(c.ID) == "" {
		return ErrInvalidID
	}

	if err := validateCompanyName(strings.TrimSpace(c.Name)); err != nil {
		return err
	}

	if strings.TrimSpace(c.Admin) == "" {
		return ErrInvalidAdmin
	}

	if c.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}

	if strings.TrimSpace(c.CreatedBy) == "" {
		return ErrInvalidCreatedBy
	}

	if c.UpdatedAt.IsZero() {
		return ErrInvalidUpdatedAt
	}

	if strings.TrimSpace(c.UpdatedBy) == "" {
		return ErrInvalidUpdatedBy
	}

	if c.UpdatedAt.Before(c.CreatedAt) {
		return ErrInvalidUpdatedAt
	}

	if c.DeletedAt != nil {
		if c.DeletedAt.Before(c.CreatedAt) {
			return ErrInvalidDeletedAt
		}
		if c.UpdatedAt.After(*c.DeletedAt) {
			return ErrInvalidDeletedAt
		}
		if c.DeletedBy != nil && strings.TrimSpace(*c.DeletedBy) == "" {
			return ErrInvalidDeletedBy
		}
	}

	return nil
}

func (c Company) validateUpdateOnly() error {
	if strings.TrimSpace(c.UpdatedBy) == "" {
		return ErrInvalidUpdatedBy
	}
	if c.UpdatedAt.IsZero() || c.UpdatedAt.Before(c.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	return nil
}

// ----------------------------------------
// Helpers
// ----------------------------------------

func validateCompanyName(name string) error {
	if name == "" {
		return ErrInvalidName
	}
	// ★ rune 数で 100 文字制限（日本語でも意図通り）
	if utf8.RuneCountInString(name) > 100 {
		return ErrInvalidName
	}
	// ★ 全角数字も許可
	if !companyNameRe.MatchString(name) {
		return ErrInvalidName
	}
	return nil
}

func normalizeStrPtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil
	}
	return &v
}

func normalizeTimePtr(p *time.Time) *time.Time {
	if p == nil || p.IsZero() {
		return nil
	}
	t := p.UTC()
	return &t
}
