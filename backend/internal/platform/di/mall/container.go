// backend/internal/platform/di/mall/container.go
package mall

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"

	"cloud.google.com/go/firestore"
	firebaseauth "firebase.google.com/go/v4/auth"

	mallhttp "narratives/internal/adapters/in/http/mall"
	mallhandler "narratives/internal/adapters/in/http/mall/handler"
	mallwebhook "narratives/internal/adapters/in/http/mall/webhook"
	"narratives/internal/adapters/in/http/middleware"
	outfs "narratives/internal/adapters/out/firestore"
	mallquery "narratives/internal/application/query/mall"
	appresolver "narratives/internal/application/resolver"
	usecase "narratives/internal/application/usecase"
	invdom "narratives/internal/domain/inventory"
	ldom "narratives/internal/domain/list"
	pbdom "narratives/internal/domain/productBlueprint"

	consoleDI "narratives/internal/platform/di/console"
)

const (
	StripeWebhookPath = "/mall/webhooks/stripe"
)

// ------------------------------------------------------------
// Hit tracing
// ------------------------------------------------------------

type hit struct {
	OK   bool
	From string
	Name string
}

func (h hit) String() string {
	if !h.OK {
		return "(nil)"
	}
	from := strings.TrimSpace(h.From)
	name := strings.TrimSpace(h.Name)
	if from == "" && name == "" {
		return "(ok)"
	}
	if from == "" {
		return name
	}
	if name == "" {
		return from
	}
	return from + ":" + name
}

// MallDeps (kept for compatibility with existing wiring)
type MallDeps struct {
	List             http.Handler
	Inventory        http.Handler
	ProductBlueprint http.Handler
	Model            http.Handler
	Catalog          http.Handler
	TokenBlueprint   http.Handler

	Company http.Handler
	Brand   http.Handler

	SignIn http.Handler

	User            http.Handler
	ShippingAddress http.Handler
	BillingAddress  http.Handler
	Avatar          http.Handler

	AvatarState http.Handler
	Wallet      http.Handler
	Post        http.Handler

	Cart    http.Handler
	Preview http.Handler

	Payment http.Handler
	Order   http.Handler
}

func NewMallDeps(
	listUC *usecase.ListUsecase,
	invUC *usecase.InventoryUsecase,
	pbUC *usecase.ProductBlueprintUsecase,
	modelUC *usecase.ModelUsecase,
	tokenBlueprintUC *usecase.TokenBlueprintUsecase,
	catalogQ *mallquery.CatalogQuery,
) MallDeps {
	return NewMallDepsWithNameResolver(
		listUC,
		invUC,
		pbUC,
		modelUC,
		tokenBlueprintUC,
		nil,
		catalogQ,
	)
}

func NewMallDepsWithNameResolver(
	listUC *usecase.ListUsecase,
	invUC *usecase.InventoryUsecase,
	pbUC *usecase.ProductBlueprintUsecase,
	modelUC *usecase.ModelUsecase,
	tokenBlueprintUC *usecase.TokenBlueprintUsecase,
	nameResolver *appresolver.NameResolver,
	catalogQ *mallquery.CatalogQuery,
) MallDeps {
	return NewMallDepsWithNameResolverAndOrgHandlers(
		listUC,
		invUC,
		pbUC,
		modelUC,
		tokenBlueprintUC,
		nil,
		nil,
		nameResolver,
		catalogQ,
	)
}

