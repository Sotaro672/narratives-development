// backend/internal/adapters/out/firestore/resale_review_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"sort"
	"time"

	gfs "cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
	common "narratives/internal/domain/common"
	resalereview "narratives/internal/domain/resale_review"
)

const (
	resaleReviewsCollection      = "resaleReviews"
	resaleReviewLikesSub         = "likes"
	resaleReviewCommentsSub      = "comments"
	resaleReviewCleanupBatchSize = 400
)

var errResaleReviewRepositoryNotConfigured = errors.New("resale_review_repository_fs: firestore client is nil")

type ResaleReviewRepositoryFS struct {
	Client *gfs.Client
}

var _ resalereview.RepositoryPort = (*ResaleReviewRepositoryFS)(nil)

func NewResaleReviewRepositoryFS(client *gfs.Client) *ResaleReviewRepositoryFS {
	return &ResaleReviewRepositoryFS{Client: client}
}

func (r *ResaleReviewRepositoryFS) Aggregates() resalereview.AggregateRepository {
	return &resaleReviewAggregateRepositoryFS{root: r}
}

func (r *ResaleReviewRepositoryFS) Likes() resalereview.LikeRepository {
	return &resaleReviewLikeRepositoryFS{root: r}
}

func (r *ResaleReviewRepositoryFS) Comments() resalereview.CommentRepository {
	return &resaleReviewCommentRepositoryFS{root: r}
}

func (r *ResaleReviewRepositoryFS) Mutations() resalereview.MutationRepository {
	return &resaleReviewMutationRepositoryFS{root: r}
}

func (r *ResaleReviewRepositoryFS) Cleanup() resalereview.CleanupRepository {
	return &resaleReviewCleanupRepositoryFS{root: r}
}

func (r *ResaleReviewRepositoryFS) rootCol() *gfs.CollectionRef {
	return r.Client.Collection(resaleReviewsCollection)
}

func (r *ResaleReviewRepositoryFS) rootDoc(resaleID string) *gfs.DocumentRef {
	return r.rootCol().Doc(resaleID)
}

func (r *ResaleReviewRepositoryFS) likesCol(resaleID string) *gfs.CollectionRef {
	return r.rootDoc(resaleID).Collection(resaleReviewLikesSub)
}

func (r *ResaleReviewRepositoryFS) likeDoc(resaleID string, avatarID string) *gfs.DocumentRef {
	return r.likesCol(resaleID).Doc(avatarID)
}

func (r *ResaleReviewRepositoryFS) commentsCol(resaleID string) *gfs.CollectionRef {
	return r.rootDoc(resaleID).Collection(resaleReviewCommentsSub)
}

func (r *ResaleReviewRepositoryFS) commentDoc(resaleID string, commentID string) *gfs.DocumentRef {
	return r.commentsCol(resaleID).Doc(commentID)
}

func (r *ResaleReviewRepositoryFS) validateConfigured() error {
	if r == nil || r.Client == nil {
		return errResaleReviewRepositoryNotConfigured
	}
	return nil
}

// ============================================================
// Firestore documents
// ============================================================

type resaleReviewAggregateDocument struct {
	ResaleID     string    `firestore:"resaleId"`
	LikeCount    int64     `firestore:"likeCount"`
	CommentCount int64     `firestore:"commentCount"`
	CreatedAt    time.Time `firestore:"createdAt"`
	UpdatedAt    time.Time `firestore:"updatedAt"`
}

type resaleReviewLikeDocument struct {
	ResaleID  string    `firestore:"resaleId"`
	AvatarID  string    `firestore:"avatarId"`
	CreatedAt time.Time `firestore:"createdAt"`
}

type resaleReviewCommentDocument struct {
	CommentID string    `firestore:"commentId"`
	ResaleID  string    `firestore:"resaleId"`
	AvatarID  string    `firestore:"avatarId"`
	Body      string    `firestore:"body"`
	Deleted   bool      `firestore:"deleted"`
	IsRead    bool      `firestore:"isRead"`
	CreatedAt time.Time `firestore:"createdAt"`
	UpdatedAt time.Time `firestore:"updatedAt"`
}

func aggregateToDocument(entity resalereview.ResaleReviewAggregate) resaleReviewAggregateDocument {
	return resaleReviewAggregateDocument{
		ResaleID:     entity.ResaleID,
		LikeCount:    entity.LikeCount,
		CommentCount: entity.CommentCount,
		CreatedAt:    entity.CreatedAt.UTC(),
		UpdatedAt:    entity.UpdatedAt.UTC(),
	}
}

