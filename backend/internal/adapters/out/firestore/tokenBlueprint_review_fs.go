// backend/internal/adapters/out/firestore/tokenBlueprint_review_fs.go
package firestore

import (
	"context"
	"errors"
	"math"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	common "narratives/internal/domain/common"
	tbReview "narratives/internal/domain/tokenBlueprint_review"
)

var (
	errTBReviewNotConfigured = errors.New("tokenBlueprint_review_fs: firestore client is nil")
)

// TokenBlueprintReviewRepositoryFS implements tbReview.RepositoryPort.
//
// Firestore model (YouTube-like, infinite nested comments):
// tokenBlueprintReviews/{tokenBlueprintId}
//   - reactions/{actorType_actorId}
//   - comments/{commentId}
//   - reactions/{actorType_actorId}
type TokenBlueprintReviewRepositoryFS struct {
	fs *firestore.Client

	// root collection
	rootCol string
}

func NewTokenBlueprintReviewRepositoryFS(fs *firestore.Client) *TokenBlueprintReviewRepositoryFS {
	return &TokenBlueprintReviewRepositoryFS{
		fs:      fs,
		rootCol: "tokenBlueprintReviews",
	}
}

// RepositoryPort
func (r *TokenBlueprintReviewRepositoryFS) TokenBlueprintAggregates() tbReview.TokenBlueprintAggregateRepository {
	return &tokenBlueprintAggregateRepoFS{root: r}
}

func (r *TokenBlueprintReviewRepositoryFS) Comments() tbReview.CommentRepository {
	return &commentRepoFS{root: r}
}

func (r *TokenBlueprintReviewRepositoryFS) TokenBlueprintReactions() tbReview.TokenBlueprintReactionRepository {
	return &tokenBlueprintReactionRepoFS{root: r}
}

func (r *TokenBlueprintReviewRepositoryFS) CommentReactions() tbReview.CommentReactionRepository {
	return &commentReactionRepoFS{root: r}
}

// -------------------------
// path helpers
// -------------------------

func (r *TokenBlueprintReviewRepositoryFS) rootDoc(tokenBlueprintID string) *firestore.DocumentRef {
	return r.fs.Collection(r.rootCol).Doc(tokenBlueprintID)
}

func (r *TokenBlueprintReviewRepositoryFS) tokenReactionsCol(tokenBlueprintID string) *firestore.CollectionRef {
	return r.rootDoc(tokenBlueprintID).Collection("reactions")
}

func (r *TokenBlueprintReviewRepositoryFS) commentsCol(tokenBlueprintID string) *firestore.CollectionRef {
	return r.rootDoc(tokenBlueprintID).Collection("comments")
}

func (r *TokenBlueprintReviewRepositoryFS) commentReactionsCol(tokenBlueprintID, commentID string) *firestore.CollectionRef {
	return r.commentsCol(tokenBlueprintID).Doc(commentID).Collection("reactions")
}

func isNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	return status.Code(err) == codes.NotFound
}

func tokenReactionDocID(actorType tbReview.ActorType, actorID string) string {
	return string(actorType) + "_" + actorID
}

// ============================================================
// Aggregate Repo: tokenBlueprintReviews/{tokenBlueprintId}
// ============================================================

type tokenBlueprintAggregateRepoFS struct {
	root *TokenBlueprintReviewRepositoryFS
}

func (a *tokenBlueprintAggregateRepoFS) GetByID(ctx context.Context, id string) (tbReview.TokenBlueprintReviewAggregate, error) {
	if a.root == nil || a.root.fs == nil {
		return tbReview.TokenBlueprintReviewAggregate{}, errTBReviewNotConfigured
	}

	snap, err := a.root.rootDoc(id).Get(ctx)
	if err != nil {
		if isNotFoundErr(err) {
			return tbReview.TokenBlueprintReviewAggregate{}, errNotFound
		}
		return tbReview.TokenBlueprintReviewAggregate{}, err
	}

	var out tbReview.TokenBlueprintReviewAggregate
	if err := snap.DataTo(&out); err != nil {
		return tbReview.TokenBlueprintReviewAggregate{}, err
	}
	return out, nil
}

func (a *tokenBlueprintAggregateRepoFS) Create(ctx context.Context, entity tbReview.TokenBlueprintReviewAggregate) (tbReview.TokenBlueprintReviewAggregate, error) {
	if a.root == nil || a.root.fs == nil {
		return tbReview.TokenBlueprintReviewAggregate{}, errTBReviewNotConfigured
	}
	if entity.TokenBlueprintID == "" {
		return tbReview.TokenBlueprintReviewAggregate{}, errors.New("tokenBlueprint_review_fs: TokenBlueprintID is required")
	}

	_, err := a.root.rootDoc(entity.TokenBlueprintID).Create(ctx, entity)
	if err != nil {
		return tbReview.TokenBlueprintReviewAggregate{}, err
	}
	return entity, nil
}

