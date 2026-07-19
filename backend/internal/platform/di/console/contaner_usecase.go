// backend/internal/platform/di/console/contaner_usecase.go
package console

import (
	"context"
	"os"

	fsrepo "narratives/internal/adapters/out/firestore"
	cloudtasksadp "narratives/internal/adapters/out/firestore/cloudtasks"
	mailadp "narratives/internal/adapters/out/mail"
	uc "narratives/internal/application/usecase"
	"narratives/internal/infra/arweave"
	solanainfra "narratives/internal/infra/solana"
)

type usecases struct {
	tokenUC *uc.TokenUsecase

	accountUC       *uc.AccountUsecase
	announcementUC  *uc.AnnouncementUsecase
	avatarUC        *uc.AvatarUsecase
	paymentMethodUC *uc.PaymentMethodUsecase
	brandUC         *uc.BrandUsecase
	companyUC       *uc.CompanyUsecase
	inquiryUC       *uc.InquiryUsecase
	inventoryUC     *uc.InventoryUsecase
	listUC          *uc.ListUsecase
	memberUC        *uc.MemberUsecase
	modelUC         *uc.ModelUsecase
	orderUC         *uc.OrderUsecase
	paymentUC       *uc.PaymentUsecase
	permissionUC    *uc.PermissionUsecase
	printUC         *uc.PrintUsecase

	productionUC               *uc.ProductionUsecase
	productBlueprintUC         *uc.ProductBlueprintUsecase
	productBlueprintCategoryUC *uc.ProductBlueprintCategoryUsecase

	inspectionUC *uc.InspectionUsecase
	mintUC       *uc.MintUsecase

	shippingAddressUC *uc.ShippingAddressUsecase

	tokenBlueprintUC *uc.TokenBlueprintUsecase

	tokenBlueprintReviewUC *uc.TokenBlueprintReviewUsecase

	productBlueprintReviewUC *uc.ProductBlueprintReviewUsecase

	userUC   *uc.UserUsecase
	walletUC *uc.WalletUsecase
	cartUC   *uc.CartUsecase

	invitationUC uc.InvitationUsecasePort

	authBootstrapSvc *uc.BootstrapService
}

