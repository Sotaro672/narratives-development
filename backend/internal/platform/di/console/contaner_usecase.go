// backend/internal/platform/di/console/contaner_usecase.go
package console

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	listcloudtasksadp "narratives/internal/adapters/out/cloudtasks"
	firebaseadp "narratives/internal/adapters/out/firebase"
	fsrepo "narratives/internal/adapters/out/firestore"
	cloudtasksadp "narratives/internal/adapters/out/firestore/cloudtasks"
	mailadp "narratives/internal/adapters/out/mail"
	stripeadapter "narratives/internal/adapters/out/stripe"
	uc "narratives/internal/application/usecase"
	settlementdom "narratives/internal/domain/settlement"
	"narratives/internal/infra/arweave"
	solanainfra "narratives/internal/infra/solana"
)

const (
	settlementStripeSecretID = "stripe-secret-key"

	settlementPlatformFeeRateEnv = "SETTLEMENT_PLATFORM_FEE_RATE"

	settlementPlatformFeeBaseEnv = "SETTLEMENT_PLATFORM_FEE_BASE"
)

type usecases struct {
	solanaMintClient                   *solanainfra.MintClient
	tokenUC                            *uc.TokenUsecase
	accountUC                          *uc.AccountUsecase
	announcementUC                     *uc.AnnouncementUsecase
	announcementAttachmentStorage      *firebaseadp.AnnouncementAttachmentStorage
	avatarUC                           *uc.AvatarUsecase
	paymentMethodUC                    *uc.PaymentMethodUsecase
	brandUC                            *uc.BrandUsecase
	companyUC                          *uc.CompanyUsecase
	inquiryUC                          *uc.InquiryUsecase
	itemRefundUC                       *uc.ItemRefundUsecase
	returnReceiptUC                    *uc.ReturnReceiptUsecase
	openedReturnReceiptUC              *uc.OpenedReturnReceiptUsecase
	inventoryUC                        *uc.InventoryUsecase
	listUC                             *uc.ListUsecase
	listSaveOperationUC                *uc.ListSaveOperationUsecase
	listSaveOperationStorage           *firebaseadp.ListSaveOperationStorage
	listSaveOperationRetryQueue        *listcloudtasksadp.ListSaveOperationQueue
	memberUC                           *uc.MemberUsecase
	modelUC                            *uc.ModelUsecase
	orderUC                            *uc.OrderUsecase
	orderDispatchNotificationUC        uc.OrderDispatchNotificationUsecasePort
	orderDispatchNotificationQueue     *listcloudtasksadp.OrderDispatchNotificationQueue
	refundCompletionNotificationUC     uc.RefundCompletionNotificationUsecasePort
	refundCompletionNotificationQueue  *listcloudtasksadp.RefundCompletionNotificationQueue
	paymentUC                          *uc.PaymentUsecase
	paymentFlowUC                      *uc.PaymentFlowUsecase
	settlementUC                       *uc.SettlementUsecase
	refundUC                           *uc.RefundUsecase
	permissionUC                       *uc.PermissionUsecase
	printUC                            *uc.PrintUsecase
	productionUC                       *uc.ProductionUsecase
	productBlueprintUC                 *uc.ProductBlueprintUsecase
	productBlueprintCategoryUC         *uc.ProductBlueprintCategoryUsecase
	inspectionUC                       *uc.InspectionUsecase
	mintUC                             *uc.MintUsecase
	shippingAddressUC                  *uc.ShippingAddressUsecase
	transportationUC                   *uc.TransportationUsecase
	tokenBlueprintUC                   *uc.TokenBlueprintUsecase
	tokenBlueprintAssetStorage         *firebaseadp.TokenBlueprintAssetStorage
	tokenBlueprintCreateOperationUC    *uc.TokenBlueprintCreateOperationUsecase
	tokenBlueprintCreateOperationQueue *listcloudtasksadp.TokenBlueprintCreateOperationQueue
	tokenBlueprintReviewUC             *uc.TokenBlueprintReviewUsecase
	productBlueprintReviewUC           *uc.ProductBlueprintReviewUsecase
	userUC                             *uc.UserUsecase
	walletUC                           *uc.WalletUsecase
	cartUC                             *uc.CartUsecase
	invitationUC                       uc.InvitationUsecasePort
	invitationDeliveryUC               uc.InvitationDeliveryUsecasePort
	invitationDeliveryQueue            *listcloudtasksadp.InvitationDeliveryQueue
	authBootstrapSvc                   *uc.BootstrapService
}

