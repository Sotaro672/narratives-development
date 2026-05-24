// backend/internal/platform/di/console/contaner_usecase.go
package console

import (
	"os"

	fsrepo "narratives/internal/adapters/out/firestore"
	mailadp "narratives/internal/adapters/out/mail"
	inspectionapp "narratives/internal/application/inspection"
	mintapp "narratives/internal/application/mint"
	productionapp "narratives/internal/application/production"
	tokenblueprintapp "narratives/internal/application/tokenBlueprint"
	uc "narratives/internal/application/usecase"
	authuc "narratives/internal/application/usecase/auth"
	avataruc "narratives/internal/application/usecase/avatar"
	listuc "narratives/internal/application/usecase/list"
	memdom "narratives/internal/domain/member"
	"narratives/internal/infra/arweave"
	solanainfra "narratives/internal/infra/solana"
)

type usecases struct {
	tokenUC *uc.TokenUsecase

	accountUC       *uc.AccountUsecase
	announcementUC  *uc.AnnouncementUsecase
	avatarUC        *avataruc.AvatarUsecase
	paymentMethodUC *uc.PaymentMethodUsecase
	brandUC         *uc.BrandUsecase
	companyUC       *uc.CompanyUsecase
	inquiryUC       *uc.InquiryUsecase
	inventoryUC     *uc.InventoryUsecase
	listUC          *listuc.ListUsecase
	memberUC        *uc.MemberUsecase
	modelUC         *uc.ModelUsecase
	orderUC         *uc.OrderUsecase
	paymentUC       *uc.PaymentUsecase
	permissionUC    *uc.PermissionUsecase
	printUC         *uc.PrintUsecase

	productionUC               *productionapp.ProductionUsecase
	productBlueprintUC         *uc.ProductBlueprintUsecase
	productBlueprintCategoryUC *uc.ProductBlueprintCategoryUsecase

	inspectionUC *inspectionapp.InspectionUsecase
	productUC    *uc.ProductUsecase
	mintUC       *mintapp.MintUsecase

	shippingAddressUC *uc.ShippingAddressUsecase

	tokenBlueprintUC      *tokenblueprintapp.TokenBlueprintUsecase
	tokenBlueprintQueryUC *tokenblueprintapp.TokenBlueprintQueryUsecase

	tokenBlueprintReviewUC *uc.TokenBlueprintReviewUsecase

	productBlueprintReviewUC *uc.ProductBlueprintReviewUsecase

	userUC   *uc.UserUsecase
	walletUC *uc.WalletUsecase
	cartUC   *uc.CartUsecase

	invitationQueryUC    uc.InvitationQueryPort
	invitationCommandUC  uc.InvitationCommandPort
	invitationCompleteUC uc.InvitationCompletePort

	authBootstrapSvc *authuc.BootstrapService
}

