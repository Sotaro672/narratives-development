package company

import (
	"errors"
	"strings"
	"time"
)

// Domain errors
var (
	ErrInvalidID         = errors.New("company: invalid id")
	ErrInvalidName       = errors.New("company: invalid name")
	ErrInvalidAdmin      = errors.New("company: invalid admin")
	ErrInvalidCreatedAt  = errors.New("company: invalid createdAt")
	ErrInvalidUpdatedAt  = errors.New("company: invalid updatedAt")
	ErrInvalidDeletedAt  = errors.New("company: invalid deletedAt")
	ErrInvalidCreatedBy  = errors.New("company: invalid createdBy")
	ErrInvalidUpdatedBy  = errors.New("company: invalid updatedBy")
	ErrInvalidDeletedBy  = errors.New("company: invalid deletedBy")
)

// Company mirrors web-app/src/shared/types
// export interface Company {
//   id: string;
//   name: string;
//   admin: string; //root権限を持ったmemberId
//   isActive: boolean;
//   createdAt: Date | string;
//   createdBy: string;
//   updatedAt: Date | string;
//   updatedBy: string;
//   deletedAt?: Date | string;
//   deletedBy?: string;
// }
type Company struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Admin     string     `json:"admin"` // root権限を持ったmemberId
	IsActive  bool       `json:"isActive"`

	CreatedAt time.Time  `json:"createdAt"`
	CreatedBy string     `json:"createdBy"`
	UpdatedAt time.Time  `json:"updatedAt"`
	UpdatedBy string     `json:"updatedBy"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
	DeletedBy *string    `json:"deletedBy,omitempty"`
}

/*
Constructors
*/

// NewCompany constructs a Company with validation.
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

// NewCompanyWithNow sets createdAt/updatedAt to now (UTC).
func NewCompanyWithNow(
	id, name, admin, createdBy, updatedBy string,
	isActive bool,
	now time.Time,
) (Company, error) {
	now = now.UTC()
	return NewCompany(id, name, admin, createdBy, updatedBy, now, now, isActive, nil, nil)
}

/*
Behavior
*/

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
	if name == "" {
		return ErrInvalidName
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

/*
Validation
*/

func (c Company) validate() error {
	if strings.TrimSpace(c.ID) == "" {
		return ErrInvalidID
	}
	if strings.TrimSpace(c.Name) == "" {
		return ErrInvalidName
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
	if c.DeletedAt != nil && c.DeletedAt.Before(c.CreatedAt) {
		return ErrInvalidDeletedAt
	}
	if c.DeletedAt != nil && c.UpdatedAt.After(*c.DeletedAt) {
		return ErrInvalidDeletedAt
	}
	if c.DeletedBy != nil && strings.TrimSpace(*c.DeletedBy) == "" {
		return ErrInvalidDeletedBy
	}
	return nil
}

// validateUpdateOnly keeps update invariants minimal (used by mutators)
func (c Company) validateUpdateOnly() error {
	if strings.TrimSpace(c.UpdatedBy) == "" {
		return ErrInvalidUpdatedBy
	}
	if c.UpdatedAt.IsZero() || c.UpdatedAt.Before(c.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	return nil
}

/*
Helpers
*/

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

/*
SQL DDL
*/

const CompaniesTableDDL = `
CREATE TABLE IF NOT EXISTS companies (
  id UUID PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  admin UUID NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_by UUID NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_by UUID NOT NULL,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by UUID NULL,
  CONSTRAINT fk_companies_admin
      FOREIGN KEY (admin)
      REFERENCES members(id)
      ON UPDATE CASCADE
      ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS company_update_logs (
  log_id BIGSERIAL PRIMARY KEY,
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  changed_fields JSONB NOT NULL,
  updated_by UUID,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  operation_type VARCHAR(20) NOT NULL CHECK (operation_type IN ('CREATE','UPDATE','DELETE'))
);

CREATE OR REPLACE FUNCTION trg_companies_update()
RETURNS TRIGGER AS $$
DECLARE
  diff JSONB := '{}'::jsonb;
BEGIN
  IF NEW.name IS DISTINCT FROM OLD.name THEN
    diff := jsonb_set(diff, '{name}', jsonb_build_object('old', OLD.name, 'new', NEW.name));
  END IF;
  IF NEW.admin IS DISTINCT FROM OLD.admin THEN
    diff := jsonb_set(diff, '{admin}', jsonb_build_object('old', OLD.admin, 'new', NEW.admin));
  END IF;
  IF NEW.is_active IS DISTINCT FROM OLD.is_active THEN
    diff := jsonb_set(diff, '{is_active}', jsonb_build_object('old', OLD.is_active, 'new', NEW.is_active));
  END IF;

  NEW.updated_at := NOW();
  NEW.updated_by := COALESCE(current_setting('app.user_id', true)::uuid, NEW.updated_by);

  INSERT INTO company_update_logs(company_id, changed_fields, updated_by, updated_at, operation_type)
  VALUES (OLD.id, diff, NEW.updated_by, NEW.updated_at, 'UPDATE');

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER companies_update_trg
BEFORE UPDATE ON companies
FOR EACH ROW
WHEN (OLD IS DISTINCT FROM NEW)
EXECUTE FUNCTION trg_companies_update();

CREATE OR REPLACE FUNCTION trg_companies_insert()
RETURNS TRIGGER AS $$
BEGIN
  INSERT INTO company_update_logs(company_id, changed_fields, updated_by, updated_at, operation_type)
  VALUES (NEW.id, '{}'::jsonb, NEW.created_by, NEW.created_at, 'CREATE');
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER companies_insert_trg
AFTER INSERT ON companies
FOR EACH ROW
EXECUTE FUNCTION trg_companies_insert();
`