func NewMallDepsWithNameResolverAndOrgHandlers(
	listUC *usecase.ListUsecase,
	invUC *usecase.InventoryUsecase,
	pbUC *usecase.ProductBlueprintUsecase,
	modelUC *usecase.ModelUsecase,
	tokenBlueprintUC *usecase.TokenBlueprintUsecase,

	companyUC *usecase.CompanyUsecase,
	brandUC *usecase.BrandUsecase,

	nameResolver *appresolver.NameResolver,
	catalogQ *mallquery.CatalogQuery,
) MallDeps {
	if catalogQ != nil && nameResolver != nil && catalogQ.NameResolver == nil {
		catalogQ.NameResolver = nameResolver
	}

	var listHandler http.Handler
	var invHandler http.Handler
	var pbHandler http.Handler
	var modelHandler http.Handler
	var catalogHandler http.Handler
	var tokenBlueprintHandler http.Handler
	var companyHandler http.Handler
	var brandHandler http.Handler

	if listUC != nil {
		listHandler = mallhandler.NewMallListHandler(listUC)
	}
	if invUC != nil {
		invHandler = mallhandler.NewMallInventoryHandler(invUC)
	}
	if pbUC != nil {
		// NOTE: 既存のハンドラ名が NewSNS* でも、mall 側で利用しているためここでは改名しない。
		pbHandler = mallhandler.NewSNSProductBlueprintHandler(pbUC)
		if nameResolver != nil {
			setOptionalResolverField(pbHandler, "BrandNameResolver", nameResolver)
			setOptionalResolverField(pbHandler, "CompanyNameResolver", nameResolver)
			setOptionalResolverField(pbHandler, "NameResolver", nameResolver)
		}
	}
	if modelUC != nil {
		modelHandler = mallhandler.NewMallModelHandler(modelUC)
	}
	if catalogQ != nil {
		// NOTE: 既存のハンドラ名が NewSNS* でも、mall 側で利用しているためここでは改名しない。
		catalogHandler = mallhandler.NewSNSCatalogHandler(catalogQ)
	}
	if companyUC != nil {
		companyHandler = mallhandler.NewSNSCompanyHandler(companyUC)
	}
	if brandUC != nil {
		brandHandler = mallhandler.NewSNSBrandHandler(brandUC)
	}

	if tokenBlueprintUC != nil {
		tokenBlueprintHandler = mallhandler.NewMallTokenBlueprintHandler(tokenBlueprintUC)
		if nameResolver != nil {
			setOptionalResolverField(tokenBlueprintHandler, "BrandNameResolver", nameResolver)
			setOptionalResolverField(tokenBlueprintHandler, "CompanyNameResolver", nameResolver)
			setOptionalResolverField(tokenBlueprintHandler, "NameResolver", nameResolver)
		}
		imgResolver := appresolver.NewImageURLResolver("")
		setOptionalResolverField(tokenBlueprintHandler, "ImageResolver", imgResolver)
		setOptionalResolverField(tokenBlueprintHandler, "ImageURLResolver", imgResolver)
		setOptionalResolverField(tokenBlueprintHandler, "IconURLResolver", imgResolver)
	}

	return MallDeps{
		List:             listHandler,
		Inventory:        invHandler,
		ProductBlueprint: pbHandler,
		Model:            modelHandler,
		Catalog:          catalogHandler,
		TokenBlueprint:   tokenBlueprintHandler,

		Company: companyHandler,
		Brand:   brandHandler,
	}
}

