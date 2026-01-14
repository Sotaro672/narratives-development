// backend/internal/platform/di/mall/container.go
package mall

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"

	"cloud.google.com/go/firestore"
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

	// ✅ Solana wallet service
	solanainfra "narratives/internal/infra/solana"

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

	// ✅ 方針A: any をやめて具体型で持つ（ResolveAvatarIDByUID を満たすことをコンパイル時に保証）
	OrderQ *mallquery.OrderQuery

	// ✅ NEW: purchased orders (paid=true & items.transfer=false) -> (modelId, tokenBlueprintId) list
	OrderPurchasedQ *mallquery.OrderPurchasedQuery

	// ✅ NEW: verify scanned pair (preview) matches purchased pair
	OrderScanVerifyQ *mallquery.OrderScanVerifyQuery

	// ✅ Shared query: walletAddress(toAddress) -> brandId / avatarId
	OwnerResolveQ *sharedquery.OwnerResolveQuery

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
	// ✅ paid と同タイミングで cart clear / inventory reserve を行う（best-effort）
	c.PaymentUC = usecase.NewPaymentUsecase(paymentRepo).
		WithInvoiceRepoForPayment(invoiceRepo).
		WithCartRepoForPayment(cartRepo).
		WithOrderRepoForPayment(orderRepo).
		WithInventoryRepoForPayment(inventoryRepo)

	c.InvoiceUC = usecase.NewInvoiceUsecase(invoiceRepo)
	c.OrderUC = usecase.NewOrderUsecase(orderRepo)

	// ✅ Case A: PaymentFlowUsecase（payment起票 + 必要なら webhook trigger）
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

		// ✅ ProductBlueprint repo（CatalogQuery でも使う）
		pbRepoFS := outfs.NewProductBlueprintRepositoryFS(fsClient)

		// ✅ outfs.NewModelRepositoryFS は GetModelVariationByID を持つ想定（PreviewQuery の ModelVariationReader を満たす）
		modelRepo := outfs.NewModelRepositoryFS(fsClient)

		c.CatalogQ = mallquery.NewCatalogQuery(listRepoFS, invRepo, pbRepoFS, modelRepo)

		c.CartQ = mallquery.NewCartQuery(fsClient)

		// ✅ PreviewQuery 用 ProductBlueprintReader を用意（pbRepoFS は GetIDByModelID を持たないため adapter が必要）
		pbReader := previewProductBlueprintReaderFS{
			fs: fsClient,
			pb: pbRepoFS,
		}

		// ✅ PreviewQuery 本体
		c.PreviewQ = mallquery.NewPreviewQuery(
			previewProductReaderFS{fs: fsClient},
			modelRepo,
			pbReader,
		)

		// ✅ tokens/{productId} を preview で返したいので TokenRepo を注入（optional）
		// ★修正点: previewTokenReaderFS ではなく TokenReaderFS を注入する（rawログや tokenBlueprintId 読み取りが反映される）
		if c.PreviewQ != nil {
			c.PreviewQ.TokenRepo = outfs.NewTokenReaderFS(fsClient)
		}

		// ✅ 方針A: any ではなく *mallquery.OrderQuery を保持
		c.OrderQ = mallquery.NewOrderQuery(fsClient)

		// ✅ NEW: OrderPurchasedQuery
		// - avatarId を受け、paid=true & items.transfer=false の (modelId, tokenBlueprintId) を返す Query
		c.OrderPurchasedQ = mallquery.NewOrderPurchasedQuery(fsClient)

		// ✅ NEW: OrderScanVerifyQuery (PurchasedQ + PreviewQ を合成)
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
	// ✅ Shared Query: OwnerResolve (walletAddress/toAddress -> brandId / avatarId)
	// --------------------------------------------------------
	{
		brandsCol := strings.TrimSpace(os.Getenv("BRANDS_COLLECTION"))
		if brandsCol == "" {
			brandsCol = defaultBrandsCollection
		}
		avatarsCol := strings.TrimSpace(os.Getenv("AVATARS_COLLECTION"))
		if avatarsCol == "" {
			avatarsCol = defaultAvatarsCollection
		}

		// ✅ sharedquery の interface は "Find..." メソッドを要求している
		brandReader := brandWalletAddressReaderFS{fs: fsClient, col: brandsCol}
		avatarReader := avatarWalletAddressReaderFS{fs: fsClient, col: avatarsCol}

		// ✅ NewOwnerResolveQuery は (avatarReader, brandReader) の順で受け取る定義
		// （= 第1引数: AvatarWalletAddressReader / 第2引数: BrandWalletAddressReader）
		c.OwnerResolveQ = sharedquery.NewOwnerResolveQuery(avatarReader, brandReader)
	}

	log.Printf(
		"[di.mall] container built (firestore=%t gcs=%t firebaseAuth=%t avatarUC=%t avatarWalletSvc=%t cartUC=%t cartRepo=%t paymentUC=%t paymentFlowUC=%t invoiceUC=%t meAvatarRepo=%t inventoryUC=%t tokenBlueprintRepo=%t tokenIconResolver=%t selfBaseURL=%t previewQ=%t ownerResolveQ=%t orderPurchasedQ=%t orderScanVerifyQ=%t)",
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
		c.PreviewQ != nil,
		c.OwnerResolveQ != nil,
		c.OrderPurchasedQ != nil,
		c.OrderScanVerifyQ != nil,
	)

	return c, nil
}

