// backend/internal/platform/di/container.go
package di

import (
	"context"
	"log"
	"os"

	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"

	"cloud.google.com/go/firestore"

	httpin "narratives/internal/adapters/in/http"
	fs "narratives/internal/adapters/out/firestore"
	mailadp "narratives/internal/adapters/out/mail"
	uc "narratives/internal/application/usecase"
	authuc "narratives/internal/application/usecase/auth"
	memdom "narratives/internal/domain/member"
	appcfg "narratives/internal/infra/config"
)

// ========================================
// dev / stub EmailClient for InvitationMailer
// ========================================

type loggingEmailClient struct{}

func (c *loggingEmailClient) Send(
	_ context.Context,
	from,
	to,
	subject,
	body string,
) error {
	log.Printf("[mail] SEND from=%s to=%s subject=%s\n%s", from, to, subject, body)
	return nil
}

// ========================================
// Container (Firestore + Firebase edition)
// ========================================
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

	// ★ Auth サインアップ後の初期化（member + company 作成）用 BootstrapService
	AuthBootstrap *authuc.BootstrapService
}

// ========================================
// NewContainer
// ========================================
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

	// ★ 招待トークン用 Repository（Firestore 実装）
	invitationTokenRepo := fs.NewInvitationTokenRepositoryFS(fsClient)

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
	brandUC := uc.NewBrandUsecase(brandRepo)
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
	modelUC := uc.NewModelUsecase(modelRepo)
	orderUC := uc.NewOrderUsecase(orderRepo)
	paymentUC := uc.NewPaymentUsecase(paymentRepo)
	permissionUC := uc.NewPermissionUsecase(permissionRepo)
	productUC := uc.NewProductUsecase(productRepo)
	productionUC := uc.NewProductionUsecase(productionRepo)
	productBlueprintUC := uc.NewProductBlueprintUsecase(productBlueprintRepo)
	saleUC := uc.NewSaleUsecase(saleRepo)
	shippingAddressUC := uc.NewShippingAddressUsecase(shippingAddressRepo)
	tokenUC := uc.NewTokenUsecase(tokenRepo)
	tokenBlueprintUC := uc.NewTokenBlueprintUsecase(tokenBlueprintRepo, nil, nil)
	tokenOperationUC := uc.NewTokenOperationUsecase(tokenOperationRepo)
	trackingUC := uc.NewTrackingUsecase(trackingRepo)
	userUC := uc.NewUserUsecase(userRepo)
	walletUC := uc.NewWalletUsecase(walletRepo)

	// ★ Auth サインアップ後の初期化用 BootstrapService
	authBootstrap := &authuc.BootstrapService{
		Members:   memberRepo,
		Companies: companyRepo,
	}

	// ★ Invitation 用メールクライアント & メーラー
	//   SENDGRID_API_KEY / INVITATION_FROM_EMAIL / CONSOLE_BASE_URL が
	//   そろっている場合は SendGrid を使い、足りない場合はログ出力のみ。
	sendGridAPIKey := os.Getenv("SENDGRID_API_KEY")
	fromAddress := os.Getenv("INVITATION_FROM_EMAIL")
	consoleBaseURL := os.Getenv("CONSOLE_BASE_URL")

	var emailClient mailadp.EmailClient

	if sendGridAPIKey == "" || fromAddress == "" || consoleBaseURL == "" {
		log.Printf("[container] WARN: SendGrid env not fully set; using loggingEmailClient only")
		emailClient = &loggingEmailClient{}
		// 開発用デフォルト値（万一 env が空でも動くように）
		if fromAddress == "" {
			fromAddress = "no-reply@example.com"
		}
		if consoleBaseURL == "" {
			consoleBaseURL = "https://narratives.jp"
		}
	} else {
		log.Printf("[container] SendGrid client enabled for invitations (from=%s)", fromAddress)
		emailClient = mailadp.NewSendGridClient(sendGridAPIKey)
	}

	invitationMailer := mailadp.NewInvitationMailer(
		emailClient,
		fromAddress,
		consoleBaseURL,
	)

	// ★ Invitation 用 Usecase（Query / Command）
	invitationQueryUC := uc.NewInvitationService(invitationTokenRepo, memberRepo)
	invitationCommandUC := uc.NewInvitationCommandService(
		invitationTokenRepo,
		memberRepo,
		invitationMailer,
	)

	// 6. Assemble container
	return &Container{
		Config:       cfg,
		Firestore:    fsClient,
		FirebaseApp:  fbApp,
		FirebaseAuth: fbAuth,
		MemberRepo:   memberRepo,
		MessageRepo:  messageRepo,

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

		AuthBootstrap: authBootstrap,
	}, nil
}

// ========================================
// RouterDeps
// ========================================
// HTTP ルーターに必要な依存関係をまとめて返す。
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

		// ★ サインアップ後初期化（/auth/bootstrap 用）
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
// Firestore クライアントを安全に閉じる。
func (c *Container) Close() error {
	if c.Firestore != nil {
		return c.Firestore.Close()
	}
	return nil
}
