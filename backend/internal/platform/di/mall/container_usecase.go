// backend/internal/platform/di/mall/container_usecase.go
package mall

import (
	"context"
	"errors"
	"fmt"

	cloudtasksadp "narratives/internal/adapters/out/cloudtasks"
	outfirebase "narratives/internal/adapters/out/firebase"
	mallfs "narratives/internal/adapters/out/firestore/mall"
	mailadp "narratives/internal/adapters/out/mail"
	stripeadapter "narratives/internal/adapters/out/stripe"
	mallquery "narratives/internal/application/query/mall"
	usecase "narratives/internal/application/usecase"
	transportationdom "narratives/internal/domain/transportation"
	solana "narratives/internal/infra/solana"
	shared "narratives/internal/platform/di/shared"
)

type mallUsecases struct {
	avatarUC                       *usecase.AvatarUsecase
	avatarRegistrationUC           *usecase.AvatarRegistrationUsecase
	setupUC                        *usecase.SetupUsecase
	shippingAddressUC              *usecase.ShippingAddressUsecase
	shippingQuoteUC                *usecase.ShippingQuoteUsecase
	paymentMethodUC                *usecase.PaymentMethodUsecase
	payoutAccountUC                *usecase.PayoutAccountUsecase
	userUC                         *usecase.UserUsecase
	walletUC                       *usecase.WalletUsecase
	cartUC                         *usecase.CartUsecase
	paymentUC                      *usecase.PaymentUsecase
	settlementUC                   *usecase.SettlementUsecase
	refundUC                       *usecase.RefundUsecase
	itemRefundUC                   *usecase.ItemRefundUsecase
	refundCompletionNotificationUC usecase.RefundCompletionNotificationUsecasePort
	orderUC                        *usecase.OrderUsecase
	inquiryUC                      *usecase.InquiryUsecase
	returnRequestUC                *usecase.ReturnRequestUsecase
	announcementUC                 *usecase.AnnouncementUsecase
	resaleUC                       *usecase.ResaleUsecase
	productBlueprintReviewUC       *usecase.ProductBlueprintReviewUsecase
	tokenBlueprintReviewUC         *usecase.TokenBlueprintReviewUsecase
	paymentFlowUC                  *usecase.PaymentFlowUsecase

	// TransferUsecaseの構築時にも利用するためContainerには公開しない。
	inventoryUC *usecase.InventoryUsecase
}

