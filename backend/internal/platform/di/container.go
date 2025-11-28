// backend/internal/platform/di/container.go
package di

import (
	"context"
	"errors"
	"log"

	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"

	"cloud.google.com/go/firestore"

	httpin "narratives/internal/adapters/in/http"
	fs "narratives/internal/adapters/out/firestore"
	mailadp "narratives/internal/adapters/out/mail"
	uc "narratives/internal/application/usecase"
	authuc "narratives/internal/application/usecase/auth"
	branddom "narratives/internal/domain/brand" // ★ Brand 用ドメインサービス
	companydom "narratives/internal/domain/company"
	memdom "narratives/internal/domain/member"
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
	MintRequestUC      *uc.MintRequestUsecase
	ModelUC            *uc.ModelUsecase
	OrderUC            *uc.OrderUsecase
	PaymentUC          *uc.PaymentUsecase
	PermissionUC       *uc.PermissionUsecase
	ProductUC          *uc.ProductUsecase
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

	// ★ 招待関連 Usecase
	InvitationQuery   uc.InvitationQueryPort
	InvitationCommand uc.InvitationCommandPort

	// ★ auth/bootstrap 用 Usecase
	AuthBootstrap *authuc.BootstrapService
}

// ========================================
// NewContainer
// ========================================
//
// Firestore / Firebase クライアントを初期化し、各 Usecase を構築して返す。
func NewContainer(ctx context.Context) (*Container, error) {
	// 1. Load config
	cfg := appcfg.Load()

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
	mintRequestRepo := fs.NewMintRequestRepositoryFS(fsClient)
	modelRepo := fs.NewModelRepositoryFS(fsClient)
	orderRepo := fs.NewOrderRepositoryFS(fsClient)
	paymentRepo := fs.NewPaymentRepositoryFS(fsClient)
	permissionRepo := fs.NewPermissionRepositoryFS(fsClient)
	productRepo := fs.NewProductRepositoryFS(fsClient)
	productBlueprintRepo := fs.NewProductBlueprintRepositoryFS(fsClient)
	productionRepo := fs.NewProductionRepositoryFS(fsClient)
	saleRepo := fs.NewSaleRepositoryFS(fsClient)
	shippingAddressRepo := fs.NewShippingAddressRepositoryFS(fsClient)
	tokenRepo := fs.NewTokenRepositoryFS(fsClient)
	tokenBlueprintRepo := fs.NewTokenBlueprintRepositoryFS(fsClient)
	tokenOperationRepo := fs.NewTokenOperationRepositoryFS(fsClient)
	trackingRepo := fs.NewTrackingRepositoryFS(fsClient)
	userRepo := fs.NewUserRepositoryFS(fsClient)
	walletRepo := fs.NewWalletRepositoryFS(fsClient)

	// ★ History repositories
	productBlueprintHistoryRepo := fs.NewProductBlueprintHistoryRepositoryFS(fsClient)
	modelHistoryRepo := fs.NewModelHistoryRepositoryFS(fsClient)

	// ★ 招待トークン用 Repository（Firestore 実装）＋ Usecase 用アダプタ
	invitationTokenFSRepo := fs.NewInvitationTokenRepositoryFS(fsClient)
	invitationTokenUCRepo := &invitationTokenRepoAdapter{
		fsRepo: invitationTokenFSRepo,
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
	mintRequestUC := uc.NewMintRequestUsecase(mintRequestRepo)

	// ★ ModelUsecase に HistoryRepo を注入
	modelUC := uc.NewModelUsecase(modelRepo, modelHistoryRepo)

	orderUC := uc.NewOrderUsecase(orderRepo)
	paymentUC := uc.NewPaymentUsecase(paymentRepo)
	permissionUC := uc.NewPermissionUsecase(permissionRepo)
	productUC := uc.NewProductUsecase(productRepo)
	productionUC := uc.NewProductionUsecase(productionRepo)

	// ★ ProductBlueprintUsecase に HistoryRepo を注入
	productBlueprintUC := uc.NewProductBlueprintUsecase(
		productBlueprintRepo,
		productBlueprintHistoryRepo,
	)

	saleUC := uc.NewSaleUsecase(saleRepo)
	shippingAddressUC := uc.NewShippingAddressUsecase(shippingAddressRepo)
	tokenUC := uc.NewTokenUsecase(tokenRepo)
	tokenBlueprintUC := uc.NewTokenBlueprintUsecase(tokenBlueprintRepo, nil, nil)
	tokenOperationUC := uc.NewTokenOperationUsecase(tokenOperationRepo)
	trackingUC := uc.NewTrackingUsecase(trackingRepo)
	userUC := uc.NewUserUsecase(userRepo)
	walletUC := uc.NewWalletUsecase(walletRepo)

	// ★ Company / Brand 用ドメインサービス（表示名解決用）
	companySvc := companydom.NewService(companyRepo)
	brandSvc := branddom.NewService(brandRepo)

	// ★ Invitation 用メールクライアント & メーラー
	//   → ここでのみ SendGrid を使用（会社名 + ブランド名表示）
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
		MintRequestUC:      mintRequestUC,
		ModelUC:            modelUC,
		OrderUC:            orderUC,
		PaymentUC:          paymentUC,
		PermissionUC:       permissionUC,
		ProductUC:          productUC,
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

		InvitationQuery:   invitationQueryUC,
		InvitationCommand: invitationCommandUC,

		AuthBootstrap: authBootstrapSvc,
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
		MintRequestUC:      c.MintRequestUC,
		ModelUC:            c.ModelUC,
		OrderUC:            c.OrderUC,
		PaymentUC:          c.PaymentUC,
		PermissionUC:       c.PermissionUC,
		ProductUC:          c.ProductUC,
		ProductionUC:       c.ProductionUC,
		ProductBlueprintUC: c.ProductBlueprintUC,
		SaleUC:             c.SaleUC,
		ShippingAddressUC:  c.ShippingAddressUC,
		TokenUC:            c.TokenUC,
		TokenBlueprintUC:   c.TokenBlueprintUC,
		TokenOperationUC:   c.TokenOperationUC,
		TrackingUC:         c.TrackingUC,
		UserUC:             c.UserUC,
		WalletUC:           c.WalletUC,

		// ★ 招待関連 Usecase を Router に渡す
		InvitationQuery:   c.InvitationQuery,
		InvitationCommand: c.InvitationCommand,

		// ★ auth/bootstrap 用
		AuthBootstrap: c.AuthBootstrap,

		// AuthMiddleware 用
		FirebaseAuth: c.FirebaseAuth,
		MemberRepo:   c.MemberRepo,

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