func aggregateFromSnapshot(snapshot *gfs.DocumentSnapshot) (resalereview.ResaleReviewAggregate, error) {
	if snapshot == nil || snapshot.Ref == nil || !snapshot.Exists() {
		return resalereview.ResaleReviewAggregate{}, resalereview.ErrNotFound
	}

	var document resaleReviewAggregateDocument
	if err := snapshot.DataTo(&document); err != nil {
		return resalereview.ResaleReviewAggregate{}, err
	}

	entity := resalereview.ResaleReviewAggregate{
		ResaleID:     document.ResaleID,
		LikeCount:    document.LikeCount,
		CommentCount: document.CommentCount,
		CreatedAt:    document.CreatedAt.UTC(),
		UpdatedAt:    document.UpdatedAt.UTC(),
	}

	if entity.ResaleID == "" {
		entity.ResaleID = snapshot.Ref.ID
	}

	if err := entity.Validate(); err != nil {
		return resalereview.ResaleReviewAggregate{}, err
	}

	return entity, nil
}

func likeToDocument(entity resalereview.Like) resaleReviewLikeDocument {
	return resaleReviewLikeDocument{
		ResaleID:  entity.ResaleID,
		AvatarID:  entity.AvatarID,
		CreatedAt: entity.CreatedAt.UTC(),
	}
}

func likeFromSnapshot(snapshot *gfs.DocumentSnapshot) (resalereview.Like, error) {
	if snapshot == nil || snapshot.Ref == nil || !snapshot.Exists() {
		return resalereview.Like{}, resalereview.ErrNotFound
	}

	var document resaleReviewLikeDocument
	if err := snapshot.DataTo(&document); err != nil {
		return resalereview.Like{}, err
	}

	entity := resalereview.Like{
		ResaleID:  document.ResaleID,
		AvatarID:  document.AvatarID,
		CreatedAt: document.CreatedAt.UTC(),
	}

	if entity.AvatarID == "" {
		entity.AvatarID = snapshot.Ref.ID
	}

	if err := entity.Validate(); err != nil {
		return resalereview.Like{}, err
	}

	return entity, nil
}

func commentToDocument(entity resalereview.Comment) resaleReviewCommentDocument {
	return resaleReviewCommentDocument{
		CommentID: string(entity.CommentID),
		ResaleID:  entity.ResaleID,
		AvatarID:  entity.AvatarID,
		Body:      entity.Body,
		Deleted:   entity.Deleted,
		IsRead:    entity.IsRead,
		CreatedAt: entity.CreatedAt.UTC(),
		UpdatedAt: entity.UpdatedAt.UTC(),
	}
}

func commentFromSnapshot(snapshot *gfs.DocumentSnapshot) (resalereview.Comment, error) {
	if snapshot == nil || snapshot.Ref == nil || !snapshot.Exists() {
		return resalereview.Comment{}, resalereview.ErrNotFound
	}

	var document resaleReviewCommentDocument
	if err := snapshot.DataTo(&document); err != nil {
		return resalereview.Comment{}, err
	}

	commentID := document.CommentID
	if commentID == "" {
		commentID = snapshot.Ref.ID
	}

	comment, err := resalereview.RestoreComment(
		resalereview.CommentID(commentID),
		document.ResaleID,
		document.AvatarID,
		document.Body,
		document.Deleted,
		document.IsRead,
		document.CreatedAt.UTC(),
		document.UpdatedAt.UTC(),
	)
	if err != nil {
		return resalereview.Comment{}, err
	}

	return *comment, nil
}

// ============================================================
// Aggregate repository
// ============================================================

type resaleReviewAggregateRepositoryFS struct {
	root *ResaleReviewRepositoryFS
}

var _ resalereview.AggregateRepository = (*resaleReviewAggregateRepositoryFS)(nil)

func (r *resaleReviewAggregateRepositoryFS) GetByID(ctx context.Context, resaleID string) (resalereview.ResaleReviewAggregate, error) {
	if r == nil || r.root == nil {
		return resalereview.ResaleReviewAggregate{}, errResaleReviewRepositoryNotConfigured
	}
	if err := r.root.validateConfigured(); err != nil {
		return resalereview.ResaleReviewAggregate{}, err
	}

	if resaleID == "" {
		return resalereview.ResaleReviewAggregate{}, resalereview.ErrInvalidResaleID
	}

	snapshot, err := r.root.rootDoc(resaleID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return resalereview.ResaleReviewAggregate{}, resalereview.ErrNotFound
		}
		return resalereview.ResaleReviewAggregate{}, err
	}

	return aggregateFromSnapshot(snapshot)
}

func (r *resaleReviewAggregateRepositoryFS) Create(ctx context.Context, entity resalereview.ResaleReviewAggregate) (resalereview.ResaleReviewAggregate, error) {
	if r == nil || r.root == nil {
		return resalereview.ResaleReviewAggregate{}, errResaleReviewRepositoryNotConfigured
	}
	if err := r.root.validateConfigured(); err != nil {
		return resalereview.ResaleReviewAggregate{}, err
	}
	if err := entity.Validate(); err != nil {
		return resalereview.ResaleReviewAggregate{}, err
	}

	_, err := r.root.rootDoc(entity.ResaleID).Create(ctx, aggregateToDocument(entity))
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return resalereview.ResaleReviewAggregate{}, resalereview.ErrConflict
		}
		return resalereview.ResaleReviewAggregate{}, err
	}

	return entity, nil
}