func buildMallUsecases(
	ctx context.Context,
	infra *shared.Infra,
	cfg mallConfig,
	r *mallRepositories,
) (*mallUsecases, error) {
	if infra == nil {
		return nil, errors.New("di.mall: shared infra is nil")
	}
	if infra.Firestore == nil {
		return nil, errors.New("di.mall: firestore client is nil")
	}
	if r == nil {
		return nil, errors.New("di.mall: repositories are nil")
	}

	authUserReader := outfirebase.NewAuthUserReader(infra.FirebaseAuth)

	var customerStore stripeadapter.PaymentMethodCustomerStore = r.paymentMethodRepo
	if err := infra.RegisterPaymentMethodGatewayFromSecret(ctx, customerStore); err != nil {
		return nil, err
	}
	if infra.PaymentMethodGateway == nil {
		return nil, errors.New("di.mall: stripe payment method gateway is nil after registration")
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

	resaleImageStorage, err := outfirebase.NewResaleImageStorageFromEnv(ctx)
	if err != nil {
		return nil, err
	}

	resendClient := mailadp.NewResendClient(cfg.ResendAPIKey)

	orderCancellationMailer := mailadp.NewOrderCancellationMailer(
		resendClient,
		cfg.ResendFrom,
	)

	inquiryMailer := mailadp.NewInquiryMailer(
		resendClient,
	)

	avatarWalletSvc := solana.NewAvatarWalletService(
		infra.ProjectID,
	)

	transportationSvc := transportationdom.NewService(
		r.transportationRepo,
	)

	setupUC := usecase.NewSetupUsecase(
		r.avatarRepo,
	)

	announcementUC := usecase.NewAnnouncementUsecase(
		r.announcementRepo,
		r.announcementAvatarRepo,
		r.announcementAttachmentRepo,
	)

	resaleUC := usecase.NewResaleUsecase(
		r.resaleRepo,
		r.resaleImageRepo,
		resaleImageStorage,
	)

	avatarUC := usecase.NewAvatarUsecase(
		r.avatarRepo,
		avatarWalletSvc,
		r.walletRepo,
		r.cartRepo,
		nil,
	)

	shippingQuoteUC := usecase.NewShippingQuoteUsecase(
		r.listRepoFS,
		r.inventoryRepo,
		r.modelRepoFS,
		r.shippingAddressRepo,
		transportationSvc,
	)

	shippingAddressUC := usecase.NewShippingAddressUsecase(
		r.shippingAddressRepo,
	)

	paymentMethodUC := usecase.NewPaymentMethodUsecase(
		r.paymentMethodRepo,
		infra.PaymentMethodGateway,
	)

	avatarRegistrationUC := usecase.NewAvatarRegistrationUsecase(
		avatarUC,
		paymentMethodUC,
		cfg.AutoCreateStripeTestPaymentMethod,
	)

	userUC := usecase.NewUserUsecase(
		r.userRepo,
		nil,
	)

	onchainReader := solana.NewOnchainWalletReaderDevnet()

	walletUC := usecase.NewWalletUsecase(
		r.walletRepo,
		onchainReader,
		r.tokenReader,
		r.brandRepo,
		r.productRepo,
		r.productBlueprintRepoFS,
		r.productBlueprintRepoFS,
	)

	productBlueprintReviewUC := usecase.NewProductBlueprintReviewUsecase(
		r.productBlueprintReviewRepo,
		r.productBlueprintRepoFS,
		r.brandRepo,
		r.memberRepo,
		walletUC,
		r.avatarRepo,
		nil,
	)

	tokenBlueprintReviewUC := usecase.NewTokenBlueprintReviewUsecase(
		r.tokenBlueprintReviewRepo,
		r.avatarRepo,
		r.tokenBlueprintRepo,
		r.brandRepo,
	)

	cartUC := usecase.NewCartUsecase(
		r.cartRepo,
	)

	paymentUC := usecase.NewPaymentUsecase(
		usecase.NewPaymentUsecaseInput{
			PaymentRepo: r.paymentRepo,
			OrderRepo:   r.orderRepo,
			ResaleRepo:  r.resaleRepo,
		},
	)

	settlementDependencies, err := shared.BuildSettlementDependencies(
		ctx,
		infra,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"di.mall: build settlement dependencies: %w",
			err,
		)
	}

	payoutAccountAllowedReturnOrigin := cfg.FrontendBaseURL
	if payoutAccountAllowedReturnOrigin == "" {
		return nil, fmt.Errorf(
			"di.mall: %s is empty",
			mallFrontendBaseURLEnv,
		)
	}

	payoutAccountUC := usecase.NewPayoutAccountUsecase(
		r.payoutAccountRepo,
		infra.AccountGateway,
		payoutAccountAllowedReturnOrigin,
	)
	if payoutAccountUC == nil {
		return nil, errors.New("di.mall: payout account usecase is nil")
	}

	settlementUC := usecase.NewSettlementUsecase(
		usecase.NewSettlementUsecaseInput{
			Repository:            r.settlementRepo,
			Calculator:            settlementDependencies.SettlementCalculator,
			StripeTransferGateway: settlementDependencies.StripeTransferGateway,
		},
	)
	if settlementUC == nil {
		return nil, errors.New("di.mall: settlement usecase is nil")
	}

	refundUC := usecase.NewRefundUsecase(
		usecase.NewRefundUsecaseInput{
			PaymentReader:                 paymentUC,
			SettlementRepository:          r.settlementRepo,
			StripeRefundGateway:           settlementDependencies.StripeRefundGateway,
			StripeTransferReversalGateway: settlementDependencies.StripeTransferReversalGateway,
		},
	)
	if refundUC == nil {
		return nil, errors.New("di.mall: refund usecase is nil")
	}

	orderUC := usecase.NewOrderUsecase(
		r.orderRepo,
		r.listRepoFS,
		r.inventoryRepo,
		r.productBlueprintRepoFS,
		r.resaleRepo,
		r.paymentMethodRepo,
		r.shippingAddressRepo,
		shippingQuoteUC,
	).
		WithCartRepository(r.cartRepo).
		WithSellerRepositories(
			r.brandRepo,
			r.accountRepo,
		).
		WithCancellationNotification(
			authUserReader,
			orderCancellationMailer,
		)

	itemRefundUC := usecase.NewItemRefundUsecase(
		usecase.NewItemRefundUsecaseInput{
			OrderReader:                   orderUC,
			PaymentReader:                 paymentUC,
			SettlementRepository:          r.settlementRepo,
			RefundRepository:              r.refundRepo,
			PlatformFeeCalculator:         settlementDependencies.Calculator,
			StripeRefundGateway:           settlementDependencies.StripeRefundGateway,
			StripeTransferReversalGateway: settlementDependencies.StripeTransferReversalGateway,
		},
	)
	if itemRefundUC == nil {
		return nil, errors.New("di.mall: item refund usecase is nil")
	}

	inquiryUC := usecase.NewInquiryUsecase(
		r.inquiryRepo,
		r.inquiryReplyRepo,
		inquiryMailer,
		cfg.ResendFrom,
		r.avatarRepo,
		authUserReader,
	)

	returnRequestUC := usecase.NewReturnRequestUsecase(
		orderUC,
		r.inquiryRepo,
		inquiryUC,
	).WithReplyRepository(
		r.inquiryReplyRepo,
	)

	if infra.PaymentMethodGateway == nil {
		return nil, errors.New("di.mall: stripe payment intent gateway is nil")
	}

	paymentFlowUC := usecase.NewPaymentFlowUsecase(
		paymentUC,
		r.orderRepo,
		infra.PaymentMethodGateway,
	)

	inventoryUC := usecase.NewInventoryUsecase(
		r.inventoryRepo,
	)

	refundCompletionNotificationQueue, err :=
		cloudtasksadp.NewRefundCompletionNotificationQueueFromEnv(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"di.mall: build refund completion notification queue: %w",
			err,
		)
	}

	refundCompletionNotificationMailer :=
		mailadp.NewRefundCompletionNotificationMailer(
			resendClient,
			cfg.ResendFrom,
		)

	refundCompletionNotificationUC :=
		usecase.NewRefundCompletionNotificationUsecase(
			r.refundCompletionNotificationRepo,
			authUserReader,
			refundCompletionNotificationMailer,
			refundCompletionNotificationQueue,
		)

	return &mallUsecases{
		avatarUC:                       avatarUC,
		avatarRegistrationUC:           avatarRegistrationUC,
		setupUC:                        setupUC,
		shippingAddressUC:              shippingAddressUC,
		shippingQuoteUC:                shippingQuoteUC,
		paymentMethodUC:                paymentMethodUC,
		payoutAccountUC:                payoutAccountUC,
		userUC:                         userUC,
		walletUC:                       walletUC,
		cartUC:                         cartUC,
		paymentUC:                      paymentUC,
		settlementUC:                   settlementUC,
		refundUC:                       refundUC,
		itemRefundUC:                   itemRefundUC,
		refundCompletionNotificationUC: refundCompletionNotificationUC,
		orderUC:                        orderUC,
		inquiryUC:                      inquiryUC,
		returnRequestUC:                returnRequestUC,
		announcementUC:                 announcementUC,
		resaleUC:                       resaleUC,
		productBlueprintReviewUC:       productBlueprintReviewUC,
		tokenBlueprintReviewUC:         tokenBlueprintReviewUC,
		paymentFlowUC:                  paymentFlowUC,
		inventoryUC:                    inventoryUC,
	}, nil
}

