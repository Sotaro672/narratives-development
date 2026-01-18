// backend/internal/platform/di/console/container.go
package console

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"time"

	httpin "narratives/internal/adapters/in/http/console"
	solanainfra "narratives/internal/infra/solana"

	fs "narratives/internal/adapters/out/firestore"
	gcso "narratives/internal/adapters/out/gcs"
	mailadp "narratives/internal/adapters/out/mail"

	// ✅ shared owner resolve query (wallet -> avatarId/brandId)
	sharedfs "narratives/internal/adapters/out/firestore/shared"
	sharedquery "narratives/internal/application/query/shared"

	// shared infra (Firestore/FirebaseAuth/GCS/MintKey/ArweaveUploader/buckets)
	shared "narratives/internal/platform/di/shared"

	// usecases / apps
	inspectionapp "narratives/internal/application/inspection"
	mintapp "narratives/internal/application/mint"
	productionapp "narratives/internal/application/production"
	companyquery "narratives/internal/application/query/console"
	resolver "narratives/internal/application/resolver"
	uc "narratives/internal/application/usecase"
	authuc "narratives/internal/application/usecase/auth"

	// domains
	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
	memdom "narratives/internal/domain/member"
	pbdom "narratives/internal/domain/productBlueprint"
)

// ========================================
// Container (Console DI)
// ========================================
//
// Console Container owns:
// - Firestore repositories (FS adapters)
// - domain services
// - application usecases
// - console-only query services
// - console router deps assembly
//
// It depends on shared.Infra for external clients and cross-cutting infra.
type Container struct {
	Infra *shared.Infra

	// Repositories (AuthMiddleware 用に memberRepo だけ保持)
	MemberRepo  memdom.Repository
	MessageRepo *fs.MessageRepositoryFS

	// member.Service / brand.Service (表示名解決用)
	MemberService *memdom.Service
	BrandService  *branddom.Service

	// History Repositories
	ProductBlueprintHistoryRepo *fs.ProductBlueprintHistoryRepositoryFS
	ModelHistoryRepo            *fs.ModelHistoryRepositoryFS

	// Application-layer usecases
	AccountUC          *uc.AccountUsecase
	AnnouncementUC     *uc.AnnouncementUsecase
	AvatarUC           *uc.AvatarUsecase
	BillingAddressUC   *uc.BillingAddressUsecase
	BrandUC            *uc.BrandUsecase
	CampaignUC         *uc.CampaignUsecase
	CompanyUC          *uc.CompanyUsecase
	InquiryUC          *uc.InquiryUsecase
	InventoryUC        *uc.InventoryUsecase
	InvoiceUC          *uc.InvoiceUsecase
	ListUC             *uc.ListUsecase
	MemberUC           *uc.MemberUsecase
	MessageUC          *uc.MessageUsecase
	ModelUC            *uc.ModelUsecase
	OrderUC            *uc.OrderUsecase
	PaymentUC          *uc.PaymentUsecase
	PermissionUC       *uc.PermissionUsecase
	PrintUC            *uc.PrintUsecase
	ProductionUC       *productionapp.ProductionUsecase
	ProductBlueprintUC *uc.ProductBlueprintUsecase
	ShippingAddressUC  *uc.ShippingAddressUsecase
	TokenUC            *uc.TokenUsecase
	TokenBlueprintUC   *uc.TokenBlueprintUsecase
	TokenOperationUC   *uc.TokenOperationUsecase
	TrackingUC         *uc.TrackingUsecase
	UserUC             *uc.UserUsecase
	WalletUC           *uc.WalletUsecase

	// Cart / Post
	CartUC *uc.CartUsecase
	PostUC *uc.PostUsecase

	// Console-only Query Services
	CompanyProductionQueryService *companyquery.CompanyProductionQueryService
	MintRequestQueryService       *companyquery.MintRequestQueryService
	InventoryQuery                *companyquery.InventoryQuery
	ListCreateQuery               *companyquery.ListCreateQuery
	ListManagementQuery           *companyquery.ListManagementQuery
	ListDetailQuery               *companyquery.ListDetailQuery

	// ✅ shared owner resolve query (wallet -> avatarId/brandId)
	OwnerResolveQ *sharedquery.OwnerResolveQuery

	// Inspector / Mint
	ProductUC    *uc.ProductUsecase
	InspectionUC *inspectionapp.InspectionUsecase
	MintUC       *mintapp.MintUsecase

	// Invitation
	InvitationQuery   uc.InvitationQueryPort
	InvitationCommand uc.InvitationCommandPort

	// auth/bootstrap
	AuthBootstrap *authuc.BootstrapService

	// NameResolver
	NameResolver *resolver.NameResolver
}