func (r *resaleReviewAggregateRepositoryFS) Update(ctx context.Context, resaleID string, patch resalereview.PatchResaleReviewAggregate) (resalereview.ResaleReviewAggregate, error) {
	if r == nil || r.root == nil {
		return resalereview.ResaleReviewAggregate{}, errResaleReviewRepositoryNotConfigured
	}
	if err := r.root.validateConfigured(); err != nil {
		return resalereview.ResaleReviewAggregate{}, err
	}

	if resaleID == "" {
		return resalereview.ResaleReviewAggregate{}, resalereview.ErrInvalidResaleID
	}

	updates := make([]gfs.Update, 0, 3)
	if patch.LikeCount != nil {
		if *patch.LikeCount < 0 {
			return resalereview.ResaleReviewAggregate{}, resalereview.ErrInvalidLikeCount
		}
		updates = append(updates, gfs.Update{Path: "likeCount", Value: *patch.LikeCount})
	}

	if patch.CommentCount != nil {
		if *patch.CommentCount < 0 {
			return resalereview.ResaleReviewAggregate{}, resalereview.ErrInvalidCommentCount
		}
		updates = append(updates, gfs.Update{Path: "commentCount", Value: *patch.CommentCount})
	}

	updates = append(updates, gfs.Update{Path: "updatedAt", Value: time.Now().UTC()})

	_, err := r.root.rootDoc(resaleID).Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return resalereview.ResaleReviewAggregate{}, resalereview.ErrNotFound
		}
		return resalereview.ResaleReviewAggregate{}, err
	}

	return r.GetByID(ctx, resaleID)
}

func (r *resaleReviewAggregateRepositoryFS) Delete(ctx context.Context, resaleID string) error {
	if r == nil || r.root == nil {
		return errResaleReviewRepositoryNotConfigured
	}
	if err := r.root.validateConfigured(); err != nil {
		return err
	}

	if resaleID == "" {
		return resalereview.ErrInvalidResaleID
	}

	_, err := r.root.rootDoc(resaleID).Delete(ctx)
	return err
}

// ============================================================
// Like repository
// ============================================================

type resaleReviewLikeRepositoryFS struct {
	root *ResaleReviewRepositoryFS
}

var _ resalereview.LikeRepository = (*resaleReviewLikeRepositoryFS)(nil)

func (r *resaleReviewLikeRepositoryFS) List(ctx context.Context, filter resalereview.FilterLike, sortSpec common.Sort, page common.Page) (common.PageResult[resalereview.Like], error) {
	if r == nil || r.root == nil {
		return common.PageResult[resalereview.Like]{}, errResaleReviewRepositoryNotConfigured
	}
	if err := r.root.validateConfigured(); err != nil {
		return common.PageResult[resalereview.Like]{}, err
	}

	if filter.ResaleID == "" && filter.AvatarID == "" {
		return common.PageResult[resalereview.Like]{}, resalereview.ErrInvalid
	}

	items := make([]resalereview.Like, 0)

	if filter.ResaleID != "" {
		it := r.root.likesCol(filter.ResaleID).Documents(ctx)
		defer it.Stop()

		for {
			snapshot, err := it.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				return common.PageResult[resalereview.Like]{}, err
			}

			item, err := likeFromSnapshot(snapshot)
			if err != nil {
				return common.PageResult[resalereview.Like]{}, err
			}
			if !matchesLikeFilter(item, filter) {
				continue
			}

			items = append(items, item)
		}
	} else {
		parentIt := r.root.rootCol().Documents(ctx)
		defer parentIt.Stop()

		for {
			parentSnapshot, err := parentIt.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				return common.PageResult[resalereview.Like]{}, err
			}

			snapshot, err := r.root.likeDoc(parentSnapshot.Ref.ID, filter.AvatarID).Get(ctx)
			if status.Code(err) == codes.NotFound {
				continue
			}
			if err != nil {
				return common.PageResult[resalereview.Like]{}, err
			}

			item, err := likeFromSnapshot(snapshot)
			if err != nil {
				return common.PageResult[resalereview.Like]{}, err
			}
			if !matchesLikeFilter(item, filter) {
				continue
			}

			items = append(items, item)
		}
	}

	sortLikes(items, sortSpec)
	return paginateLikes(items, page), nil
}