// RegisterMallFromContainer registers mall routes onto mux using *console.Container.
func RegisterMallFromContainer(mux *http.ServeMux, cont *consoleDI.Container) {
	if mux == nil || cont == nil {
		return
	}

	depsAny := any(cont.RouterDeps())

	var (
		hitSignIn   hit
		hitShip     hit
		hitPay      hit
		hitOrder    hit
		hitWallet   hit
		hitNameRes  hit
		hitCatalogQ hit
		hitCartQ    hit
		hitPrevQ    hit
		hitListRepo hit

		hitCart    hit
		hitPreview hit

		hitStripeWH hit
	)

	// --------------------------------------------------------
	// Core clients
	// --------------------------------------------------------
	fsClient := getFirestoreClientStrict(cont, depsAny)

	// --------------------------------------------------------
	// Resolver
	// --------------------------------------------------------
	nameResolver := getFieldPtr[*appresolver.NameResolver](depsAny, "NameResolver")
	hitNameRes = hit{OK: nameResolver != nil, From: "RouterDeps.field/Container.field", Name: "NameResolver"}

	// --------------------------------------------------------
	// Queries (mall/container.go owns them; no Container methods)
	// --------------------------------------------------------
	var catalogQ *mallquery.CatalogQuery
	var cartQ *mallquery.CartQuery
	var previewQ *mallquery.PreviewQuery
	var orderQAny any

	if fsClient != nil {
		listRepoFS := outfs.NewListRepositoryFS(fsClient)

		// ✅ Adapter: mallquery.InventoryRepository expects inventory.Mint return types
		baseInvRepo := outfs.NewInventoryRepositoryFS(fsClient)
		invRepo := &catalogInventoryRepoAdapter{base: baseInvRepo}

		// ✅ Adapter: mallquery.ProductBlueprintRepository expects value return (not *T)
		basePBRepo := outfs.NewProductBlueprintRepositoryFS(fsClient)
		pbRepo := &catalogProductBlueprintRepoAdapter{base: basePBRepo}

		modelRepo := outfs.NewModelRepositoryFS(fsClient)

		catalogQ = mallquery.NewCatalogQuery(
			listRepoFS,
			invRepo,
			pbRepo,
			modelRepo,
		)
		if catalogQ != nil && nameResolver != nil && catalogQ.NameResolver == nil {
			catalogQ.NameResolver = nameResolver
		}

		cartQ = mallquery.NewCartQuery(fsClient)
		if cartQ != nil && nameResolver != nil && cartQ.Resolver == nil {
			cartQ.Resolver = nameResolver
		}

		previewQ = mallquery.NewPreviewQuery(fsClient)
		if previewQ != nil && nameResolver != nil && previewQ.Resolver == nil {
			previewQ.Resolver = nameResolver
		}

		orderQAny = mallquery.NewOrderQuery(fsClient)
	}

	hitCatalogQ = hit{OK: catalogQ != nil, From: "constructed", Name: "mallquery.NewCatalogQuery"}
	hitCartQ = hit{OK: cartQ != nil, From: "constructed", Name: "mallquery.NewCartQuery"}
	hitPrevQ = hit{OK: previewQ != nil, From: "constructed", Name: "mallquery.NewPreviewQuery"}

	// Prefer existing order query if any
	if orderQAny == nil {
		orderQAny = getOrderQueryBestEffort(cont, depsAny)
	}

	// --------------------------------------------------------
	// Direct handlers (mall/container.go owns them; no Container methods)
	// --------------------------------------------------------
	signInH := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	hitSignIn = hit{OK: signInH != nil, From: "constructed", Name: "sign-in 204"}

	// shipping address
	var shipH http.Handler
	{
		shipUC := getFieldPtr[*usecase.ShippingAddressUsecase](depsAny, "ShippingAddressUC", "ShippingAddressUsecase")
		if shipUC != nil {
			shipH = mallhandler.NewShippingAddressHandler(shipUC)
		}
	}
	hitShip = hit{OK: shipH != nil, From: "constructed", Name: "mallhandler.NewShippingAddressHandler"}

	// --------------------------------------------------------
	// listRepo (for CartQuery/List name join etc.)
	// --------------------------------------------------------
	var listRepo ldom.Repository
	{
		if fsClient != nil {
			listRepo = outfs.NewListRepositoryFS(fsClient)
		}
		hitListRepo = hit{OK: listRepo != nil, From: "constructed", Name: "outfs.NewListRepositoryFS"}
	}

	if cartQ != nil && listRepo != nil && cartQ.ListRepo == nil {
		cartQ.ListRepo = listRepo
	}
	if previewQ != nil && listRepo != nil {
		setOptionalResolverField(previewQ, "ListRepo", listRepo)
		setOptionalResolverField(previewQ, "ListRepository", listRepo)
	}

	// --------------------------------------------------------
	// Usecases (from RouterDeps)
	// --------------------------------------------------------
	listUC := getFieldPtr[*usecase.ListUsecase](depsAny, "ListUC", "ListUsecase")
	invUC := getFieldPtr[*usecase.InventoryUsecase](depsAny, "InventoryUC", "InventoryUsecase")
	pbUC := getFieldPtr[*usecase.ProductBlueprintUsecase](depsAny, "ProductBlueprintUC", "ProductBlueprintUsecase")
	modelUC := getFieldPtr[*usecase.ModelUsecase](depsAny, "ModelUC", "ModelUsecase")
	tokenBlueprintUC := getFieldPtr[*usecase.TokenBlueprintUsecase](depsAny, "TokenBlueprintUC", "TokenBlueprintUsecase")

	companyUC := getFieldPtr[*usecase.CompanyUsecase](depsAny, "CompanyUC", "CompanyUsecase")
	brandUC := getFieldPtr[*usecase.BrandUsecase](depsAny, "BrandUC", "BrandUsecase")

	orderUC := getFieldPtr[*usecase.OrderUsecase](depsAny, "OrderUC", "OrderUsecase")
	invoiceUC := getFieldPtr[*usecase.InvoiceUsecase](depsAny, "InvoiceUC", "InvoiceUsecase")
	paymentUC := getFieldPtr[*usecase.PaymentUsecase](depsAny, "PaymentUC", "PaymentUsecase")
	walletUC := getFieldPtr[*usecase.WalletUsecase](depsAny, "WalletUC", "WalletUsecase")

	cartUC := getFieldPtr[*usecase.CartUsecase](depsAny, "CartUC", "CartUsecase")

	// --------------------------------------------------------
	// Base deps (REST handlers)
	// --------------------------------------------------------
	mallDeps := NewMallDepsWithNameResolverAndOrgHandlers(
		listUC, invUC, pbUC, modelUC, tokenBlueprintUC,
		companyUC, brandUC, nameResolver, catalogQ,
	)

	mallDeps.SignIn = signInH
	mallDeps.ShippingAddress = shipH

	// User
	{
		userUC := getFieldPtr[*usecase.UserUsecase](depsAny, "UserUC", "UserUsecase")
		if userUC != nil {
			mallDeps.User = mallhandler.NewUserHandler(userUC)
		}
	}

	// Billing
	{
		billUC := getFieldPtr[*usecase.BillingAddressUsecase](depsAny, "BillingAddressUC", "BillingAddressUsecase")
		if billUC != nil {
			mallDeps.BillingAddress = mallhandler.NewBillingAddressHandler(billUC)
		}
	}

	// Avatar
	{
		avatarUC := getFieldPtr[*usecase.AvatarUsecase](depsAny, "AvatarUC", "AvatarUsecase")
		if avatarUC != nil {
			mallDeps.Avatar = mallhandler.NewAvatarHandler(avatarUC)
		}
	}

	// Wallet
	if walletUC != nil {
		mallDeps.Wallet = mallhandler.NewWalletHandler(walletUC)
		hitWallet = hit{OK: mallDeps.Wallet != nil, From: "constructed", Name: "mallhandler.NewWalletHandler"}
	} else {
		hitWallet = hit{OK: false, From: "RouterDeps.field", Name: "WalletUC"}
	}

	// Payment (with order query)
	if paymentUC != nil {
		mallDeps.Payment = mallhandler.NewPaymentHandlerWithOrderQuery(paymentUC, orderQAny)
		hitPay = hit{OK: mallDeps.Payment != nil, From: "constructed", Name: "mallhandler.NewPaymentHandlerWithOrderQuery"}
	} else {
		hitPay = hit{OK: false, From: "RouterDeps.field", Name: "PaymentUC"}
	}

	// Order
	if orderUC != nil {
		mallDeps.Order = mallhandler.NewOrderHandler(orderUC)
		hitOrder = hit{OK: mallDeps.Order != nil, From: "constructed", Name: "mallhandler.NewOrderHandler"}
	} else {
		hitOrder = hit{OK: false, From: "RouterDeps.field", Name: "OrderUC"}
	}

	// Cart/Preview
	if cartQ != nil {
		qh := mallhandler.NewCartQueryHandler(cartQ)

		var core http.Handler
		if cartUC != nil {
			core = mallhandler.NewCartHandlerWithQueries(cartUC, cartQ, previewQ)
			hitPreview = hit{OK: core != nil, From: "constructed", Name: "CartHandlerWithQueries (as preview)"}
		}

		mallDeps.Cart = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := r.URL.Path

			if r.Method == http.MethodGet {
				if p == "/mall/cart" || p == "/mall/cart/" || strings.HasPrefix(p, "/mall/cart/query") {
					qh.ServeHTTP(w, r)
					return
				}
			}

			if core != nil {
				core.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		})

		hitCart = hit{OK: mallDeps.Cart != nil, From: "constructed", Name: "wrapped(GET->CartQueryHandler, else->CartHandler/405)"}

		if core != nil {
			mallDeps.Preview = core
		}
	} else if cartUC != nil {
		core := mallhandler.NewCartHandlerWithQueries(cartUC, nil, previewQ)
		mallDeps.Cart = core
		mallDeps.Preview = core
		hitCart = hit{OK: true, From: "constructed", Name: "CartHandlerWithQueries"}
		hitPreview = hit{OK: true, From: "constructed", Name: "CartHandlerWithQueries (as preview)"}
	}

	// user_auth
	{
		fbClient := getFirebaseAuthClientStrict(cont, depsAny)
		userAuth := newUserAuthMiddlewareBestEffort(fbClient)
		if userAuth == nil {
			log.Printf("[mall_container] WARN: user_auth middleware is not available (firebase auth client missing). protected routes may 401.")
		} else {
			wrap := func(h http.Handler) http.Handler {
				if h == nil {
					return nil
				}
				return userAuth.Handler(h)
			}

			mallDeps.User = wrap(mallDeps.User)
			mallDeps.ShippingAddress = wrap(mallDeps.ShippingAddress)
			mallDeps.BillingAddress = wrap(mallDeps.BillingAddress)
			mallDeps.Avatar = wrap(mallDeps.Avatar)
			mallDeps.Wallet = wrap(mallDeps.Wallet)

			mallDeps.Cart = wrap(mallDeps.Cart)
			mallDeps.Preview = wrap(mallDeps.Preview)
			mallDeps.Payment = wrap(mallDeps.Payment)
			mallDeps.Order = wrap(mallDeps.Order)
		}
	}

	// routes
	mallhttp.Register(mux, mallhttp.Deps{
		List:             mallDeps.List,
		Inventory:        mallDeps.Inventory,
		ProductBlueprint: mallDeps.ProductBlueprint,
		Model:            mallDeps.Model,
		Catalog:          mallDeps.Catalog,
		TokenBlueprint:   mallDeps.TokenBlueprint,

		Company: mallDeps.Company,
		Brand:   mallDeps.Brand,

		SignIn: mallDeps.SignIn,

		User:            mallDeps.User,
		ShippingAddress: mallDeps.ShippingAddress,
		BillingAddress:  mallDeps.BillingAddress,
		Avatar:          mallDeps.Avatar,
		AvatarState:     mallDeps.AvatarState,
		Wallet:          mallDeps.Wallet,

		Cart:    mallDeps.Cart,
		Post:    mallDeps.Post,
		Payment: mallDeps.Payment,
		Preview: mallDeps.Preview,
		Order:   mallDeps.Order,
	})

	// logs
	log.Printf("[mall_container] inject hits "+
		"signIn=%s ship=%s payment=%s order=%s wallet=%s cart=%s preview=%s "+
		"nameResolver=%s catalogQ=%s cartQ=%s previewQ=%s listRepo=%s orderQ=%t",
		hitSignIn.String(),
		hitShip.String(),
		hitPay.String(),
		hitOrder.String(),
		hitWallet.String(),
		hitCart.String(),
		hitPreview.String(),
		hitNameRes.String(),
		hitCatalogQ.String(),
		hitCartQ.String(),
		hitPrevQ.String(),
		hitListRepo.String(),
		orderQAny != nil,
	)

	// webhook (no user_auth)
	if invoiceUC != nil && paymentUC != nil {
		stripeWH := mallwebhook.NewStripeWebhookHandler(invoiceUC, paymentUC)
		hitStripeWH = hit{OK: stripeWH != nil, From: "constructed", Name: "mallwebhook.NewStripeWebhookHandler"}

		mux.Handle(StripeWebhookPath, stripeWH)
		mux.Handle(StripeWebhookPath+"/", stripeWH)

		log.Printf("[mall_container] webhook registered stripe=%s paths=%s,%s",
			hitStripeWH.String(), StripeWebhookPath, StripeWebhookPath+"/",
		)
	} else {
		hitStripeWH = hit{OK: false, From: "RouterDeps.field", Name: "InvoiceUC/PaymentUC"}
		log.Printf("[mall_container] webhook NOT registered stripe=%s (missing InvoiceUC or PaymentUC)", hitStripeWH.String())
	}
}

