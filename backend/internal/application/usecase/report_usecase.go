// backend/internal/application/usecase/report_usecase.go
package usecase

import (
	"context"
	"errors"
	"time"

	applicationport "narratives/internal/application/port"
	avatar "narratives/internal/domain/avatar"
	common "narratives/internal/domain/common"
	inventory "narratives/internal/domain/inventory"
	listdom "narratives/internal/domain/list"
	pbr "narratives/internal/domain/productBlueprintReview"
	reportdom "narratives/internal/domain/report"
	tokenblueprint "narratives/internal/domain/tokenBlueprint"
	tokenreview "narratives/internal/domain/tokenBlueprint_review"
)

// ============================================================
// Errors
// ============================================================

var (
	ErrReportUsecaseNotConfigured = errors.New("report_usecase: not configured")
	ErrReportForbidden            = errors.New("report_usecase: forbidden")
	ErrReportSelfReport           = errors.New("report_usecase: self report is not allowed")
	ErrReportInvalidDecision      = errors.New("report_usecase: invalid decision")
)

// ============================================================
// Ports
// ============================================================

// ReportProductReviewModerator owns Admin moderation of product reviews.
type ReportProductReviewModerator interface {
	RemoveProductBlueprintReviewByAdmin(
		ctx context.Context,
		in RemoveProductBlueprintReviewByAdminInput,
	) (pbr.Review, error)
}

// ReportListInventoryReader owns the minimum inventory read required to resolve
// the ProductBlueprint, Brand, and Company behind a List.
type ReportListInventoryReader interface {
	GetByID(ctx context.Context, id string) (inventory.Mint, error)
}

// ReportListModerator owns Admin moderation of Lists.
// LIST + REMOVE in Report means suspending the List from Mall publication.
// It must not physically delete the List or its images.
type ReportListModerator interface {
	SuspendListByAdmin(
		ctx context.Context,
		input SuspendListByAdminInput,
	) error
}

// ReportTokenBlueprintModerator owns Admin moderation of token blueprints.
// TOKEN_BLUEPRINT + REMOVE in Report means hiding the token blueprint from AMOL UI.
// It must not delete on-chain metadata or physically delete the token blueprint.
type ReportTokenBlueprintModerator interface {
	HideTokenBlueprintByAdmin(
		ctx context.Context,
		input HideTokenBlueprintByAdminInput,
	) error
}

type HideTokenBlueprintByAdminInput struct {
	TokenBlueprintID string
	Reason           string
	AdminID          string
}

// ReportTokenCommentModerator owns Admin moderation of token comments.
type ReportTokenCommentModerator interface {
	RemoveCommentByAdmin(
		ctx context.Context,
		input RemoveCommentByAdminInput,
	) error
}

// ReportAvatarResaleModerator owns Admin moderation of avatar resale access.
// AVATAR + REMOVE in Report means resale-service suspension only.
// It must not delete or disable the avatar itself.
type ReportAvatarResaleModerator interface {
	SuspendAvatarResaleByAdmin(
		ctx context.Context,
		input SuspendAvatarResaleByAdminInput,
	) error
}

type SuspendAvatarResaleByAdminInput struct {
	AvatarID string
	Reason   string
	AdminID  string
}

// ============================================================
// Usecase
// ============================================================

type ReportUsecase struct {
	reportRepo               reportdom.RepositoryPort
	decisionNotificationRepo reportdom.DecisionNotificationRepository

	productReviewRepo       pbr.Repository
	productBlueprintRepo    applicationport.ProductBlueprintGetter
	productPurchaseResolver applicationport.OwnedProductResolver
	productReviewModerator  ReportProductReviewModerator

	listRepo      listdom.Repository
	inventoryRepo ReportListInventoryReader
	listModerator ReportListModerator

	tokenCommentRepo        tokenreview.CommentRepository
	tokenBlueprintRepo      tokenblueprint.RepositoryPort
	tokenAccessResolver     applicationport.ReportTokenAccessResolver
	tokenBlueprintModerator ReportTokenBlueprintModerator
	tokenCommentModerator   ReportTokenCommentModerator

	avatarRepo            avatar.Repository
	avatarResaleModerator ReportAvatarResaleModerator

	now func() time.Time
}

type ReportUsecaseDeps struct {
	ReportRepo               reportdom.RepositoryPort
	DecisionNotificationRepo reportdom.DecisionNotificationRepository

	ProductReviewRepo       pbr.Repository
	ProductBlueprintRepo    applicationport.ProductBlueprintGetter
	ProductPurchaseResolver applicationport.OwnedProductResolver
	ProductReviewModerator  ReportProductReviewModerator

	ListRepo      listdom.Repository
	InventoryRepo ReportListInventoryReader
	ListModerator ReportListModerator

	TokenCommentRepo        tokenreview.CommentRepository
	TokenBlueprintRepo      tokenblueprint.RepositoryPort
	TokenAccessResolver     applicationport.ReportTokenAccessResolver
	TokenBlueprintModerator ReportTokenBlueprintModerator
	TokenCommentModerator   ReportTokenCommentModerator

	AvatarRepo            avatar.Repository
	AvatarResaleModerator ReportAvatarResaleModerator

	Now func() time.Time
}