func (r *resaleReviewLikeRepositoryFS) FindByAvatar(ctx context.Context, resaleID string, avatarID string) (resalereview.Like, error) {
	if r == nil || r.root == nil {
		return resalereview.Like{}, errResaleReviewRepositoryNotConfigured
	}
	if err := r.root.validateConfigured(); err != nil {
		return resalereview.Like{}, err
	}
	if err := resalereview.ValidateLikeTarget(resaleID, avatarID); err != nil {
		return resalereview.Like{}, err
	}

	snapshot, err := r.root.likeDoc(resaleID, avatarID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return resalereview.Like{}, resalereview.ErrNotFound
		}
		return resalereview.Like{}, err
	}

	return likeFromSnapshot(snapshot)
}

func (r *resaleReviewLikeRepositoryFS) ExistsByAvatar(ctx context.Context, resaleID string, avatarID string) (bool, error) {
	_, err := r.FindByAvatar(ctx, resaleID, avatarID)
	if err == nil {
		return true, nil
	}
	if resalereview.IsNotFound(err) {
		return false, nil
	}
	return false, err
}

func (r *resaleReviewLikeRepositoryFS) CreateUnderParent(ctx context.Context, resaleID string, like resalereview.Like) (resalereview.Like, error) {
	if r == nil || r.root == nil {
		return resalereview.Like{}, errResaleReviewRepositoryNotConfigured
	}
	if err := r.root.validateConfigured(); err != nil {
		return resalereview.Like{}, err
	}

	if err := like.Validate(); err != nil {
		return resalereview.Like{}, err
	}
	if resaleID == "" || resaleID != like.ResaleID {
		return resalereview.Like{}, resalereview.ErrInvalidResaleID
	}

	_, err := r.root.likeDoc(resaleID, like.AvatarID).Create(ctx, likeToDocument(like))
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return resalereview.Like{}, resalereview.ErrConflict
		}
		return resalereview.Like{}, err
	}

	return like, nil
}

func (r *resaleReviewLikeRepositoryFS) DeleteByAvatar(ctx context.Context, resaleID string, avatarID string) error {
	if r == nil || r.root == nil {
		return errResaleReviewRepositoryNotConfigured
	}
	if err := r.root.validateConfigured(); err != nil {
		return err
	}
	if err := resalereview.ValidateLikeTarget(resaleID, avatarID); err != nil {
		return err
	}

	_, err := r.root.likeDoc(resaleID, avatarID).Delete(ctx)
	return err
}

// ============================================================
// Comment repository
// ============================================================

type resaleReviewCommentRepositoryFS struct {
	root *ResaleReviewRepositoryFS
}

var _ resalereview.CommentRepository = (*resaleReviewCommentRepositoryFS)(nil)

func (r *resaleReviewCommentRepositoryFS) List(ctx context.Context, filter resalereview.FilterComment, sortSpec common.Sort, page common.Page) (common.PageResult[resalereview.Comment], error) {
	if r == nil || r.root == nil {
		return common.PageResult[resalereview.Comment]{}, errResaleReviewRepositoryNotConfigured
	}
	if err := r.root.validateConfigured(); err != nil {
		return common.PageResult[resalereview.Comment]{}, err
	}

	if filter.ResaleID == "" {
		return common.PageResult[resalereview.Comment]{}, resalereview.ErrInvalidResaleID
	}

	it := r.root.commentsCol(filter.ResaleID).Documents(ctx)
	defer it.Stop()

	items := make([]resalereview.Comment, 0)
	for {
		snapshot, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return common.PageResult[resalereview.Comment]{}, err
		}

		item, err := commentFromSnapshot(snapshot)
		if err != nil {
			return common.PageResult[resalereview.Comment]{}, err
		}
		if !matchesCommentFilter(item, filter) {
			continue
		}

		items = append(items, item)
	}

	sortComments(items, sortSpec)
	return paginateComments(items, page), nil
}

func (r *resaleReviewCommentRepositoryFS) GetByParentID(ctx context.Context, resaleID string, commentID string) (resalereview.Comment, error) {
	if r == nil || r.root == nil {
		return resalereview.Comment{}, errResaleReviewRepositoryNotConfigured
	}
	if err := r.root.validateConfigured(); err != nil {
		return resalereview.Comment{}, err
	}

	if resaleID == "" {
		return resalereview.Comment{}, resalereview.ErrInvalidResaleID
	}
	if commentID == "" {
		return resalereview.Comment{}, resalereview.ErrInvalidCommentID
	}

	snapshot, err := r.root.commentDoc(resaleID, commentID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return resalereview.Comment{}, resalereview.ErrNotFound
		}
		return resalereview.Comment{}, err
	}

	return commentFromSnapshot(snapshot)
}

