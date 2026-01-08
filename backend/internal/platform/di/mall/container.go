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
	ListUC            *usecase.ListUsecase
	ShippingAddressUC *usecase.ShippingAddressUsecase
	BillingAddressUC  *usecase.BillingAddressUsecase
	UserUC            *usecase.UserUsecase
	WalletUC          *usecase.WalletUsecase
	CartUC            *usecase.CartUsecase
	PaymentUC         *usecase.PaymentUsecase
	OrderUC           *usecase.OrderUsecase
	InvoiceUC         *usecase.InvoiceUsecase

	// ✅ Inventory (buyer-facing, read-only)
	InventoryUC *usecase.InventoryUsecase

	// ✅ TokenBlueprint (buyer-facing patch handler 用に repo を保持)
	TokenBlueprintRepo any

	// ✅ Token Icon public URL resolver（objectPath -> public URL）
	// 命名は TokenIcon に寄せる
	TokenIconURLResolver mallhandler.ImageURLResolver

	// ✅ 互換: 旧命名を参照している箇所があっても落ちないように残す
	TokenImageURLResolver mallhandler.ImageURLResolver

	// Optional resolver (for query enrich)
	NameResolver *appresolver.NameResolver

	// Queries (mall-facing)
	CatalogQ *mallquery.CatalogQuery
	CartQ    *mallquery.CartQuery
	PreviewQ *mallquery.PreviewQuery
	OrderQ   any

	// Repos sometimes needed by queries/joins
	ListRepo ldom.Repository

	// ✅ /mall/me/avatar 用: uid -> avatarId を解決するRepo
	MeAvatarRepo *mallfs.MeAvatarRepo
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
	shippingAddressRepo := outfs.NewShippingAddressRepositoryFS(fsClient)
	billingAddressRepo := outfs.NewBillingAddressRepositoryFS(fsClient)
	userRepo := outfs.NewUserRepositoryFS(fsClient)
	walletRepo := outfs.NewWalletRepositoryFS(fsClient)

	cartRepo := outfs.NewCartRepositoryFS(fsClient)
	paymentRepo := outfs.NewPaymentRepositoryFS(fsClient)
	orderRepo := outfs.NewOrderRepositoryFS(fsClient)
	invoiceRepo := outfs.NewInvoiceRepositoryFS(fsClient)

	// ✅ Inventory (usecase)
	inventoryRepo := outfs.NewInventoryRepositoryFS(fsClient)

	// ✅ TokenBlueprint repo（handler に渡す & NameResolver にも使う）
	tokenBlueprintRepo := outfs.NewTokenBlueprintRepositoryFS(fsClient)
	c.TokenBlueprintRepo = tokenBlueprintRepo

	// ✅ /mall/me/avatar 用 Repo（uid -> avatarId）
	c.MeAvatarRepo = mallfs.NewMeAvatarRepo(fsClient)

	// List repo
	listRepoFS := outfs.NewListRepositoryFS(fsClient)
	c.ListRepo = listRepoFS

	// List repo for usecase ports (console側と同じ変換を使う)
	listRepoForUC := outfs.NewListRepositoryForUsecase(listRepoFS)

	// --------------------------------------------------------
	// GCS repositories (ListUsecase needs image ports)
	// --------------------------------------------------------
	listImageRepo := gcso.NewListImageRepositoryGCS(gcsClient, infra.ListImageBucket)

	// ✅ ListPatcher は di/mall/adapter.go から分離済み
	listPatcher := mallfs.NewListPatcherRepo(fsClient)

	// --------------------------------------------------------
	// Usecases
	// --------------------------------------------------------
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
	c.PaymentUC = usecase.NewPaymentUsecase(paymentRepo)
	c.InvoiceUC = usecase.NewInvoiceUsecase(invoiceRepo)
	c.OrderUC = usecase.NewOrderUsecase(orderRepo).WithInvoiceUsecase(c.InvoiceUC)

	// ✅ InventoryUsecase（read-only handler が GetByID を呼ぶ想定）
	c.InventoryUC = usecase.NewInventoryUsecase(inventoryRepo)

	// --------------------------------------------------------
	// TokenIcon URL Resolver (objectPath -> publicURL)
	// --------------------------------------------------------
	{
		b := strings.TrimSpace(os.Getenv("TOKEN_ICON_BUCKET"))
		// 空なら tokenIcon_repository_gcs.go のデフォルトに寄せる
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
		// CatalogQuery の InventoryRepository は out/firestore/mall の adapter を使う
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

	log.Printf("[di.mall] container built (firestore=%t gcs=%t firebaseAuth=%t meAvatarRepo=%t inventoryUC=%t tokenBlueprintRepo=%t tokenIconResolver=%t)",
		c.Infra.Firestore != nil,
		c.Infra.GCS != nil,
		c.Infra.FirebaseAuth != nil,
		c.MeAvatarRepo != nil,
		c.InventoryUC != nil,
		c.TokenBlueprintRepo != nil,
		c.TokenIconURLResolver != nil,
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
