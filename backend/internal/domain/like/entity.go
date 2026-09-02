// backend/internal/domain/like/entity.go
package like

import (
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const MaxReferenceIDLength = 128

var (
	ErrNotFound = errors.New("like: not found")
	ErrConflict = errors.New("like: conflict")
	ErrInvalid  = errors.New("like: invalid")
	ErrInternal = errors.New("like: internal")

	ErrInvalidAvatarID   = errors.New("like: invalid avatarId")
	ErrInvalidTargetType = errors.New("like: invalid targetType")
	ErrInvalidTargetID   = errors.New("like: invalid targetId")
	ErrInvalidCreatedAt  = errors.New("like: invalid createdAt")
)

// TargetType represents the kind of resource saved as a favorite.
//
// A Like may currently target:
// - list: primary-sale catalog/list
// - resale: secondary-market resale listing
type TargetType string

const (
	TargetTypeList   TargetType = "list"
	TargetTypeResale TargetType = "resale"
)

func IsValidTargetType(targetType TargetType) bool {
	switch targetType {
	case TargetTypeList, TargetTypeResale:
		return true
	default:
		return false
	}
}

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func IsConflict(err error) bool {
	return errors.Is(err, ErrConflict)
}

func IsInvalid(err error) bool {
	return errors.Is(err, ErrInvalid) ||
		errors.Is(err, ErrInvalidAvatarID) ||
		errors.Is(err, ErrInvalidTargetType) ||
		errors.Is(err, ErrInvalidTargetID) ||
		errors.Is(err, ErrInvalidCreatedAt)
}

func IsInternal(err error) bool {
	return errors.Is(err, ErrInternal)
}

// Like represents one favorite saved by one avatar.
//
// Recommended Firestore mapping:
//
// likes/{avatarId}/items/{documentId}
//
// documentId:
// - list_{listId}
// - resale_{resaleId}
//
// Example:
//
// likes/avatar-001/items/list_list-001
//
//	avatarId: "avatar-001"
//	targetType: "list"
//	targetId: "list-001"
//	createdAt: ...
//
// likes/avatar-001/items/resale_resale-001
//
//	avatarId: "avatar-001"
//	targetType: "resale"
//	targetId: "resale-001"
//	createdAt: ...
//
// The deterministic document ID guarantees that one avatar can have
// at most one Like for the same target.
type Like struct {
	AvatarID   string     `json:"avatarId"`
	TargetType TargetType `json:"targetType"`
	TargetID   string     `json:"targetId"`
	CreatedAt  time.Time  `json:"createdAt"`
}

type NewLikeParams struct {
	AvatarID   string
	TargetType TargetType
	TargetID   string
	Now        time.Time
}

func NewLike(params NewLikeParams) (*Like, error) {
	avatarID := strings.TrimSpace(params.AvatarID)
	targetID := strings.TrimSpace(params.TargetID)

	if !isValidReferenceID(avatarID) {
		return nil, ErrInvalidAvatarID
	}

	if !IsValidTargetType(params.TargetType) {
		return nil, ErrInvalidTargetType
	}

	if !isValidReferenceID(targetID) {
		return nil, ErrInvalidTargetID
	}

	now := params.Now
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	entity := &Like{
		AvatarID:   avatarID,
		TargetType: params.TargetType,
		TargetID:   targetID,
		CreatedAt:  now,
	}

	if err := entity.Validate(); err != nil {
		return nil, err
	}

	return entity, nil
}

func RestoreLike(
	avatarID string,
	targetType TargetType,
	targetID string,
	createdAt time.Time,
) (*Like, error) {
	entity := &Like{
		AvatarID:   strings.TrimSpace(avatarID),
		TargetType: targetType,
		TargetID:   strings.TrimSpace(targetID),
		CreatedAt:  createdAt.UTC(),
	}

	if err := entity.Validate(); err != nil {
		return nil, err
	}

	return entity, nil
}

func (l Like) Validate() error {
	if !isValidReferenceID(l.AvatarID) {
		return ErrInvalidAvatarID
	}

	if !IsValidTargetType(l.TargetType) {
		return ErrInvalidTargetType
	}

	if !isValidReferenceID(l.TargetID) {
		return ErrInvalidTargetID
	}

	if l.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}

	return nil
}

// DocumentID returns the deterministic document ID used below:
//
// likes/{avatarId}/items/{documentId}
func (l Like) DocumentID() (string, error) {
	if !IsValidTargetType(l.TargetType) {
		return "", ErrInvalidTargetType
	}

	if !isValidReferenceID(l.TargetID) {
		return "", ErrInvalidTargetID
	}

	return string(l.TargetType) + "_" + l.TargetID, nil
}

func (l Like) IsList() bool {
	return l.TargetType == TargetTypeList
}

func (l Like) IsResale() bool {
	return l.TargetType == TargetTypeResale
}

func isValidReferenceID(value string) bool {
	if value == "" {
		return false
	}

	if !utf8.ValidString(value) {
		return false
	}

	if len(value) > MaxReferenceIDLength {
		return false
	}

	if strings.Contains(value, "/") {
		return false
	}

	firstRune, _ := utf8.DecodeRuneInString(value)
	lastRune, _ := utf8.DecodeLastRuneInString(value)

	if unicode.IsSpace(firstRune) || unicode.IsSpace(lastRune) {
		return false
	}

	return true
}
