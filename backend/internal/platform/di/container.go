// backend/internal/platform/di/container.go
package di

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

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

	// ★ CompanyProductionQueryService / MintRequestQueryService / InventoryQuery / ListCreateQuery
	companyquery "narratives/internal/application/query"

	resolver "narratives/internal/application/resolver"

	// ★ InspectionUsecase 移動先
	inspectionapp "narratives/internal/application/inspection"

	uc "narratives/internal/application/usecase"
	authuc "narratives/internal/application/usecase/auth"

	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
	memdom "narratives/internal/domain/member"
	mintdom "narratives/internal/domain/mint"
	productbpdom "narratives/internal/domain/productBlueprint"

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

// ============================================================
// Adapters (for query ports)
// ============================================================

// pbQueryRepoAdapter adapts ProductBlueprintRepositoryFS to query.ProductBlueprintQueryRepo.
type pbQueryRepoAdapter struct {
	repo interface {
		ListIDsByCompany(ctx context.Context, companyID string) ([]string, error)
		GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error) // ★ value 戻り
	}
}

func (a *pbQueryRepoAdapter) ListIDsByCompany(ctx context.Context, companyID string) ([]string, error) {
	return a.repo.ListIDsByCompany(ctx, companyID)
}

func (a *pbQueryRepoAdapter) GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error) {
	return a.repo.GetByID(ctx, id)
}

// pbIDsByCompanyAdapter adapts ProductBlueprintRepositoryFS to query.productBlueprintIDsByCompanyReader
type pbIDsByCompanyAdapter struct {
	repo interface {
		ListIDsByCompany(ctx context.Context, companyID string) ([]string, error)
	}
}

func (a *pbIDsByCompanyAdapter) ListIDsByCompanyID(ctx context.Context, companyID string) ([]string, error) {
	return a.repo.ListIDsByCompany(ctx, companyID)
}

// pbPatchByIDAdapter adapts ProductBlueprintRepositoryFS to query.productBlueprintPatchReader
type pbPatchByIDAdapter struct {
	repo interface {
		GetPatchByID(ctx context.Context, id string) (productbpdom.Patch, error)
	}
}

func (a *pbPatchByIDAdapter) GetPatchByID(ctx context.Context, id string) (productbpdom.Patch, error) {
	return a.repo.GetPatchByID(ctx, id)
}

// ============================================================
// Adapter: MintRepositoryFS -> mintdom.MintRepository (Update補完)
// ============================================================

type mintRepoWithUpdate struct {
	*fs.MintRepositoryFS
	Client *firestore.Client
}

func (r *mintRepoWithUpdate) Update(ctx context.Context, m mintdom.Mint) (mintdom.Mint, error) {
	if r == nil || r.Client == nil {
		return mintdom.Mint{}, errorsNew("mint repo is nil")
	}

	id := strings.TrimSpace(m.ID)
	if id == "" {
		return mintdom.Mint{}, errorsNew("mint id is empty")
	}
	m.ID = id

	type mintRecord struct {
		ID                string            `firestore:"id"`
		BrandID           string            `firestore:"brandId"`
		TokenBlueprintID  string            `firestore:"tokenBlueprintId"`
		Products          map[string]string `firestore:"products"`
		CreatedAt         time.Time         `firestore:"createdAt"`
		CreatedBy         string            `firestore:"createdBy"`
		MintedAt          *time.Time        `firestore:"mintedAt"`
		Minted            bool              `firestore:"minted"`
		ScheduledBurnDate *time.Time        `firestore:"scheduledBurnDate"`
	}

	rec := mintRecord{
		ID:                id,
		BrandID:           strings.TrimSpace(m.BrandID),
		TokenBlueprintID:  strings.TrimSpace(m.TokenBlueprintID),
		Products:          m.Products,
		CreatedAt:         m.CreatedAt,
		CreatedBy:         strings.TrimSpace(m.CreatedBy),
		MintedAt:          m.MintedAt,
		Minted:            m.Minted,
		ScheduledBurnDate: m.ScheduledBurnDate,
	}

	_, err := r.Client.Collection("mints").Doc(id).Set(ctx, rec, firestore.MergeAll)
	if err != nil {
		return mintdom.Mint{}, err
	}
	return m, nil
}

func errorsNew(msg string) error { return &simpleErr{s: msg} }

type simpleErr struct{ s string }

func (e *simpleErr) Error() string { return e.s }

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
	pbSvc := productbpdom.NewService(pbDomainRepo)

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
	var listUC *uc.ListUsecase = nil
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

	// ✅ invitationTokenUCRepo を定義済みなので undefined にならない
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
		&tbPatchByIDAdapter{repo: tokenBlueprintRepo}, // adapters.go 側
		nameResolver,
	)

	// ✅ NEW: ListCreateQuery（pb/tb -> brandName/productName/tokenName + inventory rows）
	listCreateQuery := companyquery.NewListCreateQueryWithInventory(
		inventoryRepo, // ★ これが無いと priceRows は常に空
		&pbPatchByIDAdapter{repo: productBlueprintRepo},
		&tbPatchByIDAdapter{repo: tokenBlueprintRepo},
		nameResolver,
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
		ListCreateQuery: listCreateQuery, // ✅ 追加

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

		InventoryQuery: c.InventoryQuery,

		// ✅ router.go の RouterDeps にこのフィールドが存在する前提
		// （もし router.go に未追加なら、RouterDeps に ListCreateQuery を追加してください）
		ListCreateQuery: c.ListCreateQuery,

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
