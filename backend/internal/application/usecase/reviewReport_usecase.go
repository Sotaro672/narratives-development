// backend/internal/application/usecase/reviewReport_usecase.go
package usecase

import (
	"context"
	"errors"
	"time"

	applicationport "narratives/internal/application/port"
	avatar "narratives/internal/domain/avatar"
	common "narratives/internal/domain/common"
	pbr "narratives/internal/domain/productBlueprintReview"
	reviewreport "narratives/internal/domain/reviewReport"
	tokenblueprint "narratives/internal/domain/tokenBlueprint"
	tokenreview "narratives/internal/domain/tokenBlueprint_review"
)

// ============================================================
// Errors
// ============================================================

var (
	ErrReviewReportUsecaseNotConfigured = errors.New("reviewReport_usecase: not configured")
	ErrReviewReportForbidden            = errors.New("reviewReport_usecase: forbidden")
	ErrReviewReportSelfReport           = errors.New("reviewReport_usecase: self report is not allowed")
	ErrReviewReportInvalidDecision      = errors.New("reviewReport_usecase: invalid decision")
)

// ============================================================
// Ports
// ============================================================

// ReviewReportProductReviewModerator owns Admin moderation of product reviews.
type ReviewReportProductReviewModerator interface {
	RemoveProductBlueprintReviewByAdmin(
		ctx context.Context,
		in RemoveProductBlueprintReviewByAdminInput,
	) (pbr.Review, error)
}

// ReviewReportTokenCommentModerator owns Admin moderation of token comments.
type ReviewReportTokenCommentModerator interface {
	RemoveCommentByAdmin(
		ctx context.Context,
		input RemoveCommentByAdminInput,
	) error
}

