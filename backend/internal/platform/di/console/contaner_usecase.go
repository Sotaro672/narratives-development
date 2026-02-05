// backend/internal/platform/di/console/container_usecases.go
package console

import (
	"os"
	"strings"

	// ✅ for unwrapping ListRepositoryForUsecase -> ListRepositoryFS
	fsrepo "narratives/internal/adapters/out/firestore"

	mailadp "narratives/internal/adapters/out/mail"
	solanainfra "narratives/internal/infra/solana"

	inspectionapp "narratives/internal/application/inspection"
	mintapp "narratives/internal/application/mint"
	pbuc "narratives/internal/application/productBlueprint/usecase"
	productionapp "narratives/internal/application/production"
	tokenblueprintapp "narratives/internal/application/tokenBlueprint"

	uc "narratives/internal/application/usecase"
	authuc "narratives/internal/application/usecase/auth"

	// ✅ moved: ListUsecase is now in subpackage usecase/list
	listuc "narratives/internal/application/usecase/list"

	"narratives/internal/infra/arweave"
)

type usecases struct {
	tokenUC *uc.TokenUsecase

	accountUC        *uc.AccountUsecase
	announcementUC   *uc.AnnouncementUsecase
	avatarUC         *uc.AvatarUsecase
	billingAddressUC *uc.BillingAddressUsecase
	brandUC          *uc.BrandUsecase
	campaignUC       *uc.CampaignUsecase
	companyUC        *uc.CompanyUsecase
	inquiryUC        *uc.InquiryUsecase
	inventoryUC      *uc.InventoryUsecase
	invoiceUC        *uc.InvoiceUsecase
	listUC           *listuc.ListUsecase // ✅ moved
	memberUC         *uc.MemberUsecase
	messageUC        *uc.MessageUsecase
	modelUC          *uc.ModelUsecase
	orderUC          *uc.OrderUsecase
	paymentUC        *uc.PaymentUsecase
	permissionUC     *uc.PermissionUsecase
	printUC          *uc.PrintUsecase

	productionUC       *productionapp.ProductionUsecase
	productBlueprintUC *pbuc.ProductBlueprintUsecase

	inspectionUC *inspectionapp.InspectionUsecase
	productUC    *uc.ProductUsecase
	mintUC       *mintapp.MintUsecase

	shippingAddressUC *uc.ShippingAddressUsecase

	tokenBlueprintUC      *tokenblueprintapp.TokenBlueprintUsecase
	tokenBlueprintQueryUC *tokenblueprintapp.TokenBlueprintQueryUsecase

	tokenOperationUC *uc.TokenOperationUsecase
	trackingUC       *uc.TrackingUsecase
	userUC           *uc.UserUsecase
	walletUC         *uc.WalletUsecase
	cartUC           *uc.CartUsecase

	invitationQueryUC   uc.InvitationQueryPort
	invitationCommandUC uc.InvitationCommandPort

	authBootstrapSvc *authuc.BootstrapService
}