// ------------------------------------------------------------
// Repo adapters for mallquery.NewCatalogQuery (signature fixes)
// ------------------------------------------------------------

type catalogInventoryRepoAdapter struct {
	base any // usually *outfs.InventoryRepositoryFS
}

// mallquery.InventoryRepository wants inventory.Mint (NOT DTO)
func (a *catalogInventoryRepoAdapter) GetByID(ctx context.Context, id string) (invdom.Mint, error) {
	if a == nil || a.base == nil {
		return invdom.Mint{}, errors.New("catalogInventoryRepoAdapter: base is nil")
	}

	v, err := callRepo(a.base,
		[]string{"GetByID", "GetById", "GetMintByID", "GetMintById"},
		ctx,
		strings.TrimSpace(id),
	)
	if err != nil {
		return invdom.Mint{}, err
	}
	if v == nil {
		return invdom.Mint{}, errors.New("mint is nil")
	}

	switch m := v.(type) {
	case invdom.Mint:
		return m, nil
	case *invdom.Mint:
		if m == nil {
			return invdom.Mint{}, errors.New("mint is nil")
		}
		return *m, nil
	default:
		rv := reflect.ValueOf(v)
		if rv.IsValid() && rv.Kind() == reflect.Pointer && !rv.IsNil() {
			if x, ok := rv.Interface().(*invdom.Mint); ok && x != nil {
				return *x, nil
			}
		}
		return invdom.Mint{}, fmt.Errorf("unexpected mint type: %T", v)
	}
}