// ReviewReportAvatarResaleModerator owns Admin moderation of avatar resale access.
// AVATAR + REMOVE in ReviewReport means resale-service suspension only.
// It must not delete or disable the avatar itself.
type ReviewReportAvatarResaleModerator interface {
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

type ReviewReportUsecase struct {
	reportRepo               reviewreport.RepositoryPort
	decisionNotificationRepo reviewreport.DecisionNotificationRepository

	productReviewRepo       pbr.Repository
	productBlueprintRepo    applicationport.ProductBlueprintGetter
	productPurchaseResolver applicationport.OwnedProductResolver
	productReviewModerator  ReviewReportProductReviewModerator

	tokenCommentRepo      tokenreview.CommentRepository
	tokenBlueprintRepo    tokenblueprint.RepositoryPort
	tokenAccessResolver   applicationport.ReviewReportTokenAccessResolver
	tokenCommentModerator ReviewReportTokenCommentModerator

	avatarRepo            avatar.Repository
	avatarResaleModerator ReviewReportAvatarResaleModerator

	now func() time.Time
}

type ReviewReportUsecaseDeps struct {
	ReportRepo               reviewreport.RepositoryPort
	DecisionNotificationRepo reviewreport.DecisionNotificationRepository

	ProductReviewRepo       pbr.Repository
	ProductBlueprintRepo    applicationport.ProductBlueprintGetter
	ProductPurchaseResolver applicationport.OwnedProductResolver
	ProductReviewModerator  ReviewReportProductReviewModerator

	TokenCommentRepo      tokenreview.CommentRepository
	TokenBlueprintRepo    tokenblueprint.RepositoryPort
	TokenAccessResolver   applicationport.ReviewReportTokenAccessResolver
	TokenCommentModerator ReviewReportTokenCommentModerator

	AvatarRepo            avatar.Repository
	AvatarResaleModerator ReviewReportAvatarResaleModerator

	Now func() time.Time
}

func NewReviewReportUsecase(deps ReviewReportUsecaseDeps) *ReviewReportUsecase {
	now := deps.Now
	if now == nil {
		now = time.Now
	}

	return &ReviewReportUsecase{
		reportRepo:               deps.ReportRepo,
		decisionNotificationRepo: deps.DecisionNotificationRepo,
		productReviewRepo:        deps.ProductReviewRepo,
		productBlueprintRepo:     deps.ProductBlueprintRepo,
		productPurchaseResolver:  deps.ProductPurchaseResolver,
		productReviewModerator:   deps.ProductReviewModerator,
		tokenCommentRepo:         deps.TokenCommentRepo,
		tokenBlueprintRepo:       deps.TokenBlueprintRepo,
		tokenAccessResolver:      deps.TokenAccessResolver,
		tokenCommentModerator:    deps.TokenCommentModerator,
		avatarRepo:               deps.AvatarRepo,
		avatarResaleModerator:    deps.AvatarResaleModerator,
		now:                      now,
	}
}

func (u *ReviewReportUsecase) ensureReportRepository() error {
	if u == nil || u.reportRepo == nil {
		return ErrReviewReportUsecaseNotConfigured
	}
	return nil
}

func (u *ReviewReportUsecase) ensureDecisionNotificationRepository() error {
	if u == nil || u.decisionNotificationRepo == nil {
		return ErrReviewReportUsecaseNotConfigured
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
	Reason             reviewreport.ReportReason
	Detail             string
}

func (u *ReviewReportUsecase) ReportProductBlueprintReviewByAvatar(
	ctx context.Context,
	input ReportProductBlueprintReviewByAvatarInput,
) (reviewreport.AddReportResult, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reviewreport.AddReportResult{}, err
	}
	if u.productReviewRepo == nil || u.productPurchaseResolver == nil {
		return reviewreport.AddReportResult{}, ErrReviewReportUsecaseNotConfigured
	}

	review, err := u.productReviewRepo.GetByProductBlueprintID(
		ctx,
		input.ProductBlueprintID,
		input.ReviewID,
	)
	if err != nil {
		return reviewreport.AddReportResult{}, err
	}
	if review.ProductBlueprintID != input.ProductBlueprintID {
		return reviewreport.AddReportResult{}, reviewreport.ErrInvalidTargetParentID
	}
	if review.Status == pbr.ReviewStatusRemoved {
		return reviewreport.AddReportResult{}, reviewreport.ErrCannotReportRemovedTarget
	}
	if review.AvatarID == input.AvatarID {
		return reviewreport.AddReportResult{}, ErrReviewReportSelfReport
	}

	allowed, err := u.productPurchaseResolver.HasOwnedProductBlueprint(
		ctx,
		input.AvatarID,
		input.ProductBlueprintID,
	)
	if err != nil {
		return reviewreport.AddReportResult{}, err
	}
	if !allowed {
		return reviewreport.AddReportResult{}, ErrReviewReportForbidden
	}

	return u.addProductBlueprintReviewReport(
		ctx,
		review,
		reviewreport.ActorTypeAvatar,
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
	Reason             reviewreport.ReportReason
	Detail             string
}

func (u *ReviewReportUsecase) ReportProductBlueprintReviewByBrand(
	ctx context.Context,
	input ReportProductBlueprintReviewByBrandInput,
) (reviewreport.AddReportResult, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reviewreport.AddReportResult{}, err
	}
	if u.productReviewRepo == nil || u.productBlueprintRepo == nil {
		return reviewreport.AddReportResult{}, ErrReviewReportUsecaseNotConfigured
	}

	productBlueprint, err := u.productBlueprintRepo.GetByID(ctx, input.ProductBlueprintID)
	if err != nil {
		return reviewreport.AddReportResult{}, err
	}
	if productBlueprint.ID != input.ProductBlueprintID ||
		productBlueprint.CompanyID != input.CompanyID ||
		productBlueprint.BrandID != input.BrandID {
		return reviewreport.AddReportResult{}, ErrReviewReportForbidden
	}

	review, err := u.productReviewRepo.GetByProductBlueprintID(
		ctx,
		input.ProductBlueprintID,
		input.ReviewID,
	)
	if err != nil {
		return reviewreport.AddReportResult{}, err
	}
	if review.ProductBlueprintID != input.ProductBlueprintID {
		return reviewreport.AddReportResult{}, reviewreport.ErrInvalidTargetParentID
	}
	if review.Status == pbr.ReviewStatusRemoved {
		return reviewreport.AddReportResult{}, reviewreport.ErrCannotReportRemovedTarget
	}

	return u.addProductBlueprintReviewReport(
		ctx,
		review,
		reviewreport.ActorTypeBrand,
		input.BrandID,
		input.CompanyID,
		input.Reason,
		input.Detail,
	)
}

func (u *ReviewReportUsecase) addProductBlueprintReviewReport(
	ctx context.Context,
	review pbr.Review,
	reporterType reviewreport.ActorType,
	reporterID string,
	companyID string,
	reason reviewreport.ReportReason,
	detail string,
) (reviewreport.AddReportResult, error) {
	now := u.now().UTC()
	rating := int(review.Rating)

	reportCase, err := reviewreport.NewReportCase(
		reviewReportNewProductReviewCaseParams(
			review,
			rating,
			now,
		),
	)
	if err != nil {
		return reviewreport.AddReportResult{}, err
	}

	report, err := reviewreport.NewReport(reviewreport.NewReportParams{
		CaseID:       reportCase.ID,
		ReporterType: reporterType,
		ReporterID:   reporterID,
		CompanyID:    companyID,
		Reason:       reason,
		Detail:       detail,
		CreatedAt:    now,
	})
	if err != nil {
		return reviewreport.AddReportResult{}, err
	}

	return u.reportRepo.AddReport(ctx, reportCase, report)
}

func reviewReportNewProductReviewCaseParams(
	review pbr.Review,
	rating int,
	now time.Time,
) reviewreport.NewReportCaseParams {
	return reviewreport.NewReportCaseParams{
		TargetType:       reviewreport.TargetTypeProductBlueprintReview,
		TargetID:         string(review.ID),
		TargetParentID:   review.ProductBlueprintID,
		TargetAuthorID:   review.AvatarID,
		TargetAuthorType: reviewreport.ActorTypeAvatar,
		SnapshotTitle:    review.Title,
		SnapshotBody:     review.Body,
		SnapshotRating:   &rating,
		CreatedAt:        now,
	}
}

// ============================================================
// TokenBlueprint comment report
// ============================================================

type ReportTokenBlueprintCommentByAvatarInput struct {
	TokenBlueprintID string
	CommentID        string
	AvatarID         string
	Reason           reviewreport.ReportReason
	Detail           string
}

func (u *ReviewReportUsecase) ReportTokenBlueprintCommentByAvatar(
	ctx context.Context,
	input ReportTokenBlueprintCommentByAvatarInput,
) (reviewreport.AddReportResult, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reviewreport.AddReportResult{}, err
	}
	if u.tokenCommentRepo == nil || u.tokenAccessResolver == nil {
		return reviewreport.AddReportResult{}, ErrReviewReportUsecaseNotConfigured
	}

	comment, err := u.tokenCommentRepo.GetByParentID(
		ctx,
		input.TokenBlueprintID,
		input.CommentID,
	)
	if err != nil {
		return reviewreport.AddReportResult{}, err
	}
	if comment.TokenBlueprintID != input.TokenBlueprintID {
		return reviewreport.AddReportResult{}, reviewreport.ErrInvalidTargetParentID
	}
	if comment.Deleted {
		return reviewreport.AddReportResult{}, reviewreport.ErrCannotReportRemovedTarget
	}

	targetAuthorType, err := reviewReportActorTypeFromCommentAuthor(comment.AuthorType)
	if err != nil {
		return reviewreport.AddReportResult{}, err
	}
	if targetAuthorType == reviewreport.ActorTypeAvatar && comment.AuthorID == input.AvatarID {
		return reviewreport.AddReportResult{}, ErrReviewReportSelfReport
	}

	allowed, err := u.tokenAccessResolver.CanReportTokenBlueprintComment(
		ctx,
		input.AvatarID,
		input.TokenBlueprintID,
	)
	if err != nil {
		return reviewreport.AddReportResult{}, err
	}
	if !allowed {
		return reviewreport.AddReportResult{}, ErrReviewReportForbidden
	}

	return u.addTokenBlueprintCommentReport(
		ctx,
		comment,
		targetAuthorType,
		reviewreport.ActorTypeAvatar,
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
	Reason           reviewreport.ReportReason
	Detail           string
}

func (u *ReviewReportUsecase) ReportTokenBlueprintCommentByBrand(
	ctx context.Context,
	input ReportTokenBlueprintCommentByBrandInput,
) (reviewreport.AddReportResult, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reviewreport.AddReportResult{}, err
	}
	if u.tokenCommentRepo == nil || u.tokenBlueprintRepo == nil {
		return reviewreport.AddReportResult{}, ErrReviewReportUsecaseNotConfigured
	}

	tokenBlueprintEntity, err := u.tokenBlueprintRepo.GetByID(ctx, input.TokenBlueprintID)
	if err != nil {
		return reviewreport.AddReportResult{}, err
	}
	if tokenBlueprintEntity == nil ||
		tokenBlueprintEntity.ID != input.TokenBlueprintID ||
		tokenBlueprintEntity.CompanyID != input.CompanyID ||
		tokenBlueprintEntity.BrandID != input.BrandID {
		return reviewreport.AddReportResult{}, ErrReviewReportForbidden
	}

	comment, err := u.tokenCommentRepo.GetByParentID(
		ctx,
		input.TokenBlueprintID,
		input.CommentID,
	)
	if err != nil {
		return reviewreport.AddReportResult{}, err
	}
	if comment.TokenBlueprintID != input.TokenBlueprintID {
		return reviewreport.AddReportResult{}, reviewreport.ErrInvalidTargetParentID
	}
	if comment.Deleted {
		return reviewreport.AddReportResult{}, reviewreport.ErrCannotReportRemovedTarget
	}

	targetAuthorType, err := reviewReportActorTypeFromCommentAuthor(comment.AuthorType)
	if err != nil {
		return reviewreport.AddReportResult{}, err
	}
	if targetAuthorType == reviewreport.ActorTypeBrand && comment.AuthorID == input.BrandID {
		return reviewreport.AddReportResult{}, ErrReviewReportSelfReport
	}

	return u.addTokenBlueprintCommentReport(
		ctx,
		comment,
		targetAuthorType,
		reviewreport.ActorTypeBrand,
		input.BrandID,
		input.CompanyID,
		input.Reason,
		input.Detail,
	)
}