func buildUsecases(c *clients, r *repos, s *services, res *resolvers) *usecases {
	// =========================================================
	// TokenUsecase
	// =========================================================
	var tokenUC *uc.TokenUsecase
	if c.infra.MintAuthorityKey != nil {
		solanaClient := solanainfra.NewMintClient(c.infra.MintAuthorityKey)
		tokenUC = uc.NewTokenUsecase(solanaClient, r.mintRequestPort)
	} else {
		tokenUC = uc.NewTokenUsecase(nil, r.mintRequestPort)
	}

	// =========================================================
	// Core usecases
	// =========================================================
	accountUC := uc.NewAccountUsecase(r.accountRepo)
	announcementUC := uc.NewAnnouncementUsecase(r.announcementRepo, nil, nil)

	// ★ここが cfg.FirestoreProjectID ではなく clients.firestoreProjectID
	brandWalletSvc := solanainfra.NewBrandWalletService(c.firestoreProjectID)
	avatarWalletSvc := solanainfra.NewAvatarWalletService(c.firestoreProjectID)

	avatarUC := uc.NewAvatarUsecase(
		r.avatarRepo,
		r.avatarStateRepo,
		r.avatarIconRepo,
		r.avatarIconRepo,
	).
		WithWalletService(avatarWalletSvc).
		WithWalletRepo(r.walletRepo)

	callOptionalMethod(avatarUC, "WithCartRepo", r.cartRepo)

	billingAddressUC := uc.NewBillingAddressUsecase(r.billingAddressRepo)
	brandUC := uc.NewBrandUsecaseWithWallet(r.brandRepo, r.memberRepo, brandWalletSvc)
	campaignUC := uc.NewCampaignUsecase(r.campaignRepo, nil, nil, nil)
	companyUC := uc.NewCompanyUsecase(r.companyRepo)
	inquiryUC := uc.NewInquiryUsecase(r.inquiryRepo, nil, nil)

	inventoryUC := uc.NewInventoryUsecase(r.inventoryRepoForUC)
	paymentUC := uc.NewPaymentUsecase(r.paymentRepo)
	invoiceUC := uc.NewInvoiceUsecase(r.invoiceRepo)

	// =========================================================
	// ListUsecase (✅ NewListUsecase だけを唯一の入口にする)
	// =========================================================
	// r.listRepo が *firestore.ListRepositoryForUsecase の場合、
	// 埋め込み元の *firestore.ListRepositoryFS を取り出して listReader/listCreator に渡す。
	// （ListRepositoryForUsecase は同名 Update(ctx,item) を持つため Update(ctx,id,patch) が昇格せず、
	//  getPatchUpdater() が失敗して readableId が永続化されない）
	var listReader listuc.ListReader = r.listRepo
	var listCreator listuc.ListCreator = r.listRepo

	if w, ok := any(r.listRepo).(*fsrepo.ListRepositoryForUsecase); ok && w != nil && w.ListRepositoryFS != nil {
		listReader = w.ListRepositoryFS
		listCreator = w.ListRepositoryFS
	}

	// ✅ ONLY ENTRYPOINT: 全配線は constructors.go(NewListUsecase) に集約
	// 重要:
	// - imageReader / imageByIDReader は Firestore (/lists/{listId}/images) を渡す
	// - imageObjectSaver は GCS (signed-url / SaveFromBucketObject / DeleteObject) を渡す
	// これにより DeleteImage が listImageRecordRepo=nil で ErrNotSupported -> 501 になるのを防ぐ
	listUC := listuc.NewListUsecase(
		listReader,
		listCreator,
		r.listPatcher,

		r.listImageRecordRepo, // ✅ Firestore: record repo (list images subcollection)
		r.listImageRecordRepo, // ✅ Firestore: record by id
		r.listImageRepo,       // ✅ GCS: object saver + signed url issuer (and optional object deleter)
	)

	// ❌ 以下は禁止（入口が複数になるため撤去）
	// - listUC.WithListImageRecordRepo(...)
	// - listUC.WithListImageDeleter(...)
	// - その他 WithXxx 系の後付け DI

	messageUC := uc.NewMessageUsecase(r.messageRepo, nil, nil)
	modelUC := uc.NewModelUsecase(r.modelRepo, r.modelHistoryRepo)
	orderUC := uc.NewOrderUsecase(r.orderRepo)
	permissionUC := uc.NewPermissionUsecase(r.permissionRepo)

	printUC := uc.NewPrintUsecase(
		r.productRepo,
		r.printLogRepo,
		r.inspectionRepo,
		r.productBlueprintRepo,
		res.nameResolver,
	)

	productionUC := productionapp.NewProductionUsecase(
		r.productionRepo,
		s.pbSvc,
		res.nameResolver,
	)

	productBlueprintUC := pbuc.NewProductBlueprintUsecase(
		r.productBlueprintRepo,
		r.productBlueprintHistoryRepo,
	)

	inspectionProductRepo := &inspectionProductRepoAdapter{repo: r.productRepo}
	inspectionUC := inspectionapp.NewInspectionUsecase(
		r.inspectionRepo,
		inspectionProductRepo,
	)

	productQueryRepo := &productQueryRepoAdapter{
		productRepo:          r.productRepo,
		modelRepo:            r.modelRepo,
		productionRepo:       r.productionRepo,
		productBlueprintRepo: r.productBlueprintRepo,
	}
	productUC := uc.NewProductUsecase(productQueryRepo, s.brandSvc, s.companySvc)

	mintUC := mintapp.NewMintUsecase(
		r.productBlueprintRepo,
		r.productionRepo,
		r.inspectionRepo,
		r.modelRepo,
		r.tokenBlueprintRepo,
		s.brandSvc,
		r.mintRepo,
		r.inspectionRepo,
		tokenUC,
	)
	mintUC.SetNameResolver(res.nameResolver)
	mintUC.SetInventoryUsecase(inventoryUC)

	// TokenBlueprint Ensurers
	tbBucketEnsurer := tokenblueprintapp.NewTokenBlueprintBucketUsecase(c.gcsClient)
	mintUC.SetTokenBlueprintBucketEnsurer(tbBucketEnsurer)

	baseURL := strings.TrimSpace(os.Getenv("ARWEAVE_BASE_URL"))
	apiKey := strings.TrimSpace(os.Getenv("IRYS_SERVICE_API_KEY"))
	uploader := arweave.NewHTTPUploader(baseURL, apiKey)

	tbMetadataUC := tokenblueprintapp.NewTokenBlueprintMetadataUsecase(r.tokenBlueprintRepo, uploader)
	mintUC.SetTokenBlueprintMetadataEnsurer(tbMetadataUC)

	shippingAddressUC := uc.NewShippingAddressUsecase(r.shippingAddressRepo)

	tokenBlueprintUC := tokenblueprintapp.NewTokenBlueprintUsecase(
		r.tokenBlueprintRepo,
		res.nameResolver,
		c.gcsClient,
	)

	tokenBlueprintQueryUC := tokenblueprintapp.NewTokenBlueprintQueryUsecase(
		r.tokenBlueprintRepo,
		res.nameResolver,
	)

	tokenOperationUC := uc.NewTokenOperationUsecase(r.tokenOperationRepo)
	trackingUC := uc.NewTrackingUsecase(r.trackingRepo)
	userUC := uc.NewUserUsecase(r.userRepo)
	walletUC := uc.NewWalletUsecase(r.walletRepo)
	cartUC := uc.NewCartUsecase(r.cartRepo)

	// Invitation
	invitationMailer := mailadp.NewInvitationMailerWithSendGrid(s.companySvc, s.brandSvc)
	invitationQueryUC := uc.NewInvitationService(r.invitationTokenUCRepo, r.memberRepo)
	invitationCommandUC := uc.NewInvitationCommandService(
		r.invitationTokenUCRepo,
		r.memberRepo,
		invitationMailer,
	)

	authBootstrapSvc := &authuc.BootstrapService{
		Members: &authMemberRepoAdapter{repo: r.memberRepo},
		Companies: &authCompanyRepoAdapter{
			repo: r.companyRepo,
		},
	}

	return &usecases{
		tokenUC: tokenUC,

		accountUC:        accountUC,
		announcementUC:   announcementUC,
		avatarUC:         avatarUC,
		billingAddressUC: billingAddressUC,
		brandUC:          brandUC,
		campaignUC:       campaignUC,
		companyUC:        companyUC,
		inquiryUC:        inquiryUC,
		inventoryUC:      inventoryUC,
		invoiceUC:        invoiceUC,
		listUC:           listUC,
		memberUC:         uc.NewMemberUsecase(r.memberRepo),
		messageUC:        messageUC,
		modelUC:          modelUC,
		orderUC:          orderUC,
		paymentUC:        paymentUC,
		permissionUC:     permissionUC,
		printUC:          printUC,

		productionUC:       productionUC,
		productBlueprintUC: productBlueprintUC,

		inspectionUC: inspectionUC,
		productUC:    productUC,
		mintUC:       mintUC,

		shippingAddressUC: shippingAddressUC,

		tokenBlueprintUC:      tokenBlueprintUC,
		tokenBlueprintQueryUC: tokenBlueprintQueryUC,

		tokenOperationUC: tokenOperationUC,
		trackingUC:       trackingUC,
		userUC:           userUC,
		walletUC:         walletUC,
		cartUC:           cartUC,

		invitationQueryUC:   invitationQueryUC,
		invitationCommandUC: invitationCommandUC,

		authBootstrapSvc: authBootstrapSvc,
	}
}
