// backend/internal/application/usecase/tokenBlueprint_review_usecase.go
package usecase

import (
	"context"
	"errors"
	"strconv"
	"time"

	avatar "narratives/internal/domain/avatar"
	brand "narratives/internal/domain/brand"
	common "narratives/internal/domain/common"
	tokenBlueprint "narratives/internal/domain/tokenBlueprint"
	tokenBlueprint_review "narratives/internal/domain/tokenBlueprint_review"
)

// TokenBlueprintReviewUsecase provides application-level orchestration for
// token blueprint reviews.
//
// Hexagonal architecture policy:
// - this usecase owns application orchestration
// - handlers must not update repositories directly for comments/reactions/aggregates
// - domain entities own invariant/state transition methods
// - repositories are outbound ports
//
// Query separation policy:
// - query services may import this usecase.
// - this usecase must not import query packages.
// - console / mall query services decide actor context, such as avatar or brand.
// - this usecase provides shared maximum-common read orchestration.
// - this usecase may compose shared view models that are independent of console / mall.
// - this usecase must not decide whether current actor should be avatar or brand.
type TokenBlueprintReviewUsecase struct {
	repos              tokenBlueprint_review.RepositoryPort
	avatarRepos        avatar.Repository
	tokenBlueprintRepo tokenBlueprint.RepositoryPort
	brandRepo          brand.Repository

	now func() time.Time
}

var (
	errReviewReposNotConfigured        = errors.New("tokenBlueprint_review_usecase: repository port not configured")
	errUsecaseNotConfigured            = errors.New("tokenBlueprint_review_usecase: avatar repository not configured")
	errTokenBlueprintRepoNotConfigured = errors.New("tokenBlueprint_review_usecase: token blueprint repository not configured")
	errBrandRepositoryNotConfigured    = errors.New("tokenBlueprint_review_usecase: brand repository not configured")
	errTokenBlueprintIDRequired        = errors.New("tokenBlueprint_review_usecase: tokenBlueprintID is required")
)

// NewTokenBlueprintReviewUsecase is the only construction entry point for
// TokenBlueprintReviewUsecase.
//
// Do not construct TokenBlueprintReviewUsecase directly from handlers or query
// services. Wire dependencies here, then pass the resulting usecase to
// console / mall query services.
func NewTokenBlueprintReviewUsecase(
	repos tokenBlueprint_review.RepositoryPort,
	avatarRepos avatar.Repository,
	tokenBlueprintRepo tokenBlueprint.RepositoryPort,
	brandRepo brand.Repository,
) *TokenBlueprintReviewUsecase {
	return &TokenBlueprintReviewUsecase{
		repos:              repos,
		avatarRepos:        avatarRepos,
		tokenBlueprintRepo: tokenBlueprintRepo,
		brandRepo:          brandRepo,
		now:                time.Now,
	}
}

// ============================================================
// Avatar / Brand lightweight getters
// ============================================================

func (u *TokenBlueprintReviewUsecase) GetNameAndIconByID(
	ctx context.Context,
	avatarID string,
) (name string, icon string, err error) {
	if u == nil || u.avatarRepos == nil {
		return "", "", errUsecaseNotConfigured
	}

	a, err := u.avatarRepos.GetByID(ctx, avatarID)
	if err != nil {
		return "", "", err
	}

	if a.AvatarIcon != nil {
		icon = *a.AvatarIcon
	}

	return a.AvatarName, icon, nil
}

func (u *TokenBlueprintReviewUsecase) GetBrandNameAndIconByID(
	ctx context.Context,
	brandID string,
) (name string, icon string, err error) {
	if u == nil || u.brandRepo == nil {
		return "", "", errBrandRepositoryNotConfigured
	}

	b, err := u.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return "", "", err
	}

	return b.Name, b.BrandIcon, nil
}

// ============================================================
// Shared view DTOs for console / mall query services
// ============================================================

