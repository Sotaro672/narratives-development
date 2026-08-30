// backend/internal/domain/resale_review/entity.go
package resale_review

import (
	"errors"
	"fmt"
	"time"
	"unicode"
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

	ErrInvalidResaleID     = errors.New("resaleReview: invalid resaleId")
	ErrInvalidAvatarID     = errors.New("resaleReview: invalid avatarId")
	ErrInvalidCommentID    = errors.New("resaleReview: invalid commentId")
	ErrInvalidCommentBody  = errors.New("resaleReview: invalid comment body")
	ErrInvalidCreatedAt    = errors.New("resaleReview: invalid createdAt")
	ErrInvalidUpdatedAt    = errors.New("resaleReview: invalid updatedAt")
	ErrInvalidLikeCount    = errors.New("resaleReview: invalid like count")
	ErrInvalidCommentCount = errors.New("resaleReview: invalid comment count")
	ErrNegativeCounter     = errors.New("resaleReview: counter would become negative")
)

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func IsConflict(err error) bool {
	return errors.Is(err, ErrConflict)
}

func IsInvalid(err error) bool {
	return errors.Is(err, ErrInvalid) ||
		errors.Is(err, ErrInvalidResaleID) ||
		errors.Is(err, ErrInvalidAvatarID) ||
		errors.Is(err, ErrInvalidCommentID) ||
		errors.Is(err, ErrInvalidCommentBody) ||
		errors.Is(err, ErrInvalidCreatedAt) ||
		errors.Is(err, ErrInvalidUpdatedAt) ||
		errors.Is(err, ErrInvalidLikeCount) ||
		errors.Is(err, ErrInvalidCommentCount)
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
//     - isRead
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
//
// Comment body is immutable after creation.
// A comment may transition:
// - unread -> read
// - visible -> logically deleted
//
// IsRead represents whether the resale owner has read the comment.
// ============================================================

type CommentID string

func NewCommentID() CommentID {
	return CommentID(uuid.NewString())
}

type Comment struct {
	CommentID CommentID `json:"commentId"`
	ResaleID  string    `json:"resaleId"`
	AvatarID  string    `json:"avatarId"`
	Body      string    `json:"body"`
	Deleted   bool      `json:"deleted"`
	IsRead    bool      `json:"isRead"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type NewCommentParams struct {
	ResaleID string
	AvatarID string
	Body     string
	IsRead   bool
	Now      time.Time
}

func NewComment(
	p NewCommentParams,
) (*Comment, error) {
	if !isValidReferenceID(p.ResaleID) {
		return nil, ErrInvalidResaleID
	}

	if !isValidReferenceID(p.AvatarID) {
		return nil, ErrInvalidAvatarID
	}

	if err := validateCommentBody(p.Body); err != nil {
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
		ResaleID:  p.ResaleID,
		AvatarID:  p.AvatarID,
		Body:      p.Body,
		Deleted:   false,
		IsRead:    p.IsRead,
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
	isRead bool,
	createdAt time.Time,
	updatedAt time.Time,
) (*Comment, error) {
	comment := &Comment{
		CommentID: commentID,
		ResaleID:  resaleID,
		AvatarID:  avatarID,
		Body:      body,
		Deleted:   deleted,
		IsRead:    isRead,
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

// MarkRead marks the comment as read by the resale owner.
//
// Authorization must be validated by application.usecase.
// Calling MarkRead repeatedly is idempotent.
func (c *Comment) MarkRead(now time.Time) error {
	if c == nil {
		return ErrInvalid
	}

	if c.IsRead {
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

	c.IsRead = true
	c.UpdatedAt = now

	return nil
}

// MarkDeleted performs a logical deletion.
//
// Keeping the document allows moderation / future reply support without
// destroying the original interaction node.
//
// Authorization must be validated by application.usecase.
// Current policy allows the resale seller to delete comments while listing.
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

// ============================================================
// Viewer-specific summary
//
// This is calculated for the current avatar.
// LikedByMe must not be persisted on the aggregate document.
// ============================================================

type InteractionSummary struct {
	ResaleID     string `json:"resaleId"`
	LikeCount    int64  `json:"likeCount"`
	CommentCount int64  `json:"commentCount"`
	LikedByMe    bool   `json:"likedByMe"`
}

func NewInteractionSummary(
	resaleID string,
	likeCount int64,
	commentCount int64,
	likedByMe bool,
) (InteractionSummary, error) {
	summary := InteractionSummary{
		ResaleID:     resaleID,
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
	if body == "" {
		return ErrInvalidCommentBody
	}

	if !utf8.ValidString(body) {
		return ErrInvalidCommentBody
	}

	hasContent := false

	for _, r := range body {
		if !unicode.IsSpace(r) {
			hasContent = true
			break
		}
	}

	if !hasContent {
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
	if value == "" {
		return false
	}

	if !utf8.ValidString(value) {
		return false
	}

	if len(value) > MaxReferenceIDLength {
		return false
	}

	firstRune, _ := utf8.DecodeRuneInString(value)
	lastRune, _ := utf8.DecodeLastRuneInString(value)

	if unicode.IsSpace(firstRune) || unicode.IsSpace(lastRune) {
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