func NewReportUsecase(deps ReportUsecaseDeps) *ReportUsecase {
	now := deps.Now
	if now == nil {
		now = time.Now
	}

	return &ReportUsecase{
		reportRepo:               deps.ReportRepo,
		decisionNotificationRepo: deps.DecisionNotificationRepo,
		productReviewRepo:        deps.ProductReviewRepo,
		productBlueprintRepo:     deps.ProductBlueprintRepo,
		productPurchaseResolver:  deps.ProductPurchaseResolver,
		productReviewModerator:   deps.ProductReviewModerator,
		listRepo:                 deps.ListRepo,
		inventoryRepo:            deps.InventoryRepo,
		listModerator:            deps.ListModerator,
		tokenCommentRepo:         deps.TokenCommentRepo,
		tokenBlueprintRepo:       deps.TokenBlueprintRepo,
		tokenAccessResolver:      deps.TokenAccessResolver,
		tokenBlueprintModerator:  deps.TokenBlueprintModerator,
		tokenCommentModerator:    deps.TokenCommentModerator,
		avatarRepo:               deps.AvatarRepo,
		avatarResaleModerator:    deps.AvatarResaleModerator,
		now:                      now,
	}
}

func (u *ReportUsecase) ensureReportRepository() error {
	if u == nil || u.reportRepo == nil {
		return ErrReportUsecaseNotConfigured
	}
	return nil
}

func (u *ReportUsecase) ensureDecisionNotificationRepository() error {
	if u == nil || u.decisionNotificationRepo == nil {
		return ErrReportUsecaseNotConfigured
	}
	return nil
}

// ============================================================
// ProductBlueprint review report
// ============================================================

type ReportProductBlueprintReviewByAvatarInput struct {
	ProductBlueprintID string
	ReviewID           string
	AvatarID           string
	Reason             reportdom.ReportReason
	Detail             string
}

func (u *ReportUsecase) ReportProductBlueprintReviewByAvatar(
	ctx context.Context,
	input ReportProductBlueprintReviewByAvatarInput,
) (reportdom.AddReportResult, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reportdom.AddReportResult{}, err
	}
	if u.productReviewRepo == nil || u.productPurchaseResolver == nil {
		return reportdom.AddReportResult{}, ErrReportUsecaseNotConfigured
	}

	review, err := u.productReviewRepo.GetByProductBlueprintID(
		ctx,
		input.ProductBlueprintID,
		input.ReviewID,
	)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if review.ProductBlueprintID != input.ProductBlueprintID {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetParentID
	}
	if review.Status == pbr.ReviewStatusRemoved {
		return reportdom.AddReportResult{}, reportdom.ErrCannotReportRemovedTarget
	}
	if review.AvatarID == input.AvatarID {
		return reportdom.AddReportResult{}, ErrReportSelfReport
	}

	allowed, err := u.productPurchaseResolver.HasOwnedProductBlueprint(
		ctx,
		input.AvatarID,
		input.ProductBlueprintID,
	)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if !allowed {
		return reportdom.AddReportResult{}, ErrReportForbidden
	}

	return u.addProductBlueprintReviewReport(
		ctx,
		review,
		reportdom.ActorTypeAvatar,
		input.AvatarID,
		"",
		input.Reason,
		input.Detail,
	)
}

type ReportProductBlueprintReviewByBrandInput struct {
	ProductBlueprintID string
	ReviewID           string
	BrandID            string
	CompanyID          string
	Reason             reportdom.ReportReason
	Detail             string
}

func (u *ReportUsecase) ReportProductBlueprintReviewByBrand(
	ctx context.Context,
	input ReportProductBlueprintReviewByBrandInput,
) (reportdom.AddReportResult, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reportdom.AddReportResult{}, err
	}
	if u.productReviewRepo == nil || u.productBlueprintRepo == nil {
		return reportdom.AddReportResult{}, ErrReportUsecaseNotConfigured
	}

	productBlueprint, err := u.productBlueprintRepo.GetByID(ctx, input.ProductBlueprintID)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if productBlueprint.ID != input.ProductBlueprintID ||
		productBlueprint.CompanyID != input.CompanyID ||
		productBlueprint.BrandID != input.BrandID {
		return reportdom.AddReportResult{}, ErrReportForbidden
	}

	review, err := u.productReviewRepo.GetByProductBlueprintID(
		ctx,
		input.ProductBlueprintID,
		input.ReviewID,
	)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if review.ProductBlueprintID != input.ProductBlueprintID {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetParentID
	}
	if review.Status == pbr.ReviewStatusRemoved {
		return reportdom.AddReportResult{}, reportdom.ErrCannotReportRemovedTarget
	}

	return u.addProductBlueprintReviewReport(
		ctx,
		review,
		reportdom.ActorTypeBrand,
		input.BrandID,
		input.CompanyID,
		input.Reason,
		input.Detail,
	)
}

func (u *ReportUsecase) addProductBlueprintReviewReport(
	ctx context.Context,
	review pbr.Review,
	reporterType reportdom.ActorType,
	reporterID string,
	companyID string,
	reason reportdom.ReportReason,
	detail string,
) (reportdom.AddReportResult, error) {
	now := u.now().UTC()
	rating := int(review.Rating)

	reportCase, err := reportdom.NewReportCase(
		reportNewProductReviewCaseParams(
			review,
			rating,
			now,
		),
	)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	report, err := reportdom.NewReport(reportdom.NewReportParams{
		CaseID:       reportCase.ID,
		ReporterType: reporterType,
		ReporterID:   reporterID,
		CompanyID:    companyID,
		Reason:       reason,
		Detail:       detail,
		CreatedAt:    now,
	})
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	return u.reportRepo.AddReport(ctx, reportCase, report)
}

func reportNewProductReviewCaseParams(
	review pbr.Review,
	rating int,
	now time.Time,
) reportdom.NewReportCaseParams {
	return reportdom.NewReportCaseParams{
		TargetType:       reportdom.TargetTypeProductBlueprintReview,
		TargetID:         string(review.ID),
		TargetParentID:   review.ProductBlueprintID,
		TargetAuthorID:   review.AvatarID,
		TargetAuthorType: reportdom.ActorTypeAvatar,
		SnapshotTitle:    review.Title,
		SnapshotBody:     review.Body,
		SnapshotRating:   &rating,
		CreatedAt:        now,
	}
}

