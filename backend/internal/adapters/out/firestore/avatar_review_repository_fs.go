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
	ErrInvalidAvatarReviewSummaryDocumentData = errors.New(
		"avatar_review_repository_fs: invalid avatar review summary document data",
	)
)

// AvatarReviewRepositoryFS implements avatarreviewdom.Repository using Firestore.
//
// Collections:
//
//	avatarReviews/{tradeId}
//	avatarReviewSummaries/{avatarId}
//
// avatarReviews stores immutable individual reviews. Review.ID and
// Review.TradeID are both equal to the Firestore document ID.
//
// avatarReviewSummaries stores mutable public evaluation totals for each Avatar.
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

func (r *AvatarReviewRepositoryFS) summaryCol() *firestore.CollectionRef {
	return r.Client.Collection("avatarReviewSummaries")
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
		return nil, avatarreviewdom.ErrInvalidPagination
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
// Primary source:
//
//	avatarReviewSummaries/{avatarId}
//
// For backward compatibility, when the summary document does not yet exist,
// totals are calculated from existing avatarReviews documents.
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

	snap, err := r.summaryCol().Doc(revieweeAvatarID).Get(ctx)
	if err == nil {
		doc, err := docToAvatarReviewSummary(snap, revieweeAvatarID)
		if err != nil {
			return avatarreviewdom.ReviewSummary{}, err
		}

		return avatarreviewdom.ReviewSummary{
			GoodCount:         doc.GoodCount,
			DisappointedCount: doc.DisappointedCount,
			Total:             doc.Total,
		}, nil
	}

	if status.Code(err) != codes.NotFound {
		return avatarreviewdom.ReviewSummary{}, err
	}

	return r.calculateSummaryFromReviews(ctx, revieweeAvatarID)
}

func (r *AvatarReviewRepositoryFS) calculateSummaryFromReviews(
	ctx context.Context,
	revieweeAvatarID string,
) (avatarreviewdom.ReviewSummary, error) {
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
// Summary
// ============================================================

// EnsureSummaryByRevieweeAvatarID ensures that a zero-valued summary document
// exists for one Avatar. Existing totals are never overwritten.
//
// Firestore:
//
//	avatarReviewSummaries/{avatarId}
func (r *AvatarReviewRepositoryFS) EnsureSummaryByRevieweeAvatarID(
	ctx context.Context,
	revieweeAvatarID string,
	now time.Time,
) error {
	if r == nil || r.Client == nil {
		return ErrAvatarReviewRepositoryNotConfigured
	}

	revieweeAvatarID = strings.TrimSpace(revieweeAvatarID)
	if revieweeAvatarID == "" {
		return avatarreviewdom.ErrInvalidRevieweeAvatarID
	}
	if now.IsZero() {
		return avatarreviewdom.ErrInvalidCreatedAt
	}

	now = now.UTC()

	doc := avatarReviewSummaryDoc{
		AvatarID:          revieweeAvatarID,
		GoodCount:         0,
		DisappointedCount: 0,
		Total:             0,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	_, err := r.summaryCol().Doc(revieweeAvatarID).Create(ctx, doc)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil
		}
		return err
	}

	return nil
}

// DeleteSummaryByRevieweeAvatarID deletes one Avatar Review summary.
// A missing summary is treated as success.
func (r *AvatarReviewRepositoryFS) DeleteSummaryByRevieweeAvatarID(
	ctx context.Context,
	revieweeAvatarID string,
) error {
	if r == nil || r.Client == nil {
		return ErrAvatarReviewRepositoryNotConfigured
	}

	revieweeAvatarID = strings.TrimSpace(revieweeAvatarID)
	if revieweeAvatarID == "" {
		return avatarreviewdom.ErrInvalidRevieweeAvatarID
	}

	_, err := r.summaryCol().Doc(revieweeAvatarID).Delete(ctx)
	if err != nil && status.Code(err) != codes.NotFound {
		return err
	}

	return nil
}

// ============================================================
// Create
// ============================================================

