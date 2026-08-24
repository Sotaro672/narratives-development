// backend/internal/platform/di/console/container_repos.go
package console

import (
	fs "narratives/internal/adapters/out/firestore"
)

type repos struct {
	// FS repos
	accountRepo                   *fs.AccountRepositoryFS
	announcementRepo              *fs.AnnouncementRepositoryFS
	announcementAttachmentRepo    *fs.AnnouncementAttachmentRepositoryFS
	avatarRepo                    *fs.AvatarRepositoryFS
	paymentMethodRepo             *fs.PaymentMethodRepositoryFS
	brandRepo                     *fs.BrandRepositoryFS
	companyRepo                   *fs.CompanyRepositoryFS
	inquiryRepo                   *fs.InquiryRepositoryFS
	inquiryReplyRepo              *fs.InquiryReplyRepositoryFS
	inventoryRepo                 *fs.InventoryRepositoryFS
	listRepoFS                    *fs.ListRepositoryFS
	listImageRecordRepo           *fs.ListImageRepositoryFS
	listSaveOperationRepo         *fs.ListSaveOperationRepositoryFS
	resaleRepo                    *fs.ResaleRepositoryFS
	memberRepo                    *fs.MemberRepositoryFS
	modelRepo                     *fs.ModelRepositoryFS
	mintRepo                      *fs.MintRepositoryFS
	tokenReaderRepo               *fs.TokenReaderFS
	transferRepo                  *fs.TransferRepositoryFS
	orderRepo                     *fs.OrderRepositoryFS
	orderDispatchNotificationRepo *fs.OrderDispatchNotificationRepositoryFS
	orderConsoleLister            *fs.OrderConsoleListerFS
	paymentRepo                   *fs.PaymentRepositoryFS
	settlementRepo                *fs.SettlementRepositoryFS
	permissionRepo                *fs.PermissionRepositoryFS
	productRepo                   *fs.ProductRepositoryFS
	productBlueprintRepo          *fs.ProductBlueprintRepositoryFS
	productBlueprintCategoryRepo  *fs.ProductBlueprintCategoryRepositoryFS
	productBlueprintReviewRepo    *fs.ProductBlueprintReviewRepositoryFS
	productionRepo                *fs.ProductionRepositoryFS
	shippingAddressRepo           *fs.ShippingAddressRepositoryFS
	transportationRepo            *fs.TransportationRepositoryFS
	tokenBlueprintRepo            *fs.TokenBlueprintRepositoryFS
	tokenBlueprintReviewRepo      *fs.TokenBlueprintReviewRepositoryFS
	userRepo                      *fs.UserRepositoryFS
	walletRepo                    *fs.WalletRepositoryFS
	cartRepo                      *fs.CartRepositoryFS
	printLogRepo                  *fs.PrintLogRepositoryFS
	inspectionRepo                *fs.InspectionRepositoryFS
	invitationTokenRepo           *fs.InvitationTokenRepositoryFS
}

