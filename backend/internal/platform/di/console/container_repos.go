// backend/internal/platform/di/console/container_repos.go
package console

import (
	fs "narratives/internal/adapters/out/firestore"
	pbfs "narratives/internal/adapters/out/firestore/productBlueprint"
	gcso "narratives/internal/adapters/out/gcs"
)

type repos struct {
	// FS repos
	accountRepo        *fs.AccountRepositoryFS
	announcementRepo   *fs.AnnouncementRepositoryFS
	avatarRepo         *fs.AvatarRepositoryFS
	avatarStateRepo    *fs.AvatarStateRepositoryFS
	billingAddressRepo *fs.BillingAddressRepositoryFS
	brandRepo          *fs.BrandRepositoryFS
	campaignRepo       *fs.CampaignRepositoryFS
	companyRepo        *fs.CompanyRepositoryFS
	inquiryRepo        *fs.InquiryRepositoryFS

	inventoryRepo      *fs.InventoryRepositoryFS
	inventoryRepoForUC *inventoryRepoTransferResultAdapter

	invoiceRepo *fs.InvoiceRepositoryFS

	listRepoFS *fs.ListRepositoryFS
	listRepo   *fs.ListRepositoryForUsecase

	// ✅ NEW: list image records in Firestore subcollection
	// /lists/{listId}/images/{imageId}
	listImageRecordRepo *fs.ListImageRepositoryFS

	memberRepo  *fs.MemberRepositoryFS
	messageRepo *fs.MessageRepositoryFS
	modelRepo   *fs.ModelRepositoryFS

	mintRepo *fs.MintRepositoryFS

	orderRepo      *fs.OrderRepositoryFS
	paymentRepo    *fs.PaymentRepositoryFS
	permissionRepo *fs.PermissionRepositoryFS
	productRepo    *fs.ProductRepositoryFS

	productBlueprintRepo *pbfs.ProductBlueprintRepositoryFS

	productionRepo      *fs.ProductionRepositoryFS
	productionRepoForUC *productionRepoTotalQuantityAdapter

	shippingAddressRepo *fs.ShippingAddressRepositoryFS
	tokenBlueprintRepo  *fs.TokenBlueprintRepositoryFS
	tokenOperationRepo  *fs.TokenOperationRepositoryFS
	trackingRepo        *fs.TrackingRepositoryFS
	userRepo            *fs.UserRepositoryFS
	walletRepo          *fs.WalletRepositoryFS

	cartRepo *fs.CartRepositoryFS

	printLogRepo   *fs.PrintLogRepositoryFS
	inspectionRepo *fs.InspectionRepositoryFS

	productBlueprintHistoryRepo *fs.ProductBlueprintHistoryRepositoryFS
	modelHistoryRepo            *fs.ModelHistoryRepositoryFS

	invitationTokenFSRepo *fs.InvitationTokenRepositoryFS
	invitationTokenUCRepo *invitationTokenRepoAdapter

	// ports
	mintRequestPort *fs.MintRequestPortFS

	// GCS repos / adapters
	listImageRepo  *gcso.ListImageRepositoryGCS
	avatarIconRepo *gcso.AvatarIconRepositoryGCS

	listPatcher *listPatcherAdapter
}

