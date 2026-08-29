// backend/internal/domain/resale_review/entity.go
package resale_review

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	MaxReferenceIDLength = 128
	MaxCommentBodyLength = 500
)

var (
	ErrNotFound  = errors.New("resaleReview: not found")
	ErrConflict  = errors.New("resaleReview: conflict")
	ErrInvalid   = errors.New("resaleReview: invalid")
	ErrForbidden = errors.New("resaleReview: forbidden")
	ErrInternal  = errors.New("resaleReview: internal")

	ErrInvalidResaleID       = errors.New("resaleReview: invalid resaleId")
	ErrInvalidAvatarID       = errors.New("resaleReview: invalid avatarId")
	ErrInvalidCommentID      = errors.New("resaleReview: invalid commentId")
	ErrInvalidCommentBody    = errors.New("resaleReview: invalid comment body")
	ErrInvalidCreatedAt      = errors.New("resaleReview: invalid createdAt")
	ErrInvalidUpdatedAt      = errors.New("resaleReview: invalid updatedAt")
	ErrInvalidLikeCount      = errors.New("resaleReview: invalid like count")
	ErrInvalidCommentCount   = errors.New("resaleReview: invalid comment count")
	ErrNegativeCounter       = errors.New("resaleReview: counter would become negative")
	ErrDeletedComment        = errors.New("resaleReview: comment is deleted")
	ErrCommentAuthorMismatch = errors.New("resaleReview: comment author mismatch")
)

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func IsConflict(err error) bool {
	return errors.Is(err, ErrConflict)
}

func IsInvalid(err error) bool {
	return errors.Is(err, ErrInvalid)
}

func IsForbidden(err error) bool {
	return errors.Is(err, ErrForbidden)
}

func IsInternal(err error) bool {
	return errors.Is(err, ErrInternal)
}

// ============================================================
// Aggregate
//
// Firestore mapping idea:
//
// resaleReviews/{resaleId}
//   - likeCount
//   - commentCount
//   - createdAt
//   - updatedAt
//
//   likes/{avatarId}
//     - resaleId
//     - avatarId
//     - createdAt
//
//   comments/{commentId}
//     - commentId
//     - resaleId
//     - avatarId
//     - body
//     - deleted
//     - createdAt
//     - updatedAt
//
// NOTE:
// - Authentication / authorization is application.usecase responsibility.
// - Whether the current avatar can interact with the target resale is not
//   decided in this domain.
// - Avatar name / icon are presentation data and are not persisted here.
// ============================================================

