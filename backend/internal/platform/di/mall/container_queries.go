// backend/internal/platform/di/mall/container_queries.go
package mall

import (
	"errors"

	sharedfs "narratives/internal/adapters/out/firestore/shared"
	outsolana "narratives/internal/adapters/out/solana"
	mallquery "narratives/internal/application/query/mall"
	mallshared "narratives/internal/application/query/mall/shared"
	sharedquery "narratives/internal/application/query/shared"
	appresolver "narratives/internal/application/resolver"
	solana "narratives/internal/infra/solana"
	shared "narratives/internal/platform/di/shared"
)

type mallQueries struct {
	nameResolver  *appresolver.NameResolver
	ownerResolveQ *sharedquery.OwnerResolveQuery

	brandQ        *mallquery.BrandQuery
	listQ         *mallquery.ListQuery
	catalogQ      *mallquery.CatalogQuery
	cartQ         *mallquery.CartQuery
	previewQ      *mallquery.PreviewQuery
	inquiryQ      *mallquery.InquiryQuery
	announcementQ *mallquery.AnnouncementQueryService
	resaleQ       *mallquery.ResaleQuery
	marketQ       *mallquery.MarketQuery
	orderQ        *mallquery.OrderQuery
	historyQ      *mallquery.HistoryQuery
	orderDetailQ  *mallquery.OrderDetailQuery
	tradeQ        *mallquery.TradeQuery
}