func (a *tokenBlueprintAggregateRepoFS) Update(ctx context.Context, id string, patch tbReview.PatchTokenBlueprintReviewAggregate) (tbReview.TokenBlueprintReviewAggregate, error) {
	if a.root == nil || a.root.fs == nil {
		return tbReview.TokenBlueprintReviewAggregate{}, errTBReviewNotConfigured
	}

	updates := make([]firestore.Update, 0, 6)
	if patch.LikeCount != nil {
		updates = append(updates, firestore.Update{Path: "LikeCount", Value: *patch.LikeCount})
	}
	if patch.DislikeCount != nil {
		updates = append(updates, firestore.Update{Path: "DislikeCount", Value: *patch.DislikeCount})
	}
	if patch.TopLevelCommentCount != nil {
		updates = append(updates, firestore.Update{Path: "TopLevelCommentCount", Value: *patch.TopLevelCommentCount})
	}
	if patch.TotalCommentCount != nil {
		updates = append(updates, firestore.Update{Path: "TotalCommentCount", Value: *patch.TotalCommentCount})
	}
	if patch.PinnedCommentID != nil {
		updates = append(updates, firestore.Update{Path: "PinnedCommentID", Value: *patch.PinnedCommentID})
	}

	now := time.Now()
	updates = append(updates, firestore.Update{Path: "UpdatedAt", Value: &now})

	_, err := a.root.rootDoc(id).Update(ctx, updates)
	if err != nil {
		if isNotFoundErr(err) {
			return tbReview.TokenBlueprintReviewAggregate{}, errNotFound
		}
		return tbReview.TokenBlueprintReviewAggregate{}, err
	}

	return a.GetByID(ctx, id)
}

func (a *tokenBlueprintAggregateRepoFS) Delete(ctx context.Context, id string) error {
	if a.root == nil || a.root.fs == nil {
		return errTBReviewNotConfigured
	}
	_, err := a.root.rootDoc(id).Delete(ctx)
	return err
}

func (a *tokenBlueprintAggregateRepoFS) List(ctx context.Context, filter tbReview.FilterTokenBlueprintReviewAggregate, sort common.Sort, page common.Page) (common.PageResult[tbReview.TokenBlueprintReviewAggregate], error) {
	if a.root == nil || a.root.fs == nil {
		return common.PageResult[tbReview.TokenBlueprintReviewAggregate]{}, errTBReviewNotConfigured
	}

	q := a.root.fs.Collection(a.root.rootCol).Query

	if filter.TokenBlueprintID != "" {
		q = q.Where(firestore.DocumentID, "==", filter.TokenBlueprintID)
	}

	if filter.Created.From != nil {
		q = q.Where("CreatedAt", ">=", *filter.Created.From)
	}
	if filter.Created.To != nil {
		q = q.Where("CreatedAt", "<=", *filter.Created.To)
	}
	if filter.Updated.From != nil {
		q = q.Where("UpdatedAt", ">=", *filter.Updated.From)
	}
	if filter.Updated.To != nil {
		q = q.Where("UpdatedAt", "<=", *filter.Updated.To)
	}

	order := firestore.Asc
	if sort.Order == common.SortDesc {
		order = firestore.Desc
	}
	switch sort.Column {
	case "", "createdAt", "CreatedAt":
		q = q.OrderBy("CreatedAt", order)
	case "likeCount", "LikeCount":
		q = q.OrderBy("LikeCount", order)
	case "topLevelCommentCount", "TopLevelCommentCount":
		q = q.OrderBy("TopLevelCommentCount", order)
	case "totalCommentCount", "TotalCommentCount":
		q = q.OrderBy("TotalCommentCount", order)
	default:
		q = q.OrderBy("CreatedAt", order)
	}

	perPage := page.PerPage
	if perPage <= 0 {
		perPage = 20
	}
	pageNo := page.Number
	if pageNo <= 0 {
		pageNo = 1
	}
	offset := (pageNo - 1) * perPage

	totalCount, _ := countDocs(ctx, q)
	q = q.Offset(offset).Limit(perPage)

	it := q.Documents(ctx)
	items := make([]tbReview.TokenBlueprintReviewAggregate, 0, perPage)
	for {
		s, err := it.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return common.PageResult[tbReview.TokenBlueprintReviewAggregate]{}, err
		}
		var v tbReview.TokenBlueprintReviewAggregate
		if err := s.DataTo(&v); err != nil {
			return common.PageResult[tbReview.TokenBlueprintReviewAggregate]{}, err
		}
		items = append(items, v)
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(perPage)))

	return common.PageResult[tbReview.TokenBlueprintReviewAggregate]{
		Items:      items,
		TotalCount: totalCount,
		TotalPages: totalPages,
		Page:       pageNo,
		PerPage:    perPage,
	}, nil
}

// ============================================================
// Comment Repo: tokenBlueprintReviews/{tokenBlueprintId}/comments/{commentId}
// ============================================================

type commentRepoFS struct {
	root *TokenBlueprintReviewRepositoryFS
}

func (r *commentRepoFS) GetByID(ctx context.Context, id string) (tbReview.Comment, error) {
	if r.root == nil || r.root.fs == nil {
		return tbReview.Comment{}, errTBReviewNotConfigured
	}

	it := r.root.fs.CollectionGroup("comments").Where("CommentID", "==", id).Limit(1).Documents(ctx)
	snap, err := it.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return tbReview.Comment{}, errNotFound
		}
		return tbReview.Comment{}, err
	}

	var out tbReview.Comment
	if err := snap.DataTo(&out); err != nil {
		return tbReview.Comment{}, err
	}
	return out, nil
}

func (r *commentRepoFS) Create(ctx context.Context, entity tbReview.Comment) (tbReview.Comment, error) {
	if entity.TokenBlueprintID == "" {
		return tbReview.Comment{}, errors.New("tokenBlueprint_review_fs: TokenBlueprintID is required for Comment.Create")
	}
	return r.CreateUnderParent(ctx, entity.TokenBlueprintID, entity)
}

