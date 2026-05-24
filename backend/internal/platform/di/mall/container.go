package mall

import (
	"context"
	"errors"

	mallquery "narratives/internal/application/query/mall"
	catalogQuery "narratives/internal/application/query/mall/catalog"
	sharedquery "narratives/internal/application/query/shared"
	appresolver "narratives/internal/application/resolver"

	usecase "narratives/internal/application/usecase"

	mallhandler "narratives/internal/adapters/in/http/mall/handler"

	outfs "narratives/internal/adapters/out/firestore"
	mallfs "narratives/internal/adapters/out/firestore/mall"
	pbfs "narratives/internal/adapters/out/firestore/productBlueprint"
	outsolana "narratives/internal/adapters/out/solana"
	stripeadapter "narratives/internal/adapters/out/stripe"

	solanainfra "narratives/internal/infra/solana"
	solanaplatform "narratives/internal/infra/solana"

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

type Container struct {
	Infra *shared.Infra

	AvatarUC          *usecase.AvatarUsecase
	ListUC            *usecase.ListUsecase
	ShippingAddressUC *usecase.ShippingAddressUsecase
	PaymentMethodUC   *usecase.PaymentMethodUsecase
	UserUC            *usecase.UserUsecase
	WalletUC          *usecase.WalletUsecase
	CartUC            *usecase.CartUsecase
	PaymentUC         *usecase.PaymentUsecase
	OrderUC           *usecase.OrderUsecase

	AvatarRepo   avatardom.Repository
	BrandService *branddom.Service

	ProductBlueprintReviewUC *usecase.ProductBlueprintReviewUsecase

	TransferUC *usecase.TransferUsecase

	ShareTransferUC *usecase.ShareTransferUsecase

	PaymentFlowUC *usecase.PaymentFlowUsecase

	InventoryUC *usecase.InventoryUsecase

	TokenBlueprintReviewRepo tokenBlueprint_review.RepositoryPort

	ResolvedTokenRepo mallhandler.ResolvedTokenRepository

	NameResolver *appresolver.NameResolver

	BrandQ   *mallquery.BrandQuery
	CatalogQ *catalogQuery.CatalogQuery
	CartQ    *mallquery.CartQuery
	PreviewQ *mallquery.PreviewQuery

	OrderQ *mallquery.OrderQuery

	HistoryQ *mallquery.HistoryQuery

	OrderPurchasedQ *mallquery.OrderPurchasedQuery

	OrderScanVerifyQ *mallquery.OrderScanVerifyQuery

	OwnerResolveQ *sharedquery.OwnerResolveQuery

	MeAvatarRepo *mallfs.MeAvatarRepo

	SetupStatusRepo *mallfs.SetupStatusRepoFirestore
}

func NewContainer(ctx context.Context, infra *shared.Infra) (*Container, error) {
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

	if infra.Config == nil {
		return nil, errors.New("di.mall: shared infra config is nil")
	}

	fsClient := infra.Firestore
	if fsClient == nil {
		return nil, errors.New("di.mall: infra.Firestore is nil")
	}

	c := &Container{Infra: infra}

	avatarRepo := outfs.NewAvatarRepositoryFS(fsClient)
	avatarStateRepo := outfs.NewAvatarStateRepositoryFS(fsClient)

	c.AvatarRepo = avatarRepo

	shippingAddressRepo := outfs.NewShippingAddressRepositoryFS(fsClient)
	paymentMethodRepo := outfs.NewPaymentMethodRepositoryFS(fsClient)
	userRepo := outfs.NewUserRepositoryFS(fsClient)
	walletRepo := outfs.NewWalletRepositoryFS(fsClient)
	productRepo := outfs.NewProductRepositoryFS(fsClient)

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

	c.ResolvedTokenRepo = outfs.NewResolvedTokenRepositoryFS(fsClient)

	brandRepo := outfs.NewBrandRepositoryFS(fsClient)
	brandSvc := branddom.NewService(brandRepo)
	c.BrandService = brandSvc

	companyRepo := outfs.NewCompanyRepositoryFS(fsClient)
	companySvc := companydom.NewService(companyRepo)

	cartRepo := outfs.NewCartRepositoryFS(fsClient)
	paymentRepo := outfs.NewPaymentRepositoryFS(fsClient)
	orderRepo := outfs.NewOrderRepositoryFS(fsClient)

	inventoryRepo := outfs.NewInventoryRepositoryFS(fsClient)

	tokenBlueprintRepo := outfs.NewTokenBlueprintRepositoryFS(fsClient)

	c.TokenBlueprintReviewRepo = outfs.NewTokenBlueprintReviewRepositoryFS(fsClient)

	productBlueprintReviewRepo := outfs.NewProductBlueprintReviewRepositoryFS(fsClient)

	productBlueprintRepoFS := pbfs.NewProductBlueprintRepositoryFS(fsClient)
	productBlueprintSvc := productbpdom.NewService(productBlueprintRepoFS)

	modelRepoFS := outfs.NewModelRepositoryFS(fsClient)

	c.MeAvatarRepo = mallfs.NewMeAvatarRepo(fsClient)

	c.SetupStatusRepo = mallfs.NewSetupStatusRepoFirestore(fsClient)

	listRepoFS := outfs.NewListRepositoryFS(fsClient)

	listImageRecordRepo := outfs.NewListImageRepositoryFS(fsClient)

	projectID := infra.ProjectID
	avatarWalletSvc := solanainfra.NewAvatarWalletService(projectID)

	c.AvatarUC = usecase.NewAvatarUsecase(
		avatarRepo,
		avatarStateRepo,
	).
		WithCartRepo(cartRepo).
		WithWalletRepo(walletRepo).
		WithWalletService(avatarWalletSvc)

	c.ListUC = usecase.NewListUsecase(
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

	onchainReader := solanaplatform.NewOnchainWalletReaderDevnet()
	tokenQuery := outfs.NewTokenReaderFS(fsClient)

	c.WalletUC = usecase.NewWalletUsecase(walletRepo).
		WithOnchainReader(onchainReader).
		WithTokenQuery(tokenQuery).
		WithBrandNameResolver(brandSvc).
		WithProductReader(productRepo).
		WithModelProductBlueprintIDResolver(productBlueprintRepoFS).
		WithProductBlueprintReader(productBlueprintRepoFS)

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

	{
		pf, configured, err := buildPaymentFlowUsecase(infra, c.PaymentUC)
		if err != nil {
			return nil, err
		}
		c.PaymentFlowUC = pf
		_ = configured
	}

	c.InventoryUC = usecase.NewInventoryUsecase(inventoryRepo)

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

		if c.CartQ != nil && c.NameResolver != nil && c.CartQ.Resolver == nil {
			c.CartQ.Resolver = c.NameResolver
		}

		if c.CartQ != nil && listRepoFS != nil && c.CartQ.ListRepo == nil {
			c.CartQ.ListRepo = listRepoFS
		}
	}

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
			).
				WithInventoryRepo(inventoryRepo).
				WithTransferDisplayResolvers(brandSvc, avatarRepo)
		} else {
			c.TransferUC = nil
		}
	}

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