// applyToContainer copies buyer-facing usecases into the public Mall container.
// TransferUC is built separately after PreviewQuery exists.
func (u *mallUsecases) applyToContainer(c *Container) {
	if u == nil || c == nil {
		return
	}

	c.AvatarUC = u.avatarUC
	c.AvatarRegistrationUC = u.avatarRegistrationUC
	c.SetupUC = u.setupUC
	c.ShippingAddressUC = u.shippingAddressUC
	c.ShippingQuoteUC = u.shippingQuoteUC
	c.PaymentMethodUC = u.paymentMethodUC
	c.PayoutAccountUC = u.payoutAccountUC
	c.UserUC = u.userUC
	c.WalletUC = u.walletUC
	c.CartUC = u.cartUC
	c.PaymentUC = u.paymentUC
	c.SettlementUC = u.settlementUC
	c.RefundUC = u.refundUC
	c.ItemRefundUC = u.itemRefundUC
	c.RefundCompletionNotificationUC = u.refundCompletionNotificationUC
	c.OrderUC = u.orderUC
	c.InquiryUC = u.inquiryUC
	c.ReturnRequestUC = u.returnRequestUC
	c.AnnouncementUC = u.announcementUC
	c.ResaleUC = u.resaleUC
	c.ProductBlueprintReviewUC = u.productBlueprintReviewUC
	c.TokenBlueprintReviewUC = u.tokenBlueprintReviewUC
	c.PaymentFlowUC = u.paymentFlowUC
}

