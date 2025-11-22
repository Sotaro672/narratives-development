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

// ---------------------------
// 正規表現
// ---------------------------

// Email
var emailRe = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// 氏名：漢字/ひらがな/カタカナ/英字/数字/長音/スペースのみ
var nameRe = regexp.MustCompile(`^[\p{Han}\p{Hiragana}\p{Katakana}A-Za-z0-9ー\s]+$`)

// 氏名かな：ひらがな・スペースのみ
var kanaRe = regexp.MustCompile(`^[\p{Hiragana}\s]+$`)

// Member is the domain entity for a user/member.
type Member struct {
	ID             string   `json:"id" firestore:"id"`
	FirstName      string   `json:"firstName,omitempty" firestore:"firstName"`
	LastName       string   `json:"lastName,omitempty" firestore:"lastName"`
	FirstNameKana  string   `json:"firstNameKana,omitempty" firestore:"firstNameKana"`
	LastNameKana   string   `json:"lastNameKana,omitempty" firestore:"lastNameKana"`
	Email          string   `json:"email,omitempty" firestore:"email"`
	Permissions    []string `json:"permissions" firestore:"permissions"`
	AssignedBrands []string `json:"assignedBrands,omitempty" firestore:"assignedBrands"`

	CompanyID string `json:"companyId,omitempty" firestore:"companyId"`
	Status    string `json:"status,omitempty" firestore:"status"`

	CreatedAt time.Time  `json:"createdAt" firestore:"createdAt"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty" firestore:"updatedAt"`
	UpdatedBy *string    `json:"updatedBy,omitempty" firestore:"updatedBy"`
	DeletedAt *time.Time `json:"deletedAt,omitempty" firestore:"deletedAt"`
	DeletedBy *string    `json:"deletedBy,omitempty" firestore:"deletedBy"`
}

var (
	ErrInvalidID          = errors.New("member: invalid id")
	ErrInvalidEmail       = errors.New("member: invalid email")
	ErrInvalidFirstName   = errors.New("member: invalid firstName")
	ErrInvalidLastName    = errors.New("member: invalid lastName")
	ErrInvalidFirstKana   = errors.New("member: invalid firstNameKana")
	ErrInvalidLastKana    = errors.New("member: invalid lastNameKana")
	ErrInvalidCreatedAt   = errors.New("member: invalid createdAt")
	ErrInvalidUpdatedAt   = errors.New("member: invalid updatedAt")
	ErrInvalidUpdatedBy   = errors.New("member: invalid updatedBy")
	ErrInvalidDeletedAt   = errors.New("member: invalid deletedAt")
	ErrInvalidDeletedBy   = errors.New("member: invalid deletedBy")
	ErrInvalidStatus      = errors.New("member: invalid status")
	ErrNotFound           = errors.New("member: not found")
	ErrConflict           = errors.New("member: conflict")
	ErrPreconditionFailed = errors.New("member: precondition failed")
)

// ----------------------
// Constructor
// ----------------------

func New(
	id string,
	createdAt time.Time,
	opts ...func(*Member),
) (Member, error) {
	m := Member{
		ID:          id,
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

func NewFromStringsTime(
	id string,
	createdAt string,
	updatedAt string,
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
		}
	}

	m, err := New(id, ct, opts...)
	if err != nil {
		return Member{}, err
	}
	m.UpdatedAt = utPtr
	return m, nil
}

// -------------------------
// Option helpers
// -------------------------

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

func WithCompanyID(companyID string) func(*Member) {
	return func(m *Member) {
		m.CompanyID = strings.TrimSpace(companyID)
	}
}

func WithStatus(status string) func(*Member) {
	return func(m *Member) {
		m.Status = strings.TrimSpace(status)
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

// ----------------------
// Mutators
// ----------------------

func (m *Member) UpdateEmail(email string, now time.Time) error {
	if email != "" && !emailRe.MatchString(email) {
		return ErrInvalidEmail
	}
	m.Email = email
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

// -------------------------
// Validation
// -------------------------

func (m Member) validate() error {
	if m.ID == "" {
		return ErrInvalidID
	}

	if m.Email != "" && !emailRe.MatchString(m.Email) {
		return ErrInvalidEmail
	}

	// ---- 氏名 ----
	if strings.TrimSpace(m.FirstName) == "" ||
		!nameRe.MatchString(m.FirstName) || len(m.FirstName) > 50 {
		return ErrInvalidFirstName
	}

	if strings.TrimSpace(m.LastName) == "" ||
		!nameRe.MatchString(m.LastName) || len(m.LastName) > 50 {
		return ErrInvalidLastName
	}

	// ---- 氏名かな（ひらがなのみ） ----
	if strings.TrimSpace(m.FirstNameKana) == "" ||
		!kanaRe.MatchString(m.FirstNameKana) || len(m.FirstNameKana) > 50 {
		return ErrInvalidFirstKana
	}

	if strings.TrimSpace(m.LastNameKana) == "" ||
		!kanaRe.MatchString(m.LastNameKana) || len(m.LastNameKana) > 50 {
		return ErrInvalidLastKana
	}

	// ---- ステータス ----
	switch strings.ToLower(strings.TrimSpace(m.Status)) {
	case "", "active", "inactive":
	default:
		return ErrInvalidStatus
	}

	// ---- 時刻 ----
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

// -------------------------
// Helpers
// -------------------------

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

// --------------------------
// MemberPatch
// --------------------------

type MemberPatch struct {
	FirstName      *string
	LastName       *string
	FirstNameKana  *string
	LastNameKana   *string
	Email          *string
	Permissions    *[]string
	AssignedBrands *[]string
	CompanyID      *string
	Status         *string
	CreatedAt      *time.Time
	UpdatedAt      *time.Time
	UpdatedBy      *string
	DeletedAt      *time.Time
	DeletedBy      *string
}

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

func (m Member) HasPermission(name string) bool {
	name = strings.TrimSpace(name)
	for _, n := range m.Permissions {
		if strings.EqualFold(n, name) {
			return true
		}
	}
	return false
}
