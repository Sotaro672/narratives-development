// backend/internal/platform/di/container.go
package di

import (
	"context"
	"log"

	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"

	solanainfra "narratives/internal/infra/solana"

	httpin "narratives/internal/adapters/in/http"
	fs "narratives/internal/adapters/out/firestore"
	mailadp "narratives/internal/adapters/out/mail"
	productionapp "narratives/internal/application/production"
	resolver "narratives/internal/application/resolver"
	uc "narratives/internal/application/usecase"
	authuc "narratives/internal/application/usecase/auth"
	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
	memdom "narratives/internal/domain/member"
	productbpdom "narratives/internal/domain/productBlueprint"
	appcfg "narratives/internal/infra/config"

	"cloud.google.com/go/firestore"
)

// ========================================
// Container (Firestore + Firebase edition)
// ========================================
//
// Firestore クライアントと Firebase Auth クライアントを中心に、
// 各 Repository と Usecase を初期化して束ねる。
type Container struct {
	// Infra
	Config       *appcfg.Config
	Firestore    *firestore.Client
	FirebaseApp  *firebase.App
	FirebaseAuth *firebaseauth.Client

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
	DiscountUC         *uc.DiscountUsecase
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
	SaleUC             *uc.SaleUsecase
	ShippingAddressUC  *uc.ShippingAddressUsecase
	TokenUC            *uc.TokenUsecase
	TokenBlueprintUC   *uc.TokenBlueprintUsecase
	TokenOperationUC   *uc.TokenOperationUsecase
	TrackingUC         *uc.TrackingUsecase
	UserUC             *uc.UserUsecase
	WalletUC           *uc.WalletUsecase

	// ★ 検品アプリ用 ProductUsecase（/inspector/products/{id}）
	ProductUC *uc.ProductUsecase

	// ★ 検品アプリ用 Usecase（バッチ検品など）
	InspectionUC *uc.InspectionUsecase

	// ★ Mint 用 Usecase（MintRequest / NFT 発行チェーン）
	MintUC *uc.MintUsecase

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
//
// Firestore / Firebase クライアントを初期化し、各 Usecase を構築して返す。
func NewContainer(ctx context.Context) (*Container, error) {
	// 1. Load config
	cfg := appcfg.Load()

	// 1.5 Solana ミント権限ウォレットの鍵を Secret Manager から読み込む
	mintKey, err := solanainfra.LoadMintAuthorityKey(
		ctx,
		cfg.FirestoreProjectID,             // = narratives-development-26c2d
		"narratives-solana-mint-authority", // Secret 名
	)
	if err != nil {
		log.Printf("[container] WARN: failed to load mint authority key: %v", err)
		// 開発中は nil 許容。本番で必須にする場合はここで return err にしても良い
		mintKey = nil
	}

	// 2. Initialize Firestore client (Application Default Credentials 前提)
	fsClient, err := firestore.NewClient(ctx, cfg.FirestoreProjectID)
	if err != nil {
		return nil, err
	}
	log.Println("[container] Firestore connected to project:", cfg.FirestoreProjectID)

	// 3. Initialize Firebase App & Auth（AuthMiddleware 用）
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

	// 4. Outbound adapters (repositories) — Firestore 版
	accountRepo := fs.NewAccountRepositoryFS(fsClient)
	announcementRepo := fs.NewAnnouncementRepositoryFS(fsClient)
	avatarRepo := fs.NewAvatarRepositoryFS(fsClient)
	billingAddressRepo := fs.NewBillingAddressRepositoryFS(fsClient)
	brandRepo := fs.NewBrandRepositoryFS(fsClient)
	campaignRepo := fs.NewCampaignRepositoryFS(fsClient)
	companyRepo := fs.NewCompanyRepositoryFS(fsClient)
	discountRepo := fs.NewDiscountRepositoryFS(fsClient)
	inquiryRepo := fs.NewInquiryRepositoryFS(fsClient)
	inventoryRepo := fs.NewInventoryRepositoryFS(fsClient)
	invoiceRepo := fs.NewInvoiceRepositoryFS(fsClient)
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
	saleRepo := fs.NewSaleRepositoryFS(fsClient)
	shippingAddressRepo := fs.NewShippingAddressRepositoryFS(fsClient)
	tokenBlueprintRepo := fs.NewTokenBlueprintRepositoryFS(fsClient)
	tokenOperationRepo := fs.NewTokenOperationRepositoryFS(fsClient)
	trackingRepo := fs.NewTrackingRepositoryFS(fsClient)
	userRepo := fs.NewUserRepositoryFS(fsClient)
	walletRepo := fs.NewWalletRepositoryFS(fsClient)

	// ★ PrintLog 用リポジトリ
	printLogRepo := fs.NewPrintLogRepositoryFS(fsClient)

	// ★ Inspection 用リポジトリ（inspections）
	inspectionRepo := fs.NewInspectionRepositoryFS(fsClient)

	// ★ History repositories
	productBlueprintHistoryRepo := fs.NewProductBlueprintHistoryRepositoryFS(fsClient)
	modelHistoryRepo := fs.NewModelHistoryRepositoryFS(fsClient)

	// ★ 招待トークン用 Repository（Firestore 実装）＋ Usecase 用アダプタ
	invitationTokenFSRepo := fs.NewInvitationTokenRepositoryFS(fsClient)
	invitationTokenUCRepo := &invitationTokenRepoAdapter{
		fsRepo: invitationTokenFSRepo,
	}

	// ★ Company / Brand 用ドメインサービス（表示名解決用）
	companySvc := companydom.NewService(companyRepo)
	brandSvc := branddom.NewService(brandRepo)

	// ★ Solana Brand Wallet Service（ブランド専用ウォレット開設 + 秘密鍵 SecretManager 保管）
	//   - 第2引数は Secret 名のプレフィックス（実装に合わせて調整）
	brandWalletSvc := solanainfra.NewBrandWalletService(cfg.FirestoreProjectID)

	// ★ member.Service（表示名解決用）
	memberSvc := memdom.NewService(memberRepo)

	// ★ productBlueprint.Service（ProductName / BrandID 解決用）
	pbDomainRepo := &productBlueprintDomainRepoAdapter{
		repo: productBlueprintRepo, // uc.ProductBlueprintRepo として扱う
	}
	pbSvc := productbpdom.NewService(pbDomainRepo)

	// ★ MintRequestPort（TokenUsecase 用）
	//   - mints ベースの MintRequestPort 実装
	mintRequestPort := fs.NewMintRequestPortFS(
		fsClient,
		"mints",            // mintsColName
		"token_blueprints", // tokenBlueprintsColName（実際の名前に合わせて）
		"brands",           // brandsColName（実際の名前に合わせて）
	)

	// ★ NameResolver（ID→名前/型番解決）
	//    TokenBlueprint はアダプタで value 戻りに揃える
	tokenBlueprintNameRepo := &tokenBlueprintNameRepoAdapter{
		repo: tokenBlueprintRepo,
	}

	nameResolver := resolver.NewNameResolver(
		brandRepo,              // BrandNameRepository
		companyRepo,            // CompanyNameRepository
		productBlueprintRepo,   // ProductBlueprintNameRepository
		memberRepo,             // MemberNameRepository
		modelRepo,              // ModelNumberRepository
		tokenBlueprintNameRepo, // TokenBlueprintNameRepository
	)

	// 5. Application-layer usecases

	// ★ TokenUsecase（Solana ミント権限ウォレット + MintRequestPort）
	var tokenUC *uc.TokenUsecase
	if mintKey != nil {
		solanaClient := solanainfra.NewMintClient(mintKey)
		tokenUC = uc.NewTokenUsecase(
			solanaClient,    // tokendom.MintAuthorityWalletPort
			mintRequestPort, // usecase.MintRequestPort
		)
	} else {
		// Mint 権限キーが取得できなかった場合でもコンテナ生成は続行
		tokenUC = uc.NewTokenUsecase(
			nil,             // tokendom.MintAuthorityWalletPort (nil 許容)
			mintRequestPort, // usecase.MintRequestPort
		)
	}

	accountUC := uc.NewAccountUsecase(accountRepo)

	announcementUC := uc.NewAnnouncementUsecase(
		announcementRepo,
		nil, // attachmentRepo not used yet
		nil, // object storage not wired yet
	)

	avatarUC := uc.NewAvatarUsecase(
		avatarRepo,
		nil, // state repo not wired yet
		nil, // icon repo not wired yet
		nil, // GCS not wired yet
	)

	billingAddressUC := uc.NewBillingAddressUsecase(billingAddressRepo)

	// ★ walletSvc 付き BrandUsecase
	brandUC := uc.NewBrandUsecaseWithWallet(brandRepo, memberRepo, brandWalletSvc)

	campaignUC := uc.NewCampaignUsecase(campaignRepo, nil, nil, nil)
	companyUC := uc.NewCompanyUsecase(companyRepo)
	discountUC := uc.NewDiscountUsecase(discountRepo)
	inquiryUC := uc.NewInquiryUsecase(inquiryRepo, nil, nil)
	inventoryUC := uc.NewInventoryUsecase(inventoryRepo)
	invoiceUC := uc.NewInvoiceUsecase(invoiceRepo)
	var listUC *uc.ListUsecase = nil
	memberUC := uc.NewMemberUsecase(memberRepo)
	messageUC := uc.NewMessageUsecase(messageRepo, nil, nil)

	// ★ ModelUsecase に HistoryRepo を注入
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

	// ★ ProductionUsecase（新パッケージ application/production）
	productionUC := productionapp.NewProductionUsecase(
		productionRepo, // ProductionRepo
		pbSvc,          // *productBlueprint.Service
		nameResolver,   // *resolver.NameResolver
	)

	// ★ ProductBlueprintUsecase に HistoryRepo を注入
	productBlueprintUC := uc.NewProductBlueprintUsecase(
		productBlueprintRepo,
		productBlueprintHistoryRepo,
	)

	// ★ products テーブル用アダプタ（inspection.Result → product.Result 変換）
	inspectionProductRepo := &inspectionProductRepoAdapter{
		repo: productRepo,
	}

	// ★ InspectionUsecase（検品アプリ専用）
	inspectionUC := uc.NewInspectionUsecase(
		inspectionRepo,        // inspections テーブル
		inspectionProductRepo, // products テーブル（inspectionResult 同期用, アダプタ経由）
	)

	// ★ ProductUsecase（Inspector 詳細画面用）
	productQueryRepo := &productQueryRepoAdapter{
		productRepo:          productRepo,
		modelRepo:            modelRepo,
		productionRepo:       productionRepo,
		productBlueprintRepo: productBlueprintRepo,
	}
	productUC := uc.NewProductUsecase(productQueryRepo, brandSvc, companySvc)

	// ★ MintUsecase（MintRequest / NFT 発行候補一覧など）
	// NewMintUsecase は
	// (mintProductBlueprintRepo, mintProductionRepo, mintInspectionRepo, mintModelRepo,
	//  mintTokenBlueprintRepo, *brand.Service, mint.MintRepository, mint.PassedProductLister,
	//  *resolver.NameResolver, usecase.TokenMintPort)
	// の 10 引数
	mintUC := uc.NewMintUsecase(
		productBlueprintRepo, // mint.MintProductBlueprintRepo
		productionRepo,       // mint.MintProductionRepo
		inspectionRepo,       // mint.MintInspectionRepo
		modelRepo,            // mint.MintModelRepo
		tokenBlueprintRepo,   // tokenBlueprint.RepositoryPort
		brandSvc,             // *brand.Service
		mintRepo,             // mint.MintRepository
		inspectionRepo,       // mint.PassedProductLister（InspectionRepositoryFS に実装追加済み前提）
		nameResolver,         // *resolver.NameResolver
		tokenUC,              // usecase.TokenMintPort（TokenUsecase のインスタンス）
	)

	saleUC := uc.NewSaleUsecase(saleRepo)
	shippingAddressUC := uc.NewShippingAddressUsecase(shippingAddressRepo)

	// ★ TokenBlueprint 用メタデータビルダー
	tokenMetadataBuilder := uc.NewTokenMetadataBuilder()

	// ★ ArweaveUploader
	//  まだ実装がなければ nil のまま運用し、Publish 時にエラーを返す想定
	var arweaveUploader uc.ArweaveUploader = nil

	// ★ TokenBlueprintUsecase に member.Service と Arweave 関連を注入
	tokenBlueprintUC := uc.NewTokenBlueprintUsecase(
		tokenBlueprintRepo,   // tbRepo
		nil,                  // tcRepo (token contents repo, 未接続なら nil)
		nil,                  // tiRepo (token icon repo, 未接続なら nil)
		memberSvc,            // *member.Service
		arweaveUploader,      // ArweaveUploader（現状 nil）
		tokenMetadataBuilder, // *TokenMetadataBuilder
	)

	tokenOperationUC := uc.NewTokenOperationUsecase(tokenOperationRepo)
	trackingUC := uc.NewTrackingUsecase(trackingRepo)
	userUC := uc.NewUserUsecase(userRepo)
	walletUC := uc.NewWalletUsecase(walletRepo)

	// ★ Invitation 用メールクライアント & メーラー
	invitationMailer := mailadp.NewInvitationMailerWithSendGrid(
		companySvc, // CompanyNameResolver
		brandSvc,   // BrandNameResolver
	)

	// ★ Invitation 用 Usecase（Query / Command）
	invitationQueryUC := uc.NewInvitationService(invitationTokenUCRepo, memberRepo)
	invitationCommandUC := uc.NewInvitationCommandService(
		invitationTokenUCRepo,
		memberRepo,
		invitationMailer, // ← SendGrid 経由でメール送信（会社名 + ブランド名表示）
	)

	// ★ auth/bootstrap 用 Usecase
	authBootstrapSvc := &authuc.BootstrapService{
		Members: &authMemberRepoAdapter{
			repo: memberRepo,
		},
		Companies: &authCompanyRepoAdapter{
			repo: companyRepo,
		},
	}

	// 6. Assemble container
	return &Container{
		Config:       cfg,
		Firestore:    fsClient,
		FirebaseApp:  fbApp,
		FirebaseAuth: fbAuth,
		MemberRepo:   memberRepo,
		MessageRepo:  messageRepo,

		// member.Service
		MemberService: memberSvc,

		// brand.Service
		BrandService: brandSvc,

		// History Repos
		ProductBlueprintHistoryRepo: productBlueprintHistoryRepo,
		ModelHistoryRepo:            modelHistoryRepo,

		AccountUC:          accountUC,
		AnnouncementUC:     announcementUC,
		AvatarUC:           avatarUC,
		BillingAddressUC:   billingAddressUC,
		BrandUC:            brandUC,
		CampaignUC:         campaignUC,
		CompanyUC:          companyUC,
		DiscountUC:         discountUC,
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
		SaleUC:             saleUC,
		ShippingAddressUC:  shippingAddressUC,
		TokenUC:            tokenUC,
		TokenBlueprintUC:   tokenBlueprintUC,
		TokenOperationUC:   tokenOperationUC,
		TrackingUC:         trackingUC,
		UserUC:             userUC,
		WalletUC:           walletUC,

		// 検品アプリ用
		ProductUC:    productUC,
		InspectionUC: inspectionUC,

		// Mint 系
		MintUC: mintUC,

		InvitationQuery:   invitationQueryUC,
		InvitationCommand: invitationCommandUC,

		AuthBootstrap: authBootstrapSvc,

		// Solana ミント権限鍵
		MintAuthorityKey: mintKey,

		// NameResolver
		NameResolver: nameResolver,
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
		DiscountUC:         c.DiscountUC,
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
		ProductionUC:       c.ProductionUC,
		ProductBlueprintUC: c.ProductBlueprintUC,
		SaleUC:             c.SaleUC,
		ShippingAddressUC:  c.ShippingAddressUC,
		TokenBlueprintUC:   c.TokenBlueprintUC,
		TokenOperationUC:   c.TokenOperationUC,
		TrackingUC:         c.TrackingUC,
		UserUC:             c.UserUC,
		WalletUC:           c.WalletUC,

		// 検品アプリ用 Usecase
		ProductUC:    c.ProductUC,
		InspectionUC: c.InspectionUC,

		// ★ Mint 用 Usecase
		MintUC: c.MintUC,

		// 招待関連 Usecase
		InvitationQuery:   c.InvitationQuery,
		InvitationCommand: c.InvitationCommand,

		// auth/bootstrap 用
		AuthBootstrap: c.AuthBootstrap,

		// AuthMiddleware 用
		FirebaseAuth: c.FirebaseAuth,
		MemberRepo:   c.MemberRepo,

		// ★ TokenBlueprintHandler で assigneeName 解決に使う
		MemberService: c.MemberService,

		// ★ TokenBlueprintHandler で brandName 解決に使う
		BrandService: c.BrandService,

		// ★ NameResolver（ID→名前/型番解決）
		NameResolver: c.NameResolver,

		// MessageHandler 用
		MessageRepo: c.MessageRepo,
	}
}

// ========================================
// Close
// ========================================
//
// Firestore クライアントを安全に閉じる。
func (c *Container) Close() error {
	if c.Firestore != nil {
		return c.Firestore.Close()
	}
	return nil
}
