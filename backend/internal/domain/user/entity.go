// backend\internal\domain\user\entity.go
package user

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// User mirrors web-app/src/shared/types/user.ts
// TS fields:
// - id: string
// - first_name?: string
// - first_name_kana?: string
// - last_name_kana?: string
// - last_name?: string
// - createdAt: Date | string
// - updatedAt: Date | string
// - deletedAt: Date | string
type User struct {
	ID            string    `json:"id"`
	FirstName     *string   `json:"first_name,omitempty"`
	FirstNameKana *string   `json:"first_name_kana,omitempty"`
	LastNameKana  *string   `json:"last_name_kana,omitempty"`
	LastName      *string   `json:"last_name,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	DeletedAt     time.Time `json:"deletedAt"`
}

// Errors (single source)
var (
	ErrInvalidID            = errors.New("user: invalid id")
	ErrInvalidFirstName     = errors.New("user: invalid first_name")
	ErrInvalidFirstNameKana = errors.New("user: invalid first_name_kana")
	ErrInvalidLastNameKana  = errors.New("user: invalid last_name_kana")
	ErrInvalidLastName      = errors.New("user: invalid last_name")
	ErrInvalidCreatedAt     = errors.New("user: invalid createdAt")
	ErrInvalidUpdatedAt     = errors.New("user: invalid updatedAt")
	ErrInvalidDeletedAt     = errors.New("user: invalid deletedAt")
)

// Policy
var (
	MaxNameLength = 100
)

// Mutators
func (u *User) SetFirstName(v *string) error {
	v = normalizePtr(v)
	if v != nil && len([]rune(*v)) > MaxNameLength {
		return ErrInvalidFirstName
	}
	u.FirstName = v
	return nil
}

func (u *User) SetFirstNameKana(v *string) error {
	v = normalizePtr(v)
	if v != nil && len([]rune(*v)) > MaxNameLength {
		return ErrInvalidFirstNameKana
	}
	u.FirstNameKana = v
	return nil
}

func (u *User) SetLastName(v *string) error {
	v = normalizePtr(v)
	if v != nil && len([]rune(*v)) > MaxNameLength {
		return ErrInvalidLastName
	}
	u.LastName = v
	return nil
}

func (u *User) SetLastNameKana(v *string) error {
	v = normalizePtr(v)
	if v != nil && len([]rune(*v)) > MaxNameLength {
		return ErrInvalidLastNameKana
	}
	u.LastNameKana = v
	return nil
}

func (u *User) TouchUpdatedAt(now time.Time) error {
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}
	u.UpdatedAt = now.UTC()
	return nil
}

// Validation
func (u User) validate() error {
	if strings.TrimSpace(u.ID) == "" {
		return ErrInvalidID
	}
	if u.FirstName != nil && len([]rune(*u.FirstName)) > MaxNameLength {
		return ErrInvalidFirstName
	}
	if u.FirstNameKana != nil && len([]rune(*u.FirstNameKana)) > MaxNameLength {
		return ErrInvalidFirstNameKana
	}
	if u.LastName != nil && len([]rune(*u.LastName)) > MaxNameLength {
		return ErrInvalidLastName
	}
	if u.LastNameKana != nil && len([]rune(*u.LastNameKana)) > MaxNameLength {
		return ErrInvalidLastNameKana
	}
	if u.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if u.UpdatedAt.IsZero() || u.UpdatedAt.Before(u.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	if u.DeletedAt.IsZero() || u.DeletedAt.Before(u.CreatedAt) {
		return ErrInvalidDeletedAt
	}
	return nil
}

// Constructors

func New(
	id string,
	firstName, firstNameKana, lastNameKana, lastName *string,
	createdAt, updatedAt, deletedAt time.Time,
) (User, error) {
	u := User{
		ID:            strings.TrimSpace(id),
		FirstName:     normalizePtr(firstName),
		FirstNameKana: normalizePtr(firstNameKana),
		LastNameKana:  normalizePtr(lastNameKana),
		LastName:      normalizePtr(lastName),
		CreatedAt:     createdAt.UTC(),
		UpdatedAt:     updatedAt.UTC(),
		DeletedAt:     deletedAt.UTC(),
	}
	if err := u.validate(); err != nil {
		return User{}, err
	}
	return u, nil
}

// NewWithNow is convenient for CreateUserInput (server sets created/updated).
func NewWithNow(
	id string,
	firstName, firstNameKana, lastNameKana, lastName *string,
	now time.Time,
	deletedAt time.Time,
) (User, error) {
	now = now.UTC()
	return New(id, firstName, firstNameKana, lastNameKana, lastName, now, now, deletedAt)
}

// NewFromStringTimes parses createdAt/updatedAt/deletedAt from RFC3339 strings.
func NewFromStringTimes(
	id string,
	firstName, firstNameKana, lastNameKana, lastName *string,
	createdAt, updatedAt, deletedAt string,
) (User, error) {
	ct, err := parseTime(createdAt)
	if err != nil {
		return User{}, fmt.Errorf("%w: %v", ErrInvalidCreatedAt, err)
	}
	ut, err := parseTime(updatedAt)
	if err != nil {
		return User{}, fmt.Errorf("%w: %v", ErrInvalidUpdatedAt, err)
	}
	dt, err := parseTime(deletedAt)
	if err != nil {
		return User{}, fmt.Errorf("%w: %v", ErrInvalidDeletedAt, err)
	}
	return New(id, firstName, firstNameKana, lastNameKana, lastName, ct, ut, dt)
}

// Helpers

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

func parseTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("empty time")
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
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time: %q", s)
}