func buildRepos(c *clients) *repos {
	fsClient := c.fsClient

	// =========================================================
	// Outbound adapters (repositories)
	// =========================================================
	accountRepo := fs.NewAccountRepositoryFS(fsClient)
	announcementRepo := fs.NewAnnouncementRepositoryFS(fsClient)
	announcementAttachmentRepo := fs.NewAnnouncementAttachmentRepositoryFS(fsClient)
	avatarRepo := fs.NewAvatarRepositoryFS(fsClient)
	paymentMethodRepo := fs.NewPaymentMethodRepositoryFS(fsClient)
	brandRepo := fs.NewBrandRepositoryFS(fsClient)
	companyRepo := fs.NewCompanyRepositoryFS(fsClient)
	inquiryRepo := fs.NewInquiryRepositoryFS(fsClient)
	inquiryReplyRepo := fs.NewInquiryReplyRepositoryFS(fsClient)
	inventoryRepo := fs.NewInventoryRepositoryFS(fsClient)
	listRepoFS := fs.NewListRepositoryFS(fsClient)
	listImageRecordRepo := fs.NewListImageRepositoryFS(fsClient)
	listSaveOperationRepo := fs.NewListSaveOperationRepositoryFS(fsClient)
	resaleRepo := fs.NewResaleRepositoryFS(fsClient)
	memberRepo := fs.NewMemberRepositoryFS(fsClient)
	modelRepo := fs.NewModelRepositoryFS(fsClient)
	mintRepo := fs.NewMintRepositoryFS(fsClient)
	tokenReaderRepo := fs.NewTokenReaderFS(fsClient)
	transferRepo := fs.NewTransferRepositoryFS(fsClient)
	orderRepo := fs.NewOrderRepositoryFS(fsClient)
	orderDispatchNotificationRepo := fs.NewOrderDispatchNotificationRepositoryFS(fsClient)
	orderConsoleLister := fs.NewOrderConsoleListerFS(fsClient)
	paymentRepo := fs.NewPaymentRepositoryFS(fsClient)
	settlementRepo := fs.NewSettlementRepositoryFS(fsClient)
	permissionRepo := fs.NewPermissionRepositoryFS(fsClient)
	productRepo := fs.NewProductRepositoryFS(fsClient)
	productBlueprintRepo := fs.NewProductBlueprintRepositoryFS(fsClient)
	productBlueprintCategoryRepo := fs.NewProductBlueprintCategoryRepositoryFS(fsClient)
	productBlueprintReviewRepo := fs.NewProductBlueprintReviewRepositoryFS(fsClient)
	productionRepo := fs.NewProductionRepositoryFS(fsClient)
	shippingAddressRepo := fs.NewShippingAddressRepositoryFS(fsClient)
	transportationRepo := fs.NewTransportationRepositoryFS(fsClient)
	tokenBlueprintRepo := fs.NewTokenBlueprintRepositoryFS(fsClient)
	tokenBlueprintReviewRepo := fs.NewTokenBlueprintReviewRepositoryFS(fsClient)
	userRepo := fs.NewUserRepositoryFS(fsClient)
	walletRepo := fs.NewWalletRepositoryFS(fsClient)
	cartRepo := fs.NewCartRepositoryFS(fsClient)
	printLogRepo := fs.NewPrintLogRepositoryFS(fsClient)
	inspectionRepo := fs.NewInspectionRepositoryFS(fsClient)
	invitationTokenRepo := fs.NewInvitationTokenRepositoryFS(fsClient)

	return &repos{
		accountRepo:                   accountRepo,
		announcementRepo:              announcementRepo,
		announcementAttachmentRepo:    announcementAttachmentRepo,
		avatarRepo:                    avatarRepo,
		paymentMethodRepo:             paymentMethodRepo,
		brandRepo:                     brandRepo,
		companyRepo:                   companyRepo,
		inquiryRepo:                   inquiryRepo,
		inquiryReplyRepo:              inquiryReplyRepo,
		inventoryRepo:                 inventoryRepo,
		listRepoFS:                    listRepoFS,
		listImageRecordRepo:           listImageRecordRepo,
		listSaveOperationRepo:         listSaveOperationRepo,
		resaleRepo:                    resaleRepo,
		memberRepo:                    memberRepo,
		modelRepo:                     modelRepo,
		mintRepo:                      mintRepo,
		tokenReaderRepo:               tokenReaderRepo,
		transferRepo:                  transferRepo,
		orderRepo:                     orderRepo,
		orderDispatchNotificationRepo: orderDispatchNotificationRepo,
		orderConsoleLister:            orderConsoleLister,
		paymentRepo:                   paymentRepo,
		settlementRepo:                settlementRepo,
		permissionRepo:                permissionRepo,
		productRepo:                   productRepo,
		productBlueprintRepo:          productBlueprintRepo,
		productBlueprintCategoryRepo:  productBlueprintCategoryRepo,
		productBlueprintReviewRepo:    productBlueprintReviewRepo,
		productionRepo:                productionRepo,
		shippingAddressRepo:           shippingAddressRepo,
		transportationRepo:            transportationRepo,
		tokenBlueprintRepo:            tokenBlueprintRepo,
		tokenBlueprintReviewRepo:      tokenBlueprintReviewRepo,
		userRepo:                      userRepo,
		walletRepo:                    walletRepo,
		cartRepo:                      cartRepo,
		printLogRepo:                  printLogRepo,
		inspectionRepo:                inspectionRepo,
		invitationTokenRepo:           invitationTokenRepo,
	}
}
