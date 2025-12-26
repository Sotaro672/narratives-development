// backend/internal/platform/di/container.go
package di

import (
	"context"
	"errors"
	"log"
	"os"
	"reflect"
	"strings"

	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"

	arweaveinfra "narratives/internal/infra/arweave"
	solanainfra "narratives/internal/infra/solana"

	httpin "narratives/internal/adapters/in/http"
	fs "narratives/internal/adapters/out/firestore"
	gcso "narratives/internal/adapters/out/gcs"
	mailadp "narratives/internal/adapters/out/mail"

	// ★ MintUsecase 移動先
	mintapp "narratives/internal/application/mint"

	// ★ ProductionUsecase（application/production）
	productionapp "narratives/internal/application/production"

	// ★ CompanyProductionQueryService / MintRequestQueryService / InventoryQuery / ListCreateQuery / ListManagementQuery / ListDetailQuery
	companyquery "narratives/internal/application/query"

	// ✅ SNS catalog query
	snsquery "narratives/internal/application/query/sns"

	resolver "narratives/internal/application/resolver"

	// ★ InspectionUsecase 移動先
	inspectionapp "narratives/internal/application/inspection"

	uc "narratives/internal/application/usecase"
	authuc "narratives/internal/application/usecase/auth"

	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
	memdom "narratives/internal/domain/member"
	pbdom "narratives/internal/domain/productBlueprint"

	appcfg "narratives/internal/infra/config"
)

// ========================================
// Container (Firestore + Firebase edition)
// ========================================
type Container struct {
	// Infra
	Config       *appcfg.Config
	Firestore    *firestore.Client
	FirebaseApp  *firebase.App
	FirebaseAuth *firebaseauth.Client

	// ★ GCS client (Token icon uploader etc.)
	GCS *storage.Client

	// Repositories（AuthMiddleware 用に memberRepo だけ保持）
	MemberRepo  memdom.Repository
	MessageRepo *fs.MessageRepositoryFS

	// ★ member.Service（表示名解決用）を保持
	MemberService *memdom.Service

	// ★ brand.Service（ブランド名解決用）を保持
	BrandService *branddom.Service

	// ★ History Repositories
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

	// ★ 追加: QueryService（GET /productions 一覧専用、Company境界付き）
	CompanyProductionQueryService *companyquery.CompanyProductionQueryService

	// ★ NEW: QueryService（GET /mint/requests 一覧専用、Company境界付き）
	MintRequestQueryService *companyquery.MintRequestQueryService

	// ★ NEW: Inventory detail の read-model assembler（GET /inventory/...）
	InventoryQuery *companyquery.InventoryQuery

	// ✅ NEW: listCreate 画面用 DTO（GET /inventory/list-create/...）
	ListCreateQuery *companyquery.ListCreateQuery

	// ✅ NEW: lists 一覧 DTO（GET /lists）= listManagement.tsx 用
	ListManagementQuery *companyquery.ListManagementQuery

	// ✅ NEW: list detail DTO（GET /lists/{id}）= listDetail.tsx 用
	ListDetailQuery *companyquery.ListDetailQuery

	// ✅ NEW: SNS catalog query（/sns/catalog/{listId}）
	SNSCatalogQ *snsquery.SNSCatalogQuery

	// ★ 検品アプリ用 ProductUsecase（/inspector/products/{id}）
	ProductUC *uc.ProductUsecase

	// ★ 検品アプリ用 Usecase（バッチ検品など）※ moved
	InspectionUC *inspectionapp.InspectionUsecase

	// ★ Mint 用 Usecase（MintRequest / NFT 発行チェーン）
	MintUC *mintapp.MintUsecase

	// ★ 招待関連 Usecase
	InvitationQuery   uc.InvitationQueryPort
	InvitationCommand uc.InvitationCommandPort

	// ★ auth/bootstrap 用 Usecase
	AuthBootstrap *authuc.BootstrapService

	// ★ Solana: Narratives ミント権限ウォレット
	MintAuthorityKey *solanainfra.MintAuthorityKey

	// ★ NameResolver（ID→名前/型番解決用）
	NameResolver *resolver.NameResolver
}

// ========================================
// NewContainer
// ========================================