func (r *resaleReviewCommentRepositoryFS) CreateUnderParent(ctx context.Context, resaleID string, comment resalereview.Comment) (resalereview.Comment, error) {
	if r == nil || r.root == nil {
		return resalereview.Comment{}, errResaleReviewRepositoryNotConfigured
	}
	if err := r.root.validateConfigured(); err != nil {
		return resalereview.Comment{}, err
	}

	if err := comment.Validate(); err != nil {
		return resalereview.Comment{}, err
	}
	if resaleID == "" || resaleID != comment.ResaleID {
		return resalereview.Comment{}, resalereview.ErrInvalidResaleID
	}

	commentID := string(comment.CommentID)
	if commentID == "" {
		return resalereview.Comment{}, resalereview.ErrInvalidCommentID
	}

	_, err := r.root.commentDoc(resaleID, commentID).Create(ctx, commentToDocument(comment))
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return resalereview.Comment{}, resalereview.ErrConflict
		}
		return resalereview.Comment{}, err
	}

	return comment, nil
}

func (r *resaleReviewCommentRepositoryFS) DeleteUnderParent(ctx context.Context, resaleID string, commentID string) error {
	if r == nil || r.root == nil {
		return errResaleReviewRepositoryNotConfigured
	}
	if err := r.root.validateConfigured(); err != nil {
		return err
	}

	if resaleID == "" {
		return resalereview.ErrInvalidResaleID
	}
	if commentID == "" {
		return resalereview.ErrInvalidCommentID
	}

	_, err := r.root.commentDoc(resaleID, commentID).Delete(ctx)
	return err
}

// ============================================================
// Atomic mutation repository
// ============================================================

type resaleReviewMutationRepositoryFS struct {
	root *ResaleReviewRepositoryFS
}

var _ resalereview.MutationRepository = (*resaleReviewMutationRepositoryFS)(nil)

func (r *resaleReviewMutationRepositoryFS) AddLike(ctx context.Context, like resalereview.Like) (resalereview.ResaleReviewAggregate, resalereview.Like, error) {
	if r == nil || r.root == nil {
		return resalereview.ResaleReviewAggregate{}, resalereview.Like{}, errResaleReviewRepositoryNotConfigured
	}
	if err := r.root.validateConfigured(); err != nil {
		return resalereview.ResaleReviewAggregate{}, resalereview.Like{}, err
	}
	if err := like.Validate(); err != nil {
		return resalereview.ResaleReviewAggregate{}, resalereview.Like{}, err
	}

	aggregateRef := r.root.rootDoc(like.ResaleID)
	likeRef := r.root.likeDoc(like.ResaleID, like.AvatarID)
	var resultAggregate resalereview.ResaleReviewAggregate

	err := r.root.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		var aggregate resalereview.ResaleReviewAggregate
		aggregateExists := true

		aggregateSnapshot, err := tx.Get(aggregateRef)
		if err != nil {
			if status.Code(err) != codes.NotFound {
				return err
			}

			aggregateExists = false
			created, err := resalereview.NewResaleReviewAggregate(like.ResaleID, like.CreatedAt)
			if err != nil {
				return err
			}
			aggregate = *created
		} else {
			aggregate, err = aggregateFromSnapshot(aggregateSnapshot)
			if err != nil {
				return err
			}
		}

		_, err = tx.Get(likeRef)
		if err == nil {
			return resalereview.ErrConflict
		}
		if status.Code(err) != codes.NotFound {
			return err
		}

		aggregate.IncrementLikeCount(like.CreatedAt)

		if aggregateExists {
			if err := tx.Set(aggregateRef, aggregateToDocument(aggregate)); err != nil {
				return err
			}
		} else {
			if err := tx.Create(aggregateRef, aggregateToDocument(aggregate)); err != nil {
				return err
			}
		}

		if err := tx.Create(likeRef, likeToDocument(like)); err != nil {
			return err
		}

		resultAggregate = aggregate
		return nil
	})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return resalereview.ResaleReviewAggregate{}, resalereview.Like{}, resalereview.ErrConflict
		}
		return resalereview.ResaleReviewAggregate{}, resalereview.Like{}, err
	}

	return resultAggregate, like, nil
}

func (r *resaleReviewMutationRepositoryFS) RemoveLike(ctx context.Context, resaleID string, avatarID string) (resalereview.ResaleReviewAggregate, error) {
	if r == nil || r.root == nil {
		return resalereview.ResaleReviewAggregate{}, errResaleReviewRepositoryNotConfigured
	}
	if err := r.root.validateConfigured(); err != nil {
		return resalereview.ResaleReviewAggregate{}, err
	}
	if err := resalereview.ValidateLikeTarget(resaleID, avatarID); err != nil {
		return resalereview.ResaleReviewAggregate{}, err
	}

	aggregateRef := r.root.rootDoc(resaleID)
	likeRef := r.root.likeDoc(resaleID, avatarID)
	var resultAggregate resalereview.ResaleReviewAggregate

	err := r.root.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		aggregateSnapshot, err := tx.Get(aggregateRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return resalereview.ErrNotFound
			}
			return err
		}

		aggregate, err := aggregateFromSnapshot(aggregateSnapshot)
		if err != nil {
			return err
		}

		_, err = tx.Get(likeRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				resultAggregate = aggregate
				return nil
			}
			return err
		}

		now := time.Now().UTC()
		if err := aggregate.DecrementLikeCount(now); err != nil {
			return err
		}
		if err := tx.Set(aggregateRef, aggregateToDocument(aggregate)); err != nil {
			return err
		}
		if err := tx.Delete(likeRef); err != nil {
			return err
		}

		resultAggregate = aggregate
		return nil
	})
	if err != nil {
		return resalereview.ResaleReviewAggregate{}, err
	}

	return resultAggregate, nil
}