func (r *commentRepoFS) Update(ctx context.Context, id string, patch tbReview.PatchComment) (tbReview.Comment, error) {
	if r.root == nil || r.root.fs == nil {
		return tbReview.Comment{}, errTBReviewNotConfigured
	}

	it := r.root.fs.CollectionGroup("comments").Where("CommentID", "==", id).Limit(1).Documents(ctx)
	snap, err := it.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return tbReview.Comment{}, errNotFound
		}
		return tbReview.Comment{}, err
	}

	updates := make([]firestore.Update, 0, 9)
	if patch.Body != nil {
		updates = append(updates, firestore.Update{Path: "Body", Value: *patch.Body})
	}
	if patch.Deleted != nil {
		updates = append(updates, firestore.Update{Path: "Deleted", Value: *patch.Deleted})
	}
	if patch.IsOwnerComment != nil {
		updates = append(updates, firestore.Update{Path: "IsOwnerComment", Value: *patch.IsOwnerComment})
	}
	if patch.LikeCount != nil {
		updates = append(updates, firestore.Update{Path: "LikeCount", Value: *patch.LikeCount})
	}
	if patch.DislikeCount != nil {
		updates = append(updates, firestore.Update{Path: "DislikeCount", Value: *patch.DislikeCount})
	}
	if patch.ChildCount != nil {
		updates = append(updates, firestore.Update{Path: "ChildCount", Value: *patch.ChildCount})
	}

	now := time.Now()
	updates = append(updates, firestore.Update{Path: "UpdatedAt", Value: &now})

	_, err = snap.Ref.Update(ctx, updates)
	if err != nil {
		return tbReview.Comment{}, err
	}

	after, err := snap.Ref.Get(ctx)
	if err != nil {
		return tbReview.Comment{}, err
	}
	var out tbReview.Comment
	if err := after.DataTo(&out); err != nil {
		return tbReview.Comment{}, err
	}
	return out, nil
}

func (r *commentRepoFS) UpdateUnderParent(ctx context.Context, tokenBlueprintID, commentID string, patch tbReview.PatchComment) (tbReview.Comment, error) {
	if r.root == nil || r.root.fs == nil {
		return tbReview.Comment{}, errTBReviewNotConfigured
	}
	if tokenBlueprintID == "" {
		return tbReview.Comment{}, errors.New("tokenBlueprint_review_fs: tokenBlueprintID is required for Comment.UpdateUnderParent")
	}
	if commentID == "" {
		return tbReview.Comment{}, errors.New("tokenBlueprint_review_fs: commentID is required for Comment.UpdateUnderParent")
	}

	docRef := r.root.commentsCol(tokenBlueprintID).Doc(commentID)

	updates := make([]firestore.Update, 0, 9)
	if patch.Body != nil {
		updates = append(updates, firestore.Update{Path: "Body", Value: *patch.Body})
	}
	if patch.Deleted != nil {
		updates = append(updates, firestore.Update{Path: "Deleted", Value: *patch.Deleted})
	}
	if patch.IsOwnerComment != nil {
		updates = append(updates, firestore.Update{Path: "IsOwnerComment", Value: *patch.IsOwnerComment})
	}
	if patch.LikeCount != nil {
		updates = append(updates, firestore.Update{Path: "LikeCount", Value: *patch.LikeCount})
	}
	if patch.DislikeCount != nil {
		updates = append(updates, firestore.Update{Path: "DislikeCount", Value: *patch.DislikeCount})
	}
	if patch.ChildCount != nil {
		updates = append(updates, firestore.Update{Path: "ChildCount", Value: *patch.ChildCount})
	}

	now := time.Now()
	updates = append(updates, firestore.Update{Path: "UpdatedAt", Value: &now})

	_, err := docRef.Update(ctx, updates)
	if err != nil {
		if isNotFoundErr(err) {
			return tbReview.Comment{}, errNotFound
		}
		return tbReview.Comment{}, err
	}

	after, err := docRef.Get(ctx)
	if err != nil {
		if isNotFoundErr(err) {
			return tbReview.Comment{}, errNotFound
		}
		return tbReview.Comment{}, err
	}

	var out tbReview.Comment
	if err := after.DataTo(&out); err != nil {
		return tbReview.Comment{}, err
	}
	return out, nil
}

func (r *commentRepoFS) Delete(ctx context.Context, id string) error {
	if r.root == nil || r.root.fs == nil {
		return errTBReviewNotConfigured
	}

	it := r.root.fs.CollectionGroup("comments").Where("CommentID", "==", id).Limit(1).Documents(ctx)
	snap, err := it.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return errNotFound
		}
		return err
	}

	_, err = snap.Ref.Delete(ctx)
	return err
}