// buildMallTransferUsecase is intentionally separated from buildMallUsecases.
// TransferUsecase depends on PreviewQuery, so it must be constructed only after
// the Mall query layer has been built.
func buildMallTransferUsecase(
	infra *shared.Infra,
	r *mallRepositories,
	u *mallUsecases,
	previewQ *mallquery.PreviewQuery,
) (*usecase.TransferUsecase, error) {
	if infra == nil || infra.Firestore == nil {
		return nil, errors.New("di.mall: firestore client is nil")
	}
	if r == nil {
		return nil, errors.New("di.mall: repositories are nil")
	}
	if u == nil {
		return nil, errors.New("di.mall: usecases are nil")
	}
	if previewQ == nil {
		return nil, errors.New("di.mall: preview query is nil")
	}
	if u.walletUC == nil {
		return nil, errors.New("di.mall: wallet usecase is nil")
	}
	if u.returnRequestUC == nil {
		return nil, errors.New("di.mall: return request usecase is nil")
	}
	if u.inventoryUC == nil {
		return nil, errors.New("di.mall: inventory usecase is nil")
	}

	var orderRepoForTransfer usecase.OrderRepoForTransfer = r.orderTransferItemRepo

	var tokenResolver usecase.TokenResolver = mallfs.NewTokenResolverFS(
		infra.Firestore,
		"tokens",
	)

	var tokenOwnerUpdater usecase.TokenOwnerUpdater = r.tokenOwnerUpdater

	var walletResolver usecase.BrandWalletResolver = r.walletResolverRepo

	var avatarWalletResolver usecase.AvatarWalletResolver = r.walletResolverRepo

	var walletTransferUpdate usecase.AvatarWalletItemTransferUpdater = r.walletRepo

	var walletSync usecase.AvatarWalletSyncer = u.walletUC

	var executor usecase.TokenTransferExecutor = solana.NewTokenTransferExecutorSolana("")

	transferExecutionUC := usecase.NewTokenTransferExecutionUsecase(
		tokenOwnerUpdater,
		walletTransferUpdate,
		walletSync,
		r.transferRepo,
		executor,
		nil,
	)

	transferUC := usecase.NewTransferUsecase(
		previewQ,
		orderRepoForTransfer,
		tokenResolver,
		walletResolver,
		avatarWalletResolver,
		r.brandRepo,
		r.avatarRepo,
		transferExecutionUC,
		u.inventoryUC,
	).
		WithResaleTransferDependencies(
			r.resaleRepo,
		).
		WithReturnOpeningHandler(
			u.returnRequestUC,
		)

	return transferUC, nil
}