func buildRepos(c *clients) *repos {
	fsClient := c.fsClient
	gcsClient := c.gcsClient
	infra := c.infra

	// =========================================================
	// Outbound adapters (repositories)
	// =========================================================
	accountRepo := fs.NewAccountRepositoryFS(fsClient)
	announcementRepo := fs.NewAnnouncementRepositoryFS(fsClient)
	avatarRepo := fs.NewAvatarRepositoryFS(fsClient)
	avatarStateRepo := fs.NewAvatarStateRepositoryFS(fsClient)

	billingAddressRepo := fs.NewBillingAddressRepositoryFS(fsClient)
	brandRepo := fs.NewBrandRepositoryFS(fsClient)
	campaignRepo := fs.NewCampaignRepositoryFS(fsClient)
	companyRepo := fs.NewCompanyRepositoryFS(fsClient)
	inquiryRepo := fs.NewInquiryRepositoryFS(fsClient)

	inventoryRepo := fs.NewInventoryRepositoryFS(fsClient)
	inventoryRepoForUC := &inventoryRepoTransferResultAdapter{InventoryRepositoryFS: inventoryRepo}

	invoiceRepo := fs.NewInvoiceRepositoryFS(fsClient)

	listRepoFS := fs.NewListRepositoryFS(fsClient)
	listRepo := fs.NewListRepositoryForUsecase(listRepoFS)

	// ✅ NEW: list images Firestore subcollection repository
	listImageRecordRepo := fs.NewListImageRepositoryFS(fsClient)

	memberRepo := fs.NewMemberRepositoryFS(fsClient)
	messageRepo := fs.NewMessageRepositoryFS(fsClient)
	modelRepo := fs.NewModelRepositoryFS(fsClient)

	mintRepo := fs.NewMintRepositoryFS(fsClient)

	orderRepo := fs.NewOrderRepositoryFS(fsClient)
	paymentRepo := fs.NewPaymentRepositoryFS(fsClient)
	permissionRepo := fs.NewPermissionRepositoryFS(fsClient)
	productRepo := fs.NewProductRepositoryFS(fsClient)

	productBlueprintRepo := pbfs.NewProductBlueprintRepositoryFS(fsClient)

	productionRepo := fs.NewProductionRepositoryFS(fsClient)
	productionRepoForUC := &productionRepoTotalQuantityAdapter{ProductionRepositoryFS: productionRepo}

	shippingAddressRepo := fs.NewShippingAddressRepositoryFS(fsClient)
	tokenBlueprintRepo := fs.NewTokenBlueprintRepositoryFS(fsClient)
	tokenOperationRepo := fs.NewTokenOperationRepositoryFS(fsClient)
	trackingRepo := fs.NewTrackingRepositoryFS(fsClient)
	userRepo := fs.NewUserRepositoryFS(fsClient)
	walletRepo := fs.NewWalletRepositoryFS(fsClient)

	cartRepo := fs.NewCartRepositoryFS(fsClient)

	printLogRepo := fs.NewPrintLogRepositoryFS(fsClient)
	inspectionRepo := fs.NewInspectionRepositoryFS(fsClient)

	productBlueprintHistoryRepo := fs.NewProductBlueprintHistoryRepositoryFS(fsClient)
	modelHistoryRepo := fs.NewModelHistoryRepositoryFS(fsClient)

	invitationTokenFSRepo := fs.NewInvitationTokenRepositoryFS(fsClient)
	invitationTokenUCRepo := &invitationTokenRepoAdapter{fsRepo: invitationTokenFSRepo}

	// mint request port
	mintRequestPort := fs.NewMintRequestPortFS(
		fsClient,
		"mints",
		"token_blueprints",
		"brands",
	)

	// =========================================================
	// GCS repositories
	// =========================================================
	listImageRepo := gcso.NewListImageRepositoryGCS(gcsClient, infra.ListImageBucket)
	avatarIconRepo := gcso.NewAvatarIconRepositoryGCS(gcsClient, infra.AvatarIconBucket)
	listPatcher := &listPatcherAdapter{repo: listRepoFS}

	return &repos{
		accountRepo:        accountRepo,
		announcementRepo:   announcementRepo,
		avatarRepo:         avatarRepo,
		avatarStateRepo:    avatarStateRepo,
		billingAddressRepo: billingAddressRepo,
		brandRepo:          brandRepo,
		campaignRepo:       campaignRepo,
		companyRepo:        companyRepo,
		inquiryRepo:        inquiryRepo,

		inventoryRepo:      inventoryRepo,
		inventoryRepoForUC: inventoryRepoForUC,

		invoiceRepo: invoiceRepo,

		listRepoFS: listRepoFS,
		listRepo:   listRepo,

		// ✅ NEW
		listImageRecordRepo: listImageRecordRepo,

		memberRepo:  memberRepo,
		messageRepo: messageRepo,
		modelRepo:   modelRepo,

		mintRepo: mintRepo,

		orderRepo:      orderRepo,
		paymentRepo:    paymentRepo,
		permissionRepo: permissionRepo,
		productRepo:    productRepo,

		productBlueprintRepo: productBlueprintRepo,

		productionRepo:      productionRepo,
		productionRepoForUC: productionRepoForUC,

		shippingAddressRepo: shippingAddressRepo,
		tokenBlueprintRepo:  tokenBlueprintRepo,
		tokenOperationRepo:  tokenOperationRepo,
		trackingRepo:        trackingRepo,
		userRepo:            userRepo,
		walletRepo:          walletRepo,

		cartRepo: cartRepo,

		printLogRepo:   printLogRepo,
		inspectionRepo: inspectionRepo,

		productBlueprintHistoryRepo: productBlueprintHistoryRepo,
		modelHistoryRepo:            modelHistoryRepo,

		invitationTokenFSRepo: invitationTokenFSRepo,
		invitationTokenUCRepo: invitationTokenUCRepo,

		mintRequestPort: mintRequestPort,

		listImageRepo:  listImageRepo,
		avatarIconRepo: avatarIconRepo,
		listPatcher:    listPatcher,
	}
}