// ============================================================
// List report
// ============================================================

type ReportListByAvatarInput struct {
	ListID   string
	AvatarID string
	Reason   reportdom.ReportReason
	Detail   string
}

type reportListTargetContext struct {
	List      listdom.List
	BrandID   string
	CompanyID string
}

func (u *ReportUsecase) ReportListByAvatar(
	ctx context.Context,
	input ReportListByAvatarInput,
) (reportdom.AddReportResult, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reportdom.AddReportResult{}, err
	}
	if u.listRepo == nil || u.inventoryRepo == nil || u.productBlueprintRepo == nil {
		return reportdom.AddReportResult{}, ErrReportUsecaseNotConfigured
	}
	if input.ListID == "" {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetID
	}
	if input.AvatarID == "" {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidReporterID
	}

	targetContext, err := u.resolveListTargetContext(ctx, input.ListID)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if targetContext.List.Status != listdom.StatusListing {
		return reportdom.AddReportResult{}, reportdom.ErrCannotReportRemovedTarget
	}

	return u.addListReport(
		ctx,
		targetContext,
		input.AvatarID,
		input.Reason,
		input.Detail,
	)
}

func (u *ReportUsecase) addListReport(
	ctx context.Context,
	targetContext reportListTargetContext,
	reporterAvatarID string,
	reason reportdom.ReportReason,
	detail string,
) (reportdom.AddReportResult, error) {
	now := u.now().UTC()
	target := targetContext.List

	reportCase, err := reportdom.NewReportCase(reportdom.NewReportCaseParams{
		TargetType:       reportdom.TargetTypeList,
		TargetID:         target.ID,
		TargetParentID:   target.ID,
		TargetAuthorID:   targetContext.BrandID,
		TargetAuthorType: reportdom.ActorTypeBrand,
		SnapshotTitle:    target.Title,
		SnapshotBody:     target.Description,
		SnapshotRating:   nil,
		CreatedAt:        now,
	})
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	report, err := reportdom.NewReport(reportdom.NewReportParams{
		CaseID:       reportCase.ID,
		ReporterType: reportdom.ActorTypeAvatar,
		ReporterID:   reporterAvatarID,
		CompanyID:    "",
		Reason:       reason,
		Detail:       detail,
		CreatedAt:    now,
	})
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	return u.reportRepo.AddReport(ctx, reportCase, report)
}

func (u *ReportUsecase) resolveListTargetContext(
	ctx context.Context,
	listID string,
) (reportListTargetContext, error) {
	if u == nil || u.listRepo == nil || u.inventoryRepo == nil || u.productBlueprintRepo == nil {
		return reportListTargetContext{}, ErrReportUsecaseNotConfigured
	}
	if listID == "" {
		return reportListTargetContext{}, reportdom.ErrInvalidTargetID
	}

	target, err := u.listRepo.GetByID(ctx, listID)
	if err != nil {
		return reportListTargetContext{}, err
	}
	if target.ID != listID {
		return reportListTargetContext{}, reportdom.ErrInvalidTargetID
	}
	if target.InventoryID == "" {
		return reportListTargetContext{}, reportdom.ErrInvalidTargetParentID
	}

	inventoryEntity, err := u.inventoryRepo.GetByID(ctx, target.InventoryID)
	if err != nil {
		return reportListTargetContext{}, err
	}
	if inventoryEntity.ProductBlueprintID == "" {
		return reportListTargetContext{}, reportdom.ErrInvalidTargetParentID
	}

	productBlueprint, err := u.productBlueprintRepo.GetByID(
		ctx,
		inventoryEntity.ProductBlueprintID,
	)
	if err != nil {
		return reportListTargetContext{}, err
	}
	if productBlueprint.ID != inventoryEntity.ProductBlueprintID {
		return reportListTargetContext{}, reportdom.ErrInvalidTargetParentID
	}
	if productBlueprint.BrandID == "" {
		return reportListTargetContext{}, reportdom.ErrInvalidTargetAuthorID
	}
	if productBlueprint.CompanyID == "" {
		return reportListTargetContext{}, reportdom.ErrInvalidCompanyID
	}

	return reportListTargetContext{
		List:      target,
		BrandID:   productBlueprint.BrandID,
		CompanyID: productBlueprint.CompanyID,
	}, nil
}

// ============================================================
// TokenBlueprint report
// ============================================================

type ReportTokenBlueprintByAvatarInput struct {
	TokenBlueprintID string
	AvatarID         string
	Reason           reportdom.ReportReason
	Detail           string
}

func (u *ReportUsecase) ReportTokenBlueprintByAvatar(
	ctx context.Context,
	input ReportTokenBlueprintByAvatarInput,
) (reportdom.AddReportResult, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reportdom.AddReportResult{}, err
	}
	if u.tokenBlueprintRepo == nil || u.tokenAccessResolver == nil {
		return reportdom.AddReportResult{}, ErrReportUsecaseNotConfigured
	}
	if input.TokenBlueprintID == "" {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetID
	}
	if input.AvatarID == "" {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidReporterID
	}

	tokenBlueprintEntity, err := u.tokenBlueprintRepo.GetByID(ctx, input.TokenBlueprintID)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if tokenBlueprintEntity == nil || tokenBlueprintEntity.ID != input.TokenBlueprintID {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetID
	}
	if tokenBlueprintEntity.IsHiddenByModeration() {
		return reportdom.AddReportResult{}, reportdom.ErrCannotReportRemovedTarget
	}

	allowed, err := u.tokenAccessResolver.CanReportTokenBlueprint(
		ctx,
		input.AvatarID,
		input.TokenBlueprintID,
	)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if !allowed {
		return reportdom.AddReportResult{}, ErrReportForbidden
	}

	return u.addTokenBlueprintReport(
		ctx,
		*tokenBlueprintEntity,
		input.AvatarID,
		input.Reason,
		input.Detail,
	)
}

