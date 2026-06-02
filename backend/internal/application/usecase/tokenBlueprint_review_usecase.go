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
type TokenBlueprintReviewUsecase struct {
	repos              tokenBlueprint_review.RepositoryPort
	avatarRepos        avatar.Repository
	tokenBlueprintRepo tokenBlueprint.RepositoryPort
	brandRepo          brand.Repository

	now func() time.Time
}

var (
	errUsecaseNotConfigured            = errors.New("tokenBlueprint_review_usecase: avatar repository not configured")
	errTokenBlueprintRepoNotConfigured = errors.New("tokenBlueprint_review_usecase: token blueprint repository not configured")
	errBrandRepositoryNotConfigured    = errors.New("tokenBlueprint_review_usecase: brand repository not configured")

	ErrTokenBlueprintReactionsListNotImplemented = errors.New("tokenBlueprint_review_usecase: token blueprint reactions list not implemented")
)

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
// Repository exposure for transitional adapters
// ============================================================

func (u *TokenBlueprintReviewUsecase) TokenBlueprintReactionRepository() any {
	if u == nil || u.repos == nil {
		return nil
	}
	return u.repos.TokenBlueprintReactions()
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
// View DTOs for console API
// ============================================================

type CommentView struct {
	tokenBlueprint_review.Comment

	AuthorAvatarName string  `json:"AuthorAvatarName,omitempty"`
	AuthorAvatarIcon *string `json:"AuthorAvatarIcon,omitempty"`

	BrandName string  `json:"BrandName,omitempty"`
	BrandIcon *string `json:"BrandIcon,omitempty"`
}

type avatarLite struct {
	name string
	icon string
}

type brandLite struct {
	name string
	icon string
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (u *TokenBlueprintReviewUsecase) resolveAvatarLite(
	ctx context.Context,
	cache map[string]avatarLite,
	avatarID string,
) (name string, icon string) {
	if avatarID == "" {
		return "", ""
	}

	if v, ok := cache[avatarID]; ok {
		return v.name, v.icon
	}

	n, ic, err := u.GetNameAndIconByID(ctx, avatarID)
	if err != nil {
		cache[avatarID] = avatarLite{name: "", icon: ""}
		return "", ""
	}

	cache[avatarID] = avatarLite{name: n, icon: ic}
	return n, ic
}

func (u *TokenBlueprintReviewUsecase) resolveBrandLite(
	ctx context.Context,
	cache map[string]brandLite,
	brandID string,
) (name string, icon string) {
	if brandID == "" {
		return "", ""
	}

	if v, ok := cache[brandID]; ok {
		return v.name, v.icon
	}

	n, ic, err := u.GetBrandNameAndIconByID(ctx, brandID)
	if err != nil {
		cache[brandID] = brandLite{name: "", icon: ""}
		return "", ""
	}

	cache[brandID] = brandLite{name: n, icon: ic}
	return n, ic
}

// ============================================================
// Internal helpers
// ============================================================

func newCommentID(now time.Time) string {
	return "cm_" + strconv.FormatInt(now.UnixNano(), 10)
}

func aggregatePatch(
	agg tokenBlueprint_review.TokenBlueprintReviewAggregate,
) tokenBlueprint_review.PatchTokenBlueprintReviewAggregate {
	return tokenBlueprint_review.PatchTokenBlueprintReviewAggregate{
		LikeCount:            &agg.LikeCount,
		DislikeCount:         &agg.DislikeCount,
		TopLevelCommentCount: &agg.TopLevelCommentCount,
		TotalCommentCount:    &agg.TotalCommentCount,
		PinnedCommentID:      &agg.PinnedCommentID,
	}
}

func commentChildCountPatch(
	comment tokenBlueprint_review.Comment,
) tokenBlueprint_review.PatchComment {
	return tokenBlueprint_review.PatchComment{
		ChildCount: &comment.ChildCount,
	}
}

func commentReactionCountPatch(
	comment tokenBlueprint_review.Comment,
) tokenBlueprint_review.PatchComment {
	return tokenBlueprint_review.PatchComment{
		LikeCount:    &comment.LikeCount,
		DislikeCount: &comment.DislikeCount,
		ChildCount:   &comment.ChildCount,
	}
}

func (u *TokenBlueprintReviewUsecase) ensureAggregate(
	ctx context.Context,
	tokenBlueprintID string,
	now time.Time,
) (tokenBlueprint_review.TokenBlueprintReviewAggregate, error) {
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
	return u.repos.TokenBlueprintAggregates().Update(ctx, tokenBlueprintID, aggregatePatch(agg))
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

	parent, err := u.repos.Comments().GetByParentID(ctx, tokenBlueprintID, parentCommentID)
	if err != nil {
		return err
	}

	parent.IncrementChildCount(now)

	_, err = u.repos.Comments().UpdateUnderParent(
		ctx,
		tokenBlueprintID,
		parent.CommentID,
		commentChildCountPatch(parent),
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
		commentChildCountPatch(parent),
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

func (u *TokenBlueprintReviewUsecase) ListAggregates(
	ctx context.Context,
	filter tokenBlueprint_review.FilterTokenBlueprintReviewAggregate,
	sort common.Sort,
	page common.Page,
) (common.PageResult[tokenBlueprint_review.TokenBlueprintReviewAggregate], error) {
	return u.repos.TokenBlueprintAggregates().List(ctx, filter, sort, page)
}

func (u *TokenBlueprintReviewUsecase) GetAggregate(
	ctx context.Context,
	tokenBlueprintID string,
) (tokenBlueprint_review.TokenBlueprintReviewAggregate, error) {
	return u.repos.TokenBlueprintAggregates().GetByID(ctx, tokenBlueprintID)
}

func (u *TokenBlueprintReviewUsecase) ListAggregatesByCompanyTokenBlueprints(
	ctx context.Context,
	companyID string,
) ([]tokenBlueprint_review.TokenBlueprintReviewAggregate, error) {
	if u == nil || u.tokenBlueprintRepo == nil {
		return nil, errors.New("tokenBlueprint_review_usecase: tokenBlueprint repo is nil")
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
// TokenBlueprint reactions
// ============================================================

type TokenBlueprintReactionResult struct {
	Aggregate tokenBlueprint_review.TokenBlueprintReviewAggregate
	Reaction  tokenBlueprint_review.TokenBlueprintReaction
}

func (u *TokenBlueprintReviewUsecase) ListTokenBlueprintReactions(
	ctx context.Context,
	tokenBlueprintID string,
) ([]tokenBlueprint_review.TokenBlueprintReaction, error) {
	type lister interface {
		ListByTokenBlueprintID(ctx context.Context, tokenBlueprintID string) ([]tokenBlueprint_review.TokenBlueprintReaction, error)
	}

	reactionRepo := u.repos.TokenBlueprintReactions()
	ls, ok := any(reactionRepo).(lister)
	if !ok {
		return nil, ErrTokenBlueprintReactionsListNotImplemented
	}

	return ls.ListByTokenBlueprintID(ctx, tokenBlueprintID)
}

func (u *TokenBlueprintReviewUsecase) ReactToTokenBlueprintDetailed(
	ctx context.Context,
	tokenBlueprintID string,
	actorID string,
	actorType tokenBlueprint_review.ActorType,
	newType tokenBlueprint_review.ReactionType,
) (TokenBlueprintReactionResult, error) {
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

func (u *TokenBlueprintReviewUsecase) ReactToTokenBlueprint(
	ctx context.Context,
	tokenBlueprintID string,
	actorID string,
	actorType tokenBlueprint_review.ActorType,
	newType tokenBlueprint_review.ReactionType,
) (tokenBlueprint_review.TokenBlueprintReviewAggregate, error) {
	result, err := u.ReactToTokenBlueprintDetailed(
		ctx,
		tokenBlueprintID,
		actorID,
		actorType,
		newType,
	)
	if err != nil {
		return tokenBlueprint_review.TokenBlueprintReviewAggregate{}, err
	}

	return result.Aggregate, nil
}

// ============================================================
// Comments list
// ============================================================

type ListCommentsInput struct {
	TokenBlueprintID string
	Filter           tokenBlueprint_review.FilterComment
	Sort             common.Sort
	Page             common.Page
}

func (u *TokenBlueprintReviewUsecase) ListComments(
	ctx context.Context,
	in ListCommentsInput,
) (common.PageResult[tokenBlueprint_review.Comment], error) {
	f := in.Filter
	f.TokenBlueprintID = in.TokenBlueprintID

	return u.repos.Comments().List(ctx, f, in.Sort, in.Page)
}

func (u *TokenBlueprintReviewUsecase) ListAllCommentsByTokenBlueprintID(
	ctx context.Context,
	tokenBlueprintID string,
	sort common.Sort,
	page common.Page,
) ([]tokenBlueprint_review.Comment, error) {
	items := make([]tokenBlueprint_review.Comment, 0, 128)

	pageNo := page.Number
	if pageNo <= 0 {
		pageNo = 1
	}

	perPage := page.PerPage
	if perPage <= 0 {
		perPage = 200
	}

	for {
		res, err := u.ListComments(ctx, ListCommentsInput{
			TokenBlueprintID: tokenBlueprintID,
			Filter:           tokenBlueprint_review.FilterComment{},
			Sort:             sort,
			Page: common.Page{
				Number:  pageNo,
				PerPage: perPage,
			},
		})
		if err != nil {
			return nil, err
		}

		items = append(items, res.Items...)

		if res.TotalPages <= 0 || pageNo >= res.TotalPages || len(res.Items) == 0 {
			break
		}

		pageNo++
	}

	return items, nil
}

func (u *TokenBlueprintReviewUsecase) ListTopLevelComments(
	ctx context.Context,
	tokenBlueprintID string,
	sort common.Sort,
	page common.Page,
) (common.PageResult[tokenBlueprint_review.Comment], error) {
	topLevel := ""

	return u.ListComments(ctx, ListCommentsInput{
		TokenBlueprintID: tokenBlueprintID,
		Filter: tokenBlueprint_review.FilterComment{
			ParentCommentID: &topLevel,
		},
		Sort: sort,
		Page: page,
	})
}

func (u *TokenBlueprintReviewUsecase) ListChildComments(
	ctx context.Context,
	tokenBlueprintID string,
	parentCommentID string,
	sort common.Sort,
	page common.Page,
) (common.PageResult[tokenBlueprint_review.Comment], error) {
	return u.ListComments(ctx, ListCommentsInput{
		TokenBlueprintID: tokenBlueprintID,
		Filter: tokenBlueprint_review.FilterComment{
			ParentCommentID: &parentCommentID,
		},
		Sort: sort,
		Page: page,
	})
}

func (u *TokenBlueprintReviewUsecase) ListThreadComments(
	ctx context.Context,
	tokenBlueprintID string,
	rootCommentID string,
	sort common.Sort,
	page common.Page,
) (common.PageResult[tokenBlueprint_review.Comment], error) {
	return u.ListComments(ctx, ListCommentsInput{
		TokenBlueprintID: tokenBlueprintID,
		Filter: tokenBlueprint_review.FilterComment{
			RootCommentID: rootCommentID,
		},
		Sort: sort,
		Page: page,
	})
}

func (u *TokenBlueprintReviewUsecase) ListAllCommentsWithAuthorByTokenBlueprintID(
	ctx context.Context,
	tokenBlueprintID string,
	sort common.Sort,
	page common.Page,
) ([]CommentView, error) {
	comments, err := u.ListAllCommentsByTokenBlueprintID(ctx, tokenBlueprintID, sort, page)
	if err != nil {
		return nil, err
	}

	avatarCache := make(map[string]avatarLite, 64)
	brandCache := make(map[string]brandLite, 32)
	out := make([]CommentView, 0, len(comments))

	for _, c := range comments {
		avatarName := ""
		avatarIcon := ""
		brandName := ""
		brandIcon := ""

		switch c.AuthorType {
		case tokenBlueprint_review.AuthorTypeAvatar:
			avatarName, avatarIcon = u.resolveAvatarLite(ctx, avatarCache, c.AuthorID)
		case tokenBlueprint_review.AuthorTypeBrand:
			brandName, brandIcon = u.resolveBrandLite(ctx, brandCache, c.AuthorID)
		}

		out = append(out, CommentView{
			Comment:          c,
			AuthorAvatarName: avatarName,
			AuthorAvatarIcon: strPtrOrNil(avatarIcon),
			BrandName:        brandName,
			BrandIcon:        strPtrOrNil(brandIcon),
		})
	}

	return out, nil
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

	if in.ParentCommentID == "" {
		agg.IncrementTopLevelCommentCount(now)
	}
	agg.IncrementTotalCommentCount(now)

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

	if comment.ParentCommentID == "" {
		_ = agg.DecrementTopLevelCommentCount(now)
	}
	_ = agg.DecrementTotalCommentCount(now)

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
		commentReactionCountPatch(comment),
	)
	if err != nil {
		return tokenBlueprint_review.Comment{}, err
	}

	return updated, nil
}
