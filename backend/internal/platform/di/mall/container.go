// backend/internal/platform/di/mall/container.go
package mall

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"

	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	// inbound (query + resolver types)
	mallquery "narratives/internal/application/query/mall"
	sharedquery "narratives/internal/application/query/shared"
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

	// Solana infra
	solanainfra "narratives/internal/infra/solana"
	solanaplatform "narratives/internal/infra/solana"

	// domains
	ldom "narratives/internal/domain/list"
	productdom "narratives/internal/domain/product"
	pbdom "narratives/internal/domain/productBlueprint"

	shared "narratives/internal/platform/di/shared"
)

const (
	StripeWebhookPath        = "/mall/webhooks/stripe"
	defaultTokenIconBucketDI = "narratives-development_token_icon" // tokenIcon_repository_gcs.go と同じ既定値

	// owner-resolve query (walletAddress -> brandId / avatarId)
	defaultBrandsCollection  = "brands"
	defaultAvatarsCollection = "avatars"

	// ✅ Design B (brand signer): SecretManager secret name prefix
	// secretId = brand-wallet-<brandId>
	defaultBrandWalletSecretPrefix = "brand-wallet-"
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

	// ✅ order scan transfer usecase
	TransferUC *usecase.TransferUsecase

	// ✅ Case A: payment起票後に（必要なら）webhook trigger
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

	// ✅ any をやめて具体型で持つ
	OrderQ *mallquery.OrderQuery

	// ✅ purchased orders
	OrderPurchasedQ *mallquery.OrderPurchasedQuery

	// ✅ verify scanned pair
	OrderScanVerifyQ *mallquery.OrderScanVerifyQuery

	// ✅ Shared query: walletAddress(toAddress) -> brandId / avatarId
	OwnerResolveQ *sharedquery.OwnerResolveQuery

	// Repos sometimes needed by handlers/queries/joins
	ListRepo ldom.Repository

	// ✅ /mall/me/avatar 用: uid -> avatarId を解決するRepo
	MeAvatarRepo *mallfs.MeAvatarRepo

	// ✅ handler が repo を直接必要とするケースに備える
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

	// ✅ IMPORTANT: Config は必須（projectID 解決に必要）
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

	// Inventory (Firestore)
	inventoryRepo := outfs.NewInventoryRepositoryFS(fsClient)
	// ✅ inventory.RepositoryPort を満たすための adapter（ApplyTransferResult を追加）
	inventoryRepoForUC := &inventoryRepoTransferResultAdapter{InventoryRepositoryFS: inventoryRepo}

	// TokenBlueprint (tokenBlueprint domain)
	tokenBlueprintRepo := outfs.NewTokenBlueprintRepositoryFS(fsClient)
	c.TokenBlueprintRepo = tokenBlueprintRepo

	// ProductBlueprint (productBlueprint domain) - shared for queries/resolvers
	productBlueprintRepoFS := outfs.NewProductBlueprintRepositoryFS(fsClient)

	// Shared instance for queries/resolvers (avoid duplication)
	modelRepoFS := outfs.NewModelRepositoryFS(fsClient)

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
	//   projectID 解決は shared.Infra に委譲済み
	// --------------------------------------------------------
	projectID := strings.TrimSpace(infra.ProjectID)
	avatarWalletSvc := solanainfra.NewAvatarWalletService(projectID)

	// --------------------------------------------------------
	// Usecases
	// --------------------------------------------------------

	// ✅ AvatarUsecase: cartRepo / walletRepo / walletSvc を必ず注入
	c.AvatarUC = usecase.NewAvatarUsecase(
		avatarRepo,
		avatarStateRepo,
		avatarIconRepo,
		avatarIconRepo,
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

	// ✅ WalletUsecase + OnchainReader (devnet default)
	onchainReader := solanaplatform.NewOnchainWalletReaderDevnet()
	c.WalletUC = usecase.NewWalletUsecase(walletRepo).WithOnchainReader(onchainReader)

	c.CartUC = usecase.NewCartUsecase(cartRepo)

	// ✅ payment 起票後に invoice.paid=true を立てるため invoiceRepo を注入
	// ✅ paid と同タイミングで cart clear / inventory reserve を行う（best-effort）
	c.PaymentUC = usecase.NewPaymentUsecase(paymentRepo).
		WithInvoiceRepoForPayment(invoiceRepo).
		WithCartRepoForPayment(cartRepo).
		WithOrderRepoForPayment(orderRepo).
		WithInventoryRepoForPayment(inventoryRepoForUC)

	c.InvoiceUC = usecase.NewInvoiceUsecase(invoiceRepo)
	c.OrderUC = usecase.NewOrderUsecase(orderRepo)

	// --------------------------------------------------------
	// ✅ Case A: PaymentFlowUsecase（payment起票 + 必要なら webhook trigger）
	// --------------------------------------------------------
	selfBaseURL := strings.TrimSpace(infra.SelfBaseURL)
	selfBaseURLConfigured := selfBaseURL != ""

	if selfBaseURLConfigured {
		stripeTrigger := httpout.NewStripeWebhookClient(selfBaseURL)
		c.PaymentFlowUC = usecase.NewPaymentFlowUsecase(c.PaymentUC, stripeTrigger)
	} else {
		c.PaymentFlowUC = usecase.NewPaymentFlowUsecase(c.PaymentUC, nil)
	}

	// ✅ FIX: inventoryRepoForUC を渡す（ApplyTransferResult を満たす）
	c.InventoryUC = usecase.NewInventoryUsecase(inventoryRepoForUC)

	// --------------------------------------------------------
	// TokenIcon URL Resolver (objectPath -> publicURL)
	// --------------------------------------------------------
	{
		c.TokenIconURLResolver = tokenIconURLResolver{bucket: strings.TrimSpace(infra.TokenIconBucket)}
	}

	// --------------------------------------------------------
	// NameResolver (optional but useful for mall queries)
	// --------------------------------------------------------
	{
		brandRepo := outfs.NewBrandRepositoryFS(fsClient)
		companyRepo := outfs.NewCompanyRepositoryFS(fsClient)
		memberRepo := outfs.NewMemberRepositoryFS(fsClient)

		// tokenBlueprintNameRepoAdapter は di/mall/adapter.go に存在
		tbNameRepo := &tokenBlueprintNameRepoAdapter{repo: tokenBlueprintRepo}

		c.NameResolver = appresolver.NewNameResolver(
			brandRepo,
			companyRepo,
			productBlueprintRepoFS,
			memberRepo,
			modelRepoFS,
			tbNameRepo,
		)
	}

	// --------------------------------------------------------
	// Queries (mall-facing)
	// --------------------------------------------------------
	{
		invRepo := mallfs.NewInventoryRepoForMallQuery(fsClient)

		c.CatalogQ = mallquery.NewCatalogQuery(listRepoFS, invRepo, productBlueprintRepoFS, modelRepoFS)
		c.CartQ = mallquery.NewCartQuery(fsClient)

		// ✅ PreviewQuery 用 ProductBlueprintReader
		pbReader := previewProductBlueprintReaderFS{
			fs: fsClient,
			pb: productBlueprintRepoFS,
		}

		c.PreviewQ = mallquery.NewPreviewQuery(
			previewProductReaderFS{fs: fsClient},
			modelRepoFS,
			pbReader,
		)

		if c.PreviewQ != nil {
			c.PreviewQ.TokenRepo = outfs.NewTokenReaderFS(fsClient)
		}

		c.OrderQ = mallquery.NewOrderQuery(fsClient)
		c.OrderPurchasedQ = mallquery.NewOrderPurchasedQuery(fsClient)
		c.OrderScanVerifyQ = mallquery.NewOrderScanVerifyQuery(c.OrderPurchasedQ, c.PreviewQ)

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
	}

	// --------------------------------------------------------
	// ✅ Shared Query: OwnerResolve
	// --------------------------------------------------------
	{
		brandsCol := strings.TrimSpace(infra.BrandsCollection)
		avatarsCol := strings.TrimSpace(infra.AvatarsCollection)

		brandReader := brandWalletAddressReaderFS{fs: fsClient, col: brandsCol}
		avatarReader := avatarWalletAddressReaderFS{fs: fsClient, col: avatarsCol}

		c.OwnerResolveQ = sharedquery.NewOwnerResolveQuery(avatarReader, brandReader)
	}

	// --------------------------------------------------------
	// ✅ TransferUsecase wiring (order scan verified -> brand->avatar transfer)
	// --------------------------------------------------------
	{
		// 0) ScanVerifier: OrderScanVerifyQuery -> usecase.ScanVerifier
		var scanVerifier usecase.ScanVerifier
		if c.OrderScanVerifyQ != nil {
			scanVerifier = mallquery.NewScanVerifierAdapter(c.OrderScanVerifyQ)
		} else {
			scanVerifier = nil
		}

		// 1) OrderRepoForTransfer
		var orderRepoForTransfer usecase.OrderRepoForTransfer = outfs.NewOrderRepoForTransferFS(fsClient)

		// 2) TokenResolver / TokenOwnerUpdater
		var tokenResolver usecase.TokenResolver = &tokenResolverFS{fs: fsClient, col: "tokens"}
		var tokenOwnerUpdater usecase.TokenOwnerUpdater = &tokenOwnerUpdaterFS{fs: fsClient, col: "tokens"}

		// 2.5) TransferRepo
		var transferRepo usecase.TransferRepo = outfs.NewTransferRepositoryFS(fsClient)

		// 3) BrandWalletResolver / AvatarWalletResolver
		brandRepo := outfs.NewBrandRepositoryFS(fsClient)
		var walletResolver usecase.BrandWalletResolver = outfs.NewWalletResolverRepoFS(brandRepo, walletRepo)
		var avatarWalletResolver usecase.AvatarWalletResolver = walletResolver.(usecase.AvatarWalletResolver)

		// 4) WalletSecretProvider (Secret Manager)
		var secrets usecase.WalletSecretProvider = nil
		if infra.SecretManager != nil && strings.TrimSpace(infra.ProjectID) != "" {
			secretPrefix := strings.TrimSpace(infra.BrandWalletSecretPrefix)
			if secretPrefix == "" {
				secretPrefix = defaultBrandWalletSecretPrefix
			}
			secrets = &brandWalletSecretProviderSM{
				sm:           infra.SecretManager,
				projectID:    strings.TrimSpace(infra.ProjectID),
				secretPrefix: secretPrefix,
				version:      "latest",
			}
		}

		// 5) TokenTransferExecutor (Solana)
		var executor usecase.TokenTransferExecutor = solanainfra.NewTokenTransferExecutorSolana("")

		// 6) Build TransferUC only when truly conditional deps exist
		if scanVerifier != nil && secrets != nil {
			c.TransferUC = usecase.NewTransferUsecase(
				scanVerifier,
				orderRepoForTransfer,
				tokenResolver,
				tokenOwnerUpdater,
				transferRepo,
				walletResolver,
				avatarWalletResolver,
				secrets,
				executor,
			).
				// ✅ 統合方針: transfer 成功後の inventory cleanup も TransferUC 内で実行する
				WithInventoryRepo(inventoryRepoForUC)
		} else {
			c.TransferUC = nil
		}
	}

	log.Printf(
		"[di.mall] container built (firestore=%t gcs=%t firebaseAuth=%t avatarUC=%t cartUC=%t cartRepo=%t paymentUC=%t paymentFlowUC=%t invoiceUC=%t meAvatarRepo=%t inventoryUC=%t tokenBlueprintRepo=%t tokenIconResolver=%t selfBaseURL=%t previewQ=%t ownerResolveQ=%t orderPurchasedQ=%t orderScanVerifyQ=%t transferUC=%t)",
		c.Infra.Firestore != nil,
		c.Infra.GCS != nil,
		c.Infra.FirebaseAuth != nil,
		c.AvatarUC != nil,
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
		c.PreviewQ != nil,
		c.OwnerResolveQ != nil,
		c.OrderPurchasedQ != nil,
		c.OrderScanVerifyQ != nil,
		c.TransferUC != nil,
	)

	return c, nil
}

// ------------------------------------------------------------
// PreviewQuery: ProductReader adapter (Firestore -> domain.Product)
// ------------------------------------------------------------

type previewProductReaderFS struct {
	fs *firestore.Client
}

func (r previewProductReaderFS) GetByID(ctx context.Context, productID string) (productdom.Product, error) {
	if r.fs == nil {
		return productdom.Product{}, mallquery.ErrPreviewQueryNotConfigured
	}
	id := strings.TrimSpace(productID)
	if id == "" {
		return productdom.Product{}, mallquery.ErrInvalidProductID
	}

	doc, err := r.fs.Collection("products").Doc(id).Get(ctx)
	if err != nil {
		return productdom.Product{}, err
	}

	var p productdom.Product
	if err := doc.DataTo(&p); err != nil {
		return productdom.Product{}, err
	}

	p.ID = doc.Ref.ID
	return p, nil
}

// ------------------------------------------------------------
// PreviewQuery: ProductBlueprintReader adapter
// ------------------------------------------------------------

type previewProductBlueprintReaderFS struct {
	fs *firestore.Client
	pb interface {
		GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error)
		GetPatchByID(ctx context.Context, id string) (pbdom.Patch, error)
	}
}

