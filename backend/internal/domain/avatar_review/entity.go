// backend/internal/domain/avatar_review/entity.go
package avatar_review

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

type Evaluation string

const (
	EvaluationGood         Evaluation = "good"
	EvaluationDisappointed Evaluation = "disappointed"

	MaxReferenceIDLength = 256
	MaxCommentLength     = 500
)

var (
	ErrInvalidID               = errors.New("avatar_review: invalid id")
	ErrInvalidTradeID          = errors.New("avatar_review: invalid tradeId")
	ErrInvalidOrderID          = errors.New("avatar_review: invalid orderId")
	ErrInvalidOrderItemIndex   = errors.New("avatar_review: invalid orderItemIndex")
	ErrInvalidReviewerAvatarID = errors.New("avatar_review: invalid reviewerAvatarId")
	ErrInvalidRevieweeAvatarID = errors.New("avatar_review: invalid revieweeAvatarId")
	ErrSameAvatar              = errors.New("avatar_review: reviewer and reviewee must differ")
	ErrInvalidEvaluation       = errors.New("avatar_review: invalid evaluation")
	ErrInvalidComment          = errors.New("avatar_review: invalid comment")
	ErrInvalidCreatedAt        = errors.New("avatar_review: invalid createdAt")
	ErrInvalidPagination       = errors.New("avatar_review: invalid pagination")
)

// Review is an immutable evaluation of one resale Trade by its buyer.
//
// Firestore:
//
//	avatarReviews/{tradeId}
//
// One Trade can have at most one Avatar Review. ID is intentionally equal to
// TradeID so the persistence layer can enforce uniqueness with a document
// create operation.
type Review struct {
	ID string `json:"id"`

	TradeID        string `json:"tradeId"`
	OrderID        string `json:"orderId"`
	OrderItemIndex int    `json:"orderItemIndex"`

	ReviewerAvatarID string `json:"reviewerAvatarId"`
	RevieweeAvatarID string `json:"revieweeAvatarId"`

	Evaluation Evaluation `json:"evaluation"`
	Comment    string     `json:"comment"`

	CreatedAt time.Time `json:"createdAt"`
}

type NewReviewParams struct {
	TradeID          string
	OrderID          string
	OrderItemIndex   int
	ReviewerAvatarID string
	RevieweeAvatarID string
	Evaluation       Evaluation
	Comment          string
	CreatedAt        time.Time
}

func NewReview(p NewReviewParams) (Review, error) {
	review := Review{
		ID:               strings.TrimSpace(p.TradeID),
		TradeID:          strings.TrimSpace(p.TradeID),
		OrderID:          strings.TrimSpace(p.OrderID),
		OrderItemIndex:   p.OrderItemIndex,
		ReviewerAvatarID: strings.TrimSpace(p.ReviewerAvatarID),
		RevieweeAvatarID: strings.TrimSpace(p.RevieweeAvatarID),
		Evaluation:       p.Evaluation,
		Comment:          strings.TrimSpace(p.Comment),
		CreatedAt:        p.CreatedAt.UTC(),
	}

	if err := review.Validate(); err != nil {
		return Review{}, err
	}

	return review, nil
}

func (r Review) Validate() error {
	if !isValidReferenceID(r.ID) {
		return ErrInvalidID
	}
	if !isValidReferenceID(r.TradeID) || r.ID != r.TradeID {
		return ErrInvalidTradeID
	}
	if !isValidReferenceID(r.OrderID) {
		return ErrInvalidOrderID
	}
	if r.OrderItemIndex < 0 {
		return ErrInvalidOrderItemIndex
	}
	if !isValidReferenceID(r.ReviewerAvatarID) {
		return ErrInvalidReviewerAvatarID
	}
	if !isValidReferenceID(r.RevieweeAvatarID) {
		return ErrInvalidRevieweeAvatarID
	}
	if r.ReviewerAvatarID == r.RevieweeAvatarID {
		return ErrSameAvatar
	}
	if !IsValidEvaluation(r.Evaluation) {
		return ErrInvalidEvaluation
	}
	if !utf8.ValidString(r.Comment) ||
		strings.TrimSpace(r.Comment) == "" ||
		utf8.RuneCountInString(r.Comment) > MaxCommentLength {
		return ErrInvalidComment
	}
	if r.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}

	return nil
}

func IsInvalid(err error) bool {
	return errors.Is(err, ErrInvalidID) ||
		errors.Is(err, ErrInvalidTradeID) ||
		errors.Is(err, ErrInvalidOrderID) ||
		errors.Is(err, ErrInvalidOrderItemIndex) ||
		errors.Is(err, ErrInvalidReviewerAvatarID) ||
		errors.Is(err, ErrInvalidRevieweeAvatarID) ||
		errors.Is(err, ErrSameAvatar) ||
		errors.Is(err, ErrInvalidEvaluation) ||
		errors.Is(err, ErrInvalidComment) ||
		errors.Is(err, ErrInvalidCreatedAt) ||
		errors.Is(err, ErrInvalidPagination)
}

func IsValidEvaluation(value Evaluation) bool {
	switch value {
	case EvaluationGood, EvaluationDisappointed:
		return true
	default:
		return false
	}
}

func isValidReferenceID(value string) bool {
	value = strings.TrimSpace(value)

	return value != "" &&
		len(value) <= MaxReferenceIDLength &&
		!strings.Contains(value, "/")
}
