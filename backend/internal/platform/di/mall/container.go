// backend/internal/platform/di/mall/container.go
package mall

import (
	"context"
	"errors"
	"os"

	firebaseauth "firebase.google.com/go/v4/auth"

	mallquery "narratives/internal/application/query/mall"
	mallshared "narratives/internal/application/query/mall/shared"
	sharedquery "narratives/internal/application/query/shared"
	appresolver "narratives/internal/application/resolver"
	usecase "narratives/internal/application/usecase"

	mallhandler "narratives/internal/adapters/in/http/mall/handler"

	outfirebase "narratives/internal/adapters/out/firebase"
	outfs "narratives/internal/adapters/out/firestore"
	mallfs "narratives/internal/adapters/out/firestore/mall"
	sharedfs "narratives/internal/adapters/out/firestore/shared"
	mailadp "narratives/internal/adapters/out/mail"
	outsolana "narratives/internal/adapters/out/solana"
	stripeadapter "narratives/internal/adapters/out/stripe"

	avatardom "narratives/internal/domain/avatar"
	branddom "narratives/internal/domain/brand"
	resaledom "narratives/internal/domain/resale"
	tokenblueprintreview "narratives/internal/domain/tokenBlueprint_review"
	transferdom "narratives/internal/domain/transfer"
	transportationdom "narratives/internal/domain/transportation"

	solana "narratives/internal/infra/solana"

	shared "narratives/internal/platform/di/shared"
)

const (
	StripeWebhookPath = "/mall/webhooks/stripe"
)

type firebaseAuthEmailGetter struct {
	client *firebaseauth.Client
}

func newFirebaseAuthEmailGetter(
	client *firebaseauth.Client,
) usecase.AuthUserEmailGetter {
	if client == nil {
		return nil
	}

	return &firebaseAuthEmailGetter{
		client: client,
	}
}

func (g *firebaseAuthEmailGetter) GetEmailByUID(
	ctx context.Context,
	uid string,
) (string, error) {
	if g == nil || g.client == nil {
		return "",
			errors.New(
				"firebase auth email getter is not configured",
			)
	}

	if uid == "" {
		return "",
			errors.New("firebase auth uid is empty")
	}

	userRecord, err := g.client.GetUser(ctx, uid)
	if err != nil {
		return "", err
	}

	if userRecord == nil {
		return "",
			errors.New(
				"firebase auth user record is nil",
			)
	}

	return userRecord.Email, nil
}

type Container struct {
	Infra *shared.Infra

	AvatarUC          *usecase.AvatarUsecase
	SetupUC           *usecase.SetupUsecase
	ListUC            *usecase.ListUsecase
	ShippingAddressUC *usecase.ShippingAddressUsecase
	ShippingQuoteUC   *usecase.ShippingQuoteUsecase
	PaymentMethodUC   *usecase.PaymentMethodUsecase
	UserUC            *usecase.UserUsecase
	WalletUC          *usecase.WalletUsecase
	CartUC            *usecase.CartUsecase
	PaymentUC         *usecase.PaymentUsecase
	OrderUC           *usecase.OrderUsecase
	InquiryUC         *usecase.InquiryUsecase
	AnnouncementUC    *usecase.AnnouncementUsecase
	ResaleUC          *usecase.ResaleUsecase

	OrderMailer   *mailadp.OrderMailer
	OrderMailFrom string

	InquiryMailer *mailadp.InquiryMailer
	InquiryMailTo string

	AvatarRepo avatardom.Repository
	BrandRepo  branddom.Repository

	ResaleRepo      resaledom.Repository
	ResaleImageRepo resaledom.ImageRepository

	MeAvatarResolver mallhandler.MeAvatarResolver

	ProductBlueprintReviewUC *usecase.ProductBlueprintReviewUsecase

	TransferUC      *usecase.TransferUsecase
	ShareTransferUC *usecase.ShareTransferUsecase
	PaymentFlowUC   *usecase.PaymentFlowUsecase
	InventoryUC     *usecase.InventoryUsecase

	TokenBlueprintReviewRepo tokenblueprintreview.RepositoryPort

	NameResolver *appresolver.NameResolver

	BrandQ        *mallquery.BrandQuery
	ListQ         *mallquery.ListQuery
	CatalogQ      *mallquery.CatalogQuery
	CartQ         *mallquery.CartQuery
	PreviewQ      *mallquery.PreviewQuery
	InquiryQ      *mallquery.InquiryQuery
	AnnouncementQ *mallquery.AnnouncementQueryService
	ResaleQ       *mallquery.ResaleQuery
	MarketQ       *mallquery.MarketQuery
	OrderQ        *mallquery.OrderQuery
	HistoryQ      *mallquery.HistoryQuery

	OwnerResolveQ *sharedquery.OwnerResolveQuery
}