// Create persists one immutable Avatar Review and updates the corresponding
// Avatar Review summary in one Firestore transaction.
//
// Firestore:
//
//	avatarReviews/{tradeId}
//	avatarReviewSummaries/{revieweeAvatarId}
//
// This guarantees that an Avatar Review cannot be created without the public
// summary being updated and prevents duplicate reviews for the same Trade.
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

	reviewRef := r.col().Doc(review.TradeID)
	summaryRef := r.summaryCol().Doc(review.RevieweeAvatarID)

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		reviewSnap, err := tx.Get(reviewRef)
		if err == nil && reviewSnap.Exists() {
			return avatarreviewdom.ErrAlreadyExists
		}
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}

		now := review.CreatedAt
		summaryDoc := avatarReviewSummaryDoc{
			AvatarID:          review.RevieweeAvatarID,
			GoodCount:         0,
			DisappointedCount: 0,
			Total:             0,
			CreatedAt:         now,
			UpdatedAt:         now,
		}

		summarySnap, err := tx.Get(summaryRef)
		if err == nil {
			existing, err := docToAvatarReviewSummary(summarySnap, review.RevieweeAvatarID)
			if err != nil {
				return err
			}
			summaryDoc = existing
		} else if status.Code(err) != codes.NotFound {
			return err
		}

		switch review.Evaluation {
		case avatarreviewdom.EvaluationGood:
			summaryDoc.GoodCount++
		case avatarreviewdom.EvaluationDisappointed:
			summaryDoc.DisappointedCount++
		default:
			return avatarreviewdom.ErrInvalidEvaluation
		}

		summaryDoc.Total++
		summaryDoc.UpdatedAt = now

		if err := tx.Create(reviewRef, avatarReviewToDoc(review)); err != nil {
			if status.Code(err) == codes.AlreadyExists {
				return avatarreviewdom.ErrAlreadyExists
			}
			return err
		}

		if err := tx.Set(summaryRef, summaryDoc); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, avatarreviewdom.ErrAlreadyExists) ||
			status.Code(err) == codes.AlreadyExists {
			return avatarreviewdom.Review{}, avatarreviewdom.ErrAlreadyExists
		}
		return avatarreviewdom.Review{}, err
	}

	return review, nil
}

// ============================================================
// Firestore documents
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

type avatarReviewSummaryDoc struct {
	AvatarID          string    `firestore:"avatarId"`
	GoodCount         int64     `firestore:"goodCount"`
	DisappointedCount int64     `firestore:"disappointedCount"`
	Total             int64     `firestore:"total"`
	CreatedAt         time.Time `firestore:"createdAt"`
	UpdatedAt         time.Time `firestore:"updatedAt"`
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

func docToAvatarReviewSummary(
	snap *firestore.DocumentSnapshot,
	expectedAvatarID string,
) (avatarReviewSummaryDoc, error) {
	if snap == nil || snap.Ref == nil || !snap.Exists() {
		return avatarReviewSummaryDoc{}, avatarreviewdom.ErrNotFound
	}

	var doc avatarReviewSummaryDoc
	if err := snap.DataTo(&doc); err != nil {
		return avatarReviewSummaryDoc{}, err
	}

	expectedAvatarID = strings.TrimSpace(expectedAvatarID)
	doc.AvatarID = strings.TrimSpace(doc.AvatarID)

	if doc.AvatarID == "" ||
		doc.AvatarID != expectedAvatarID ||
		snap.Ref.ID != expectedAvatarID ||
		doc.GoodCount < 0 ||
		doc.DisappointedCount < 0 ||
		doc.Total < 0 ||
		doc.Total != doc.GoodCount+doc.DisappointedCount ||
		doc.CreatedAt.IsZero() ||
		doc.UpdatedAt.IsZero() {
		return avatarReviewSummaryDoc{}, fmt.Errorf(
			"avatar review summary %s: %w",
			snap.Ref.ID,
			ErrInvalidAvatarReviewSummaryDocumentData,
		)
	}

	doc.CreatedAt = doc.CreatedAt.UTC()
	doc.UpdatedAt = doc.UpdatedAt.UTC()

	return doc, nil
}