func NewContainer(ctx context.Context) (*Container, error) {
	// 1. Load config
	cfg := appcfg.Load()

	// 1.2 Arweave HTTP uploader (optional)
	var arweaveUploader uc.ArweaveUploader
	if cfg.ArweaveBaseURL != "" {
		httpUp := arweaveinfra.NewHTTPUploader(cfg.ArweaveBaseURL, cfg.ArweaveAPIKey)
		arweaveUploader = httpUp
		log.Printf("[container] Arweave HTTPUploader initialized baseURL=%s", cfg.ArweaveBaseURL)
	} else {
		log.Printf("[container] Arweave HTTPUploader not configured (ARWEAVE_BASE_URL empty)")
	}

	// 1.5 Solana ミント権限ウォレットの鍵を Secret Manager から読み込む
	mintKey, err := solanainfra.LoadMintAuthorityKey(
		ctx,
		cfg.FirestoreProjectID,
		"narratives-solana-mint-authority",
	)
	if err != nil {
		log.Printf("[container] WARN: failed to load mint authority key: %v", err)
		mintKey = nil
	}

	// 2. Initialize Firestore client
	fsClient, err := firestore.NewClient(ctx, cfg.FirestoreProjectID)
	if err != nil {
		return nil, err
	}
	log.Println("[container] Firestore connected to project:", cfg.FirestoreProjectID)

	// 2.5 Initialize GCS client
	gcsClient, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	log.Println("[container] GCS storage client initialized")

	// 3. Initialize Firebase App & Auth
	var fbApp *firebase.App
	var fbAuth *firebaseauth.Client

	fbApp, err = firebase.NewApp(ctx, &firebase.Config{
		ProjectID: cfg.FirestoreProjectID,
	})
	if err != nil {
		log.Printf("[container] WARN: firebase app init failed: %v", err)
	} else {
		authClient, err := fbApp.Auth(ctx)
		if err != nil {
			log.Printf("[container] WARN: firebase auth init failed: %v", err)
		} else {
			fbAuth = authClient
			log.Printf("[container] Firebase Auth initialized")
		}
	}

	// 4. Outbound adapters (repositories)
	accountRepo := fs.NewAccountRepositoryFS(fsClient)
	announcementRepo := fs.NewAnnouncementRepositoryFS(fsClient)
	avatarRepo := fs.NewAvatarRepositoryFS(fsClient)
	billingAddressRepo := fs.NewBillingAddressRepositoryFS(fsClient)
	brandRepo := fs.NewBrandRepositoryFS(fsClient)
	campaignRepo := fs.NewCampaignRepositoryFS(fsClient)
	companyRepo := fs.NewCompanyRepositoryFS(fsClient)
	inquiryRepo := fs.NewInquiryRepositoryFS(fsClient)
	inventoryRepo := fs.NewInventoryRepositoryFS(fsClient)
	invoiceRepo := fs.NewInvoiceRepositoryFS(fsClient)

	// ✅ List (Firestore)
	listRepoFS := fs.NewListRepositoryFS(fsClient)
	listRepo := fs.NewListRepositoryForUsecase(listRepoFS)

	memberRepo := fs.NewMemberRepositoryFS(fsClient)
	messageRepo := fs.NewMessageRepositoryFS(fsClient)
	modelRepo := fs.NewModelRepositoryFS(fsClient)

	// ★ MintRepositoryFS（Update未実装分は mintRepoWithUpdate で補完）
	mintRepoFS := fs.NewMintRepositoryFS(fsClient)
	mintRepo := &mintRepoWithUpdate{
		MintRepositoryFS: mintRepoFS,
		Client:           fsClient,
	}

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

	printLogRepo := fs.NewPrintLogRepositoryFS(fsClient)
	inspectionRepo := fs.NewInspectionRepositoryFS(fsClient)

	productBlueprintHistoryRepo := fs.NewProductBlueprintHistoryRepositoryFS(fsClient)
	modelHistoryRepo := fs.NewModelHistoryRepositoryFS(fsClient)

	// ★ 招待トークン用 Repository（Firestore 実装）＋ Usecase 用アダプタ
	invitationTokenFSRepo := fs.NewInvitationTokenRepositoryFS(fsClient)
	invitationTokenUCRepo := &invitationTokenRepoAdapter{
		fsRepo: invitationTokenFSRepo,
	}

	// ★ Company / Brand 用ドメインサービス
	companySvc := companydom.NewService(companyRepo)
	brandSvc := branddom.NewService(brandRepo)

	// ★ Solana Brand Wallet Service
	brandWalletSvc := solanainfra.NewBrandWalletService(cfg.FirestoreProjectID)

	// ★ member.Service
	memberSvc := memdom.NewService(memberRepo)

	// ★ productBlueprint.Service（ProductName / BrandID 解決用）
	pbDomainRepo := &productBlueprintDomainRepoAdapter{repo: productBlueprintRepo}
	pbSvc := pbdom.NewService(pbDomainRepo)

	// ★ MintRequestPort（TokenUsecase 用）
	mintRequestPort := fs.NewMintRequestPortFS(
		fsClient,
		"mints",
		"token_blueprints",
		"brands",
	)

	// ★ NameResolver
	tokenBlueprintNameRepo := &tokenBlueprintNameRepoAdapter{repo: tokenBlueprintRepo}
	nameResolver := resolver.NewNameResolver(
		brandRepo,
		companyRepo,
		productBlueprintRepo,
		memberRepo,
		modelRepo,
		tokenBlueprintNameRepo,
	)

	// ★ Token icon repository (GCS)
	tokenIconRepo := gcso.NewTokenIconRepositoryGCS(gcsClient, cfg.TokenIconBucket)

	// ★ Token contents repository (GCS)
	tokenContentsBucket := strings.TrimSpace(os.Getenv("TOKEN_CONTENTS_BUCKET"))
	if tokenContentsBucket == "" {
		tokenContentsBucket = "narratives-development-token"
	}
	tokenContentsRepo := gcso.NewTokenContentsRepositoryGCS(gcsClient, tokenContentsBucket)

	// ✅ ListImage repository (GCS)
	listImageBucket := strings.TrimSpace(os.Getenv("LIST_IMAGE_BUCKET"))
	listImageRepo := gcso.NewListImageRepositoryGCS(gcsClient, listImageBucket)

	// ✅ ListPatcher adapter（imageId 更新専用）
	listPatcher := &listPatcherAdapter{repo: listRepoFS}

	// 5. Application-layer usecases

	// ★ TokenUsecase
	var tokenUC *uc.TokenUsecase
	if mintKey != nil {
		solanaClient := solanainfra.NewMintClient(mintKey)
		tokenUC = uc.NewTokenUsecase(solanaClient, mintRequestPort)
	} else {
		tokenUC = uc.NewTokenUsecase(nil, mintRequestPort)
	}

	accountUC := uc.NewAccountUsecase(accountRepo)
	announcementUC := uc.NewAnnouncementUsecase(announcementRepo, nil, nil)
	avatarUC := uc.NewAvatarUsecase(avatarRepo, nil, nil, nil)
	billingAddressUC := uc.NewBillingAddressUsecase(billingAddressRepo)

	brandUC := uc.NewBrandUsecaseWithWallet(brandRepo, memberRepo, brandWalletSvc)

	campaignUC := uc.NewCampaignUsecase(campaignRepo, nil, nil, nil)
	companyUC := uc.NewCompanyUsecase(companyRepo)

	inquiryUC := uc.NewInquiryUsecase(inquiryRepo, nil, nil)
	inventoryUC := uc.NewInventoryUsecase(inventoryRepo)
	invoiceUC := uc.NewInvoiceUsecase(invoiceRepo)

	listUC := uc.NewListUsecaseWithCreator(
		listRepo,      // ListReader (+ ListLister/ListUpdater)
		listRepo,      // ListCreator
		listPatcher,   // ListPatcher (imageId only)
		listImageRepo, // ListImageReader
		listImageRepo, // ListImageByIDReader
		listImageRepo, // ListImageObjectSaver (+ SignedURLIssuer)
	)

	memberUC := uc.NewMemberUsecase(memberRepo)
	messageUC := uc.NewMessageUsecase(messageRepo, nil, nil)

	modelUC := uc.NewModelUsecase(modelRepo, modelHistoryRepo)

	orderUC := uc.NewOrderUsecase(orderRepo)
	paymentUC := uc.NewPaymentUsecase(paymentRepo)
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

	pbQueryRepo := &pbQueryRepoAdapter{repo: productBlueprintRepo}
	companyProductionQueryService := companyquery.NewCompanyProductionQueryService(
		pbQueryRepo,
		productionRepo,
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

	mintRequestQueryService := companyquery.NewMintRequestQueryService(
		mintUC,
		productionUC,
		nameResolver,
	)
	mintRequestQueryService.SetModelRepo(modelRepo)

	shippingAddressUC := uc.NewShippingAddressUsecase(shippingAddressRepo)

	tokenMetadataBuilder := uc.NewTokenMetadataBuilder()
	tokenBlueprintUC := uc.NewTokenBlueprintUsecase(
		tokenBlueprintRepo,
		tokenContentsRepo,
		tokenIconRepo,
		memberSvc,
		arweaveUploader,
		tokenMetadataBuilder,
	)

	tokenOperationUC := uc.NewTokenOperationUsecase(tokenOperationRepo)
	trackingUC := uc.NewTrackingUsecase(trackingRepo)
	userUC := uc.NewUserUsecase(userRepo)
	walletUC := uc.NewWalletUsecase(walletRepo)

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

	// ★ NEW: InventoryQuery
	inventoryQuery := companyquery.NewInventoryQueryWithTokenBlueprintPatch(
		inventoryRepo,
		&pbIDsByCompanyAdapter{repo: productBlueprintRepo},
		&pbPatchByIDAdapter{repo: productBlueprintRepo},
		&tbPatchByIDAdapter{repo: tokenBlueprintRepo},
		nameResolver,
	)

	// ✅ NEW: ListCreateQuery
	listCreateQuery := companyquery.NewListCreateQueryWithInventory(
		inventoryRepo,
		&pbPatchByIDAdapter{repo: productBlueprintRepo},
		&tbPatchByIDAdapter{repo: tokenBlueprintRepo},
		nameResolver,
	)

	// ✅ NEW: ListManagementQuery（/lists 一覧 = listManagement.tsx 用）
	listManagementQuery := companyquery.NewListManagementQueryWithBrandInventoryAndInventoryRows(
		listRepo,
		nameResolver,
		productBlueprintRepo,
		&tbGetterAdapter{repo: tokenBlueprintRepo},
		inventoryQuery,
	)

	// ✅ NEW: ListDetailQuery（/lists/{id} = listDetail.tsx 用）
	listDetailQuery := companyquery.NewListDetailQueryWithBrandInventoryAndInventoryRows(
		listRepo,
		nameResolver,
		productBlueprintRepo,
		&tbGetterAdapter{repo: tokenBlueprintRepo},
		inventoryQuery,
		inventoryQuery,
	)

	// ✅ NEW: SNSCatalogQuery（buyer-facing /sns/catalog/{listId} 用）
	snsCatalogQ := snsquery.NewSNSCatalogQuery(
		listRepoFS,
		&snsCatalogInventoryRepoAdapter{repo: inventoryRepo},
		&snsCatalogProductBlueprintRepoAdapter{repo: productBlueprintRepo},
		modelRepo,
	)

	// 6. Assemble container
	return &Container{
		Config:       cfg,
		Firestore:    fsClient,
		FirebaseApp:  fbApp,
		FirebaseAuth: fbAuth,
		GCS:          gcsClient,

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

		CompanyProductionQueryService: companyProductionQueryService,
		MintRequestQueryService:       mintRequestQueryService,

		InventoryQuery:  inventoryQuery,
		ListCreateQuery: listCreateQuery,

		ListManagementQuery: listManagementQuery,
		ListDetailQuery:     listDetailQuery,

		SNSCatalogQ: snsCatalogQ,

		ProductUC:    productUC,
		InspectionUC: inspectionUC,

		MintUC: mintUC,

		InvitationQuery:   invitationQueryUC,
		InvitationCommand: invitationCommandUC,

		AuthBootstrap: authBootstrapSvc,

		MintAuthorityKey: mintKey,
		NameResolver:     nameResolver,
	}, nil
}

// ✅ sns_container.go から reflection で呼ばれる想定の getter
func (c *Container) SNSCatalogQuery() *snsquery.SNSCatalogQuery {
	if c == nil {
		return nil
	}
	return c.SNSCatalogQ
}

// ========================================
// RouterDeps
// ========================================

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

		FirebaseAuth: c.FirebaseAuth,
		MemberRepo:   c.MemberRepo,

		MemberService: c.MemberService,
		BrandService:  c.BrandService,
		NameResolver:  c.NameResolver,

		MessageRepo: c.MessageRepo,
	}
}

