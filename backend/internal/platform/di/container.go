// backend/internal/platform/di/container.go
package di

import (
	"context"
	"errors"
	"log"
	"strings"

	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"

	solanainfra "narratives/internal/infra/solana"

	"cloud.google.com/go/firestore"

	httpin "narratives/internal/adapters/in/http"
	fs "narratives/internal/adapters/out/firestore"
	mailadp "narratives/internal/adapters/out/mail"
	uc "narratives/internal/application/usecase"
	authuc "narratives/internal/application/usecase/auth"
	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
	inspectiondom "narratives/internal/domain/inspection"
	memdom "narratives/internal/domain/member"
	modeldom "narratives/internal/domain/model"     // ★ productQueryRepoAdapter / modelNumberRepoAdapter 用
	productdom "narratives/internal/domain/product" // ★ productQueryRepoAdapter 用
	productbpdom "narratives/internal/domain/productBlueprint"
	appcfg "narratives/internal/infra/config"
)

//
// ========================================
// auth.BootstrapService 用アダプタ
// ========================================
//

// memdom.Repository → auth.MemberRepository
type authMemberRepoAdapter struct {
	repo memdom.Repository
}

// Save: *member を memdom.Repository.Save に委譲
func (a *authMemberRepoAdapter) Save(ctx context.Context, m *memdom.Member) error {
	if m == nil {
		return errors.New("authMemberRepoAdapter.Save: nil member")
	}
	saved, err := a.repo.Save(ctx, *m, nil)
	if err != nil {
		return err
	}
	// Save 側で CreatedAt / UpdatedAt などが上書きされた場合に反映しておく
	*m = saved
	return nil
}

