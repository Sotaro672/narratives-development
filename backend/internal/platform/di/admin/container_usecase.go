// backend/internal/platform/di/admin/container_usecase.go
package admin

import (
	"context"
	"errors"
	"time"

	firebaseadp "narratives/internal/adapters/out/firebase"
	usecase "narratives/internal/application/usecase"
)

type usecases struct {
	contactUsecase *usecase.ContactUsecase
	newsUsecase    *usecase.NewsUsecase
	reportUsecase  *usecase.ReportUsecase

	productBlueprintReviewUsecase *usecase.ProductBlueprintReviewUsecase
	tokenBlueprintReviewUsecase   *usecase.TokenBlueprintReviewUsecase
	resaleReviewUsecase           *usecase.ResaleReviewUsecase

	newsImageStorage              *firebaseadp.NewsImageStorage
	announcementAttachmentStorage *firebaseadp.AnnouncementAttachmentStorage
}

func buildUsecases(ctx context.Context, r *repos) (*usecases, error) {
	if r == nil {
		return nil, errors.New("di.admin: repositories are nil")
	}

	if r.newsRepo == nil {
		return nil, errors.New("di.admin: news repository is nil")
	}
	if r.newsReadRepo == nil {
		return nil, errors.New("di.admin: news read repository is nil")
	}
	if r.reportRepo == nil {
		return nil, errors.New("di.admin: report repository is nil")
	}
	if r.productBlueprintReviewRepo == nil {
		return nil, errors.New("di.admin: product blueprint review repository is nil")
	}
	if r.reportDecisionNotificationRepo == nil {
		return nil, errors.New("di.admin: report decision notification repository is nil")
	}
	if r.tokenBlueprintReviewRepo == nil {
		return nil, errors.New("di.admin: token blueprint review repository is nil")
	}
	if r.resaleRepo == nil {
		return nil, errors.New("di.admin: resale repository is nil")
	}
	if r.resaleReviewRepo == nil {
		return nil, errors.New("di.admin: resale review repository is nil")
	}
	if r.cartRepo == nil {
		return nil, errors.New("di.admin: cart repository is nil")
	}
	if r.announcementRepo == nil {
		return nil, errors.New("di.admin: announcement repository is nil")
	}
	if r.announcementAttachmentRepo == nil {
		return nil, errors.New("di.admin: announcement attachment repository is nil")
	}

	contactUsecase := usecase.NewContactUsecase(
		r.contactRepo,
		nil,
		nil,
	)

	newsImageStorage, err := firebaseadp.NewNewsImageStorageFromEnv(ctx)
	if err != nil {
		return nil, err
	}
	if newsImageStorage == nil {
		return nil, errors.New("di.admin: news image storage is nil")
	}

	announcementAttachmentStorage, err := firebaseadp.NewAnnouncementAttachmentStorageFromEnv(ctx)
	if err != nil {
		_ = newsImageStorage.Close()
		return nil, err
	}
	if announcementAttachmentStorage == nil {
		_ = newsImageStorage.Close()
		return nil, errors.New("di.admin: announcement attachment storage is nil")
	}

	closeStorages := func(err error) (*usecases, error) {
		_ = announcementAttachmentStorage.Close()
		_ = newsImageStorage.Close()
		return nil, err
	}

	newsUsecase := usecase.NewNewsUsecase(
		r.newsRepo,
		r.newsReadRepo,
	).WithImageStorage(newsImageStorage)
	if newsUsecase == nil {
		return closeStorages(errors.New("di.admin: news usecase is nil"))
	}

	productBlueprintReviewUsecase := usecase.NewProductBlueprintReviewUsecase(
		r.productBlueprintReviewRepo,
		r.productBlueprintRepo,
		r.brandRepo,
		r.memberRepo,
		nil,
		r.avatarRepo,
		nil,
	)
	if productBlueprintReviewUsecase == nil {
		return closeStorages(errors.New("di.admin: product blueprint review usecase is nil"))
	}

	tokenBlueprintReviewUsecase := usecase.NewTokenBlueprintReviewUsecase(
		r.tokenBlueprintReviewRepo,
		r.avatarRepo,
		r.tokenBlueprintRepo,
		r.brandRepo,
	)
	if tokenBlueprintReviewUsecase == nil {
		return closeStorages(errors.New("di.admin: token blueprint review usecase is nil"))
	}

	resaleReviewUsecase := usecase.NewResaleReviewUsecase(
		r.resaleRepo,
		r.resaleReviewRepo,
		r.avatarRepo,
		time.Now,
	)
	if resaleReviewUsecase == nil {
		return closeStorages(errors.New("di.admin: resale review usecase is nil"))
	}

	// Admin側のTokenBlueprintUsecaseは通報裁定によるAMOL上の非表示専用。
	// TokenBlueprint本体、Firebase Storage、metadataUri、
	// オンチェーン上のトークン・メタデータは削除しない。
	tokenBlueprintUsecase := usecase.NewTokenBlueprintUsecase(
		r.tokenBlueprintRepo,
		nil,
		nil,
		nil,
	)
	if tokenBlueprintUsecase == nil {
		return closeStorages(errors.New("di.admin: token blueprint usecase is nil"))
	}

	// Admin側のListUsecaseは通報裁定によるMall上の出品停止専用。
	// List本体、ListImage、Firebase Storage上の画像は削除せず、
	// statusをsuspendedへ変更し、既存カートから対象Listを除去する。
	listUsecase := usecase.NewListUsecase(
		r.listRepo,
		nil,
		nil,
	).WithCartItemCleanup(r.cartRepo)
	if listUsecase == nil {
		return closeStorages(errors.New("di.admin: list usecase is nil"))
	}

	// Admin側のResaleUsecaseはアバター通報および個別Resale通報の裁定専用。
	// 出品作成・画像操作は行わないため、imageRepo / imageStorage /
	// product identity repositories は不要。
	// 個別ResaleのREMOVEでは対象Resaleをsuspendedへ変更し、既存カートから除去する。
	resaleUsecase := usecase.NewResaleUsecase(
		r.resaleRepo,
		nil,
		nil,
		time.Now,
	).WithCartItemCleanup(r.cartRepo)
	if resaleUsecase == nil {
		return closeStorages(errors.New("di.admin: resale usecase is nil"))
	}

	// Admin側のAnnouncementUsecaseは通報裁定によるAnnouncement物理削除専用。
	// Consoleの通常削除とは異なり、公開済みAnnouncementも
	// DeleteAnnouncementByAdminによって削除できる。
	announcementUsecase := usecase.NewAnnouncementUsecase(
		r.announcementRepo,
		nil,
		r.announcementAttachmentRepo,
	).WithAttachmentStorage(announcementAttachmentStorage)
	if announcementUsecase == nil {
		return closeStorages(errors.New("di.admin: announcement usecase is nil"))
	}

	reportUsecase := usecase.NewReportUsecase(
		usecase.ReportUsecaseDeps{
			ReportRepo:               r.reportRepo,
			DecisionNotificationRepo: r.reportDecisionNotificationRepo,
			ProductBlueprintRepo:     r.productBlueprintRepo,
			ProductReviewModerator:   productBlueprintReviewUsecase,

			ListRepo:      r.listRepo,
			InventoryRepo: r.inventoryRepo,
			ListModerator: listUsecase,

			TokenBlueprintRepo:      r.tokenBlueprintRepo,
			TokenBlueprintModerator: tokenBlueprintUsecase,
			TokenCommentModerator:   tokenBlueprintReviewUsecase,

			AvatarRepo:            r.avatarRepo,
			AvatarResaleModerator: resaleUsecase,

			ResaleRepo:      r.resaleRepo,
			ResaleModerator: resaleUsecase,

			AnnouncementRepo:      r.announcementRepo,
			AnnouncementModerator: announcementUsecase,
		},
	)
	if reportUsecase == nil {
		return closeStorages(errors.New("di.admin: report usecase is nil"))
	}

	return &usecases{
		contactUsecase:                contactUsecase,
		newsUsecase:                   newsUsecase,
		reportUsecase:                 reportUsecase,
		productBlueprintReviewUsecase: productBlueprintReviewUsecase,
		tokenBlueprintReviewUsecase:   tokenBlueprintReviewUsecase,
		resaleReviewUsecase:           resaleReviewUsecase,
		newsImageStorage:              newsImageStorage,
		announcementAttachmentStorage: announcementAttachmentStorage,
	}, nil
}

func (u *usecases) applyToContainer(c *Container) {
	if u == nil || c == nil {
		return
	}

	c.contactUsecase = u.contactUsecase
	c.newsUsecase = u.newsUsecase
	c.reportUsecase = u.reportUsecase
	c.resaleReviewUsecase = u.resaleReviewUsecase
}

func (u *usecases) closeOnBuildError() {
	if u == nil {
		return
	}

	if u.announcementAttachmentStorage != nil {
		_ = u.announcementAttachmentStorage.Close()
	}
	if u.newsImageStorage != nil {
		_ = u.newsImageStorage.Close()
	}
}