func (r *commentRepoFS) List(ctx context.Context, filter tbReview.FilterComment, sort common.Sort, page common.Page) (common.PageResult[tbReview.Comment], error) {
	if r.root == nil || r.root.fs == nil {
		return common.PageResult[tbReview.Comment]{}, errTBReviewNotConfigured
	}
	if filter.TokenBlueprintID == "" {
		return common.PageResult[tbReview.Comment]{}, errors.New("tokenBlueprint_review_fs: TokenBlueprintID is required for Comment.List")
	}

	q := r.root.commentsCol(filter.TokenBlueprintID).Query

	if filter.ParentCommentID != nil {
		q = q.Where("ParentCommentID", "==", *filter.ParentCommentID)
	}
	if filter.RootCommentID != "" {
		q = q.Where("RootCommentID", "==", filter.RootCommentID)
	}
	if filter.AuthorID != "" {
		q = q.Where("AuthorID", "==", filter.AuthorID)
	}
	if filter.AuthorType != nil {
		q = q.Where("AuthorType", "==", *filter.AuthorType)
	}
	if filter.IsOwnerComment != nil {
		q = q.Where("IsOwnerComment", "==", *filter.IsOwnerComment)
	}
	if filter.Deleted != nil {
		q = q.Where("Deleted", "==", *filter.Deleted)
	}
	if filter.Depth != nil {
		q = q.Where("Depth", "==", *filter.Depth)
	}

	if filter.Created.From != nil {
		q = q.Where("CreatedAt", ">=", *filter.Created.From)
	}
	if filter.Created.To != nil {
		q = q.Where("CreatedAt", "<=", *filter.Created.To)
	}
	if filter.Updated.From != nil {
		q = q.Where("UpdatedAt", ">=", *filter.Updated.From)
	}
	if filter.Updated.To != nil {
		q = q.Where("UpdatedAt", "<=", *filter.Updated.To)
	}

	order := firestore.Asc
	if sort.Order == common.SortDesc {
		order = firestore.Desc
	}
	switch sort.Column {
	case "", "createdAt", "CreatedAt":
		q = q.OrderBy("CreatedAt", order)
	case "likeCount", "LikeCount":
		q = q.OrderBy("LikeCount", order)
	case "childCount", "ChildCount":
		q = q.OrderBy("ChildCount", order)
	case "depth", "Depth":
		q = q.OrderBy("Depth", order)
	default:
		q = q.OrderBy("CreatedAt", order)
	}

	perPage := page.PerPage
	if perPage <= 0 {
		perPage = 20
	}
	pageNo := page.Number
	if pageNo <= 0 {
		pageNo = 1
	}
	offset := (pageNo - 1) * perPage

	totalCount, _ := countDocs(ctx, q)
	q = q.Offset(offset).Limit(perPage)

	it := q.Documents(ctx)
	items := make([]tbReview.Comment, 0, perPage)
	for {
		s, err := it.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return common.PageResult[tbReview.Comment]{}, err
		}
		var v tbReview.Comment
		if err := s.DataTo(&v); err != nil {
			return common.PageResult[tbReview.Comment]{}, err
		}
		items = append(items, v)
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(perPage)))

	return common.PageResult[tbReview.Comment]{
		Items:      items,
		TotalCount: totalCount,
		TotalPages: totalPages,
		Page:       pageNo,
		PerPage:    perPage,
	}, nil
}

// domain extra methods
func (r *commentRepoFS) GetByParentID(ctx context.Context, tokenBlueprintID, commentID string) (tbReview.Comment, error) {
	if r.root == nil || r.root.fs == nil {
		return tbReview.Comment{}, errTBReviewNotConfigured
	}
	snap, err := r.root.commentsCol(tokenBlueprintID).Doc(commentID).Get(ctx)
	if err != nil {
		if isNotFoundErr(err) {
			return tbReview.Comment{}, errNotFound
		}
		return tbReview.Comment{}, err
	}
	var out tbReview.Comment
	if err := snap.DataTo(&out); err != nil {
		return tbReview.Comment{}, err
	}
	return out, nil
}

