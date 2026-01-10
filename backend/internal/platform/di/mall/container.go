// backend/internal/platform/di/mall/container.go
package mall

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"

	// inbound (query + resolver types)
	mallquery "narratives/internal/application/query/mall"
	appresolver "narratives/internal/application/resolver"
	usecase "narratives/internal/application/usecase"

	// inbound (for ImageURLResolver interface type)
	mallhandler "narratives/internal/adapters/in/http/mall/handler"

	// outbound
	outfs "narratives/internal/adapters/out/firestore"
	mallfs "narratives/internal/adapters/out/firestore/mall"
	gcso "narratives/internal/adapters/out/gcs"
	gcscommon "narratives/internal/adapters/out/gcs/common"
	httpout "narratives/internal/adapters/out/http"

	// ✅ Solana wallet service
	solanainfra "narratives/internal/infra/solana"

	// domains
	ldom "narratives/internal/domain/list"

	shared "narratives/internal/platform/di/shared"
)

const (
	StripeWebhookPath        = "/mall/webhooks/stripe"
	defaultTokenIconBucketDI = "narratives-development_token_icon" // tokenIcon_repository_gcs.go と同じ既定値
)

// Container is Mall DI container.
// Pure DI: build deps only. No routing branching, no reflection tricks.
type Container struct {
	Infra *shared.Infra

	// Usecases (mall-facing)
	AvatarUC          *usecase.AvatarUsecase // ✅ /mall/avatars 用
	ListUC            *usecase.ListUsecase
	ShippingAddressUC *usecase.ShippingAddressUsecase
	BillingAddressUC  *usecase.BillingAddressUsecase
	UserUC            *usecase.UserUsecase
	WalletUC          *usecase.WalletUsecase
	CartUC            *usecase.CartUsecase
	PaymentUC         *usecase.PaymentUsecase
	OrderUC           *usecase.OrderUsecase
	InvoiceUC         *usecase.InvoiceUsecase

	// ✅ Case A: /mall/me/payments で payment 起票後に（必要なら）webhook を叩くオーケストレーション
	PaymentFlowUC *usecase.PaymentFlowUsecase

	// ✅ Inventory (buyer-facing, read-only)
	InventoryUC *usecase.InventoryUsecase

	// ✅ TokenBlueprint (buyer-facing patch handler 用に repo を保持)
	TokenBlueprintRepo any

	// ✅ Token Icon public URL resolver（objectPath -> public URL）
	TokenIconURLResolver mallhandler.ImageURLResolver

	// Optional resolver (for query enrich)
	NameResolver *appresolver.NameResolver

	// Queries (mall-facing)
	CatalogQ *mallquery.CatalogQuery
	CartQ    *mallquery.CartQuery
	PreviewQ *mallquery.PreviewQuery
	OrderQ   any

	// Repos sometimes needed by handlers/queries/joins
	ListRepo ldom.Repository

	// ✅ /mall/me/avatar 用: uid -> avatarId を解決するRepo
	MeAvatarRepo *mallfs.MeAvatarRepo

	// ✅ 追加: Avatar作成時に「空カートを作る」等で handler が repo を直接必要とするケースに備える
	CartRepo any
}

