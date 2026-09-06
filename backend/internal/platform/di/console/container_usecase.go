// backend\internal\platform\di\console\container_usecase.go
package console

import (
	"context"
	"errors"
	"os"
	"strings"

	cloudtasksadp "narratives/internal/adapters/out/cloudtasks"
	firebaseadp "narratives/internal/adapters/out/firebase"
	fsrepo "narratives/internal/adapters/out/firestore"
	mailadp "narratives/internal/adapters/out/mail"
	stripeadapter "narratives/internal/adapters/out/stripe"
	uc "narratives/internal/application/usecase"
	"narratives/internal/infra/arweave"
	solanainfra "narratives/internal/infra/solana"
	shared "narratives/internal/platform/di/shared"
)

type usecases struct {
	resources                       *containerResources
	solanaMintClient                *solanainfra.MintClient
	tokenUC                         *uc.TokenUsecase
	accountUC                       *uc.AccountUsecase
	announcementUC                  *uc.AnnouncementUsecase
	avatarUC                        *uc.AvatarUsecase
	paymentMethodUC                 *uc.PaymentMethodUsecase
	brandUC                         *uc.BrandUsecase
	companyUC                       *uc.CompanyUsecase
	inquiryUC                       *uc.InquiryUsecase
	itemRefundUC                    *uc.ItemRefundUsecase
	returnReceiptUC                 *uc.ReturnReceiptUsecase
	openedReturnReceiptUC           *uc.OpenedReturnReceiptUsecase
	inventoryUC                     *uc.InventoryUsecase
	listUC                          *uc.ListUsecase
	listSaveOperationUC             *uc.ListSaveOperationUsecase
	memberUC                        *uc.MemberUsecase
	modelUC                         *uc.ModelUsecase
	orderUC                         *uc.OrderUsecase
	orderDispatchNotificationUC     uc.OrderDispatchNotificationUsecasePort
	refundCompletionNotificationUC  uc.RefundCompletionNotificationUsecasePort
	paymentUC                       *uc.PaymentUsecase
	paymentFlowUC                   *uc.PaymentFlowUsecase
	salesReceivableUC               *uc.SalesReceivableUsecase
	settlementUC                    *uc.SettlementUsecase
	refundUC                        *uc.RefundUsecase
	permissionUC                    *uc.PermissionUsecase
	printUC                         *uc.PrintUsecase
	productionUC                    *uc.ProductionUsecase
	productBlueprintUC              *uc.ProductBlueprintUsecase
	productBlueprintCategoryUC      *uc.ProductBlueprintCategoryUsecase
	inspectionUC                    *uc.InspectionUsecase
	mintUC                          *uc.MintUsecase
	shippingAddressUC               *uc.ShippingAddressUsecase
	transportationUC                *uc.TransportationUsecase
	tokenBlueprintUC                *uc.TokenBlueprintUsecase
	tokenBlueprintCreateOperationUC *uc.TokenBlueprintCreateOperationUsecase
	tokenBlueprintReviewUC          *uc.TokenBlueprintReviewUsecase
	productBlueprintReviewUC        *uc.ProductBlueprintReviewUsecase
	reviewReportUC                  *uc.ReportUsecase
	userUC                          *uc.UserUsecase
	walletUC                        *uc.WalletUsecase
	cartUC                          *uc.CartUsecase
	invitationUC                    uc.InvitationUsecasePort
	invitationDeliveryUC            uc.InvitationDeliveryUsecasePort
	authBootstrapSvc                *uc.BootstrapService
}

func buildSettlementUsecase(
	r *repos,
	salesReceivableUC *uc.SalesReceivableUsecase,
	dependencies *shared.SettlementDependencies,
) (*uc.SettlementUsecase, error) {
	if r == nil || r.settlementRepo == nil {
		return nil, errors.New("di.console: settlement repository is nil")
	}
	if salesReceivableUC == nil {
		return nil, errors.New("di.console: sales receivable usecase is nil")
	}
	if dependencies == nil {
		return nil, errors.New("di.console: settlement dependencies are nil")
	}
	if dependencies.SettlementCalculator == nil {
		return nil, errors.New("di.console: settlement calculator is nil")
	}
	if dependencies.StripeTransferGateway == nil {
		return nil, errors.New("di.console: Stripe settlement transfer gateway is nil")
	}

	settlementUC := uc.NewSettlementUsecase(uc.NewSettlementUsecaseInput{
		Repository:             r.settlementRepo,
		Calculator:             dependencies.SettlementCalculator,
		SalesReceivableUsecase: salesReceivableUC,
		StripeTransferGateway:  dependencies.StripeTransferGateway,
	})
	if settlementUC == nil {
		return nil, errors.New("di.console: settlement usecase is nil")
	}

	return settlementUC, nil
}

