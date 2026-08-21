// backend/internal/application/usecase/tokenBlueprint_review_usecase.go
package usecase

import (
	"context"
	"errors"
	"strconv"
	"time"

	applicationport "narratives/internal/application/port"
	avatar "narratives/internal/domain/avatar"
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
	brandRepo          applicationport.BrandGetter

	now func() time.Time
}

var (
	ErrCommentDeleteForbidden          = errors.New("tokenBlueprint_review_usecase: comment delete forbidden")
	errReviewReposNotConfigured        = errors.New("tokenBlueprint_review_usecase: repository port not configured")
	errUsecaseNotConfigured            = errors.New("tokenBlueprint_review_usecase: avatar repository not configured")
	errTokenBlueprintRepoNotConfigured = errors.New("tokenBlueprint_review_usecase: token blueprint repository not configured")
	errBrandRepositoryNotConfigured    = errors.New("tokenBlueprint_review_usecase: brand repository not configured")
	errTokenBlueprintIDRequired        = errors.New("tokenBlueprint_review_usecase: tokenBlueprintID is required")
	errCommentIDRequired               = errors.New("tokenBlueprint_review_usecase: commentID is required")
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
	brandRepo applicationport.BrandGetter,
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
		name, icon, err := u.GetNameAndIconByID(
			ctx,
			comment.AuthorID,
		)
		if err == nil {
			view.AuthorAvatarName = name
			view.AuthorAvatarIcon = strPtrOrNil(icon)
		}

	case tokenBlueprint_review.AuthorTypeBrand:
		name, icon, err := u.GetBrandNameAndIconByID(
			ctx,
			comment.AuthorID,
		)
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
	out := make(
		[]CommentView,
		0,
		len(comments),
	)

	for _, comment := range comments {
		out = append(
			out,
			u.BuildComment(
				ctx,
				comment,
			),
		)
	}

	return out
}

// ============================================================
// Internal helpers
// ============================================================

func newCommentID(now time.Time) string {
	return "cm_" + strconv.FormatInt(
		now.UnixNano(),
		10,
	)
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
) (
	tokenBlueprint_review.TokenBlueprintReviewAggregate,
	error,
) {
	if err := u.ensureConfigured(); err != nil {
		return tokenBlueprint_review.TokenBlueprintReviewAggregate{},
			err
	}

	aggregateRepository :=
		u.repos.TokenBlueprintAggregates()

	aggregate, err :=
		aggregateRepository.GetByID(
			ctx,
			tokenBlueprintID,
		)
	if err == nil {
		return aggregate, nil
	}

	created, createErr :=
		tokenBlueprint_review.NewTokenBlueprintReviewAggregate(
			tokenBlueprintID,
			now,
		)
	if createErr != nil {
		return tokenBlueprint_review.TokenBlueprintReviewAggregate{},
			createErr
	}

	aggregate, err =
		aggregateRepository.Create(
			ctx,
			*created,
		)
	if err != nil {
		return tokenBlueprint_review.TokenBlueprintReviewAggregate{},
			err
	}

	return aggregate, nil
}

