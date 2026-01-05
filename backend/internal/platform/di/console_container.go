// backend/internal/platform/di/container.go
package di

import (
	"context"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"

	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"

	arweaveinfra "narratives/internal/infra/arweave"
	solanainfra "narratives/internal/infra/solana"

	httpin "narratives/internal/adapters/in/http/console"
	fs "narratives/internal/adapters/out/firestore"
	gcso "narratives/internal/adapters/out/gcs"
	mailadp "narratives/internal/adapters/out/mail"

	// ✅ SNS handlers（Cart/Payment/ShippingAddress など）
	snshandler "narratives/internal/adapters/in/http/mall/handler"

	// ★ MintUsecase 移動先
	mintapp "narratives/internal/application/mint"

	// ★ ProductionUsecase（application/production）
	productionapp "narratives/internal/application/production"

	// ★ CompanyProductionQueryService / MintRequestQueryService / InventoryQuery / ListCreateQuery / ListManagementQuery / ListDetailQuery
	companyquery "narratives/internal/application/query"

	// ✅ SNS queries (catalog / order / cart / preview)
	snsquery "narratives/internal/application/query/mall"

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

	// ✅ NEW: Cart / Post
	CartUC *uc.CartUsecase
	PostUC *uc.PostUsecase

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

	// ✅ NEW: SNS cart / preview queries（/sns/cart, /sns/preview）
	SNSCartQ    *snsquery.SNSCartQuery
	SNSPreviewQ *snsquery.SNSPreviewQuery

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

	// ✅ SNS handlers（sns_container.go が best-effort で探す対象）
	// NOTE: Go は field と method の同名を許さないため、field は小文字にする
	snsSignInHandler          http.Handler
	snsCartHandler            http.Handler
	snsPaymentHandler         http.Handler
	snsShippingAddressHandler http.Handler
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

	// ✅ AvatarState repo（Firestore実装）を usecase 互換（Upsert 揺れ吸収）
	avatarStateRepoFS := fs.NewAvatarStateRepositoryFS(fsClient)
	avatarStateRepo := &avatarStateRepoAdapter{repo: avatarStateRepoFS}

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

	// ✅ NEW: Cart / Post repositories
	cartRepo := fs.NewCartRepositoryFS(fsClient)
	postRepo := fs.NewPostRepositoryFS(fsClient)

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

	// ✅ Solana Avatar Wallet Service (Wallet Open for Avatar)
	avatarWalletSvc := solanainfra.NewAvatarWalletService(cfg.FirestoreProjectID)

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

	// ✅ AvatarIcon repository (GCS)
	avatarIconBucket := strings.TrimSpace(os.Getenv("AVATAR_ICON_BUCKET"))
	if avatarIconBucket == "" {
		avatarIconBucket = "narratives-development_avatar_icon"
	}
	avatarIconRepo := gcso.NewAvatarIconRepositoryGCS(gcsClient, avatarIconBucket)

	// ✅ PostImage repository (GCS) を usecase 互換（戻り値3つ + objectPath/publicURL）
	postImageBucket := strings.TrimSpace(os.Getenv("POST_IMAGE_BUCKET"))
	if postImageBucket == "" {
		postImageBucket = "narratives-development-posts"
	}
	postImageRepoGCS := gcso.NewPostImageRepositoryGCS(gcsClient, postImageBucket)
	postImageRepo := &postImageRepoAdapter{repo: postImageRepoGCS}

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

	// ✅ AvatarUsecase
	avatarUC := uc.NewAvatarUsecase(
		avatarRepo,
		avatarStateRepo, // ✅ usecase.AvatarStateRepo
		avatarIconRepo,  // AvatarIconRepo
		avatarIconRepo,  // AvatarIconObjectStoragePort
	).
		WithWalletService(avatarWalletSvc).
		WithWalletRepo(walletRepo)

	// ✅ optional: avatar 作成時に cart を同時起票する実装に備え best-effort 注入
	callOptionalMethod(avatarUC, "WithCartRepo", cartRepo)

	billingAddressUC := uc.NewBillingAddressUsecase(billingAddressRepo)
	brandUC := uc.NewBrandUsecaseWithWallet(brandRepo, memberRepo, brandWalletSvc)
	campaignUC := uc.NewCampaignUsecase(campaignRepo, nil, nil, nil)
	companyUC := uc.NewCompanyUsecase(companyRepo)
	inquiryUC := uc.NewInquiryUsecase(inquiryRepo, nil, nil)
	inventoryUC := uc.NewInventoryUsecase(inventoryRepo)

	// ✅ PaymentUsecase を先に作る（InvoiceUsecase が PaymentCreator を要求するため）
	paymentUC := uc.NewPaymentUsecase(paymentRepo)

	// ✅ InvoiceUsecase は (InvoiceRepo, PaymentCreator) を受け取る
	invoiceUC := uc.NewInvoiceUsecase(invoiceRepo, paymentUC)

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

	// ✅ 修正: OrderUsecase は invoiceRepo を受け取る（Order 作成直後に Invoice 起票するため）
	orderUC := uc.NewOrderUsecase(orderRepo).
		WithInvoiceUsecase(invoiceUC)

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

	// ✅ NEW: Cart / Post usecases
	cartUC := uc.NewCartUsecase(cartRepo)
	postUC := uc.NewPostUsecase(postRepo, postImageRepo)

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

	// ✅ NEW: SNSCartQuery（buyer-facing /sns/cart 用）
	snsCartQ := snsquery.NewSNSCartQuery(fsClient)
	snsCartQ.Resolver = nameResolver

	// ✅ NEW: SNSPreviewQuery（buyer-facing /sns/preview 用）
	snsPreviewQ := snsquery.NewSNSPreviewQuery(fsClient)
	snsPreviewQ.Resolver = nameResolver

	// ============================================================
	// ✅ SNSOrderQuery（buyer-facing /sns/payment 用）
	// ============================================================
	orderQ := snsquery.NewSNSOrderQuery(fsClient)

	// ✅ SNS handlers（sns_container.go が拾えるようにコンテナに保持）
	snsSignInHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	snsCartHandler := snshandler.NewCartHandler(cartUC)
	snsPaymentHandler := snshandler.NewPaymentHandlerWithOrderQuery(paymentUC, orderQ)
	snsShippingAddressHandler := snshandler.NewShippingAddressHandler(shippingAddressUC) // ✅ 追加

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

		CartUC: cartUC,
		PostUC: postUC,

		CompanyProductionQueryService: companyProductionQueryService,
		MintRequestQueryService:       mintRequestQueryService,

		InventoryQuery:  inventoryQuery,
		ListCreateQuery: listCreateQuery,

		ListManagementQuery: listManagementQuery,
		ListDetailQuery:     listDetailQuery,

		SNSCatalogQ: snsCatalogQ,
		SNSCartQ:    snsCartQ,
		SNSPreviewQ: snsPreviewQ,

		ProductUC:    productUC,
		InspectionUC: inspectionUC,

		MintUC: mintUC,

		InvitationQuery:   invitationQueryUC,
		InvitationCommand: invitationCommandUC,

		AuthBootstrap: authBootstrapSvc,

		MintAuthorityKey: mintKey,
		NameResolver:     nameResolver,

		// ✅ SNS handlers（private fields）
		snsSignInHandler:          snsSignInHandler,
		snsCartHandler:            snsCartHandler,
		snsPaymentHandler:         snsPaymentHandler,
		snsShippingAddressHandler: snsShippingAddressHandler, // ✅ 追加
	}, nil
}