func (u *ReportUsecase) addTokenBlueprintReport(
	ctx context.Context,
	target tokenblueprint.TokenBlueprint,
	reporterAvatarID string,
	reason reportdom.ReportReason,
	detail string,
) (reportdom.AddReportResult, error) {
	now := u.now().UTC()

	reportCase, err := reportdom.NewReportCase(reportdom.NewReportCaseParams{
		TargetType:       reportdom.TargetTypeTokenBlueprint,
		TargetID:         target.ID,
		TargetParentID:   target.ID,
		TargetAuthorID:   target.BrandID,
		TargetAuthorType: reportdom.ActorTypeBrand,
		SnapshotTitle:    target.Name,
		SnapshotBody:     target.Description,
		SnapshotRating:   nil,
		CreatedAt:        now,
	})
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	report, err := reportdom.NewReport(reportdom.NewReportParams{
		CaseID:       reportCase.ID,
		ReporterType: reportdom.ActorTypeAvatar,
		ReporterID:   reporterAvatarID,
		CompanyID:    "",
		Reason:       reason,
		Detail:       detail,
		CreatedAt:    now,
	})
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	return u.reportRepo.AddReport(ctx, reportCase, report)
}

// ============================================================
// TokenBlueprint comment report
// ============================================================

type ReportTokenBlueprintCommentByAvatarInput struct {
	TokenBlueprintID string
	CommentID        string
	AvatarID         string
	Reason           reportdom.ReportReason
	Detail           string
}

func (u *ReportUsecase) ReportTokenBlueprintCommentByAvatar(
	ctx context.Context,
	input ReportTokenBlueprintCommentByAvatarInput,
) (reportdom.AddReportResult, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reportdom.AddReportResult{}, err
	}
	if u.tokenCommentRepo == nil || u.tokenAccessResolver == nil {
		return reportdom.AddReportResult{}, ErrReportUsecaseNotConfigured
	}

	comment, err := u.tokenCommentRepo.GetByParentID(
		ctx,
		input.TokenBlueprintID,
		input.CommentID,
	)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if comment.TokenBlueprintID != input.TokenBlueprintID {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetParentID
	}
	if comment.Deleted {
		return reportdom.AddReportResult{}, reportdom.ErrCannotReportRemovedTarget
	}

	targetAuthorType, err := reportActorTypeFromCommentAuthor(comment.AuthorType)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if targetAuthorType == reportdom.ActorTypeAvatar && comment.AuthorID == input.AvatarID {
		return reportdom.AddReportResult{}, ErrReportSelfReport
	}

	allowed, err := u.tokenAccessResolver.CanReportTokenBlueprint(
		ctx,
		input.AvatarID,
		input.TokenBlueprintID,
	)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if !allowed {
		return reportdom.AddReportResult{}, ErrReportForbidden
	}

	return u.addTokenBlueprintCommentReport(
		ctx,
		comment,
		targetAuthorType,
		reportdom.ActorTypeAvatar,
		input.AvatarID,
		"",
		input.Reason,
		input.Detail,
	)
}

type ReportTokenBlueprintCommentByBrandInput struct {
	TokenBlueprintID string
	CommentID        string
	BrandID          string
	CompanyID        string
	Reason           reportdom.ReportReason
	Detail           string
}

func (u *ReportUsecase) ReportTokenBlueprintCommentByBrand(
	ctx context.Context,
	input ReportTokenBlueprintCommentByBrandInput,
) (reportdom.AddReportResult, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reportdom.AddReportResult{}, err
	}
	if u.tokenCommentRepo == nil || u.tokenBlueprintRepo == nil {
		return reportdom.AddReportResult{}, ErrReportUsecaseNotConfigured
	}

	tokenBlueprintEntity, err := u.tokenBlueprintRepo.GetByID(ctx, input.TokenBlueprintID)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if tokenBlueprintEntity == nil ||
		tokenBlueprintEntity.ID != input.TokenBlueprintID ||
		tokenBlueprintEntity.CompanyID != input.CompanyID ||
		tokenBlueprintEntity.BrandID != input.BrandID {
		return reportdom.AddReportResult{}, ErrReportForbidden
	}

	comment, err := u.tokenCommentRepo.GetByParentID(
		ctx,
		input.TokenBlueprintID,
		input.CommentID,
	)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if comment.TokenBlueprintID != input.TokenBlueprintID {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetParentID
	}
	if comment.Deleted {
		return reportdom.AddReportResult{}, reportdom.ErrCannotReportRemovedTarget
	}

	targetAuthorType, err := reportActorTypeFromCommentAuthor(comment.AuthorType)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if targetAuthorType == reportdom.ActorTypeBrand && comment.AuthorID == input.BrandID {
		return reportdom.AddReportResult{}, ErrReportSelfReport
	}

	return u.addTokenBlueprintCommentReport(
		ctx,
		comment,
		targetAuthorType,
		reportdom.ActorTypeBrand,
		input.BrandID,
		input.CompanyID,
		input.Reason,
		input.Detail,
	)
}