func (r *commentRepoFS) ListByTokenBlueprintID(ctx context.Context, tokenBlueprintID string) ([]tbReview.Comment, error) {
	if r.root == nil || r.root.fs == nil {
		return nil, errTBReviewNotConfigured
	}
	if tokenBlueprintID == "" {
		return nil, errors.New("tokenBlueprint_review_fs: tokenBlueprintID is required")
	}

	q := r.root.commentsCol(tokenBlueprintID).Query.OrderBy("CreatedAt", firestore.Desc)

	it := q.Documents(ctx)
	out := make([]tbReview.Comment, 0, 50)
	for {
		s, err := it.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return nil, err
		}
		var v tbReview.Comment
		if err := s.DataTo(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (r *commentRepoFS) ListTopLevelByTokenBlueprintID(ctx context.Context, tokenBlueprintID string) ([]tbReview.Comment, error) {
	if r.root == nil || r.root.fs == nil {
		return nil, errTBReviewNotConfigured
	}
	if tokenBlueprintID == "" {
		return nil, errors.New("tokenBlueprint_review_fs: tokenBlueprintID is required")
	}

	q := r.root.commentsCol(tokenBlueprintID).
		Where("ParentCommentID", "==", "").
		OrderBy("CreatedAt", firestore.Desc)

	it := q.Documents(ctx)
	out := make([]tbReview.Comment, 0, 50)
	for {
		s, err := it.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return nil, err
		}
		var v tbReview.Comment
		if err := s.DataTo(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (r *commentRepoFS) ListByParentCommentID(ctx context.Context, tokenBlueprintID, parentCommentID string) ([]tbReview.Comment, error) {
	if r.root == nil || r.root.fs == nil {
		return nil, errTBReviewNotConfigured
	}
	if tokenBlueprintID == "" {
		return nil, errors.New("tokenBlueprint_review_fs: tokenBlueprintID is required")
	}
	if parentCommentID == "" {
		return nil, errors.New("tokenBlueprint_review_fs: parentCommentID is required")
	}

	q := r.root.commentsCol(tokenBlueprintID).
		Where("ParentCommentID", "==", parentCommentID).
		OrderBy("CreatedAt", firestore.Asc)

	it := q.Documents(ctx)
	out := make([]tbReview.Comment, 0, 50)
	for {
		s, err := it.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return nil, err
		}
		var v tbReview.Comment
		if err := s.DataTo(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (r *commentRepoFS) ListByRootCommentID(ctx context.Context, tokenBlueprintID, rootCommentID string) ([]tbReview.Comment, error) {
	if r.root == nil || r.root.fs == nil {
		return nil, errTBReviewNotConfigured
	}
	if tokenBlueprintID == "" {
		return nil, errors.New("tokenBlueprint_review_fs: tokenBlueprintID is required")
	}
	if rootCommentID == "" {
		return nil, errors.New("tokenBlueprint_review_fs: rootCommentID is required")
	}

	q := r.root.commentsCol(tokenBlueprintID).
		Where("RootCommentID", "==", rootCommentID).
		OrderBy("Depth", firestore.Asc).
		OrderBy("CreatedAt", firestore.Asc)

	it := q.Documents(ctx)
	out := make([]tbReview.Comment, 0, 100)
	for {
		s, err := it.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return nil, err
		}
		var v tbReview.Comment
		if err := s.DataTo(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (r *commentRepoFS) ListOwnerCommentsByTokenBlueprintID(ctx context.Context, tokenBlueprintID string) ([]tbReview.Comment, error) {
	if r.root == nil || r.root.fs == nil {
		return nil, errTBReviewNotConfigured
	}
	if tokenBlueprintID == "" {
		return nil, errors.New("tokenBlueprint_review_fs: tokenBlueprintID is required")
	}

	q := r.root.commentsCol(tokenBlueprintID).
		Where("IsOwnerComment", "==", true).
		OrderBy("CreatedAt", firestore.Desc)

	it := q.Documents(ctx)
	out := make([]tbReview.Comment, 0, 50)
	for {
		s, err := it.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return nil, err
		}
		var v tbReview.Comment
		if err := s.DataTo(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (r *commentRepoFS) CreateUnderParent(ctx context.Context, tokenBlueprintID string, comment tbReview.Comment) (tbReview.Comment, error) {
	if r.root == nil || r.root.fs == nil {
		return tbReview.Comment{}, errTBReviewNotConfigured
	}
	if tokenBlueprintID == "" {
		return tbReview.Comment{}, errors.New("tokenBlueprint_review_fs: tokenBlueprintID is required")
	}

	col := r.root.commentsCol(tokenBlueprintID)
	if comment.CommentID == "" {
		doc := col.NewDoc()
		comment.CommentID = doc.ID
	}

	comment.TokenBlueprintID = tokenBlueprintID

	if comment.RootCommentID == "" {
		if comment.ParentCommentID == "" {
			comment.RootCommentID = comment.CommentID
			comment.Depth = 0
		}
	}

	_, err := col.Doc(comment.CommentID).Set(ctx, comment)
	if err != nil {
		return tbReview.Comment{}, err
	}
	return comment, nil
}

func (r *commentRepoFS) DeleteUnderParent(ctx context.Context, tokenBlueprintID, commentID string) error {
	if r.root == nil || r.root.fs == nil {
		return errTBReviewNotConfigured
	}
	_, err := r.root.commentsCol(tokenBlueprintID).Doc(commentID).Delete(ctx)
	return err
}

// ============================================================
// Reactions
// ============================================================

type tokenBlueprintReactionRepoFS struct {
	root *TokenBlueprintReviewRepositoryFS
}

func (r *tokenBlueprintReactionRepoFS) GetByID(ctx context.Context, id string) (tbReview.TokenBlueprintReaction, error) {
	if r.root == nil || r.root.fs == nil {
		return tbReview.TokenBlueprintReaction{}, errTBReviewNotConfigured
	}

	it := r.root.fs.CollectionGroup("reactions").Where(firestore.DocumentID, "==", id).Limit(1).Documents(ctx)
	snap, err := it.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return tbReview.TokenBlueprintReaction{}, errNotFound
		}
		return tbReview.TokenBlueprintReaction{}, err
	}

	var out tbReview.TokenBlueprintReaction
	if err := snap.DataTo(&out); err != nil {
		return tbReview.TokenBlueprintReaction{}, err
	}
	return out, nil
}

func (r *tokenBlueprintReactionRepoFS) Create(ctx context.Context, entity tbReview.TokenBlueprintReaction) (tbReview.TokenBlueprintReaction, error) {
	return r.Upsert(ctx, entity)
}

func (r *tokenBlueprintReactionRepoFS) Update(ctx context.Context, id string, patch tbReview.PatchTokenBlueprintReaction) (tbReview.TokenBlueprintReaction, error) {
	if r.root == nil || r.root.fs == nil {
		return tbReview.TokenBlueprintReaction{}, errTBReviewNotConfigured
	}

	current, err := r.GetByID(ctx, id)
	if err != nil {
		return tbReview.TokenBlueprintReaction{}, err
	}

	if patch.Type != nil {
		current.Type = *patch.Type
	}
	current.UpdatedAt = time.Now()

	return r.Upsert(ctx, current)
}

func (r *tokenBlueprintReactionRepoFS) Delete(ctx context.Context, id string) error {
	if r.root == nil || r.root.fs == nil {
		return errTBReviewNotConfigured
	}

	it := r.root.fs.CollectionGroup("reactions").Where(firestore.DocumentID, "==", id).Limit(1).Documents(ctx)
	snap, err := it.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return errNotFound
		}
		return err
	}

	_, err = snap.Ref.Delete(ctx)
	return err
}

func (r *tokenBlueprintReactionRepoFS) List(ctx context.Context, filter tbReview.FilterTokenBlueprintReaction, sort common.Sort, page common.Page) (common.PageResult[tbReview.TokenBlueprintReaction], error) {
	if r.root == nil || r.root.fs == nil {
		return common.PageResult[tbReview.TokenBlueprintReaction]{}, errTBReviewNotConfigured
	}
	if filter.TokenBlueprintID == "" {
		return common.PageResult[tbReview.TokenBlueprintReaction]{}, errors.New("tokenBlueprint_review_fs: tokenBlueprintID is required for TokenBlueprintReaction.List")
	}

	q := r.root.tokenReactionsCol(filter.TokenBlueprintID).Query

	if filter.ActorID != "" {
		q = q.Where("ActorID", "==", filter.ActorID)
	}
	if filter.ActorType != nil {
		q = q.Where("ActorType", "==", *filter.ActorType)
	}
	if filter.Type != nil {
		q = q.Where("Type", "==", *filter.Type)
	}
	if filter.Created.From != nil {
		q = q.Where("CreatedAt", ">=", *filter.Created.From)
	}
	if filter.Created.To != nil {
		q = q.Where("CreatedAt", "<=", *filter.Created.To)
	}
	if filter.Updated.From != nil {
		q = q.Where("UpdatedAt", ">=", *filter.Updated.From)
	}
	if filter.Updated.To != nil {
		q = q.Where("UpdatedAt", "<=", *filter.Updated.To)
	}

	order := firestore.Asc
	if sort.Order == common.SortDesc {
		order = firestore.Desc
	}
	switch sort.Column {
	case "", "createdAt", "CreatedAt":
		q = q.OrderBy("CreatedAt", order)
	case "updatedAt", "UpdatedAt":
		q = q.OrderBy("UpdatedAt", order)
	default:
		q = q.OrderBy("CreatedAt", order)
	}

	perPage := page.PerPage
	if perPage <= 0 {
		perPage = 20
	}
	pageNo := page.Number
	if pageNo <= 0 {
		pageNo = 1
	}
	offset := (pageNo - 1) * perPage

	totalCount, _ := countDocs(ctx, q)
	q = q.Offset(offset).Limit(perPage)

	it := q.Documents(ctx)
	items := make([]tbReview.TokenBlueprintReaction, 0, perPage)
	for {
		s, err := it.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return common.PageResult[tbReview.TokenBlueprintReaction]{}, err
		}
		var v tbReview.TokenBlueprintReaction
		if err := s.DataTo(&v); err != nil {
			return common.PageResult[tbReview.TokenBlueprintReaction]{}, err
		}
		items = append(items, v)
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(perPage)))

	return common.PageResult[tbReview.TokenBlueprintReaction]{
		Items:      items,
		TotalCount: totalCount,
		TotalPages: totalPages,
		Page:       pageNo,
		PerPage:    perPage,
	}, nil
}

func (r *tokenBlueprintReactionRepoFS) FindByActor(ctx context.Context, tokenBlueprintID string, actorType tbReview.ActorType, actorID string) (tbReview.TokenBlueprintReaction, error) {
	if r.root == nil || r.root.fs == nil {
		return tbReview.TokenBlueprintReaction{}, errTBReviewNotConfigured
	}
	docID := tokenReactionDocID(actorType, actorID)
	snap, err := r.root.tokenReactionsCol(tokenBlueprintID).Doc(docID).Get(ctx)
	if err != nil {
		if isNotFoundErr(err) {
			return tbReview.TokenBlueprintReaction{}, errNotFound
		}
		return tbReview.TokenBlueprintReaction{}, err
	}

	var out tbReview.TokenBlueprintReaction
	if err := snap.DataTo(&out); err != nil {
		return tbReview.TokenBlueprintReaction{}, err
	}
	return out, nil
}

func (r *tokenBlueprintReactionRepoFS) FindByDocumentID(ctx context.Context, tokenBlueprintID, reactionDocumentID string) (tbReview.TokenBlueprintReaction, error) {
	if r.root == nil || r.root.fs == nil {
		return tbReview.TokenBlueprintReaction{}, errTBReviewNotConfigured
	}
	snap, err := r.root.tokenReactionsCol(tokenBlueprintID).Doc(reactionDocumentID).Get(ctx)
	if err != nil {
		if isNotFoundErr(err) {
			return tbReview.TokenBlueprintReaction{}, errNotFound
		}
		return tbReview.TokenBlueprintReaction{}, err
	}

	var out tbReview.TokenBlueprintReaction
	if err := snap.DataTo(&out); err != nil {
		return tbReview.TokenBlueprintReaction{}, err
	}
	return out, nil
}

func (r *tokenBlueprintReactionRepoFS) ListByTokenBlueprintID(ctx context.Context, tokenBlueprintID string) ([]tbReview.TokenBlueprintReaction, error) {
	if r.root == nil || r.root.fs == nil {
		return nil, errTBReviewNotConfigured
	}
	if tokenBlueprintID == "" {
		return nil, errors.New("tokenBlueprint_review_fs: tokenBlueprintID is required")
	}

	q := r.root.tokenReactionsCol(tokenBlueprintID).Query.OrderBy("CreatedAt", firestore.Desc)

	it := q.Documents(ctx)
	out := make([]tbReview.TokenBlueprintReaction, 0, 50)
	for {
		s, err := it.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return nil, err
		}
		var v tbReview.TokenBlueprintReaction
		if err := s.DataTo(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (r *tokenBlueprintReactionRepoFS) ListByActor(ctx context.Context, actorType tbReview.ActorType, actorID string) ([]tbReview.TokenBlueprintReaction, error) {
	if r.root == nil || r.root.fs == nil {
		return nil, errTBReviewNotConfigured
	}
	if actorID == "" {
		return nil, errors.New("tokenBlueprint_review_fs: actorID is required")
	}

	it := r.root.fs.CollectionGroup("reactions").
		Where("ActorType", "==", actorType).
		Where("ActorID", "==", actorID).
		Documents(ctx)

	out := make([]tbReview.TokenBlueprintReaction, 0, 50)
	for {
		s, err := it.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return nil, err
		}
		var v tbReview.TokenBlueprintReaction
		if err := s.DataTo(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (r *tokenBlueprintReactionRepoFS) Upsert(ctx context.Context, reaction tbReview.TokenBlueprintReaction) (tbReview.TokenBlueprintReaction, error) {
	if r.root == nil || r.root.fs == nil {
		return tbReview.TokenBlueprintReaction{}, errTBReviewNotConfigured
	}
	if reaction.TokenBlueprintID == "" || reaction.ActorID == "" {
		return tbReview.TokenBlueprintReaction{}, errors.New("tokenBlueprint_review_fs: TokenBlueprintID and ActorID are required for TokenBlueprintReaction.Upsert")
	}
	if err := reaction.ActorType.Validate(); err != nil {
		return tbReview.TokenBlueprintReaction{}, err
	}

	docID, err := reaction.ReactionDocumentID()
	if err != nil {
		return tbReview.TokenBlueprintReaction{}, err
	}

	_, err = r.root.tokenReactionsCol(reaction.TokenBlueprintID).Doc(docID).Set(ctx, reaction)
	if err != nil {
		return tbReview.TokenBlueprintReaction{}, err
	}
	return reaction, nil
}

func (r *tokenBlueprintReactionRepoFS) DeleteByActor(ctx context.Context, tokenBlueprintID string, actorType tbReview.ActorType, actorID string) error {
	if r.root == nil || r.root.fs == nil {
		return errTBReviewNotConfigured
	}
	docID := tokenReactionDocID(actorType, actorID)
	_, err := r.root.tokenReactionsCol(tokenBlueprintID).Doc(docID).Delete(ctx)
	return err
}

type commentReactionRepoFS struct {
	root *TokenBlueprintReviewRepositoryFS
}

func (r *commentReactionRepoFS) GetByID(ctx context.Context, id string) (tbReview.CommentReaction, error) {
	if r.root == nil || r.root.fs == nil {
		return tbReview.CommentReaction{}, errTBReviewNotConfigured
	}

	it := r.root.fs.CollectionGroup("reactions").Where(firestore.DocumentID, "==", id).Limit(1).Documents(ctx)
	snap, err := it.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return tbReview.CommentReaction{}, errNotFound
		}
		return tbReview.CommentReaction{}, err
	}

	var out tbReview.CommentReaction
	if err := snap.DataTo(&out); err != nil {
		return tbReview.CommentReaction{}, err
	}
	return out, nil
}

func (r *commentReactionRepoFS) Create(ctx context.Context, entity tbReview.CommentReaction) (tbReview.CommentReaction, error) {
	return r.Upsert(ctx, entity)
}

func (r *commentReactionRepoFS) Update(ctx context.Context, id string, patch tbReview.PatchCommentReaction) (tbReview.CommentReaction, error) {
	if r.root == nil || r.root.fs == nil {
		return tbReview.CommentReaction{}, errTBReviewNotConfigured
	}

	current, err := r.GetByID(ctx, id)
	if err != nil {
		return tbReview.CommentReaction{}, err
	}

	if patch.Type != nil {
		current.Type = *patch.Type
	}
	current.UpdatedAt = time.Now()

	return r.Upsert(ctx, current)
}

func (r *commentReactionRepoFS) Delete(ctx context.Context, id string) error {
	if r.root == nil || r.root.fs == nil {
		return errTBReviewNotConfigured
	}

	it := r.root.fs.CollectionGroup("reactions").Where(firestore.DocumentID, "==", id).Limit(1).Documents(ctx)
	snap, err := it.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return errNotFound
		}
		return err
	}

	_, err = snap.Ref.Delete(ctx)
	return err
}

func (r *commentReactionRepoFS) List(ctx context.Context, filter tbReview.FilterCommentReaction, sort common.Sort, page common.Page) (common.PageResult[tbReview.CommentReaction], error) {
	if r.root == nil || r.root.fs == nil {
		return common.PageResult[tbReview.CommentReaction]{}, errTBReviewNotConfigured
	}
	if filter.TokenBlueprintID == "" || filter.CommentID == "" {
		return common.PageResult[tbReview.CommentReaction]{}, errors.New("tokenBlueprint_review_fs: tokenBlueprintID and commentID are required for CommentReaction.List")
	}

	q := r.root.commentReactionsCol(filter.TokenBlueprintID, filter.CommentID).Query

	if filter.ActorID != "" {
		q = q.Where("ActorID", "==", filter.ActorID)
	}
	if filter.ActorType != nil {
		q = q.Where("ActorType", "==", *filter.ActorType)
	}
	if filter.Type != nil {
		q = q.Where("Type", "==", *filter.Type)
	}
	if filter.Created.From != nil {
		q = q.Where("CreatedAt", ">=", *filter.Created.From)
	}
	if filter.Created.To != nil {
		q = q.Where("CreatedAt", "<=", *filter.Created.To)
	}
	if filter.Updated.From != nil {
		q = q.Where("UpdatedAt", ">=", *filter.Updated.From)
	}
	if filter.Updated.To != nil {
		q = q.Where("UpdatedAt", "<=", *filter.Updated.To)
	}

	order := firestore.Asc
	if sort.Order == common.SortDesc {
		order = firestore.Desc
	}
	switch sort.Column {
	case "", "createdAt", "CreatedAt":
		q = q.OrderBy("CreatedAt", order)
	case "updatedAt", "UpdatedAt":
		q = q.OrderBy("UpdatedAt", order)
	default:
		q = q.OrderBy("CreatedAt", order)
	}

	perPage := page.PerPage
	if perPage <= 0 {
		perPage = 20
	}
	pageNo := page.Number
	if pageNo <= 0 {
		pageNo = 1
	}
	offset := (pageNo - 1) * perPage

	totalCount, _ := countDocs(ctx, q)
	q = q.Offset(offset).Limit(perPage)

	it := q.Documents(ctx)
	items := make([]tbReview.CommentReaction, 0, perPage)
	for {
		s, err := it.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return common.PageResult[tbReview.CommentReaction]{}, err
		}
		var v tbReview.CommentReaction
		if err := s.DataTo(&v); err != nil {
			return common.PageResult[tbReview.CommentReaction]{}, err
		}
		items = append(items, v)
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(perPage)))

	return common.PageResult[tbReview.CommentReaction]{
		Items:      items,
		TotalCount: totalCount,
		TotalPages: totalPages,
		Page:       pageNo,
		PerPage:    perPage,
	}, nil
}

func (r *commentReactionRepoFS) FindByActor(ctx context.Context, tokenBlueprintID, commentID string, actorType tbReview.ActorType, actorID string) (tbReview.CommentReaction, error) {
	if r.root == nil || r.root.fs == nil {
		return tbReview.CommentReaction{}, errTBReviewNotConfigured
	}

	docID := tokenReactionDocID(actorType, actorID)
	snap, err := r.root.commentReactionsCol(tokenBlueprintID, commentID).Doc(docID).Get(ctx)
	if err != nil {
		if isNotFoundErr(err) {
			return tbReview.CommentReaction{}, errNotFound
		}
		return tbReview.CommentReaction{}, err
	}

	var out tbReview.CommentReaction
	if err := snap.DataTo(&out); err != nil {
		return tbReview.CommentReaction{}, err
	}
	return out, nil
}

func (r *commentReactionRepoFS) FindByDocumentID(ctx context.Context, tokenBlueprintID, commentID, reactionDocumentID string) (tbReview.CommentReaction, error) {
	if r.root == nil || r.root.fs == nil {
		return tbReview.CommentReaction{}, errTBReviewNotConfigured
	}

	snap, err := r.root.commentReactionsCol(tokenBlueprintID, commentID).Doc(reactionDocumentID).Get(ctx)
	if err != nil {
		if isNotFoundErr(err) {
			return tbReview.CommentReaction{}, errNotFound
		}
		return tbReview.CommentReaction{}, err
	}

	var out tbReview.CommentReaction
	if err := snap.DataTo(&out); err != nil {
		return tbReview.CommentReaction{}, err
	}
	return out, nil
}

func (r *commentReactionRepoFS) ListByCommentID(ctx context.Context, tokenBlueprintID, commentID string) ([]tbReview.CommentReaction, error) {
	if r.root == nil || r.root.fs == nil {
		return nil, errTBReviewNotConfigured
	}
	if tokenBlueprintID == "" || commentID == "" {
		return nil, errors.New("tokenBlueprint_review_fs: tokenBlueprintID and commentID are required")
	}

	q := r.root.commentReactionsCol(tokenBlueprintID, commentID).Query.OrderBy("CreatedAt", firestore.Desc)

	it := q.Documents(ctx)
	out := make([]tbReview.CommentReaction, 0, 50)
	for {
		s, err := it.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return nil, err
		}
		var v tbReview.CommentReaction
		if err := s.DataTo(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (r *commentReactionRepoFS) ListByActor(ctx context.Context, actorType tbReview.ActorType, actorID string) ([]tbReview.CommentReaction, error) {
	if r.root == nil || r.root.fs == nil {
		return nil, errTBReviewNotConfigured
	}
	if actorID == "" {
		return nil, errors.New("tokenBlueprint_review_fs: actorID is required")
	}

	it := r.root.fs.CollectionGroup("reactions").
		Where("ActorType", "==", actorType).
		Where("ActorID", "==", actorID).
		Documents(ctx)

	out := make([]tbReview.CommentReaction, 0, 50)
	for {
		s, err := it.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return nil, err
		}
		var v tbReview.CommentReaction
		if err := s.DataTo(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (r *commentReactionRepoFS) Upsert(ctx context.Context, reaction tbReview.CommentReaction) (tbReview.CommentReaction, error) {
	if r.root == nil || r.root.fs == nil {
		return tbReview.CommentReaction{}, errTBReviewNotConfigured
	}
	if reaction.TokenBlueprintID == "" || reaction.CommentID == "" || reaction.ActorID == "" {
		return tbReview.CommentReaction{}, errors.New("tokenBlueprint_review_fs: TokenBlueprintID, CommentID, ActorID are required for CommentReaction.Upsert")
	}
	if err := reaction.ActorType.Validate(); err != nil {
		return tbReview.CommentReaction{}, err
	}

	docID, err := reaction.ReactionDocumentID()
	if err != nil {
		return tbReview.CommentReaction{}, err
	}

	_, err = r.root.commentReactionsCol(reaction.TokenBlueprintID, reaction.CommentID).Doc(docID).Set(ctx, reaction)
	if err != nil {
		return tbReview.CommentReaction{}, err
	}
	return reaction, nil
}

func (r *commentReactionRepoFS) DeleteByActor(ctx context.Context, tokenBlueprintID, commentID string, actorType tbReview.ActorType, actorID string) error {
	if r.root == nil || r.root.fs == nil {
		return errTBReviewNotConfigured
	}
	docID := tokenReactionDocID(actorType, actorID)
	_, err := r.root.commentReactionsCol(tokenBlueprintID, commentID).Doc(docID).Delete(ctx)
	return err
}

// ============================================================
// small helper
// ============================================================

func countDocs(ctx context.Context, q firestore.Query) (int, error) {
	it := q.Documents(ctx)
	n := 0
	for {
		_, err := it.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				return n, nil
			}
			return 0, err
		}
		n++
	}
}
