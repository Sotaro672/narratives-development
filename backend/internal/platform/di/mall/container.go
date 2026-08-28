// backend/internal/platform/di/mall/container.go
package mall

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	mallquery "narratives/internal/application/query/mall"
	mallshared "narratives/internal/application/query/mall/shared"
	sharedquery "narratives/internal/application/query/shared"
	appresolver "narratives/internal/application/resolver"
	usecase "narratives/internal/application/usecase"

	mallhandler "narratives/internal/adapters/in/http/mall/handler"

	cloudtasksadp "narratives/internal/adapters/out/cloudtasks"
	outfirebase "narratives/internal/adapters/out/firebase"
	outfs "narratives/internal/adapters/out/firestore"
	mallfs "narratives/internal/adapters/out/firestore/mall"
	sharedfs "narratives/internal/adapters/out/firestore/shared"
	mailadp "narratives/internal/adapters/out/mail"
	outsolana "narratives/internal/adapters/out/solana"
	stripeadapter "narratives/internal/adapters/out/stripe"

	refunddom "narratives/internal/domain/refund"
	transferdom "narratives/internal/domain/transfer"
	transportationdom "narratives/internal/domain/transportation"

	solana "narratives/internal/infra/solana"

	shared "narratives/internal/platform/di/shared"
)

const (
	StripeWebhookPath = "/mall/webhooks/stripe"

	mallAutoCreateStripeTestPaymentMethodEnv = "MALL_AUTO_CREATE_STRIPE_TEST_PAYMENT_METHOD"

	mallPayoutAccountAllowedReturnOriginEnv = "MALL_FRONTEND_BASE_URL"
)