type CommentView struct {
	tokenBlueprint_review.Comment

	AuthorAvatarName string  `json:"AuthorAvatarName,omitempty"`
	AuthorAvatarIcon *string `json:"AuthorAvatarIcon,omitempty"`

	BrandName string  `json:"BrandName,omitempty"`
	BrandIcon *string `json:"BrandIcon,omitempty"`
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (u *TokenBlueprintReviewUsecase) BuildComment(
	ctx context.Context,
	comment tokenBlueprint_review.Comment,
) CommentView {
	view := CommentView{
		Comment: comment,
	}

	switch comment.AuthorType {
	case tokenBlueprint_review.AuthorTypeAvatar:
		name, icon, err := u.GetNameAndIconByID(ctx, comment.AuthorID)
		if err == nil {
			view.AuthorAvatarName = name
			view.AuthorAvatarIcon = strPtrOrNil(icon)
		}

	case tokenBlueprint_review.AuthorTypeBrand:
		name, icon, err := u.GetBrandNameAndIconByID(ctx, comment.AuthorID)
		if err == nil {
			view.BrandName = name
			view.BrandIcon = strPtrOrNil(icon)
		}
	}

	return view
}

func (u *TokenBlueprintReviewUsecase) BuildComments(
	ctx context.Context,
	comments []tokenBlueprint_review.Comment,
) []CommentView {
	out := make([]CommentView, 0, len(comments))
	for _, c := range comments {
		out = append(out, u.BuildComment(ctx, c))
	}
	return out
}

// ============================================================
// Internal helpers
// ============================================================

func newCommentID(now time.Time) string {
	return "cm_" + strconv.FormatInt(now.UnixNano(), 10)
}

func (u *TokenBlueprintReviewUsecase) ensureConfigured() error {
	if u == nil || u.repos == nil {
		return errReviewReposNotConfigured
	}
	return nil
}

func (u *TokenBlueprintReviewUsecase) ensureAggregate(
	ctx context.Context,
	tokenBlueprintID string,
	now time.Time,
) (tokenBlueprint_review.TokenBlueprintReviewAggregate, error) {
	if err := u.ensureConfigured(); err != nil {
		return tokenBlueprint_review.TokenBlueprintReviewAggregate{}, err
	}

	aggRepo := u.repos.TokenBlueprintAggregates()

	agg, err := aggRepo.GetByID(ctx, tokenBlueprintID)
	if err == nil {
		return agg, nil
	}

	created, cerr := tokenBlueprint_review.NewTokenBlueprintReviewAggregate(tokenBlueprintID, now)
	if cerr != nil {
		return tokenBlueprint_review.TokenBlueprintReviewAggregate{}, cerr
	}

	agg, err = aggRepo.Create(ctx, *created)
	if err != nil {
		return tokenBlueprint_review.TokenBlueprintReviewAggregate{}, err
	}

	return agg, nil
}

func (u *TokenBlueprintReviewUsecase) updateAggregate(
	ctx context.Context,
	tokenBlueprintID string,
	agg tokenBlueprint_review.TokenBlueprintReviewAggregate,
) (tokenBlueprint_review.TokenBlueprintReviewAggregate, error) {
	if err := u.ensureConfigured(); err != nil {
		return tokenBlueprint_review.TokenBlueprintReviewAggregate{}, err
	}

	return u.repos.TokenBlueprintAggregates().Update(
		ctx,
		tokenBlueprintID,
		tokenBlueprint_review.NewPatchFromTokenBlueprintReviewAggregate(agg),
	)
}

func (u *TokenBlueprintReviewUsecase) incrementParentChildCount(
	ctx context.Context,
	tokenBlueprintID string,
	parentCommentID string,
	now time.Time,
) error {
	if parentCommentID == "" {
		return nil
	}
	if err := u.ensureConfigured(); err != nil {
		return err
	}

	parent, err := u.repos.Comments().GetByParentID(ctx, tokenBlueprintID, parentCommentID)
	if err != nil {
		return err
	}

	parent.IncrementChildCount(now)

	_, err = u.repos.Comments().UpdateUnderParent(
		ctx,
		tokenBlueprintID,
		parent.CommentID,
		tokenBlueprint_review.NewChildCountPatchFromComment(parent),
	)
	return err
}

func (u *TokenBlueprintReviewUsecase) decrementParentChildCount(
	ctx context.Context,
	tokenBlueprintID string,
	parentCommentID string,
	now time.Time,
) error {
	if parentCommentID == "" {
		return nil
	}
	if err := u.ensureConfigured(); err != nil {
		return err
	}

	parent, err := u.repos.Comments().GetByParentID(ctx, tokenBlueprintID, parentCommentID)
	if err != nil {
		return err
	}

	if err := parent.DecrementChildCount(now); err != nil {
		return err
	}

	_, err = u.repos.Comments().UpdateUnderParent(
		ctx,
		tokenBlueprintID,
		parent.CommentID,
		tokenBlueprint_review.NewChildCountPatchFromComment(parent),
	)
	return err
}

// ============================================================
// TokenBlueprint lightweight getter
// ============================================================

func (u *TokenBlueprintReviewUsecase) GetTokenBlueprintPatchByID(
	ctx context.Context,
	tokenBlueprintID string,
) (tokenBlueprint.Patch, error) {
	if u == nil || u.tokenBlueprintRepo == nil {
		return tokenBlueprint.Patch{}, errTokenBlueprintRepoNotConfigured
	}

	tb, err := u.tokenBlueprintRepo.GetByID(ctx, tokenBlueprintID)
	if err != nil {
		return tokenBlueprint.Patch{}, err
	}
	if tb == nil {
		return tokenBlueprint.Patch{}, errors.New("tokenBlueprint_review_usecase: token blueprint not found")
	}

	patch := tokenBlueprint.Patch{
		ID:          tb.ID,
		TokenName:   tb.Name,
		Symbol:      tb.Symbol,
		BrandID:     tb.BrandID,
		CompanyID:   tb.CompanyID,
		Description: tb.Description,
		Minted:      tb.Minted,
		MetadataURI: tb.MetadataURI,
		IconURL:     tb.IconURL,
	}

	if patch.BrandID != "" && u.brandRepo != nil {
		if b, berr := u.brandRepo.GetByID(ctx, patch.BrandID); berr == nil && b.Name != "" {
			patch.BrandName = b.Name
		}
	}

	return patch, nil
}

// ============================================================
// Aggregates
// ============================================================

func (u *TokenBlueprintReviewUsecase) GetAggregate(
	ctx context.Context,
	tokenBlueprintID string,
) (tokenBlueprint_review.TokenBlueprintReviewAggregate, error) {
	if err := u.ensureConfigured(); err != nil {
		return tokenBlueprint_review.TokenBlueprintReviewAggregate{}, err
	}

	return u.repos.TokenBlueprintAggregates().GetByID(ctx, tokenBlueprintID)
}

func (u *TokenBlueprintReviewUsecase) ListAggregatesByCompanyTokenBlueprints(
	ctx context.Context,
	companyID string,
) ([]tokenBlueprint_review.TokenBlueprintReviewAggregate, error) {
	if err := u.ensureConfigured(); err != nil {
		return nil, err
	}
	if u.tokenBlueprintRepo == nil {
		return nil, errTokenBlueprintRepoNotConfigured
	}

	tbIDs, err := u.listAllTokenBlueprintIDsByCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}

	aggRepo := u.repos.TokenBlueprintAggregates()

	items := make([]tokenBlueprint_review.TokenBlueprintReviewAggregate, 0, len(tbIDs))
	for _, tbid := range tbIDs {
		agg, err := aggRepo.GetByID(ctx, tbid)
		if err != nil {
			continue
		}
		items = append(items, agg)
	}

	return items, nil
}