func (u *TokenBlueprintReviewUsecase) updateAggregate(
	ctx context.Context,
	tokenBlueprintID string,
	aggregate tokenBlueprint_review.TokenBlueprintReviewAggregate,
) (
	tokenBlueprint_review.TokenBlueprintReviewAggregate,
	error,
) {
	if err := u.ensureConfigured(); err != nil {
		return tokenBlueprint_review.TokenBlueprintReviewAggregate{},
			err
	}

	return u.repos.TokenBlueprintAggregates().Update(
		ctx,
		tokenBlueprintID,
		tokenBlueprint_review.NewPatchFromTokenBlueprintReviewAggregate(
			aggregate,
		),
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

	parent, err :=
		u.repos.Comments().GetByParentID(
			ctx,
			tokenBlueprintID,
			parentCommentID,
		)
	if err != nil {
		return err
	}

	parent.IncrementChildCount(now)

	_, err =
		u.repos.Comments().UpdateUnderParent(
			ctx,
			tokenBlueprintID,
			parent.CommentID,
			tokenBlueprint_review.NewChildCountPatchFromComment(
				parent,
			),
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

	parent, err :=
		u.repos.Comments().GetByParentID(
			ctx,
			tokenBlueprintID,
			parentCommentID,
		)
	if err != nil {
		return err
	}

	if err := parent.DecrementChildCount(now); err != nil {
		return err
	}

	_, err =
		u.repos.Comments().UpdateUnderParent(
			ctx,
			tokenBlueprintID,
			parent.CommentID,
			tokenBlueprint_review.NewChildCountPatchFromComment(
				parent,
			),
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
		return tokenBlueprint.Patch{},
			errTokenBlueprintRepoNotConfigured
	}

	tokenBlueprintEntity, err :=
		u.tokenBlueprintRepo.GetByID(
			ctx,
			tokenBlueprintID,
		)
	if err != nil {
		return tokenBlueprint.Patch{}, err
	}

	if tokenBlueprintEntity == nil {
		return tokenBlueprint.Patch{},
			errors.New(
				"tokenBlueprint_review_usecase: token blueprint not found",
			)
	}

	patch := tokenBlueprint.Patch{
		ID:          tokenBlueprintEntity.ID,
		TokenName:   tokenBlueprintEntity.Name,
		Symbol:      tokenBlueprintEntity.Symbol,
		BrandID:     tokenBlueprintEntity.BrandID,
		CompanyID:   tokenBlueprintEntity.CompanyID,
		Description: tokenBlueprintEntity.Description,
		Minted:      tokenBlueprintEntity.Minted,
		MetadataURI: tokenBlueprintEntity.MetadataURI,
		IconURL:     tokenBlueprintEntity.IconURL,
	}

	if patch.BrandID != "" &&
		u.brandRepo != nil {
		brandEntity, brandErr :=
			u.brandRepo.GetByID(
				ctx,
				patch.BrandID,
			)

		if brandErr == nil &&
			brandEntity.Name != "" {
			patch.BrandName = brandEntity.Name
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
) (
	tokenBlueprint_review.TokenBlueprintReviewAggregate,
	error,
) {
	if err := u.ensureConfigured(); err != nil {
		return tokenBlueprint_review.TokenBlueprintReviewAggregate{},
			err
	}

	return u.repos.TokenBlueprintAggregates().GetByID(
		ctx,
		tokenBlueprintID,
	)
}

func (u *TokenBlueprintReviewUsecase) ListAggregatesByCompanyTokenBlueprints(
	ctx context.Context,
	companyID string,
) (
	[]tokenBlueprint_review.TokenBlueprintReviewAggregate,
	error,
) {
	if err := u.ensureConfigured(); err != nil {
		return nil, err
	}

	if u.tokenBlueprintRepo == nil {
		return nil,
			errTokenBlueprintRepoNotConfigured
	}

	tokenBlueprintIDs, err :=
		u.listAllTokenBlueprintIDsByCompany(
			ctx,
			companyID,
		)
	if err != nil {
		return nil, err
	}

	aggregateRepository :=
		u.repos.TokenBlueprintAggregates()

	items := make(
		[]tokenBlueprint_review.TokenBlueprintReviewAggregate,
		0,
		len(tokenBlueprintIDs),
	)

	for _, tokenBlueprintID := range tokenBlueprintIDs {
		aggregate, err :=
			aggregateRepository.GetByID(
				ctx,
				tokenBlueprintID,
			)
		if err != nil {
			continue
		}

		items = append(
			items,
			aggregate,
		)
	}

	return items, nil
}

func (u *TokenBlueprintReviewUsecase) listAllTokenBlueprintIDsByCompany(
	ctx context.Context,
	companyID string,
) ([]string, error) {
	if u == nil ||
		u.tokenBlueprintRepo == nil {
		return nil,
			errTokenBlueprintRepoNotConfigured
	}

	ids := make([]string, 0, 128)

	pageNumber := 1
	perPage := 200

	for {
		result, err :=
			u.tokenBlueprintRepo.ListByCompanyID(
				ctx,
				companyID,
				common.Page{
					Number:  pageNumber,
					PerPage: perPage,
				},
			)
		if err != nil {
			return nil, err
		}

		for _, tokenBlueprintEntity := range result.Items {
			if tokenBlueprintEntity.ID != "" {
				ids = append(
					ids,
					tokenBlueprintEntity.ID,
				)
			}
		}

		if result.TotalPages <= 0 ||
			pageNumber >= result.TotalPages ||
			len(result.Items) == 0 {
			break
		}

		pageNumber++
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
) (
	TokenBlueprintReactionResult,
	error,
) {
	if err := u.ensureConfigured(); err != nil {
		return TokenBlueprintReactionResult{},
			err
	}

	if err := actorType.Validate(); err != nil {
		return TokenBlueprintReactionResult{},
			err
	}

	pressedType := newType

	if err := pressedType.Validate(); err != nil {
		return TokenBlueprintReactionResult{},
			err
	}

	now := u.now()

	oldType :=
		tokenBlueprint_review.ReactionComment

	existingReaction, err :=
		u.repos.TokenBlueprintReactions().
			FindByActor(
				ctx,
				tokenBlueprintID,
				actorType,
				actorID,
			)
	if err == nil {
		oldType = existingReaction.Type
	}

	nextType, err :=
		tokenBlueprint_review.NextReactionType(
			oldType,
			pressedType,
		)
	if err != nil {
		return TokenBlueprintReactionResult{},
			err
	}

	aggregate, err := u.ensureAggregate(
		ctx,
		tokenBlueprintID,
		now,
	)
	if err != nil {
		return TokenBlueprintReactionResult{},
			err
	}

	if err := aggregate.ApplyReaction(
		oldType,
		nextType,
		now,
	); err != nil {
		return TokenBlueprintReactionResult{},
			err
	}

	reaction, err :=
		tokenBlueprint_review.NewTokenBlueprintReaction(
			tokenBlueprintID,
			actorID,
			actorType,
			nextType,
			now,
		)
	if err != nil {
		return TokenBlueprintReactionResult{},
			err
	}

	savedReaction, err :=
		u.repos.TokenBlueprintReactions().Upsert(
			ctx,
			*reaction,
		)
	if err != nil {
		return TokenBlueprintReactionResult{},
			err
	}

	updatedAggregate, err :=
		u.updateAggregate(
			ctx,
			tokenBlueprintID,
			aggregate,
		)
	if err != nil {
		return TokenBlueprintReactionResult{},
			err
	}

	return TokenBlueprintReactionResult{
		Aggregate: updatedAggregate,
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
	input ListCommentsInput,
) (
	common.PageResult[CommentView],
	error,
) {
	if err := u.ensureConfigured(); err != nil {
		return common.PageResult[CommentView]{},
			err
	}

	if input.TokenBlueprintID == "" {
		return common.PageResult[CommentView]{},
			errTokenBlueprintIDRequired
	}

	filter :=
		tokenBlueprint_review.FilterComment{
			FilterCommon: common.FilterCommon{
				SearchQuery: input.SearchQuery,
			},
			TokenBlueprintID: input.TokenBlueprintID,
			ParentCommentID:  input.ParentCommentID,
			RootCommentID:    input.RootCommentID,
			AuthorID:         input.AuthorID,
			AuthorType:       input.AuthorType,
			IsOwnerComment:   input.IsOwnerComment,
			Deleted:          input.Deleted,
			Depth:            input.Depth,
		}

	result, err :=
		u.repos.Comments().List(
			ctx,
			filter,
			input.Sort,
			input.Page,
		)
	if err != nil {
		return common.PageResult[CommentView]{},
			err
	}

	return common.PageResult[CommentView]{
		Items:      u.BuildComments(ctx, result.Items),
		Page:       result.Page,
		PerPage:    result.PerPage,
		TotalCount: result.TotalCount,
		TotalPages: result.TotalPages,
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
	input CreateCommentInput,
) (
	tokenBlueprint_review.Comment,
	error,
) {
	if err := u.ensureConfigured(); err != nil {
		return tokenBlueprint_review.Comment{},
			err
	}

	now := u.now()

	commentID := input.CommentID

	if commentID == "" {
		commentID = newCommentID(now)
	}

	var comment *tokenBlueprint_review.Comment
	var err error

	if input.ParentCommentID == "" {
		comment, err =
			tokenBlueprint_review.NewTopLevelComment(
				commentID,
				input.TokenBlueprintID,
				input.AuthorID,
				input.AuthorType,
				input.IsOwnerComment,
				input.Body,
				now,
			)
		if err != nil {
			return tokenBlueprint_review.Comment{},
				err
		}
	} else {
		parent, err :=
			u.repos.Comments().GetByParentID(
				ctx,
				input.TokenBlueprintID,
				input.ParentCommentID,
			)
		if err != nil {
			return tokenBlueprint_review.Comment{},
				err
		}

		comment, err =
			tokenBlueprint_review.NewReplyComment(
				commentID,
				input.TokenBlueprintID,
				&parent,
				input.AuthorID,
				input.AuthorType,
				input.IsOwnerComment,
				input.Body,
				now,
			)
		if err != nil {
			return tokenBlueprint_review.Comment{},
				err
		}
	}

	created, err :=
		u.repos.Comments().CreateUnderParent(
			ctx,
			input.TokenBlueprintID,
			*comment,
		)
	if err != nil {
		return tokenBlueprint_review.Comment{},
			err
	}

	if err := u.incrementParentChildCount(
		ctx,
		input.TokenBlueprintID,
		input.ParentCommentID,
		now,
	); err != nil {
		return tokenBlueprint_review.Comment{},
			err
	}

	aggregate, err := u.ensureAggregate(
		ctx,
		input.TokenBlueprintID,
		now,
	)
	if err != nil {
		return tokenBlueprint_review.Comment{},
			err
	}

	aggregate.ApplyCommentCreated(
		created,
		now,
	)

	if _, err := u.updateAggregate(
		ctx,
		input.TokenBlueprintID,
		aggregate,
	); err != nil {
		return tokenBlueprint_review.Comment{},
			err
	}

	return created, nil
}

type DeleteCommentInput struct {
	TokenBlueprintID string
	CommentID        string
	AuthorID         string
	AuthorType       tokenBlueprint_review.AuthorType
}

func (u *TokenBlueprintReviewUsecase) DeleteComment(
	ctx context.Context,
	input DeleteCommentInput,
) error {
	if err := u.ensureConfigured(); err != nil {
		return err
	}

	if input.TokenBlueprintID == "" {
		return errTokenBlueprintIDRequired
	}

	if input.CommentID == "" {
		return errCommentIDRequired
	}

	if input.AuthorID == "" {
		return ErrCommentDeleteForbidden
	}

	if err := input.AuthorType.Validate(); err != nil {
		return err
	}

	comment, err :=
		u.repos.Comments().GetByParentID(
			ctx,
			input.TokenBlueprintID,
			input.CommentID,
		)
	if err != nil {
		return err
	}

	if comment.AuthorID != input.AuthorID ||
		comment.AuthorType != input.AuthorType {
		return ErrCommentDeleteForbidden
	}

	if err :=
		u.repos.Comments().DeleteUnderParent(
			ctx,
			input.TokenBlueprintID,
			input.CommentID,
		); err != nil {
		return err
	}

	now := u.now()

	if err := u.decrementParentChildCount(
		ctx,
		input.TokenBlueprintID,
		comment.ParentCommentID,
		now,
	); err != nil {
		return err
	}

	aggregate, err :=
		u.repos.TokenBlueprintAggregates().GetByID(
			ctx,
			input.TokenBlueprintID,
		)
	if err != nil {
		return nil
	}

	if err := aggregate.ApplyCommentDeleted(
		comment,
		now,
	); err != nil {
		return err
	}

	_, err = u.updateAggregate(
		ctx,
		input.TokenBlueprintID,
		aggregate,
	)

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
) (
	tokenBlueprint_review.Comment,
	error,
) {
	if err := u.ensureConfigured(); err != nil {
		return tokenBlueprint_review.Comment{},
			err
	}

	if err := actorType.Validate(); err != nil {
		return tokenBlueprint_review.Comment{},
			err
	}

	pressedType := newType

	if err := pressedType.Validate(); err != nil {
		return tokenBlueprint_review.Comment{},
			err
	}

	now := u.now()

	comment, err :=
		u.repos.Comments().GetByParentID(
			ctx,
			tokenBlueprintID,
			commentID,
		)
	if err != nil {
		return tokenBlueprint_review.Comment{},
			err
	}

	oldType :=
		tokenBlueprint_review.ReactionComment

	existingReaction, err :=
		u.repos.CommentReactions().FindByActor(
			ctx,
			tokenBlueprintID,
			commentID,
			actorType,
			actorID,
		)
	if err == nil {
		oldType = existingReaction.Type
	}

	nextType, err :=
		tokenBlueprint_review.NextReactionType(
			oldType,
			pressedType,
		)
	if err != nil {
		return tokenBlueprint_review.Comment{},
			err
	}

	if err := comment.ApplyReaction(
		oldType,
		nextType,
		now,
	); err != nil {
		return tokenBlueprint_review.Comment{},
			err
	}

	reaction, err :=
		tokenBlueprint_review.NewCommentReaction(
			tokenBlueprintID,
			commentID,
			actorID,
			actorType,
			nextType,
			now,
		)
	if err != nil {
		return tokenBlueprint_review.Comment{},
			err
	}

	if _, err :=
		u.repos.CommentReactions().Upsert(
			ctx,
			*reaction,
		); err != nil {
		return tokenBlueprint_review.Comment{},
			err
	}

	updated, err :=
		u.repos.Comments().UpdateUnderParent(
			ctx,
			tokenBlueprintID,
			commentID,
			tokenBlueprint_review.NewReactionCountPatchFromComment(
				comment,
			),
		)
	if err != nil {
		return tokenBlueprint_review.Comment{},
			err
	}

	return updated, nil
}