// NewContainer builds console container using shared infra.
func NewContainer(ctx context.Context, infra *shared.Infra) (*Container, error) {
	// ---------------------------------------------------------
	// shared infra
	// ---------------------------------------------------------
	if infra == nil {
		var err error
		infra, err = shared.NewInfra(ctx)
		if err != nil {
			return nil, err
		}
	}
	if infra == nil {
		return nil, errors.New("shared infra is nil")
	}

	// IMPORTANT: Config は後続で参照するので必須
	if infra.Config == nil {
		return nil, errors.New("shared infra config is nil")
	}

	// ---------------------------------------------------------
	// Required clients
	// ---------------------------------------------------------
	fsClient := infra.Firestore
	gcsClient := infra.GCS
	cfg := infra.Config

	if fsClient == nil {
		projectID := strings.TrimSpace(cfg.FirestoreProjectID)
		if projectID == "" {
			projectID = strings.TrimSpace(os.Getenv("FIRESTORE_PROJECT_ID"))
		}
		if projectID == "" {
			projectID = strings.TrimSpace(os.Getenv("GOOGLE_CLOUD_PROJECT"))
		}
		hasCredFile := strings.TrimSpace(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")) != ""
		log.Printf("[di.console] ERROR: infra.Firestore is nil (projectID=%q, GOOGLE_APPLICATION_CREDENTIALS_set=%t)", projectID, hasCredFile)
		return nil, fmt.Errorf("shared infra firestore client is nil (projectID=%q). shared.NewInfra likely failed to initialize Firestore client", projectID)
	}
	if gcsClient == nil {
		log.Printf("[di.console] ERROR: infra.GCS is nil")
		return nil, errors.New("shared infra gcs client is nil")
	}

	// =========================================================
	// 4. Outbound adapters (repositories)
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

	// Inventory repo (Firestore)
	inventoryRepo := fs.NewInventoryRepositoryFS(fsClient)
	// ✅ Adapter: inventory.RepositoryPort に追加された ApplyTransferResult を満たす
	inventoryRepoForUC := &inventoryRepoTransferResultAdapter{InventoryRepositoryFS: inventoryRepo}

	invoiceRepo := fs.NewInvoiceRepositoryFS(fsClient)

	listRepoFS := fs.NewListRepositoryFS(fsClient)
	listRepo := fs.NewListRepositoryForUsecase(listRepoFS)

	memberRepo := fs.NewMemberRepositoryFS(fsClient)
	messageRepo := fs.NewMessageRepositoryFS(fsClient)
	modelRepo := fs.NewModelRepositoryFS(fsClient)

	mintRepo := fs.NewMintRepositoryFS(fsClient)

	orderRepo := fs.NewOrderRepositoryFS(fsClient)
	paymentRepo := fs.NewPaymentRepositoryFS(fsClient)
	permissionRepo := fs.NewPermissionRepositoryFS(fsClient)
	productRepo := fs.NewProductRepositoryFS(fsClient)
	productBlueprintRepo := fs.NewProductBlueprintRepositoryFS(fsClient)
	productionRepo := fs.NewProductionRepositoryFS(fsClient)
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
	invitationTokenUCRepo := &invitationTokenRepoAdapter{
		fsRepo: invitationTokenFSRepo,
	}

	// =========================================================
	// ✅ shared OwnerResolveQuery (wallet -> avatarId / brandId)
	// - OwnerResolveQuery は (string, error) を要求するので AddressReaderFS を使う
	// =========================================================
	avatarAddrReader := sharedfs.NewAvatarWalletAddressReaderFS(fsClient, "avatars")
	brandAddrReader := sharedfs.NewBrandWalletAddressReaderFS(fsClient, "brands")
	ownerResolveQ := sharedquery.NewOwnerResolveQuery(avatarAddrReader, brandAddrReader)

	// =========================================================
	// Domain services
	// =========================================================
	companySvc := companydom.NewService(companyRepo)
	brandSvc := branddom.NewService(brandRepo)
	memberSvc := memdom.NewService(memberRepo)

	pbDomainRepo := &productBlueprintDomainRepoAdapter{repo: productBlueprintRepo}
	pbSvc := pbdom.NewService(pbDomainRepo)

	// =========================================================
	// NameResolver
	// =========================================================
	tokenBlueprintNameRepo := &tokenBlueprintNameRepoAdapter{repo: tokenBlueprintRepo}
	nameResolver := resolver.NewNameResolver(
		brandRepo,
		companyRepo,
		productBlueprintRepo,
		memberRepo,
		modelRepo,
		tokenBlueprintNameRepo,
	)

	// =========================================================
	// GCS repositories
	// =========================================================
	tokenIconRepo := gcso.NewTokenIconRepositoryGCS(gcsClient, infra.TokenIconBucket)
	tokenContentsRepo := gcso.NewTokenContentsRepositoryGCS(gcsClient, infra.TokenContentsBucket)
	listImageRepo := gcso.NewListImageRepositoryGCS(gcsClient, infra.ListImageBucket)
	avatarIconRepo := gcso.NewAvatarIconRepositoryGCS(gcsClient, infra.AvatarIconBucket)

	listPatcher := &listPatcherAdapter{repo: listRepoFS}

	// =========================================================
	// 5. Application-layer usecases
	// =========================================================
	mintRequestPort := fs.NewMintRequestPortFS(
		fsClient,
		"mints",
		"token_blueprints",
		"brands",
	)

	var tokenUC *uc.TokenUsecase
	if infra.MintAuthorityKey != nil {
		solanaClient := solanainfra.NewMintClient(infra.MintAuthorityKey)
		tokenUC = uc.NewTokenUsecase(solanaClient, mintRequestPort)
	} else {
		tokenUC = uc.NewTokenUsecase(nil, mintRequestPort)
	}

	accountUC := uc.NewAccountUsecase(accountRepo)
	announcementUC := uc.NewAnnouncementUsecase(announcementRepo, nil, nil)

	brandWalletSvc := solanainfra.NewBrandWalletService(cfg.FirestoreProjectID)
	avatarWalletSvc := solanainfra.NewAvatarWalletService(cfg.FirestoreProjectID)

	avatarUC := uc.NewAvatarUsecase(
		avatarRepo,
		avatarStateRepo,
		avatarIconRepo,
		avatarIconRepo,
	).
		WithWalletService(avatarWalletSvc).
		WithWalletRepo(walletRepo)

	callOptionalMethod(avatarUC, "WithCartRepo", cartRepo)

	billingAddressUC := uc.NewBillingAddressUsecase(billingAddressRepo)
	brandUC := uc.NewBrandUsecaseWithWallet(brandRepo, memberRepo, brandWalletSvc)
	campaignUC := uc.NewCampaignUsecase(campaignRepo, nil, nil, nil)
	companyUC := uc.NewCompanyUsecase(companyRepo)
	inquiryUC := uc.NewInquiryUsecase(inquiryRepo, nil, nil)

	// ✅ FIX: inventoryRepoForUC を渡す（ApplyTransferResult を満たす）
	inventoryUC := uc.NewInventoryUsecase(inventoryRepoForUC)

	paymentUC := uc.NewPaymentUsecase(paymentRepo)
	invoiceUC := uc.NewInvoiceUsecase(invoiceRepo)

	listUC := uc.NewListUsecaseWithCreator(
		listRepo,
		listRepo,
		listPatcher,
		listImageRepo,
		listImageRepo,
		listImageRepo,
	)

	memberUC := uc.NewMemberUsecase(memberRepo)
	messageUC := uc.NewMessageUsecase(messageRepo, nil, nil)
	modelUC := uc.NewModelUsecase(modelRepo, modelHistoryRepo)

	orderUC := uc.NewOrderUsecase(orderRepo)

	permissionUC := uc.NewPermissionUsecase(permissionRepo)

	printUC := uc.NewPrintUsecase(
		productRepo,
		printLogRepo,
		inspectionRepo,
		productBlueprintRepo,
		nameResolver,
	)

	productionUC := productionapp.NewProductionUsecase(
		productionRepo,
		pbSvc,
		nameResolver,
	)

	productBlueprintUC := uc.NewProductBlueprintUsecase(
		productBlueprintRepo,
		productBlueprintHistoryRepo,
	)

	inspectionProductRepo := &inspectionProductRepoAdapter{repo: productRepo}
	inspectionUC := inspectionapp.NewInspectionUsecase(
		inspectionRepo,
		inspectionProductRepo,
	)

	productQueryRepo := &productQueryRepoAdapter{
		productRepo:          productRepo,
		modelRepo:            modelRepo,
		productionRepo:       productionRepo,
		productBlueprintRepo: productBlueprintRepo,
	}
	productUC := uc.NewProductUsecase(productQueryRepo, brandSvc, companySvc)

	mintUC := mintapp.NewMintUsecase(
		productBlueprintRepo,
		productionRepo,
		inspectionRepo,
		modelRepo,
		tokenBlueprintRepo,
		brandSvc,
		mintRepo,
		inspectionRepo,
		tokenUC,
	)
	mintUC.SetNameResolver(nameResolver)
	mintUC.SetInventoryUsecase(inventoryUC)

	shippingAddressUC := uc.NewShippingAddressUsecase(shippingAddressRepo)

	tokenMetadataBuilder := uc.NewTokenMetadataBuilder()
	tokenBlueprintUC := uc.NewTokenBlueprintUsecase(
		tokenBlueprintRepo,
		tokenContentsRepo,
		tokenIconRepo,
		memberSvc,
		infra.ArweaveUploader,
		tokenMetadataBuilder,
	)

	tokenOperationUC := uc.NewTokenOperationUsecase(tokenOperationRepo)
	trackingUC := uc.NewTrackingUsecase(trackingRepo)
	userUC := uc.NewUserUsecase(userRepo)

	// ✅ WalletUsecase: 現行のコンストラクタは (walletRepo) のみ
	// Console 側では on-chain 同期が必要になったタイミングで、
	// di 側で uc.NewWalletUsecase(walletRepo).WithOnchainReader(...) を追加する。
	walletUC := uc.NewWalletUsecase(walletRepo)

	cartUC := uc.NewCartUsecase(cartRepo)

	invitationMailer := mailadp.NewInvitationMailerWithSendGrid(companySvc, brandSvc)
	invitationQueryUC := uc.NewInvitationService(invitationTokenUCRepo, memberRepo)
	invitationCommandUC := uc.NewInvitationCommandService(
		invitationTokenUCRepo,
		memberRepo,
		invitationMailer,
	)

	authBootstrapSvc := &authuc.BootstrapService{
		Members: &authMemberRepoAdapter{repo: memberRepo},
		Companies: &authCompanyRepoAdapter{
			repo: companyRepo,
		},
	}

	// =========================================================
	// Console-only queries
	// =========================================================
	pbQueryRepo := &pbQueryRepoAdapter{repo: productBlueprintRepo}
	companyProductionQueryService := companyquery.NewCompanyProductionQueryService(
		pbQueryRepo,
		productionRepo,
		nameResolver,
	)

	mintRequestQueryService := companyquery.NewMintRequestQueryService(
		mintUC,
		productionUC,
		nameResolver,
	)
	mintRequestQueryService.SetModelRepo(modelRepo)

	inventoryQuery := companyquery.NewInventoryQueryWithTokenBlueprintPatch(
		inventoryRepoForUC, // ✅ interfaceを満たす側を渡しておく
		&pbIDsByCompanyAdapter{repo: productBlueprintRepo},
		&pbPatchByIDAdapter{repo: productBlueprintRepo},
		&tbPatchByIDAdapter{repo: tokenBlueprintRepo},
		nameResolver,
	)

	listCreateQuery := companyquery.NewListCreateQueryWithInventoryAndModels(
		inventoryRepoForUC, // ✅ interfaceを満たす側を渡しておく
		modelRepo,
		&pbPatchByIDAdapter{repo: productBlueprintRepo},
		&tbPatchByIDAdapter{repo: tokenBlueprintRepo},
		nameResolver,
	)

	listManagementQuery := companyquery.NewListManagementQueryWithBrandInventoryAndInventoryRows(
		listRepo,
		nameResolver,
		productBlueprintRepo,
		&tbGetterAdapter{repo: tokenBlueprintRepo},
		inventoryQuery,
	)

	listDetailQuery := companyquery.NewListDetailQueryWithBrandInventoryAndInventoryRows(
		listRepo,
		nameResolver,
		productBlueprintRepo,
		&tbGetterAdapter{repo: tokenBlueprintRepo},
		inventoryQuery,
		inventoryQuery,
	)

	// =========================================================
	// Assemble
	// =========================================================
	return &Container{
		Infra: infra,

		MemberRepo:  memberRepo,
		MessageRepo: messageRepo,

		MemberService: memberSvc,
		BrandService:  brandSvc,

		ProductBlueprintHistoryRepo: productBlueprintHistoryRepo,
		ModelHistoryRepo:            modelHistoryRepo,

		AccountUC:          accountUC,
		AnnouncementUC:     announcementUC,
		AvatarUC:           avatarUC,
		BillingAddressUC:   billingAddressUC,
		BrandUC:            brandUC,
		CampaignUC:         campaignUC,
		CompanyUC:          companyUC,
		InquiryUC:          inquiryUC,
		InventoryUC:        inventoryUC,
		InvoiceUC:          invoiceUC,
		ListUC:             listUC,
		MemberUC:           memberUC,
		MessageUC:          messageUC,
		ModelUC:            modelUC,
		OrderUC:            orderUC,
		PaymentUC:          paymentUC,
		PermissionUC:       permissionUC,
		PrintUC:            printUC,
		ProductionUC:       productionUC,
		ProductBlueprintUC: productBlueprintUC,
		ShippingAddressUC:  shippingAddressUC,
		TokenUC:            tokenUC,
		TokenBlueprintUC:   tokenBlueprintUC,
		TokenOperationUC:   tokenOperationUC,
		TrackingUC:         trackingUC,
		UserUC:             userUC,
		WalletUC:           walletUC,

		CartUC: cartUC,

		CompanyProductionQueryService: companyProductionQueryService,
		MintRequestQueryService:       mintRequestQueryService,

		InventoryQuery:      inventoryQuery,
		ListCreateQuery:     listCreateQuery,
		ListManagementQuery: listManagementQuery,
		ListDetailQuery:     listDetailQuery,

		// ✅ NEW
		OwnerResolveQ: ownerResolveQ,

		ProductUC:         productUC,
		InspectionUC:      inspectionUC,
		MintUC:            mintUC,
		InvitationQuery:   invitationQueryUC,
		InvitationCommand: invitationCommandUC,
		AuthBootstrap:     authBootstrapSvc,
		NameResolver:      nameResolver,
	}, nil
}

// RouterDeps builds console router deps (no mall wiring here).
func (c *Container) RouterDeps() httpin.RouterDeps {
	return httpin.RouterDeps{
		AccountUC:          c.AccountUC,
		AnnouncementUC:     c.AnnouncementUC,
		AvatarUC:           c.AvatarUC,
		BillingAddressUC:   c.BillingAddressUC,
		BrandUC:            c.BrandUC,
		CampaignUC:         c.CampaignUC,
		CompanyUC:          c.CompanyUC,
		InquiryUC:          c.InquiryUC,
		InventoryUC:        c.InventoryUC,
		InvoiceUC:          c.InvoiceUC,
		ListUC:             c.ListUC,
		MemberUC:           c.MemberUC,
		MessageUC:          c.MessageUC,
		ModelUC:            c.ModelUC,
		OrderUC:            c.OrderUC,
		PaymentUC:          c.PaymentUC,
		PermissionUC:       c.PermissionUC,
		PrintUC:            c.PrintUC,
		TokenUC:            c.TokenUC,
		ProductionUC:       c.ProductionUC,
		ProductBlueprintUC: c.ProductBlueprintUC,
		ShippingAddressUC:  c.ShippingAddressUC,
		TokenBlueprintUC:   c.TokenBlueprintUC,
		TokenOperationUC:   c.TokenOperationUC,
		TrackingUC:         c.TrackingUC,
		UserUC:             c.UserUC,
		WalletUC:           c.WalletUC,

		CompanyProductionQueryService: c.CompanyProductionQueryService,
		MintRequestQueryService:       c.MintRequestQueryService,

		InventoryQuery:  c.InventoryQuery,
		ListCreateQuery: c.ListCreateQuery,

		ListManagementQuery: c.ListManagementQuery,
		ListDetailQuery:     c.ListDetailQuery,

		ProductUC:    c.ProductUC,
		InspectionUC: c.InspectionUC,

		MintUC: c.MintUC,

		InvitationQuery:   c.InvitationQuery,
		InvitationCommand: c.InvitationCommand,

		AuthBootstrap: c.AuthBootstrap,

		FirebaseAuth: c.Infra.FirebaseAuth,
		MemberRepo:   c.MemberRepo,

		MemberService: c.MemberService,
		BrandService:  c.BrandService,
		NameResolver:  c.NameResolver,

		MessageRepo: c.MessageRepo,

		// ✅ owner resolve query
		OwnerResolveQ: c.OwnerResolveQ,
	}
}

func (c *Container) Close() error {
	if c != nil && c.Infra != nil {
		return c.Infra.Close()
	}
	return nil
}

// callOptionalMethod calls obj.<methodName>(arg) when such method exists (best-effort).
func callOptionalMethod(obj any, methodName string, arg any) {
	if obj == nil || strings.TrimSpace(methodName) == "" || arg == nil {
		return
	}
	rv := reflect.ValueOf(obj)
	m := rv.MethodByName(methodName)
	if !m.IsValid() {
		return
	}
	if m.Type().NumIn() != 1 {
		return
	}
	av := reflect.ValueOf(arg)
	if !av.IsValid() {
		return
	}
	if !av.Type().AssignableTo(m.Type().In(0)) {
		if m.Type().In(0).Kind() == reflect.Interface && av.Type().Implements(m.Type().In(0)) {
			m.Call([]reflect.Value{av})
		}
		return
	}
	m.Call([]reflect.Value{av})
}

func init() {
	log.Printf("[di.console] container package loaded")
}

// ============================================================
// ✅ Adapter: fs.InventoryRepositoryFS に ApplyTransferResult を付与
// - inventory.RepositoryPort が要求する ApplyTransferResult の戻り値は error
// - 既存実装 ReleaseReservationAfterTransfer が (int, error) を返すため、
//   removedCount はログ用途に消費して error のみ返す
// ============================================================

type inventoryRepoTransferResultAdapter struct {
	*fs.InventoryRepositoryFS
}

// ApplyTransferResult updates inventory after transfer by removing productId and decrementing reservation for orderId.
func (a *inventoryRepoTransferResultAdapter) ApplyTransferResult(
	ctx context.Context,
	productID string,
	orderID string,
	now time.Time,
) error {
	if a == nil || a.InventoryRepositoryFS == nil {
		return errors.New("inventory repo adapter is nil")
	}

	removed, err := a.InventoryRepositoryFS.ReleaseReservationAfterTransfer(ctx, productID, orderID, now)
	if err != nil {
		return err
	}

	// best-effort log (removed can be 0 on idempotent re-run)
	log.Printf(
		"[inventory_repo_adapter] ApplyTransferResult ok productId=%q orderId=%q removed=%d at=%s",
		strings.TrimSpace(productID),
		strings.TrimSpace(orderID),
		removed,
		now.UTC().Format(time.RFC3339),
	)

	return nil
}