func (u *TokenBlueprintReviewUsecase) listAllTokenBlueprintIDsByCompany(
	ctx context.Context,
	companyID string,
) ([]string, error) {
	if u == nil || u.tokenBlueprintRepo == nil {
		return nil, errTokenBlueprintRepoNotConfigured
	}

	ids := make([]string, 0, 128)

	pageNo := 1
	perPage := 200

	for {
		res, err := u.tokenBlueprintRepo.ListByCompanyID(ctx, companyID, common.Page{
			Number:  pageNo,
			PerPage: perPage,
		})
		if err != nil {
			return nil, err
		}

		for _, tb := range res.Items {
			if tb.ID != "" {
				ids = append(ids, tb.ID)
			}
		}

		if res.TotalPages <= 0 || pageNo >= res.TotalPages || len(res.Items) == 0 {
			break
		}

		pageNo++
	}

	return ids, nil
}

// ============================================================
// TokenBlueprint reaction command
// ============================================================

type TokenBlueprintReactionResult struct {
	Aggregate tokenBlueprint_review.TokenBlueprintReviewAggregate
	Reaction  tokenBlueprint_review.TokenBlueprintReaction
}

func (u *TokenBlueprintReviewUsecase) ReactToTokenBlueprintDetailed(
	ctx context.Context,
	tokenBlueprintID string,
	actorID string,
	actorType tokenBlueprint_review.ActorType,
	newType tokenBlueprint_review.ReactionType,
) (TokenBlueprintReactionResult, error) {
	if err := u.ensureConfigured(); err != nil {
		return TokenBlueprintReactionResult{}, err
	}

	if err := actorType.Validate(); err != nil {
		return TokenBlueprintReactionResult{}, err
	}

	pressedType := newType
	if err := pressedType.Validate(); err != nil {
		return TokenBlueprintReactionResult{}, err
	}

	now := u.now()

	oldType := tokenBlueprint_review.ReactionComment
	if ex, err := u.repos.TokenBlueprintReactions().FindByActor(ctx, tokenBlueprintID, actorType, actorID); err == nil {
		oldType = ex.Type
	}

	nextType, err := tokenBlueprint_review.NextReactionType(oldType, pressedType)
	if err != nil {
		return TokenBlueprintReactionResult{}, err
	}

	agg, err := u.ensureAggregate(ctx, tokenBlueprintID, now)
	if err != nil {
		return TokenBlueprintReactionResult{}, err
	}

	if err := agg.ApplyReaction(oldType, nextType, now); err != nil {
		return TokenBlueprintReactionResult{}, err
	}

	reaction, err := tokenBlueprint_review.NewTokenBlueprintReaction(
		tokenBlueprintID,
		actorID,
		actorType,
		nextType,
		now,
	)
	if err != nil {
		return TokenBlueprintReactionResult{}, err
	}

	savedReaction, err := u.repos.TokenBlueprintReactions().Upsert(ctx, *reaction)
	if err != nil {
		return TokenBlueprintReactionResult{}, err
	}

	updatedAgg, err := u.updateAggregate(ctx, tokenBlueprintID, agg)
	if err != nil {
		return TokenBlueprintReactionResult{}, err
	}

	return TokenBlueprintReactionResult{
		Aggregate: updatedAgg,
		Reaction:  savedReaction,
	}, nil
}

