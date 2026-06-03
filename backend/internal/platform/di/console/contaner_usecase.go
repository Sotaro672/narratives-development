// backend/internal/platform/di/console/contaner_usecase.go
package console

import (
	"os"

	fsrepo "narratives/internal/adapters/out/firestore"
	mailadp "narratives/internal/adapters/out/mail"
	uc "narratives/internal/application/usecase"
	memdom "narratives/internal/domain/member"
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

	invitationQueryUC    uc.InvitationQueryPort
	invitationCommandUC  uc.InvitationCommandPort
	invitationCompleteUC uc.InvitationCompletePort

	authBootstrapSvc *uc.BootstrapService
}

func buildUsecases(c *clients, r *repos, s *services, res *resolvers) *usecases {
	var tokenUC *uc.TokenUsecase
	if c.infra.MintAuthorityKey != nil {
		solanaClient := solanainfra.NewMintClient(c.infra.MintAuthorityKey)
		tokenUC = uc.NewTokenUsecase(solanaClient)
	} else {
		tokenUC = uc.NewTokenUsecase(nil)
	}

	accountUC := uc.NewAccountUsecase(r.accountRepo)
	announcementUC := uc.NewAnnouncementUsecase(r.announcementRepo, nil, nil)

	brandWalletSvc := solanainfra.NewBrandWalletService(c.firestoreProjectID)
	avatarWalletSvc := solanainfra.NewAvatarWalletService(c.firestoreProjectID)

	avatarUC := uc.NewAvatarUsecase(
		r.avatarRepo,
		r.avatarStateRepo,
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
	inquiryUC := uc.NewInquiryUsecase(r.inquiryRepo, nil, nil)

	inventoryUC := uc.NewInventoryUsecase(r.inventoryRepo)
	if r.productRepo != nil {
		if resolver, ok := any(r.productRepo).(uc.ProductModelResolver); ok {
			inventoryUC.WithProductModelResolver(resolver)
		}
	}

	paymentUC := uc.NewPaymentUsecase(r.paymentRepo)

	listUC := uc.NewListUsecase(
		r.listRepoFS,
		r.listImageRecordRepo,
	)

	modelUC := uc.NewModelUsecase(r.modelRepo)

	orderUC := uc.NewOrderUsecase(r.orderRepo, r.cartRepo)

	permissionUC := uc.NewPermissionUsecase(r.permissionRepo)

	printUC := uc.NewPrintUsecase(
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

	baseURL := os.Getenv("ARWEAVE_BASE_URL")
	apiKey := os.Getenv("IRYS_SERVICE_API_KEY")
	uploader := arweave.NewHTTPUploader(baseURL, apiKey)

	tbReviewRepo := fsrepo.NewTokenBlueprintReviewRepositoryFS(c.fsClient)

	tokenBlueprintUC := uc.NewTokenBlueprintUsecase(
		r.tokenBlueprintRepo,
		tbReviewRepo,
		uploader,
	)

	mintUC.SetTokenBlueprintMetadataEnsurer(tokenBlueprintUC)

	shippingAddressUC := uc.NewShippingAddressUsecase(r.shippingAddressRepo)

	tokenBlueprintReviewUC := uc.NewTokenBlueprintReviewUsecase(
		tbReviewRepo,
		r.avatarRepo,
		r.tokenBlueprintRepo,
		r.brandRepo,
	)

	userUC := uc.NewUserUsecase(r.userRepo)

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

	invitationMailer := mailadp.NewInvitationMailerWithResend(s.companySvc, r.brandRepo)

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

	memberUC := uc.NewMemberUsecase(
		r.memberRepo,
		invitationCommandUC,
	)

	authBootstrapSvc := &uc.BootstrapService{
		Members:   r.memberRepo,
		Companies: r.companyRepo,
	}

	_ = res // reserved for future use; keeps signature stable

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

			memberSvc := memdom.NewService(r.memberRepo)

			return uc.NewProductBlueprintReviewUsecase(
				r.productBlueprintReviewRepo,
				r.walletRepo,
				r.productBlueprintRepo,
				r.brandRepo,
				memberSvc,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
			)
		}(),

		userUC:   userUC,
		walletUC: walletUC,
		cartUC:   cartUC,

		invitationQueryUC:    invitationQueryUC,
		invitationCommandUC:  invitationCommandUC,
		invitationCompleteUC: invitationCompleteUC,

		authBootstrapSvc: authBootstrapSvc,
	}
}