func (u *ReviewReportUsecase) addTokenBlueprintCommentReport(
	ctx context.Context,
	comment tokenreview.Comment,
	targetAuthorType reviewreport.ActorType,
	reporterType reviewreport.ActorType,
	reporterID string,
	companyID string,
	reason reviewreport.ReportReason,
	detail string,
) (reviewreport.AddReportResult, error) {
	now := u.now().UTC()

	reportCase, err := reviewreport.NewReportCase(reviewreport.NewReportCaseParams{
		TargetType:       reviewreport.TargetTypeTokenBlueprintComment,
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
		return reviewreport.AddReportResult{}, err
	}

	report, err := reviewreport.NewReport(reviewreport.NewReportParams{
		CaseID:       reportCase.ID,
		ReporterType: reporterType,
		ReporterID:   reporterID,
		CompanyID:    companyID,
		Reason:       reason,
		Detail:       detail,
		CreatedAt:    now,
	})
	if err != nil {
		return reviewreport.AddReportResult{}, err
	}

	return u.reportRepo.AddReport(ctx, reportCase, report)
}

func reviewReportActorTypeFromCommentAuthor(
	authorType tokenreview.AuthorType,
) (reviewreport.ActorType, error) {
	switch authorType {
	case tokenreview.AuthorTypeAvatar:
		return reviewreport.ActorTypeAvatar, nil
	case tokenreview.AuthorTypeBrand:
		return reviewreport.ActorTypeBrand, nil
	default:
		return "", reviewreport.ErrInvalidActorType
	}
}

// ============================================================
// Avatar report
// ============================================================

type ReportAvatarByAvatarInput struct {
	TargetAvatarID   string
	ReporterAvatarID string
	Reason           reviewreport.ReportReason
	Detail           string
}

func (u *ReviewReportUsecase) ReportAvatarByAvatar(
	ctx context.Context,
	input ReportAvatarByAvatarInput,
) (reviewreport.AddReportResult, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reviewreport.AddReportResult{}, err
	}
	if u.avatarRepo == nil {
		return reviewreport.AddReportResult{}, ErrReviewReportUsecaseNotConfigured
	}
	if input.TargetAvatarID == "" {
		return reviewreport.AddReportResult{}, reviewreport.ErrInvalidTargetID
	}
	if input.ReporterAvatarID == "" {
		return reviewreport.AddReportResult{}, reviewreport.ErrInvalidReporterID
	}
	if input.TargetAvatarID == input.ReporterAvatarID {
		return reviewreport.AddReportResult{}, ErrReviewReportSelfReport
	}

	targetAvatar, err := u.avatarRepo.GetByID(ctx, input.TargetAvatarID)
	if err != nil {
		return reviewreport.AddReportResult{}, err
	}
	if targetAvatar.ID != input.TargetAvatarID {
		return reviewreport.AddReportResult{}, reviewreport.ErrInvalidTargetID
	}

	return u.addAvatarReport(
		ctx,
		targetAvatar,
		input.ReporterAvatarID,
		input.Reason,
		input.Detail,
	)
}

