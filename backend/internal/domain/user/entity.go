// backend\internal\domain\user\entity.go
package user

import (
	"errors"
	"time"
)

// User mirrors web-app/src/shared/types/user.ts
// TS fields:
// - id: string
// - first_name: string
// - first_name_kana: string
// - last_name_kana: string
// - last_name: string
// - createdAt: Date | string
// - updatedAt: Date | string
type User struct {
	ID            string    `json:"id"`
	FirstName     string    `json:"first_name"`
	FirstNameKana string    `json:"first_name_kana"`
	LastNameKana  string    `json:"last_name_kana"`
	LastName      string    `json:"last_name"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
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
)

// Policy
var (
	MaxNameLength = 100
)

func validateOptionalName(v string, err error) error {
	if len([]rune(v)) > MaxNameLength {
		return err
	}

	return nil
}

// Mutators
func (u *User) SetFirstName(v string) error {
	if err := validateOptionalName(v, ErrInvalidFirstName); err != nil {
		return err
	}

	u.FirstName = v
	return nil
}

func (u *User) SetFirstNameKana(v string) error {
	if err := validateOptionalName(v, ErrInvalidFirstNameKana); err != nil {
		return err
	}

	u.FirstNameKana = v
	return nil
}

func (u *User) SetLastName(v string) error {
	if err := validateOptionalName(v, ErrInvalidLastName); err != nil {
		return err
	}

	u.LastName = v
	return nil
}

func (u *User) SetLastNameKana(v string) error {
	if err := validateOptionalName(v, ErrInvalidLastNameKana); err != nil {
		return err
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

	if err := validateOptionalName(u.FirstName, ErrInvalidFirstName); err != nil {
		return err
	}

	if err := validateOptionalName(u.FirstNameKana, ErrInvalidFirstNameKana); err != nil {
		return err
	}

	if err := validateOptionalName(u.LastNameKana, ErrInvalidLastNameKana); err != nil {
		return err
	}

	if err := validateOptionalName(u.LastName, ErrInvalidLastName); err != nil {
		return err
	}

	// created/updated must be set by server
	if u.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}

	if u.UpdatedAt.IsZero() || u.UpdatedAt.Before(u.CreatedAt) {
		return ErrInvalidUpdatedAt
	}

	return nil
}

// Constructors
func New(
	id string,
	firstName, firstNameKana, lastNameKana, lastName string,
	createdAt, updatedAt time.Time,
) (User, error) {
	u := User{
		ID:            id,
		FirstName:     firstName,
		FirstNameKana: firstNameKana,
		LastNameKana:  lastNameKana,
		LastName:      lastName,
		CreatedAt:     createdAt.UTC(),
		UpdatedAt:     updatedAt.UTC(),
	}

	if err := u.validate(); err != nil {
		return User{}, err
	}

	return u, nil
}

// NewWithNow is convenient for CreateUserInput (server sets created/updated).
func NewWithNow(
	id string,
	firstName, firstNameKana, lastNameKana, lastName string,
	now time.Time,
) (User, error) {
	now = now.UTC()

	return New(id, firstName, firstNameKana, lastNameKana, lastName, now, now)
}
