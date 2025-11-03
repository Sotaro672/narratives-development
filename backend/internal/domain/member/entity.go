package member

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	permdom "narratives/internal/domain/permission"
)

// MemberRole mirrors web-app/src/shared/types/member.ts
// Use a type alias so string can be assigned directly from adapters.
type MemberRole = string

const (
	RoleAdmin              MemberRole = "admin"
	RoleBrandManager       MemberRole = "brand-manager"
	RoleTokenManager       MemberRole = "token-manager"
	RoleInquiryHandler     MemberRole = "inquiry-handler"
	RoleProductionDesigner MemberRole = "production-designer"
)

func IsValidRole(r MemberRole) bool {
	switch r {
	case RoleAdmin, RoleBrandManager, RoleTokenManager, RoleInquiryHandler, RoleProductionDesigner:
		return true
	default:
		return false
	}
}

var emailRe = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

type Member struct {
	ID             string     `json:"id"`
	FirstName      string     `json:"first_name,omitempty"`
	LastName       string     `json:"last_name,omitempty"`
	FirstNameKana  string     `json:"first_name_kana,omitempty"`
	LastNameKana   string     `json:"last_name_kana,omitempty"`
	Email          string     `json:"email,omitempty"` // optional in TS; empty string means unset
	Role           MemberRole `json:"role"`
	Permissions    []string   `json:"permissions"`
	AssignedBrands []string   `json:"assignedBrands,omitempty"`

	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy *string    `json:"updatedBy,omitempty"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
	DeletedBy *string    `json:"deletedBy,omitempty"`
}

var (
	ErrInvalidID          = errors.New("member: invalid id")
	ErrInvalidRole        = errors.New("member: invalid role")
	ErrInvalidEmail       = errors.New("member: invalid email")
	ErrInvalidCreatedAt   = errors.New("member: invalid createdAt")
	ErrInvalidUpdatedAt   = errors.New("member: invalid updatedAt")
	ErrInvalidUpdatedBy   = errors.New("member: invalid updatedBy")
	ErrInvalidDeletedAt   = errors.New("member: invalid deletedAt")
	ErrInvalidDeletedBy   = errors.New("member: invalid deletedBy")
	ErrNotFound           = errors.New("member: not found")
	ErrConflict           = errors.New("member: conflict")
	ErrPreconditionFailed = errors.New("member: precondition failed") // 競合（楽観ロック等）
)

// New constructs a Member with validation. Use empty strings/slices for optional fields.
func New(
	id string,
	role MemberRole,
	createdAt time.Time,
	opts ...func(*Member),
) (Member, error) {
	m := Member{
		ID:          id,
		Role:        role,
		Permissions: nil,
		CreatedAt:   createdAt,
	}
	for _, opt := range opts {
		opt(&m)
	}
	m.dedupAll()
	if err := m.validate(); err != nil {
		return Member{}, err
	}
	return m, nil
}

// NewFromStringsTime accepts createdAt/updatedAt as string (ISO8601). Empty updatedAt means unset.
func NewFromStringsTime(
	id string,
	role MemberRole,
	createdAt string,
	updatedAt string, // optional; pass "" if none
	opts ...func(*Member),
) (Member, error) {
	ct, err := parseTime(createdAt)
	if err != nil {
		return Member{}, fmt.Errorf("%w: %v", ErrInvalidCreatedAt, err)
	}
	var utPtr *time.Time
	if updatedAt != "" {
		if ut, err := parseTime(updatedAt); err == nil {
			utPtr = &ut
		} else {
			utPtr = nil
		}
	}
	m, err := New(id, role, ct, opts...)
	if err != nil {
		return Member{}, err
	}
	m.UpdatedAt = utPtr
	return m, nil
}

// Option helpers to set optional fields

func WithName(first, last string) func(*Member) {
	return func(m *Member) {
		m.FirstName, m.LastName = first, last
	}
}

func WithNameKana(firstKana, lastKana string) func(*Member) {
	return func(m *Member) {
		m.FirstNameKana, m.LastNameKana = firstKana, lastKana
	}
}

func WithEmail(email string) func(*Member) {
	return func(m *Member) {
		m.Email = email
	}
}

func WithPermissions(permissions []string) func(*Member) {
	return func(m *Member) {
		m.Permissions = append([]string(nil), permissions...)
	}
}

func WithAssignedBrands(brands []string) func(*Member) {
	return func(m *Member) {
		m.AssignedBrands = append([]string(nil), brands...)
	}
}

func WithUpdated(by string, at time.Time) func(*Member) {
	return func(m *Member) {
		b := by
		t := at
		m.UpdatedBy = &b
		m.UpdatedAt = &t
	}
}

func WithDeleted(by string, at time.Time) func(*Member) {
	return func(m *Member) {
		b := by
		t := at
		m.DeletedBy = &b
		m.DeletedAt = &t
	}
}

// Mutators

func (m *Member) UpdateEmail(email string, now time.Time) error {
	if email != "" && !emailRe.MatchString(email) {
		return ErrInvalidEmail
	}
	m.Email = email
	m.touch(now)
	return nil
}

func (m *Member) UpdateRole(role MemberRole, now time.Time) error {
	if !IsValidRole(role) {
		return ErrInvalidRole
	}
	m.Role = role
	m.touch(now)
	return nil
}

func (m *Member) AssignBrand(id string, now time.Time) {
	if id == "" {
		return
	}
	if !contains(m.AssignedBrands, id) {
		m.AssignedBrands = append(m.AssignedBrands, id)
		m.touch(now)
	}
}

func (m *Member) UnassignBrand(id string, now time.Time) {
	if id == "" {
		return
	}
	m.AssignedBrands = remove(m.AssignedBrands, id)
	m.touch(now)
}

// TouchUpdated sets UpdatedAt and optionally UpdatedBy.
func (m *Member) TouchUpdated(now time.Time, by *string) error {
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}
	t := now
	m.UpdatedAt = &t
	if by != nil {
		if *by == "" {
			return ErrInvalidUpdatedBy
		}
		b := *by
		m.UpdatedBy = &b
	}
	return nil
}

func (m *Member) MarkDeleted(now time.Time, by *string) error {
	if now.IsZero() {
		return ErrInvalidDeletedAt
	}
	t := now
	m.DeletedAt = &t
	if by != nil {
		if *by == "" {
			return ErrInvalidDeletedBy
		}
		b := *by
		m.DeletedBy = &b
	}
	return nil
}

func (m *Member) ClearDeleted() {
	m.DeletedAt = nil
	m.DeletedBy = nil
}

// Validation and helpers

func (m Member) validate() error {
	if m.ID == "" {
		return ErrInvalidID
	}
	if !IsValidRole(m.Role) {
		return ErrInvalidRole
	}
	if m.Email != "" && !emailRe.MatchString(m.Email) {
		return ErrInvalidEmail
	}
	if m.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if m.UpdatedAt != nil && m.UpdatedAt.Before(m.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	if m.UpdatedBy != nil && *m.UpdatedBy == "" {
		return ErrInvalidUpdatedBy
	}
	if m.DeletedAt != nil && m.DeletedAt.Before(m.CreatedAt) {
		return ErrInvalidDeletedAt
	}
	if m.DeletedBy != nil && *m.DeletedBy == "" {
		return ErrInvalidDeletedBy
	}
	return nil
}

func (m *Member) touch(now time.Time) {
	m.UpdatedAt = &now
}

func (m *Member) dedupAll() {
	m.Permissions = dedup(m.Permissions)
	m.AssignedBrands = dedup(m.AssignedBrands)
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
		return time.Time{}, errors.New("empty time")
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time: %q", s)
}

// MemberPatch represents partial updates for Member.
// nil fields are ignored by repositories.
type MemberPatch struct {
	FirstName      *string
	LastName       *string
	FirstNameKana  *string
	LastNameKana   *string
	Email          *string
	Role           *MemberRole
	Permissions    *[]string
	AssignedBrands *[]string

	CreatedAt *time.Time
	UpdatedAt *time.Time
	UpdatedBy *string
	DeletedAt *time.Time
	DeletedBy *string
}

// DDL reference (for schema alignment with migrations)
const MembersTableDDL = `
CREATE TABLE members (
  id UUID PRIMARY KEY,
  first_name VARCHAR(100),
  last_name VARCHAR(100),
  first_name_kana VARCHAR(100),
  last_name_kana VARCHAR(100),
  email VARCHAR(255) UNIQUE,
  role VARCHAR(50) NOT NULL,
  authorizations TEXT[] NOT NULL,
  assigned_brands TEXT[],

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ,
  updated_by TEXT,
  deleted_at TIMESTAMPTZ,
  deleted_by TEXT
);
`

// SetPermissionsByName はカタログに存在する Permission.Name のみを設定します（重複排除・整列）。
func (m *Member) SetPermissionsByName(names []string, catalog []permdom.Permission) error {
	if m == nil {
		return nil
	}
	allow := make(map[string]struct{}, len(catalog))
	for _, p := range catalog {
		n := strings.TrimSpace(p.Name)
		if n != "" {
			allow[n] = struct{}{}
		}
	}
	seen := make(map[string]struct{}, len(names))
	out := make([]string, 0, len(names))
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if _, ok := allow[n]; !ok {
			// カタログ外はスキップ（必要であればエラーに変更可）
			continue
		}
		if _, dup := seen[n]; dup {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	sort.Strings(out)
	m.Permissions = out
	return nil
}

// ValidatePermissions は現在の Permissions がカタログに含まれるか検証します。
func (m Member) ValidatePermissions(catalog []permdom.Permission) error {
	allow := make(map[string]struct{}, len(catalog))
	for _, p := range catalog {
		n := strings.TrimSpace(p.Name)
		if n != "" {
			allow[n] = struct{}{}
		}
	}
	for _, n := range m.Permissions {
		if _, ok := allow[strings.TrimSpace(n)]; !ok {
			return errors.New("member: permission not found in catalog: " + n)
		}
	}
	return nil
}

// HasPermission は指定 Permission.Name を保持しているかを返します。
func (m Member) HasPermission(name string) bool {
	name = strings.TrimSpace(name)
	for _, n := range m.Permissions {
		if strings.EqualFold(n, name) {
			return true
		}
	}
	return false
}
