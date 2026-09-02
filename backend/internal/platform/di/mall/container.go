// backend/internal/platform/di/mall/container.go
package mall

import (
	"context"
	"errors"

	mallhandler "narratives/internal/adapters/in/http/mall/handler"

	mallquery "narratives/internal/application/query/mall"
	sharedquery "narratives/internal/application/query/shared"
	appresolver "narratives/internal/application/resolver"
	usecase "narratives/internal/application/usecase"

	refunddom "narratives/internal/domain/refund"

	shared "narratives/internal/platform/di/shared"
)

const StripeWebhookPath = "/mall/webhooks/stripe"

type Container struct {
	Infra  *shared.Infra
	config mallConfig

	AvatarUC                       *usecase.AvatarUsecase
	AvatarRegistrationUC           *usecase.AvatarRegistrationUsecase
	SetupUC                        *usecase.SetupUsecase
	ShippingAddressUC              *usecase.ShippingAddressUsecase
	ShippingQuoteUC                *usecase.ShippingQuoteUsecase
	PaymentMethodUC                *usecase.PaymentMethodUsecase
	PayoutAccountUC                *usecase.PayoutAccountUsecase
	UserUC                         *usecase.UserUsecase
	WalletUC                       *usecase.WalletUsecase
	CartUC                         *usecase.CartUsecase
	PaymentUC                      *usecase.PaymentUsecase
	SettlementUC                   *usecase.SettlementUsecase
	BrandFeeSettlementUC           *usecase.BrandFeeSettlementUsecase
	BrandFeeSettlementTransferUC   *usecase.BrandFeeSettlementTransferUsecase
	BrandFeeSettlementQueue        usecase.BrandFeeSettlementTransferQueue
	RefundUC                       *usecase.RefundUsecase
	ItemRefundUC                   *usecase.ItemRefundUsecase
	RefundRepo                     refunddom.RepositoryPort
	RefundCompletionNotificationUC usecase.RefundCompletionNotificationUsecasePort
	ResalePayoutNotificationUC     usecase.ResalePayoutNotificationUsecasePort
	OrderUC                        *usecase.OrderUsecase
	TradeUC                        *usecase.TradeUsecase
	TradeMessageUC                 *usecase.TradeMessageUsecase
	ResaleTradeDispatchUC          *usecase.ResaleTradeDispatchUsecase
	ResaleTradeReturnReceiptUC     *usecase.ResaleTradeReturnReceiptUsecase
	InquiryUC                      *usecase.InquiryUsecase
	ReturnRequestUC                *usecase.ReturnRequestUsecase
	AnnouncementUC                 *usecase.AnnouncementUsecase
	ResaleUC                       *usecase.ResaleUsecase
	ResaleReviewUC                 *usecase.ResaleReviewUsecase
	LikeUC                         *usecase.LikeUsecase

	MeAvatarResolver mallhandler.MeAvatarResolver

	ProductBlueprintReviewUC *usecase.ProductBlueprintReviewUsecase
	TokenBlueprintReviewUC   *usecase.TokenBlueprintReviewUsecase

	TransferUC    *usecase.TransferUsecase
	PaymentFlowUC *usecase.PaymentFlowUsecase

	NameResolver *appresolver.NameResolver

	BrandQ        *mallquery.BrandQuery
	ListQ         *mallquery.ListQuery
	CatalogQ      *mallquery.CatalogQuery
	CartQ         *mallquery.CartQuery
	PreviewQ      *mallquery.PreviewQuery
	InquiryQ      *mallquery.InquiryQuery
	AnnouncementQ *mallquery.AnnouncementQueryService
	ResaleQ       *mallquery.ResaleQuery
	MarketQ       *mallquery.MarketQuery
	OrderQ        *mallquery.OrderQuery
	HistoryQ      *mallquery.HistoryQuery
	OrderDetailQ  *mallquery.OrderDetailQuery
	TradeQ        *mallquery.TradeQuery

	OwnerResolveQ *sharedquery.OwnerResolveQuery
}

func NewContainer(ctx context.Context, infra *shared.Infra) (*Container, error) {
	if infra == nil {
		var err error
		infra, err = shared.NewInfra(ctx)
		if err != nil {
			return nil, err
		}
	}

	if infra == nil {
		return nil, errors.New("di.mall: shared infra is nil")
	}
	if infra.Config == nil {
		return nil, errors.New("di.mall: shared infra config is nil")
	}
	if infra.Firestore == nil {
		return nil, errors.New("di.mall: infra.Firestore is nil")
	}

	cfg := loadMallConfigFromEnv()

	c := &Container{
		Infra:  infra,
		config: cfg,
	}

	repos := buildMallRepositories(infra.Firestore)
	if repos == nil {
		return nil, errors.New("di.mall: repositories are nil")
	}

	if repos.avatarRepo == nil {
		return nil, errors.New("di.mall: avatar repository is nil")
	}
	if repos.refundRepo == nil {
		return nil, errors.New("di.mall: refund repository is nil")
	}

	c.MeAvatarResolver = repos.avatarRepo
	c.RefundRepo = repos.refundRepo

	usecases, err := buildMallUsecases(
		ctx,
		infra,
		cfg,
		repos,
	)
	if err != nil {
		return nil, err
	}
	if usecases == nil {
		return nil, errors.New("di.mall: usecases are nil")
	}

	usecases.applyToContainer(c)

	if usecases.brandFeeSettlementQueue == nil {
		return nil, errors.New("di.mall: brand fee settlement queue is nil")
	}
	c.BrandFeeSettlementQueue = usecases.brandFeeSettlementQueue

	queries, err := buildMallQueries(
		infra,
		repos,
		usecases,
	)
	if err != nil {
		return nil, err
	}
	if queries == nil {
		return nil, errors.New("di.mall: queries are nil")
	}

	queries.applyToContainer(c)

	transferUC, err := buildMallTransferUsecase(
		infra,
		repos,
		usecases,
		queries.previewQ,
	)
	if err != nil {
		return nil, err
	}
	if transferUC == nil {
		return nil, errors.New("di.mall: transfer usecase is nil")
	}

	c.TransferUC = transferUC

	return c, nil
}