// GetByID: 値戻りをポインタに変換
func (a *authMemberRepoAdapter) GetByID(ctx context.Context, id string) (*memdom.Member, error) {
	v, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// CompanyRepositoryFS → auth.CompanyRepository
type authCompanyRepoAdapter struct {
	repo *fs.CompanyRepositoryFS
}

// NewID: Firestore の companies コレクションから DocID を採番
func (a *authCompanyRepoAdapter) NewID(ctx context.Context) (string, error) {
	if a.repo == nil || a.repo.Client == nil {
		return "", errors.New("authCompanyRepoAdapter.NewID: repo or client is nil")
	}
	doc := a.repo.Client.Collection("companies").NewDoc()
	return doc.ID, nil
}

// Save: companydom.Company を CompanyRepositoryFS.Save に委譲
func (a *authCompanyRepoAdapter) Save(ctx context.Context, c *companydom.Company) error {
	if c == nil {
		return errors.New("authCompanyRepoAdapter.Save: nil company")
	}
	saved, err := a.repo.Save(ctx, *c, nil)
	if err != nil {
		return err
	}
	*c = saved
	return nil
}

//
// ========================================
// InvitationTokenRepository 用アダプタ
// ========================================
//
// Firestore 実装 (*fs.InvitationTokenRepositoryFS) を
// usecase.InvitationTokenRepository に合わせてラップする。
//   - ResolveInvitationInfoByToken
//   - CreateInvitationToken
//

type invitationTokenRepoAdapter struct {
	fsRepo *fs.InvitationTokenRepositoryFS
}

// ResolveInvitationInfoByToken は token から InvitationInfo を取得します。
func (a *invitationTokenRepoAdapter) ResolveInvitationInfoByToken(
	ctx context.Context,
	token string,
) (memdom.InvitationInfo, error) {
	if a.fsRepo == nil {
		return memdom.InvitationInfo{}, errors.New("invitationTokenRepoAdapter.ResolveInvitationInfoByToken: fsRepo is nil")
	}

	it, err := a.fsRepo.FindByToken(ctx, token)
	if err != nil {
		return memdom.InvitationInfo{}, err
	}

	return memdom.InvitationInfo{
		MemberID:         it.MemberID,
		CompanyID:        it.CompanyID,
		AssignedBrandIDs: it.AssignedBrandIDs,
		Permissions:      it.Permissions,
	}, nil
}

// CreateInvitationToken は InvitationInfo を受け取り、
// Firestore 側に招待トークンを作成して token 文字列を返します。
func (a *invitationTokenRepoAdapter) CreateInvitationToken(
	ctx context.Context,
	info memdom.InvitationInfo,
) (string, error) {
	if a.fsRepo == nil {
		return "", errors.New("invitationTokenRepoAdapter.CreateInvitationToken: fsRepo is nil")
	}
	// FS 実装は既に (ctx, member.InvitationInfo) を受け取るように
	// 変更済みという前提で、そのまま委譲する。
	return a.fsRepo.CreateInvitationToken(ctx, info)
}

// ========================================
// productBlueprint ドメインサービス用アダプタ
// ========================================
//
// fs.ProductBlueprintRepositoryFS（= uc.ProductBlueprintRepo 実装）を
// productBlueprint.Service の期待する productBlueprint.Repository に
// 合わせるための薄いアダプタです。
// Service 側では GetByID しか使わない前提。
type productBlueprintDomainRepoAdapter struct {
	repo uc.ProductBlueprintRepo
}

func (a *productBlueprintDomainRepoAdapter) GetByID(
	ctx context.Context,
	id string,
) (productbpdom.ProductBlueprint, error) {
	if a == nil || a.repo == nil {
		return productbpdom.ProductBlueprint{}, productbpdom.ErrInternal
	}
	return a.repo.GetByID(ctx, id)
}

// ========================================
// ModelNumberRepo アダプタ
// ========================================
//
// InspectionUsecase が期待する uc.ModelNumberRepo を
// Firestore の ModelRepositoryFS に接続するアダプタ。
// modelId → modelNumber の解決を担当する。
type modelNumberRepoAdapter struct {
	modelRepo *fs.ModelRepositoryFS
}

// GetModelVariationByID は interface usecase.ModelNumberRepo が要求するメソッド。
// modelID から ModelVariation（値）を返します。
func (a *modelNumberRepoAdapter) GetModelVariationByID(
	ctx context.Context,
	modelID string,
) (modeldom.ModelVariation, error) {
	if a == nil || a.modelRepo == nil {
		return modeldom.ModelVariation{}, errors.New("modelNumberRepoAdapter: modelRepo is nil")
	}

	id := strings.TrimSpace(modelID)
	if id == "" {
		return modeldom.ModelVariation{}, modeldom.ErrInvalidID
	}

	mv, err := a.modelRepo.GetModelVariationByID(ctx, id)
	if err != nil {
		return modeldom.ModelVariation{}, err
	}
	if mv == nil {
		return modeldom.ModelVariation{}, modeldom.ErrVariationNotFound
	}
	return *mv, nil
}

// （オプション）modelID から直接 modelNumber を解決したい場合用のヘルパ。
// usecase 側で使っていなければコンパイルには影響しない。
func (a *modelNumberRepoAdapter) GetModelNumberByModelID(
	ctx context.Context,
	modelID string,
) (string, error) {
	mv, err := a.GetModelVariationByID(ctx, modelID)
	if err != nil {
		return "", err
	}

	num := strings.TrimSpace(mv.ModelNumber)
	if num == "" {
		return "", modeldom.ErrInvalidModelNumber
	}
	return num, nil
}

// ========================================
// inspection 用: products.UpdateInspectionResult アダプタ
// ========================================
//
// usecase.ProductInspectionRepo が期待する
//
//	UpdateInspectionResult(ctx, productID string, result inspection.InspectionResult)
//
// を、ProductRepositoryFS が持つ
//
//	UpdateInspectionResult(ctx, productID string, result product.InspectionResult)
//
// に橋渡しする。
type inspectionProductRepoAdapter struct {
	repo interface {
		UpdateInspectionResult(ctx context.Context, productID string, result productdom.InspectionResult) error
	}
}

// InspectionUsecase.ProductInspectionRepo を満たす
func (a *inspectionProductRepoAdapter) UpdateInspectionResult(
	ctx context.Context,
	productID string,
	result inspectiondom.InspectionResult,
) error {
	if a == nil || a.repo == nil {
		return errors.New("inspectionProductRepoAdapter: repo is nil")
	}
	// inspection.InspectionResult → product.InspectionResult に変換して委譲
	return a.repo.UpdateInspectionResult(ctx, productID, productdom.InspectionResult(result))
}

// ========================================
// ProductUsecase 用 ProductQueryRepo アダプタ
// ========================================
//
// 既存の Firestore Repository 群を束ねて usecase.ProductQueryRepo を実装します。
// - productRepo          → products 取得
// - modelRepo            → model variations 取得
// - productionRepo       → productions 取得
// - productBlueprintRepo → product_blueprints 取得
type productQueryRepoAdapter struct {
	productRepo          *fs.ProductRepositoryFS
	modelRepo            *fs.ModelRepositoryFS
	productionRepo       *fs.ProductionRepositoryFS
	productBlueprintRepo *fs.ProductBlueprintRepositoryFS
}

// GetProductByID implements usecase.ProductQueryRepo.
func (a *productQueryRepoAdapter) GetProductByID(
	ctx context.Context,
	productID string,
) (productdom.Product, error) {
	if a == nil || a.productRepo == nil {
		return productdom.Product{}, errors.New("productQueryRepoAdapter: productRepo is nil")
	}
	return a.productRepo.GetByID(ctx, productID)
}

// GetModelByID implements usecase.ProductQueryRepo.
func (a *productQueryRepoAdapter) GetModelByID(
	ctx context.Context,
	modelID string,
) (modeldom.ModelVariation, error) {
	if a == nil || a.modelRepo == nil {
		return modeldom.ModelVariation{}, errors.New("productQueryRepoAdapter: modelRepo is nil")
	}
	mv, err := a.modelRepo.GetModelVariationByID(ctx, modelID)
	if err != nil {
		return modeldom.ModelVariation{}, err
	}
	if mv == nil {
		return modeldom.ModelVariation{}, errors.New("productQueryRepoAdapter: modelRepo returned nil model variation")
	}
	return *mv, nil
}

// GetProductionByID implements usecase.ProductQueryRepo.
func (a *productQueryRepoAdapter) GetProductionByID(
	ctx context.Context,
	productionID string,
) (interface{}, error) {
	if a == nil || a.productionRepo == nil {
		return nil, errors.New("productQueryRepoAdapter: productionRepo is nil")
	}
	// productiondom.Production 型を interface{} として返す
	return a.productionRepo.GetByID(ctx, productionID)
}

// GetProductBlueprintByID implements usecase.ProductQueryRepo.
func (a *productQueryRepoAdapter) GetProductBlueprintByID(
	ctx context.Context,
	bpID string,
) (productbpdom.ProductBlueprint, error) {
	if a == nil || a.productBlueprintRepo == nil {
		return productbpdom.ProductBlueprint{}, errors.New("productQueryRepoAdapter: productBlueprintRepo is nil")
	}
	return a.productBlueprintRepo.GetByID(ctx, bpID)
}

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
	ProductionUC       *uc.ProductionUsecase
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

	// ★ member.Service（表示名解決用）
	memberSvc := memdom.NewService(memberRepo)

	// ★ productBlueprint.Service（ProductName / BrandID 解決用）
	pbDomainRepo := &productBlueprintDomainRepoAdapter{
		repo: productBlueprintRepo, // uc.ProductBlueprintRepo として扱う
	}
	pbSvc := productbpdom.NewService(pbDomainRepo)

	// ★ modelId → modelNumber 解決用 Repo アダプタ
	modelNumberRepo := &modelNumberRepoAdapter{
		modelRepo: modelRepo,
	}

	// 5. Application-layer usecases
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
	brandUC := uc.NewBrandUsecase(brandRepo, memberRepo)
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

	// ★ PrintUsecase に PrintLogRepo + InspectionRepo を注入
	printUC := uc.NewPrintUsecase(
		productRepo,
		printLogRepo,
		inspectionRepo,
		modelNumberRepo,
		productBlueprintRepo,
	)

	// ★ ProductionUsecase に member.Service + productBlueprint.Service + brand.Service を注入
	productionUC := uc.NewProductionUsecase(
		productionRepo,
		memberSvc,
		pbSvc,
		brandSvc,
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
		modelNumberRepo,       // modelId → modelNumber 解決用
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
	// (mintProductBlueprintRepo, mintProductionRepo, mintInspectionRepo, mintModelRepo, mintTokenBlueprintRepo, *brand.Service)
	// の 6 引数
	mintUC := uc.NewMintUsecase(
		productBlueprintRepo,
		productionRepo,
		inspectionRepo,
		modelRepo,
		tokenBlueprintRepo, // ★ 追加：TokenBlueprint 用リポジトリ
		brandSvc,
	)

	saleUC := uc.NewSaleUsecase(saleRepo)
	shippingAddressUC := uc.NewShippingAddressUsecase(shippingAddressRepo)

	// ★ TokenBlueprintUsecase に member.Service を注入
	tokenBlueprintUC := uc.NewTokenBlueprintUsecase(
		tokenBlueprintRepo,
		nil,
		nil,
		memberSvc,
	)

	tokenOperationUC := uc.NewTokenOperationUsecase(tokenOperationRepo)
	trackingUC := uc.NewTrackingUsecase(trackingRepo)
	userUC := uc.NewUserUsecase(userRepo)
	walletUC := uc.NewWalletUsecase(walletRepo)

	// ★ TokenUsecase（Solana ミント権限ウォレットを使用）
	var tokenUC *uc.TokenUsecase
	if mintKey != nil {
		solanaClient := solanainfra.NewMintClient(mintKey)
		// MintRequestPort はまだ実装していないので nil を渡しておき、後で接続する
		tokenUC = uc.NewTokenUsecase(solanaClient, nil)
	} else {
		// Mint 権限キーが取得できなかった場合でもコンテナ生成は続行
		tokenUC = uc.NewTokenUsecase(nil, nil)
	}

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