type Container struct {
	Infra *shared.Infra

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
	RefundUC                       *usecase.RefundUsecase
	ItemRefundUC                   *usecase.ItemRefundUsecase
	RefundRepo                     refunddom.RepositoryPort
	RefundCompletionNotificationUC usecase.RefundCompletionNotificationUsecasePort
	OrderUC                        *usecase.OrderUsecase
	InquiryUC                      *usecase.InquiryUsecase
	ReturnRequestUC                *usecase.ReturnRequestUsecase
	AnnouncementUC                 *usecase.AnnouncementUsecase
	ResaleUC                       *usecase.ResaleUsecase

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

	fsClient := infra.Firestore
	if fsClient == nil {
		return nil, errors.New("di.mall: infra.Firestore is nil")
	}

	c := &Container{Infra: infra}

	authUserReader := outfirebase.NewAuthUserReader(infra.FirebaseAuth)
	avatarRepo := outfs.NewAvatarRepositoryFS(fsClient)

	c.MeAvatarResolver = avatarRepo
	c.SetupUC = usecase.NewSetupUsecase(avatarRepo)

	shippingAddressRepo := outfs.NewShippingAddressRepositoryFS(fsClient)
	paymentMethodRepo := outfs.NewPaymentMethodRepositoryFS(fsClient)
	payoutAccountRepo := outfs.NewPayoutAccountRepositoryFS(fsClient)
	userRepo := outfs.NewUserRepositoryFS(fsClient)
	memberRepo := outfs.NewMemberRepositoryFS(fsClient)
	walletRepo := outfs.NewWalletRepositoryFS(fsClient)
	productRepo := outfs.NewProductRepositoryFS(fsClient)

	{
		var customerStore stripeadapter.PaymentMethodCustomerStore = paymentMethodRepo

		if err := infra.RegisterPaymentMethodGatewayFromSecret(ctx, customerStore); err != nil {
			return nil, err
		}
		if infra.PaymentMethodGateway == nil {
			return nil, errors.New("di.mall: stripe payment method gateway is nil after registration")
		}
	}

	brandRepo := outfs.NewBrandRepositoryFS(fsClient)

	accountRepo := outfs.NewAccountRepositoryFS(fsClient)
	companyRepo := outfs.NewCompanyRepositoryFS(fsClient)
	cartRepo := outfs.NewCartRepositoryFS(fsClient)
	paymentRepo := outfs.NewPaymentRepositoryFS(fsClient)
	settlementRepo := outfs.NewSettlementRepositoryFS(fsClient)
	refundRepo := outfs.NewRefundRepositoryFS(fsClient)
	orderRepo := outfs.NewOrderRepositoryFS(fsClient)

	c.RefundRepo = refundRepo

	// The projection repository is shared by PreviewQuery and TransferUsecase.
	orderTransferItemRepo := outfs.NewOrderRepoForTransferFS(fsClient)

	inventoryRepo := outfs.NewInventoryRepositoryFS(fsClient)
	tokenBlueprintRepo := outfs.NewTokenBlueprintRepositoryFS(fsClient)
	productBlueprintRepoFS := outfs.NewProductBlueprintRepositoryFS(fsClient)
	modelRepoFS := outfs.NewModelRepositoryFS(fsClient)

	mallDisplayResolver := mallshared.NewDisplayResolver(
		productRepo,
		modelRepoFS,
		productBlueprintRepoFS,
		tokenBlueprintRepo,
		brandRepo,
	)

	inquiryRepo := outfs.NewInquiryRepositoryFS(fsClient)
	inquiryReplyRepo := outfs.NewInquiryReplyRepositoryFS(fsClient)

	c.InquiryQ = mallquery.NewInquiryQuery(
		inquiryRepo,
		inquiryReplyRepo,
		mallDisplayResolver,
		orderRepo,
		avatarRepo,
	)

	announcementRepo := outfs.NewAnnouncementRepositoryFS(fsClient)
	announcementAvatarRepo := outfs.NewAnnouncementAvatarRepositoryFS(fsClient)
	announcementAttachmentRepo := outfs.NewAnnouncementAttachmentRepositoryFS(fsClient)

	c.AnnouncementUC = usecase.NewAnnouncementUsecase(
		announcementRepo,
		announcementAvatarRepo,
		announcementAttachmentRepo,
	)

	c.AnnouncementQ = mallquery.NewAnnouncementQueryService(
		announcementRepo,
		announcementAvatarRepo,
		announcementAttachmentRepo,
		tokenBlueprintRepo,
	)

	tokenBlueprintReviewRepo := outfs.NewTokenBlueprintReviewRepositoryFS(fsClient)
	productBlueprintReviewRepo := outfs.NewProductBlueprintReviewRepositoryFS(fsClient)
	listRepoFS := outfs.NewListRepositoryFS(fsClient)
	listImageRecordRepo := outfs.NewListImageRepositoryFS(fsClient)
	resaleRepo := outfs.NewResaleRepositoryFS(fsClient)
	resaleImageRepo := outfs.NewResaleImageRepositoryFS(fsClient)

	resaleImageStorage, err := outfirebase.NewResaleImageStorageFromEnv(ctx)
	if err != nil {
		return nil, err
	}

	c.ResaleUC = usecase.NewResaleUsecase(
		resaleRepo,
		resaleImageRepo,
		resaleImageStorage,
	)

	c.ResaleQ = mallquery.NewResaleQuery(
		resaleRepo,
		resaleImageRepo,
		mallDisplayResolver,
	)

	c.MarketQ = mallquery.NewMarketQuery(
		resaleRepo,
		resaleImageRepo,
		mallDisplayResolver,
		avatarRepo,
	)

	orderMailer := mailadp.NewOrderMailer(
		mailadp.NewResendClient(os.Getenv("RESEND_API_KEY")),
		modelRepoFS,
		inventoryRepo,
		productBlueprintRepoFS,
		tokenBlueprintRepo,
		brandRepo,
		companyRepo,
	)

	orderMailFrom := os.Getenv("RESEND_FROM")

	orderCancellationMailer := mailadp.NewOrderCancellationMailer(
		mailadp.NewResendClient(os.Getenv("RESEND_API_KEY")),
		orderMailFrom,
	)

	inquiryMailer := mailadp.NewInquiryMailer(
		mailadp.NewResendClient(os.Getenv("RESEND_API_KEY")),
	)

	projectID := infra.ProjectID
	avatarWalletSvc := solana.NewAvatarWalletService(projectID)

	c.AvatarUC = usecase.NewAvatarUsecase(
		avatarRepo,
		avatarWalletSvc,
		walletRepo,
		cartRepo,
		nil,
	)

	transportationRepo := outfs.NewTransportationRepositoryFS(fsClient)
	transportationSvc := transportationdom.NewService(transportationRepo)

	c.ShippingQuoteUC = usecase.NewShippingQuoteUsecase(
		listRepoFS,
		inventoryRepo,
		modelRepoFS,
		shippingAddressRepo,
		transportationSvc,
	)

	c.ListQ = mallquery.NewListQuery(
		listRepoFS,
		listImageRecordRepo,
	)

	c.ShippingAddressUC = usecase.NewShippingAddressUsecase(shippingAddressRepo)

	c.PaymentMethodUC = usecase.NewPaymentMethodUsecase(
		paymentMethodRepo,
		infra.PaymentMethodGateway,
	)

	autoCreateDevelopmentPaymentMethod := strings.EqualFold(
		strings.TrimSpace(os.Getenv(mallAutoCreateStripeTestPaymentMethodEnv)),
		"true",
	)

	c.AvatarRegistrationUC = usecase.NewAvatarRegistrationUsecase(
		c.AvatarUC,
		c.PaymentMethodUC,
		autoCreateDevelopmentPaymentMethod,
	)

	c.UserUC = usecase.NewUserUsecase(userRepo, nil)

	onchainReader := solana.NewOnchainWalletReaderDevnet()
	tokenQuery := outfs.NewTokenReaderFS(fsClient)

	c.WalletUC = usecase.NewWalletUsecase(
		walletRepo,
		onchainReader,
		tokenQuery,
		brandRepo,
		productRepo,
		productBlueprintRepoFS,
		productBlueprintRepoFS,
	)

	c.ProductBlueprintReviewUC = usecase.NewProductBlueprintReviewUsecase(
		productBlueprintReviewRepo,
		productBlueprintRepoFS,
		brandRepo,
		memberRepo,
		c.WalletUC,
		avatarRepo,
		nil,
	)

	c.TokenBlueprintReviewUC = usecase.NewTokenBlueprintReviewUsecase(
		tokenBlueprintReviewRepo,
		avatarRepo,
		tokenBlueprintRepo,
		brandRepo,
	)

	c.CartUC = usecase.NewCartUsecase(cartRepo)

	c.PaymentUC = usecase.NewPaymentUsecase(
		usecase.NewPaymentUsecaseInput{
			PaymentRepo: paymentRepo,

			CartRepo:      cartRepo,
			OrderRepo:     orderRepo,
			InventoryRepo: inventoryRepo,
			ResaleRepo:    resaleRepo,

			AuthUserGetter: authUserReader,
			MailSender:     orderMailer,
			MailFrom:       orderMailFrom,
		},
	)

	settlementDependencies, err := shared.BuildSettlementDependencies(ctx, infra)
	if err != nil {
		return nil, fmt.Errorf("di.mall: build settlement dependencies: %w", err)
	}

	{
		payoutAccountAllowedReturnOrigin := strings.TrimSpace(
			os.Getenv(mallPayoutAccountAllowedReturnOriginEnv),
		)
		if payoutAccountAllowedReturnOrigin == "" {
			return nil, fmt.Errorf(
				"di.mall: %s is empty",
				mallPayoutAccountAllowedReturnOriginEnv,
			)
		}

		if infra.AccountGateway == nil {
			if err := infra.RegisterAccountGatewayFromSecret(ctx); err != nil {
				return nil, fmt.Errorf(
					"di.mall: register Stripe payout account gateway: %w",
					err,
				)
			}
		}
		if infra.AccountGateway == nil {
			return nil, errors.New("di.mall: Stripe payout account gateway is nil")
		}

		c.PayoutAccountUC = usecase.NewPayoutAccountUsecase(
			payoutAccountRepo,
			infra.AccountGateway,
			payoutAccountAllowedReturnOrigin,
		)
		if c.PayoutAccountUC == nil {
			return nil, errors.New("di.mall: payout account usecase is nil")
		}
	}

	c.SettlementUC = usecase.NewSettlementUsecase(
		usecase.NewSettlementUsecaseInput{
			Repository:            settlementRepo,
			Calculator:            settlementDependencies.SettlementCalculator,
			StripeTransferGateway: settlementDependencies.StripeTransferGateway,
		},
	)
	if c.SettlementUC == nil {
		return nil, errors.New("di.mall: settlement usecase is nil")
	}

	c.RefundUC = usecase.NewRefundUsecase(
		usecase.NewRefundUsecaseInput{
			PaymentReader:                 c.PaymentUC,
			SettlementRepository:          settlementRepo,
			StripeRefundGateway:           settlementDependencies.StripeRefundGateway,
			StripeTransferReversalGateway: settlementDependencies.StripeTransferReversalGateway,
		},
	)
	if c.RefundUC == nil {
		return nil, errors.New("di.mall: refund usecase is nil")
	}

	c.OrderUC = usecase.NewOrderUsecase(
		orderRepo,
		listRepoFS,
		inventoryRepo,
		productBlueprintRepoFS,
		resaleRepo,
		paymentMethodRepo,
		shippingAddressRepo,
		c.ShippingQuoteUC,
	).
		WithCartRepository(cartRepo).
		WithSellerRepositories(
			brandRepo,
			accountRepo,
		).
		WithCancellationNotification(
			authUserReader,
			orderCancellationMailer,
		)

	c.ItemRefundUC = usecase.NewItemRefundUsecase(
		usecase.NewItemRefundUsecaseInput{
			OrderReader:                   c.OrderUC,
			PaymentReader:                 c.PaymentUC,
			SettlementRepository:          settlementRepo,
			RefundRepository:              refundRepo,
			PlatformFeeCalculator:         settlementDependencies.Calculator,
			StripeRefundGateway:           settlementDependencies.StripeRefundGateway,
			StripeTransferReversalGateway: settlementDependencies.StripeTransferReversalGateway,
		},
	)
	if c.ItemRefundUC == nil {
		return nil, errors.New("di.mall: item refund usecase is nil")
	}

	c.InquiryUC = usecase.NewInquiryUsecase(
		inquiryRepo,
		inquiryReplyRepo,
		inquiryMailer,
		orderMailFrom,
		avatarRepo,
		authUserReader,
	)

	c.ReturnRequestUC = usecase.NewReturnRequestUsecase(
		c.OrderUC,
		inquiryRepo,
		c.InquiryUC,
	).WithReplyRepository(inquiryReplyRepo)

	if infra.PaymentMethodGateway == nil {
		return nil, errors.New("di.mall: stripe payment intent gateway is nil")
	}

	c.PaymentFlowUC = usecase.NewPaymentFlowUsecase(
		c.PaymentUC,
		orderRepo,
		infra.PaymentMethodGateway,
	)

	inventoryUC := usecase.NewInventoryUsecase(inventoryRepo)

	{
		c.NameResolver = appresolver.NewNameResolver(
			brandRepo,
			companyRepo,
			productBlueprintRepoFS,
			memberRepo,
			userRepo,
			modelRepoFS,
			tokenBlueprintRepo,
		)
	}

	{
		brandsCol := infra.BrandsCollection
		avatarsCol := infra.AvatarsCollection

		brandReader := sharedfs.NewBrandWalletAddressReaderFS(
			fsClient,
			brandsCol,
		)

		avatarReader := sharedfs.NewAvatarWalletAddressReaderFS(
			fsClient,
			avatarsCol,
		)

		c.OwnerResolveQ = sharedquery.NewOwnerResolveQuery(
			avatarReader,
			brandReader,
			avatarRepo,
			brandRepo,
		)
	}

	{
		c.BrandQ = mallquery.NewBrandQuery(
			brandRepo,
			companyRepo,
			productBlueprintRepoFS,
			inventoryRepo,
			listRepoFS,
		)

		c.CatalogQ = mallquery.NewCatalogQuery(
			listRepoFS,
			inventoryRepo,
			productBlueprintRepoFS,
			modelRepoFS,
			listImageRecordRepo,
			tokenBlueprintRepo,
			productBlueprintReviewRepo,
			c.NameResolver,
		)

		c.CartQ = mallquery.NewCartQuery(
			cartRepo,
			listRepoFS,
			listImageRecordRepo,
			inventoryRepo,
			productBlueprintRepoFS,
			resaleRepo,
			resaleImageRepo,
			mallDisplayResolver,
			mallquery.WithCartQueryBrandRepo(brandRepo),
		)

		tokenReader := outfs.NewTokenReaderFS(fsClient)

		solanaTransferReader := solana.NewTokenTransferReaderSolana("")
		previewTransferReader := outsolana.NewPreviewTransferReader(solanaTransferReader)

		c.PreviewQ = mallquery.NewPreviewQuery(
			productRepo,
			productBlueprintRepoFS,
			orderTransferItemRepo,
			c.NameResolver,
			tokenReader,
			tokenBlueprintRepo,
			c.OwnerResolveQ,
			brandRepo,
			avatarRepo,
			previewTransferReader,
		)

		c.OrderQ = mallquery.NewOrderQuery(
			avatarRepo,
			cartRepo,
			shippingAddressRepo,
			paymentMethodRepo,
			productBlueprintRepoFS,
			resaleRepo,
			resaleImageRepo,
			c.NameResolver,
		)

		c.HistoryQ = mallquery.NewHistoryQuery(
			inventoryRepo,
			mallDisplayResolver,
		)

		c.OrderDetailQ = mallquery.NewOrderDetailQuery(
			inventoryRepo,
			mallDisplayResolver,
			c.PaymentUC,
		)
	}

	{
		var orderRepoForTransfer usecase.OrderRepoForTransfer = orderTransferItemRepo

		var tokenResolver usecase.TokenResolver = mallfs.NewTokenResolverFS(
			fsClient,
			"tokens",
		)

		var tokenOwnerUpdater usecase.TokenOwnerUpdater = outfs.NewTokenOwnerUpdaterFS(fsClient)
		var transferRepo transferdom.RepositoryPort = outfs.NewTransferRepositoryFS(fsClient)

		walletResolverRepo := outfs.NewWalletResolverRepoFS(
			brandRepo,
			walletRepo,
		)

		var walletResolver usecase.BrandWalletResolver = walletResolverRepo
		var avatarWalletResolver usecase.AvatarWalletResolver = walletResolverRepo

		var walletTransferUpdate usecase.AvatarWalletItemTransferUpdater = walletRepo
		var walletSync usecase.AvatarWalletSyncer = c.WalletUC
		var executor usecase.TokenTransferExecutor = solana.NewTokenTransferExecutorSolana("")

		transferExecutionUC := usecase.NewTokenTransferExecutionUsecase(
			tokenOwnerUpdater,
			walletTransferUpdate,
			walletSync,
			transferRepo,
			executor,
			nil,
		)

		c.TransferUC = usecase.NewTransferUsecase(
			c.PreviewQ,
			orderRepoForTransfer,
			tokenResolver,
			walletResolver,
			avatarWalletResolver,
			brandRepo,
			avatarRepo,
			transferExecutionUC,
			inventoryUC,
		).
			WithResaleTransferDependencies(resaleRepo).
			WithReturnOpeningHandler(c.ReturnRequestUC)
	}

	// Stripe Refund webhook が succeeded を確定した後に、
	// purchaser向け返金完了通知deliveryを作成・Cloud Tasksへ投入します。
	refundCompletionNotificationRepo := outfs.NewRefundCompletionNotificationRepositoryFS(fsClient)

	refundCompletionNotificationQueue, err := cloudtasksadp.NewRefundCompletionNotificationQueueFromEnv(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"di.mall: build refund completion notification queue: %w",
			err,
		)
	}

	refundCompletionNotificationMailer := mailadp.NewRefundCompletionNotificationMailerWithResend()

	c.RefundCompletionNotificationUC = usecase.NewRefundCompletionNotificationUsecase(
		refundCompletionNotificationRepo,
		authUserReader,
		refundCompletionNotificationMailer,
		refundCompletionNotificationQueue,
	)

	return c, nil
}