func (r *resaleReviewMutationRepositoryFS) AddComment(ctx context.Context, comment resalereview.Comment) (resalereview.ResaleReviewAggregate, resalereview.Comment, error) {
	if r == nil || r.root == nil {
		return resalereview.ResaleReviewAggregate{}, resalereview.Comment{}, errResaleReviewRepositoryNotConfigured
	}
	if err := r.root.validateConfigured(); err != nil {
		return resalereview.ResaleReviewAggregate{}, resalereview.Comment{}, err
	}
	if err := comment.Validate(); err != nil {
		return resalereview.ResaleReviewAggregate{}, resalereview.Comment{}, err
	}

	commentID := string(comment.CommentID)
	aggregateRef := r.root.rootDoc(comment.ResaleID)
	commentRef := r.root.commentDoc(comment.ResaleID, commentID)
	var resultAggregate resalereview.ResaleReviewAggregate

	err := r.root.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		var aggregate resalereview.ResaleReviewAggregate
		aggregateExists := true

		aggregateSnapshot, err := tx.Get(aggregateRef)
		if err != nil {
			if status.Code(err) != codes.NotFound {
				return err
			}

			aggregateExists = false
			created, err := resalereview.NewResaleReviewAggregate(comment.ResaleID, comment.CreatedAt)
			if err != nil {
				return err
			}
			aggregate = *created
		} else {
			aggregate, err = aggregateFromSnapshot(aggregateSnapshot)
			if err != nil {
				return err
			}
		}

		_, err = tx.Get(commentRef)
		if err == nil {
			return resalereview.ErrConflict
		}
		if status.Code(err) != codes.NotFound {
			return err
		}

		aggregate.IncrementCommentCount(comment.CreatedAt)

		if aggregateExists {
			if err := tx.Set(aggregateRef, aggregateToDocument(aggregate)); err != nil {
				return err
			}
		} else {
			if err := tx.Create(aggregateRef, aggregateToDocument(aggregate)); err != nil {
				return err
			}
		}

		if err := tx.Create(commentRef, commentToDocument(comment)); err != nil {
			return err
		}

		resultAggregate = aggregate
		return nil
	})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return resalereview.ResaleReviewAggregate{}, resalereview.Comment{}, resalereview.ErrConflict
		}
		return resalereview.ResaleReviewAggregate{}, resalereview.Comment{}, err
	}

	return resultAggregate, comment, nil
}

func (r *resaleReviewMutationRepositoryFS) MarkCommentsRead(ctx context.Context, resaleID string) (int, error) {
	if r == nil || r.root == nil {
		return 0, errResaleReviewRepositoryNotConfigured
	}
	if err := r.root.validateConfigured(); err != nil {
		return 0, err
	}
	if resaleID == "" {
		return 0, resalereview.ErrInvalidResaleID
	}

	it := r.root.commentsCol(resaleID).Documents(ctx)
	defer it.Stop()

	now := time.Now().UTC()
	refs := make([]*gfs.DocumentRef, 0, resaleReviewCleanupBatchSize)
	markedCount := 0

	commit := func() error {
		if len(refs) == 0 {
			return nil
		}

		batch := r.root.Client.Batch()
		for _, ref := range refs {
			batch.Update(ref, []gfs.Update{
				{Path: "isRead", Value: true},
				{Path: "updatedAt", Value: now},
			})
		}

		if _, err := batch.Commit(ctx); err != nil {
			return err
		}

		refs = refs[:0]
		return nil
	}

	for {
		snapshot, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return 0, err
		}

		comment, err := commentFromSnapshot(snapshot)
		if err != nil {
			return 0, err
		}
		if comment.Deleted || comment.IsRead {
			continue
		}

		refs = append(refs, snapshot.Ref)
		markedCount++

		if len(refs) >= resaleReviewCleanupBatchSize {
			if err := commit(); err != nil {
				return 0, err
			}
		}
	}

	if err := commit(); err != nil {
		return 0, err
	}

	return markedCount, nil
}