func (u *ReportUsecase) addTokenBlueprintCommentReport(
	ctx context.Context,
	comment tokenreview.Comment,
	targetAuthorType reportdom.ActorType,
	reporterType reportdom.ActorType,
	reporterID string,
	companyID string,
	reason reportdom.ReportReason,
	detail string,
) (reportdom.AddReportResult, error) {
	now := u.now().UTC()

	reportCase, err := reportdom.NewReportCase(reportdom.NewReportCaseParams{
		TargetType:       reportdom.TargetTypeTokenBlueprintComment,
		TargetID:         comment.CommentID,
		TargetParentID:   comment.TokenBlueprintID,
		TargetAuthorID:   comment.AuthorID,
		TargetAuthorType: targetAuthorType,
		SnapshotTitle:    "",
		SnapshotBody:     comment.Body,
		SnapshotRating:   nil,
		CreatedAt:        now,
	})
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	report, err := reportdom.NewReport(reportdom.NewReportParams{
		CaseID:       reportCase.ID,
		ReporterType: reporterType,
		ReporterID:   reporterID,
		CompanyID:    companyID,
		Reason:       reason,
		Detail:       detail,
		CreatedAt:    now,
	})
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	return u.reportRepo.AddReport(ctx, reportCase, report)
}

func reportActorTypeFromCommentAuthor(
	authorType tokenreview.AuthorType,
) (reportdom.ActorType, error) {
	switch authorType {
	case tokenreview.AuthorTypeAvatar:
		return reportdom.ActorTypeAvatar, nil
	case tokenreview.AuthorTypeBrand:
		return reportdom.ActorTypeBrand, nil
	default:
		return "", reportdom.ErrInvalidActorType
	}
}

// ============================================================
// Avatar report
// ============================================================

type ReportAvatarByAvatarInput struct {
	TargetAvatarID   string
	ReporterAvatarID string
	Reason           reportdom.ReportReason
	Detail           string
}

func (u *ReportUsecase) ReportAvatarByAvatar(
	ctx context.Context,
	input ReportAvatarByAvatarInput,
) (reportdom.AddReportResult, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reportdom.AddReportResult{}, err
	}
	if u.avatarRepo == nil {
		return reportdom.AddReportResult{}, ErrReportUsecaseNotConfigured
	}
	if input.TargetAvatarID == "" {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetID
	}
	if input.ReporterAvatarID == "" {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidReporterID
	}
	if input.TargetAvatarID == input.ReporterAvatarID {
		return reportdom.AddReportResult{}, ErrReportSelfReport
	}

	targetAvatar, err := u.avatarRepo.GetByID(ctx, input.TargetAvatarID)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if targetAvatar.ID != input.TargetAvatarID {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetID
	}

	return u.addAvatarReport(
		ctx,
		targetAvatar,
		input.ReporterAvatarID,
		input.Reason,
		input.Detail,
	)
}

func (u *ReportUsecase) addAvatarReport(
	ctx context.Context,
	target avatar.Avatar,
	reporterAvatarID string,
	reason reportdom.ReportReason,
	detail string,
) (reportdom.AddReportResult, error) {
	now := u.now().UTC()
	snapshotBody := ""
	if target.Profile != nil {
		snapshotBody = *target.Profile
	}

	reportCase, err := reportdom.NewReportCase(reportdom.NewReportCaseParams{
		TargetType:       reportdom.TargetTypeAvatar,
		TargetID:         target.ID,
		TargetParentID:   target.ID,
		TargetAuthorID:   target.ID,
		TargetAuthorType: reportdom.ActorTypeAvatar,
		SnapshotTitle:    target.AvatarName,
		SnapshotBody:     snapshotBody,
		SnapshotRating:   nil,
		CreatedAt:        now,
	})
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	report, err := reportdom.NewReport(reportdom.NewReportParams{
		CaseID:       reportCase.ID,
		ReporterType: reportdom.ActorTypeAvatar,
		ReporterID:   reporterAvatarID,
		CompanyID:    "",
		Reason:       reason,
		Detail:       detail,
		CreatedAt:    now,
	})
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	return u.reportRepo.AddReport(ctx, reportCase, report)
}

// ============================================================
// Admin read
// ============================================================

func (u *ReportUsecase) ListReportCases(
	ctx context.Context,
	filter reportdom.CaseFilter,
	sort common.Sort,
	page common.Page,
) (common.PageResult[reportdom.ReportCase], error) {
	if err := u.ensureReportRepository(); err != nil {
		return common.PageResult[reportdom.ReportCase]{}, err
	}
	return u.reportRepo.ListCases(ctx, filter, sort, page)
}

func (u *ReportUsecase) GetReportCase(
	ctx context.Context,
	caseID reportdom.CaseID,
) (reportdom.ReportCase, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reportdom.ReportCase{}, err
	}
	if caseID == "" {
		return reportdom.ReportCase{}, reportdom.ErrInvalidCaseID
	}
	return u.reportRepo.GetCase(ctx, caseID)
}

func (u *ReportUsecase) ListReports(
	ctx context.Context,
	caseID reportdom.CaseID,
	filter reportdom.ReportFilter,
	sort common.Sort,
	page common.Page,
) (common.PageResult[reportdom.Report], error) {
	if err := u.ensureReportRepository(); err != nil {
		return common.PageResult[reportdom.Report]{}, err
	}
	if caseID == "" {
		return common.PageResult[reportdom.Report]{}, reportdom.ErrInvalidCaseID
	}
	return u.reportRepo.ListReports(ctx, caseID, filter, sort, page)
}

// ============================================================
// Decision notifications
// ============================================================