// ========================================
// Close
// ========================================
func (c *Container) Close() error {
	if c.Firestore != nil {
		_ = c.Firestore.Close()
	}
	if c.GCS != nil {
		_ = c.GCS.Close()
	}
	return nil
}

// ============================================================
// sns catalog adapters (DI-only helpers)
// - compile-time で inventory domain 型に依存しないため、reflection で吸収する
// ============================================================

type snsCatalogInventoryRepoAdapter struct {
	repo any
}

func (a *snsCatalogInventoryRepoAdapter) GetByID(ctx context.Context, id string) (*snsquery.SNSCatalogInventoryDTO, error) {
	if a == nil || a.repo == nil {
		return nil, errors.New("sns catalog inventory repo: repo is nil")
	}
	v, err := callRepo(a.repo, []string{"GetByID", "GetById"}, ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	return toSNSCatalogInventoryDTO(v)
}

func (a *snsCatalogInventoryRepoAdapter) GetByProductAndTokenBlueprintID(ctx context.Context, productBlueprintID, tokenBlueprintID string) (*snsquery.SNSCatalogInventoryDTO, error) {
	if a == nil || a.repo == nil {
		return nil, errors.New("sns catalog inventory repo: repo is nil")
	}
	pb := strings.TrimSpace(productBlueprintID)
	tb := strings.TrimSpace(tokenBlueprintID)

	// method 名揺れ吸収
	methods := []string{
		"GetByProductAndTokenBlueprintID",
		"GetByProductAndTokenBlueprintId",
		"GetByProductAndTokenBlueprintIDs",
		"GetByProductAndTokenBlueprintIds",
	}
	v, err := callRepo(a.repo, methods, ctx, pb, tb)
	if err != nil {
		return nil, err
	}
	return toSNSCatalogInventoryDTO(v)
}

type snsCatalogProductBlueprintRepoAdapter struct {
	repo any
}

func (a *snsCatalogProductBlueprintRepoAdapter) GetByID(ctx context.Context, id string) (*pbdom.ProductBlueprint, error) {
	if a == nil || a.repo == nil {
		return nil, errors.New("sns catalog product repo: repo is nil")
	}
	v, err := callRepo(a.repo, []string{"GetByID", "GetById"}, ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, errors.New("productBlueprint is nil")
	}
	if pb, ok := v.(*pbdom.ProductBlueprint); ok {
		return pb, nil
	}
	if pb, ok := v.(pbdom.ProductBlueprint); ok {
		cp := pb
		return &cp, nil
	}

	// 最後の手段：pointer/struct を reflection で解釈（型が一致しない場合はエラー）
	rv := reflect.ValueOf(v)
	if rv.IsValid() && rv.Kind() == reflect.Pointer && !rv.IsNil() {
		if x, ok := rv.Interface().(*pbdom.ProductBlueprint); ok {
			return x, nil
		}
	}
	return nil, errors.New("unexpected productBlueprint type")
}

func callRepo(repo any, methodNames []string, args ...any) (any, error) {
	rv := reflect.ValueOf(repo)
	if !rv.IsValid() {
		return nil, errors.New("repo is invalid")
	}

	for _, name := range methodNames {
		m := rv.MethodByName(name)
		if !m.IsValid() {
			continue
		}

		in := make([]reflect.Value, 0, len(args))
		for _, a := range args {
			in = append(in, reflect.ValueOf(a))
		}

		out := m.Call(in)
		if len(out) == 0 {
			return nil, nil
		}

		// (T, error) を想定。最後が error なら拾う
		if len(out) >= 2 {
			if e, ok := out[len(out)-1].Interface().(error); ok && e != nil {
				return nil, e
			}
		}
		return out[0].Interface(), nil
	}

	return nil, errors.New("method not found on repo")
}

func toSNSCatalogInventoryDTO(v any) (*snsquery.SNSCatalogInventoryDTO, error) {
	if v == nil {
		return nil, errors.New("inventory is nil")
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return nil, errors.New("inventory is invalid")
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, errors.New("inventory is nil")
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, errors.New("inventory is not struct")
	}

	getStr := func(names ...string) string {
		for _, n := range names {
			f := rv.FieldByName(n)
			if !f.IsValid() {
				continue
			}
			if f.Kind() == reflect.String {
				return strings.TrimSpace(f.String())
			}
		}
		return ""
	}

	id := getStr("ID", "Id", "InventoryID", "InventoryId")
	pbID := getStr("ProductBlueprintID", "ProductBlueprintId", "productBlueprintId")
	tbID := getStr("TokenBlueprintID", "TokenBlueprintId", "tokenBlueprintId")

	// ✅ Stock を products 付きで詰める（value も活用）
	stock := map[string]snsquery.SNSCatalogInventoryModelStockDTO{}

	// map field name tolerant
	var sf reflect.Value
	for _, n := range []string{"Stock", "Stocks", "stock"} {
		f := rv.FieldByName(n)
		if f.IsValid() {
			sf = f
			break
		}
	}

	if sf.IsValid() {
		if sf.Kind() == reflect.Pointer {
			if !sf.IsNil() {
				sf = sf.Elem()
			}
		}

		switch sf.Kind() {
		case reflect.Map:
			// map[string]X を想定（X: slice/map/struct{Products...}/etc）
			if sf.Type().Key().Kind() == reflect.String {
				iter := sf.MapRange()
				for iter.Next() {
					modelID := strings.TrimSpace(iter.Key().String())
					if modelID == "" {
						continue
					}

					ids := extractProductIDsFromStockValue(iter.Value())
					products := make(map[string]bool, len(ids))
					for _, pid := range ids {
						pid = strings.TrimSpace(pid)
						if pid == "" {
							continue
						}
						products[pid] = true
					}

					stock[modelID] = snsquery.SNSCatalogInventoryModelStockDTO{
						Products: products,
					}
				}
			}

		case reflect.Slice, reflect.Array:
			// 万一 []string / []any で「modelId の配列」が入っているだけのケース（best-effort）
			// → products は空で入れる
			for i := 0; i < sf.Len(); i++ {
				it := sf.Index(i)
				if it.Kind() == reflect.Interface && !it.IsNil() {
					it = it.Elem()
				}
				if it.Kind() == reflect.Pointer && !it.IsNil() {
					it = it.Elem()
				}
				if !it.IsValid() {
					continue
				}
				if it.Kind() == reflect.String {
					modelID := strings.TrimSpace(it.String())
					if modelID == "" {
						continue
					}
					if _, ok := stock[modelID]; !ok {
						stock[modelID] = snsquery.SNSCatalogInventoryModelStockDTO{Products: map[string]bool{}}
					}
				}
			}
		}
	}

	return &snsquery.SNSCatalogInventoryDTO{
		ID:                 id,
		ProductBlueprintID: pbID,
		TokenBlueprintID:   tbID,
		Stock:              stock,
	}, nil
}

// ------------------------------------------------------------
// stock reflection helpers (modelId -> products)
// ------------------------------------------------------------

// extractProductIDsFromStockValue supports:
// - stock[modelId] = []string
// - stock[modelId] = map[string]bool / map[string]any (key = productId)
// - stock[modelId] = struct{ Products ... } (Products is slice/map)
// - pointers/interfaces nested
func extractProductIDsFromStockValue(v reflect.Value) []string {
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Interface && !v.IsNil() {
		v = v.Elem()
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	// struct { Products: ... }
	if v.Kind() == reflect.Struct {
		pf := v.FieldByName("Products")
		if pf.IsValid() {
			return extractStringIDs(pf)
		}
	}

	// direct map/slice
	return extractStringIDs(v)
}

func extractStringIDs(v reflect.Value) []string {
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Interface && !v.IsNil() {
		v = v.Elem()
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		out := make([]string, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			e := v.Index(i)
			if e.Kind() == reflect.Interface && !e.IsNil() {
				e = e.Elem()
			}
			if e.Kind() == reflect.Pointer {
				if e.IsNil() {
					continue
				}
				e = e.Elem()
			}
			if e.Kind() == reflect.String {
				s := strings.TrimSpace(e.String())
				if s != "" {
					out = append(out, s)
				}
			}
		}
		return out

	case reflect.Map:
		// map[string]bool / map[string]any など: key を productId とみなす
		if v.Type().Key().Kind() != reflect.String {
			return nil
		}
		out := make([]string, 0, v.Len())
		iter := v.MapRange()
		for iter.Next() {
			k := iter.Key()
			if k.Kind() != reflect.String {
				continue
			}
			s := strings.TrimSpace(k.String())
			if s != "" {
				out = append(out, s)
			}
		}
		return out

	default:
		return nil
	}
}
