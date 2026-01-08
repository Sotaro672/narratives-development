// backend/internal/platform/di/mall/container.go
package mall

import (
	"context"
	"errors"
	"log"

	// inbound (query + resolver types)
	mallquery "narratives/internal/application/query/mall"
	appresolver "narratives/internal/application/resolver"
	usecase "narratives/internal/application/usecase"

	// outbound
	outfs "narratives/internal/adapters/out/firestore"
	mallfs "narratives/internal/adapters/out/firestore/mall"
	gcso "narratives/internal/adapters/out/gcs"

	// domains
	ldom "narratives/internal/domain/list"

	shared "narratives/internal/platform/di/shared"
)

const (
	StripeWebhookPath = "/mall/webhooks/stripe"
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

	// Optional resolver (for query enrich)
	NameResolver *appresolver.NameResolver

	// Queries (mall-facing)
	CatalogQ *mallquery.CatalogQuery
	CartQ    *mallquery.CartQuery
	PreviewQ *mallquery.PreviewQuery
	OrderQ   any

	// Repos sometimes needed by queries/joins
	ListRepo ldom.Repository
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
	// -> backend/internal/adapters/out/firestore/mall/list_patcher_repo.go
	listPatcher := mallfs.NewListPatcherRepo(fsClient)

	// --------------------------------------------------------
	// Usecases
	// --------------------------------------------------------
	c.ListUC = usecase.NewListUsecase(
		listRepoForUC, // ListReader
		listPatcher,   // ListPatcher
		listImageRepo, // ListImageReader
		listImageRepo, // ListImageByIDReader
		listImageRepo, // ListImageObjectSaver (+ SignedURLIssuer)
	)

	c.ShippingAddressUC = usecase.NewShippingAddressUsecase(shippingAddressRepo)
	c.BillingAddressUC = usecase.NewBillingAddressUsecase(billingAddressRepo)
	c.UserUC = usecase.NewUserUsecase(userRepo)
	c.WalletUC = usecase.NewWalletUsecase(walletRepo)

	c.CartUC = usecase.NewCartUsecase(cartRepo)
	c.PaymentUC = usecase.NewPaymentUsecase(paymentRepo)
	c.InvoiceUC = usecase.NewInvoiceUsecase(invoiceRepo)
	c.OrderUC = usecase.NewOrderUsecase(orderRepo).WithInvoiceUsecase(c.InvoiceUC)

	// --------------------------------------------------------
	// NameResolver (optional but useful for mall queries)
	// --------------------------------------------------------
	{
		brandRepo := outfs.NewBrandRepositoryFS(fsClient)
		companyRepo := outfs.NewCompanyRepositoryFS(fsClient)
		productBlueprintRepo := outfs.NewProductBlueprintRepositoryFS(fsClient)
		memberRepo := outfs.NewMemberRepositoryFS(fsClient)
		modelRepo := outfs.NewModelRepositoryFS(fsClient)
		tokenBlueprintRepo := outfs.NewTokenBlueprintRepositoryFS(fsClient)

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
		// ✅ CatalogQuery の InventoryRepository は outfs.InventoryRepositoryFS では満たせない
		// → out/firestore/mall の adapter (InventoryRepoForMallQuery) を使う
		invRepo := mallfs.NewInventoryRepoForMallQuery(fsClient)

		pbRepo := outfs.NewProductBlueprintRepositoryFS(fsClient)
		modelRepo := outfs.NewModelRepositoryFS(fsClient)

		c.CatalogQ = mallquery.NewCatalogQuery(listRepoFS, invRepo, pbRepo, modelRepo)

		c.CartQ = mallquery.NewCartQuery(fsClient)
		c.PreviewQ = mallquery.NewPreviewQuery(fsClient)
		c.OrderQ = mallquery.NewOrderQuery(fsClient)

		// light injection (no reflection)
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

	log.Printf("[di.mall] container built (firestore=%t gcs=%t firebaseAuth=%t)",
		c.Infra.Firestore != nil,
		c.Infra.GCS != nil,
		c.Infra.FirebaseAuth != nil,
	)

	return c, nil
}