func NewContainer(
	ctx context.Context,
	infra *shared.Infra,
) (*Container, error) {
	if infra == nil {
		var err error

		infra, err = shared.NewInfra(ctx)
		if err != nil {
			return nil, err
		}
	}

	if infra == nil {
		return nil,
			errors.New("di.mall: shared infra is nil")
	}

	if infra.Config == nil {
		return nil,
			errors.New(
				"di.mall: shared infra config is nil",
			)
	}

	fsClient := infra.Firestore
	if fsClient == nil {
		return nil,
			errors.New(
				"di.mall: infra.Firestore is nil",
			)
	}

	c := &Container{
		Infra: infra,
	}

	authUserEmailGetter :=
		newFirebaseAuthEmailGetter(
			infra.FirebaseAuth,
		)

	avatarRepo :=
		outfs.NewAvatarRepositoryFS(
			fsClient,
		)

	c.AvatarRepo = avatarRepo
	c.MeAvatarResolver = avatarRepo
	c.SetupUC = usecase.NewSetupUsecase(
		avatarRepo,
	)

	shippingAddressRepo :=
		outfs.NewShippingAddressRepositoryFS(
			fsClient,
		)

	paymentMethodRepo :=
		outfs.NewPaymentMethodRepositoryFS(
			fsClient,
		)

	userRepo :=
		outfs.NewUserRepositoryFS(
			fsClient,
		)

	memberRepo :=
		outfs.NewMemberRepositoryFS(
			fsClient,
		)

	walletRepo :=
		outfs.NewWalletRepositoryFS(
			fsClient,
		)

	productRepo :=
		outfs.NewProductRepositoryFS(
			fsClient,
		)

	{
		var customerStore stripeadapter.PaymentMethodCustomerStore

		if value, ok :=
			any(paymentMethodRepo).(stripeadapter.PaymentMethodCustomerStore); ok {
			customerStore = value
		} else if value, ok :=
			any(userRepo).(stripeadapter.PaymentMethodCustomerStore); ok {
			customerStore = value
		}

		if customerStore == nil {
			return nil,
				errors.New(
					"di.mall: PaymentMethodCustomerStore is not implemented by current repositories",
				)
		}

		if err :=
			infra.RegisterPaymentMethodGatewayFromSecret(
				ctx,
				customerStore,
			); err != nil {
			return nil, err
		}

		if infra.PaymentMethodGateway == nil {
			return nil,
				errors.New(
					"di.mall: stripe payment method gateway is nil after registration",
				)
		}
	}

	brandRepo :=
		outfs.NewBrandRepositoryFS(
			fsClient,
		)

	c.BrandRepo = brandRepo

	companyRepo :=
		outfs.NewCompanyRepositoryFS(
			fsClient,
		)

	cartRepo :=
		outfs.NewCartRepositoryFS(
			fsClient,
		)

	paymentRepo :=
		outfs.NewPaymentRepositoryFS(
			fsClient,
		)

	orderRepo :=
		outfs.NewOrderRepositoryFS(
			fsClient,
		)

	// The projection repository is shared by PreviewQuery and TransferUsecase.
	orderTransferItemRepo :=
		outfs.NewOrderRepoForTransferFS(
			fsClient,
		)

	inventoryRepo :=
		outfs.NewInventoryRepositoryFS(
			fsClient,
		)

	tokenBlueprintRepo :=
		outfs.NewTokenBlueprintRepositoryFS(
			fsClient,
		)

	productBlueprintRepoFS :=
		outfs.NewProductBlueprintRepositoryFS(
			fsClient,
		)

	modelRepoFS :=
		outfs.NewModelRepositoryFS(
			fsClient,
		)

	mallDisplayResolver :=
		mallshared.NewDisplayResolver(
			productRepo,
			modelRepoFS,
			productBlueprintRepoFS,
			tokenBlueprintRepo,
			brandRepo,
		)

	inquiryRepo :=
		outfs.NewInquiryRepositoryFS(
			fsClient,
		)

	inquiryReplyRepo :=
		outfs.NewInquiryReplyRepositoryFS(
			fsClient,
		)

	c.InquiryQ =
		mallquery.NewInquiryQuery(
			inquiryRepo,
			inquiryReplyRepo,
		)

	announcementRepo :=
		outfs.NewAnnouncementRepositoryFS(
			fsClient,
		)

	announcementAvatarRepo :=
		outfs.NewAnnouncementAvatarRepositoryFS(
			fsClient,
		)

	announcementAttachmentRepo :=
		outfs.NewAnnouncementAttachmentRepositoryFS(
			fsClient,
		)

	c.AnnouncementUC =
		usecase.NewAnnouncementUsecase(
			announcementRepo,
			announcementAvatarRepo,
			announcementAttachmentRepo,
		)

	c.AnnouncementQ =
		mallquery.NewAnnouncementQueryService(
			announcementRepo,
			announcementAvatarRepo,
			announcementAttachmentRepo,
			tokenBlueprintRepo,
		)

	c.TokenBlueprintReviewRepo =
		outfs.NewTokenBlueprintReviewRepositoryFS(
			fsClient,
		)

	productBlueprintReviewRepo :=
		outfs.NewProductBlueprintReviewRepositoryFS(
			fsClient,
		)

	listRepoFS :=
		outfs.NewListRepositoryFS(
			fsClient,
		)

	listImageRecordRepo :=
		outfs.NewListImageRepositoryFS(
			fsClient,
		)

	resaleRepo :=
		outfs.NewResaleRepositoryFS(
			fsClient,
		)

	resaleImageRepo :=
		outfs.NewResaleImageRepositoryFS(
			fsClient,
		)

	resaleImageStorage, err :=
		outfirebase.NewResaleImageStorageFromEnv(
			ctx,
		)
	if err != nil {
		return nil, err
	}

	c.ResaleRepo = resaleRepo
	c.ResaleImageRepo = resaleImageRepo

	c.ResaleUC =
		usecase.NewResaleUsecase(
			resaleRepo,
			resaleImageRepo,
			resaleImageStorage,
		)

	c.ResaleQ =
		mallquery.NewResaleQuery(
			resaleRepo,
			resaleImageRepo,
			mallDisplayResolver,
		)

	c.MarketQ =
		mallquery.NewMarketQuery(
			resaleRepo,
			resaleImageRepo,
			mallDisplayResolver,
			avatarRepo,
		)

	c.OrderMailer =
		mailadp.NewOrderMailer(
			mailadp.NewResendClient(
				os.Getenv("RESEND_API_KEY"),
			),
			modelRepoFS,
			inventoryRepo,
			productBlueprintRepoFS,
			tokenBlueprintRepo,
			brandRepo,
			companyRepo,
		)

	c.OrderMailFrom =
		os.Getenv("RESEND_FROM")

	c.InquiryMailer =
		mailadp.NewInquiryMailer(
			mailadp.NewResendClient(
				os.Getenv("RESEND_API_KEY"),
			),
		)

	c.InquiryMailTo =
		os.Getenv("INQUIRY_MAIL_TO")

	projectID := infra.ProjectID

	avatarWalletSvc :=
		solana.NewAvatarWalletService(
			projectID,
		)

	c.AvatarUC =
		usecase.NewAvatarUsecase(
			avatarRepo,
			avatarWalletSvc,
			walletRepo,
			cartRepo,
			nil,
		)

	transportationRepo :=
		outfs.NewTransportationRepositoryFS(
			fsClient,
		)

	transportationSvc :=
		transportationdom.NewService(
			transportationRepo,
		)

	c.ShippingQuoteUC =
		usecase.NewShippingQuoteUsecase(
			listRepoFS,
			inventoryRepo,
			modelRepoFS,
			shippingAddressRepo,
			transportationSvc,
		)

	c.ListUC =
		usecase.NewListUsecase(
			listRepoFS,
			listImageRecordRepo,
			transportationRepo,
			nil,
		)

	c.ListQ =
		mallquery.NewListQuery(
			listRepoFS,
			listImageRecordRepo,
		)

	c.ShippingAddressUC =
		usecase.NewShippingAddressUsecase(
			shippingAddressRepo,
		)

	c.PaymentMethodUC =
		usecase.NewPaymentMethodUsecase(
			paymentMethodRepo,
			infra.PaymentMethodGateway,
		)

	c.UserUC =
		usecase.NewUserUsecase(
			userRepo,
			nil,
		)

	onchainReader :=
		solana.NewOnchainWalletReaderDevnet()

	tokenQuery :=
		outfs.NewTokenReaderFS(
			fsClient,
		)

	c.WalletUC =
		usecase.NewWalletUsecase(
			walletRepo,
			onchainReader,
			tokenQuery,
			brandRepo,
			productRepo,
			productBlueprintRepoFS,
			productBlueprintRepoFS,
		)

	c.ProductBlueprintReviewUC =
		usecase.NewProductBlueprintReviewUsecase(
			productBlueprintReviewRepo,
			productBlueprintRepoFS,
			brandRepo,
			memberRepo,
			c.WalletUC,
			avatarRepo,
			nil,
		)

	c.CartUC =
		usecase.NewCartUsecase(
			cartRepo,
		)

	c.PaymentUC =
		usecase.NewPaymentUsecase(
			usecase.NewPaymentUsecaseInput{
				PaymentRepo: paymentRepo,

				CartRepo:      cartRepo,
				OrderRepo:     orderRepo,
				InventoryRepo: inventoryRepo,
				ResaleRepo:    resaleRepo,

				AuthUserGetter: authUserEmailGetter,
				MailSender:     c.OrderMailer,
				MailFrom:       c.OrderMailFrom,
			},
		)

	c.OrderUC =
		usecase.NewOrderUsecase(
			orderRepo,
			listRepoFS,
			inventoryRepo,
			resaleRepo,
			paymentMethodRepo,
			shippingAddressRepo,
			c.ShippingQuoteUC,
		)

	c.InquiryUC =
		usecase.NewInquiryUsecase(
			inquiryRepo,
			inquiryReplyRepo,
			c.InquiryMailer,
			c.OrderMailFrom,
			c.InquiryMailTo,
			avatarRepo,
			authUserEmailGetter,
		)

	{
		paymentFlowUC, configured, err :=
			buildPaymentFlowUsecase(
				infra,
				c.PaymentUC,
				orderRepo,
			)
		if err != nil {
			return nil, err
		}

		c.PaymentFlowUC = paymentFlowUC
		_ = configured
	}

	c.InventoryUC =
		usecase.NewInventoryUsecase(
			inventoryRepo,
		)

	{
		c.NameResolver =
			appresolver.NewNameResolver(
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
		brandsCol :=
			infra.BrandsCollection

		avatarsCol :=
			infra.AvatarsCollection

		brandReader :=
			sharedfs.NewBrandWalletAddressReaderFS(
				fsClient,
				brandsCol,
			)

		avatarReader :=
			sharedfs.NewAvatarWalletAddressReaderFS(
				fsClient,
				avatarsCol,
			)

		c.OwnerResolveQ =
			sharedquery.NewOwnerResolveQuery(
				avatarReader,
				brandReader,
				avatarRepo,
				brandRepo,
			)
	}

	{
		c.BrandQ =
			mallquery.NewBrandQuery(
				brandRepo,
				companyRepo,
				productBlueprintRepoFS,
				inventoryRepo,
				listRepoFS,
			)

		c.CatalogQ =
			mallquery.NewCatalogQuery(
				listRepoFS,
				inventoryRepo,
				productBlueprintRepoFS,
				modelRepoFS,
				listImageRecordRepo,
				tokenBlueprintRepo,
				productBlueprintReviewRepo,
				c.NameResolver,
			)

		c.CartQ =
			mallquery.NewCartQuery(
				cartRepo,
				listRepoFS,
				listImageRecordRepo,
				inventoryRepo,
				productBlueprintRepoFS,
				resaleRepo,
				resaleImageRepo,
				mallDisplayResolver,
				mallquery.WithCartQueryBrandRepo(
					brandRepo,
				),
			)

		tokenReader :=
			outfs.NewTokenReaderFS(
				fsClient,
			)

		solanaTransferReader :=
			solana.NewTokenTransferReaderSolana(
				"",
			)

		previewTransferReader :=
			outsolana.NewPreviewTransferReader(
				solanaTransferReader,
			)

		c.PreviewQ =
			mallquery.NewPreviewQuery(
				productRepo,
				productBlueprintRepoFS,
				orderTransferItemRepo,
				c.NameResolver,
				tokenReader,
				tokenBlueprintRepo,
				c.OwnerResolveQ,
				brandRepo,
				avatarRepo,
				previewTransferReader,
			)

		c.OrderQ =
			mallquery.NewOrderQuery(
				avatarRepo,
				cartRepo,
				shippingAddressRepo,
				paymentMethodRepo,
				productBlueprintRepoFS,
				resaleRepo,
				resaleImageRepo,
				c.NameResolver,
			)

		c.HistoryQ =
			mallquery.NewHistoryQuery(
				inventoryRepo,
				mallDisplayResolver,
			)
	}

	{
		scanVerifier :=
			buildScanVerifier(
				c.PreviewQ,
			)
		if scanVerifier == nil {
			return nil,
				errors.New(
					"di.mall: scan verifier is nil",
				)
		}

		var orderRepoForTransfer usecase.OrderRepoForTransfer = orderTransferItemRepo

		var tokenResolver usecase.TokenResolver = mallfs.NewTokenResolverFS(
			fsClient,
			"tokens",
		)

		var tokenOwnerUpdater usecase.TokenOwnerUpdater = outfs.NewTokenOwnerUpdaterFS(
			fsClient,
		)

		var transferRepo transferdom.RepositoryPort = outfs.NewTransferRepositoryFS(
			fsClient,
		)

		var walletResolver usecase.BrandWalletResolver = outfs.NewWalletResolverRepoFS(
			brandRepo,
			walletRepo,
		)

		avatarWalletResolver, ok :=
			any(walletResolver).(usecase.AvatarWalletResolver)
		if !ok {
			return nil,
				errors.New(
					"di.mall: wallet resolver does not implement AvatarWalletResolver",
				)
		}

		var walletTransferUpdate usecase.AvatarWalletItemTransferUpdater = walletRepo

		var walletSync usecase.AvatarWalletSyncer = c.WalletUC

		var executor usecase.TokenTransferExecutor = solana.NewTokenTransferExecutorSolana(
			"",
		)

		transferExecutionUC :=
			usecase.NewTokenTransferExecutionUsecase(
				tokenOwnerUpdater,
				walletTransferUpdate,
				walletSync,
				transferRepo,
				executor,
				nil,
			)

		c.TransferUC =
			usecase.NewTransferUsecase(
				scanVerifier,
				orderRepoForTransfer,
				tokenResolver,
				walletResolver,
				avatarWalletResolver,
				brandRepo,
				avatarRepo,
				transferExecutionUC,
				c.InventoryUC,
			).
				WithResaleTransferDependencies(
					resaleRepo,
				)

		c.ShareTransferUC =
			usecase.NewShareTransferUsecase(
				tokenResolver,
				avatarWalletResolver,
				transferExecutionUC,
			)
	}

	return c, nil
}