func (u *ReviewReportUsecase) addAvatarReport(
	ctx context.Context,
	target avatar.Avatar,
	reporterAvatarID string,
	reason reviewreport.ReportReason,
	detail string,
) (reviewreport.AddReportResult, error) {
	now := u.now().UTC()
	snapshotBody := ""
	if target.Profile != nil {
		snapshotBody = *target.Profile
	}

	reportCase, err := reviewreport.NewReportCase(reviewreport.NewReportCaseParams{
		TargetType:       reviewreport.TargetTypeAvatar,
		TargetID:         target.ID,
		TargetParentID:   target.ID,
		TargetAuthorID:   target.ID,
		TargetAuthorType: reviewreport.ActorTypeAvatar,
		SnapshotTitle:    target.AvatarName,
		SnapshotBody:     snapshotBody,
		SnapshotRating:   nil,
		CreatedAt:        now,
	})
	if err != nil {
		return reviewreport.AddReportResult{}, err
	}

	report, err := reviewreport.NewReport(reviewreport.NewReportParams{
		CaseID:       reportCase.ID,
		ReporterType: reviewreport.ActorTypeAvatar,
		ReporterID:   reporterAvatarID,
		CompanyID:    "",
		Reason:       reason,
		Detail:       detail,
		CreatedAt:    now,
	})
	if err != nil {
		return reviewreport.AddReportResult{}, err
	}

	return u.reportRepo.AddReport(ctx, reportCase, report)
}

