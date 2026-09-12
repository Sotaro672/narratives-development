// backend/internal/platform/di/admin/container_repos.go
package admin

import (
	"cloud.google.com/go/firestore"

	fsrepo "narratives/internal/adapters/out/firestore"
)

type repos struct {
	contactRepo                    *fsrepo.ContactRepositoryFS
	companyRepo                    *fsrepo.CompanyRepositoryFS
	memberRepo                     *fsrepo.MemberRepositoryFS
	userRepo                       *fsrepo.UserRepositoryFS
	avatarRepo                     *fsrepo.AvatarRepositoryFS
	brandRepo                      *fsrepo.BrandRepositoryFS
	announcementRepo               *fsrepo.AnnouncementRepositoryFS
	announcementAttachmentRepo     *fsrepo.AnnouncementAttachmentRepositoryFS
	productBlueprintRepo           *fsrepo.ProductBlueprintRepositoryFS
	productBlueprintReviewRepo     *fsrepo.ProductBlueprintReviewRepositoryFS
	tokenBlueprintRepo             *fsrepo.TokenBlueprintRepositoryFS
	tokenBlueprintReviewRepo       *fsrepo.TokenBlueprintReviewRepositoryFS
	productionRepo                 *fsrepo.ProductionRepositoryFS
	productRepo                    *fsrepo.ProductRepositoryFS
	listRepo                       *fsrepo.ListRepositoryFS
	listImageRepo                  *fsrepo.ListImageRepositoryFS
	inventoryRepo                  *fsrepo.InventoryRepositoryFS
	modelRepo                      *fsrepo.ModelRepositoryFS
	orderConsoleLister             *fsrepo.OrderConsoleListerFS
	mintRepo                       *fsrepo.MintRepositoryFS
	newsRepo                       *fsrepo.NewsRepositoryFS
	newsReadRepo                   *fsrepo.NewsReadRepositoryFS
	reportRepo                     *fsrepo.ReportRepositoryFS
	reportDecisionNotificationRepo *fsrepo.ReportDecisionNotificationRepositoryFS
	resaleRepo                     *fsrepo.ResaleRepositoryFS
	resaleImageRepo                *fsrepo.ResaleImageRepositoryFS
	resaleReviewRepo               *fsrepo.ResaleReviewRepositoryFS
	resaleTradeReader              *fsrepo.ResaleTradeReaderFS
	tradeMessageRepo               *fsrepo.TradeMessageRepositoryFS
	tradeMessageStatsReader        *fsrepo.TradeMessageStatsReaderFS
	cartRepo                       *fsrepo.CartRepositoryFS
}

func buildRepos(fsClient *firestore.Client) *repos {
	if fsClient == nil {
		return nil
	}

	return &repos{
		contactRepo:                    fsrepo.NewContactRepositoryFS(fsClient),
		companyRepo:                    fsrepo.NewCompanyRepositoryFS(fsClient),
		memberRepo:                     fsrepo.NewMemberRepositoryFS(fsClient),
		userRepo:                       fsrepo.NewUserRepositoryFS(fsClient),
		avatarRepo:                     fsrepo.NewAvatarRepositoryFS(fsClient),
		brandRepo:                      fsrepo.NewBrandRepositoryFS(fsClient),
		announcementRepo:               fsrepo.NewAnnouncementRepositoryFS(fsClient),
		announcementAttachmentRepo:     fsrepo.NewAnnouncementAttachmentRepositoryFS(fsClient),
		productBlueprintRepo:           fsrepo.NewProductBlueprintRepositoryFS(fsClient),
		productBlueprintReviewRepo:     fsrepo.NewProductBlueprintReviewRepositoryFS(fsClient),
		tokenBlueprintRepo:             fsrepo.NewTokenBlueprintRepositoryFS(fsClient),
		tokenBlueprintReviewRepo:       fsrepo.NewTokenBlueprintReviewRepositoryFS(fsClient),
		productionRepo:                 fsrepo.NewProductionRepositoryFS(fsClient),
		productRepo:                    fsrepo.NewProductRepositoryFS(fsClient),
		listRepo:                       fsrepo.NewListRepositoryFS(fsClient),
		listImageRepo:                  fsrepo.NewListImageRepositoryFS(fsClient),
		inventoryRepo:                  fsrepo.NewInventoryRepositoryFS(fsClient),
		modelRepo:                      fsrepo.NewModelRepositoryFS(fsClient),
		orderConsoleLister:             fsrepo.NewOrderConsoleListerFS(fsClient),
		mintRepo:                       fsrepo.NewMintRepositoryFS(fsClient),
		newsRepo:                       fsrepo.NewNewsRepositoryFS(fsClient),
		newsReadRepo:                   fsrepo.NewNewsReadRepositoryFS(fsClient),
		reportRepo:                     fsrepo.NewReportRepositoryFS(fsClient),
		reportDecisionNotificationRepo: fsrepo.NewReportDecisionNotificationRepositoryFS(fsClient),
		resaleRepo:                     fsrepo.NewResaleRepositoryFS(fsClient),
		resaleImageRepo:                fsrepo.NewResaleImageRepositoryFS(fsClient),
		resaleReviewRepo:               fsrepo.NewResaleReviewRepositoryFS(fsClient),
		resaleTradeReader:              fsrepo.NewResaleTradeReaderFS(fsClient),
		tradeMessageRepo:               fsrepo.NewTradeMessageRepositoryFS(fsClient),
		tradeMessageStatsReader:        fsrepo.NewTradeMessageStatsReaderFS(fsClient),
		cartRepo:                       fsrepo.NewCartRepositoryFS(fsClient),
	}
}

func (r *repos) applyToContainer(c *Container) {
	if r == nil || c == nil {
		return
	}

	c.companyRepo = r.companyRepo
	c.memberRepo = r.memberRepo
	c.userRepo = r.userRepo
	c.avatarRepo = r.avatarRepo
	c.brandRepo = r.brandRepo
	c.announcementRepo = r.announcementRepo
	c.productBlueprintRepo = r.productBlueprintRepo
	c.tokenBlueprintRepo = r.tokenBlueprintRepo
	c.newsRepo = r.newsRepo
	c.newsReadRepo = r.newsReadRepo
	c.reportRepo = r.reportRepo
	c.resaleRepo = r.resaleRepo
	c.resaleImageRepo = r.resaleImageRepo
	c.resaleReviewRepo = r.resaleReviewRepo
	c.resaleTradeReader = r.resaleTradeReader
	c.tradeMessageRepo = r.tradeMessageRepo
	c.tradeMessageStatsReader = r.tradeMessageStatsReader
	c.reportDecisionNotificationRepo = r.reportDecisionNotificationRepo
}