// If mallquery.InventoryRepository requires this, implement it here.
func (a *catalogInventoryRepoAdapter) GetByProductAndTokenBlueprintID(ctx context.Context, productBlueprintID, tokenBlueprintID string) (invdom.Mint, error) {
	if a == nil || a.base == nil {
		return invdom.Mint{}, errors.New("catalogInventoryRepoAdapter: base is nil")
	}

	pbID := strings.TrimSpace(productBlueprintID)
	tbID := strings.TrimSpace(tokenBlueprintID)

	v, err := callRepo(a.base,
		[]string{
			"GetByProductAndTokenBlueprintID",
			"GetByProductAndTokenBlueprintId",
			"GetMintByProductAndTokenBlueprintID",
			"GetMintByProductAndTokenBlueprintId",
		},
		ctx,
		pbID,
		tbID,
	)
	if err != nil {
		return invdom.Mint{}, err
	}
	if v == nil {
		return invdom.Mint{}, errors.New("mint is nil")
	}

	switch m := v.(type) {
	case invdom.Mint:
		return m, nil
	case *invdom.Mint:
		if m == nil {
			return invdom.Mint{}, errors.New("mint is nil")
		}
		return *m, nil
	default:
		rv := reflect.ValueOf(v)
		if rv.IsValid() && rv.Kind() == reflect.Pointer && !rv.IsNil() {
			if x, ok := rv.Interface().(*invdom.Mint); ok && x != nil {
				return *x, nil
			}
		}
		return invdom.Mint{}, fmt.Errorf("unexpected mint type: %T", v)
	}
}

