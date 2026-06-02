// backend/internal/domain/avatarIcon/entity.go
package avatarIcon

import (
	"errors"
	"net/url"
)

// Domain errors
var (
	ErrInvalidID  = errors.New("avatarIcon: invalid id")
	ErrInvalidURL = errors.New("avatarIcon: invalid url")
)

// AvatarIcon mirrors web-app/src/shared/types (without audit fields)
type AvatarIcon struct {
	ID       string  `json:"id"`
	AvatarID *string `json:"avatarId,omitempty"`
	URL      string  `json:"url"`
}

/*
Constructors
*/

// New constructs an AvatarIcon with validation.
// Domain layer does not normalize or parse values; it only validates them.
func New(
	id string,
	urlStr string,
	avatarID *string,
) (AvatarIcon, error) {
	a := AvatarIcon{
		ID:       id,
		AvatarID: avatarID,
		URL:      urlStr,
	}
	if err := a.validate(); err != nil {
		return AvatarIcon{}, err
	}
	return a, nil
}

// NewForCreate kept for compatibility (no audit fields anymore).
func NewForCreate(
	id string,
	input struct {
		URL      string
		AvatarID *string
	},
) (AvatarIcon, error) {
	return New(
		id,
		input.URL,
		input.AvatarID,
	)
}

/*
Mutators
*/

func (a *AvatarIcon) UpdateURL(u string) error {
	if err := validateURL(u); err != nil {
		return err
	}
	a.URL = u
	return nil
}

func (a *AvatarIcon) SetAvatarID(v *string) {
	a.AvatarID = v
}

/*
Validation
*/

func (a AvatarIcon) validate() error {
	if a.ID == "" {
		return ErrInvalidID
	}
	if err := validateURL(a.URL); err != nil {
		return err
	}
	return nil
}

/*
Helpers
*/

// validateURL returns domain error on invalid URL.
// Domain layer validates only; it does not expose parse/build helpers.
func validateURL(u string) error {
	if u == "" {
		return ErrInvalidURL
	}
	parsed, err := url.Parse(u)
	if err != nil {
		return ErrInvalidURL
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return ErrInvalidURL
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrInvalidURL
	}
	return nil
}
