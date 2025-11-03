// internal/platform/di/container.go
package di

import (
	"context"
	"database/sql"
	"log"
	"time"

	_ "github.com/lib/pq"

	httpin "narratives/internal/adapters/in/http"
	db "narratives/internal/adapters/out/db"

	uc "narratives/internal/application/usecase"

	appcfg "narratives/internal/infra/config"
)

// Container centralizes dependency wiring (infra → repos → usecases).
// main.go can either build everything manually (what you have now, which compiles),
// or it can delegate that wiring to this Container.
type Container struct {
	// Infra
	Config *appcfg.Config
	DB     *sql.DB

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
	ListUC             *uc.ListUsecase // may be nil for now
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
}

// NewContainer builds everything similarly to main.go.
// NOTE: For now, ListUC is intentionally left nil to avoid interface
// wiring issues around ListReader/ListPatcher/etc. We'll bring it back
// once ListRepositoryPG and ListImageRepositoryPG are confirmed to
// satisfy all required ports (UpdateImageID, ListByListID, GetByID, ...).
func NewContainer(ctx context.Context) (*Container, error) {
	// 1. Load runtime config
	cfg := appcfg.Load()

	// 2. Connect to Postgres
	pgDB, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	pgDB.SetMaxOpenConns(20)
	pgDB.SetConnMaxLifetime(30 * time.Minute)

	// Best-effort ping (non-fatal warning if it fails)
	_ = pingDB(ctx, pgDB)

	// 3. Outbound adapters (repositories)
	accountRepo := db.NewAccountRepositoryPG(pgDB)

	announcementRepo := db.NewAnnouncementRepositoryPG(pgDB)
	announcementAttachmentRepo := db.NewAnnouncementAttachmentRepositoryPG(pgDB)

	avatarRepo := db.NewAvatarRepositoryPG(pgDB)
	avatarStateRepo := db.NewAvatarStateRepositoryPG(pgDB)
	avatarIconRepo := db.NewAvatarIconRepositoryPG(pgDB)

	billingAddressRepo := db.NewBillingAddressRepositoryPG(pgDB)

	brandRepo := db.NewBrandRepositoryPG(pgDB)

	campaignRepo := db.NewCampaignRepositoryPG(pgDB)
	campaignImageRepo := db.NewCampaignImageRepositoryPG(pgDB)
	campaignPerformanceRepo := db.NewCampaignPerformanceRepositoryPG(pgDB)

	companyRepo := db.NewCompanyRepositoryPG(pgDB)

	discountRepo := db.NewDiscountRepositoryPG(pgDB)

	inquiryRepo := db.NewInquiryRepositoryPG(pgDB)
	// We intentionally do NOT construct inquiryAttachmentRepo here,
	// because main.go (which compiles) didn't pass one to the UC either.

	inventoryRepo := db.NewInventoryRepositoryPG(pgDB)

	invoiceRepo := db.NewInvoiceRepositoryPG(pgDB)

	// ⚠ listRepo / listImageRepo intentionally omitted for now,
	// because ListUsecase wiring breaks on interface mismatch.
	// listRepo := db.NewListRepositoryPG(pgDB)
	// listImageRepo := db.NewListImageRepositoryPG(pgDB)

	memberRepo := db.NewMemberRepositoryPG(pgDB)

	messageRepo := db.NewMessageRepositoryPG(pgDB)
	messageImageRepo := db.NewMessageImageRepositoryPG(pgDB)

	mintRequestRepo := db.NewMintRequestRepositoryPG(pgDB)

	modelRepo := db.NewModelRepositoryPG(pgDB)

	orderRepo := db.NewOrderRepositoryPG(pgDB)

	paymentRepo := db.NewPaymentRepositoryPG(pgDB)

	permissionRepo := db.NewPermissionRepositoryPG(pgDB)

	productRepo := db.NewProductRepositoryPG(pgDB)
	productBlueprintRepo := db.NewProductBlueprintRepositoryPG(pgDB)

	productionRepo := db.NewProductionRepositoryPG(pgDB)

	saleRepo := db.NewSaleRepositoryPG(pgDB)

	shippingAddressRepo := db.NewShippingAddressRepositoryPG(pgDB)

	tokenRepo := db.NewTokenRepositoryPG(pgDB)
	tokenBlueprintRepo := db.NewTokenBlueprintRepositoryPG(pgDB)
	tokenContentsRepo := db.NewTokenContentsRepositoryPG(pgDB)
	tokenIconRepo := db.NewTokenIconRepositoryPG(pgDB)
	tokenOperationRepo := db.NewTokenOperationRepositoryPG(pgDB)

	trackingRepo := db.NewTrackingRepositoryPG(pgDB)

	userRepo := db.NewUserRepositoryPG(pgDB)

	walletRepo := db.NewWalletRepositoryPG(pgDB)

	// 4. Application layer usecases

	accountUC := uc.NewAccountUsecase(
		accountRepo,
	)

	announcementUC := uc.NewAnnouncementUsecase(
		announcementRepo,
		announcementAttachmentRepo,
		nil, // ObjectStoragePort for announcement attachments (GCS etc.) not wired yet
	)

	avatarUC := uc.NewAvatarUsecase(
		avatarRepo,
		avatarStateRepo,
		avatarIconRepo,
		nil, // avatar image object storage not wired yet
	)

	billingAddressUC := uc.NewBillingAddressUsecase(
		billingAddressRepo,
	)

	brandUC := uc.NewBrandUsecase(
		brandRepo,
	)

	campaignUC := uc.NewCampaignUsecase(
		campaignRepo,
		campaignImageRepo,
		campaignPerformanceRepo,
		nil, // campaign image object storage not wired yet
	)

	companyUC := uc.NewCompanyUsecase(
		companyRepo,
	)

	discountUC := uc.NewDiscountUsecase(
		discountRepo,
	)

	orderUC := uc.NewOrderUsecase(
		orderRepo,
		// (extend here if NewOrderUsecase signature expands later)
	)

	inquiryUC := uc.NewInquiryUsecase(
		inquiryRepo,
		nil, // inquiryAttachmentRepo not passed in main.go either
		nil, // attachment object storage not wired
	)

	inventoryUC := uc.NewInventoryUsecase(
		inventoryRepo,
	)

	invoiceUC := uc.NewInvoiceUsecase(
		invoiceRepo,
	)

	// ─────────────────────────────
	// ListUC
	// ─────────────────────────────
	// main.go (the "source of truth") currently compiles.
	// container.go was failing because the ListXxx* repos here in this file
	// don't satisfy all List* interfaces (ListPatcher.UpdateImageID, etc.).
	//
	// Until we unify those interfaces or write adapter wrappers,
	// we keep ListUC nil in the container. This keeps container.go buildable
	// without fighting interface completeness right now.
	var listUC *uc.ListUsecase = nil

	memberUC := uc.NewMemberUsecase(
		memberRepo,
	)

	// MessageUsecase signature (confirmed from main.go):
	//   NewMessageUsecase(message.Repository,
	//                     messageImage.RepositoryPort,
	//                     messageImage.ObjectStoragePort)
	messageUC := uc.NewMessageUsecase(
		messageRepo,
		messageImageRepo,
		nil, // message image object storage (e.g. GCS) not wired yet
	)

	mintRequestUC := uc.NewMintRequestUsecase(
		mintRequestRepo,
	)

	modelUC := uc.NewModelUsecase(
		modelRepo,
	)

	paymentUC := uc.NewPaymentUsecase(
		paymentRepo,
	)

	permissionUC := uc.NewPermissionUsecase(
		permissionRepo,
	)

	productUC := uc.NewProductUsecase(
		productRepo,
	)

	productionUC := uc.NewProductionUsecase(
		productionRepo,
	)

	productBlueprintUC := uc.NewProductBlueprintUsecase(
		productBlueprintRepo,
	)

	saleUC := uc.NewSaleUsecase(
		saleRepo,
	)

	shippingAddressUC := uc.NewShippingAddressUsecase(
		shippingAddressRepo,
	)

	tokenUC := uc.NewTokenUsecase(
		tokenRepo,
	)

	tokenBlueprintUC := uc.NewTokenBlueprintUsecase(
		tokenBlueprintRepo,
		tokenContentsRepo,
		tokenIconRepo,
	)

	tokenOperationUC := uc.NewTokenOperationUsecase(
		tokenOperationRepo,
	)

	trackingUC := uc.NewTrackingUsecase(
		trackingRepo,
	)

	userUC := uc.NewUserUsecase(
		userRepo,
	)

	walletUC := uc.NewWalletUsecase(
		walletRepo,
	)

	// 5. Return assembled container
	return &Container{
		Config: cfg,
		DB:     pgDB,

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
		ListUC:             listUC, // (currently nil)
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
	}, nil
}

// RouterDeps exposes the deps bundle for the HTTP router,
// mirroring main.go's final httpin.NewRouter(...) call.
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
		FulfillmentUC:      nil, // not wired yet
		InquiryUC:          c.InquiryUC,
		InventoryUC:        c.InventoryUC,
		InvoiceUC:          c.InvoiceUC,
		ListUC:             c.ListUC, // will be nil until we wire it properly
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
	}
}

// Close releases shared infra, mainly the DB connection pool.
func (c *Container) Close() error {
	if c.DB != nil {
		return c.DB.Close()
	}
	return nil
}

// pingDB is best-effort DB health check at boot.
func pingDB(ctx context.Context, dbConn *sql.DB) error {
	ctxPing, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := dbConn.PingContext(ctxPing); err != nil {
		log.Printf("[container] WARN: DB ping failed: %v", err)
		return err
	}
	return nil
}