func (r *resaleReviewMutationRepositoryFS) MarkCommentDeleted(ctx context.Context, resaleID string, commentID string) (resalereview.ResaleReviewAggregate, resalereview.Comment, error) {
	if r == nil || r.root == nil {
		return resalereview.ResaleReviewAggregate{}, resalereview.Comment{}, errResaleReviewRepositoryNotConfigured
	}
	if err := r.root.validateConfigured(); err != nil {
		return resalereview.ResaleReviewAggregate{}, resalereview.Comment{}, err
	}

	if resaleID == "" {
		return resalereview.ResaleReviewAggregate{}, resalereview.Comment{}, resalereview.ErrInvalidResaleID
	}
	if commentID == "" {
		return resalereview.ResaleReviewAggregate{}, resalereview.Comment{}, resalereview.ErrInvalidCommentID
	}

	aggregateRef := r.root.rootDoc(resaleID)
	commentRef := r.root.commentDoc(resaleID, commentID)
	var resultAggregate resalereview.ResaleReviewAggregate
	var resultComment resalereview.Comment

	err := r.root.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		aggregateSnapshot, err := tx.Get(aggregateRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return resalereview.ErrNotFound
			}
			return err
		}

		commentSnapshot, err := tx.Get(commentRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return resalereview.ErrNotFound
			}
			return err
		}

		aggregate, err := aggregateFromSnapshot(aggregateSnapshot)
		if err != nil {
			return err
		}

		comment, err := commentFromSnapshot(commentSnapshot)
		if err != nil {
			return err
		}

		if comment.Deleted {
			resultAggregate = aggregate
			resultComment = comment
			return nil
		}

		now := time.Now().UTC()
		if err := comment.MarkDeleted(now); err != nil {
			return err
		}
		if err := aggregate.DecrementCommentCount(now); err != nil {
			return err
		}
		if err := tx.Set(aggregateRef, aggregateToDocument(aggregate)); err != nil {
			return err
		}
		if err := tx.Set(commentRef, commentToDocument(comment)); err != nil {
			return err
		}

		resultAggregate = aggregate
		resultComment = comment
		return nil
	})
	if err != nil {
		return resalereview.ResaleReviewAggregate{}, resalereview.Comment{}, err
	}

	return resultAggregate, resultComment, nil
}

// ============================================================
// Cleanup repository
// ============================================================

type resaleReviewCleanupRepositoryFS struct {
	root *ResaleReviewRepositoryFS
}

var _ resalereview.CleanupRepository = (*resaleReviewCleanupRepositoryFS)(nil)

func (r *resaleReviewCleanupRepositoryFS) DeleteByResaleID(ctx context.Context, resaleID string) error {
	if r == nil || r.root == nil {
		return errResaleReviewRepositoryNotConfigured
	}
	if err := r.root.validateConfigured(); err != nil {
		return err
	}

	if resaleID == "" {
		return resalereview.ErrInvalidResaleID
	}

	if err := deleteResaleReviewCollection(ctx, r.root.Client, r.root.likesCol(resaleID)); err != nil {
		return err
	}
	if err := deleteResaleReviewCollection(ctx, r.root.Client, r.root.commentsCol(resaleID)); err != nil {
		return err
	}

	_, err := r.root.rootDoc(resaleID).Delete(ctx)
	return err
}

func deleteResaleReviewCollection(ctx context.Context, client *gfs.Client, collection *gfs.CollectionRef) error {
	if client == nil {
		return errResaleReviewRepositoryNotConfigured
	}
	if collection == nil {
		return nil
	}

	it := collection.Documents(ctx)
	defer it.Stop()

	refs := make([]*gfs.DocumentRef, 0, resaleReviewCleanupBatchSize)

	commit := func() error {
		if len(refs) == 0 {
			return nil
		}

		batch := client.Batch()
		for _, ref := range refs {
			batch.Delete(ref)
		}

		if _, err := batch.Commit(ctx); err != nil {
			return err
		}

		refs = refs[:0]
		return nil
	}

	for {
		snapshot, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}

		refs = append(refs, snapshot.Ref)
		if len(refs) >= resaleReviewCleanupBatchSize {
			if err := commit(); err != nil {
				return err
			}
		}
	}

	return commit()
}

// ============================================================
// List helpers
// ============================================================

func matchesLikeFilter(item resalereview.Like, filter resalereview.FilterLike) bool {
	if filter.ResaleID != "" && item.ResaleID != filter.ResaleID {
		return false
	}
	if filter.AvatarID != "" && item.AvatarID != filter.AvatarID {
		return false
	}
	if filter.Created.From != nil && item.CreatedAt.Before(*filter.Created.From) {
		return false
	}
	if filter.Created.To != nil && item.CreatedAt.After(*filter.Created.To) {
		return false
	}
	return true
}

