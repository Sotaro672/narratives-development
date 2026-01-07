// backend/internal/platform/di/mall_container.go
package di

import (
	"log"
	"net/http"
	"reflect"
	"strings"

	"cloud.google.com/go/firestore"
	firebaseauth "firebase.google.com/go/v4/auth"

	mall "narratives/internal/adapters/in/http/mall"
	mallHandler "narratives/internal/adapters/in/http/mall/handler"
	mallwebhook "narratives/internal/adapters/in/http/mall/webhook"
	"narratives/internal/adapters/in/http/middleware"
	outfs "narratives/internal/adapters/out/firestore"
	mallquery "narratives/internal/application/query/mall"
	appresolver "narratives/internal/application/resolver"
	usecase "narratives/internal/application/usecase"
	ldom "narratives/internal/domain/list"
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

// SNSDeps (kept for compatibility with existing wiring)
type SNSDeps struct {
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

func NewSNSDeps(
	listUC *usecase.ListUsecase,
	invUC *usecase.InventoryUsecase,
	pbUC *usecase.ProductBlueprintUsecase,
	modelUC *usecase.ModelUsecase,
	tokenBlueprintUC *usecase.TokenBlueprintUsecase,
	catalogQ *mallquery.SNSCatalogQuery,
) SNSDeps {
	return NewSNSDepsWithNameResolver(
		listUC,
		invUC,
		pbUC,
		modelUC,
		tokenBlueprintUC,
		nil,
		catalogQ,
	)
}

func NewSNSDepsWithNameResolver(
	listUC *usecase.ListUsecase,
	invUC *usecase.InventoryUsecase,
	pbUC *usecase.ProductBlueprintUsecase,
	modelUC *usecase.ModelUsecase,
	tokenBlueprintUC *usecase.TokenBlueprintUsecase,
	nameResolver *appresolver.NameResolver,
	catalogQ *mallquery.SNSCatalogQuery,
) SNSDeps {
	return NewSNSDepsWithNameResolverAndOrgHandlers(
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

func NewSNSDepsWithNameResolverAndOrgHandlers(
	listUC *usecase.ListUsecase,
	invUC *usecase.InventoryUsecase,
	pbUC *usecase.ProductBlueprintUsecase,
	modelUC *usecase.ModelUsecase,
	tokenBlueprintUC *usecase.TokenBlueprintUsecase,

	companyUC *usecase.CompanyUsecase,
	brandUC *usecase.BrandUsecase,

	nameResolver *appresolver.NameResolver,
	catalogQ *mallquery.SNSCatalogQuery,
) SNSDeps {
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
		listHandler = mallHandler.NewMallListHandler(listUC)
	}
	if invUC != nil {
		invHandler = mallHandler.NewMallInventoryHandler(invUC)
	}
	if pbUC != nil {
		pbHandler = mallHandler.NewSNSProductBlueprintHandler(pbUC)
		if nameResolver != nil {
			setOptionalResolverField(pbHandler, "BrandNameResolver", nameResolver)
			setOptionalResolverField(pbHandler, "CompanyNameResolver", nameResolver)
			setOptionalResolverField(pbHandler, "NameResolver", nameResolver)
		}
	}
	if modelUC != nil {
		modelHandler = mallHandler.NewMallModelHandler(modelUC)
	}
	if catalogQ != nil {
		catalogHandler = mallHandler.NewSNSCatalogHandler(catalogQ)
	}
	if companyUC != nil {
		companyHandler = mallHandler.NewSNSCompanyHandler(companyUC)
	}
	if brandUC != nil {
		brandHandler = mallHandler.NewSNSBrandHandler(brandUC)
	}

	if tokenBlueprintUC != nil {
		tokenBlueprintHandler = mallHandler.NewMallTokenBlueprintHandler(tokenBlueprintUC)
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

	return SNSDeps{
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

// RegisterMallFromContainer registers mall routes onto mux using *Container.
func RegisterMallFromContainer(mux *http.ServeMux, cont *Container) {
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
	// Queries (mall_container.go owns them; no Container methods)
	// --------------------------------------------------------
	var catalogQ *mallquery.SNSCatalogQuery
	var cartQ *mallquery.CartQuery
	var previewQ *mallquery.PreviewQuery
	var orderQAny any

	// Catalog query needs several repos; build best-effort when Firestore is available.
	if fsClient != nil {
		// These constructors exist in outfs (same as container.go wiring).
		listRepoFS := outfs.NewListRepositoryFS(fsClient)
		invRepo := outfs.NewInventoryRepositoryFS(fsClient)
		pbRepo := outfs.NewProductBlueprintRepositoryFS(fsClient)
		modelRepo := outfs.NewModelRepositoryFS(fsClient)

		// NOTE: adapter types are expected to exist in package di (they are already used in container.go).
		catalogQ = mallquery.NewSNSCatalogQuery(
			listRepoFS,
			&snsCatalogInventoryRepoAdapter{repo: invRepo},
			&snsCatalogProductBlueprintRepoAdapter{repo: pbRepo},
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

		// Payment context uses order query.
		orderQAny = mallquery.NewOrderQuery(fsClient)
	}

	hitCatalogQ = hit{OK: catalogQ != nil, From: "constructed", Name: "mallquery.NewSNSCatalogQuery"}
	hitCartQ = hit{OK: cartQ != nil, From: "constructed", Name: "mallquery.NewCartQuery"}
	hitPrevQ = hit{OK: previewQ != nil, From: "constructed", Name: "mallquery.NewPreviewQuery"}

	// If something else already provides an order query, prefer it (best-effort).
	if orderQAny == nil {
		orderQAny = getOrderQueryBestEffort(cont, depsAny)
	}

	// --------------------------------------------------------
	// Direct handlers (mall_container.go owns them; no Container methods)
	// --------------------------------------------------------
	// sign-in: currently a noop (204). Replace with real impl when needed.
	signInH := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	hitSignIn = hit{OK: signInH != nil, From: "constructed", Name: "sign-in 204"}

	// shipping address
	var shipH http.Handler
	{
		shipUC := getFieldPtr[*usecase.ShippingAddressUsecase](depsAny, "ShippingAddressUC", "ShippingAddressUsecase")
		if shipUC != nil {
			shipH = mallHandler.NewShippingAddressHandler(shipUC)
		}
	}
	hitShip = hit{OK: shipH != nil, From: "constructed", Name: "mallHandler.NewShippingAddressHandler"}

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

	// Inject to queries where applicable
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
	snsDeps := NewSNSDepsWithNameResolverAndOrgHandlers(
		listUC, invUC, pbUC, modelUC, tokenBlueprintUC,
		companyUC, brandUC, nameResolver, catalogQ,
	)

	snsDeps.SignIn = signInH
	snsDeps.ShippingAddress = shipH

	// User
	{
		userUC := getFieldPtr[*usecase.UserUsecase](depsAny, "UserUC", "UserUsecase")
		if userUC != nil {
			snsDeps.User = mallHandler.NewUserHandler(userUC)
		}
	}

	// Billing
	{
		billUC := getFieldPtr[*usecase.BillingAddressUsecase](depsAny, "BillingAddressUC", "BillingAddressUsecase")
		if billUC != nil {
			snsDeps.BillingAddress = mallHandler.NewBillingAddressHandler(billUC)
		}
	}

	// Avatar
	{
		avatarUC := getFieldPtr[*usecase.AvatarUsecase](depsAny, "AvatarUC", "AvatarUsecase")
		if avatarUC != nil {
			snsDeps.Avatar = mallHandler.NewAvatarHandler(avatarUC)
		}
	}

	// Wallet
	if walletUC != nil {
		snsDeps.Wallet = mallHandler.NewWalletHandler(walletUC)
		hitWallet = hit{OK: snsDeps.Wallet != nil, From: "constructed", Name: "mallHandler.NewWalletHandler"}
	} else {
		hitWallet = hit{OK: false, From: "RouterDeps.field", Name: "WalletUC"}
	}

	// Payment (with order query)
	if paymentUC != nil {
		snsDeps.Payment = mallHandler.NewPaymentHandlerWithOrderQuery(paymentUC, orderQAny)
		hitPay = hit{OK: snsDeps.Payment != nil, From: "constructed", Name: "mallHandler.NewPaymentHandlerWithOrderQuery"}
	} else {
		hitPay = hit{OK: false, From: "RouterDeps.field", Name: "PaymentUC"}
	}

	// Order
	if orderUC != nil {
		snsDeps.Order = mallHandler.NewOrderHandler(orderUC)
		hitOrder = hit{OK: snsDeps.Order != nil, From: "constructed", Name: "mallHandler.NewOrderHandler"}
	} else {
		hitOrder = hit{OK: false, From: "RouterDeps.field", Name: "OrderUC"}
	}

	// Cart/Preview
	if cartQ != nil {
		qh := mallHandler.NewCartQueryHandler(cartQ)

		var core http.Handler
		if cartUC != nil {
			core = mallHandler.NewCartHandlerWithQueries(cartUC, cartQ, previewQ)
			hitPreview = hit{OK: core != nil, From: "constructed", Name: "CartHandlerWithQueries (as preview)"}
		}

		snsDeps.Cart = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		hitCart = hit{OK: snsDeps.Cart != nil, From: "constructed", Name: "wrapped(GET->CartQueryHandler, else->CartHandler/405)"}

		if core != nil {
			snsDeps.Preview = core
		}
	} else if cartUC != nil {
		core := mallHandler.NewCartHandlerWithQueries(cartUC, nil, previewQ)
		snsDeps.Cart = core
		snsDeps.Preview = core
		hitCart = hit{OK: true, From: "constructed", Name: "CartHandlerWithQueries"}
		hitPreview = hit{OK: true, From: "constructed", Name: "CartHandlerWithQueries (as preview)"}
	}

	// user_auth
	{
		userAuth := newUserAuthMiddlewareBestEffort(cont.FirebaseAuth)
		if userAuth == nil {
			log.Printf("[mall_container] WARN: user_auth middleware is not available (firebase auth client missing). protected routes may 401.")
		} else {
			wrap := func(h http.Handler) http.Handler {
				if h == nil {
					return nil
				}
				return userAuth.Handler(h)
			}

			snsDeps.User = wrap(snsDeps.User)
			snsDeps.ShippingAddress = wrap(snsDeps.ShippingAddress)
			snsDeps.BillingAddress = wrap(snsDeps.BillingAddress)
			snsDeps.Avatar = wrap(snsDeps.Avatar)
			snsDeps.Wallet = wrap(snsDeps.Wallet)

			snsDeps.Cart = wrap(snsDeps.Cart)
			snsDeps.Preview = wrap(snsDeps.Preview)
			snsDeps.Payment = wrap(snsDeps.Payment)
			snsDeps.Order = wrap(snsDeps.Order)
		}
	}

	// routes
	mall.Register(mux, mall.Deps{
		List:             snsDeps.List,
		Inventory:        snsDeps.Inventory,
		ProductBlueprint: snsDeps.ProductBlueprint,
		Model:            snsDeps.Model,
		Catalog:          snsDeps.Catalog,
		TokenBlueprint:   snsDeps.TokenBlueprint,
		Company:          snsDeps.Company,
		Brand:            snsDeps.Brand,
		SignIn:           snsDeps.SignIn,
		User:             snsDeps.User,
		ShippingAddress:  snsDeps.ShippingAddress,
		BillingAddress:   snsDeps.BillingAddress,
		Avatar:           snsDeps.Avatar,
		AvatarState:      snsDeps.AvatarState,
		Wallet:           snsDeps.Wallet,
		Cart:             snsDeps.Cart,
		Post:             snsDeps.Post,
		Payment:          snsDeps.Payment,
		Preview:          snsDeps.Preview,
		Order:            snsDeps.Order,
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

		// ✅ エラー発見優先: safeHandle せず、そのまま登録（重複時は panic）
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
// order query getter (best-effort, type-mismatch safe)
// ------------------------------------------------------------

func getOrderQueryBestEffort(cont *Container, depsAny any) any {
	// 1) Container methods (reflect)
	if cont != nil {
		rv := reflect.ValueOf(cont)
		if rv.IsValid() {
			for _, name := range []string{
				"SNSOrderQuery",
				"MallOrderQuery",
				"OrderQuery",
				"SNSOrderQueryService",
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
				"SNSOrderQuery",
				"OrderQuery",
				"SNSOrderQueryService",
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

func setOptionalResolverField(handler any, fieldName string, value any) {
	if handler == nil || value == nil || strings.TrimSpace(fieldName) == "" {
		return
	}

	rv := reflect.ValueOf(handler)
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

func getFirestoreClientStrict(cont *Container, depsAny any) *firestore.Client {
	// 1) Common Container fields (direct)
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
				if fs, ok := f.Interface().(*firestore.Client); ok && fs != nil {
					return fs
				}
			}
		}
	}

	// 2) Common RouterDeps fields (direct)
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
				if fs, ok := f.Interface().(*firestore.Client); ok && fs != nil {
					return fs
				}
			}
		}
	}

	return nil
}