type catalogProductBlueprintRepoAdapter struct {
	base any // usually *outfs.ProductBlueprintRepositoryFS
}

// mallquery.ProductBlueprintRepository wants value (NOT *T)
func (a *catalogProductBlueprintRepoAdapter) GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error) {
	if a == nil || a.base == nil {
		return pbdom.ProductBlueprint{}, errors.New("catalogProductBlueprintRepoAdapter: base is nil")
	}

	v, err := callRepo(a.base,
		[]string{"GetByID", "GetById"},
		ctx,
		strings.TrimSpace(id),
	)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}
	if v == nil {
		return pbdom.ProductBlueprint{}, errors.New("productBlueprint is nil")
	}

	switch pb := v.(type) {
	case pbdom.ProductBlueprint:
		return pb, nil
	case *pbdom.ProductBlueprint:
		if pb == nil {
			return pbdom.ProductBlueprint{}, errors.New("productBlueprint is nil")
		}
		return *pb, nil
	default:
		rv := reflect.ValueOf(v)
		if rv.IsValid() && rv.Kind() == reflect.Pointer && !rv.IsNil() {
			if x, ok := rv.Interface().(*pbdom.ProductBlueprint); ok && x != nil {
				return *x, nil
			}
		}
		return pbdom.ProductBlueprint{}, fmt.Errorf("unexpected productBlueprint type: %T", v)
	}
}

