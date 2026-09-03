// backend/internal/adapters/out/firestore/avatar_review_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	avatarreviewdom "narratives/internal/domain/avatar_review"
)

var (
	ErrAvatarReviewRepositoryNotConfigured = errors.New(
		"avatar_review_repository_fs: not configured",
	)

	ErrInvalidAvatarReviewDocumentData = errors.New(
		"avatar_review_repository_fs: invalid avatar review document data",
	)
)

// AvatarReviewRepositoryFS implements avatarreviewdom.Repository using Firestore.
//
// Collection:
//
//	avatarReviews/{tradeId}
//
// Review.ID and Review.TradeID are both equal to the Firestore document ID.
//
// Because one Trade can have at most one Avatar Review, using tradeId directly
// as the document ID allows Firestore DocumentRef.Create to enforce uniqueness
// atomically.
//
// Avatar Reviews are immutable after creation, therefore this repository does
// not expose Update or Delete operations.
type AvatarReviewRepositoryFS struct {
	Client *firestore.Client
}

var _ avatarreviewdom.Repository = (*AvatarReviewRepositoryFS)(nil)

func NewAvatarReviewRepositoryFS(client *firestore.Client) *AvatarReviewRepositoryFS {
	return &AvatarReviewRepositoryFS{Client: client}
}

func (r *AvatarReviewRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("avatarReviews")
}

// ============================================================
// Read
// ============================================================

// GetByTradeID retrieves the Avatar Review associated with one Trade.
//
// Firestore:
//
//	avatarReviews/{tradeId}
func (r *AvatarReviewRepositoryFS) GetByTradeID(
	ctx context.Context,
	tradeID string,
) (avatarreviewdom.Review, error) {
	if r == nil || r.Client == nil {
		return avatarreviewdom.Review{}, ErrAvatarReviewRepositoryNotConfigured
	}

	tradeID = strings.TrimSpace(tradeID)
	if tradeID == "" {
		return avatarreviewdom.Review{}, avatarreviewdom.ErrNotFound
	}

	snap, err := r.col().Doc(tradeID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return avatarreviewdom.Review{}, avatarreviewdom.ErrNotFound
		}

		return avatarreviewdom.Review{}, err
	}

	review, err := docToAvatarReview(snap)
	if err != nil {
		return avatarreviewdom.Review{}, err
	}

	if review.ID != tradeID || review.TradeID != tradeID {
		return avatarreviewdom.Review{}, ErrInvalidAvatarReviewDocumentData
	}

	return review, nil
}

// ListByRevieweeAvatarID retrieves Avatar Reviews received by one Avatar.
//
// Firestore query:
//
//	avatarReviews
//	  where revieweeAvatarId == {avatarId}
//	  order by createdAt desc
//	  offset {offset}
//	  limit {limit}
//
// No matching Review is a normal result and returns an empty slice.
func (r *AvatarReviewRepositoryFS) ListByRevieweeAvatarID(
	ctx context.Context,
	params avatarreviewdom.ListByRevieweeAvatarIDParams,
) ([]avatarreviewdom.Review, error) {
	if r == nil || r.Client == nil {
		return nil, ErrAvatarReviewRepositoryNotConfigured
	}

	revieweeAvatarID := strings.TrimSpace(params.RevieweeAvatarID)
	if revieweeAvatarID == "" {
		return nil, avatarreviewdom.ErrInvalidRevieweeAvatarID
	}
	if params.Limit <= 0 || params.Offset < 0 {
		return nil, avatarreviewdom.ErrInvalidRevieweeAvatarID
	}

	query := r.col().
		Where("revieweeAvatarId", "==", revieweeAvatarID).
		OrderBy("createdAt", firestore.Desc).
		Offset(params.Offset).
		Limit(params.Limit)

	it := query.Documents(ctx)
	defer it.Stop()

	reviews := make([]avatarreviewdom.Review, 0, params.Limit)

	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}

		review, err := docToAvatarReview(snap)
		if err != nil {
			return nil, err
		}

		if review.RevieweeAvatarID != revieweeAvatarID {
			return nil, fmt.Errorf(
				"avatar review %s: %w: reviewee avatar id mismatch",
				snap.Ref.ID,
				ErrInvalidAvatarReviewDocumentData,
			)
		}

		reviews = append(reviews, review)
	}

	return reviews, nil
}

