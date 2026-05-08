// backend/internal/domain/productBlueprintReview/entity.go
package productBlueprintReview

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// 汎用エラー（ドメイン共通）
var (
	ErrNotFound     = errors.New("productBlueprintReview: not found")
	ErrConflict     = errors.New("productBlueprintReview: conflict")
	ErrInvalid      = errors.New("productBlueprintReview: invalid")
	ErrUnauthorized = errors.New("productBlueprintReview: unauthorized")
	ErrForbidden    = errors.New("productBlueprintReview: forbidden")
	ErrInternal     = errors.New("productBlueprintReview: internal")
)

func IsNotFound(err error) bool     { return errors.Is(err, ErrNotFound) }
func IsConflict(err error) bool     { return errors.Is(err, ErrConflict) }
func IsInvalid(err error) bool      { return errors.Is(err, ErrInvalid) }
func IsUnauthorized(err error) bool { return errors.Is(err, ErrUnauthorized) }
func IsForbidden(err error) bool    { return errors.Is(err, ErrForbidden) }
func IsInternal(err error) bool     { return errors.Is(err, ErrInternal) }

// ======================================
// ID
// ======================================

type ReviewID string

func NewReviewID() ReviewID {
	return ReviewID(uuid.NewString())
}

// ======================================
// Enums
// ======================================

type ReviewStatus string

const (
	ReviewStatusPublished ReviewStatus = "PUBLISHED"
	ReviewStatusHidden    ReviewStatus = "HIDDEN"  // 審査/一時停止など
	ReviewStatusRemoved   ReviewStatus = "REMOVED" // 規約違反などで削除扱い
)

func isValidStatus(v ReviewStatus) bool {
	switch v {
	case ReviewStatusPublished, ReviewStatusHidden, ReviewStatusRemoved:
		return true
	default:
		return false
	}
}

type Rating int

const (
	RatingMin Rating = 1
	RatingMax Rating = 5
)

func (r Rating) validate() error {
	if r < RatingMin || r > RatingMax {
		return errors.New("rating must be between 1 and 5")
	}
	return nil
}

// ======================================
// Entity（画像添付なし前提 / VineVoiceなし）
// ======================================

type Review struct {
	ID ReviewID

	// 口コミ対象（productBlueprint）
	ProductBlueprintID string

	// 投稿者（avatar）
	AvatarID string

	Rating Rating
	Title  string
	Body   string

	// Helpful 投票（集計）
	HelpfulVotes int
	TotalVotes   int

	// Amazonの「レビュー済み日」相当
	ReviewedAt time.Time

	Status ReviewStatus

	// 監査
	CreatedAt time.Time
	CreatedBy string
	UpdatedAt time.Time
	UpdatedBy string

	ModerationReason *string
}

// ======================================
// Errors（Validation）
// ======================================

var (
	ErrInvalidID                 = errors.New("productBlueprintReview: invalid id")
	ErrInvalidProductBlueprintID = errors.New("productBlueprintReview: invalid productBlueprintId")
	ErrInvalidAvatarID           = errors.New("productBlueprintReview: invalid avatarId")
	ErrInvalidTitle              = errors.New("productBlueprintReview: invalid title")
	ErrInvalidBody               = errors.New("productBlueprintReview: invalid body")
	ErrInvalidReviewedAt         = errors.New("productBlueprintReview: invalid reviewedAt")
	ErrInvalidCreatedAt          = errors.New("productBlueprintReview: invalid createdAt")
	ErrInvalidCreatedBy          = errors.New("productBlueprintReview: invalid createdBy")
	ErrInvalidVotes              = errors.New("productBlueprintReview: invalid votes")
	ErrInvalidStatus             = errors.New("productBlueprintReview: invalid status")
)

// ======================================
// Constructors
// ======================================

type NewReviewParams struct {
	ProductBlueprintID string
	AvatarID           string

	Rating Rating
	Title  string
	Body   string

	ReviewedAt time.Time
	CreatedAt  time.Time
	CreatedBy  string
	PublishNow bool // trueなら初期Status=PUBLISHED, falseならHIDDEN（審査待ち）
}

