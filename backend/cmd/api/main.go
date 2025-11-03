package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	_ "github.com/lib/pq"

	// inbound (primary adapters)
	httpin "narratives/internal/adapters/in/http"

	// outbound (secondary adapters / infrastructure persistence)
	db "narratives/internal/adapters/out/db"

	// application layer (use cases)
	uc "narratives/internal/application/usecase"

	// infra / platform config
	appcfg "narratives/internal/infra/config"
)

func main() {
	log.Println("[boot] starting service")

	// ─────────────────────────────────
	// 1. Load config
	// ─────────────────────────────────
	cfg := appcfg.Load()
	log.Printf("[boot] config loaded: port=%s db=%s\n", cfg.Port, redactDSN(cfg.DatabaseURL))

	// ─────────────────────────────────
	// 2. Root context
	// ─────────────────────────────────
	ctx := context.Background()

	// ─────────────────────────────────
	// 3. Initialize shared infra (Postgres)
	// ─────────────────────────────────
	pgDB, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Printf("[boot] FATAL: cannot open DB: %v", err)
		startHealthOnlyServer(cfg.Port)
		return
	}
	pgDB.SetMaxOpenConns(20)
	pgDB.SetConnMaxLifetime(30 * time.Minute)

	if err := pingDB(ctx, pgDB); err != nil {
		log.Printf("[boot] WARN: DB ping failed: %v (continuing in degraded mode)", err)
	}

	// ─────────────────────────────────
	// 4. Outbound adapters (DB repositories)
	//    各PGリポジトリは application/usecase が定義する Port を実装する想定。
	// ─────────────────────────────────
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

	inventoryRepo := db.NewInventoryRepositoryPG(pgDB)

	invoiceRepo := db.NewInvoiceRepositoryPG(pgDB)

	listRepo := db.NewListRepositoryPG(pgDB)
	listImageRepo := db.NewListImageRepositoryPG(pgDB)

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

	// ─────────────────────────────────
	// 5. Application layer (usecases)
	//    main.go は「どのPort実装を、どのUsecaseに渡しているか」を宣言する場所。
	//    ここでの引数は usecase 側の NewXxxUsecase シグネチャに合わせる。
	// ─────────────────────────────────

	accountUC := uc.NewAccountUsecase(
		accountRepo,
	)

	// AnnouncementUsecase は (annRepo, attRepo, objStore) の3引数が正。
	// まだGCS等のオブジェクトストレージAdapterが無いので objStore は nil で渡す。
	announcementUC := uc.NewAnnouncementUsecase(
		announcementRepo,
		announcementAttachmentRepo,
		nil, // <- aa.ObjectStoragePort (後でGCS等を実装したらここに渡す)
	)

	avatarUC := uc.NewAvatarUsecase(
		avatarRepo,
		avatarStateRepo,
		avatarIconRepo,
		/* objStore */ nil, // 画像アップロード用のオブジェクトストレージPortを後で実装
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
		/* imageStore */ nil, // 画像保存ポート(将来的にGCS等)
	)

	companyUC := uc.NewCompanyUsecase(
		companyRepo,
	)

	discountUC := uc.NewDiscountUsecase(
		discountRepo,
	)

	// Fulfillment / Payment / Tracking / etc. は OrderUC で使いたい想定なら
	// Usecase 側のNewOrderUsecaseに合わせて渡す。
	orderUC := uc.NewOrderUsecase(
		orderRepo,
		// 他のRepoが必要なら usecase.NewOrderUsecase の定義に合わせて渡す
	)

	inquiryUC := uc.NewInquiryUsecase(
		inquiryRepo,
		inquiryAttachmentRepo,
		/* attachmentStore */ nil,
	)

	inventoryUC := uc.NewInventoryUsecase(
		inventoryRepo,
	)

	invoiceUC := uc.NewInvoiceUsecase(
		invoiceRepo,
	)

	listUC := uc.NewListUsecase(
		listRepo,
		listImageRepo,
		/* listImageObjectStore */ nil,
	)

	memberUC := uc.NewMemberUsecase(
		memberRepo,
	)

	messageUC := uc.NewMessageUsecase(
		messageRepo,
		messageImageRepo,
		memberRepo,
		/* msgImageStore */ nil,
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
		tokenContentsRepo,
		tokenIconRepo,
		/* blueprintImageStore */ nil,
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

	// ─────────────────────────────────
	// 6. Inbound adapter (HTTP router)
	// ─────────────────────────────────
	router := httpin.NewRouter(httpin.RouterDeps{
		AccountUC:          accountUC,
		AnnouncementUC:     announcementUC,
		AvatarUC:           avatarUC,
		BillingAddressUC:   billingAddressUC,
		BrandUC:            brandUC,
		CampaignUC:         campaignUC,
		CompanyUC:          companyUC,
		DiscountUC:         discountUC,
		FulfillmentUC:      nil, // if you build uc.NewFulfillmentUsecase(...)
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
	})

	// ─────────────────────────────────
	// 7. HTTP server startup
	// ─────────────────────────────────
	port := cfg.Port
	if port == "" {
		port = "8080"
	}
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Printf("[boot] listening on :%s", port)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("[boot] server error: %v", err)
	}
}

// pingDB pings Postgres with timeout.
func pingDB(ctx context.Context, db *sql.DB) error {
	ctxPing, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return db.PingContext(ctxPing)
}

// startHealthOnlyServer spins up a degraded /healthz-only HTTP server.
func startHealthOnlyServer(port string) {
	if port == "" {
		port = "8080"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	log.Printf("[boot] degraded mode: health-only on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil && err != http.ErrServerClosed {
		log.Fatalf("[boot] health server error: %v", err)
	}
}

// redactDSN prevents leaking credentials in logs.
func redactDSN(dsn string) string {
	if dsn == "" {
		return ""
	}
	return "[REDACTED]"
}
