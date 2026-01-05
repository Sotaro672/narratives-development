// backend\internal\platform\di\mall_container.go
package di

import (
	"log"
	"net/http"
	"reflect"
	"strings"

	"cloud.google.com/go/firestore"
	firebaseauth "firebase.google.com/go/v4/auth"

	snshttp "narratives/internal/adapters/in/http/mall"
	snshandler "narratives/internal/adapters/in/http/mall/handler"
	"narratives/internal/adapters/in/http/middleware"
	outfs "narratives/internal/adapters/out/firestore"
	snsquery "narratives/internal/application/query/mall"
	appresolver "narratives/internal/application/resolver"
	usecase "narratives/internal/application/usecase"
	ldom "narratives/internal/domain/list"
)

// ------------------------------------------------------------
// ✅ Route name constants (freeze naming variance)
// ------------------------------------------------------------

const (
	SNSPaymentPath = "/sns/payment" // ✅ official payment endpoint (single source of truth)
	SNSOrdersPath  = "/sns/orders"  // ✅ official orders endpoint
)

// ------------------------------------------------------------
// Hit tracing (minimal / deterministic)
// ------------------------------------------------------------

type hit struct {
	OK   bool
	From string // "Container.method" / "RouterDeps.field" / "constructed"
	Name string // method/field/ctor name
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

// SNSDeps is a buyer-facing (sns) HTTP dependency set.
type SNSDeps struct {
	// Handlers
	List             http.Handler
	Inventory        http.Handler
	ProductBlueprint http.Handler
	Model            http.Handler
	Catalog          http.Handler

	TokenBlueprint http.Handler // patch

	// name resolver endpoints
	Company http.Handler
	Brand   http.Handler

	// auth entry (cart empty ok)
	SignIn http.Handler

	// auth onboarding resources
	User            http.Handler
	ShippingAddress http.Handler
	BillingAddress  http.Handler
	Avatar          http.Handler

	// optional (currently may be nil)
	AvatarState http.Handler
	Wallet      http.Handler
	Post        http.Handler

	// cart/preview
	Cart    http.Handler
	Preview http.Handler

	// payment (order context / checkout)
	Payment http.Handler

	// ✅ NEW: orders (create/get)
	Order http.Handler
}

// NewSNSDeps wires SNS handlers.
func NewSNSDeps(
	listUC *usecase.ListUsecase,
	invUC *usecase.InventoryUsecase,
	pbUC *usecase.ProductBlueprintUsecase,
	modelUC *usecase.ModelUsecase,
	tokenBlueprintUC *usecase.TokenBlueprintUsecase,
	catalogQ *snsquery.SNSCatalogQuery,
) SNSDeps {
	return NewSNSDepsWithNameResolver(
		listUC,
		invUC,
		pbUC,
		modelUC,
		tokenBlueprintUC,
		nil, // nameResolver
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
	catalogQ *snsquery.SNSCatalogQuery,
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
	catalogQ *snsquery.SNSCatalogQuery,
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
		listHandler = snshandler.NewSNSListHandler(listUC)
	}
	if invUC != nil {
		invHandler = snshandler.NewSNSInventoryHandler(invUC)
	}
	if pbUC != nil {
		pbHandler = snshandler.NewSNSProductBlueprintHandler(pbUC)
		if nameResolver != nil {
			setOptionalResolverField(pbHandler, "BrandNameResolver", nameResolver)
			setOptionalResolverField(pbHandler, "CompanyNameResolver", nameResolver)
			setOptionalResolverField(pbHandler, "NameResolver", nameResolver)
		}
	}
	if modelUC != nil {
		modelHandler = snshandler.NewSNSModelHandler(modelUC)
	}
	if catalogQ != nil {
		catalogHandler = snshandler.NewSNSCatalogHandler(catalogQ)
	}
	if companyUC != nil {
		companyHandler = snshandler.NewSNSCompanyHandler(companyUC)
	}
	if brandUC != nil {
		brandHandler = snshandler.NewSNSBrandHandler(brandUC)
	}

	if tokenBlueprintUC != nil {
		tokenBlueprintHandler = snshandler.NewSNSTokenBlueprintHandler(tokenBlueprintUC)
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

		SignIn:          nil,
		User:            nil,
		ShippingAddress: nil,
		BillingAddress:  nil,
		Avatar:          nil,

		AvatarState: nil,
		Wallet:      nil,
		Post:        nil,

		Cart:    nil,
		Preview: nil,

		Payment: nil,
		Order:   nil,
	}
}

// RegisterSNSFromContainer registers SNS routes using *Container.
//
// ✅ This version keeps the “minimalized naming variance” policy,
// but restores cart_query connectivity deterministically:
//
//   - /sns/cart (GET) is ALWAYS served by SNSCartQueryHandler when cartQ != nil
//     (even if cartUC is nil). This prevents 404.
//   - /sns/cart (non-GET) is served by CartHandlerWithQueries ONLY when cartUC != nil,
//     otherwise returns 405.
//
// listRepo is always constructed from Firestore.
func RegisterSNSFromContainer(mux *http.ServeMux, cont *Container) {
	if mux == nil || cont == nil {
		return
	}

	depsAny := any(cont.RouterDeps())

	// --------------------------------------------
	// 1) Direct Container.method (fixed names)
	// --------------------------------------------
	var (
		hitSignIn   hit
		hitShip     hit
		hitPay      hit
		hitOrder    hit
		hitNameRes  hit
		hitCatalogQ hit
		hitCartQ    hit
		hitPrevQ    hit
		hitListRepo hit

		hitCart    hit
		hitPreview hit
	)

	// Queries / resolver (direct)
	nameResolver := cont.SNSNameResolver()
	hitNameRes = hit{OK: nameResolver != nil, From: "Container.method", Name: "SNSNameResolver"}

	catalogQ := cont.SNSCatalogQuery()
	hitCatalogQ = hit{OK: catalogQ != nil, From: "Container.method", Name: "SNSCatalogQuery"}

	cartQ := cont.SNSCartQuery()
	hitCartQ = hit{OK: cartQ != nil, From: "Container.method", Name: "SNSCartQuery"}

	previewQ := cont.SNSPreviewQuery()
	hitPrevQ = hit{OK: previewQ != nil, From: "Container.method", Name: "SNSPreviewQuery"}

	// Handlers (direct per your log)
	signInH := cont.SNSSignInHandler()
	hitSignIn = hit{OK: signInH != nil, From: "Container.method", Name: "SNSSignInHandler"}

	shipH := cont.SNSShippingAddressHandler()
	hitShip = hit{OK: shipH != nil, From: "Container.method", Name: "SNSShippingAddressHandler"}

	paymentH := cont.SNSPaymentHandler()
	hitPay = hit{OK: paymentH != nil, From: "Container.method", Name: "SNSPaymentHandler"}

	// --------------------------------------------
	// 2) ListRepo: always construct from Firestore
	// --------------------------------------------
	var listRepo ldom.Repository
	{
		fs := getFirestoreClientStrict(cont, depsAny)
		if fs != nil {
			listRepo = outfs.NewListRepositoryFS(fs)
		}
		hitListRepo = hit{OK: listRepo != nil, From: "constructed", Name: "outfs.NewListRepositoryFS"}
	}

	// --------------------------------------------
	// 3) Inject resolver/repo into queries (fixed)
	// --------------------------------------------
	if catalogQ != nil && nameResolver != nil && catalogQ.NameResolver == nil {
		catalogQ.NameResolver = nameResolver
	}
	if cartQ != nil && nameResolver != nil && cartQ.Resolver == nil {
		cartQ.Resolver = nameResolver
	}
	if cartQ != nil && listRepo != nil && cartQ.ListRepo == nil {
		cartQ.ListRepo = listRepo
	}
	if previewQ != nil && nameResolver != nil && previewQ.Resolver == nil {
		previewQ.Resolver = nameResolver
	}
	if previewQ != nil && listRepo != nil {
		// field may not exist → safe via reflection
		setOptionalResolverField(previewQ, "ListRepo", listRepo)
		setOptionalResolverField(previewQ, "ListRepository", listRepo)
	}

	// --------------------------------------------
	// 4) Core usecases (RouterDeps.field)
	// --------------------------------------------
	listUC := getFieldPtr[*usecase.ListUsecase](depsAny, "ListUC", "ListUsecase")
	invUC := getFieldPtr[*usecase.InventoryUsecase](depsAny, "InventoryUC", "InventoryUsecase")
	pbUC := getFieldPtr[*usecase.ProductBlueprintUsecase](depsAny, "ProductBlueprintUC", "ProductBlueprintUsecase")
	modelUC := getFieldPtr[*usecase.ModelUsecase](depsAny, "ModelUC", "ModelUsecase")
	tokenBlueprintUC := getFieldPtr[*usecase.TokenBlueprintUsecase](depsAny, "TokenBlueprintUC", "TokenBlueprintUsecase")

	companyUC := getFieldPtr[*usecase.CompanyUsecase](depsAny, "CompanyUC", "CompanyUsecase")
	brandUC := getFieldPtr[*usecase.BrandUsecase](depsAny, "BrandUC", "BrandUsecase")

	// ✅ NEW: OrderUsecase -> SNS OrderHandler
	orderUC := getFieldPtr[*usecase.OrderUsecase](depsAny, "OrderUC", "OrderUsecase", "SNSOrderUC", "SNSOrderUsecase")

	// cartUC（ここは “cart_query 復旧” に必須ではないが、write を生かすため拾えるなら拾う）
	cartUC := getFieldPtr[*usecase.CartUsecase](depsAny, "CartUC", "CartUsecase")
	if cartUC == nil {
		// 最小限の追加：Container に cartUC getter がある場合だけ拾う（名揺れ吸収は増やさない）
		if x, ok := any(cont).(interface{ CartUsecase() *usecase.CartUsecase }); ok {
			cartUC = x.CartUsecase()
		} else if x, ok := any(cont).(interface{ GetCartUsecase() *usecase.CartUsecase }); ok {
			cartUC = x.GetCartUsecase()
		}
	}

	// --------------------------------------------
	// 5) Build base deps (list/inv/pb/model/token/catalog/org)
	// --------------------------------------------
	snsDeps := NewSNSDepsWithNameResolverAndOrgHandlers(
		listUC, invUC, pbUC, modelUC, tokenBlueprintUC,
		companyUC, brandUC, nameResolver, catalogQ,
	)

	// Set direct handlers
	snsDeps.SignIn = signInH
	snsDeps.ShippingAddress = shipH
	snsDeps.Payment = paymentH

	// --------------------------------------------
	// 6) Construct-fixed handlers (per your prior working log)
	// --------------------------------------------
	// User
	{
		userUC := getFieldPtr[*usecase.UserUsecase](depsAny, "UserUC", "UserUsecase", "SNSUserUC", "SNSUserUsecase")
		if userUC != nil {
			snsDeps.User = snshandler.NewUserHandler(userUC)
		}
	}
	// Billing
	{
		billUC := getFieldPtr[*usecase.BillingAddressUsecase](depsAny, "BillingAddressUC", "BillingAddressUsecase", "SNSBillingAddressUC", "SNSBillingAddressUsecase")
		if billUC != nil {
			snsDeps.BillingAddress = snshandler.NewBillingAddressHandler(billUC)
		}
	}
	// Avatar
	{
		avatarUC := getFieldPtr[*usecase.AvatarUsecase](depsAny, "AvatarUC", "AvatarUsecase", "SNSAvatarUC", "SNSAvatarUsecase")
		if avatarUC != nil {
			snsDeps.Avatar = snshandler.NewAvatarHandler(avatarUC)
		}
	}
	// ✅ NEW: Order
	if orderUC != nil {
		snsDeps.Order = snshandler.NewOrderHandler(orderUC)
		hitOrder = hit{OK: snsDeps.Order != nil, From: "constructed", Name: "snshandler.NewOrderHandler"}
	} else {
		hitOrder = hit{OK: false, From: "RouterDeps.field", Name: "OrderUC"}
	}

	// --------------------------------------------
	// 7) ✅ Cart Query connectivity restore (always-on for GET)
	// --------------------------------------------
	// - If cartQ exists: register /sns/cart GET via SNSCartQueryHandler (prevents 404)
	// - If cartUC exists: non-GET goes to write handler; else 405
	if cartQ != nil {
		qh := snshandler.NewSNSCartQueryHandler(cartQ)

		var core http.Handler
		if cartUC != nil {
			core = snshandler.NewCartHandlerWithQueries(cartUC, cartQ, previewQ)
			hitPreview = hit{OK: core != nil, From: "constructed", Name: "CartHandlerWithQueries (as preview)"}
		}

		snsDeps.Cart = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := r.URL.Path

			// GET -> read-model
			if r.Method == http.MethodGet {
				// /sns/cart , /sns/cart/ , (if mux routes here) /sns/cart/query*
				if p == "/sns/cart" || p == "/sns/cart/" || strings.HasPrefix(p, "/sns/cart/query") {
					qh.ServeHTTP(w, r)
					return
				}
			}

			// non-GET -> write handler if available
			if core != nil {
				core.ServeHTTP(w, r)
				return
			}

			// write not supported in this config
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		})

		hitCart = hit{OK: snsDeps.Cart != nil, From: "constructed", Name: "wrapped(GET->SNSCartQueryHandler, else->CartHandler/405)"}

		// Preview route is optional; keep it only when we have core
		if core != nil {
			snsDeps.Preview = core
		}
	} else if cartUC != nil {
		// (rare) cartQ nil but write exists
		core := snshandler.NewCartHandlerWithQueries(cartUC, nil, previewQ)
		snsDeps.Cart = core
		snsDeps.Preview = core
		hitCart = hit{OK: true, From: "constructed", Name: "CartHandlerWithQueries"}
		hitPreview = hit{OK: true, From: "constructed", Name: "CartHandlerWithQueries (as preview)"}
	}

	// --------------------------------------------
	// 8) Logs (inject result)
	// --------------------------------------------
	log.Printf("[sns_container] inject result signIn=%t user=%t ship=%t bill=%t avatar=%t cart=%t preview=%t payment=%t order=%t cartUC=%t cartQ=%t previewQ=%t listRepo=%t",
		snsDeps.SignIn != nil,
		snsDeps.User != nil,
		snsDeps.ShippingAddress != nil,
		snsDeps.BillingAddress != nil,
		snsDeps.Avatar != nil,
		snsDeps.Cart != nil,
		snsDeps.Preview != nil,
		snsDeps.Payment != nil,
		snsDeps.Order != nil,
		cartUC != nil,
		cartQ != nil,
		previewQ != nil,
		listRepo != nil,
	)

	// hits (minimal / deterministic)
	log.Printf("[sns_container] inject hits "+
		"signIn=%s ship=%s payment=%s order=%s cart=%s preview=%s "+
		"nameResolver=%s catalogQ=%s cartQ=%s previewQ=%s listRepo=%s",
		hitSignIn.String(),
		hitShip.String(),
		hitPay.String(),
		hitOrder.String(),
		hitCart.String(),
		hitPreview.String(),
		hitNameRes.String(),
		hitCatalogQ.String(),
		hitCartQ.String(),
		hitPrevQ.String(),
		hitListRepo.String(),
	)

	// --------------------------------------------
	// 9) Apply user_auth where needed (existing behavior)
	// --------------------------------------------
	{
		userAuth := newUserAuthMiddlewareBestEffort(cont.FirebaseAuth)
		if userAuth == nil {
			log.Printf("[sns_container] WARN: user_auth middleware is not available (firebase auth client missing). protected routes may 401.")
		} else {
			wrap := func(h http.Handler) http.Handler {
				if h == nil {
					return nil
				}
				return userAuth.Handler(h)
			}

			// buyer-auth required
			snsDeps.User = wrap(snsDeps.User)
			snsDeps.ShippingAddress = wrap(snsDeps.ShippingAddress)
			snsDeps.BillingAddress = wrap(snsDeps.BillingAddress)
			snsDeps.Avatar = wrap(snsDeps.Avatar)

			// cart/preview/payment/order are protected in your current design
			snsDeps.Cart = wrap(snsDeps.Cart)
			snsDeps.Preview = wrap(snsDeps.Preview)
			snsDeps.Payment = wrap(snsDeps.Payment)
			snsDeps.Order = wrap(snsDeps.Order)

			log.Printf("[sns_container] user_auth applied: user=%t ship=%t bill=%t avatar=%t cart=%t preview=%t payment=%t order=%t",
				snsDeps.User != nil,
				snsDeps.ShippingAddress != nil,
				snsDeps.BillingAddress != nil,
				snsDeps.Avatar != nil,
				snsDeps.Cart != nil,
				snsDeps.Preview != nil,
				snsDeps.Payment != nil,
				snsDeps.Order != nil,
			)
		}
	}

	RegisterSNSRoutes(mux, snsDeps)
}

