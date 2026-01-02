// backend/internal/platform/di/sns_container.go
package di

import (
	"log"
	"net/http"
	"reflect"
	"strings"

	snshandler "narratives/internal/adapters/in/http/sns/handler"
	snsquery "narratives/internal/application/query/sns"
	appresolver "narratives/internal/application/resolver"
	usecase "narratives/internal/application/usecase"
	ldom "narratives/internal/domain/list"
)

// RegisterSNSFromContainer registers SNS routes using *Container.
//
// ✅ After migration:
// - Route registration / auth wrapping / cart split are delegated to SNSAPI (sns_api.go).
// - This file focuses on "wiring": resolve deps (reduced best-effort) -> build SNSAPI -> api.Register(mux).
func RegisterSNSFromContainer(mux *http.ServeMux, cont *Container) {
	if mux == nil || cont == nil {
		return
	}

	depsAny := any(cont.RouterDeps())

	// ------------------------------------------------------------
	// Resolve shared deps (FIXED names only; remove fuzzy name-matching)
	// ------------------------------------------------------------

	// catalog query (fixed)
	var catalogQ *snsquery.SNSCatalogQuery
	{
		if x, ok := any(cont).(interface {
			SNSCatalogQuery() *snsquery.SNSCatalogQuery
		}); ok {
			catalogQ = x.SNSCatalogQuery()
		}
	}

	// name resolver (fixed)
	var nameResolver *appresolver.NameResolver
	{
		if x, ok := any(cont).(interface {
			SNSNameResolver() *appresolver.NameResolver
		}); ok {
			nameResolver = x.SNSNameResolver()
		}
	}

	// ✅ list repo (prefer fixed; fallback: RouterDeps field only)
	var listRepo ldom.Repository
	{
		if x, ok := any(cont).(interface{ ListRepo() ldom.Repository }); ok {
			listRepo = x.ListRepo()
		} else if x, ok := any(cont).(interface{ ListRepository() ldom.Repository }); ok {
			listRepo = x.ListRepository()
		}

		if listRepo == nil {
			// RouterDeps 側に入っているケースは拾う（struct field だけ）
			listRepo = getFieldPtr[ldom.Repository](depsAny, "ListRepo", "ListRepository")
		}
	}

	// ✅ cart / preview queries (fixed)
	var cartQ *snsquery.SNSCartQuery
	{
		if x, ok := any(cont).(interface{ SNSCartQuery() *snsquery.SNSCartQuery }); ok {
			cartQ = x.SNSCartQuery()
		}
		if cartQ != nil && nameResolver != nil && cartQ.Resolver == nil {
			cartQ.Resolver = nameResolver
		}
		if cartQ != nil && listRepo != nil && cartQ.ListRepo == nil {
			cartQ.ListRepo = listRepo
		}
	}

	var previewQ *snsquery.SNSPreviewQuery
	{
		if x, ok := any(cont).(interface {
			SNSPreviewQuery() *snsquery.SNSPreviewQuery
		}); ok {
			previewQ = x.SNSPreviewQuery()
		}
		if previewQ != nil && nameResolver != nil && previewQ.Resolver == nil {
			previewQ.Resolver = nameResolver
		}
		// preview 側も listRepo を持っている場合だけ注入（フィールドが無い実装でも安全）
		if previewQ != nil && listRepo != nil {
			setOptionalResolverField(previewQ, "ListRepo", listRepo)
			setOptionalResolverField(previewQ, "ListRepository", listRepo)
		}
	}

	// company / brand usecase (fixed)
	var companyUC *usecase.CompanyUsecase
	{
		if x, ok := any(cont).(interface {
			CompanyUsecase() *usecase.CompanyUsecase
		}); ok {
			companyUC = x.CompanyUsecase()
		}
	}
	var brandUC *usecase.BrandUsecase
	{
		if x, ok := any(cont).(interface{ BrandUsecase() *usecase.BrandUsecase }); ok {
			brandUC = x.BrandUsecase()
		}
	}

	// ✅ cart usecase（fixed + RouterDeps field fallback）
	var cartUC *usecase.CartUsecase
	{
		if x, ok := any(cont).(interface{ CartUsecase() *usecase.CartUsecase }); ok {
			cartUC = x.CartUsecase()
		}
		if cartUC == nil {
			cartUC = getFieldPtr[*usecase.CartUsecase](depsAny, "CartUC", "CartUsecase")
		}
	}

	// core usecases (from RouterDeps)
	listUC := getFieldPtr[*usecase.ListUsecase](depsAny, "ListUC", "ListUsecase")
	invUC := getFieldPtr[*usecase.InventoryUsecase](depsAny, "InventoryUC", "InventoryUsecase")
	pbUC := getFieldPtr[*usecase.ProductBlueprintUsecase](depsAny, "ProductBlueprintUC", "ProductBlueprintUsecase")
	modelUC := getFieldPtr[*usecase.ModelUsecase](depsAny, "ModelUC", "ModelUsecase")
	tokenBlueprintUC := getFieldPtr[*usecase.TokenBlueprintUsecase](depsAny, "TokenBlueprintUC", "TokenBlueprintUsecase")

	// ------------------------------------------------------------
	// Build fixed API surface (SNSAPI) and delegate registration to sns_api.go
	// ------------------------------------------------------------

	api := buildSNSAPI(
		listUC, invUC, pbUC, modelUC, tokenBlueprintUC,
		companyUC, brandUC,
		nameResolver, catalogQ,
	)

	// ------------------------------------------------------------
	// Optional handlers (fixed names only)
	// ------------------------------------------------------------

	// ✅ sign-in
	{
		var h http.Handler
		if x, ok := any(cont).(interface{ SNSSignInHandler() http.Handler }); ok {
			h = x.SNSSignInHandler()
		} else if x, ok := any(cont).(interface{ SNSSignIn() http.Handler }); ok {
			h = x.SNSSignIn()
		}
		api.SignIn = h
	}

	// ✅ avatar state
	{
		var h http.Handler
		if x, ok := any(cont).(interface{ SNSAvatarStateHandler() http.Handler }); ok {
			h = x.SNSAvatarStateHandler()
		} else if x, ok := any(cont).(interface{ SNSAvatarState() http.Handler }); ok {
			h = x.SNSAvatarState()
		}
		api.AvatarState = h
	}

	// ✅ wallet
	{
		var h http.Handler
		if x, ok := any(cont).(interface{ SNSWalletHandler() http.Handler }); ok {
			h = x.SNSWalletHandler()
		} else if x, ok := any(cont).(interface{ SNSWallet() http.Handler }); ok {
			h = x.SNSWallet()
		}
		api.Wallet = h
	}

	// ============================================================
	// ✅ cart + preview（重要）
	//
	// 目的:
	// - GET /sns/cart は cart_query.go を確実に通す（read-model）
	// - それ以外（POST/PUT/DELETE /sns/cart/*）は cart handler（write）へ委譲
	// ============================================================

	if cartUC != nil {
		core := snshandler.NewCartHandlerWithQueries(cartUC, cartQ, previewQ)

		api.CartWrite = core
		api.Preview = core

		if cartQ != nil {
			api.CartQuery = snshandler.NewSNSCartQueryHandler(cartQ)
		}
	} else {
		// fallback: fixed names only（後で完全固定にする）
		{
			var h http.Handler
			if x, ok := any(cont).(interface{ SNSCartHandler() http.Handler }); ok {
				h = x.SNSCartHandler()
			} else if x, ok := any(cont).(interface{ SNSCart() http.Handler }); ok {
				h = x.SNSCart()
			}
			api.CartWrite = h
		}
		{
			var h http.Handler
			if x, ok := any(cont).(interface{ SNSPreviewHandler() http.Handler }); ok {
				h = x.SNSPreviewHandler()
			} else if x, ok := any(cont).(interface{ SNSPreview() http.Handler }); ok {
				h = x.SNSPreview()
			}
			api.Preview = h
		}

		// 旧: reflection 注入（exported field の場合のみ効く可能性があるため残す）
		if api.CartWrite != nil && cartQ != nil {
			setOptionalResolverField(api.CartWrite, "CartQuery", cartQ)
			setOptionalResolverField(api.CartWrite, "Query", cartQ)
			setOptionalResolverField(api.CartWrite, "CartQ", cartQ)
		}
		if api.Preview != nil && previewQ != nil {
			setOptionalResolverField(api.Preview, "PreviewQuery", previewQ)
			setOptionalResolverField(api.Preview, "Query", previewQ)
			setOptionalResolverField(api.Preview, "PreviewQ", previewQ)
		}
	}

	// ✅ posts
	{
		var h http.Handler
		if x, ok := any(cont).(interface{ SNSPostHandler() http.Handler }); ok {
			h = x.SNSPostHandler()
		} else if x, ok := any(cont).(interface{ SNSPost() http.Handler }); ok {
			h = x.SNSPost()
		}
		api.Post = h
	}

	// ✅ payment
	{
		var h http.Handler
		if x, ok := any(cont).(interface{ SNSPaymentHandler() http.Handler }); ok {
			h = x.SNSPaymentHandler()
		} else if x, ok := any(cont).(interface{ SNSPayment() http.Handler }); ok {
			h = x.SNSPayment()
		}
		api.Payment = h
	}

	// firebase auth client (used for buyer-auth wrapping in SNSAPI.Register)
	api.FirebaseAuth = cont.FirebaseAuth

	// logs (retain current observability)
	log.Printf("[sns_container] inject result signIn=%t cartWrite=%t cartQuery=%t preview=%t payment=%t cartUC=%t cartQ=%t previewQ=%t listRepo=%t",
		api.SignIn != nil,
		api.CartWrite != nil,
		api.CartQuery != nil,
		api.Preview != nil,
		api.Payment != nil,
		cartUC != nil,
		cartQ != nil,
		previewQ != nil,
		listRepo != nil,
	)

	if api.SignIn == nil {
		log.Printf("[sns_container] WARN: SignIn handler not wired (expected SNSSignInHandler/SNSSignIn)")
	}
	if api.CartWrite == nil && api.CartQuery == nil {
		log.Printf("[sns_container] WARN: Cart handler not wired (expected CartUsecase or SNSCartHandler/SNSCart)")
	}
	if api.Preview == nil {
		log.Printf("[sns_container] WARN: Preview handler not wired (expected CartUsecase or SNSPreviewHandler/SNSPreview)")
	}
	if api.Payment == nil {
		log.Printf("[sns_container] WARN: Payment handler not wired (expected SNSPaymentHandler/SNSPayment)")
	}

	// Delegate final route registration + auth wrapping + cart split to SNSAPI.
	api.Register(mux)
}