func (u *ReportUsecase) ListDecisionNotificationsForAvatar(
	ctx context.Context,
	avatarID string,
	isRead *bool,
	page common.Page,
) (common.PageResult[reportdom.DecisionNotification], error) {
	if err := u.ensureDecisionNotificationRepository(); err != nil {
		return common.PageResult[reportdom.DecisionNotification]{}, err
	}
	if avatarID == "" {
		return common.PageResult[reportdom.DecisionNotification]{}, reportdom.ErrInvalidReporterID
	}

	recipientType := reportdom.ActorTypeAvatar
	return u.decisionNotificationRepo.List(
		ctx,
		reportdom.DecisionNotificationFilter{
			RecipientType: &recipientType,
			RecipientID:   avatarID,
			IsRead:        isRead,
		},
		common.Sort{
			Column: "createdAt",
			Order:  common.SortDesc,
		},
		page,
	)
}

func (u *ReportUsecase) MarkDecisionNotificationReadForAvatar(
	ctx context.Context,
	notificationID reportdom.DecisionNotificationID,
	avatarID string,
) (reportdom.DecisionNotification, error) {
	if err := u.ensureDecisionNotificationRepository(); err != nil {
		return reportdom.DecisionNotification{}, err
	}
	if notificationID == "" {
		return reportdom.DecisionNotification{}, reportdom.ErrInvalidDecisionNotificationID
	}
	if avatarID == "" {
		return reportdom.DecisionNotification{}, reportdom.ErrInvalidReporterID
	}

	notification, err := u.decisionNotificationRepo.GetByID(ctx, notificationID)
	if err != nil {
		return reportdom.DecisionNotification{}, err
	}
	if notification.RecipientType != reportdom.ActorTypeAvatar ||
		notification.RecipientID != avatarID {
		return reportdom.DecisionNotification{}, ErrReportForbidden
	}

	return u.decisionNotificationRepo.MarkRead(
		ctx,
		notificationID,
		reportdom.ActorTypeAvatar,
		avatarID,
		u.now().UTC(),
	)
}

func (u *ReportUsecase) ListDecisionNotificationsForCompany(
	ctx context.Context,
	companyID string,
	isRead *bool,
	page common.Page,
) (common.PageResult[reportdom.DecisionNotification], error) {
	if err := u.ensureDecisionNotificationRepository(); err != nil {
		return common.PageResult[reportdom.DecisionNotification]{}, err
	}
	if companyID == "" {
		return common.PageResult[reportdom.DecisionNotification]{}, reportdom.ErrInvalidCompanyID
	}

	recipientType := reportdom.ActorTypeBrand
	return u.decisionNotificationRepo.List(
		ctx,
		reportdom.DecisionNotificationFilter{
			RecipientType: &recipientType,
			CompanyID:     companyID,
			IsRead:        isRead,
		},
		common.Sort{
			Column: "createdAt",
			Order:  common.SortDesc,
		},
		page,
	)
}

func (u *ReportUsecase) MarkDecisionNotificationReadForCompany(
	ctx context.Context,
	notificationID reportdom.DecisionNotificationID,
	companyID string,
) (reportdom.DecisionNotification, error) {
	if err := u.ensureDecisionNotificationRepository(); err != nil {
		return reportdom.DecisionNotification{}, err
	}
	if notificationID == "" {
		return reportdom.DecisionNotification{}, reportdom.ErrInvalidDecisionNotificationID
	}
	if companyID == "" {
		return reportdom.DecisionNotification{}, reportdom.ErrInvalidCompanyID
	}

	notification, err := u.decisionNotificationRepo.GetByID(ctx, notificationID)
	if err != nil {
		return reportdom.DecisionNotification{}, err
	}
	if notification.RecipientType != reportdom.ActorTypeBrand ||
		notification.CompanyID != companyID {
		return reportdom.DecisionNotification{}, ErrReportForbidden
	}

	return u.decisionNotificationRepo.MarkRead(
		ctx,
		notificationID,
		reportdom.ActorTypeBrand,
		notification.RecipientID,
		u.now().UTC(),
	)
}

// ============================================================
// Admin decision
// ============================================================

type ReportDecision string

const (
	ReportDecisionKeep   ReportDecision = "KEEP"
	ReportDecisionRemove ReportDecision = "REMOVE"
)

type DecideReportCaseInput struct {
	CaseID    reportdom.CaseID
	Decision  ReportDecision
	Reason    string
	DecidedBy string
}

func (u *ReportUsecase) DecideReportCase(
	ctx context.Context,
	input DecideReportCaseInput,
) (reportdom.ReportCase, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reportdom.ReportCase{}, err
	}
	if input.CaseID == "" {
		return reportdom.ReportCase{}, reportdom.ErrInvalidCaseID
	}

	switch input.Decision {
	case ReportDecisionKeep, ReportDecisionRemove:
		if err := u.ensureDecisionNotificationRepository(); err != nil {
			return reportdom.ReportCase{}, err
		}
	default:
		return reportdom.ReportCase{}, ErrReportInvalidDecision
	}

	var (
		decidedCase reportdom.ReportCase
		err         error
	)

	switch input.Decision {
	case ReportDecisionKeep:
		decidedCase, err = u.keepReportCase(ctx, input)
	case ReportDecisionRemove:
		decidedCase, err = u.removeReportCase(ctx, input)
	}
	if err != nil {
		return reportdom.ReportCase{}, err
	}

	if decidedCase.Status == reportdom.CaseStatusKept ||
		decidedCase.Status == reportdom.CaseStatusRemoved {
		if err := u.createDecisionNotifications(ctx, decidedCase); err != nil {
			return reportdom.ReportCase{}, err
		}
	}

	return decidedCase, nil
}

func (u *ReportUsecase) keepReportCase(
	ctx context.Context,
	input DecideReportCaseInput,
) (reportdom.ReportCase, error) {
	reportCase, err := u.reportRepo.GetCase(ctx, input.CaseID)
	if err != nil {
		return reportdom.ReportCase{}, err
	}

	if err := reportCase.Keep(
		input.Reason,
		u.now().UTC(),
		input.DecidedBy,
	); err != nil {
		return reportdom.ReportCase{}, err
	}

	return u.reportRepo.UpdateCase(
		ctx,
		reportCase.ID,
		reportdom.NewCasePatchFromEntity(reportCase),
	)
}

