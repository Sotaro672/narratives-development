// backend/internal/application/usecase/report_usecase.go
package usecase

import (
	"errors"
	"time"

	applicationport "narratives/internal/application/port"
	announcementdom "narratives/internal/domain/announcement"
	avatar "narratives/internal/domain/avatar"
	listdom "narratives/internal/domain/list"
	pbr "narratives/internal/domain/productBlueprintReview"
	reportdom "narratives/internal/domain/report"
	resaledom "narratives/internal/domain/resale"
	tokenblueprint "narratives/internal/domain/tokenBlueprint"
	tokenreview "narratives/internal/domain/tokenBlueprint_review"
	tradedom "narratives/internal/domain/trade"
)

var (
	ErrReportUsecaseNotConfigured = errors.New("report_usecase: not configured")
	ErrReportForbidden            = errors.New("report_usecase: forbidden")
	ErrReportSelfReport           = errors.New("report_usecase: self report is not allowed")
	ErrReportInvalidDecision      = errors.New("report_usecase: invalid decision")
)

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

	resaleRepo      resaledom.Repository
	resaleModerator ReportResaleModerator

	tradeRepo        tradedom.Repository
	tradeMessageRepo tradedom.MessageRepository

	announcementRepo      announcementdom.Repository
	announcementModerator ReportAnnouncementModerator

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

	ResaleRepo      resaledom.Repository
	ResaleModerator ReportResaleModerator

	TradeRepo        tradedom.Repository
	TradeMessageRepo tradedom.MessageRepository

	AnnouncementRepo      announcementdom.Repository
	AnnouncementModerator ReportAnnouncementModerator

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
		resaleRepo:               deps.ResaleRepo,
		resaleModerator:          deps.ResaleModerator,
		tradeRepo:                deps.TradeRepo,
		tradeMessageRepo:         deps.TradeMessageRepo,
		announcementRepo:         deps.AnnouncementRepo,
		announcementModerator:    deps.AnnouncementModerator,
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