// ============================================================
// Admin read
// ============================================================

func (u *ReviewReportUsecase) ListReportCases(
	ctx context.Context,
	filter reviewreport.CaseFilter,
	sort common.Sort,
	page common.Page,
) (common.PageResult[reviewreport.ReportCase], error) {
	if err := u.ensureReportRepository(); err != nil {
		return common.PageResult[reviewreport.ReportCase]{}, err
	}
	return u.reportRepo.ListCases(ctx, filter, sort, page)
}

func (u *ReviewReportUsecase) GetReportCase(
	ctx context.Context,
	caseID reviewreport.CaseID,
) (reviewreport.ReportCase, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reviewreport.ReportCase{}, err
	}
	if caseID == "" {
		return reviewreport.ReportCase{}, reviewreport.ErrInvalidCaseID
	}
	return u.reportRepo.GetCase(ctx, caseID)
}

func (u *ReviewReportUsecase) ListReports(
	ctx context.Context,
	caseID reviewreport.CaseID,
	filter reviewreport.ReportFilter,
	sort common.Sort,
	page common.Page,
) (common.PageResult[reviewreport.Report], error) {
	if err := u.ensureReportRepository(); err != nil {
		return common.PageResult[reviewreport.Report]{}, err
	}
	if caseID == "" {
		return common.PageResult[reviewreport.Report]{}, reviewreport.ErrInvalidCaseID
	}
	return u.reportRepo.ListReports(ctx, caseID, filter, sort, page)
}

