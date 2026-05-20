// backend/internal/platform/di/mall/container.go
package mall

import (
	"context"
	"errors"

	// inbound (query + resolver types)
	mallquery "narratives/internal/application/query/mall"
	catalogQuery "narratives/internal/application/query/mall/catalog"
	sharedquery "narratives/internal/application/query/shared"
	appresolver "narratives/internal/application/resolver"

	// base usecases
	usecase "narratives/internal/application/usecase"

	// moved: AvatarUsecase is now in subpackage usecase/avatar
	avataruc "narratives/internal/application/usecase/avatar"

	// moved: ListUsecase is now in subpackage usecase/list
	listuc "narratives/internal/application/usecase/list"

	// inbound
	mallhandler "narratives/internal/adapters/in/http/mall/handler"

	// outbound
	outfs "narratives/internal/adapters/out/firestore"
	mallfs "narratives/internal/adapters/out/firestore/mall"
	pbfs "narratives/internal/adapters/out/firestore/productBlueprint"
	outsolana "narratives/internal/adapters/out/solana"
	stripeadapter "narratives/internal/adapters/out/stripe"

	// Solana infra
	solanainfra "narratives/internal/infra/solana"
	solanaplatform "narratives/internal/infra/solana"

	// domains
	avatardom "narratives/internal/domain/avatar"
	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
	productbpdom "narratives/internal/domain/productBlueprint"
	tokenBlueprint_review "narratives/internal/domain/tokenBlueprint_review"

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
	AvatarUC          *avataruc.AvatarUsecase
	ListUC            *listuc.ListUsecase
	ShippingAddressUC *usecase.ShippingAddressUsecase
	PaymentMethodUC   *usecase.PaymentMethodUsecase
	UserUC            *usecase.UserUsecase
	WalletUC          *usecase.WalletUsecase
	CartUC            *usecase.CartUsecase
	PaymentUC         *usecase.PaymentUsecase
	OrderUC           *usecase.OrderUsecase

	AvatarRepo   avatardom.Repository
	BrandService *branddom.Service

	// ProductBlueprintReview Usecase（/mall/catalog + /mall/me/catalog）
	ProductBlueprintReviewUC *usecase.ProductBlueprintReviewUsecase

	// order scan transfer usecase
	TransferUC *usecase.TransferUsecase

	// share transfer usecase
	ShareTransferUC *usecase.ShareTransferUsecase

	// Case A: payment起票後に（必要なら）webhook trigger
	PaymentFlowUC *usecase.PaymentFlowUsecase

	// Inventory (buyer-facing, read-only)
	InventoryUC *usecase.InventoryUsecase

	// TokenBlueprint Review (YouTube-like comments)
	TokenBlueprintReviewRepo tokenBlueprint_review.RepositoryPort

	// resolvedTokens cache repo (wallets/{avatarId}/resolvedTokens/{mint})
	ResolvedTokenRepo mallhandler.ResolvedTokenRepository

	// Optional resolver (for query enrich)
	NameResolver *appresolver.NameResolver

	// Queries (mall-facing)
	BrandQ   *mallquery.BrandQuery
	CatalogQ *catalogQuery.CatalogQuery
	CartQ    *mallquery.CartQuery
	PreviewQ *mallquery.PreviewQuery

	OrderQ *mallquery.OrderQuery

	// Wallet history enrich query.
	// GET /mall/me/orders の response に、
	// productName / measurements / color / tokenName / tokenIcon / brandName / brandIcon
	// を補完するため OrderHandler へ注入する。
	HistoryQ *mallquery.HistoryQuery

	// purchased orders
	OrderPurchasedQ *mallquery.OrderPurchasedQuery

	// verify scanned pair
	OrderScanVerifyQ *mallquery.OrderScanVerifyQuery

	// Shared query: walletAddress(toAddress) -> brandId / avatarId
	OwnerResolveQ *sharedquery.OwnerResolveQuery

	// /mall/me/avatar 用: uid -> avatarId を解決するRepo
	MeAvatarRepo *mallfs.MeAvatarRepo

	// /mall/me/setup-status 用 Repo（Firestore existence checks）
	SetupStatusRepo *mallfs.SetupStatusRepoFirestore
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

	// IMPORTANT: Config は必須（projectID 解決に必要）
	if infra.Config == nil {
		return nil, errors.New("di.mall: shared infra config is nil")
	}

	// required clients
	fsClient := infra.Firestore
	if fsClient == nil {
		return nil, errors.New("di.mall: infra.Firestore is nil")
	}

	c := &Container{Infra: infra}

	// --------------------------------------------------------
	// Firestore repositories
	// --------------------------------------------------------
	avatarRepo := outfs.NewAvatarRepositoryFS(fsClient)
	avatarStateRepo := outfs.NewAvatarStateRepositoryFS(fsClient)

	c.AvatarRepo = avatarRepo

	shippingAddressRepo := outfs.NewShippingAddressRepositoryFS(fsClient)
	paymentMethodRepo := outfs.NewPaymentMethodRepositoryFS(fsClient)
	userRepo := outfs.NewUserRepositoryFS(fsClient)
	walletRepo := outfs.NewWalletRepositoryFS(fsClient)
	productRepo := outfs.NewProductRepositoryFS(fsClient)

	// --------------------------------------------------------
	// Stripe adapter registration
	//
	// Stripe secret key policy:
	// - 旧 STRIPE_SECRET_KEY / infra.Config.StripeSecretKey は使わない
	// - Secret Manager の stripe-secret-key を正とする
	// - PaymentMethodGateway が nil のまま起動継続しない
	// --------------------------------------------------------
	{
		var customerStore stripeadapter.PaymentMethodCustomerStore
		if v, ok := any(paymentMethodRepo).(stripeadapter.PaymentMethodCustomerStore); ok {
			customerStore = v
		} else if v, ok := any(userRepo).(stripeadapter.PaymentMethodCustomerStore); ok {
			customerStore = v
		}

		if customerStore == nil {
			return nil, errors.New("di.mall: PaymentMethodCustomerStore is not implemented by current repositories")
		}

		if err := infra.RegisterPaymentMethodGatewayFromSecret(ctx, customerStore); err != nil {
			return nil, err
		}

		if infra.PaymentMethodGateway == nil {
			return nil, errors.New("di.mall: stripe payment method gateway is nil after registration")
		}
	}

	// resolvedTokens repo (wallets/{avatarId}/resolvedTokens/{mint})
	c.ResolvedTokenRepo = outfs.NewResolvedTokenRepositoryFS(fsClient)

	brandRepo := outfs.NewBrandRepositoryFS(fsClient)
	brandSvc := branddom.NewService(brandRepo)
	c.BrandService = brandSvc

	companyRepo := outfs.NewCompanyRepositoryFS(fsClient)
	companySvc := companydom.NewService(companyRepo)

	cartRepo := outfs.NewCartRepositoryFS(fsClient)
	paymentRepo := outfs.NewPaymentRepositoryFS(fsClient)
	orderRepo := outfs.NewOrderRepositoryFS(fsClient)

	// Inventory (Firestore)
	inventoryRepo := outfs.NewInventoryRepositoryFS(fsClient)

	// TokenBlueprint (tokenBlueprint domain)
	tokenBlueprintRepo := outfs.NewTokenBlueprintRepositoryFS(fsClient)

	// TokenBlueprintReview (YouTube-like comments)
	c.TokenBlueprintReviewRepo = outfs.NewTokenBlueprintReviewRepositoryFS(fsClient)

	// ProductBlueprintReview (Amazon-like product reviews) - reuse for UC + Query
	productBlueprintReviewRepo := outfs.NewProductBlueprintReviewRepositoryFS(fsClient)

	// ProductBlueprint (productBlueprint domain) - shared for queries/resolvers
	productBlueprintRepoFS := pbfs.NewProductBlueprintRepositoryFS(fsClient)
	productBlueprintSvc := productbpdom.NewService(productBlueprintRepoFS)

	// Shared instance for queries/resolvers (avoid duplication)
	modelRepoFS := outfs.NewModelRepositoryFS(fsClient)

	// /mall/me/avatar 用 Repo（uid -> avatarId）
	c.MeAvatarRepo = mallfs.NewMeAvatarRepo(fsClient)

	// /mall/me/setup-status 用 Repo（Firestore existence checks）
	c.SetupStatusRepo = mallfs.NewSetupStatusRepoFirestore(fsClient)

	// List repo
	listRepoFS := outfs.NewListRepositoryFS(fsClient)

	// Firestore repo for list images subcollection
	//
	// Firebase Storage migration policy:
	// - frontend が Firebase Storage へ直接 upload
	// - backend は Firestore の /lists/{listId}/images/{imageId} record を保存・取得・削除する
	// - ListImage.URL は Firebase Storage downloadURL
	listImageRecordRepo := outfs.NewListImageRepositoryFS(fsClient)

	// --------------------------------------------------------
	// Solana wallet service (AvatarWalletService)
	// projectID 解決は shared.Infra に委譲済み
	// --------------------------------------------------------
	projectID := infra.ProjectID
	avatarWalletSvc := solanainfra.NewAvatarWalletService(projectID)

	// --------------------------------------------------------
	// Usecases
	// --------------------------------------------------------

	// AvatarUsecase:
	// - avatarIcon は Firebase Storage download URL を Avatar.AvatarIcon に保存するだけ
	// - GCS avatar icon repo / object storage repo は渡さない
	// - cartRepo / walletRepo / walletSvc を必ず注入
	c.AvatarUC = avataruc.NewAvatarUsecase(
		avatarRepo,
		avatarStateRepo,
	).
		WithCartRepo(cartRepo).
		WithWalletRepo(walletRepo).
		WithWalletService(avatarWalletSvc)

	// ListUsecase: NewListUsecase だけを唯一の入口にする（With系は禁止）
	//
	// Firebase Storage migration policy:
	// - GCS list image repository は渡さない
	// - Firestore list image record repository のみ渡す
	c.ListUC = listuc.NewListUsecase(
		listRepoFS,
		listRepoFS,
		listRepoFS,
		listImageRecordRepo,
		listImageRecordRepo,
	)

	c.ShippingAddressUC = usecase.NewShippingAddressUsecase(shippingAddressRepo)
	c.PaymentMethodUC = usecase.NewPaymentMethodUsecase(
		paymentMethodRepo,
		infra.PaymentMethodGateway,
	)
	c.UserUC = usecase.NewUserUsecase(userRepo)

	// ========================================================
	// WalletUsecase
	// ========================================================
	onchainReader := solanaplatform.NewOnchainWalletReaderDevnet()
	tokenQuery := outfs.NewTokenReaderFS(fsClient)

	// productBlueprintId -> productName
	c.WalletUC = usecase.NewWalletUsecase(walletRepo).
		WithOnchainReader(onchainReader).
		WithTokenQuery(tokenQuery).
		WithBrandNameResolver(brandSvc).
		WithProductReader(productRepo).
		WithModelProductBlueprintIDResolver(productBlueprintRepoFS).
		WithProductBlueprintReader(productBlueprintRepoFS)

	// ========================================================
	// ProductBlueprintReviewUsecase
	// - VerifiedPurchase 判定のため WalletUsecase と同じ deps を注入
	// - avatarName/icon を返すため AvatarRepo も注入
	// ========================================================
	c.ProductBlueprintReviewUC = usecase.NewProductBlueprintReviewUsecase(
		productBlueprintReviewRepo,
		walletRepo,
	).
		WithOnchainReader(onchainReader).
		WithTokenQuery(tokenQuery).
		WithProductReader(productRepo).
		WithModelProductBlueprintIDResolver(productBlueprintRepoFS).
		WithAvatarRepo(avatarRepo)

	c.CartUC = usecase.NewCartUsecase(cartRepo)

	c.PaymentUC = usecase.NewPaymentUsecase(paymentRepo).
		WithCartRepoForPayment(cartRepo).
		WithOrderRepoForPayment(orderRepo).
		WithInventoryRepoForPayment(inventoryRepo).
		WithUserRepoForPayment(userRepo)

	c.OrderUC = usecase.NewOrderUsecase(orderRepo, cartRepo)

	// --------------------------------------------------------
	// Case A: PaymentFlowUsecase（payment起票 + 必要なら webhook trigger）
	// --------------------------------------------------------
	{
		pf, configured, err := buildPaymentFlowUsecase(infra, c.PaymentUC)
		if err != nil {
			return nil, err
		}
		c.PaymentFlowUC = pf
		_ = configured
	}

	// InventoryUsecase
	c.InventoryUC = usecase.NewInventoryUsecase(inventoryRepo)

	// --------------------------------------------------------
	// NameResolver (optional but useful for mall queries)
	// --------------------------------------------------------
	{
		memberRepo := outfs.NewMemberRepositoryFS(fsClient)

		c.NameResolver = appresolver.NewNameResolver(
			brandRepo,
			companyRepo,
			productBlueprintRepoFS,
			memberRepo,
			modelRepoFS,
			tokenBlueprintRepo,
		)
	}

	// --------------------------------------------------------
	// Shared Query: OwnerResolve
	// --------------------------------------------------------
	{
		brandsCol := infra.BrandsCollection
		avatarsCol := infra.AvatarsCollection

		brandReader := mallfs.NewBrandWalletAddressReaderFS(fsClient, brandsCol)
		avatarReader := mallfs.NewAvatarWalletAddressReaderFS(fsClient, avatarsCol)

		c.OwnerResolveQ = sharedquery.NewOwnerResolveQuery(
			avatarReader,
			brandReader,
			avatarRepo,
			brandSvc,
		)
	}

	// --------------------------------------------------------
	// Queries (mall-facing)
	// --------------------------------------------------------
	{
		c.BrandQ = mallquery.NewBrandQuery(
			brandRepo,
			companySvc,
			productBlueprintRepoFS,
			tokenBlueprintRepo,
			listRepoFS,
		)

		c.CatalogQ = catalogQuery.NewCatalogQuery(
			listRepoFS,
			inventoryRepo,
			productBlueprintRepoFS,
			modelRepoFS,
			catalogQuery.WithListImageRepo(listImageRecordRepo),
			catalogQuery.WithTokenBlueprintPatchRepo(tokenBlueprintRepo),
			catalogQuery.WithProductBlueprintReviewRepo(productBlueprintReviewRepo),
			catalogQuery.WithNameResolver(c.NameResolver),
		)

		c.CartQ = mallquery.NewCartQuery(fsClient)

		tokenReader := outfs.NewTokenReaderFS(fsClient)

		solanaTransferReader := solanainfra.NewTokenTransferReaderSolana("")
		previewTransferReader := outsolana.NewPreviewTransferReader(solanaTransferReader)

		c.PreviewQ = mallquery.NewPreviewQuery(
			productRepo,
			modelRepoFS,
			productBlueprintRepoFS,
			mallquery.WithNameResolver(c.NameResolver),
			mallquery.WithTokenRepo(tokenReader),
			mallquery.WithTokenBlueprintRepo(tokenBlueprintRepo),
			mallquery.WithOwnerResolveQuery(c.OwnerResolveQ),
			mallquery.WithBrandNameIconRepo(brandSvc),
			mallquery.WithAvatarNameIconRepo(avatarRepo),
			mallquery.WithTransferRepo(previewTransferReader),
		)

		c.OrderQ = mallquery.NewOrderQuery(fsClient)

		historyModelResolver := mallquery.NewHistoryModelResolver(modelRepoFS)
		c.HistoryQ = mallquery.NewHistoryQuery(
			inventoryRepo,
			productBlueprintSvc,
			tokenBlueprintRepo,
			brandSvc,
			historyModelResolver,
		)

		c.OrderPurchasedQ = mallquery.NewOrderPurchasedQuery(fsClient)
		c.OrderScanVerifyQ = mallquery.NewOrderScanVerifyQuery(c.OrderPurchasedQ, c.PreviewQ)

		// light injection
		if c.CartQ != nil && c.NameResolver != nil && c.CartQ.Resolver == nil {
			c.CartQ.Resolver = c.NameResolver
		}
		if c.CartQ != nil && listRepoFS != nil && c.CartQ.ListRepo == nil {
			c.CartQ.ListRepo = listRepoFS
		}
	}

	// --------------------------------------------------------
	// TransferUsecase wiring (order scan verified -> brand->avatar transfer)
	// --------------------------------------------------------
	{
		scanVerifier := buildScanVerifier(c.OrderScanVerifyQ)

		var orderRepoForTransfer usecase.OrderRepoForTransfer = outfs.NewOrderRepoForTransferFS(fsClient)

		var tokenResolver usecase.TokenResolver = mallfs.NewTokenResolverFS(fsClient, "tokens")
		var tokenOwnerUpdater usecase.TokenOwnerUpdater = mallfs.NewTokenOwnerUpdaterFS(fsClient, "tokens")

		var walletItemUpdater usecase.WalletItemUpdater = walletRepo
		var transferRepo usecase.TransferRepo = outfs.NewTransferRepositoryFS(fsClient)

		var walletResolver usecase.BrandWalletResolver = outfs.NewWalletResolverRepoFS(brandRepo, walletRepo)
		var avatarWalletResolver usecase.AvatarWalletResolver = walletResolver.(usecase.AvatarWalletResolver)

		secrets, err := buildWalletSecretProvider(infra)
		if err != nil {
			return nil, err
		}

		var executor usecase.TokenTransferExecutor = solanainfra.NewTokenTransferExecutorSolana("")

		if scanVerifier != nil && secrets != nil {
			c.TransferUC = usecase.NewTransferUsecase(
				scanVerifier,
				orderRepoForTransfer,
				tokenResolver,
				tokenOwnerUpdater,
				walletItemUpdater,
				transferRepo,
				walletResolver,
				avatarWalletResolver,
				secrets,
				executor,
			).WithInventoryRepo(inventoryRepo)
		} else {
			c.TransferUC = nil
		}
	}

	// --------------------------------------------------------
	// ShareTransferUsecase wiring (avatar -> avatar transfer)
	// --------------------------------------------------------
	{
		var tokenResolver usecase.TokenResolver = mallfs.NewTokenResolverFS(fsClient, "tokens")
		var tokenOwnerUpdater usecase.TokenOwnerUpdater = mallfs.NewTokenOwnerUpdaterFS(fsClient, "tokens")
		var transferRepo usecase.TransferRepo = outfs.NewTransferRepositoryFS(fsClient)

		var walletResolver usecase.BrandWalletResolver = outfs.NewWalletResolverRepoFS(brandRepo, walletRepo)
		var avatarWalletResolver usecase.AvatarWalletResolver = walletResolver.(usecase.AvatarWalletResolver)

		secretsBase, err := buildWalletSecretProvider(infra)
		if err != nil {
			return nil, err
		}

		var executor usecase.TokenTransferExecutor = solanainfra.NewTokenTransferExecutorSolana("")

		walletUpdate, walletOK := any(walletRepo).(usecase.AvatarWalletItemTransferUpdater)
		avatarSecrets, secretOK := any(secretsBase).(usecase.AvatarSecretProvider)
		walletSync, syncOK := any(c.WalletUC).(usecase.AvatarWalletSyncer)

		switch {
		case !walletOK:
			c.ShareTransferUC = nil
		case !secretOK:
			c.ShareTransferUC = nil
		case !syncOK:
			c.ShareTransferUC = nil
		default:
			c.ShareTransferUC = usecase.NewShareTransferUsecase(
				tokenResolver,
				tokenOwnerUpdater,
				walletUpdate,
				walletSync,
				transferRepo,
				avatarWalletResolver,
				avatarSecrets,
				executor,
			)
		}
	}

	return c, nil
}