func buildSettlementUsecase(
	ctx context.Context,
	c *clients,
	r *repos,
) (*uc.SettlementUsecase, error) {
	if c == nil ||
		c.infra == nil {
		return nil, errors.New(
			"di.console: shared infra is nil",
		)
	}

	if r == nil ||
		r.settlementRepo == nil {
		return nil, errors.New(
			"di.console: settlement repository is nil",
		)
	}

	stripeSecretKey, err :=
		c.infra.AccessSecretVersion(
			ctx,
			settlementStripeSecretID,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"di.console: load Stripe settlement secret: %w",
			err,
		)
	}

	stripeSecretKey = strings.TrimSpace(
		stripeSecretKey,
	)
	if stripeSecretKey == "" ||
		!strings.HasPrefix(
			stripeSecretKey,
			"sk_",
		) {
		return nil, errors.New(
			"di.console: Stripe settlement secret is invalid",
		)
	}

	platformFeeRateText := strings.TrimSpace(
		os.Getenv(
			settlementPlatformFeeRateEnv,
		),
	)
	if platformFeeRateText == "" {
		return nil, fmt.Errorf(
			"di.console: %s is empty",
			settlementPlatformFeeRateEnv,
		)
	}

	platformFeeRate, err := strconv.Atoi(
		platformFeeRateText,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"di.console: invalid %s: %w",
			settlementPlatformFeeRateEnv,
			err,
		)
	}

	platformFeeBaseText := strings.TrimSpace(
		os.Getenv(
			settlementPlatformFeeBaseEnv,
		),
	)
	if platformFeeBaseText == "" {
		return nil, fmt.Errorf(
			"di.console: %s is empty",
			settlementPlatformFeeBaseEnv,
		)
	}

	platformFeeCalculator, err :=
		settlementdom.NewPercentagePlatformFeeCalculator(
			platformFeeRate,
			settlementdom.PlatformFeeBase(
				platformFeeBaseText,
			),
		)
	if err != nil {
		return nil, fmt.Errorf(
			"di.console: build settlement platform fee calculator: %w",
			err,
		)
	}

	calculator := settlementdom.NewCalculator(
		platformFeeCalculator,
	)
	if calculator == nil {
		return nil, errors.New(
			"di.console: settlement calculator is nil",
		)
	}

	stripeTransferGateway :=
		stripeadapter.NewTransferGateway(
			stripeSecretKey,
		)
	if stripeTransferGateway == nil {
		return nil, errors.New(
			"di.console: Stripe settlement transfer gateway is nil",
		)
	}

	settlementUC := uc.NewSettlementUsecase(
		uc.NewSettlementUsecaseInput{
			Repository: r.settlementRepo,

			Calculator: calculator,

			StripeTransferGateway: stripeTransferGateway,
		},
	)
	if settlementUC == nil {
		return nil, errors.New(
			"di.console: settlement usecase is nil",
		)
	}

	return settlementUC, nil
}

