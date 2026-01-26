// backend/internal/platform/di/mall/container.go
package mall

import (
	"context"
	"errors"
	"log"
	"strings"

	"cloud.google.com/go/firestore"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"google.golang.org/api/iterator"

	// inbound (query + resolver types)
	mallquery "narratives/internal/application/query/mall"
	sharedquery "narratives/internal/application/query/shared"
	appresolver "narratives/internal/application/resolver"
	usecase "narratives/internal/application/usecase"

	// ✅ tokenBlueprint usecases (package was changed to tokenBlueprint)
	tokenbp "narratives/internal/application/tokenBlueprint"

	// inbound (for ImageURLResolver interface type)
	mallhandler "narratives/internal/adapters/in/http/mall/handler"

	// outbound
	outfs "narratives/internal/adapters/out/firestore"
	mallfs "narratives/internal/adapters/out/firestore/mall"
	gcso "narratives/internal/adapters/out/gcs"

	// Solana infra
	solanainfra "narratives/internal/infra/solana"
	solanaplatform "narratives/internal/infra/solana"

	// domains
	avatardom "narratives/internal/domain/avatar"
	branddom "narratives/internal/domain/brand"
	ldom "narratives/internal/domain/list"
	tokendom "narratives/internal/domain/token"

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

	// ✅ TokenBlueprint Bucket Usecase (tokenBlueprint package)
	TokenBlueprintBucketUC *tokenbp.TokenBlueprintBucketUsecase

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

	// ✅ reuse across WalletUsecase / NameResolver
	brandRepo := outfs.NewBrandRepositoryFS(fsClient)

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
	// ✅ TokenBlueprint Bucket Usecase (tokenBlueprint package)
	// --------------------------------------------------------
	c.TokenBlueprintBucketUC = tokenbp.NewTokenBlueprintBucketUsecase(gcsClient)

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

	// ========================================================
	// ✅ WalletUsecase
	// - NewWalletUsecase は 1 引数（WalletRepository）
	// - OnchainReader は WithOnchainReader で注入
	// - TokenQuery は WithTokenQuery で注入（mintAddress -> productId/docId, brandId, metadataUri）
	// - BrandNameResolver / productName 解決系も注入（ResolveTokenByMintAddressWithBrandName を拡張して productName まで返す前提）
	// ========================================================
	onchainReader := solanaplatform.NewOnchainWalletReaderDevnet()
	tokenQuery := newTokenQueryFS(fsClient)

	// ✅ brandId -> brandName
	brandSvc := branddom.NewService(brandRepo)

	// ✅ productId -> product(modelId)
	// previewProductReaderFS は di/mall/adapter.go に存在（Firestore直読み）
	prodReader := previewProductReaderFS{fs: fsClient}

	// ✅ modelId -> productBlueprintId
	// previewProductBlueprintReaderFS は di/mall/adapter.go に存在（Firestore直読み）
	pbReaderForModel := previewProductBlueprintReaderFS{
		fs: fsClient,
		pb: productBlueprintRepoFS,
	}
	modelPBResolver := modelPBIDResolverAdapter{r: pbReaderForModel}

	// ✅ productBlueprintId -> productName
	// productBlueprintRepoFS が読み取りを満たす前提で注入
	c.WalletUC = usecase.NewWalletUsecase(walletRepo).
		WithOnchainReader(onchainReader).
		WithTokenQuery(tokenQuery).
		WithBrandNameResolver(brandSvc).
		WithProductReader(prodReader).
		WithModelProductBlueprintIDResolver(modelPBResolver).
		WithProductBlueprintReader(productBlueprintRepoFS)

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
	//   -> wiring_policy.go に分離
	// --------------------------------------------------------
	{
		pf, configured, err := buildPaymentFlowUsecase(infra, c.PaymentUC)
		if err != nil {
			return nil, err
		}
		c.PaymentFlowUC = pf

		// for logging only
		_ = configured
	}

	// ✅ FIX: inventoryRepoForUC を渡す（ApplyTransferResult を満たす）
	c.InventoryUC = usecase.NewInventoryUsecase(inventoryRepoForUC)

	// --------------------------------------------------------
	// TokenIcon URL Resolver (objectPath -> publicURL)
	//   -> adapters/out/gcs/token_icon_url_resolver.go に移譲
	// --------------------------------------------------------
	{
		// NOTE:
		// We intentionally build resolver from gcs package to keep DI clean.
		// (type is *gcs.TokenIconURLResolver; it satisfies mallhandler.ImageURLResolver)
		c.TokenIconURLResolver = gcso.NewTokenIconURLResolver(strings.TrimSpace(infra.TokenIconBucket))
	}

	// --------------------------------------------------------
	// NameResolver (optional but useful for mall queries)
	// --------------------------------------------------------
	{
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

		// ✅ NEW: avatarId -> avatarName (GetNameByID)
		avatarName := avatarNameReaderAdapter{repo: avatarRepo}

		// ✅ FIX: NewOwnerResolveQuery now requires 4 args
		// - AvatarWalletAddressReader, BrandWalletAddressReader
		// - AvatarNameReader, BrandNameReader
		c.OwnerResolveQ = sharedquery.NewOwnerResolveQuery(
			avatarReader,
			brandReader,
			avatarName,
			brandSvc,
		)
	}

	// --------------------------------------------------------
	// ✅ TransferUsecase wiring (order scan verified -> brand->avatar transfer)
	//   -> wiring_policy.go に分離（条件分岐・secret provider 構築）
	//   NOTE: wiring_policy.go では TransferUsecase は作らない方針。
	// --------------------------------------------------------
	{
		// 0) ScanVerifier: OrderScanVerifyQuery -> usecase.ScanVerifier (policy)
		scanVerifier := buildScanVerifier(c.OrderScanVerifyQ)

		// 1) OrderRepoForTransfer
		var orderRepoForTransfer usecase.OrderRepoForTransfer = outfs.NewOrderRepoForTransferFS(fsClient)

		// 2) TokenResolver / TokenOwnerUpdater (moved to adapter.go)
		var tokenResolver usecase.TokenResolver = &tokenResolverFS{fs: fsClient, col: "tokens"}
		var tokenOwnerUpdater usecase.TokenOwnerUpdater = &tokenOwnerUpdaterFS{fs: fsClient, col: "tokens"}

		// 2.25) ✅ WalletItemUpdater: Firestore 実装(walletRepo)をそのまま利用
		// ※ walletRepo が AddMintToAvatarWalletItems を実装している前提
		var walletItemUpdater usecase.WalletItemUpdater = walletRepo

		// 2.5) TransferRepo
		var transferRepo usecase.TransferRepo = outfs.NewTransferRepositoryFS(fsClient)

		// 3) BrandWalletResolver / AvatarWalletResolver
		var walletResolver usecase.BrandWalletResolver = outfs.NewWalletResolverRepoFS(brandRepo, walletRepo)
		var avatarWalletResolver usecase.AvatarWalletResolver = walletResolver.(usecase.AvatarWalletResolver)

		// 4) WalletSecretProvider (Secret Manager) (policy)
		secrets, err := buildWalletSecretProvider(infra)
		if err != nil {
			return nil, err
		}

		// 5) TokenTransferExecutor (Solana)
		var executor usecase.TokenTransferExecutor = solanainfra.NewTokenTransferExecutorSolana("")

		// 6) Build TransferUC (container-go owns construction; policy only supplies conditional deps)
		// IMPORTANT:
		// - wiring_policy.go では TransferUsecase を作らない（採用済み）
		// - wiring_policy.go では WithInventoryRepo を呼ばない（採用済み）
		if scanVerifier != nil && secrets != nil {
			c.TransferUC = usecase.NewTransferUsecase(
				scanVerifier,
				orderRepoForTransfer,
				tokenResolver,
				tokenOwnerUpdater,
				walletItemUpdater, // ✅ NEW ARG
				transferRepo,
				walletResolver,
				avatarWalletResolver,
				secrets,
				executor,
			).WithInventoryRepo(inventoryRepoForUC)
		} else {
			c.TransferUC = nil
		}
	}

	// NOTE: wiring_policy.go で configured を返しているが、ここではログ用にだけ使う。
	selfBaseURLConfigured := strings.TrimSpace(infra.SelfBaseURL) != ""

	log.Printf(
		"[di.mall] container built (firestore=%t gcs=%t firebaseAuth=%t avatarUC=%t cartUC=%t cartRepo=%t paymentUC=%t paymentFlowUC=%t invoiceUC=%t meAvatarRepo=%t inventoryUC=%t tokenBlueprintRepo=%t tokenBlueprintBucketUC=%t tokenIconResolver=%t selfBaseURL=%t previewQ=%t ownerResolveQ=%t orderPurchasedQ=%t orderScanVerifyQ=%t transferUC=%t walletUC=%t)",
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
		c.TokenBlueprintBucketUC != nil,
		c.TokenIconURLResolver != nil,
		selfBaseURLConfigured,
		c.PreviewQ != nil,
		c.OwnerResolveQ != nil,
		c.OrderPurchasedQ != nil,
		c.OrderScanVerifyQ != nil,
		c.TransferUC != nil,
		c.WalletUC != nil,
	)

	return c, nil
}

