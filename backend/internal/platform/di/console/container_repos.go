// backend/internal/platform/di/console/container_repos.go
package console

import (
	fs "narratives/internal/adapters/out/firestore"
	pbfs "narratives/internal/adapters/out/firestore/productBlueprint"
)

type repos struct {
	// FS repos
	accountRepo       *fs.AccountRepositoryFS
	announcementRepo  *fs.AnnouncementRepositoryFS
	avatarRepo        *fs.AvatarRepositoryFS
	avatarStateRepo   *fs.AvatarStateRepositoryFS
	paymentMethodRepo *fs.PaymentMethodRepositoryFS
	brandRepo         *fs.BrandRepositoryFS
	companyRepo       *fs.CompanyRepositoryFS
	inquiryRepo       *fs.InquiryRepositoryFS

	inventoryRepo *fs.InventoryRepositoryFS

	listRepoFS *fs.ListRepositoryFS

	listImageRecordRepo *fs.ListImageRepositoryFS

	memberRepo *fs.MemberRepositoryFS
	modelRepo  *fs.ModelRepositoryFS

	mintRepo        *fs.MintRepositoryFS
	tokenReaderRepo *fs.TokenReaderFS

	orderRepo      *fs.OrderRepositoryFS
	paymentRepo    *fs.PaymentRepositoryFS
	permissionRepo *fs.PermissionRepositoryFS
	productRepo    *fs.ProductRepositoryFS

	productBlueprintRepo         *pbfs.ProductBlueprintRepositoryFS
	productBlueprintCategoryRepo *fs.ProductBlueprintCategoryRepositoryFS

	productBlueprintReviewRepo *fs.ProductBlueprintReviewRepositoryFS

	productionRepo *fs.ProductionRepositoryFS

	shippingAddressRepo *fs.ShippingAddressRepositoryFS
	tokenBlueprintRepo  *fs.TokenBlueprintRepositoryFS

	tokenBlueprintReviewRepo *fs.TokenBlueprintReviewRepositoryFS

	userRepo   *fs.UserRepositoryFS
	walletRepo *fs.WalletRepositoryFS

	cartRepo *fs.CartRepositoryFS

	printLogRepo   *fs.PrintLogRepositoryFS
	inspectionRepo *fs.InspectionRepositoryFS

	invitationTokenRepo *fs.InvitationTokenRepositoryFS

	// ports
	mintRequestPort *fs.MintRequestPortFS
}

func buildRepos(c *clients) *repos {
	fsClient := c.fsClient

	// =========================================================
	// Outbound adapters (repositories)
	// =========================================================
	accountRepo := fs.NewAccountRepositoryFS(fsClient)
	announcementRepo := fs.NewAnnouncementRepositoryFS(fsClient)
	avatarRepo := fs.NewAvatarRepositoryFS(fsClient)
	avatarStateRepo := fs.NewAvatarStateRepositoryFS(fsClient)

	paymentMethodRepo := fs.NewPaymentMethodRepositoryFS(fsClient)
	brandRepo := fs.NewBrandRepositoryFS(fsClient)
	companyRepo := fs.NewCompanyRepositoryFS(fsClient)
	inquiryRepo := fs.NewInquiryRepositoryFS(fsClient)

	inventoryRepo := fs.NewInventoryRepositoryFS(fsClient)

	listRepoFS := fs.NewListRepositoryFS(fsClient)

	listImageRecordRepo := fs.NewListImageRepositoryFS(fsClient)

	memberRepo := fs.NewMemberRepositoryFS(fsClient)
	modelRepo := fs.NewModelRepositoryFS(fsClient)

	mintRepo := fs.NewMintRepositoryFS(fsClient)
	tokenReaderRepo := fs.NewTokenReaderFS(fsClient)

	orderRepo := fs.NewOrderRepositoryFS(fsClient)
	paymentRepo := fs.NewPaymentRepositoryFS(fsClient)
	permissionRepo := fs.NewPermissionRepositoryFS(fsClient)
	productRepo := fs.NewProductRepositoryFS(fsClient)

	productBlueprintRepo := pbfs.NewProductBlueprintRepositoryFS(fsClient)
	productBlueprintCategoryRepo := fs.NewProductBlueprintCategoryRepositoryFS(fsClient)

	productBlueprintReviewRepo := fs.NewProductBlueprintReviewRepositoryFS(fsClient)

	productionRepo := fs.NewProductionRepositoryFS(fsClient)

	shippingAddressRepo := fs.NewShippingAddressRepositoryFS(fsClient)
	tokenBlueprintRepo := fs.NewTokenBlueprintRepositoryFS(fsClient)

	tokenBlueprintReviewRepo := fs.NewTokenBlueprintReviewRepositoryFS(fsClient)

	userRepo := fs.NewUserRepositoryFS(fsClient)
	walletRepo := fs.NewWalletRepositoryFS(fsClient)

	cartRepo := fs.NewCartRepositoryFS(fsClient)

	printLogRepo := fs.NewPrintLogRepositoryFS(fsClient)
	inspectionRepo := fs.NewInspectionRepositoryFS(fsClient)

	invitationTokenRepo := fs.NewInvitationTokenRepositoryFS(fsClient)

	mintRequestPort := fs.NewMintRequestPortFS(
		fsClient,
		"mints",
		"token_blueprints",
		"brands",
	)

	return &repos{
		accountRepo:       accountRepo,
		announcementRepo:  announcementRepo,
		avatarRepo:        avatarRepo,
		avatarStateRepo:   avatarStateRepo,
		paymentMethodRepo: paymentMethodRepo,
		brandRepo:         brandRepo,
		companyRepo:       companyRepo,
		inquiryRepo:       inquiryRepo,

		inventoryRepo: inventoryRepo,

		listRepoFS: listRepoFS,

		listImageRecordRepo: listImageRecordRepo,

		memberRepo: memberRepo,
		modelRepo:  modelRepo,

		mintRepo:        mintRepo,
		tokenReaderRepo: tokenReaderRepo,

		orderRepo:      orderRepo,
		paymentRepo:    paymentRepo,
		permissionRepo: permissionRepo,
		productRepo:    productRepo,

		productBlueprintRepo:         productBlueprintRepo,
		productBlueprintCategoryRepo: productBlueprintCategoryRepo,

		productBlueprintReviewRepo: productBlueprintReviewRepo,

		productionRepo: productionRepo,

		shippingAddressRepo: shippingAddressRepo,
		tokenBlueprintRepo:  tokenBlueprintRepo,

		tokenBlueprintReviewRepo: tokenBlueprintReviewRepo,

		userRepo:   userRepo,
		walletRepo: walletRepo,

		cartRepo: cartRepo,

		printLogRepo:   printLogRepo,
		inspectionRepo: inspectionRepo,

		invitationTokenRepo: invitationTokenRepo,

		mintRequestPort: mintRequestPort,
	}
}