func NewContainer(ctx context.Context, infra *shared.Infra) (*Container, error) {
	// shared infra
	if infra == nil {
		var err error
		infra, err = shared.NewInfra(ctx)
		if err != nil {
			return nil, err
		}
	}
	if infra == nil {
		return nil, errors.New("di.mall: shared infra is nil")
	}

	// ✅ IMPORTANT: console と同様に Config は必須（projectID 解決に必要）
	if infra.Config == nil {
		return nil, errors.New("di.mall: shared infra config is nil")
	}

	// required clients
	fsClient := infra.Firestore
	if fsClient == nil {
		return nil, errors.New("di.mall: infra.Firestore is nil")
	}
	gcsClient := infra.GCS
	if gcsClient == nil {
		return nil, errors.New("di.mall: infra.GCS is nil")
	}

	c := &Container{Infra: infra}

	// --------------------------------------------------------
	// Firestore repositories
	// --------------------------------------------------------
	avatarRepo := outfs.NewAvatarRepositoryFS(fsClient)
	avatarStateRepo := outfs.NewAvatarStateRepositoryFS(fsClient)

	shippingAddressRepo := outfs.NewShippingAddressRepositoryFS(fsClient)
	billingAddressRepo := outfs.NewBillingAddressRepositoryFS(fsClient)
	userRepo := outfs.NewUserRepositoryFS(fsClient)
	walletRepo := outfs.NewWalletRepositoryFS(fsClient)

	cartRepo := outfs.NewCartRepositoryFS(fsClient)
	paymentRepo := outfs.NewPaymentRepositoryFS(fsClient)
	orderRepo := outfs.NewOrderRepositoryFS(fsClient)
	invoiceRepo := outfs.NewInvoiceRepositoryFS(fsClient)

	// handler 側に注入できるように保持
	c.CartRepo = cartRepo

	// Inventory
	inventoryRepo := outfs.NewInventoryRepositoryFS(fsClient)

	// TokenBlueprint
	tokenBlueprintRepo := outfs.NewTokenBlueprintRepositoryFS(fsClient)
	c.TokenBlueprintRepo = tokenBlueprintRepo

	// /mall/me/avatar 用 Repo（uid -> avatarId）
	c.MeAvatarRepo = mallfs.NewMeAvatarRepo(fsClient)

	// List repo
	listRepoFS := outfs.NewListRepositoryFS(fsClient)
	c.ListRepo = listRepoFS

	// List repo for usecase ports
	listRepoForUC := outfs.NewListRepositoryForUsecase(listRepoFS)

	// --------------------------------------------------------
	// GCS repositories
	// --------------------------------------------------------
	listImageRepo := gcso.NewListImageRepositoryGCS(gcsClient, infra.ListImageBucket)

	// AvatarIcon (GCS)
	avatarIconRepo := gcso.NewAvatarIconRepositoryGCS(gcsClient, "")

	// ListPatcher
	listPatcher := mallfs.NewListPatcherRepo(fsClient)

	// --------------------------------------------------------
	// ✅ Solana wallet service (AvatarWalletService)
	// --------------------------------------------------------
	projectID := strings.TrimSpace(infra.Config.FirestoreProjectID)
	if projectID == "" {
		projectID = strings.TrimSpace(os.Getenv("FIRESTORE_PROJECT_ID"))
	}
	if projectID == "" {
		projectID = strings.TrimSpace(os.Getenv("GOOGLE_CLOUD_PROJECT"))
	}
	avatarWalletSvc := solanainfra.NewAvatarWalletService(projectID)

	// --------------------------------------------------------
	// ✅ Case A: Self webhook trigger client (outbound)
	// --------------------------------------------------------
	// Cloud Run / local の自分自身の base URL を指定する
	// 例:
	//   SELF_BASE_URL=https://xxxxx.asia-northeast1.run.app
	//   SELF_BASE_URL=http://localhost:8080
	//
	// NOTE:
	// - 未設定でも Mall 全体は起動させる（PaymentFlowUC だけ "trigger=nil" で動かす/無効化できる）
	selfBaseURL := strings.TrimSpace(os.Getenv("SELF_BASE_URL"))
	selfBaseURL = strings.TrimRight(selfBaseURL, "/")
	selfBaseURLConfigured := selfBaseURL != ""
	if !selfBaseURLConfigured {
		log.Printf("[di.mall] WARN: SELF_BASE_URL is empty; webhook trigger will be disabled (PaymentFlowUC will still exist if PaymentUC exists)")
	}

	// --------------------------------------------------------
	// Usecases
	// --------------------------------------------------------

	// ✅ AvatarUsecase: cartRepo / walletRepo / walletSvc を必ず注入（500解消）
	c.AvatarUC = usecase.NewAvatarUsecase(
		avatarRepo,
		avatarStateRepo,
		avatarIconRepo, // AvatarIconRepo
		avatarIconRepo, // AvatarIconObjectStoragePort
	).
		WithCartRepo(cartRepo).
		WithWalletRepo(walletRepo).
		WithWalletService(avatarWalletSvc)

	c.ListUC = usecase.NewListUsecase(
		listRepoForUC,
		listPatcher,
		listImageRepo,
		listImageRepo,
		listImageRepo,
	)

	c.ShippingAddressUC = usecase.NewShippingAddressUsecase(shippingAddressRepo)
	c.BillingAddressUC = usecase.NewBillingAddressUsecase(billingAddressRepo)
	c.UserUC = usecase.NewUserUsecase(userRepo)
	c.WalletUC = usecase.NewWalletUsecase(walletRepo)

	c.CartUC = usecase.NewCartUsecase(cartRepo)

	// ✅ payment 起票後に invoice.paid=true を立てるため invoiceRepo を注入
	c.PaymentUC = usecase.NewPaymentUsecase(paymentRepo).WithInvoiceRepoForPayment(invoiceRepo)

	c.InvoiceUC = usecase.NewInvoiceUsecase(invoiceRepo)
	c.OrderUC = usecase.NewOrderUsecase(orderRepo)

	// ✅ Case A: PaymentFlowUsecase（payment起票 + 必要なら webhook trigger）
	// - NewPaymentFlowUsecase は "PaymentUsecase" を取る（InvoiceUsecaseではない）
	// - selfBaseURL が無い場合は trigger=nil（＝外部から webhook が来る運用/または dev で trigger しない）
	if c.PaymentUC != nil {
		if selfBaseURLConfigured {
			stripeTrigger := httpout.NewStripeWebhookClient(selfBaseURL)
			c.PaymentFlowUC = usecase.NewPaymentFlowUsecase(c.PaymentUC, stripeTrigger)
		} else {
			c.PaymentFlowUC = usecase.NewPaymentFlowUsecase(c.PaymentUC, nil)
		}
	} else {
		c.PaymentFlowUC = nil
	}

	c.InventoryUC = usecase.NewInventoryUsecase(inventoryRepo)

	// --------------------------------------------------------
	// TokenIcon URL Resolver (objectPath -> publicURL)
	// --------------------------------------------------------
	{
		b := strings.TrimSpace(os.Getenv("TOKEN_ICON_BUCKET"))
		c.TokenIconURLResolver = tokenIconURLResolver{bucket: b}
	}

	// --------------------------------------------------------
	// NameResolver (optional but useful for mall queries)
	// --------------------------------------------------------
	{
		brandRepo := outfs.NewBrandRepositoryFS(fsClient)
		companyRepo := outfs.NewCompanyRepositoryFS(fsClient)
		productBlueprintRepo := outfs.NewProductBlueprintRepositoryFS(fsClient)
		memberRepo := outfs.NewMemberRepositoryFS(fsClient)
		modelRepo := outfs.NewModelRepositoryFS(fsClient)

		// tokenBlueprintNameRepoAdapter は di/mall/adapter.go に存在
		tbNameRepo := &tokenBlueprintNameRepoAdapter{repo: tokenBlueprintRepo}

		c.NameResolver = appresolver.NewNameResolver(
			brandRepo,
			companyRepo,
			productBlueprintRepo,
			memberRepo,
			modelRepo,
			tbNameRepo,
		)
	}

	// --------------------------------------------------------
	// Queries (mall-facing)
	// --------------------------------------------------------
	{
		invRepo := mallfs.NewInventoryRepoForMallQuery(fsClient)

		pbRepo := outfs.NewProductBlueprintRepositoryFS(fsClient)
		modelRepo := outfs.NewModelRepositoryFS(fsClient)

		c.CatalogQ = mallquery.NewCatalogQuery(listRepoFS, invRepo, pbRepo, modelRepo)

		c.CartQ = mallquery.NewCartQuery(fsClient)
		c.PreviewQ = mallquery.NewPreviewQuery(fsClient)
		c.OrderQ = mallquery.NewOrderQuery(fsClient)

		// light injection
		if c.CatalogQ != nil && c.NameResolver != nil && c.CatalogQ.NameResolver == nil {
			c.CatalogQ.NameResolver = c.NameResolver
		}
		if c.CartQ != nil && c.NameResolver != nil && c.CartQ.Resolver == nil {
			c.CartQ.Resolver = c.NameResolver
		}
		if c.CartQ != nil && c.ListRepo != nil && c.CartQ.ListRepo == nil {
			c.CartQ.ListRepo = c.ListRepo
		}
		if c.PreviewQ != nil && c.NameResolver != nil && c.PreviewQ.Resolver == nil {
			c.PreviewQ.Resolver = c.NameResolver
		}
	}

	log.Printf(
		"[di.mall] container built (firestore=%t gcs=%t firebaseAuth=%t avatarUC=%t avatarWalletSvc=%t cartUC=%t cartRepo=%t paymentUC=%t paymentFlowUC=%t invoiceUC=%t meAvatarRepo=%t inventoryUC=%t tokenBlueprintRepo=%t tokenIconResolver=%t selfBaseURL=%t)",
		c.Infra.Firestore != nil,
		c.Infra.GCS != nil,
		c.Infra.FirebaseAuth != nil,
		c.AvatarUC != nil,
		avatarWalletSvc != nil,
		c.CartUC != nil,
		c.CartRepo != nil,
		c.PaymentUC != nil,
		c.PaymentFlowUC != nil,
		c.InvoiceUC != nil,
		c.MeAvatarRepo != nil,
		c.InventoryUC != nil,
		c.TokenBlueprintRepo != nil,
		c.TokenIconURLResolver != nil,
		selfBaseURLConfigured,
	)

	return c, nil
}

// tokenIconURLResolver resolves icon URL from stored objectPath (or URL).
type tokenIconURLResolver struct {
	bucket string
}

func (r tokenIconURLResolver) ResolveForResponse(storedObjectPath string, storedIconURL string) string {
	if u := strings.TrimSpace(storedIconURL); u != "" {
		return u
	}
	p := strings.TrimSpace(storedObjectPath)
	if p == "" {
		return ""
	}

	// already a public URL
	if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
		return p
	}

	// gs://bucket/object -> public URL
	if b, obj, ok := gcscommon.ParseGCSURL(p); ok {
		return gcscommon.GCSPublicURL(b, obj, defaultTokenIconBucketDI)
	}

	// treat as objectPath within token icon bucket
	b := strings.TrimSpace(r.bucket)
	if b == "" {
		b = strings.TrimSpace(os.Getenv("TOKEN_ICON_BUCKET"))
	}
	if b == "" {
		b = defaultTokenIconBucketDI
	}

	p = strings.TrimLeft(p, "/")
	return gcscommon.GCSPublicURL(b, p, defaultTokenIconBucketDI)
}