func (u *ReportUsecase) removeReportCase(
	ctx context.Context,
	input DecideReportCaseInput,
) (reportdom.ReportCase, error) {
	reportCase, err := u.reportRepo.GetCase(ctx, input.CaseID)
	if err != nil {
		return reportdom.ReportCase{}, err
	}

	// REMOVED 済みの LIST / TOKEN_BLUEPRINT / AVATAR 裁定は、対象側の措置だけを
	// 再実行できるようにする。各 moderator は冪等に実装する。
	if reportCase.IsRemoved() {
		switch reportCase.TargetType {
		case reportdom.TargetTypeList:
			if err := u.suspendListTarget(
				ctx,
				reportCase,
				input.Reason,
				input.DecidedBy,
			); err != nil {
				return reportdom.ReportCase{}, err
			}

		case reportdom.TargetTypeTokenBlueprint:
			if err := u.hideTokenBlueprintTarget(
				ctx,
				reportCase,
				input.Reason,
				input.DecidedBy,
			); err != nil {
				return reportdom.ReportCase{}, err
			}

		case reportdom.TargetTypeAvatar:
			if err := u.suspendAvatarResaleTarget(
				ctx,
				reportCase,
				input.Reason,
				input.DecidedBy,
			); err != nil {
				return reportdom.ReportCase{}, err
			}
		}
		return reportCase, nil
	}

	// IMPORTANT:
	// REMOVE の対象側処理を先に完了する。
	// 商品レビュー/コメントは削除、LIST は Mall 上で出品停止、
	// TOKEN_BLUEPRINT は AMOL UI 上で非表示、AVATAR は再販サービスのみ利用停止とする。
	// 対象側処理に失敗した場合、ReportCase を REMOVED にしてはいけない。
	switch reportCase.TargetType {
	case reportdom.TargetTypeProductBlueprintReview:
		if err := u.removeProductBlueprintReviewTarget(
			ctx,
			reportCase,
			input.Reason,
			input.DecidedBy,
		); err != nil {
			return reportdom.ReportCase{}, err
		}

	case reportdom.TargetTypeList:
		if err := u.suspendListTarget(
			ctx,
			reportCase,
			input.Reason,
			input.DecidedBy,
		); err != nil {
			return reportdom.ReportCase{}, err
		}

	case reportdom.TargetTypeTokenBlueprint:
		if err := u.hideTokenBlueprintTarget(
			ctx,
			reportCase,
			input.Reason,
			input.DecidedBy,
		); err != nil {
			return reportdom.ReportCase{}, err
		}

	case reportdom.TargetTypeTokenBlueprintComment:
		if err := u.removeTokenBlueprintCommentTarget(
			ctx,
			reportCase,
		); err != nil {
			return reportdom.ReportCase{}, err
		}

	case reportdom.TargetTypeAvatar:
		if err := u.suspendAvatarResaleTarget(
			ctx,
			reportCase,
			input.Reason,
			input.DecidedBy,
		); err != nil {
			return reportdom.ReportCase{}, err
		}

	default:
		return reportdom.ReportCase{}, reportdom.ErrInvalidTargetType
	}

	if err := reportCase.Remove(
		input.Reason,
		u.now().UTC(),
		input.DecidedBy,
	); err != nil {
		return reportdom.ReportCase{}, err
	}

	updatedCase, err := u.reportRepo.UpdateCase(
		ctx,
		reportCase.ID,
		reportdom.NewCasePatchFromEntity(reportCase),
	)
	if err != nil {
		return reportdom.ReportCase{}, err
	}

	// LIST / AVATAR は REMOVED を永続化した後でもう一度対象側の措置を実行する。
	// 1 回目の措置と REMOVED 永続化の間に対象が再公開される競合を閉じるための後処理。
	// 失敗した場合は次回の同じ REMOVE 裁定で REMOVED 済み分岐から再試行できる。
	switch updatedCase.TargetType {
	case reportdom.TargetTypeList:
		if err := u.suspendListTarget(
			ctx,
			updatedCase,
			input.Reason,
			input.DecidedBy,
		); err != nil {
			return reportdom.ReportCase{}, err
		}

	case reportdom.TargetTypeAvatar:
		if err := u.suspendAvatarResaleTarget(
			ctx,
			updatedCase,
			input.Reason,
			input.DecidedBy,
		); err != nil {
			return reportdom.ReportCase{}, err
		}
	}

	return updatedCase, nil
}

func (u *ReportUsecase) createDecisionNotifications(
	ctx context.Context,
	reportCase reportdom.ReportCase,
) error {
	if err := u.ensureReportRepository(); err != nil {
		return err
	}
	if err := u.ensureDecisionNotificationRepository(); err != nil {
		return err
	}
	if reportCase.DecidedAt == nil || reportCase.DecidedAt.IsZero() {
		return reportdom.ErrDecisionNotificationCaseNotDecided
	}

	decidedAt := reportCase.DecidedAt.UTC()
	filter := reportdom.ReportFilter{
		CaseID: reportCase.ID,
		CreatedAt: common.TimeRange{
			To: &decidedAt,
		},
	}
	sort := common.Sort{
		Column: "createdAt",
		Order:  common.SortAsc,
	}

	const perPage = 100

	for pageNumber := 1; ; pageNumber++ {
		result, err := u.reportRepo.ListReports(
			ctx,
			reportCase.ID,
			filter,
			sort,
			common.Page{
				Number:  pageNumber,
				PerPage: perPage,
			},
		)
		if err != nil {
			return err
		}

		for _, report := range result.Items {
			notification, err := reportdom.NewDecisionNotification(
				reportCase,
				report,
				decidedAt,
			)
			if err != nil {
				return err
			}

			if _, err := u.decisionNotificationRepo.CreateIfAbsent(
				ctx,
				notification,
			); err != nil {
				return err
			}
		}

		if pageNumber >= result.TotalPages {
			break
		}
	}

	return u.createTargetEnforcementDecisionNotification(
		ctx,
		reportCase,
		decidedAt,
	)
}