func New(p NewReviewParams) (Review, error) {
	r := Review{
		ID:                 NewReviewID(),
		ProductBlueprintID: p.ProductBlueprintID,
		AvatarID:           p.AvatarID,
		Rating:             p.Rating,
		Title:              p.Title,
		Body:               p.Body,
		HelpfulVotes:       0,
		TotalVotes:         0,
		ReviewedAt:         p.ReviewedAt,
		Status:             ReviewStatusHidden,
		CreatedAt:          p.CreatedAt.UTC(),
		CreatedBy:          p.CreatedBy,
		UpdatedAt:          p.CreatedAt.UTC(),
		UpdatedBy:          p.CreatedBy,
		ModerationReason:   nil,
	}

	if p.PublishNow {
		r.Status = ReviewStatusPublished
	}

	if err := r.validate(); err != nil {
		return Review{}, err
	}
	return r, nil
}

// ======================================
// Methods（Amazon的な挙動に寄せる）
// ======================================

func (r *Review) Publish(now time.Time, updatedBy string) error {
	if updatedBy == "" {
		return ErrInvalidCreatedBy
	}
	if r.Status == ReviewStatusRemoved {
		return ErrForbidden
	}
	r.Status = ReviewStatusPublished
	r.ModerationReason = nil
	r.touch(now, updatedBy)
	return nil
}

func (r *Review) Hide(reason string, now time.Time, updatedBy string) error {
	if reason == "" {
		return ErrInvalid
	}
	if updatedBy == "" {
		return ErrInvalidCreatedBy
	}
	if r.Status == ReviewStatusRemoved {
		return ErrForbidden
	}
	r.Status = ReviewStatusHidden
	r.ModerationReason = &reason
	r.touch(now, updatedBy)
	return nil
}

func (r *Review) Remove(reason string, now time.Time, updatedBy string) error {
	if reason == "" {
		return ErrInvalid
	}
	if updatedBy == "" {
		return ErrInvalidCreatedBy
	}
	r.Status = ReviewStatusRemoved
	r.ModerationReason = &reason
	r.touch(now, updatedBy)
	return nil
}

// 役に立った / 役に立たなかった
func (r *Review) AddHelpfulVote() error {
	if r.Status != ReviewStatusPublished {
		return ErrForbidden
	}
	r.HelpfulVotes++
	r.TotalVotes++
	return nil
}

func (r *Review) AddNotHelpfulVote() error {
	if r.Status != ReviewStatusPublished {
		return ErrForbidden
	}
	r.TotalVotes++
	return nil
}

func (r *Review) UpdateContent(title, body string, rating Rating, now time.Time, updatedBy string) error {
	if r.Status == ReviewStatusRemoved {
		return ErrForbidden
	}
	if title == "" {
		return ErrInvalidTitle
	}
	if body == "" {
		return ErrInvalidBody
	}
	if err := rating.validate(); err != nil {
		return err
	}
	if updatedBy == "" {
		return ErrInvalidCreatedBy
	}

	r.Title = title
	r.Body = body
	r.Rating = rating
	r.touch(now, updatedBy)
	return nil
}

// ======================================
// Validation / Helpers
// ======================================

func (r Review) validate() error {
	if r.ID == "" {
		return ErrInvalidID
	}
	if r.ProductBlueprintID == "" {
		return ErrInvalidProductBlueprintID
	}
	if r.AvatarID == "" {
		return ErrInvalidAvatarID
	}
	if err := r.Rating.validate(); err != nil {
		return err
	}
	if r.Title == "" {
		return ErrInvalidTitle
	}
	if r.Body == "" {
		return ErrInvalidBody
	}
	if r.ReviewedAt.IsZero() {
		return ErrInvalidReviewedAt
	}
	if r.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if r.CreatedBy == "" {
		return ErrInvalidCreatedBy
	}
	if r.HelpfulVotes < 0 || r.TotalVotes < 0 || r.HelpfulVotes > r.TotalVotes {
		return ErrInvalidVotes
	}
	if !isValidStatus(r.Status) {
		return ErrInvalidStatus
	}

	return nil
}

func (r *Review) touch(now time.Time, updatedBy string) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	r.UpdatedAt = now.UTC()
	r.UpdatedBy = updatedBy
}