func buildRefundUsecase(
	r *repos,
	paymentUC *uc.PaymentUsecase,
	salesReceivableUC *uc.SalesReceivableUsecase,
	dependencies *shared.SettlementDependencies,
) (*uc.RefundUsecase, error) {
	if r == nil || r.settlementRepo == nil {
		return nil, errors.New("di.console: settlement repository is nil")
	}
	if paymentUC == nil {
		return nil, errors.New("di.console: payment usecase is nil")
	}
	if salesReceivableUC == nil {
		return nil, errors.New("di.console: sales receivable usecase is nil")
	}
	if dependencies == nil {
		return nil, errors.New("di.console: settlement dependencies are nil")
	}
	if dependencies.StripeRefundGateway == nil {
		return nil, errors.New("di.console: Stripe refund gateway is nil")
	}
	if dependencies.StripeTransferReversalGateway == nil {
		return nil, errors.New("di.console: Stripe transfer reversal gateway is nil")
	}

	refundUC := uc.NewRefundUsecase(uc.NewRefundUsecaseInput{
		PaymentReader:                 paymentUC,
		SettlementRepository:          r.settlementRepo,
		SalesReceivableService:        salesReceivableUC,
		StripeRefundGateway:           dependencies.StripeRefundGateway,
		StripeTransferReversalGateway: dependencies.StripeTransferReversalGateway,
	})
	if refundUC == nil {
		return nil, errors.New("di.console: refund usecase is nil")
	}

	return refundUC, nil
}