// buildSNSAPI wires core SNS handlers from explicit usecases/queries.
// (Auth resources like User/Shipping/Billing/Avatar are expected to be wired elsewhere for now.)
func buildSNSAPI(
	listUC *usecase.ListUsecase,
	invUC *usecase.InventoryUsecase,
	pbUC *usecase.ProductBlueprintUsecase,
	modelUC *usecase.ModelUsecase,
	tokenBlueprintUC *usecase.TokenBlueprintUsecase,

	companyUC *usecase.CompanyUsecase,
	brandUC *usecase.BrandUsecase,

	nameResolver *appresolver.NameResolver,
	catalogQ *snsquery.SNSCatalogQuery,
) SNSAPI {
	if catalogQ != nil && nameResolver != nil && catalogQ.NameResolver == nil {
		catalogQ.NameResolver = nameResolver
	}

	var api SNSAPI

	if listUC != nil {
		api.List = snshandler.NewSNSListHandler(listUC)
	}
	if invUC != nil {
		api.Inventory = snshandler.NewSNSInventoryHandler(invUC)
	}
	if pbUC != nil {
		h := snshandler.NewSNSProductBlueprintHandler(pbUC)
		if nameResolver != nil {
			setOptionalResolverField(h, "BrandNameResolver", nameResolver)
			setOptionalResolverField(h, "CompanyNameResolver", nameResolver)
			setOptionalResolverField(h, "NameResolver", nameResolver)
		}
		api.ProductBlueprint = h
	}
	if modelUC != nil {
		api.Model = snshandler.NewSNSModelHandler(modelUC)
	}
	if catalogQ != nil {
		api.Catalog = snshandler.NewSNSCatalogHandler(catalogQ)
	}
	if companyUC != nil {
		api.Company = snshandler.NewSNSCompanyHandler(companyUC)
	}
	if brandUC != nil {
		api.Brand = snshandler.NewSNSBrandHandler(brandUC)
	}

	if tokenBlueprintUC != nil {
		h := snshandler.NewSNSTokenBlueprintHandler(tokenBlueprintUC)
		if nameResolver != nil {
			setOptionalResolverField(h, "BrandNameResolver", nameResolver)
			setOptionalResolverField(h, "CompanyNameResolver", nameResolver)
			setOptionalResolverField(h, "NameResolver", nameResolver)
		}
		imgResolver := appresolver.NewImageURLResolver("")
		setOptionalResolverField(h, "ImageResolver", imgResolver)
		setOptionalResolverField(h, "ImageURLResolver", imgResolver)
		setOptionalResolverField(h, "IconURLResolver", imgResolver)
		api.TokenBlueprint = h
	}

	return api
}

// ------------------------------------------------------------
// Reflection helpers (kept for now; used for optional field injection + RouterDeps reads)
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