// ------------------------------------------------------------
// PreviewQuery: ProductReader adapter (Firestore -> domain.Product)
// ------------------------------------------------------------

// previewProductReaderFS implements mallquery.ProductReader
// by reading a product document from Firestore.
//
// NOTE: collection 名は "products" を前提にしています。
// もし実際の保存先が異なる場合は、この1箇所だけ直せば PreviewQuery は動き続けます。
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

	// Firestore doc id を優先して上書き
	p.ID = doc.Ref.ID
	return p, nil
}

// ------------------------------------------------------------
// PreviewQuery: ProductBlueprintReader adapter
// - pbRepoFS は GetIDByModelID を持たないため、ここで補う
// ------------------------------------------------------------

// previewProductBlueprintReaderFS implements mallquery.ProductBlueprintReader.
type previewProductBlueprintReaderFS struct {
	fs *firestore.Client
	pb interface {
		GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error)
		GetPatchByID(ctx context.Context, id string) (pbdom.Patch, error)
	}
}

// GetIDByModelID resolves productBlueprintId from modelId.
// NOTE: "models/{modelId}" に productBlueprintId がある前提です。
//
//	フィールド名は productBlueprintId / productBlueprintID / product_blueprint_id を許容します。
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
		// model が無い場合は上位で "resolved productBlueprintId is empty" に落ちるのでもOK
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
// PreviewQuery: TokenReader adapter (Firestore tokens/{productId})
// ------------------------------------------------------------

type previewTokenReaderFS struct {
	fs *firestore.Client
}

// GetByProductID reads tokens/{productId} and maps to mallquery.TokenInfo.
// - token が存在しない（未mint）なら (nil, nil) を返す
func (r previewTokenReaderFS) GetByProductID(ctx context.Context, productID string) (*mallquery.TokenInfo, error) {
	if r.fs == nil {
		return nil, mallquery.ErrPreviewQueryNotConfigured
	}
	pid := strings.TrimSpace(productID)
	if pid == "" {
		return nil, mallquery.ErrInvalidProductID
	}

	snap, err := r.fs.Collection("tokens").Doc(pid).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}

	raw := snap.Data()
	if raw == nil {
		return nil, nil
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

	// ✅ TokenInfo から TokenBlueprintID は削除済み前提
	// ✅ A案: tokens 側に toAddress / metadataUri をキャッシュしている前提
	out := &mallquery.TokenInfo{
		ProductID: pid,
		BrandID:   getStr("brandId", "brandID"),

		MintAddress: getStr("mintAddress", "mint_address"),

		ToAddress:   getStr("toAddress", "to_address"),
		MetadataURI: getStr("metadataUri", "metadataURI", "metadata_uri"),

		OnChainTxSignature: getStr(
			"onChainTxSignature",
			"onchainTxSignature",
			"txSignature",
			"signature",
		),
	}

	return out, nil
}

// ------------------------------------------------------------
// ✅ SharedQuery OwnerResolve: walletAddress readers (Firestore)
// ------------------------------------------------------------

// brandWalletAddressReaderFS implements sharedquery.BrandWalletAddressReader.
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

// avatarWalletAddressReaderFS implements sharedquery.AvatarWalletAddressReader.
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