func buildUsecases(
	c *clients,
	r *repos,
	s *services,
	res *resolvers,
) *usecases {
	var tokenUC *uc.TokenUsecase
	if c.infra.MintAuthorityKey != nil {
		solanaClient := solanainfra.NewMintClient(
			c.infra.MintAuthorityKey,
		)
		tokenUC = uc.NewTokenUsecase(solanaClient)
	} else {
		tokenUC = uc.NewTokenUsecase(nil)
	}

	accountUC := uc.NewAccountUsecase(r.accountRepo)

	announcementAvatarRepo :=
		fsrepo.NewAnnouncementAvatarRepositoryFS(c.fsClient)
	announcementAttachmentRepo :=
		fsrepo.NewAnnouncementAttachmentRepositoryFS(c.fsClient)

	announcementUC := uc.NewAnnouncementUsecase(
		r.announcementRepo,
		announcementAvatarRepo,
		announcementAttachmentRepo,
	)

	brandWalletSvc :=
		solanainfra.NewBrandWalletService(
			c.firestoreProjectID,
		)
	avatarWalletSvc :=
		solanainfra.NewAvatarWalletService(
			c.firestoreProjectID,
		)

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
		uc.WithBrandWalletService(brandWalletSvc),
	)

	companyUC := uc.NewCompanyUsecase(r.companyRepo)

	inquiryReplyRepo :=
		fsrepo.NewInquiryReplyRepositoryFS(c.fsClient)

	inquiryUC := uc.NewInquiryUsecase(
		r.inquiryRepo,
		inquiryReplyRepo,
		nil,
		"",
		"",
		nil,
		nil,
	)

	inventoryUC := uc.NewInventoryUsecase(
		r.inventoryRepo,
	)
	if r.productRepo != nil {
		if resolver, ok :=
			any(r.productRepo).(uc.ProductModelResolver); ok {
			inventoryUC.WithProductModelResolver(resolver)
		}
	}

	paymentUC := uc.NewPaymentUsecase(
		uc.NewPaymentUsecaseInput{
			PaymentRepo: r.paymentRepo,
		},
	)

	listUC := uc.NewListUsecase(
		r.listRepoFS,
		r.listImageRecordRepo,
	)

	modelUC := uc.NewModelUsecase(
		r.modelRepo,
		r.productBlueprintRepo,
	)

	orderUC := uc.NewOrderUsecase(
		r.orderRepo,
		r.listRepoFS,
		r.inventoryRepo,
		r.resaleRepo,
		r.paymentMethodRepo,
	)

	permissionUC := uc.NewPermissionUsecase(
		r.permissionRepo,
	)

	printUC := uc.NewPrintUsecase(
		r.productionRepo,
		r.productRepo,
		r.printLogRepo,
		r.inspectionRepo,
		r.productBlueprintRepo,
	)

	productionUC := uc.NewProductionUsecase(
		r.productionRepo,
	)

	productBlueprintUC := uc.NewProductBlueprintUsecase(
		r.productBlueprintRepo,
		r.productBlueprintReviewRepo,
	)

	productBlueprintCategoryUC :=
		uc.NewProductBlueprintCategoryUsecase(
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

	// 1件ずつmintするための task repository / token保存recorder を注入します。
	//
	// r.mintRepo は firestore.MintRepositoryFS を想定しており、以下を実装しています。
	// - mint.MintRepository
	// - mint.MintProductTaskRepository
	// - usecase.MintRequestPort
	// - usecase.MintProductMintRecorder
	//
	// これが未注入だと、UpdateRequestInfo内で
	// "mint task repo is nil" となり、productId単位のtaskを作成できません。
	mintUC.SetMintTaskRepository(r.mintRepo)
	mintUC.SetMintProductMintRecorder(r.mintRepo)

	// Cloud Tasksへ「次の1件mint処理」を投入するenqueuerを注入します。
	//
	// 必要環境変数:
	// - CLOUD_TASKS_PROJECT_ID
	// - CLOUD_TASKS_LOCATION
	// - CLOUD_TASKS_QUEUE_ID
	// - INTERNAL_BASE_URL
	// - CLOUD_TASKS_SERVICE_ACCOUNT
	//
	// ここが未注入だと、mint request / product tasks はQUEUEDになっても
	// 自動で /internal/mint/tasks/{mintID}/execute が呼ばれません。
	if mintTaskQueue, err :=
		cloudtasksadp.NewMintTaskQueueFromEnv(
			context.Background(),
		); err == nil && mintTaskQueue != nil {
		mintUC.SetMintTaskEnqueuer(mintTaskQueue)
	}

	baseURL := os.Getenv("ARWEAVE_BASE_URL")
	apiKey := os.Getenv("IRYS_SERVICE_API_KEY")
	uploader := arweave.NewHTTPUploader(
		baseURL,
		apiKey,
	)

	tbReviewRepo :=
		fsrepo.NewTokenBlueprintReviewRepositoryFS(
			c.fsClient,
		)

	tokenBlueprintUC := uc.NewTokenBlueprintUsecase(
		r.tokenBlueprintRepo,
		tbReviewRepo,
		uploader,
	)

	mintUC.SetTokenBlueprintMetadataEnsurer(
		tokenBlueprintUC,
	)
	mintUC.SetTokenBlueprintMintMarker(
		tokenBlueprintUC,
	)

	shippingAddressUC :=
		uc.NewShippingAddressUsecase(
			r.shippingAddressRepo,
		)

	tokenBlueprintReviewUC :=
		uc.NewTokenBlueprintReviewUsecase(
			tbReviewRepo,
			r.avatarRepo,
			r.tokenBlueprintRepo,
			r.brandRepo,
		)

	userUC := uc.NewUserUsecase(r.userRepo, nil)

	onchainReader :=
		solanainfra.NewOnchainWalletReaderDevnet()
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

	invitationMailer :=
		mailadp.NewInvitationMailerWithResend(
			r.companyRepo,
			r.brandRepo,
		)

	invitationUC := uc.NewInvitationUsecase(
		r.invitationTokenRepo,
		r.memberRepo,
		invitationMailer,
	)

	memberUC := uc.NewMemberUsecase(
		r.memberRepo,
		invitationUC,
	)

	authBootstrapSvc := &uc.BootstrapService{
		Members:   r.memberRepo,
		Companies: r.companyRepo,
	}

	_ = s
	_ = res

	return &usecases{
		tokenUC: tokenUC,

		accountUC:       accountUC,
		announcementUC:  announcementUC,
		avatarUC:        avatarUC,
		paymentMethodUC: paymentMethodUC,
		brandUC:         brandUC,
		companyUC:       companyUC,
		inquiryUC:       inquiryUC,
		inventoryUC:     inventoryUC,
		listUC:          listUC,
		memberUC:        memberUC,
		modelUC:         modelUC,
		orderUC:         orderUC,
		paymentUC:       paymentUC,
		permissionUC:    permissionUC,
		printUC:         printUC,

		productionUC:               productionUC,
		productBlueprintUC:         productBlueprintUC,
		productBlueprintCategoryUC: productBlueprintCategoryUC,

		inspectionUC: inspectionUC,
		mintUC:       mintUC,

		shippingAddressUC: shippingAddressUC,

		tokenBlueprintUC:       tokenBlueprintUC,
		tokenBlueprintReviewUC: tokenBlueprintReviewUC,

		productBlueprintReviewUC: func() *uc.ProductBlueprintReviewUsecase {
			if r.productBlueprintReviewRepo == nil ||
				r.productBlueprintRepo == nil ||
				r.walletRepo == nil {
				return nil
			}

			return uc.NewProductBlueprintReviewUsecase(
				r.productBlueprintReviewRepo,
				r.walletRepo,
				r.productBlueprintRepo,
				r.brandRepo,
				r.memberRepo,
				onchainReader,
				tokenQuery,
				r.productRepo,
				r.productBlueprintRepo,
				r.avatarRepo,
				nil,
			)
		}(),

		userUC:   userUC,
		walletUC: walletUC,
		cartUC:   cartUC,

		invitationUC: invitationUC,

		authBootstrapSvc: authBootstrapSvc,
	}
}