func (r previewProductBlueprintReaderFS) GetIDByModelID(ctx context.Context, modelID string) (string, error) {
	if r.fs == nil {
		return "", mallquery.ErrPreviewQueryNotConfigured
	}
	id := strings.TrimSpace(modelID)
	if id == "" {
		return "", mallquery.ErrInvalidModelID
	}

	snap, err := r.fs.Collection("models").Doc(id).Get(ctx)
	if err != nil {
		return "", err
	}

	data := snap.Data()
	if data == nil {
		return "", nil
	}

	for _, k := range []string{"productBlueprintId", "productBlueprintID", "product_blueprint_id"} {
		if v, ok := data[k]; ok {
			if s, ok := v.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					return s, nil
				}
			}
		}
	}

	return "", nil
}

func (r previewProductBlueprintReaderFS) GetPatchByID(ctx context.Context, id string) (pbdom.Patch, error) {
	if r.pb == nil {
		return pbdom.Patch{}, mallquery.ErrPreviewQueryNotConfigured
	}
	return r.pb.GetPatchByID(ctx, id)
}

func (r previewProductBlueprintReaderFS) GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error) {
	if r.pb == nil {
		return pbdom.ProductBlueprint{}, mallquery.ErrPreviewQueryNotConfigured
	}
	return r.pb.GetByID(ctx, id)
}