// RegisterSNSRoutes registers buyer-facing routes onto mux.
func RegisterSNSRoutes(mux *http.ServeMux, deps SNSDeps) {
	if mux == nil {
		return
	}

	// existing sns register
	snshttp.Register(mux, snshttp.Deps{
		List:             deps.List,
		Inventory:        deps.Inventory,
		ProductBlueprint: deps.ProductBlueprint,
		Model:            deps.Model,
		Catalog:          deps.Catalog,

		TokenBlueprint: deps.TokenBlueprint,

		Company: deps.Company,
		Brand:   deps.Brand,

		SignIn: deps.SignIn,

		User:            deps.User,
		ShippingAddress: deps.ShippingAddress,
		BillingAddress:  deps.BillingAddress,
		Avatar:          deps.Avatar,

		AvatarState: deps.AvatarState,
		Wallet:      deps.Wallet,
		Cart:        deps.Cart,
		Preview:     deps.Preview,
		Post:        deps.Post,

		Payment: deps.Payment,
	})

	// ✅ payment endpoint hard-bind (freeze naming variance)
	// - Official path is SNSPaymentPath ("/sns/payment")
	// - Register both exact and trailing-slash variants.
	safeHandle(mux, SNSPaymentPath, deps.Payment)
	safeHandle(mux, SNSPaymentPath+"/", deps.Payment)

	// ✅ orders endpoint hard-bind
	// - Official path is SNSOrdersPath ("/sns/orders")
	// - Register both exact and trailing-slash variants.
	safeHandle(mux, SNSOrdersPath, deps.Order)
	safeHandle(mux, SNSOrdersPath+"/", deps.Order)
}

// ------------------------------------------------------------
// Middleware builder
// ------------------------------------------------------------

// newUserAuthMiddlewareBestEffort builds middleware.UserAuthMiddleware from firebaseauth.Client.
//
// user_auth.go expects *middleware.FirebaseAuthClient.
// In your codebase, middleware.FirebaseAuthClient is a type alias of firebaseauth.Client
// (declared in member_auth.go), so we can pass it directly.
func newUserAuthMiddlewareBestEffort(fb *firebaseauth.Client) *middleware.UserAuthMiddleware {
	if fb == nil {
		return nil
	}
	return &middleware.UserAuthMiddleware{FirebaseAuth: fb}
}

// ------------------------------------------------------------
// ServeMux safe handle (avoid panic on duplicate registration)
// ------------------------------------------------------------

func safeHandle(mux *http.ServeMux, pattern string, h http.Handler) {
	if mux == nil || h == nil {
		return
	}
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			// already registered: ignore
		}
	}()
	mux.Handle(pattern, h)
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
// Firestore client (minimal; used only for ListRepo)
// ------------------------------------------------------------

// getFirestoreClientStrict tries a small set of stable places for *firestore.Client.
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