func buildRefundUsecase(
	ctx context.Context,
	c *clients,
	r *repos,
	paymentUC *uc.PaymentUsecase,
) (*uc.RefundUsecase, error) {
	if c == nil ||
		c.infra == nil {
		return nil, errors.New(
			"di.console: shared infra is nil",
		)
	}

	if r == nil ||
		r.settlementRepo == nil {
		return nil, errors.New(
			"di.console: settlement repository is nil",
		)
	}

	if paymentUC == nil {
		return nil, errors.New(
			"di.console: payment usecase is nil",
		)
	}

	stripeSecretKey, err :=
		c.infra.AccessSecretVersion(
			ctx,
			settlementStripeSecretID,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"di.console: load Stripe refund secret: %w",
			err,
		)
	}

	stripeSecretKey = strings.TrimSpace(
		stripeSecretKey,
	)
	if stripeSecretKey == "" ||
		!strings.HasPrefix(
			stripeSecretKey,
			"sk_",
		) {
		return nil, errors.New(
			"di.console: Stripe refund secret is invalid",
		)
	}

	stripeRefundGateway :=
		stripeadapter.NewRefundGateway(
			stripeSecretKey,
		)
	if stripeRefundGateway == nil {
		return nil, errors.New(
			"di.console: Stripe refund gateway is nil",
		)
	}

	stripeTransferReversalGateway :=
		stripeadapter.NewTransferReversalGateway(
			stripeSecretKey,
		)
	if stripeTransferReversalGateway == nil {
		return nil, errors.New(
			"di.console: Stripe transfer reversal gateway is nil",
		)
	}

	refundUC := uc.NewRefundUsecase(
		uc.NewRefundUsecaseInput{
			PaymentReader: paymentUC,

			SettlementRepository: r.settlementRepo,

			StripeRefundGateway: stripeRefundGateway,

			StripeTransferReversalGateway: stripeTransferReversalGateway,
		},
	)
	if refundUC == nil {
		return nil, errors.New(
			"di.console: refund usecase is nil",
		)
	}

	return refundUC, nil
}

func buildItemRefundUsecase(
	ctx context.Context,
	c *clients,
	r *repos,
	orderUC *uc.OrderUsecase,
	paymentUC *uc.PaymentUsecase,
) (*uc.ItemRefundUsecase, error) {
	if c == nil ||
		c.infra == nil {
		return nil, errors.New(
			"di.console: shared infra is nil",
		)
	}

	if r == nil ||
		r.settlementRepo == nil {
		return nil, errors.New(
			"di.console: settlement repository is nil",
		)
	}

	if r.refundRepo == nil {
		return nil, errors.New(
			"di.console: refund repository is nil",
		)
	}

	if orderUC == nil {
		return nil, errors.New(
			"di.console: order usecase is nil",
		)
	}

	if paymentUC == nil {
		return nil, errors.New(
			"di.console: payment usecase is nil",
		)
	}

	stripeSecretKey, err :=
		c.infra.AccessSecretVersion(
			ctx,
			settlementStripeSecretID,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"di.console: load Stripe item refund secret: %w",
			err,
		)
	}

	stripeSecretKey = strings.TrimSpace(
		stripeSecretKey,
	)
	if stripeSecretKey == "" ||
		!strings.HasPrefix(
			stripeSecretKey,
			"sk_",
		) {
		return nil, errors.New(
			"di.console: Stripe item refund secret is invalid",
		)
	}

	platformFeeRateText := strings.TrimSpace(
		os.Getenv(
			settlementPlatformFeeRateEnv,
		),
	)
	if platformFeeRateText == "" {
		return nil, fmt.Errorf(
			"di.console: %s is empty",
			settlementPlatformFeeRateEnv,
		)
	}

	platformFeeRate, err := strconv.Atoi(
		platformFeeRateText,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"di.console: invalid %s: %w",
			settlementPlatformFeeRateEnv,
			err,
		)
	}

	platformFeeBaseText := strings.TrimSpace(
		os.Getenv(
			settlementPlatformFeeBaseEnv,
		),
	)
	if platformFeeBaseText == "" {
		return nil, fmt.Errorf(
			"di.console: %s is empty",
			settlementPlatformFeeBaseEnv,
		)
	}

	platformFeeCalculator, err :=
		settlementdom.NewPercentagePlatformFeeCalculator(
			platformFeeRate,
			settlementdom.PlatformFeeBase(
				platformFeeBaseText,
			),
		)
	if err != nil {
		return nil, fmt.Errorf(
			"di.console: build item refund platform fee calculator: %w",
			err,
		)
	}

	stripeRefundGateway :=
		stripeadapter.NewRefundGateway(
			stripeSecretKey,
		)
	if stripeRefundGateway == nil {
		return nil, errors.New(
			"di.console: Stripe item refund gateway is nil",
		)
	}

	stripeTransferReversalGateway :=
		stripeadapter.NewTransferReversalGateway(
			stripeSecretKey,
		)
	if stripeTransferReversalGateway == nil {
		return nil, errors.New(
			"di.console: Stripe item transfer reversal gateway is nil",
		)
	}

	itemRefundUC := uc.NewItemRefundUsecase(
		uc.NewItemRefundUsecaseInput{
			OrderReader: orderUC,

			PaymentReader: paymentUC,

			SettlementRepository: r.settlementRepo,

			RefundRepository: r.refundRepo,

			PlatformFeeCalculator: platformFeeCalculator,

			StripeRefundGateway: stripeRefundGateway,

			StripeTransferReversalGateway: stripeTransferReversalGateway,
		},
	)
	if itemRefundUC == nil {
		return nil, errors.New(
			"di.console: item refund usecase is nil",
		)
	}

	return itemRefundUC, nil
}