func buildMallQueries(
	infra *shared.Infra,
	r *mallRepositories,
	u *mallUsecases,
) (*mallQueries, error) {
	if infra == nil {
		return nil, errors.New("di.mall: shared infra is nil")
	}
	if infra.Firestore == nil {
		return nil, errors.New("di.mall: firestore client is nil")
	}
	if r == nil {
		return nil, errors.New("di.mall: repositories are nil")
	}
	if u == nil {
		return nil, errors.New("di.mall: usecases are nil")
	}
	if u.paymentUC == nil {
		return nil, errors.New("di.mall: payment usecase is nil")
	}
	if r.tradeRepo == nil {
		return nil, errors.New("di.mall: trade repository is nil")
	}
	if r.tradeMessageRepo == nil {
		return nil, errors.New("di.mall: trade message repository is nil")
	}
	if r.orderRepo == nil {
		return nil, errors.New("di.mall: order repository is nil")
	}

	mallDisplayResolver := mallshared.NewDisplayResolver(
		r.productRepo,
		r.modelRepoFS,
		r.productBlueprintRepoFS,
		r.tokenBlueprintRepo,
		r.brandRepo,
	)

	nameResolver := appresolver.NewNameResolver(
		r.brandRepo,
		r.companyRepo,
		r.productBlueprintRepoFS,
		r.memberRepo,
		r.userRepo,
		r.modelRepoFS,
		r.tokenBlueprintRepo,
	)

	brandReader := sharedfs.NewBrandWalletAddressReaderFS(
		infra.Firestore,
		infra.BrandsCollection,
	)

	avatarReader := sharedfs.NewAvatarWalletAddressReaderFS(
		infra.Firestore,
		infra.AvatarsCollection,
	)

	ownerResolveQ := sharedquery.NewOwnerResolveQuery(
		avatarReader,
		brandReader,
		r.avatarRepo,
		r.brandRepo,
	)

	inquiryQ := mallquery.NewInquiryQuery(
		r.inquiryRepo,
		r.inquiryReplyRepo,
		mallDisplayResolver,
		r.orderRepo,
		r.avatarRepo,
	)

	announcementQ := mallquery.NewAnnouncementQueryService(
		r.announcementRepo,
		r.announcementAvatarRepo,
		r.announcementAttachmentRepo,
		r.tokenBlueprintRepo,
	)

	resaleQ := mallquery.NewResaleQuery(
		r.resaleRepo,
		r.resaleImageRepo,
		r.resaleReviewRepo,
		mallDisplayResolver,
		r.avatarRepo,
	)

	marketQ := mallquery.NewMarketQuery(
		r.resaleRepo,
		r.resaleImageRepo,
		mallDisplayResolver,
		r.avatarRepo,
	)

	listQ := mallquery.NewListQuery(
		r.listRepoFS,
		r.listImageRecordRepo,
	)

	brandQ := mallquery.NewBrandQuery(
		r.brandRepo,
		r.companyRepo,
		r.productBlueprintRepoFS,
		r.inventoryRepo,
		r.listRepoFS,
	)

	catalogQ := mallquery.NewCatalogQuery(
		r.listRepoFS,
		r.inventoryRepo,
		r.productBlueprintRepoFS,
		r.modelRepoFS,
		r.listImageRecordRepo,
		r.tokenBlueprintRepo,
		r.productBlueprintReviewRepo,
		nameResolver,
	)

	cartQ := mallquery.NewCartQuery(
		r.cartRepo,
		r.listRepoFS,
		r.listImageRecordRepo,
		r.inventoryRepo,
		r.productBlueprintRepoFS,
		r.resaleRepo,
		r.resaleImageRepo,
		mallDisplayResolver,
		mallquery.WithCartQueryBrandRepo(
			r.brandRepo,
		),
	)

	solanaTransferReader := solana.NewTokenTransferReaderSolana("")
	previewTransferReader := outsolana.NewPreviewTransferReader(
		solanaTransferReader,
	)

	previewQ := mallquery.NewPreviewQuery(
		r.productRepo,
		r.productBlueprintRepoFS,
		r.orderTransferItemRepo,
		nameResolver,
		r.tokenReader,
		r.tokenBlueprintRepo,
		ownerResolveQ,
		r.brandRepo,
		r.avatarRepo,
		previewTransferReader,
	)

	orderQ := mallquery.NewOrderQuery(
		r.avatarRepo,
		r.cartRepo,
		r.shippingAddressRepo,
		r.paymentMethodRepo,
		r.productBlueprintRepoFS,
		r.resaleRepo,
		r.resaleImageRepo,
		nameResolver,
	)

	historyQ := mallquery.NewHistoryQuery(
		r.inventoryRepo,
		mallDisplayResolver,
	)

	orderDetailQ := mallquery.NewOrderDetailQuery(
		r.inventoryRepo,
		mallDisplayResolver,
		u.paymentUC,
	)

	tradeQ := mallquery.NewTradeQuery(
		r.tradeRepo,
		r.tradeMessageRepo,
		r.orderRepo,
		r.resaleRepo,
		r.resaleImageRepo,
		mallDisplayResolver,
		r.avatarRepo,
	)

	return &mallQueries{
		nameResolver:  nameResolver,
		ownerResolveQ: ownerResolveQ,

		brandQ:        brandQ,
		listQ:         listQ,
		catalogQ:      catalogQ,
		cartQ:         cartQ,
		previewQ:      previewQ,
		inquiryQ:      inquiryQ,
		announcementQ: announcementQ,
		resaleQ:       resaleQ,
		marketQ:       marketQ,
		orderQ:        orderQ,
		historyQ:      historyQ,
		orderDetailQ:  orderDetailQ,
		tradeQ:        tradeQ,
	}, nil
}

func (q *mallQueries) applyToContainer(c *Container) {
	if q == nil || c == nil {
		return
	}

	c.NameResolver = q.nameResolver
	c.OwnerResolveQ = q.ownerResolveQ

	c.BrandQ = q.brandQ
	c.ListQ = q.listQ
	c.CatalogQ = q.catalogQ
	c.CartQ = q.cartQ
	c.PreviewQ = q.previewQ
	c.InquiryQ = q.inquiryQ
	c.AnnouncementQ = q.announcementQ
	c.ResaleQ = q.resaleQ
	c.MarketQ = q.marketQ
	c.OrderQ = q.orderQ
	c.HistoryQ = q.historyQ
	c.OrderDetailQ = q.orderDetailQ
	c.TradeQ = q.tradeQ
}