// ============================================================
// Comments list
// ============================================================

type ListCommentsInput struct {
	TokenBlueprintID string

	SearchQuery     string
	ParentCommentID *string
	RootCommentID   string
	AuthorID        string
	AuthorType      *tokenBlueprint_review.AuthorType
	IsOwnerComment  *bool
	Deleted         *bool
	Depth           *int

	Sort common.Sort
	Page common.Page
}

func (u *TokenBlueprintReviewUsecase) ListComments(
	ctx context.Context,
	in ListCommentsInput,
) (common.PageResult[CommentView], error) {
	if err := u.ensureConfigured(); err != nil {
		return common.PageResult[CommentView]{}, err
	}
	if in.TokenBlueprintID == "" {
		return common.PageResult[CommentView]{}, errTokenBlueprintIDRequired
	}

	filter := tokenBlueprint_review.FilterComment{
		FilterCommon: common.FilterCommon{
			SearchQuery: in.SearchQuery,
		},
		TokenBlueprintID: in.TokenBlueprintID,
		ParentCommentID:  in.ParentCommentID,
		RootCommentID:    in.RootCommentID,
		AuthorID:         in.AuthorID,
		AuthorType:       in.AuthorType,
		IsOwnerComment:   in.IsOwnerComment,
		Deleted:          in.Deleted,
		Depth:            in.Depth,
	}

	res, err := u.repos.Comments().List(ctx, filter, in.Sort, in.Page)
	if err != nil {
		return common.PageResult[CommentView]{}, err
	}

	return common.PageResult[CommentView]{
		Items:      u.BuildComments(ctx, res.Items),
		Page:       res.Page,
		PerPage:    res.PerPage,
		TotalCount: res.TotalCount,
		TotalPages: res.TotalPages,
	}, nil
}

// ============================================================
// Comments command
// ============================================================

type CreateCommentInput struct {
	CommentID        string
	TokenBlueprintID string
	ParentCommentID  string
	AuthorID         string
	AuthorType       tokenBlueprint_review.AuthorType
	IsOwnerComment   bool
	Body             string
}