// ------------------------------------------------------------
// ✅ SharedQuery OwnerResolve: walletAddress readers (Firestore)
// ------------------------------------------------------------

type brandWalletAddressReaderFS struct {
	fs  *firestore.Client
	col string
}

func (r brandWalletAddressReaderFS) FindBrandIDByWalletAddress(ctx context.Context, walletAddress string) (string, error) {
	if r.fs == nil {
		return "", sharedquery.ErrOwnerResolveNotConfigured
	}
	addr := strings.TrimSpace(walletAddress)
	if addr == "" {
		return "", sharedquery.ErrInvalidWalletAddress
	}

	col := strings.TrimSpace(r.col)
	if col == "" {
		col = defaultBrandsCollection
	}

	it := r.fs.Collection(col).
		Where("walletAddress", "==", addr).
		Limit(1).
		Documents(ctx)

	doc, err := it.Next()
	if err != nil {
		if err == iterator.Done {
			return "", nil
		}
		return "", err
	}
	if doc == nil || doc.Ref == nil {
		return "", nil
	}
	return strings.TrimSpace(doc.Ref.ID), nil
}

type avatarWalletAddressReaderFS struct {
	fs  *firestore.Client
	col string
}

func (r avatarWalletAddressReaderFS) FindAvatarIDByWalletAddress(ctx context.Context, walletAddress string) (string, error) {
	if r.fs == nil {
		return "", sharedquery.ErrOwnerResolveNotConfigured
	}
	addr := strings.TrimSpace(walletAddress)
	if addr == "" {
		return "", sharedquery.ErrInvalidWalletAddress
	}

	col := strings.TrimSpace(r.col)
	if col == "" {
		col = defaultAvatarsCollection
	}

	it := r.fs.Collection(col).
		Where("walletAddress", "==", addr).
		Limit(1).
		Documents(ctx)

	doc, err := it.Next()
	if err != nil {
		if err == iterator.Done {
			return "", nil
		}
		return "", err
	}
	if doc == nil || doc.Ref == nil {
		return "", nil
	}
	return strings.TrimSpace(doc.Ref.ID), nil
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

	if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
		return p
	}

	if b, obj, ok := gcscommon.ParseGCSURL(p); ok {
		return gcscommon.GCSPublicURL(b, obj, defaultTokenIconBucketDI)
	}

	b := strings.TrimSpace(r.bucket)
	if b == "" {
		b = defaultTokenIconBucketDI
	}

	p = strings.TrimLeft(p, "/")
	return gcscommon.GCSPublicURL(b, p, defaultTokenIconBucketDI)
}