type ResaleReviewAggregate struct {
	ResaleID     string
	LikeCount    int64
	CommentCount int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewResaleReviewAggregate(
	resaleID string,
	now time.Time,
) (*ResaleReviewAggregate, error) {
	resaleID = strings.TrimSpace(resaleID)

	if !isValidReferenceID(resaleID) {
		return nil, ErrInvalidResaleID
	}

	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	return &ResaleReviewAggregate{
		ResaleID:     resaleID,
		LikeCount:    0,
		CommentCount: 0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (r ResaleReviewAggregate) Validate() error {
	if !isValidReferenceID(r.ResaleID) {
		return ErrInvalidResaleID
	}

	if r.LikeCount < 0 {
		return ErrInvalidLikeCount
	}

	if r.CommentCount < 0 {
		return ErrInvalidCommentCount
	}

	if r.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}

	if r.UpdatedAt.IsZero() || r.UpdatedAt.Before(r.CreatedAt) {
		return ErrInvalidUpdatedAt
	}

	return nil
}

func (r *ResaleReviewAggregate) IncrementLikeCount(now time.Time) {
	r.LikeCount++
	r.touch(now)
}

func (r *ResaleReviewAggregate) DecrementLikeCount(now time.Time) error {
	if r.LikeCount <= 0 {
		return ErrNegativeCounter
	}

	r.LikeCount--
	r.touch(now)

	return nil
}

func (r *ResaleReviewAggregate) IncrementCommentCount(now time.Time) {
	r.CommentCount++
	r.touch(now)
}

func (r *ResaleReviewAggregate) DecrementCommentCount(now time.Time) error {
	if r.CommentCount <= 0 {
		return ErrNegativeCounter
	}

	r.CommentCount--
	r.touch(now)

	return nil
}

func (r *ResaleReviewAggregate) touch(now time.Time) {
	if now.IsZero() {
		now = time.Now().UTC()
	}

	r.UpdatedAt = now.UTC()
}

// ============================================================
// Like
//
// One avatar can have at most one Like for one resale.
//
// Recommended Firestore document:
//
// resaleReviews/{resaleId}/likes/{avatarId}
//
// Using avatarId as document ID naturally enforces uniqueness.
// ============================================================

type Like struct {
	ResaleID  string
	AvatarID  string
	CreatedAt time.Time
}

func NewLike(
	resaleID string,
	avatarID string,
	now time.Time,
) (*Like, error) {
	resaleID = strings.TrimSpace(resaleID)
	avatarID = strings.TrimSpace(avatarID)

	if !isValidReferenceID(resaleID) {
		return nil, ErrInvalidResaleID
	}

	if !isValidReferenceID(avatarID) {
		return nil, ErrInvalidAvatarID
	}

	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	return &Like{
		ResaleID:  resaleID,
		AvatarID:  avatarID,
		CreatedAt: now,
	}, nil
}

func (l Like) Validate() error {
	if !isValidReferenceID(l.ResaleID) {
		return ErrInvalidResaleID
	}

	if !isValidReferenceID(l.AvatarID) {
		return ErrInvalidAvatarID
	}

	if l.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}

	return nil
}

func (l Like) DocumentID() (string, error) {
	if !isValidReferenceID(l.AvatarID) {
		return "", ErrInvalidAvatarID
	}

	return l.AvatarID, nil
}

// ============================================================
// Comment
// ============================================================

type CommentID string

func NewCommentID() CommentID {
	return CommentID(uuid.NewString())
}

type Comment struct {
	CommentID CommentID
	ResaleID  string
	AvatarID  string
	Body      string
	Deleted   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type NewCommentParams struct {
	ResaleID string
	AvatarID string
	Body     string
	Now      time.Time
}

func NewComment(
	p NewCommentParams,
) (*Comment, error) {
	resaleID := strings.TrimSpace(p.ResaleID)
	avatarID := strings.TrimSpace(p.AvatarID)
	body := strings.TrimSpace(p.Body)

	if !isValidReferenceID(resaleID) {
		return nil, ErrInvalidResaleID
	}

	if !isValidReferenceID(avatarID) {
		return nil, ErrInvalidAvatarID
	}

	if err := validateCommentBody(body); err != nil {
		return nil, err
	}

	now := p.Now
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	return &Comment{
		CommentID: NewCommentID(),
		ResaleID:  resaleID,
		AvatarID:  avatarID,
		Body:      body,
		Deleted:   false,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func RestoreComment(
	commentID CommentID,
	resaleID string,
	avatarID string,
	body string,
	deleted bool,
	createdAt time.Time,
	updatedAt time.Time,
) (*Comment, error) {
	comment := &Comment{
		CommentID: commentID,
		ResaleID:  strings.TrimSpace(resaleID),
		AvatarID:  strings.TrimSpace(avatarID),
		Body:      body,
		Deleted:   deleted,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	if err := comment.Validate(); err != nil {
		return nil, err
	}

	return comment, nil
}

func (c Comment) Validate() error {
	if !isValidReferenceID(string(c.CommentID)) {
		return ErrInvalidCommentID
	}

	if !isValidReferenceID(c.ResaleID) {
		return ErrInvalidResaleID
	}

	if !isValidReferenceID(c.AvatarID) {
		return ErrInvalidAvatarID
	}

	if !c.Deleted {
		if err := validateCommentBody(c.Body); err != nil {
			return err
		}
	}

	if c.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}

	if c.UpdatedAt.IsZero() || c.UpdatedAt.Before(c.CreatedAt) {
		return ErrInvalidUpdatedAt
	}

	return nil
}

func (c *Comment) UpdateBody(
	body string,
	now time.Time,
) error {
	if c == nil {
		return ErrInvalid
	}

	if c.Deleted {
		return ErrDeletedComment
	}

	body = strings.TrimSpace(body)

	if err := validateCommentBody(body); err != nil {
		return err
	}

	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	if !c.CreatedAt.IsZero() && now.Before(c.CreatedAt) {
		return ErrInvalidUpdatedAt
	}

	c.Body = body
	c.UpdatedAt = now

	return nil
}

// MarkDeleted performs a logical deletion.
//
// Keeping the document allows moderation / future reply support without
// destroying the original interaction node.
//
// Whether this method is called by the comment author or an administrator
// must be validated by application.usecase.
func (c *Comment) MarkDeleted(now time.Time) error {
	if c == nil {
		return ErrInvalid
	}

	if c.Deleted {
		return nil
	}

	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	if !c.CreatedAt.IsZero() && now.Before(c.CreatedAt) {
		return ErrInvalidUpdatedAt
	}

	c.Body = ""
	c.Deleted = true
	c.UpdatedAt = now

	return nil
}

func (c Comment) IsOwnedBy(
	avatarID string,
) bool {
	return c.AvatarID == strings.TrimSpace(avatarID)
}

func (c Comment) RequireOwner(
	avatarID string,
) error {
	if !isValidReferenceID(avatarID) {
		return ErrInvalidAvatarID
	}

	if !c.IsOwnedBy(avatarID) {
		return ErrCommentAuthorMismatch
	}

	return nil
}

// ============================================================
// Viewer-specific summary
//
// This is calculated for the current avatar.
// LikedByMe must not be persisted on the aggregate document.
// ============================================================

type InteractionSummary struct {
	ResaleID     string
	LikeCount    int64
	CommentCount int64
	LikedByMe    bool
}

func NewInteractionSummary(
	resaleID string,
	likeCount int64,
	commentCount int64,
	likedByMe bool,
) (InteractionSummary, error) {
	summary := InteractionSummary{
		ResaleID:     strings.TrimSpace(resaleID),
		LikeCount:    likeCount,
		CommentCount: commentCount,
		LikedByMe:    likedByMe,
	}

	if err := summary.Validate(); err != nil {
		return InteractionSummary{}, err
	}

	return summary, nil
}

func (s InteractionSummary) Validate() error {
	if !isValidReferenceID(s.ResaleID) {
		return ErrInvalidResaleID
	}

	if s.LikeCount < 0 {
		return ErrInvalidLikeCount
	}

	if s.CommentCount < 0 {
		return ErrInvalidCommentCount
	}

	return nil
}

// ============================================================
// Helpers
// ============================================================

func validateCommentBody(
	body string,
) error {
	body = strings.TrimSpace(body)

	if body == "" {
		return ErrInvalidCommentBody
	}

	if utf8.RuneCountInString(body) > MaxCommentBodyLength {
		return ErrInvalidCommentBody
	}

	return nil
}

func isValidReferenceID(
	value string,
) bool {
	value = strings.TrimSpace(value)

	if value == "" {
		return false
	}

	if len(value) > MaxReferenceIDLength {
		return false
	}

	return true
}

func ValidateLikeTarget(
	resaleID string,
	avatarID string,
) error {
	if !isValidReferenceID(resaleID) {
		return fmt.Errorf("%w: resaleId", ErrInvalidResaleID)
	}

	if !isValidReferenceID(avatarID) {
		return fmt.Errorf("%w: avatarId", ErrInvalidAvatarID)
	}

	return nil
}