func buildUsecases(
	ctx context.Context,
	c *clients,
	r *repos,
	s *services,
	res *resolvers,
) (*usecases, error) {
	solanaClient, err := solanainfra.NewMintClient(ctx)
	if err != nil {
		return nil, err
	}

	tokenUC := uc.NewTokenUsecase(solanaClient)

	if c == nil ||
		c.infra == nil {
		return nil, errors.New(
			"di.console: shared infra is nil",
		)
	}

	if c.infra.PaymentMethodGateway == nil {
		var customerStore stripeadapter.PaymentMethodCustomerStore

		if value, ok :=
			any(r.paymentMethodRepo).(stripeadapter.PaymentMethodCustomerStore); ok {
			customerStore = value
		} else if value, ok :=
			any(r.userRepo).(stripeadapter.PaymentMethodCustomerStore); ok {
			customerStore = value
		}

		if customerStore == nil {
			return nil, errors.New(
				"di.console: PaymentMethodCustomerStore is not implemented by current repositories",
			)
		}

		if err :=
			c.infra.RegisterPaymentMethodGatewayFromSecret(
				ctx,
				customerStore,
			); err != nil {
			return nil, err
		}

		if c.infra.PaymentMethodGateway == nil {
			return nil, errors.New(
				"di.console: stripe payment method gateway is nil after registration",
			)
		}
	}

	if c.infra.AccountGateway == nil {
		if err :=
			c.infra.RegisterAccountGatewayFromSecret(
				ctx,
			); err != nil {
			return nil, err
		}

		if c.infra.AccountGateway == nil {
			return nil, errors.New(
				"di.console: stripe account gateway is nil after registration",
			)
		}
	}

	accountUC := uc.NewAccountUsecase(
		r.accountRepo,
		c.infra.AccountGateway,
	)

	announcementAvatarRepo := fsrepo.NewAnnouncementAvatarRepositoryFS(c.fsClient)
	announcementAttachmentRepo := fsrepo.NewAnnouncementAttachmentRepositoryFS(c.fsClient)

	announcementAttachmentStorage, err := firebaseadp.NewAnnouncementAttachmentStorageFromEnv(ctx)
	if err != nil {
		return nil, err
	}

	announcementUC := uc.NewAnnouncementUsecase(
		r.announcementRepo,
		announcementAvatarRepo,
		announcementAttachmentRepo,
	).WithAttachmentStorage(announcementAttachmentStorage)

	brandWalletSvc := solanainfra.NewBrandWalletService(c.firestoreProjectID)
	avatarWalletSvc := solanainfra.NewAvatarWalletService(c.firestoreProjectID)

	avatarUC := uc.NewAvatarUsecase(
		r.avatarRepo,
		avatarWalletSvc,
		r.walletRepo,
		r.cartRepo,
		nil,
	)

	paymentMethodUC := uc.NewPaymentMethodUsecase(
		r.paymentMethodRepo,
		c.infra.PaymentMethodGateway,
	)

	brandUC := uc.NewBrandUsecase(
		r.brandRepo,
		r.memberRepo,
		r.accountRepo,
		uc.WithBrandWalletService(brandWalletSvc),
	)

	companyUC := uc.NewCompanyUsecase(r.companyRepo)

	inquiryUC := uc.NewInquiryUsecase(
		r.inquiryRepo,
		r.inquiryReplyRepo,
		nil,
		"",
		"",
		nil,
		nil,
	)

	inventoryUC := uc.NewInventoryUsecase(r.inventoryRepo)

	inventoryUC.WithShippingAddressAssignment(
		r.shippingAddressRepo,
		r.productBlueprintRepo,
	)

	if r.productRepo != nil {
		if resolver, ok := any(r.productRepo).(uc.ProductModelResolver); ok {
			inventoryUC.WithProductModelResolver(resolver)
		}
	}

	paymentUC := uc.NewPaymentUsecase(
		uc.NewPaymentUsecaseInput{
			PaymentRepo: r.paymentRepo,
			OrderRepo:   r.orderRepo,
			ResaleRepo:  r.resaleRepo,
		},
	)

	settlementUC, err := buildSettlementUsecase(
		ctx,
		c,
		r,
	)
	if err != nil {
		_ = announcementAttachmentStorage.Close()
		return nil, err
	}

	refundUC, err := buildRefundUsecase(
		ctx,
		c,
		r,
		paymentUC,
	)
	if err != nil {
		_ = announcementAttachmentStorage.Close()
		return nil, err
	}

	listSaveOperationStorage, err := firebaseadp.NewListSaveOperationStorageFromEnv(ctx)
	if err != nil {
		_ = announcementAttachmentStorage.Close()
		return nil, err
	}

	transportationRepo := fsrepo.NewTransportationRepositoryFS(c.fsClient)

	inventoryUC.WithTransportationAssignment(
		transportationRepo,
		r.productBlueprintRepo,
	)

	listUC := uc.NewListUsecase(
		r.listRepoFS,
		r.listImageRecordRepo,
		listSaveOperationStorage,
	)

	listSaveOperationRetryQueue, err := listcloudtasksadp.NewListSaveOperationQueueFromEnv(ctx)
	if err != nil {
		_ = listSaveOperationStorage.Close()
		_ = announcementAttachmentStorage.Close()
		return nil, err
	}

	listSaveOperationUC := uc.NewListSaveOperationUsecase(
		uc.NewListSaveOperationUsecaseParams{
			ListRepository:      r.listRepoFS,
			ImageRepository:     r.listImageRecordRepo,
			OperationRepository: r.listSaveOperationRepo,
			Storage:             listSaveOperationStorage,
			RetryQueue:          listSaveOperationRetryQueue,
		},
	)

	modelUC := uc.NewModelUsecase(
		r.modelRepo,
		r.productBlueprintRepo,
	)

	shippingQuoteUC := uc.NewShippingQuoteUsecase(
		r.listRepoFS,
		r.inventoryRepo,
		r.modelRepo,
		r.shippingAddressRepo,
		s.transportationSvc,
	)

	orderUC := uc.NewOrderUsecase(
		r.orderRepo,
		r.listRepoFS,
		r.inventoryRepo,
		r.productBlueprintRepo,
		r.resaleRepo,
		r.paymentMethodRepo,
		r.shippingAddressRepo,
		shippingQuoteUC,
	).WithSellerRepositories(
		r.brandRepo,
		r.accountRepo,
	)

	itemRefundUC, err := buildItemRefundUsecase(
		ctx,
		c,
		r,
		orderUC,
		paymentUC,
	)
	if err != nil {
		_ = listSaveOperationRetryQueue.Close()
		_ = listSaveOperationStorage.Close()
		_ = announcementAttachmentStorage.Close()
		return nil, err
	}

	if paymentUC == nil {
		_ = listSaveOperationRetryQueue.Close()
		_ = listSaveOperationStorage.Close()
		_ = announcementAttachmentStorage.Close()

		return nil, errors.New(
			"di.console: payment usecase is nil",
		)
	}

	if r.orderRepo == nil {
		_ = listSaveOperationRetryQueue.Close()
		_ = listSaveOperationStorage.Close()
		_ = announcementAttachmentStorage.Close()

		return nil, errors.New(
			"di.console: order repository is nil",
		)
	}

	if c.infra.PaymentMethodGateway == nil {
		_ = listSaveOperationRetryQueue.Close()
		_ = listSaveOperationStorage.Close()
		_ = announcementAttachmentStorage.Close()

		return nil, errors.New(
			"di.console: payment method gateway is nil",
		)
	}

	paymentFlowUC := uc.NewPaymentFlowUsecase(
		paymentUC,
		r.orderRepo,
		c.infra.PaymentMethodGateway,
	)

	if paymentFlowUC == nil {
		_ = listSaveOperationRetryQueue.Close()
		_ = listSaveOperationStorage.Close()
		_ = announcementAttachmentStorage.Close()

		return nil, errors.New(
			"di.console: payment flow usecase is nil",
		)
	}

	permissionUC := uc.NewPermissionUsecase(r.permissionRepo)

	printUC := uc.NewPrintUsecase(
		r.productionRepo,
		r.productRepo,
		r.printLogRepo,
		r.inspectionRepo,
		r.productBlueprintRepo,
	)

	productionUC := uc.NewProductionUsecase(r.productionRepo)

	productBlueprintUC := uc.NewProductBlueprintUsecase(
		r.productBlueprintRepo,
		r.productBlueprintReviewRepo,
	)

	productBlueprintCategoryUC := uc.NewProductBlueprintCategoryUsecase(
		r.productBlueprintCategoryRepo,
	)

	inspectionUC := uc.NewInspectionUsecase(
		r.inspectionRepo,
		r.productRepo,
	)

	mintUC := uc.NewMintUsecase(
		r.productionRepo,
		r.tokenBlueprintRepo,
		r.mintRepo,
		r.inspectionRepo,
		tokenUC,
	)

	mintUC.SetInventoryUsecase(inventoryUC)

	// 1件ずつmintするためのtask repositoryと
	// token保存recorderを注入します。
	mintUC.SetMintTaskRepository(r.mintRepo)
	mintUC.SetMintProductMintRecorder(r.mintRepo)

	// Cloud Tasksへ次のmint処理を投入するenqueuerを注入します。
	// mint worker は必須依存のため、初期化失敗を握り潰さず
	// application startup 自体を失敗させます。
	mintTaskQueue, err := cloudtasksadp.NewMintTaskQueueFromEnv(ctx)
	if err != nil {
		_ = listSaveOperationStorage.Close()
		_ = announcementAttachmentStorage.Close()
		return nil, err
	}

	if mintTaskQueue == nil {
		_ = listSaveOperationStorage.Close()
		_ = announcementAttachmentStorage.Close()
		return nil, errors.New("mint task queue is nil")
	}

	mintUC.SetMintTaskEnqueuer(mintTaskQueue)

	baseURL := os.Getenv("ARWEAVE_BASE_URL")
	apiKey := os.Getenv("IRYS_SERVICE_API_KEY")
	uploader := arweave.NewHTTPUploader(baseURL, apiKey)

	tbReviewRepo := fsrepo.NewTokenBlueprintReviewRepositoryFS(c.fsClient)

	tokenBlueprintAssetStorage, err := firebaseadp.NewTokenBlueprintAssetStorageFromEnv(ctx)
	if err != nil {
		_ = listSaveOperationStorage.Close()
		_ = announcementAttachmentStorage.Close()
		return nil, err
	}

	tokenBlueprintUC := uc.NewTokenBlueprintUsecase(
		r.tokenBlueprintRepo,
		tbReviewRepo,
		tokenBlueprintAssetStorage,
		uploader,
	)

	tokenBlueprintCreateOperationQueue, err :=
		listcloudtasksadp.NewTokenBlueprintCreateOperationQueueFromEnv(
			ctx,
		)
	if err != nil {
		_ = tokenBlueprintAssetStorage.Close()
		_ = listSaveOperationRetryQueue.Close()
		_ = listSaveOperationStorage.Close()
		_ = announcementAttachmentStorage.Close()
		return nil, err
	}

	tokenBlueprintCreateOperationUC :=
		uc.NewTokenBlueprintCreateOperationUsecase(
			uc.NewTokenBlueprintCreateOperationUsecaseParams{
				TokenBlueprintUsecase: tokenBlueprintUC,
				OperationRepository:   r.tokenBlueprintCreateOperationRepo,
				Storage:               tokenBlueprintAssetStorage,
				Queue:                 tokenBlueprintCreateOperationQueue,
			},
		)

	mintUC.SetTokenBlueprintMetadataEnsurer(tokenBlueprintUC)
	mintUC.SetTokenBlueprintMintMarker(tokenBlueprintUC)

	shippingAddressUC := uc.NewShippingAddressUsecase(
		r.shippingAddressRepo,
	).WithInventoryCleaner(
		r.inventoryRepo,
	)

	transportationUC := uc.NewTransportationUsecase(transportationRepo)

	tokenBlueprintReviewUC := uc.NewTokenBlueprintReviewUsecase(
		tbReviewRepo,
		r.avatarRepo,
		r.tokenBlueprintRepo,
		r.brandRepo,
	)

	userUC := uc.NewUserUsecase(
		r.userRepo,
		nil,
	)

	onchainReader := solanainfra.NewOnchainWalletReaderDevnet()
	tokenQuery := fsrepo.NewTokenReaderFS(c.fsClient)

	walletUC := uc.NewWalletUsecase(
		r.walletRepo,
		onchainReader,
		tokenQuery,
		r.brandRepo,
		r.productRepo,
		r.productBlueprintRepo,
		r.productBlueprintRepo,
	)

	cartUC := uc.NewCartUsecase(r.cartRepo)

	invitationDeliveryQueue, err := listcloudtasksadp.NewInvitationDeliveryQueueFromEnv(ctx)
	if err != nil {
		_ = tokenBlueprintCreateOperationQueue.Close()
		_ = tokenBlueprintAssetStorage.Close()
		_ = listSaveOperationStorage.Close()
		_ = announcementAttachmentStorage.Close()
		return nil, err
	}

	invitationMailer := mailadp.NewInvitationMailerWithResend(
		r.companyRepo,
		r.brandRepo,
	)

	invitationDeliveryUC := uc.NewInvitationDeliveryUsecase(
		r.invitationTokenRepo,
		invitationMailer,
		invitationDeliveryQueue,
	)

	invitationUC := uc.NewInvitationUsecase(
		r.invitationTokenRepo,
		r.invitationTokenRepo,
		r.memberRepo,
		invitationDeliveryQueue,
	)

	orderDispatchNotificationQueue, err := listcloudtasksadp.NewOrderDispatchNotificationQueueFromEnv(ctx)
	if err != nil {
		_ = invitationDeliveryQueue.Close()
		_ = tokenBlueprintCreateOperationQueue.Close()
		_ = tokenBlueprintAssetStorage.Close()
		_ = listSaveOperationStorage.Close()
		_ = announcementAttachmentStorage.Close()
		return nil, err
	}

	authUserReader := firebaseadp.NewAuthUserReader(
		c.infra.FirebaseAuth,
	)

	orderDispatchNotificationMailer := mailadp.NewOrderDispatchNotificationMailerWithResend()

	orderDispatchNotificationUC := uc.NewOrderDispatchNotificationUsecase(
		r.orderDispatchNotificationRepo,
		authUserReader,
		uc.CompanyIDFromContext,
		r.productBlueprintRepo,
		orderDispatchNotificationMailer,
		orderDispatchNotificationQueue,
	)

	refundCompletionNotificationQueue, err :=
		listcloudtasksadp.NewRefundCompletionNotificationQueueFromEnv(
			ctx,
		)
	if err != nil {
		_ = orderDispatchNotificationQueue.Close()
		_ = invitationDeliveryQueue.Close()
		_ = tokenBlueprintCreateOperationQueue.Close()
		_ = tokenBlueprintAssetStorage.Close()
		_ = listSaveOperationStorage.Close()
		_ = announcementAttachmentStorage.Close()
		return nil, err
	}

	refundCompletionNotificationRepo :=
		fsrepo.NewRefundCompletionNotificationRepositoryFS(
			c.fsClient,
		)

	refundCompletionNotificationMailer :=
		mailadp.NewRefundCompletionNotificationMailerWithResend()

	refundCompletionNotificationUC :=
		uc.NewRefundCompletionNotificationUsecase(
			refundCompletionNotificationRepo,
			authUserReader,
			refundCompletionNotificationMailer,
			refundCompletionNotificationQueue,
		)

	// ReturnReceiptUsecase は未開封返品受領の orchestration を担当します。
	//
	// item-level partial refund は既存 RefundUsecase の full Payment refund と
	// 責務が異なるため、ItemRefundUsecase を明示的に注入します。
	returnReceiptUC := uc.NewReturnReceiptUsecase(
		orderUC,
		r.inquiryRepo,
		inquiryUC,
		itemRefundUC,
	).WithRefundCompletionNotifier(
		refundCompletionNotificationUC,
	)

	// OpenedReturnReceiptUsecase は開封後返品受領の orchestration を担当します。
	//
	// 返金額そのものは frontend から受け取らず、選択された refund policy と
	// Order snapshot から ItemRefundUsecase が権威的に算出します。
	openedReturnReceiptUC := uc.NewOpenedReturnReceiptUsecase(
		orderUC,
		r.inquiryRepo,
		inquiryUC,
		itemRefundUC,
	).WithRefundCompletionNotifier(
		refundCompletionNotificationUC,
	)

	memberUC := uc.NewMemberUsecase(r.memberRepo)

	autoCreateTestAccount := strings.Contains(
		strings.ToLower(
			strings.TrimSpace(
				c.firestoreProjectID,
			),
		),
		"development",
	)

	authBootstrapSvc := &uc.BootstrapService{
		Members:               r.memberRepo,
		Companies:             r.companyRepo,
		Accounts:              accountUC,
		AutoCreateTestAccount: autoCreateTestAccount,
	}

	_ = res

	return &usecases{
		solanaMintClient:                   solanaClient,
		tokenUC:                            tokenUC,
		accountUC:                          accountUC,
		announcementUC:                     announcementUC,
		announcementAttachmentStorage:      announcementAttachmentStorage,
		avatarUC:                           avatarUC,
		paymentMethodUC:                    paymentMethodUC,
		brandUC:                            brandUC,
		companyUC:                          companyUC,
		inquiryUC:                          inquiryUC,
		itemRefundUC:                       itemRefundUC,
		returnReceiptUC:                    returnReceiptUC,
		openedReturnReceiptUC:              openedReturnReceiptUC,
		inventoryUC:                        inventoryUC,
		listUC:                             listUC,
		listSaveOperationUC:                listSaveOperationUC,
		listSaveOperationStorage:           listSaveOperationStorage,
		listSaveOperationRetryQueue:        listSaveOperationRetryQueue,
		memberUC:                           memberUC,
		modelUC:                            modelUC,
		orderUC:                            orderUC,
		orderDispatchNotificationUC:        orderDispatchNotificationUC,
		orderDispatchNotificationQueue:     orderDispatchNotificationQueue,
		refundCompletionNotificationUC:     refundCompletionNotificationUC,
		refundCompletionNotificationQueue:  refundCompletionNotificationQueue,
		paymentUC:                          paymentUC,
		paymentFlowUC:                      paymentFlowUC,
		settlementUC:                       settlementUC,
		refundUC:                           refundUC,
		permissionUC:                       permissionUC,
		printUC:                            printUC,
		productionUC:                       productionUC,
		productBlueprintUC:                 productBlueprintUC,
		productBlueprintCategoryUC:         productBlueprintCategoryUC,
		inspectionUC:                       inspectionUC,
		mintUC:                             mintUC,
		shippingAddressUC:                  shippingAddressUC,
		transportationUC:                   transportationUC,
		tokenBlueprintUC:                   tokenBlueprintUC,
		tokenBlueprintAssetStorage:         tokenBlueprintAssetStorage,
		tokenBlueprintCreateOperationUC:    tokenBlueprintCreateOperationUC,
		tokenBlueprintCreateOperationQueue: tokenBlueprintCreateOperationQueue,
		tokenBlueprintReviewUC:             tokenBlueprintReviewUC,

		productBlueprintReviewUC: func() *uc.ProductBlueprintReviewUsecase {
			if r.productBlueprintReviewRepo == nil ||
				r.productBlueprintRepo == nil ||
				r.walletRepo == nil {
				return nil
			}

			return uc.NewProductBlueprintReviewUsecase(
				r.productBlueprintReviewRepo,
				r.productBlueprintRepo,
				r.brandRepo,
				r.memberRepo,
				walletUC,
				r.avatarRepo,
				nil,
			)
		}(),

		userUC:                  userUC,
		walletUC:                walletUC,
		cartUC:                  cartUC,
		invitationUC:            invitationUC,
		invitationDeliveryUC:    invitationDeliveryUC,
		invitationDeliveryQueue: invitationDeliveryQueue,
		authBootstrapSvc:        authBootstrapSvc,
	}, nil
}