// ============================================================
// Transfer: minimal Firestore ports
// ============================================================

var (
	errTokenResolverNotConfigured  = errors.New("di.mall: tokenResolverFS not configured")
	errTokenDocNotFound            = errors.New("di.mall: token doc not found")
	errSecretProviderNotConfigured = errors.New("di.mall: brandWalletSecretProviderSM not configured")
)

type tokenResolverFS struct {
	fs  *firestore.Client
	col string
}

func (r *tokenResolverFS) ResolveTokenByProductID(ctx context.Context, productID string) (usecase.TokenForTransfer, error) {
	if r == nil || r.fs == nil {
		return usecase.TokenForTransfer{}, errTokenResolverNotConfigured
	}
	pid := strings.TrimSpace(productID)
	if pid == "" {
		return usecase.TokenForTransfer{}, errors.New("tokenResolverFS: productId is empty")
	}
	col := strings.TrimSpace(r.col)
	if col == "" {
		col = "tokens"
	}

	snap, err := r.fs.Collection(col).Doc(pid).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return usecase.TokenForTransfer{}, errTokenDocNotFound
		}
		return usecase.TokenForTransfer{}, err
	}
	raw := snap.Data()
	if raw == nil {
		return usecase.TokenForTransfer{}, errTokenDocNotFound
	}

	getStr := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := raw[k]; ok {
				if s, ok := v.(string); ok {
					s = strings.TrimSpace(s)
					if s != "" {
						return s
					}
				}
			}
		}
		return ""
	}

	return usecase.TokenForTransfer{
		ProductID: pid,
		BrandID:   getStr("brandId", "brandID"),
		MintAddress: getStr(
			"mintAddress",
			"mint_address",
		),
		TokenBlueprintID: getStr(
			"tokenBlueprintId",
			"tokenBlueprintID",
			"token_blueprint_id",
		),
		ToAddress: getStr("toAddress", "to_address"),
	}, nil
}