// ============================================================
// ✅ Design B: Brand signer provider (Secret Manager)
// ============================================================

var (
	errSecretProviderNotConfigured = errors.New("di.mall: brandWalletSecretProviderSM not configured")
)

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
		return nil, errors.New("brandWalletSecretProviderSM: secretPrefix is empty")
	}
	ver := strings.TrimSpace(p.version)
	if ver == "" {
		ver = "latest"
	}

	secretID := prefix + bid
	name := "projects/" + prj + "/secrets/" + secretID + "/versions/" + ver
	resp, err := p.sm.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{Name: name})
	if err != nil {
		return nil, errors.New("brandWalletSecretProviderSM: AccessSecretVersion failed (" + name + "): " + err.Error())
	}
	if resp == nil || resp.Payload == nil {
		return nil, errors.New("brandWalletSecretProviderSM: empty payload (" + name + ")")
	}

	return strings.TrimSpace(string(resp.Payload.Data)), nil
}

// ============================================================
// ✅ NEW: TokenQueryFS (mintAddress -> productId(docId), brandId, metadataUri)
//   - outfs.NewTokenQueryFS(...) が無い前提で、DI側に最小実装を置く
// ============================================================

type tokenQueryFS struct {
	client *firestore.Client
	col    string
}

func newTokenQueryFS(client *firestore.Client) *tokenQueryFS {
	return &tokenQueryFS{
		client: client,
		col:    "tokens",
	}
}