func buildItemRefundUsecase(
	r *repos,
	orderUC *uc.OrderUsecase,
	paymentUC *uc.PaymentUsecase,
	salesReceivableUC *uc.SalesReceivableUsecase,
	dependencies *shared.SettlementDependencies,
) (*uc.ItemRefundUsecase, error) {
	if r == nil || r.settlementRepo == nil {
		return nil, errors.New("di.console: settlement repository is nil")
	}
	if r.refundRepo == nil {
		return nil, errors.New("di.console: refund repository is nil")
	}
	if orderUC == nil {
		return nil, errors.New("di.console: order usecase is nil")
	}
	if paymentUC == nil {
		return nil, errors.New("di.console: payment usecase is nil")
	}
	if salesReceivableUC == nil {
		return nil, errors.New("di.console: sales receivable usecase is nil")
	}
	if dependencies == nil {
		return nil, errors.New("di.console: settlement dependencies are nil")
	}
	if dependencies.Calculator == nil {
		return nil, errors.New("di.console: platform fee calculator is nil")
	}
	if dependencies.StripeRefundGateway == nil {
		return nil, errors.New("di.console: Stripe item refund gateway is nil")
	}
	if dependencies.StripeTransferReversalGateway == nil {
		return nil, errors.New("di.console: Stripe item transfer reversal gateway is nil")
	}

	itemRefundUC := uc.NewItemRefundUsecase(uc.NewItemRefundUsecaseInput{
		OrderReader:                   orderUC,
		PaymentReader:                 paymentUC,
		SettlementRepository:          r.settlementRepo,
		SalesReceivableService:        salesReceivableUC,
		RefundRepository:              r.refundRepo,
		PlatformFeeCalculator:         dependencies.Calculator,
		StripeRefundGateway:           dependencies.StripeRefundGateway,
		StripeTransferReversalGateway: dependencies.StripeTransferReversalGateway,
	})
	if itemRefundUC == nil {
		return nil, errors.New("di.console: item refund usecase is nil")
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
	if c == nil || c.infra == nil {
		return nil, errors.New("di.console: shared infra is nil")
	}
	if r == nil {
		return nil, errors.New("di.console: repositories are nil")
	}
	if s == nil {
		return nil, errors.New("di.console: services are nil")
	}

	resources := newContainerResources()

	solanaClient, err := solanainfra.NewMintClient(ctx)
	if err != nil {
		return nil, resources.CloseWithError(err)
	}
	tokenUC := uc.NewTokenUsecase(solanaClient)

	if c.infra.PaymentMethodGateway == nil {
		var customerStore stripeadapter.PaymentMethodCustomerStore

		if value, ok := any(r.paymentMethodRepo).(stripeadapter.PaymentMethodCustomerStore); ok {
			customerStore = value
		} else if value, ok := any(r.userRepo).(stripeadapter.PaymentMethodCustomerStore); ok {
			customerStore = value
		}

		if customerStore == nil {
			return nil, resources.CloseWithError(errors.New(
				"di.console: PaymentMethodCustomerStore is not implemented by current repositories",
			))
		}

		if err := c.infra.RegisterPaymentMethodGatewayFromSecret(ctx, customerStore); err != nil {
			return nil, resources.CloseWithError(err)
		}
		if c.infra.PaymentMethodGateway == nil {
			return nil, resources.CloseWithError(errors.New(
				"di.console: stripe payment method gateway is nil after registration",
			))
		}
	}

	if c.infra.AccountGateway == nil {
		if err := c.infra.RegisterAccountGatewayFromSecret(ctx); err != nil {
			return nil, resources.CloseWithError(err)
		}
		if c.infra.AccountGateway == nil {
			return nil, resources.CloseWithError(errors.New(
				"di.console: stripe account gateway is nil after registration",
			))
		}
	}

	accountUC := uc.NewAccountUsecase(r.accountRepo, c.infra.AccountGateway)

	announcementAvatarRepo := fsrepo.NewAnnouncementAvatarRepositoryFS(c.fsClient)
	announcementAttachmentRepo := fsrepo.NewAnnouncementAttachmentRepositoryFS(c.fsClient)

	announcementAttachmentStorage, err := firebaseadp.NewAnnouncementAttachmentStorageFromEnv(ctx)
	if err != nil {
		return nil, resources.CloseWithError(err)
	}
	resources.Add("announcement attachment storage", announcementAttachmentStorage)

	announcementUC := uc.NewAnnouncementUsecase(
		r.announcementRepo,
		announcementAvatarRepo,
		announcementAttachmentRepo,
	).WithAttachmentStorage(announcementAttachmentStorage)

	brandWalletSvc := solanainfra.NewBrandWalletService(c.firestoreProjectID)
	avatarWalletSvc := solanainfra.NewAvatarWalletService(c.firestoreProjectID)
	avatarReviewRepo := fsrepo.NewAvatarReviewRepositoryFS(c.fsClient)

	avatarUC := uc.NewAvatarUsecase(
		r.avatarRepo,
		avatarReviewRepo,
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

	paymentUC := uc.NewPaymentUsecase(uc.NewPaymentUsecaseInput{
		PaymentRepo: r.paymentRepo,
		OrderRepo:   r.orderRepo,
		ResaleRepo:  r.resaleRepo,
	})

	if r.salesReceivableRepo == nil {
		return nil, resources.CloseWithError(errors.New("di.console: sales receivable repository is nil"))
	}
	salesReceivableUC := uc.NewSalesReceivableUsecase(r.salesReceivableRepo)
	if salesReceivableUC == nil {
		return nil, resources.CloseWithError(errors.New("di.console: sales receivable usecase is nil"))
	}

	settlementDependencies, err := shared.BuildSettlementDependencies(ctx, c.infra)
	if err != nil {
		return nil, resources.CloseWithError(err)
	}

	settlementUC, err := buildSettlementUsecase(r, salesReceivableUC, settlementDependencies)
	if err != nil {
		return nil, resources.CloseWithError(err)
	}

	refundUC, err := buildRefundUsecase(r, paymentUC, salesReceivableUC, settlementDependencies)
	if err != nil {
		return nil, resources.CloseWithError(err)
	}

	listSaveOperationStorage, err := firebaseadp.NewListSaveOperationStorageFromEnv(ctx)
	if err != nil {
		return nil, resources.CloseWithError(err)
	}
	resources.Add("list save operation storage", listSaveOperationStorage)

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

	listSaveOperationRetryQueue, err := cloudtasksadp.NewListSaveOperationQueueFromEnv(ctx)
	if err != nil {
		return nil, resources.CloseWithError(err)
	}
	resources.Add("list save operation retry queue", listSaveOperationRetryQueue)

	listSaveOperationUC := uc.NewListSaveOperationUsecase(
		uc.NewListSaveOperationUsecaseParams{
			ListRepository:      r.listRepoFS,
			ImageRepository:     r.listImageRecordRepo,
			OperationRepository: r.listSaveOperationRepo,
			Storage:             listSaveOperationStorage,
			RetryQueue:          listSaveOperationRetryQueue,
			CartItemCleanup:     r.cartRepo,
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
		r,
		orderUC,
		paymentUC,
		salesReceivableUC,
		settlementDependencies,
	)
	if err != nil {
		return nil, resources.CloseWithError(err)
	}

	if paymentUC == nil {
		return nil, resources.CloseWithError(errors.New("di.console: payment usecase is nil"))
	}
	if r.orderRepo == nil {
		return nil, resources.CloseWithError(errors.New("di.console: order repository is nil"))
	}
	if c.infra.PaymentMethodGateway == nil {
		return nil, resources.CloseWithError(errors.New("di.console: payment method gateway is nil"))
	}

	paymentFlowUC := uc.NewPaymentFlowUsecase(
		paymentUC,
		r.orderRepo,
		c.infra.PaymentMethodGateway,
	)
	if paymentFlowUC == nil {
		return nil, resources.CloseWithError(errors.New("di.console: payment flow usecase is nil"))
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
		return nil, resources.CloseWithError(err)
	}

	if mintTaskQueue == nil {
		return nil, resources.CloseWithError(errors.New("di.console: mint task queue is nil"))
	}

	resources.Add("mint task queue", mintTaskQueue)
	mintUC.SetMintTaskEnqueuer(mintTaskQueue)

	baseURL := os.Getenv("ARWEAVE_BASE_URL")
	apiKey := os.Getenv("IRYS_SERVICE_API_KEY")
	uploader := arweave.NewHTTPUploader(baseURL, apiKey)

	tbReviewRepo := fsrepo.NewTokenBlueprintReviewRepositoryFS(c.fsClient)

	tokenBlueprintAssetStorage, err := firebaseadp.NewTokenBlueprintAssetStorageFromEnv(ctx)
	if err != nil {
		return nil, resources.CloseWithError(err)
	}
	resources.Add("token blueprint asset storage", tokenBlueprintAssetStorage)

	tokenBlueprintUC := uc.NewTokenBlueprintUsecase(
		r.tokenBlueprintRepo,
		tbReviewRepo,
		tokenBlueprintAssetStorage,
		uploader,
	)

	tokenBlueprintCreateOperationQueue, err :=
		cloudtasksadp.NewTokenBlueprintCreateOperationQueueFromEnv(ctx)
	if err != nil {
		return nil, resources.CloseWithError(err)
	}
	resources.Add("token blueprint create operation queue", tokenBlueprintCreateOperationQueue)

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

	if r.productBlueprintReviewRepo == nil {
		return nil, resources.CloseWithError(errors.New("di.console: product blueprint review repository is nil"))
	}
	if r.productBlueprintRepo == nil {
		return nil, resources.CloseWithError(errors.New("di.console: product blueprint repository is nil"))
	}

	productBlueprintReviewUC := uc.NewProductBlueprintReviewUsecase(
		r.productBlueprintReviewRepo,
		r.productBlueprintRepo,
		r.brandRepo,
		r.memberRepo,
		walletUC,
		r.avatarRepo,
		nil,
	)
	if productBlueprintReviewUC == nil {
		return nil, resources.CloseWithError(errors.New("di.console: product blueprint review usecase is nil"))
	}

	reviewReportRepo := fsrepo.NewReportRepositoryFS(c.fsClient)
	if reviewReportRepo == nil {
		return nil, resources.CloseWithError(errors.New("di.console: review report repository is nil"))
	}

	reviewReportUC := uc.NewReportUsecase(
		uc.ReportUsecaseDeps{
			ReportRepo:               reviewReportRepo,
			DecisionNotificationRepo: r.reviewReportDecisionNotificationRepo,
			ProductReviewRepo:        r.productBlueprintReviewRepo,
			ProductBlueprintRepo:     r.productBlueprintRepo,
			ProductPurchaseResolver:  walletUC,
			ProductReviewModerator:   productBlueprintReviewUC,
			TokenCommentRepo:         tbReviewRepo.Comments(),
			TokenBlueprintRepo:       r.tokenBlueprintRepo,
			TokenAccessResolver:      walletUC,
			TokenCommentModerator:    tokenBlueprintReviewUC,
		},
	)
	if reviewReportUC == nil {
		return nil, resources.CloseWithError(errors.New("di.console: review report usecase is nil"))
	}

	cartUC := uc.NewCartUsecase(r.cartRepo)

	invitationDeliveryQueue, err :=
		cloudtasksadp.NewInvitationDeliveryQueueFromEnv(ctx)
	if err != nil {
		return nil, resources.CloseWithError(err)
	}
	resources.Add("invitation delivery queue", invitationDeliveryQueue)

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

	orderDispatchNotificationQueue, err :=
		cloudtasksadp.NewOrderDispatchNotificationQueueFromEnv(ctx)
	if err != nil {
		return nil, resources.CloseWithError(err)
	}
	resources.Add("order dispatch notification queue", orderDispatchNotificationQueue)

	authUserReader := firebaseadp.NewAuthUserReader(c.infra.FirebaseAuth)
	orderDispatchNotificationMailer :=
		mailadp.NewOrderDispatchNotificationMailerWithResend()

	orderDispatchNotificationUC := uc.NewOrderDispatchNotificationUsecase(
		r.orderDispatchNotificationRepo,
		authUserReader,
		uc.CompanyIDFromContext,
		r.productBlueprintRepo,
		orderDispatchNotificationMailer,
		orderDispatchNotificationQueue,
	)

	refundCompletionNotificationQueue, err :=
		cloudtasksadp.NewRefundCompletionNotificationQueueFromEnv(ctx)
	if err != nil {
		return nil, resources.CloseWithError(err)
	}
	resources.Add("refund completion notification queue", refundCompletionNotificationQueue)

	refundCompletionNotificationRepo :=
		fsrepo.NewRefundCompletionNotificationRepositoryFS(c.fsClient)

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
		strings.ToLower(strings.TrimSpace(c.firestoreProjectID)),
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
		resources:                       resources,
		solanaMintClient:                solanaClient,
		tokenUC:                         tokenUC,
		accountUC:                       accountUC,
		announcementUC:                  announcementUC,
		avatarUC:                        avatarUC,
		paymentMethodUC:                 paymentMethodUC,
		brandUC:                         brandUC,
		companyUC:                       companyUC,
		inquiryUC:                       inquiryUC,
		itemRefundUC:                    itemRefundUC,
		returnReceiptUC:                 returnReceiptUC,
		openedReturnReceiptUC:           openedReturnReceiptUC,
		inventoryUC:                     inventoryUC,
		listUC:                          listUC,
		listSaveOperationUC:             listSaveOperationUC,
		memberUC:                        memberUC,
		modelUC:                         modelUC,
		orderUC:                         orderUC,
		orderDispatchNotificationUC:     orderDispatchNotificationUC,
		refundCompletionNotificationUC:  refundCompletionNotificationUC,
		paymentUC:                       paymentUC,
		paymentFlowUC:                   paymentFlowUC,
		salesReceivableUC:               salesReceivableUC,
		settlementUC:                    settlementUC,
		refundUC:                        refundUC,
		permissionUC:                    permissionUC,
		printUC:                         printUC,
		productionUC:                    productionUC,
		productBlueprintUC:              productBlueprintUC,
		productBlueprintCategoryUC:      productBlueprintCategoryUC,
		inspectionUC:                    inspectionUC,
		mintUC:                          mintUC,
		shippingAddressUC:               shippingAddressUC,
		transportationUC:                transportationUC,
		tokenBlueprintUC:                tokenBlueprintUC,
		tokenBlueprintCreateOperationUC: tokenBlueprintCreateOperationUC,
		tokenBlueprintReviewUC:          tokenBlueprintReviewUC,
		productBlueprintReviewUC:        productBlueprintReviewUC,
		reviewReportUC:                  reviewReportUC,
		userUC:                          userUC,
		walletUC:                        walletUC,
		cartUC:                          cartUC,
		invitationUC:                    invitationUC,
		invitationDeliveryUC:            invitationDeliveryUC,
		authBootstrapSvc:                authBootstrapSvc,
	}, nil
}
