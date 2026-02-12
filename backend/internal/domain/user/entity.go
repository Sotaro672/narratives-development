// backend/internal/domain/user/entity.go
package user

import (
	"errors"
	"fmt"
	"time"

	domcommon "narratives/internal/domain/common"
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

	// soft delete: zero means "not deleted"
	DeletedAt time.Time `json:"deletedAt"`
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
	v = domcommon.NormalizeStringPtr(v)
	if v != nil && len([]rune(*v)) > MaxNameLength {
		return ErrInvalidFirstName
	}
	u.FirstName = v
	return nil
}

func (u *User) SetFirstNameKana(v *string) error {
	v = domcommon.NormalizeStringPtr(v)
	if v != nil && len([]rune(*v)) > MaxNameLength {
		return ErrInvalidFirstNameKana
	}
	u.FirstNameKana = v
	return nil
}

func (u *User) SetLastName(v *string) error {
	v = domcommon.NormalizeStringPtr(v)
	if v != nil && len([]rune(*v)) > MaxNameLength {
		return ErrInvalidLastName
	}
	u.LastName = v
	return nil
}

func (u *User) SetLastNameKana(v *string) error {
	v = domcommon.NormalizeStringPtr(v)
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
	if u.ID == "" {
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

	// created/updated must be set by server
	if u.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if u.UpdatedAt.IsZero() || u.UpdatedAt.Before(u.CreatedAt) {
		return ErrInvalidUpdatedAt
	}

	// ✅ soft delete:
	// - DeletedAt is allowed to be zero (not deleted)
	// - If set, it must not be before CreatedAt
	if !u.DeletedAt.IsZero() && u.DeletedAt.Before(u.CreatedAt) {
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
		ID:            id,
		FirstName:     domcommon.NormalizeStringPtr(firstName),
		FirstNameKana: domcommon.NormalizeStringPtr(firstNameKana),
		LastNameKana:  domcommon.NormalizeStringPtr(lastNameKana),
		LastName:      domcommon.NormalizeStringPtr(lastName),
		CreatedAt:     createdAt.UTC(),
		UpdatedAt:     updatedAt.UTC(),
		DeletedAt:     deletedAt.UTC(), // zero stays zero
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
	// deletedAt can be zero (not deleted)
	return New(id, firstName, firstNameKana, lastNameKana, lastName, now, now, deletedAt)
}

// NewFromStringTimes parses createdAt/updatedAt/deletedAt from RFC3339 strings.
func NewFromStringTimes(
	id string,
	firstName, firstNameKana, lastNameKana, lastName *string,
	createdAt, updatedAt, deletedAt string,
) (User, error) {
	ct, err := domcommon.ParseTime(createdAt)
	if err != nil {
		return User{}, fmt.Errorf("%w: %v", ErrInvalidCreatedAt, err)
	}
	ut, err := domcommon.ParseTime(updatedAt)
	if err != nil {
		return User{}, fmt.Errorf("%w: %v", ErrInvalidUpdatedAt, err)
	}

	// ✅ deletedAt は空文字なら "not deleted" 扱いで zero を許容
	var dt time.Time
	if deletedAt != "" {
		parsed, derr := domcommon.ParseTime(deletedAt)
		if derr != nil {
			return User{}, fmt.Errorf("%w: %v", ErrInvalidDeletedAt, derr)
		}
		dt = parsed
	} else {
		dt = time.Time{}
	}

	return New(id, firstName, firstNameKana, lastNameKana, lastName, ct, ut, dt)
}