// ============================================================
// Decision notifications
// ============================================================

func (u *ReviewReportUsecase) ListDecisionNotificationsForAvatar(
	ctx context.Context,
	avatarID string,
	isRead *bool,
	page common.Page,
) (common.PageResult[reviewreport.DecisionNotification], error) {
	if err := u.ensureDecisionNotificationRepository(); err != nil {
		return common.PageResult[reviewreport.DecisionNotification]{}, err
	}
	if avatarID == "" {
		return common.PageResult[reviewreport.DecisionNotification]{}, reviewreport.ErrInvalidReporterID
	}

	recipientType := reviewreport.ActorTypeAvatar
	return u.decisionNotificationRepo.List(
		ctx,
		reviewreport.DecisionNotificationFilter{
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

func (u *ReviewReportUsecase) MarkDecisionNotificationReadForAvatar(
	ctx context.Context,
	notificationID reviewreport.DecisionNotificationID,
	avatarID string,
) (reviewreport.DecisionNotification, error) {
	if err := u.ensureDecisionNotificationRepository(); err != nil {
		return reviewreport.DecisionNotification{}, err
	}
	if notificationID == "" {
		return reviewreport.DecisionNotification{}, reviewreport.ErrInvalidDecisionNotificationID
	}
	if avatarID == "" {
		return reviewreport.DecisionNotification{}, reviewreport.ErrInvalidReporterID
	}

	notification, err := u.decisionNotificationRepo.GetByID(ctx, notificationID)
	if err != nil {
		return reviewreport.DecisionNotification{}, err
	}
	if notification.RecipientType != reviewreport.ActorTypeAvatar ||
		notification.RecipientID != avatarID {
		return reviewreport.DecisionNotification{}, ErrReviewReportForbidden
	}

	return u.decisionNotificationRepo.MarkRead(
		ctx,
		notificationID,
		reviewreport.ActorTypeAvatar,
		avatarID,
		u.now().UTC(),
	)
}

func (u *ReviewReportUsecase) ListDecisionNotificationsForCompany(
	ctx context.Context,
	companyID string,
	isRead *bool,
	page common.Page,
) (common.PageResult[reviewreport.DecisionNotification], error) {
	if err := u.ensureDecisionNotificationRepository(); err != nil {
		return common.PageResult[reviewreport.DecisionNotification]{}, err
	}
	if companyID == "" {
		return common.PageResult[reviewreport.DecisionNotification]{}, reviewreport.ErrInvalidCompanyID
	}

	recipientType := reviewreport.ActorTypeBrand
	return u.decisionNotificationRepo.List(
		ctx,
		reviewreport.DecisionNotificationFilter{
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

func (u *ReviewReportUsecase) MarkDecisionNotificationReadForCompany(
	ctx context.Context,
	notificationID reviewreport.DecisionNotificationID,
	companyID string,
) (reviewreport.DecisionNotification, error) {
	if err := u.ensureDecisionNotificationRepository(); err != nil {
		return reviewreport.DecisionNotification{}, err
	}
	if notificationID == "" {
		return reviewreport.DecisionNotification{}, reviewreport.ErrInvalidDecisionNotificationID
	}
	if companyID == "" {
		return reviewreport.DecisionNotification{}, reviewreport.ErrInvalidCompanyID
	}

	notification, err := u.decisionNotificationRepo.GetByID(ctx, notificationID)
	if err != nil {
		return reviewreport.DecisionNotification{}, err
	}
	if notification.RecipientType != reviewreport.ActorTypeBrand ||
		notification.CompanyID != companyID {
		return reviewreport.DecisionNotification{}, ErrReviewReportForbidden
	}

	return u.decisionNotificationRepo.MarkRead(
		ctx,
		notificationID,
		reviewreport.ActorTypeBrand,
		notification.RecipientID,
		u.now().UTC(),
	)
}

// ============================================================
// Admin decision
// ============================================================

type ReviewReportDecision string

const (
	ReviewReportDecisionKeep   ReviewReportDecision = "KEEP"
	ReviewReportDecisionRemove ReviewReportDecision = "REMOVE"
)

type DecideReviewReportCaseInput struct {
	CaseID    reviewreport.CaseID
	Decision  ReviewReportDecision
	Reason    string
	DecidedBy string
}

func (u *ReviewReportUsecase) DecideReportCase(
	ctx context.Context,
	input DecideReviewReportCaseInput,
) (reviewreport.ReportCase, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reviewreport.ReportCase{}, err
	}
	if input.CaseID == "" {
		return reviewreport.ReportCase{}, reviewreport.ErrInvalidCaseID
	}

	switch input.Decision {
	case ReviewReportDecisionKeep, ReviewReportDecisionRemove:
		if err := u.ensureDecisionNotificationRepository(); err != nil {
			return reviewreport.ReportCase{}, err
		}
	default:
		return reviewreport.ReportCase{}, ErrReviewReportInvalidDecision
	}

	var (
		decidedCase reviewreport.ReportCase
		err         error
	)

	switch input.Decision {
	case ReviewReportDecisionKeep:
		decidedCase, err = u.keepReportCase(ctx, input)
	case ReviewReportDecisionRemove:
		decidedCase, err = u.removeReportCase(ctx, input)
	}
	if err != nil {
		return reviewreport.ReportCase{}, err
	}

	if decidedCase.Status == reviewreport.CaseStatusKept ||
		decidedCase.Status == reviewreport.CaseStatusRemoved {
		if err := u.createDecisionNotifications(ctx, decidedCase); err != nil {
			return reviewreport.ReportCase{}, err
		}
	}

	return decidedCase, nil
}

func (u *ReviewReportUsecase) keepReportCase(
	ctx context.Context,
	input DecideReviewReportCaseInput,
) (reviewreport.ReportCase, error) {
	reportCase, err := u.reportRepo.GetCase(ctx, input.CaseID)
	if err != nil {
		return reviewreport.ReportCase{}, err
	}

	if err := reportCase.Keep(
		input.Reason,
		u.now().UTC(),
		input.DecidedBy,
	); err != nil {
		return reviewreport.ReportCase{}, err
	}

	return u.reportRepo.UpdateCase(
		ctx,
		reportCase.ID,
		reviewreport.NewCasePatchFromEntity(reportCase),
	)
}

func (u *ReviewReportUsecase) removeReportCase(
	ctx context.Context,
	input DecideReviewReportCaseInput,
) (reviewreport.ReportCase, error) {
	reportCase, err := u.reportRepo.GetCase(ctx, input.CaseID)
	if err != nil {
		return reviewreport.ReportCase{}, err
	}

	// REMOVED 済みの AVATAR 裁定は、再販停止処理だけを再実行できるようにする。
	// SuspendAvatarResaleByAdmin は冪等なので、前回の裁定直後に競合して残った
	// listing や、前回の後処理失敗を安全に修復できる。
	if reportCase.IsRemoved() {
		if reportCase.TargetType == reviewreport.TargetTypeAvatar {
			if err := u.suspendAvatarResaleTarget(
				ctx,
				reportCase,
				input.Reason,
				input.DecidedBy,
			); err != nil {
				return reviewreport.ReportCase{}, err
			}
		}
		return reportCase, nil
	}

	// IMPORTANT:
	// REMOVE の対象側処理を先に完了する。
	// 商品レビュー/コメントは削除、AVATARは再販サービスのみ利用停止とする。
	// 対象側処理に失敗した場合、ReportCase を REMOVED にしてはいけない。
	switch reportCase.TargetType {
	case reviewreport.TargetTypeProductBlueprintReview:
		if err := u.removeProductBlueprintReviewTarget(
			ctx,
			reportCase,
			input.Reason,
			input.DecidedBy,
		); err != nil {
			return reviewreport.ReportCase{}, err
		}

	case reviewreport.TargetTypeTokenBlueprintComment:
		if err := u.removeTokenBlueprintCommentTarget(
			ctx,
			reportCase,
		); err != nil {
			return reviewreport.ReportCase{}, err
		}

	case reviewreport.TargetTypeAvatar:
		if err := u.suspendAvatarResaleTarget(
			ctx,
			reportCase,
			input.Reason,
			input.DecidedBy,
		); err != nil {
			return reviewreport.ReportCase{}, err
		}

	default:
		return reviewreport.ReportCase{}, reviewreport.ErrInvalidTargetType
	}

	if err := reportCase.Remove(
		input.Reason,
		u.now().UTC(),
		input.DecidedBy,
	); err != nil {
		return reviewreport.ReportCase{}, err
	}

	updatedCase, err := u.reportRepo.UpdateCase(
		ctx,
		reportCase.ID,
		reviewreport.NewCasePatchFromEntity(reportCase),
	)
	if err != nil {
		return reviewreport.ReportCase{}, err
	}

	// AVATAR は REMOVED を永続化した後でもう一度スイープする。
	// 1 回目のスイープと REMOVED 永続化の間に新規 listing が作成される
	// 競合を閉じるための後処理。失敗した場合はエラーを返し、次回の同じ
	// REMOVE 裁定で上の REMOVED 済み分岐から再試行できる。
	if updatedCase.TargetType == reviewreport.TargetTypeAvatar {
		if err := u.suspendAvatarResaleTarget(
			ctx,
			updatedCase,
			input.Reason,
			input.DecidedBy,
		); err != nil {
			return reviewreport.ReportCase{}, err
		}
	}

	return updatedCase, nil
}

func (u *ReviewReportUsecase) createDecisionNotifications(
	ctx context.Context,
	reportCase reviewreport.ReportCase,
) error {
	if err := u.ensureReportRepository(); err != nil {
		return err
	}
	if err := u.ensureDecisionNotificationRepository(); err != nil {
		return err
	}
	if reportCase.DecidedAt == nil || reportCase.DecidedAt.IsZero() {
		return reviewreport.ErrDecisionNotificationCaseNotDecided
	}

	decidedAt := reportCase.DecidedAt.UTC()
	filter := reviewreport.ReportFilter{
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
			notification, err := reviewreport.NewDecisionNotification(
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

func (u *ReviewReportUsecase) createTargetEnforcementDecisionNotification(
	ctx context.Context,
	reportCase reviewreport.ReportCase,
	createdAt time.Time,
) error {
	if reportCase.Status != reviewreport.CaseStatusRemoved {
		return nil
	}

	// 裁定対象者への通知は、実際に対象者へ措置が発生するケースだけ生成する。
	// PRODUCT_BLUEPRINT_REVIEW はレビュー削除、AVATAR は再販サービス利用停止。
	// TOKEN_BLUEPRINT_COMMENT は現時点では対象者通知の対象外とする。
	switch reportCase.TargetType {
	case reviewreport.TargetTypeProductBlueprintReview,
		reviewreport.TargetTypeAvatar:
	default:
		return nil
	}

	notification, err := reviewreport.NewTargetEnforcementNotification(
		reportCase,
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

func (u *ReviewReportUsecase) removeProductBlueprintReviewTarget(
	ctx context.Context,
	reportCase reviewreport.ReportCase,
	reason string,
	adminID string,
) error {
	if u.productReviewModerator == nil {
		return ErrReviewReportUsecaseNotConfigured
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

func (u *ReviewReportUsecase) removeTokenBlueprintCommentTarget(
	ctx context.Context,
	reportCase reviewreport.ReportCase,
) error {
	if u.tokenCommentModerator == nil {
		return ErrReviewReportUsecaseNotConfigured
	}

	return u.tokenCommentModerator.RemoveCommentByAdmin(
		ctx,
		RemoveCommentByAdminInput{
			TokenBlueprintID: reportCase.TargetParentID,
			CommentID:        reportCase.TargetID,
		},
	)
}

func (u *ReviewReportUsecase) suspendAvatarResaleTarget(
	ctx context.Context,
	reportCase reviewreport.ReportCase,
	reason string,
	adminID string,
) error {
	if u.avatarResaleModerator == nil {
		return ErrReviewReportUsecaseNotConfigured
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
