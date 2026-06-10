// backend/internal/platform/di/mall/container.go
package mall

import (
	"context"
	"errors"

	mallquery "narratives/internal/application/query/mall"
	sharedquery "narratives/internal/application/query/shared"
	appresolver "narratives/internal/application/resolver"

	usecase "narratives/internal/application/usecase"

	mallhandler "narratives/internal/adapters/in/http/mall/handler"

	outfs "narratives/internal/adapters/out/firestore"
	mallfs "narratives/internal/adapters/out/firestore/mall"
	outsolana "narratives/internal/adapters/out/solana"
	stripeadapter "narratives/internal/adapters/out/stripe"

	solana "narratives/internal/infra/solana"

	avatardom "narratives/internal/domain/avatar"
	branddom "narratives/internal/domain/brand"
	tokenblueprintreview "narratives/internal/domain/tokenBlueprint_review"

	shared "narratives/internal/platform/di/shared"
)

const (
	StripeWebhookPath = "/mall/webhooks/stripe"
)

type Container struct {
	Infra *shared.Infra

	AvatarUC          *usecase.AvatarUsecase
	SetupUC           *usecase.SetupUsecase
	ListUC            *usecase.ListUsecase
	ShippingAddressUC *usecase.ShippingAddressUsecase
	PaymentMethodUC   *usecase.PaymentMethodUsecase
	UserUC            *usecase.UserUsecase
	WalletUC          *usecase.WalletUsecase
	CartUC            *usecase.CartUsecase
	PaymentUC         *usecase.PaymentUsecase
	OrderUC           *usecase.OrderUsecase

	AvatarRepo avatardom.Repository
	BrandRepo  branddom.Repository

	// MeAvatarResolver resolves Firebase UID -> avatarId + walletAddress.
	// AvatarRepositoryFS implements this via ResolveAvatarByUID.
	MeAvatarResolver mallhandler.MeAvatarResolver

	ProductBlueprintReviewUC *usecase.ProductBlueprintReviewUsecase

	TransferUC *usecase.TransferUsecase

	ShareTransferUC *usecase.ShareTransferUsecase

	PaymentFlowUC *usecase.PaymentFlowUsecase

	InventoryUC *usecase.InventoryUsecase

	TokenBlueprintReviewRepo tokenblueprintreview.RepositoryPort

	NameResolver *appresolver.NameResolver

	BrandQ   *mallquery.BrandQuery
	ListQ    *mallquery.ListQuery
	CatalogQ *mallquery.CatalogQuery
	CartQ    *mallquery.CartQuery
	PreviewQ *mallquery.PreviewQuery

	OrderQ *mallquery.OrderQuery

	HistoryQ *mallquery.HistoryQuery

	OwnerResolveQ *sharedquery.OwnerResolveQuery
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
	c.MeAvatarResolver = avatarRepo
	c.SetupUC = usecase.NewSetupUsecase(avatarRepo)

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

	brandRepo := outfs.NewBrandRepositoryFS(fsClient)
	c.BrandRepo = brandRepo

	companyRepo := outfs.NewCompanyRepositoryFS(fsClient)

	cartRepo := outfs.NewCartRepositoryFS(fsClient)
	paymentRepo := outfs.NewPaymentRepositoryFS(fsClient)
	orderRepo := outfs.NewOrderRepositoryFS(fsClient)

	inventoryRepo := outfs.NewInventoryRepositoryFS(fsClient)

	tokenBlueprintRepo := outfs.NewTokenBlueprintRepositoryFS(fsClient)

	c.TokenBlueprintReviewRepo = outfs.NewTokenBlueprintReviewRepositoryFS(fsClient)

	productBlueprintReviewRepo := outfs.NewProductBlueprintReviewRepositoryFS(fsClient)

	productBlueprintRepoFS := outfs.NewProductBlueprintRepositoryFS(fsClient)

	modelRepoFS := outfs.NewModelRepositoryFS(fsClient)

	listRepoFS := outfs.NewListRepositoryFS(fsClient)

	listImageRecordRepo := outfs.NewListImageRepositoryFS(fsClient)

	projectID := infra.ProjectID
	avatarWalletSvc := solana.NewAvatarWalletService(projectID)

	c.AvatarUC = usecase.NewAvatarUsecase(
		avatarRepo,
		avatarStateRepo,
		avatarWalletSvc,
		walletRepo,
		cartRepo,
		nil,
	)

	c.ListUC = usecase.NewListUsecase(
		listRepoFS,
		listImageRecordRepo,
	)

	c.ListQ = mallquery.NewListQuery(
		listRepoFS,
		listImageRecordRepo,
	)

	c.ShippingAddressUC = usecase.NewShippingAddressUsecase(shippingAddressRepo)
	c.PaymentMethodUC = usecase.NewPaymentMethodUsecase(
		paymentMethodRepo,
		infra.PaymentMethodGateway,
	)
	c.UserUC = usecase.NewUserUsecase(userRepo, nil)

	onchainReader := solana.NewOnchainWalletReaderDevnet()
	tokenQuery := outfs.NewTokenReaderFS(fsClient)

	c.WalletUC = usecase.NewWalletUsecase(
		walletRepo,
		onchainReader,
		tokenQuery,
		brandRepo,
		productRepo,
		productBlueprintRepoFS,
		productBlueprintRepoFS,
	)

	c.ProductBlueprintReviewUC = usecase.NewProductBlueprintReviewUsecase(
		productBlueprintReviewRepo,
		walletRepo,
		productBlueprintRepoFS,
		brandRepo,
		onchainReader,
		tokenQuery,
		productRepo,
		productBlueprintRepoFS,
		avatarRepo,
		nil,
	)

	c.CartUC = usecase.NewCartUsecase(cartRepo)

	c.PaymentUC = usecase.NewPaymentUsecase(usecase.NewPaymentUsecaseInput{
		PaymentRepo: paymentRepo,

		CartRepo:      cartRepo,
		OrderRepo:     orderRepo,
		InventoryRepo: inventoryRepo,

		UserRepo: userRepo,
	})

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
			userRepo,
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
			brandRepo,
		)
	}

	{
		c.BrandQ = mallquery.NewBrandQuery(
			brandRepo,
			companyRepo,
			productBlueprintRepoFS,
			inventoryRepo,
			listRepoFS,
		)

		c.CatalogQ = mallquery.NewCatalogQuery(
			listRepoFS,
			inventoryRepo,
			productBlueprintRepoFS,
			modelRepoFS,
			listImageRecordRepo,
			tokenBlueprintRepo,
			productBlueprintReviewRepo,
			c.NameResolver,
		)

		c.CartQ = mallquery.NewCartQuery(
			cartRepo,
			listRepoFS,
			inventoryRepo,
			c.NameResolver,
		)

		tokenReader := outfs.NewTokenReaderFS(fsClient)

		solanaTransferReader := solana.NewTokenTransferReaderSolana("")
		previewTransferReader := outsolana.NewPreviewTransferReader(solanaTransferReader)

		c.PreviewQ = mallquery.NewPreviewQuery(
			productRepo,
			productBlueprintRepoFS,
			orderRepo,
			c.NameResolver,
			tokenReader,
			tokenBlueprintRepo,
			c.OwnerResolveQ,
			brandRepo,
			avatarRepo,
			previewTransferReader,
		)

		c.OrderQ = mallquery.NewOrderQuery(
			avatarRepo,
			cartRepo,
			shippingAddressRepo,
			paymentMethodRepo,
			c.NameResolver,
		)

		c.HistoryQ = mallquery.NewHistoryQuery(
			inventoryRepo,
			productBlueprintRepoFS,
			tokenBlueprintRepo,
			brandRepo,
			c.NameResolver,
		)
	}

	{
		scanVerifier := buildScanVerifier(c.PreviewQ)
		if scanVerifier == nil {
			return nil, errors.New("di.mall: scan verifier is nil")
		}

		var orderRepoForTransfer usecase.OrderRepoForTransfer = outfs.NewOrderRepoForTransferFS(fsClient)

		var tokenResolver usecase.TokenResolver = mallfs.NewTokenResolverFS(fsClient, "tokens")
		var tokenOwnerUpdater usecase.TokenOwnerUpdater = outfs.NewTokenOwnerUpdaterFS(fsClient)

		var walletItemUpdater usecase.WalletItemUpdater = walletRepo
		var transferRepo usecase.TransferRepo = outfs.NewTransferRepositoryFS(fsClient)

		var walletResolver usecase.BrandWalletResolver = outfs.NewWalletResolverRepoFS(brandRepo, walletRepo)
		var avatarWalletResolver usecase.AvatarWalletResolver = walletResolver.(usecase.AvatarWalletResolver)

		secrets, err := buildWalletSecretProvider(infra)
		if err != nil {
			return nil, err
		}
		if secrets == nil {
			return nil, errors.New("di.mall: wallet secret provider is nil")
		}

		var executor usecase.TokenTransferExecutor = solana.NewTokenTransferExecutorSolana("")

		c.TransferUC = usecase.NewTransferUsecase(
			scanVerifier,
			orderRepoForTransfer,
			tokenResolver,
			tokenOwnerUpdater,
			walletItemUpdater,
			transferRepo,
			walletResolver,
			avatarWalletResolver,
			brandRepo,
			avatarRepo,
			secrets,
			executor,
			nil,
			c.InventoryUC,
		)
	}

	{
		var tokenResolver usecase.TokenResolver = mallfs.NewTokenResolverFS(fsClient, "tokens")
		var tokenOwnerUpdater usecase.TokenOwnerUpdater = outfs.NewTokenOwnerUpdaterFS(fsClient)
		var transferRepo usecase.TransferRepo = outfs.NewTransferRepositoryFS(fsClient)

		var walletResolver usecase.BrandWalletResolver = outfs.NewWalletResolverRepoFS(brandRepo, walletRepo)
		var avatarWalletResolver usecase.AvatarWalletResolver = walletResolver.(usecase.AvatarWalletResolver)

		secretsBase, err := buildWalletSecretProvider(infra)
		if err != nil {
			return nil, err
		}

		var executor usecase.TokenTransferExecutor = solana.NewTokenTransferExecutorSolana("")

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
