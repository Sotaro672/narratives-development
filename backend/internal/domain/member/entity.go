// backend/internal/domain/member/entity.go
package member

import (
	"errors"
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
//
// NOTE:
//   - Firestore document ID is the internal member record ID.
//   - UID is the Firebase Auth UID.
//   - Invitation flow may create a member document before Firebase Auth UID is known.
//     In that case UID is set later after the invited user signs up/signs in.
type Member struct {
	UID string `json:"uid,omitempty" firestore:"uid"`

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
	ErrInvalidUID         = errors.New("member: invalid uid")
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
	createdAt time.Time,
	opts ...func(*Member),
) (Member, error) {
	m := Member{
		Permissions: nil,
		CreatedAt:   createdAt,
	}
	for _, opt := range opts {
		opt(&m)
	}
	m.normalize()
	m.dedupAll()

	if err := m.validate(); err != nil {
		return Member{}, err
	}
	return m, nil
}

// -------------------------
// Option helpers
// -------------------------

func WithUID(uid string) func(*Member) {
	return func(m *Member) {
		m.UID = uid
	}
}

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
		m.CompanyID = companyID
	}
}

func WithStatus(status string) func(*Member) {
	return func(m *Member) {
		m.Status = status
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

func (m *Member) BindUID(uid string, now time.Time) error {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return ErrInvalidUID
	}
	m.UID = uid
	m.touch(now)
	return nil
}

func (m *Member) UpdateEmail(email string, now time.Time) error {
	email = strings.TrimSpace(email)
	if email != "" && !emailRe.MatchString(email) {
		return ErrInvalidEmail
	}
	m.Email = email
	m.touch(now)
	return nil
}

func (m *Member) AssignBrand(id string, now time.Time) {
	id = strings.TrimSpace(id)
	if id == "" {
		return
	}
	if !contains(m.AssignedBrands, id) {
		m.AssignedBrands = append(m.AssignedBrands, id)
		m.touch(now)
	}
}

func (m *Member) UnassignBrand(id string, now time.Time) {
	id = strings.TrimSpace(id)
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
		b := strings.TrimSpace(*by)
		if b == "" {
			return ErrInvalidUpdatedBy
		}
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
		b := strings.TrimSpace(*by)
		if b == "" {
			return ErrInvalidDeletedBy
		}
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
	if strings.TrimSpace(m.UID) != m.UID {
		return ErrInvalidUID
	}

	if m.Email != "" && !emailRe.MatchString(m.Email) {
		return ErrInvalidEmail
	}

	// ---- 氏名 ----
	if m.FirstName == "" ||
		!nameRe.MatchString(m.FirstName) || len(m.FirstName) > 50 {
		return ErrInvalidFirstName
	}

	if m.LastName == "" ||
		!nameRe.MatchString(m.LastName) || len(m.LastName) > 50 {
		return ErrInvalidLastName
	}

	// ---- 氏名かな（ひらがなのみ） ----
	if m.FirstNameKana == "" ||
		!kanaRe.MatchString(m.FirstNameKana) || len(m.FirstNameKana) > 50 {
		return ErrInvalidFirstKana
	}

	if m.LastNameKana == "" ||
		!kanaRe.MatchString(m.LastNameKana) || len(m.LastNameKana) > 50 {
		return ErrInvalidLastKana
	}

	// ---- ステータス ----
	switch strings.ToLower(m.Status) {
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
	if m.UpdatedBy != nil && strings.TrimSpace(*m.UpdatedBy) == "" {
		return ErrInvalidUpdatedBy
	}
	if m.DeletedAt != nil && m.DeletedAt.Before(m.CreatedAt) {
		return ErrInvalidDeletedAt
	}
	if m.DeletedBy != nil && strings.TrimSpace(*m.DeletedBy) == "" {
		return ErrInvalidDeletedBy
	}

	return nil
}

// -------------------------
// Helpers
// -------------------------

func (m *Member) normalize() {
	m.UID = strings.TrimSpace(m.UID)
	m.Email = strings.TrimSpace(m.Email)
	m.FirstName = strings.TrimSpace(m.FirstName)
	m.LastName = strings.TrimSpace(m.LastName)
	m.FirstNameKana = strings.TrimSpace(m.FirstNameKana)
	m.LastNameKana = strings.TrimSpace(m.LastNameKana)
	m.CompanyID = strings.TrimSpace(m.CompanyID)
	m.Status = strings.TrimSpace(m.Status)
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
		x = strings.TrimSpace(x)
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

// --------------------------
// MemberPatch
// --------------------------

type MemberPatch struct {
	UID            *string
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
		n := p.Name
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
		n := p.Name
		if n != "" {
			allow[n] = struct{}{}
		}
	}
	for _, n := range m.Permissions {
		if _, ok := allow[n]; !ok {
			return errors.New("member: permission not found in catalog: " + n)
		}
	}
	return nil
}

func (m Member) HasPermission(name string) bool {
	for _, n := range m.Permissions {
		if strings.EqualFold(n, name) {
			return true
		}
	}
	return false
}

type InvitationToken struct {
	Token            string   `firestore:"token"`
	MemberID         string   `firestore:"memberId"`
	CompanyID        string   `firestore:"companyId"`
	AssignedBrandIDs []string `firestore:"assignedBrands"`
	Permissions      []string `firestore:"permissions"`
	Email            string   `firestore:"email"`

	CreatedAt time.Time  `firestore:"createdAt"`
	ExpiresAt *time.Time `firestore:"expiresAt,omitempty"`
	UsedAt    *time.Time `firestore:"usedAt,omitempty"`
	UpdatedAt *time.Time `firestore:"updatedAt,omitempty"`
}

// 招待リンク表示用
type InvitationInfo struct {
	MemberID         string   `json:"memberId"`
	CompanyID        string   `json:"companyId"`
	AssignedBrandIDs []string `json:"assignedBrandIds"`
	Permissions      []string `json:"permissions"`
	Email            string   `json:"email"`
}