func buildUsecases(c *clients, r *repos, s *services, res *resolvers) *usecases {
	var tokenUC *uc.TokenUsecase
	if c.infra.MintAuthorityKey != nil {
		solanaClient := solanainfra.NewMintClient(c.infra.MintAuthorityKey)
		tokenUC = uc.NewTokenUsecase(solanaClient, r.mintRequestPort)
	} else {
		tokenUC = uc.NewTokenUsecase(nil, r.mintRequestPort)
	}

	accountUC := uc.NewAccountUsecase(r.accountRepo)
	announcementUC := uc.NewAnnouncementUsecase(r.announcementRepo, nil, nil)

	brandWalletSvc := solanainfra.NewBrandWalletService(c.firestoreProjectID)
	avatarWalletSvc := solanainfra.NewAvatarWalletService(c.firestoreProjectID)

	avatarUC := avataruc.NewAvatarUsecase(
		r.avatarRepo,
		r.avatarStateRepo,
	).
		WithWalletService(avatarWalletSvc).
		WithWalletRepo(r.walletRepo).
		WithCartRepo(r.cartRepo)

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
	inquiryUC := uc.NewInquiryUsecase(r.inquiryRepo, nil, nil)

	inventoryUC := uc.NewInventoryUsecase(r.inventoryRepo)
	paymentUC := uc.NewPaymentUsecase(r.paymentRepo)

	var listReader listuc.ListReader = r.listRepoFS
	var listCreator listuc.ListCreator = r.listRepoFS

	var listPatcher listuc.ListPatcher
	if r.listRepoFS != nil {
		listPatcher = r.listRepoFS
	} else {
		listPatcher = nil
	}

	// Firebase Storage 移行後:
	// - frontend が Firebase Storage へ直接 upload する
	// - backend は GCS signed URL / GCS object / bucket を扱わない
	// - list image は Firestore record repository のみを usecase に渡す
	listUC := listuc.NewListUsecase(
		listReader,
		listCreator,
		listPatcher,
		r.listImageRecordRepo,
		r.listImageRecordRepo,
	)

	modelUC := uc.NewModelUsecase(r.modelRepo)

	orderUC := uc.NewOrderUsecase(r.orderRepo, r.cartRepo)

	permissionUC := uc.NewPermissionUsecase(r.permissionRepo)

	printUC := uc.NewPrintUsecase(
		r.productRepo,
		r.printLogRepo,
		r.inspectionRepo,
		res.nameResolver,
		r.productBlueprintRepo,
	)

	productionUC := productionapp.NewProductionUsecase(
		r.productionRepo,
		s.pbSvc,
		res.nameResolver,
	)

	productBlueprintUC := uc.NewProductBlueprintUsecase(
		r.productBlueprintRepo,
		r.productBlueprintReviewRepo,
	)

	productBlueprintCategoryUC := uc.NewProductBlueprintCategoryUsecase(
		r.productBlueprintCategoryRepo,
	)

	inspectionUC := inspectionapp.NewInspectionUsecase(
		r.inspectionRepo,
		r.productRepo,
		r.mintRepo,
		r.modelRepo,
	)

	productUC := uc.NewProductUsecase(
		r.productRepo,
		r.modelRepo,
		r.productionRepo,
		r.productBlueprintRepo,
		s.brandSvc,
		s.companySvc,
	)

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

	// GCS bucket ensurer は廃止。
	// tokenBlueprint icon / contents は frontend が Firebase Storage へ直接 upload し、
	// backend は Firestore に保存された downloadURL / objectPath を扱う。

	baseURL := os.Getenv("ARWEAVE_BASE_URL")
	apiKey := os.Getenv("IRYS_SERVICE_API_KEY")
	uploader := arweave.NewHTTPUploader(baseURL, apiKey)

	tbMetadataUC := tokenblueprintapp.NewTokenBlueprintMetadataUsecase(
		r.tokenBlueprintRepo,
		uploader,
	)
	mintUC.SetTokenBlueprintMetadataEnsurer(tbMetadataUC)

	shippingAddressUC := uc.NewShippingAddressUsecase(r.shippingAddressRepo)

	tbReviewRepo := fsrepo.NewTokenBlueprintReviewRepositoryFS(c.fsClient)

	tokenBlueprintUC := tokenblueprintapp.NewTokenBlueprintUsecase(
		r.tokenBlueprintRepo,
		tbReviewRepo,
		res.nameResolver,
	)

	tokenBlueprintQueryUC := tokenblueprintapp.NewTokenBlueprintQueryUsecase(
		r.tokenBlueprintRepo,
		res.nameResolver,
	)

	tokenBlueprintReviewUC := uc.NewTokenBlueprintReviewUsecase(
		tbReviewRepo,
		r.avatarRepo,
		r.tokenBlueprintRepo,
		s.brandSvc,
	)

	var productBlueprintReviewUC *uc.ProductBlueprintReviewUsecase
	if r.productBlueprintReviewRepo != nil && r.productBlueprintRepo != nil && r.walletRepo != nil {
		memberSvc := memdom.NewService(r.memberRepo)

		productBlueprintReviewUC = uc.NewProductBlueprintReviewUsecase(
			r.productBlueprintReviewRepo,
			r.walletRepo,
		).
			WithProductBlueprintRepo(r.productBlueprintRepo).
			WithBrandService(s.brandSvc).
			WithMemberService(memberSvc)
	}

	userUC := uc.NewUserUsecase(r.userRepo)
	walletUC := uc.NewWalletUsecase(r.walletRepo)
	cartUC := uc.NewCartUsecase(r.cartRepo)

	invitationMailer := mailadp.NewInvitationMailerWithResend(s.companySvc, s.brandSvc)

	invitationQueryUC := uc.NewInvitationService(
		r.invitationTokenRepo,
		r.memberRepo,
	)

	invitationCommandUC := uc.NewInvitationCommandService(
		r.invitationTokenRepo,
		r.memberRepo,
		invitationMailer,
	)

	invitationCompleteUC := uc.NewInvitationCompleteService(
		r.invitationTokenRepo,
		r.memberRepo,
	)

	memberUC := uc.NewMemberUsecaseWithInvitationCommand(
		r.memberRepo,
		invitationCommandUC,
	)

	authBootstrapSvc := &authuc.BootstrapService{
		Members: &authMemberRepoAdapter{
			repo: r.memberRepo,
		},
		Companies: &authCompanyRepoAdapter{
			repo: r.companyRepo,
		},
	}

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
		productUC:    productUC,
		mintUC:       mintUC,

		shippingAddressUC: shippingAddressUC,

		tokenBlueprintUC:         tokenBlueprintUC,
		tokenBlueprintQueryUC:    tokenBlueprintQueryUC,
		tokenBlueprintReviewUC:   tokenBlueprintReviewUC,
		productBlueprintReviewUC: productBlueprintReviewUC,

		userUC:   userUC,
		walletUC: walletUC,
		cartUC:   cartUC,

		invitationQueryUC:    invitationQueryUC,
		invitationCommandUC:  invitationCommandUC,
		invitationCompleteUC: invitationCompleteUC,

		authBootstrapSvc: authBootstrapSvc,
	}
}