// ✅ sns_container.go から reflection で呼ばれる想定の getter
func (c *Container) SNSCatalogQuery() *snsquery.SNSCatalogQuery {
	if c == nil {
		return nil
	}
	return c.SNSCatalogQ
}

func (c *Container) SNSCartQuery() *snsquery.SNSCartQuery {
	if c == nil {
		return nil
	}
	return c.SNSCartQ
}

func (c *Container) SNSPreviewQuery() *snsquery.SNSPreviewQuery {
	if c == nil {
		return nil
	}
	return c.SNSPreviewQ
}

// ✅ sns_container.go から取得される想定（best-effort）
func (c *Container) CartUsecase() *uc.CartUsecase {
	if c == nil {
		return nil
	}
	return c.CartUC
}

// ✅ post を DI に入れたので getter も用意（sns 側で参照する場合に備える）
func (c *Container) PostUsecase() *uc.PostUsecase {
	if c == nil {
		return nil
	}
	return c.PostUC
}

// ✅ sns_container.go が「handler」として拾えるメソッド群
// NOTE: field と method の同名を避けるため field は小文字、method はこの名前を維持
func (c *Container) SNSSignInHandler() http.Handler {
	if c == nil {
		return nil
	}
	return c.snsSignInHandler
}

func (c *Container) SNSCartHandler() http.Handler {
	if c == nil {
		return nil
	}
	return c.snsCartHandler
}

func (c *Container) SNSPaymentHandler() http.Handler {
	if c == nil {
		return nil
	}
	return c.snsPaymentHandler
}

// ✅ NEW: shipping address handler getter（sns_container.go が拾う）
func (c *Container) SNSShippingAddressHandler() http.Handler {
	if c == nil {
		return nil
	}
	return c.snsShippingAddressHandler
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
	// expect 1 input
	if m.Type().NumIn() != 1 {
		return
	}
	av := reflect.ValueOf(arg)
	if !av.IsValid() {
		return
	}
	// assignable only
	if !av.Type().AssignableTo(m.Type().In(0)) {
		// if method expects interface, and arg implements it
		if m.Type().In(0).Kind() == reflect.Interface && av.Type().Implements(m.Type().In(0)) {
			m.Call([]reflect.Value{av})
		}
		return
	}
	m.Call([]reflect.Value{av})
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