// ------------------------------------------------------------
// order query getter (best-effort, type-mismatch safe)
// ------------------------------------------------------------

func getOrderQueryBestEffort(cont *consoleDI.Container, depsAny any) any {
	// 1) Container methods (reflect)
	if cont != nil {
		rv := reflect.ValueOf(cont)
		if rv.IsValid() {
			for _, name := range []string{
				"MallOrderQuery",
				"OrderQuery",
				"OrderQueryService",
			} {
				m := rv.MethodByName(name)
				if m.IsValid() && m.Type().NumIn() == 0 && m.Type().NumOut() == 1 {
					out := m.Call(nil)
					if len(out) == 1 && out[0].IsValid() && !out[0].IsNil() {
						return out[0].Interface()
					}
				}
			}
		}
	}

	// 2) RouterDeps fields (reflect)
	if depsAny != nil {
		rv := reflect.ValueOf(depsAny)
		if rv.Kind() == reflect.Interface && !rv.IsNil() {
			rv = rv.Elem()
		}
		if rv.Kind() == reflect.Pointer && !rv.IsNil() {
			rv = rv.Elem()
		}
		if rv.IsValid() && rv.Kind() == reflect.Struct {
			for _, n := range []string{
				"OrderQ",
				"MallOrderQuery",
				"OrderQuery",
				"OrderQueryService",
			} {
				f := rv.FieldByName(n)
				if !f.IsValid() || !f.CanInterface() {
					continue
				}
				v := f.Interface()
				if v != nil {
					return v
				}
			}
		}
	}

	return nil
}

// ------------------------------------------------------------
// Middleware builder
// ------------------------------------------------------------

func newUserAuthMiddlewareBestEffort(fb *firebaseauth.Client) *middleware.UserAuthMiddleware {
	if fb == nil {
		return nil
	}
	return &middleware.UserAuthMiddleware{FirebaseAuth: fb}
}

// ------------------------------------------------------------
// Reflection helpers
// ------------------------------------------------------------

func setOptionalResolverField(target any, fieldName string, value any) {
	if target == nil || value == nil || strings.TrimSpace(fieldName) == "" {
		return
	}

	rv := reflect.ValueOf(target)
	if !rv.IsValid() {
		return
	}
	if rv.Kind() == reflect.Interface && !rv.IsNil() {
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return
	}

	f := rv.FieldByName(fieldName)
	if !f.IsValid() || !f.CanSet() {
		return
	}

	vv := reflect.ValueOf(value)
	if !vv.IsValid() {
		return
	}

	if vv.Type().AssignableTo(f.Type()) {
		f.Set(vv)
		return
	}
	if f.Kind() == reflect.Interface && vv.Type().Implements(f.Type()) {
		f.Set(vv)
		return
	}
}