func (u *TokenBlueprintReviewUsecase) CreateComment(
	ctx context.Context,
	in CreateCommentInput,
) (tokenBlueprint_review.Comment, error) {
	if err := u.ensureConfigured(); err != nil {
		return tokenBlueprint_review.Comment{}, err
	}

	now := u.now()

	commentID := in.CommentID
	if commentID == "" {
		commentID = newCommentID(now)
	}

	var comment *tokenBlueprint_review.Comment
	var err error

	if in.ParentCommentID == "" {
		comment, err = tokenBlueprint_review.NewTopLevelComment(
			commentID,
			in.TokenBlueprintID,
			in.AuthorID,
			in.AuthorType,
			in.IsOwnerComment,
			in.Body,
			now,
		)
		if err != nil {
			return tokenBlueprint_review.Comment{}, err
		}
	} else {
		parent, err := u.repos.Comments().GetByParentID(ctx, in.TokenBlueprintID, in.ParentCommentID)
		if err != nil {
			return tokenBlueprint_review.Comment{}, err
		}

		comment, err = tokenBlueprint_review.NewReplyComment(
			commentID,
			in.TokenBlueprintID,
			&parent,
			in.AuthorID,
			in.AuthorType,
			in.IsOwnerComment,
			in.Body,
			now,
		)
		if err != nil {
			return tokenBlueprint_review.Comment{}, err
		}
	}

	created, err := u.repos.Comments().CreateUnderParent(ctx, in.TokenBlueprintID, *comment)
	if err != nil {
		return tokenBlueprint_review.Comment{}, err
	}

	if err := u.incrementParentChildCount(ctx, in.TokenBlueprintID, in.ParentCommentID, now); err != nil {
		return tokenBlueprint_review.Comment{}, err
	}

	agg, err := u.ensureAggregate(ctx, in.TokenBlueprintID, now)
	if err != nil {
		return tokenBlueprint_review.Comment{}, err
	}

	agg.ApplyCommentCreated(created, now)

	if _, err := u.updateAggregate(ctx, in.TokenBlueprintID, agg); err != nil {
		return tokenBlueprint_review.Comment{}, err
	}

	return created, nil
}

func (u *TokenBlueprintReviewUsecase) DeleteComment(
	ctx context.Context,
	tokenBlueprintID string,
	commentID string,
) error {
	if err := u.ensureConfigured(); err != nil {
		return err
	}

	comment, err := u.repos.Comments().GetByParentID(ctx, tokenBlueprintID, commentID)
	if err != nil {
		return err
	}

	if err := u.repos.Comments().DeleteUnderParent(ctx, tokenBlueprintID, commentID); err != nil {
		return err
	}

	now := u.now()

	if err := u.decrementParentChildCount(ctx, tokenBlueprintID, comment.ParentCommentID, now); err != nil {
		return err
	}

	agg, err := u.repos.TokenBlueprintAggregates().GetByID(ctx, tokenBlueprintID)
	if err != nil {
		return nil
	}

	if err := agg.ApplyCommentDeleted(comment, now); err != nil {
		return err
	}

	_, err = u.updateAggregate(ctx, tokenBlueprintID, agg)
	return err
}

// ============================================================
// Comment reaction
// ============================================================

func (u *TokenBlueprintReviewUsecase) ReactToComment(
	ctx context.Context,
	tokenBlueprintID string,
	commentID string,
	actorID string,
	actorType tokenBlueprint_review.ActorType,
	newType tokenBlueprint_review.ReactionType,
) (tokenBlueprint_review.Comment, error) {
	if err := u.ensureConfigured(); err != nil {
		return tokenBlueprint_review.Comment{}, err
	}

	if err := actorType.Validate(); err != nil {
		return tokenBlueprint_review.Comment{}, err
	}

	pressedType := newType
	if err := pressedType.Validate(); err != nil {
		return tokenBlueprint_review.Comment{}, err
	}

	now := u.now()

	comment, err := u.repos.Comments().GetByParentID(ctx, tokenBlueprintID, commentID)
	if err != nil {
		return tokenBlueprint_review.Comment{}, err
	}

	oldType := tokenBlueprint_review.ReactionComment
	if ex, err := u.repos.CommentReactions().FindByActor(
		ctx,
		tokenBlueprintID,
		commentID,
		actorType,
		actorID,
	); err == nil {
		oldType = ex.Type
	}

	nextType, err := tokenBlueprint_review.NextReactionType(oldType, pressedType)
	if err != nil {
		return tokenBlueprint_review.Comment{}, err
	}

	if err := comment.ApplyReaction(oldType, nextType, now); err != nil {
		return tokenBlueprint_review.Comment{}, err
	}

	reaction, err := tokenBlueprint_review.NewCommentReaction(
		tokenBlueprintID,
		commentID,
		actorID,
		actorType,
		nextType,
		now,
	)
	if err != nil {
		return tokenBlueprint_review.Comment{}, err
	}

	if _, err := u.repos.CommentReactions().Upsert(ctx, *reaction); err != nil {
		return tokenBlueprint_review.Comment{}, err
	}

	updated, err := u.repos.Comments().UpdateUnderParent(
		ctx,
		tokenBlueprintID,
		commentID,
		tokenBlueprint_review.NewReactionCountPatchFromComment(comment),
	)
	if err != nil {
		return tokenBlueprint_review.Comment{}, err
	}

	return updated, nil
}