// GetSummaryByRevieweeAvatarID returns public evaluation totals for one Avatar.
//
// No matching Review is a normal result and returns:
//
//	GoodCount:         0
//	DisappointedCount: 0
//	Total:             0
func (r *AvatarReviewRepositoryFS) GetSummaryByRevieweeAvatarID(
	ctx context.Context,
	revieweeAvatarID string,
) (avatarreviewdom.ReviewSummary, error) {
	if r == nil || r.Client == nil {
		return avatarreviewdom.ReviewSummary{}, ErrAvatarReviewRepositoryNotConfigured
	}

	revieweeAvatarID = strings.TrimSpace(revieweeAvatarID)
	if revieweeAvatarID == "" {
		return avatarreviewdom.ReviewSummary{}, avatarreviewdom.ErrInvalidRevieweeAvatarID
	}

	it := r.col().
		Where("revieweeAvatarId", "==", revieweeAvatarID).
		Documents(ctx)
	defer it.Stop()

	var summary avatarreviewdom.ReviewSummary

	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return avatarreviewdom.ReviewSummary{}, err
		}

		review, err := docToAvatarReview(snap)
		if err != nil {
			return avatarreviewdom.ReviewSummary{}, err
		}

		if review.RevieweeAvatarID != revieweeAvatarID {
			return avatarreviewdom.ReviewSummary{}, fmt.Errorf(
				"avatar review %s: %w: reviewee avatar id mismatch",
				snap.Ref.ID,
				ErrInvalidAvatarReviewDocumentData,
			)
		}

		switch review.Evaluation {
		case avatarreviewdom.EvaluationGood:
			summary.GoodCount++
		case avatarreviewdom.EvaluationDisappointed:
			summary.DisappointedCount++
		default:
			return avatarreviewdom.ReviewSummary{}, fmt.Errorf(
				"avatar review %s: %w: invalid evaluation",
				snap.Ref.ID,
				ErrInvalidAvatarReviewDocumentData,
			)
		}

		summary.Total++
	}

	return summary, nil
}

// ============================================================
// Create
// ============================================================

// Create persists one immutable Avatar Review.
//
// Review.ID must equal Review.TradeID.
//
// Firestore:
//
//	avatarReviews/{tradeId}
//
// DocumentRef.Create is intentionally used instead of Set.
//
// This guarantees that an existing Avatar Review cannot be overwritten and
// that concurrent attempts to review the same Trade result in
// avatarreviewdom.ErrAlreadyExists.
func (r *AvatarReviewRepositoryFS) Create(
	ctx context.Context,
	review avatarreviewdom.Review,
) (avatarreviewdom.Review, error) {
	if r == nil || r.Client == nil {
		return avatarreviewdom.Review{}, ErrAvatarReviewRepositoryNotConfigured
	}

	if err := review.Validate(); err != nil {
		return avatarreviewdom.Review{}, err
	}

	if review.ID != review.TradeID {
		return avatarreviewdom.Review{}, avatarreviewdom.ErrInvalidTradeID
	}

	review.CreatedAt = review.CreatedAt.UTC()

	ref := r.col().Doc(review.TradeID)

	_, err := ref.Create(ctx, avatarReviewToDoc(review))
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return avatarreviewdom.Review{}, avatarreviewdom.ErrAlreadyExists
		}

		return avatarreviewdom.Review{}, err
	}

	return review, nil
}

// ============================================================
// Firestore document
// ============================================================

type avatarReviewDoc struct {
	ID string `firestore:"id"`

	TradeID        string `firestore:"tradeId"`
	OrderID        string `firestore:"orderId"`
	OrderItemIndex int    `firestore:"orderItemIndex"`

	ReviewerAvatarID string `firestore:"reviewerAvatarId"`
	RevieweeAvatarID string `firestore:"revieweeAvatarId"`

	Evaluation string `firestore:"evaluation"`
	Comment    string `firestore:"comment"`

	CreatedAt time.Time `firestore:"createdAt"`
}

// ============================================================
// Mapping
// ============================================================

func avatarReviewToDoc(review avatarreviewdom.Review) avatarReviewDoc {
	return avatarReviewDoc{
		ID:               review.ID,
		TradeID:          review.TradeID,
		OrderID:          review.OrderID,
		OrderItemIndex:   review.OrderItemIndex,
		ReviewerAvatarID: review.ReviewerAvatarID,
		RevieweeAvatarID: review.RevieweeAvatarID,
		Evaluation:       string(review.Evaluation),
		Comment:          review.Comment,
		CreatedAt:        review.CreatedAt.UTC(),
	}
}

func docToAvatarReview(
	snap *firestore.DocumentSnapshot,
) (avatarreviewdom.Review, error) {
	if snap == nil || snap.Ref == nil || !snap.Exists() {
		return avatarreviewdom.Review{}, avatarreviewdom.ErrNotFound
	}

	var doc avatarReviewDoc
	if err := snap.DataTo(&doc); err != nil {
		return avatarreviewdom.Review{}, err
	}

	review := avatarreviewdom.Review{
		ID:               doc.ID,
		TradeID:          doc.TradeID,
		OrderID:          doc.OrderID,
		OrderItemIndex:   doc.OrderItemIndex,
		ReviewerAvatarID: doc.ReviewerAvatarID,
		RevieweeAvatarID: doc.RevieweeAvatarID,
		Evaluation:       avatarreviewdom.Evaluation(doc.Evaluation),
		Comment:          doc.Comment,
		CreatedAt:        doc.CreatedAt.UTC(),
	}

	// avatarReviews/{tradeId} must contain an entity whose ID and TradeID
	// both match the actual Firestore document ID.
	if review.ID != snap.Ref.ID || review.TradeID != snap.Ref.ID {
		return avatarreviewdom.Review{}, fmt.Errorf(
			"avatar review %s: %w: document id mismatch",
			snap.Ref.ID,
			ErrInvalidAvatarReviewDocumentData,
		)
	}

	if err := review.Validate(); err != nil {
		return avatarreviewdom.Review{}, fmt.Errorf(
			"avatar review %s: %w: %v",
			snap.Ref.ID,
			ErrInvalidAvatarReviewDocumentData,
			err,
		)
	}

	return review, nil
}