func (u *ReportUsecase) createTargetEnforcementDecisionNotification(
	ctx context.Context,
	reportCase reportdom.ReportCase,
	createdAt time.Time,
) error {
	if reportCase.Status != reportdom.CaseStatusRemoved {
		return nil
	}

	companyID := ""
	notificationCase := reportCase

	// PRODUCT_BLUEPRINT_REVIEW はレビュー投稿Avatar、AVATAR は対象Avatar自身、
	// LIST は出品元Brand、TOKEN_BLUEPRINT はそのTokenBlueprintを所有するBrandへ
	// 措置通知を送る。TOKEN_BLUEPRINT_COMMENT は現時点では対象者通知の対象外とする。
	switch reportCase.TargetType {
	case reportdom.TargetTypeProductBlueprintReview,
		reportdom.TargetTypeAvatar:

	case reportdom.TargetTypeList:
		targetContext, err := u.resolveListTargetContext(ctx, reportCase.TargetID)
		if err != nil {
			return err
		}

		notificationCase.TargetAuthorID = targetContext.BrandID
		notificationCase.TargetAuthorType = reportdom.ActorTypeBrand
		companyID = targetContext.CompanyID

	case reportdom.TargetTypeTokenBlueprint:
		if u.tokenBlueprintRepo == nil {
			return ErrReportUsecaseNotConfigured
		}

		target, err := u.tokenBlueprintRepo.GetByID(ctx, reportCase.TargetID)
		if err != nil {
			return err
		}
		if target == nil || target.ID != reportCase.TargetID {
			return reportdom.ErrInvalidTargetID
		}
		if target.BrandID == "" {
			return reportdom.ErrInvalidTargetAuthorID
		}
		if target.CompanyID == "" {
			return reportdom.ErrInvalidCompanyID
		}

		// 旧ReportCaseがCreatedBy(member ID)をTargetAuthorIDに保持している場合でも、
		// 通知時にはTokenBlueprintの現在のBrandIDを正しい宛先として利用する。
		notificationCase.TargetAuthorID = target.BrandID
		notificationCase.TargetAuthorType = reportdom.ActorTypeBrand
		companyID = target.CompanyID

	default:
		return nil
	}

	notification, err := reportdom.NewTargetEnforcementNotification(
		notificationCase,
		companyID,
		createdAt,
	)
	if err != nil {
		return err
	}

	_, err = u.decisionNotificationRepo.CreateIfAbsent(
		ctx,
		notification,
	)
	return err
}

func (u *ReportUsecase) removeProductBlueprintReviewTarget(
	ctx context.Context,
	reportCase reportdom.ReportCase,
	reason string,
	adminID string,
) error {
	if u.productReviewModerator == nil {
		return ErrReportUsecaseNotConfigured
	}

	_, err := u.productReviewModerator.RemoveProductBlueprintReviewByAdmin(
		ctx,
		RemoveProductBlueprintReviewByAdminInput{
			ProductBlueprintID: reportCase.TargetParentID,
			ReviewID:           reportCase.TargetID,
			Reason:             reason,
			AdminID:            adminID,
		},
	)
	return err
}

func (u *ReportUsecase) suspendListTarget(
	ctx context.Context,
	reportCase reportdom.ReportCase,
	reason string,
	adminID string,
) error {
	if u.listModerator == nil {
		return ErrReportUsecaseNotConfigured
	}

	return u.listModerator.SuspendListByAdmin(
		ctx,
		SuspendListByAdminInput{
			ListID:  reportCase.TargetID,
			Reason:  reason,
			AdminID: adminID,
		},
	)
}

func (u *ReportUsecase) hideTokenBlueprintTarget(
	ctx context.Context,
	reportCase reportdom.ReportCase,
	reason string,
	adminID string,
) error {
	if u.tokenBlueprintModerator == nil {
		return ErrReportUsecaseNotConfigured
	}

	return u.tokenBlueprintModerator.HideTokenBlueprintByAdmin(
		ctx,
		HideTokenBlueprintByAdminInput{
			TokenBlueprintID: reportCase.TargetID,
			Reason:           reason,
			AdminID:          adminID,
		},
	)
}

func (u *ReportUsecase) removeTokenBlueprintCommentTarget(
	ctx context.Context,
	reportCase reportdom.ReportCase,
) error {
	if u.tokenCommentModerator == nil {
		return ErrReportUsecaseNotConfigured
	}

	return u.tokenCommentModerator.RemoveCommentByAdmin(
		ctx,
		RemoveCommentByAdminInput{
			TokenBlueprintID: reportCase.TargetParentID,
			CommentID:        reportCase.TargetID,
		},
	)
}

func (u *ReportUsecase) suspendAvatarResaleTarget(
	ctx context.Context,
	reportCase reportdom.ReportCase,
	reason string,
	adminID string,
) error {
	if u.avatarResaleModerator == nil {
		return ErrReportUsecaseNotConfigured
	}

	return u.avatarResaleModerator.SuspendAvatarResaleByAdmin(
		ctx,
		SuspendAvatarResaleByAdminInput{
			AvatarID: reportCase.TargetID,
			Reason:   reason,
			AdminID:  adminID,
		},
	)
}