func matchesCommentFilter(item resalereview.Comment, filter resalereview.FilterComment) bool {
	if filter.ResaleID != "" && item.ResaleID != filter.ResaleID {
		return false
	}
	if filter.AvatarID != "" && item.AvatarID != filter.AvatarID {
		return false
	}
	if filter.Deleted != nil && item.Deleted != *filter.Deleted {
		return false
	}
	if filter.IsRead != nil && item.IsRead != *filter.IsRead {
		return false
	}
	if filter.Created.From != nil && item.CreatedAt.Before(*filter.Created.From) {
		return false
	}
	if filter.Created.To != nil && item.CreatedAt.After(*filter.Created.To) {
		return false
	}
	if filter.Updated.From != nil && item.UpdatedAt.Before(*filter.Updated.From) {
		return false
	}
	if filter.Updated.To != nil && item.UpdatedAt.After(*filter.Updated.To) {
		return false
	}
	return true
}

func sortLikes(items []resalereview.Like, sortSpec common.Sort) {
	descending := sortSpec.Order != common.SortAsc

	sort.SliceStable(items, func(i, j int) bool {
		left := items[i]
		right := items[j]

		switch sortSpec.Column {
		case "avatarId", "AvatarID":
			if left.AvatarID == right.AvatarID {
				return left.CreatedAt.After(right.CreatedAt)
			}
			if descending {
				return left.AvatarID > right.AvatarID
			}
			return left.AvatarID < right.AvatarID

		default:
			if left.CreatedAt.Equal(right.CreatedAt) {
				if descending {
					return left.AvatarID > right.AvatarID
				}
				return left.AvatarID < right.AvatarID
			}
			if descending {
				return left.CreatedAt.After(right.CreatedAt)
			}
			return left.CreatedAt.Before(right.CreatedAt)
		}
	})
}

func sortComments(items []resalereview.Comment, sortSpec common.Sort) {
	descending := sortSpec.Order != common.SortAsc

	sort.SliceStable(items, func(i, j int) bool {
		left := items[i]
		right := items[j]

		switch sortSpec.Column {
		case "updatedAt", "UpdatedAt":
			if left.UpdatedAt.Equal(right.UpdatedAt) {
				return compareCommentID(left.CommentID, right.CommentID, descending)
			}
			if descending {
				return left.UpdatedAt.After(right.UpdatedAt)
			}
			return left.UpdatedAt.Before(right.UpdatedAt)

		case "avatarId", "AvatarID":
			if left.AvatarID == right.AvatarID {
				return compareCommentID(left.CommentID, right.CommentID, descending)
			}
			if descending {
				return left.AvatarID > right.AvatarID
			}
			return left.AvatarID < right.AvatarID

		default:
			if left.CreatedAt.Equal(right.CreatedAt) {
				return compareCommentID(left.CommentID, right.CommentID, descending)
			}
			if descending {
				return left.CreatedAt.After(right.CreatedAt)
			}
			return left.CreatedAt.Before(right.CreatedAt)
		}
	})
}

func compareCommentID(left resalereview.CommentID, right resalereview.CommentID, descending bool) bool {
	if descending {
		return string(left) > string(right)
	}
	return string(left) < string(right)
}

func paginateLikes(items []resalereview.Like, page common.Page) common.PageResult[resalereview.Like] {
	pageNum, perPage, offset := fscommon.NormalizePage(page.Number, page.PerPage, 20, 200)
	totalCount := len(items)
	totalPages := fscommon.ComputeTotalPages(totalCount, perPage)

	if offset >= totalCount {
		return common.PageResult[resalereview.Like]{
			Items:      []resalereview.Like{},
			TotalCount: totalCount,
			TotalPages: totalPages,
			Page:       pageNum,
			PerPage:    perPage,
		}
	}

	end := offset + perPage
	if end > totalCount {
		end = totalCount
	}

	return common.PageResult[resalereview.Like]{
		Items:      items[offset:end],
		TotalCount: totalCount,
		TotalPages: totalPages,
		Page:       pageNum,
		PerPage:    perPage,
	}
}

func paginateComments(items []resalereview.Comment, page common.Page) common.PageResult[resalereview.Comment] {
	pageNum, perPage, offset := fscommon.NormalizePage(page.Number, page.PerPage, 20, 200)
	totalCount := len(items)
	totalPages := fscommon.ComputeTotalPages(totalCount, perPage)

	if offset >= totalCount {
		return common.PageResult[resalereview.Comment]{
			Items:      []resalereview.Comment{},
			TotalCount: totalCount,
			TotalPages: totalPages,
			Page:       pageNum,
			PerPage:    perPage,
		}
	}

	end := offset + perPage
	if end > totalCount {
		end = totalCount
	}

	return common.PageResult[resalereview.Comment]{
		Items:      items[offset:end],
		TotalCount: totalCount,
		TotalPages: totalPages,
		Page:       pageNum,
		PerPage:    perPage,
	}
}