func getFieldPtr[T any](src any, names ...string) T {
	var zero T
	if src == nil {
		return zero
	}

	rv := reflect.ValueOf(src)
	if !rv.IsValid() {
		return zero
	}
	if rv.Kind() == reflect.Interface && !rv.IsNil() {
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return zero
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return zero
	}

	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		f := rv.FieldByName(n)
		if !f.IsValid() || !f.CanInterface() {
			continue
		}
		if v, ok := f.Interface().(T); ok {
			return v
		}
	}

	return zero
}

// ------------------------------------------------------------
// Firestore client (minimal; used for repos/queries)
// ------------------------------------------------------------

func getFirestoreClientStrict(cont *consoleDI.Container, depsAny any) *firestore.Client {
	// 1) Container fields
	if cont != nil {
		rv := reflect.ValueOf(cont)
		if rv.IsValid() && rv.Kind() == reflect.Pointer && !rv.IsNil() {
			rv = rv.Elem()
		}
		if rv.IsValid() && rv.Kind() == reflect.Struct {
			for _, n := range []string{"Firestore", "FirestoreClient", "Client"} {
				f := rv.FieldByName(n)
				if !f.IsValid() || !f.CanInterface() {
					continue
				}
				if fsClient, ok := f.Interface().(*firestore.Client); ok && fsClient != nil {
					return fsClient
				}
			}
		}
	}

	// 2) RouterDeps fields
	if depsAny != nil {
		rv := reflect.ValueOf(depsAny)
		if rv.Kind() == reflect.Interface && !rv.IsNil() {
			rv = rv.Elem()
		}
		if rv.Kind() == reflect.Pointer && !rv.IsNil() {
			rv = rv.Elem()
		}
		if rv.IsValid() && rv.Kind() == reflect.Struct {
			for _, n := range []string{"Firestore", "FirestoreClient", "Client"} {
				f := rv.FieldByName(n)
				if !f.IsValid() || !f.CanInterface() {
					continue
				}
				if fsClient, ok := f.Interface().(*firestore.Client); ok && fsClient != nil {
					return fsClient
				}
			}
		}
	}

	return nil
}

// ------------------------------------------------------------
// Firebase Auth client (best-effort; mirrors Firestore getter)
// ------------------------------------------------------------

func getFirebaseAuthClientStrict(cont *consoleDI.Container, depsAny any) *firebaseauth.Client {
	// 1) Container fields
	if cont != nil {
		rv := reflect.ValueOf(cont)
		if rv.IsValid() && rv.Kind() == reflect.Pointer && !rv.IsNil() {
			rv = rv.Elem()
		}
		if rv.IsValid() && rv.Kind() == reflect.Struct {
			for _, n := range []string{"FirebaseAuth", "FirebaseAuthClient", "Auth", "AuthClient"} {
				f := rv.FieldByName(n)
				if !f.IsValid() || !f.CanInterface() {
					continue
				}
				if c, ok := f.Interface().(*firebaseauth.Client); ok && c != nil {
					return c
				}
			}
		}
	}

	// 2) RouterDeps fields
	if depsAny != nil {
		rv := reflect.ValueOf(depsAny)
		if rv.Kind() == reflect.Interface && !rv.IsNil() {
			rv = rv.Elem()
		}
		if rv.Kind() == reflect.Pointer && !rv.IsNil() {
			rv = rv.Elem()
		}
		if rv.IsValid() && rv.Kind() == reflect.Struct {
			for _, n := range []string{"FirebaseAuth", "FirebaseAuthClient", "Auth", "AuthClient"} {
				f := rv.FieldByName(n)
				if !f.IsValid() || !f.CanInterface() {
					continue
				}
				if c, ok := f.Interface().(*firebaseauth.Client); ok && c != nil {
					return c
				}
			}
		}
	}

	return nil
}