type tokenOwnerUpdaterFS struct {
	fs  *firestore.Client
	col string
}

func (r *tokenOwnerUpdaterFS) UpdateToAddressByProductID(ctx context.Context, productID string, newToAddress string, now time.Time, txSignature string) error {
	if r == nil || r.fs == nil {
		return errTokenResolverNotConfigured
	}
	pid := strings.TrimSpace(productID)
	if pid == "" {
		return errors.New("tokenOwnerUpdaterFS: productId is empty")
	}
	addr := strings.TrimSpace(newToAddress)
	if addr == "" {
		return errors.New("tokenOwnerUpdaterFS: newToAddress is empty")
	}
	col := strings.TrimSpace(r.col)
	if col == "" {
		col = "tokens"
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()

	ref := r.fs.Collection(col).Doc(pid)

	_, err := ref.Set(ctx, map[string]any{
		"toAddress":       addr,
		"updatedAt":       now,
		"lastTxSignature": strings.TrimSpace(txSignature),
		"ownerUpdatedAt":  now,
	}, firestore.MergeAll)
	return err
}

// ============================================================
// ✅ Design B: Brand signer provider (Secret Manager)
// ============================================================

type brandWalletSecretProviderSM struct {
	sm           *secretmanager.Client
	projectID    string
	secretPrefix string
	version      string
}

func (p *brandWalletSecretProviderSM) GetBrandSigner(ctx context.Context, brandID string) (any, error) {
	if p == nil || p.sm == nil {
		return nil, errSecretProviderNotConfigured
	}
	bid := strings.TrimSpace(brandID)
	if bid == "" {
		return nil, errors.New("brandWalletSecretProviderSM: brandID is empty")
	}
	prj := strings.TrimSpace(p.projectID)
	if prj == "" {
		return nil, errors.New("brandWalletSecretProviderSM: projectID is empty")
	}

	prefix := strings.TrimSpace(p.secretPrefix)
	if prefix == "" {
		prefix = defaultBrandWalletSecretPrefix
	}
	ver := strings.TrimSpace(p.version)
	if ver == "" {
		ver = "latest"
	}

	secretID := prefix + bid
	name := fmt.Sprintf("projects/%s/secrets/%s/versions/%s", prj, secretID, ver)
	resp, err := p.sm.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{Name: name})
	if err != nil {
		return nil, fmt.Errorf("brandWalletSecretProviderSM: AccessSecretVersion failed (%s): %w", name, err)
	}
	if resp == nil || resp.Payload == nil {
		return nil, fmt.Errorf("brandWalletSecretProviderSM: empty payload (%s)", name)
	}

	return strings.TrimSpace(string(resp.Payload.Data)), nil
}

// ============================================================
// ✅ Adapter: outfs.InventoryRepositoryFS に ApplyTransferResult を付与
// ============================================================

type inventoryRepoTransferResultAdapter struct {
	*outfs.InventoryRepositoryFS
}

func (a *inventoryRepoTransferResultAdapter) ApplyTransferResult(
	ctx context.Context,
	productID string,
	orderID string,
	now time.Time,
) error {
	if a == nil || a.InventoryRepositoryFS == nil {
		return errors.New("inventory repo adapter is nil")
	}

	removed, err := a.InventoryRepositoryFS.ReleaseReservationAfterTransfer(ctx, productID, orderID, now)
	if err != nil {
		return err
	}

	log.Printf(
		"[inventory_repo_adapter.mall] ApplyTransferResult ok productId=%q orderId=%q removed=%d at=%s",
		strings.TrimSpace(productID),
		strings.TrimSpace(orderID),
		removed,
		now.UTC().Format(time.RFC3339),
	)

	return nil
}