func (q *tokenQueryFS) ResolveTokenByMintAddress(
	ctx context.Context,
	mintAddress string,
) (tokendom.ResolveTokenByMintAddressResult, error) {

	if q == nil || q.client == nil {
		return tokendom.ResolveTokenByMintAddressResult{}, errors.New("tokenQueryFS is not initialized")
	}

	m := strings.TrimSpace(mintAddress)
	if m == "" {
		return tokendom.ResolveTokenByMintAddressResult{}, errors.New("mintAddress is empty")
	}

	it := q.client.Collection(q.col).
		Where("mintAddress", "==", m).
		Limit(2).
		Documents(ctx)
	defer it.Stop()

	// 1件目
	doc, err := it.Next()
	if errors.Is(err, iterator.Done) {
		return tokendom.ResolveTokenByMintAddressResult{}, errors.New("token not found for mintAddress")
	}
	if err != nil {
		return tokendom.ResolveTokenByMintAddressResult{}, err
	}

	// 2件目があればユニーク違反
	doc2, err := it.Next()
	if err == nil && doc2 != nil {
		return tokendom.ResolveTokenByMintAddressResult{}, errors.New("multiple tokens found for mintAddress")
	}
	if err != nil && !errors.Is(err, iterator.Done) {
		return tokendom.ResolveTokenByMintAddressResult{}, err
	}

	raw := doc.Data()

	brandID, _ := raw["brandId"].(string)
	metadataURI, _ := raw["metadataUri"].(string)

	brandID = strings.TrimSpace(brandID)
	metadataURI = strings.TrimSpace(metadataURI)

	// Firestore では docID が productId（あなたの設計前提）
	productID := strings.TrimSpace(doc.Ref.ID)
	if productID == "" {
		return tokendom.ResolveTokenByMintAddressResult{}, errors.New("resolved productId is empty")
	}

	return tokendom.ResolveTokenByMintAddressResult{
		ProductID:   productID,
		MintAddress: m,
		BrandID:     brandID,
		MetadataURI: metadataURI,
	}, nil
}

// ============================================================
// ✅ Adapter: modelId -> productBlueprintId (usecase port)
// ============================================================

type modelPBIDResolverAdapter struct {
	r interface {
		GetIDByModelID(ctx context.Context, modelID string) (string, error)
	}
}

func (a modelPBIDResolverAdapter) GetProductBlueprintIDByModelID(ctx context.Context, modelID string) (string, error) {
	if a.r == nil {
		return "", errors.New("modelPBIDResolverAdapter: resolver is nil")
	}
	return a.r.GetIDByModelID(ctx, strings.TrimSpace(modelID))
}

// ============================================================
// ✅ Adapter: avatarId -> avatarName (sharedquery.AvatarNameReader)
// ============================================================

type avatarNameReaderAdapter struct {
	repo interface {
		GetByID(ctx context.Context, id string) (avatardom.Avatar, error)
	}
}

func (a avatarNameReaderAdapter) GetNameByID(ctx context.Context, avatarID string) (string, error) {
	id := strings.TrimSpace(avatarID)
	if id == "" {
		return "", errors.New("avatarNameReaderAdapter: avatarID is empty")
	}

	av, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(av.AvatarName), nil
}
