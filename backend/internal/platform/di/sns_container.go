// backend/internal/platform/di/sns_container.go
package di

import (
	"net/http"
	"reflect"
	"strings"

	snshttp "narratives/internal/adapters/in/http/sns"
	snshandler "narratives/internal/adapters/in/http/sns/handler"
	snsquery "narratives/internal/application/query/sns"
	appresolver "narratives/internal/application/resolver"
	usecase "narratives/internal/application/usecase"
)

// SNSDeps is a buyer-facing (sns) HTTP dependency set.
type SNSDeps struct {
	// Handlers
	List             http.Handler
	Inventory        http.Handler
	ProductBlueprint http.Handler
	Model            http.Handler
	Catalog          http.Handler

	TokenBlueprint http.Handler // patch

	// ✅ NEW: name resolver endpoints
	Company http.Handler
	Brand   http.Handler

	// ✅ NEW: auth onboarding resources
	User            http.Handler
	ShippingAddress http.Handler
	BillingAddress  http.Handler
	Avatar          http.Handler
}

// NewSNSDeps wires SNS handlers.
// （後方互換のため、NameResolver なしの関数を残す）
//
// NOTE:
// - Company/Brand は v2 関数（NewSNSDepsWithNameResolverAndOrgHandlers）側で注入する。
// - 既存呼び出しを壊さないため、ここでは nil 注入で OK（ルーティングは RegisterSNSFromContainer 側が担当）。
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

// NewSNSDepsWithNameResolver wires SNS handlers with optional NameResolver.
//
// SNS は companyId 境界が無い（公開）ため、console 用 query は使わない。
// NameResolver は「brandName / companyName 解決」に利用する。
func NewSNSDepsWithNameResolver(
	listUC *usecase.ListUsecase,
	invUC *usecase.InventoryUsecase,
	pbUC *usecase.ProductBlueprintUsecase,
	modelUC *usecase.ModelUsecase,
	tokenBlueprintUC *usecase.TokenBlueprintUsecase,

	// name resolver (brandName/companyName)
	nameResolver *appresolver.NameResolver,

	// catalog query
	catalogQ *snsquery.SNSCatalogQuery,
) SNSDeps {
	return NewSNSDepsWithNameResolverAndOrgHandlers(
		listUC,
		invUC,
		pbUC,
		modelUC,
		tokenBlueprintUC,
		nil, // companyUC
		nil, // brandUC
		nameResolver,
		catalogQ,
	)
}

// NewSNSDepsWithNameResolverAndOrgHandlers wires SNS handlers with optional NameResolver + GET-only org handlers.
//
// - /sns/companies/{id}
// - /sns/brands/{id}
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
	// ✅ IMPORTANT:
	// CatalogQuery 側にも NameResolver を注入しないと、
	// sns_catalog の fillProductBlueprintNames() が呼ばれず、name_resolver のログも出ない。
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

		// ✅ NEW: productBlueprint 側にも name resolver を注入（handler 側にフィールドがあれば入る）
		if nameResolver != nil {
			setOptionalResolverField(pbHandler, "BrandNameResolver", nameResolver)
			setOptionalResolverField(pbHandler, "CompanyNameResolver", nameResolver)
			setOptionalResolverField(pbHandler, "NameResolver", nameResolver) // 将来用
		}
	}

	if modelUC != nil {
		modelHandler = snshandler.NewSNSModelHandler(modelUC)
	}

	if catalogQ != nil {
		catalogHandler = snshandler.NewSNSCatalogHandler(catalogQ)
	}

	// ✅ NEW: companies/brands (GET only)
	if companyUC != nil {
		companyHandler = snshandler.NewSNSCompanyHandler(companyUC)
	}
	if brandUC != nil {
		brandHandler = snshandler.NewSNSBrandHandler(brandUC)
	}

	// tokenBlueprint patch handler
	if tokenBlueprintUC != nil {
		tokenBlueprintHandler = snshandler.NewSNSTokenBlueprintHandler(tokenBlueprintUC)

		if nameResolver != nil {
			setOptionalResolverField(tokenBlueprintHandler, "BrandNameResolver", nameResolver)
			setOptionalResolverField(tokenBlueprintHandler, "CompanyNameResolver", nameResolver)
			setOptionalResolverField(tokenBlueprintHandler, "NameResolver", nameResolver) // 将来用
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

		// ✅ onboarding handlers are injected at RegisterSNSFromContainer() best-effort
		User:            nil,
		ShippingAddress: nil,
		BillingAddress:  nil,
		Avatar:          nil,
	}
}

// RegisterSNSFromContainer registers SNS routes using *Container.
func RegisterSNSFromContainer(mux *http.ServeMux, cont *Container) {
	if mux == nil || cont == nil {
		return
	}

	depsAny := any(cont.RouterDeps())

	// ✅ try to obtain catalog query from Container without touching RouterDeps fields.
	var catalogQ *snsquery.SNSCatalogQuery
	{
		if x, ok := any(cont).(interface {
			SNSCatalogQuery() *snsquery.SNSCatalogQuery
		}); ok {
			catalogQ = x.SNSCatalogQuery()
		} else if x, ok := any(cont).(interface {
			GetSNSCatalogQuery() *snsquery.SNSCatalogQuery
		}); ok {
			catalogQ = x.GetSNSCatalogQuery()
		} else if x, ok := any(cont).(interface {
			CatalogQuery() *snsquery.SNSCatalogQuery
		}); ok {
			catalogQ = x.CatalogQuery()
		} else if x, ok := any(cont).(interface {
			SNSCatalogQ() *snsquery.SNSCatalogQuery
		}); ok {
			catalogQ = x.SNSCatalogQ()
		}
	}

	// ✅ SNS name resolver
	var nameResolver *appresolver.NameResolver
	{
		if x, ok := any(cont).(interface {
			SNSNameResolver() *appresolver.NameResolver
		}); ok {
			nameResolver = x.SNSNameResolver()
		} else if x, ok := any(cont).(interface {
			GetSNSNameResolver() *appresolver.NameResolver
		}); ok {
			nameResolver = x.GetSNSNameResolver()
		}

		if nameResolver == nil {
			nameResolver = getSNSNameResolverFieldBestEffort(cont)
		}
	}

	// ✅ try to obtain CompanyUsecase / BrandUsecase from Container (best-effort)
	var companyUC *usecase.CompanyUsecase
	{
		if x, ok := any(cont).(interface {
			CompanyUsecase() *usecase.CompanyUsecase
		}); ok {
			companyUC = x.CompanyUsecase()
		} else if x, ok := any(cont).(interface {
			GetCompanyUsecase() *usecase.CompanyUsecase
		}); ok {
			companyUC = x.GetCompanyUsecase()
		}
	}
	var brandUC *usecase.BrandUsecase
	{
		if x, ok := any(cont).(interface {
			BrandUsecase() *usecase.BrandUsecase
		}); ok {
			brandUC = x.BrandUsecase()
		} else if x, ok := any(cont).(interface {
			GetBrandUsecase() *usecase.BrandUsecase
		}); ok {
			brandUC = x.GetBrandUsecase()
		}
	}

	// ✅ NEW: obtain onboarding usecases from Container (best-effort)
	var userUCFromCont *usecase.UserUsecase
	{
		if x, ok := any(cont).(interface {
			UserUsecase() *usecase.UserUsecase
		}); ok {
			userUCFromCont = x.UserUsecase()
		} else if x, ok := any(cont).(interface {
			GetUserUsecase() *usecase.UserUsecase
		}); ok {
			userUCFromCont = x.GetUserUsecase()
		}
	}
	var shipUCFromCont *usecase.ShippingAddressUsecase
	{
		if x, ok := any(cont).(interface {
			ShippingAddressUsecase() *usecase.ShippingAddressUsecase
		}); ok {
			shipUCFromCont = x.ShippingAddressUsecase()
		} else if x, ok := any(cont).(interface {
			GetShippingAddressUsecase() *usecase.ShippingAddressUsecase
		}); ok {
			shipUCFromCont = x.GetShippingAddressUsecase()
		}
	}
	var billUCFromCont *usecase.BillingAddressUsecase
	{
		if x, ok := any(cont).(interface {
			BillingAddressUsecase() *usecase.BillingAddressUsecase
		}); ok {
			billUCFromCont = x.BillingAddressUsecase()
		} else if x, ok := any(cont).(interface {
			GetBillingAddressUsecase() *usecase.BillingAddressUsecase
		}); ok {
			billUCFromCont = x.GetBillingAddressUsecase()
		}
	}
	var avatarUCFromCont *usecase.AvatarUsecase
	{
		if x, ok := any(cont).(interface {
			AvatarUsecase() *usecase.AvatarUsecase
		}); ok {
			avatarUCFromCont = x.AvatarUsecase()
		} else if x, ok := any(cont).(interface {
			GetAvatarUsecase() *usecase.AvatarUsecase
		}); ok {
			avatarUCFromCont = x.GetAvatarUsecase()
		}
	}

	// ✅ obtain core usecases from RouterDeps
	listUC := getFieldPtr[*usecase.ListUsecase](depsAny, "ListUC", "ListUsecase")
	invUC := getFieldPtr[*usecase.InventoryUsecase](depsAny, "InventoryUC", "InventoryUsecase")
	pbUC := getFieldPtr[*usecase.ProductBlueprintUsecase](depsAny, "ProductBlueprintUC", "ProductBlueprintUsecase")
	modelUC := getFieldPtr[*usecase.ModelUsecase](depsAny, "ModelUC", "ModelUsecase")
	tokenBlueprintUC := getFieldPtr[*usecase.TokenBlueprintUsecase](depsAny, "TokenBlueprintUC", "TokenBlueprintUsecase")

	snsDeps := NewSNSDepsWithNameResolverAndOrgHandlers(
		listUC,
		invUC,
		pbUC,
		modelUC,
		tokenBlueprintUC,
		companyUC,
		brandUC,
		nameResolver,
		catalogQ,
	)

	// ✅ try to inject onboarding handlers (user/shipping/billing/avatar)
	// - prioritize Container methods
	// - fallback to RouterDeps fields (http.Handler)
	snsDeps.User = getHandlerBestEffort(cont, depsAny,
		[]string{"SNSUserHandler", "SNSUser", "UserHandler", "User"},
		[]string{"User", "UserHandler", "SNSUser", "SNSUserHandler"},
	)
	snsDeps.ShippingAddress = getHandlerBestEffort(cont, depsAny,
		[]string{"SNSShippingAddressHandler", "SNSShippingAddress", "ShippingAddressHandler", "ShippingAddress"},
		[]string{"ShippingAddress", "ShippingAddressHandler", "SNSShippingAddress", "SNSShippingAddressHandler"},
	)
	snsDeps.BillingAddress = getHandlerBestEffort(cont, depsAny,
		[]string{"SNSBillingAddressHandler", "SNSBillingAddress", "BillingAddressHandler", "BillingAddress"},
		[]string{"BillingAddress", "BillingAddressHandler", "SNSBillingAddress", "SNSBillingAddressHandler"},
	)
	snsDeps.Avatar = getHandlerBestEffort(cont, depsAny,
		[]string{"SNSAvatarHandler", "SNSAvatar", "AvatarHandler", "Avatar"},
		[]string{"Avatar", "AvatarHandler", "SNSAvatar", "SNSAvatarHandler"},
	)

	// ✅ NEW: if handler is still nil, build it from Usecase pointers (Container -> RouterDeps)
	if snsDeps.User == nil {
		uc := userUCFromCont
		if uc == nil {
			uc = getFieldPtr[*usecase.UserUsecase](depsAny, "UserUC", "UserUsecase")
		}
		if uc != nil {
			snsDeps.User = snshandler.NewUserHandler(uc)
		}
	}
	if snsDeps.ShippingAddress == nil {
		uc := shipUCFromCont
		if uc == nil {
			uc = getFieldPtr[*usecase.ShippingAddressUsecase](depsAny, "ShippingAddressUC", "ShippingAddressUsecase")
		}
		if uc != nil {
			snsDeps.ShippingAddress = snshandler.NewShippingAddressHandler(uc)
		}
	}
	if snsDeps.BillingAddress == nil {
		uc := billUCFromCont
		if uc == nil {
			uc = getFieldPtr[*usecase.BillingAddressUsecase](depsAny, "BillingAddressUC", "BillingAddressUsecase")
		}
		if uc != nil {
			snsDeps.BillingAddress = snshandler.NewBillingAddressHandler(uc)
		}
	}
	if snsDeps.Avatar == nil {
		uc := avatarUCFromCont
		if uc == nil {
			uc = getFieldPtr[*usecase.AvatarUsecase](depsAny, "AvatarUC", "AvatarUsecase")
		}
		if uc != nil {
			snsDeps.Avatar = snshandler.NewAvatarHandler(uc)
		}
	}

	RegisterSNSRoutes(mux, snsDeps)
}

// RegisterSNSRoutes registers buyer-facing routes onto mux.
func RegisterSNSRoutes(mux *http.ServeMux, deps SNSDeps) {
	if mux == nil {
		return
	}

	// ✅ IMPORTANT:
	// snshttp.Register() は /sns/** だけでなく alias (/users, /avatars, ...) も登録します。
	// ここで重ねて mux.Handle すると「multiple registrations」で panic するため、
	// 追加の alias 登録は一切しない。
	snshttp.Register(mux, snshttp.Deps{
		List:             deps.List,
		Inventory:        deps.Inventory,
		ProductBlueprint: deps.ProductBlueprint,
		Model:            deps.Model,
		Catalog:          deps.Catalog,

		TokenBlueprint: deps.TokenBlueprint,

		Company: deps.Company,
		Brand:   deps.Brand,

		User:            deps.User,
		ShippingAddress: deps.ShippingAddress,
		BillingAddress:  deps.BillingAddress,
		Avatar:          deps.Avatar,
	})
}

// setOptionalResolverField sets handler.<fieldName> = value when possible (best-effort).
func setOptionalResolverField(handler http.Handler, fieldName string, value any) {
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

// getSNSNameResolverFieldBestEffort tries to read a resolver from Container fields
func getSNSNameResolverFieldBestEffort(cont *Container) *appresolver.NameResolver {
	if cont == nil {
		return nil
	}

	rv := reflect.ValueOf(cont)
	if !rv.IsValid() {
		return nil
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}

	tryField := func(name string) *appresolver.NameResolver {
		f := rv.FieldByName(name)
		if !f.IsValid() {
			return nil
		}
		if f.Kind() == reflect.Interface {
			if f.IsNil() {
				return nil
			}
			v := f.Interface()
			if nr, ok := v.(*appresolver.NameResolver); ok {
				return nr
			}
			return nil
		}
		if f.Kind() == reflect.Pointer {
			if f.IsNil() {
				return nil
			}
			if nr, ok := f.Interface().(*appresolver.NameResolver); ok {
				return nr
			}
			return nil
		}
		return nil
	}

	for _, n := range []string{
		"SNSNameResolver",
		"SnsNameResolver",
		"snsNameResolver",
		"NameResolver",
		"nameResolver",
	} {
		if nr := tryField(n); nr != nil {
			return nr
		}
	}

	return nil
}

// getFieldPtr reads a pointer field from an arbitrary struct (or *struct) by name, best-effort.
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
		if !f.IsValid() {
			continue
		}
		if !f.CanInterface() {
			continue
		}
		if v, ok := f.Interface().(T); ok {
			return v
		}
	}

	return zero
}

// getHandlerBestEffort finds a handler from Container (methods/fields) or RouterDeps fields, best-effort.
func getHandlerBestEffort(cont *Container, depsAny any, containerMethodNames []string, depsFieldNames []string) http.Handler {
	// 1) container method
	if cont != nil {
		rv := reflect.ValueOf(cont)
		for _, mname := range containerMethodNames {
			mname = strings.TrimSpace(mname)
			if mname == "" {
				continue
			}
			m := rv.MethodByName(mname)
			if !m.IsValid() {
				continue
			}
			if m.Type().NumIn() != 0 || m.Type().NumOut() != 1 {
				continue
			}
			out := m.Call(nil)
			if len(out) != 1 {
				continue
			}
			if h, ok := out[0].Interface().(http.Handler); ok {
				return h
			}
		}

		// 2) container field
		rve := reflect.ValueOf(cont)
		if rve.IsValid() && rve.Kind() == reflect.Pointer && !rve.IsNil() {
			rve = rve.Elem()
		}
		if rve.IsValid() && rve.Kind() == reflect.Struct {
			for _, fname := range containerMethodNames {
				fname = strings.TrimSpace(fname)
				if fname == "" {
					continue
				}
				f := rve.FieldByName(fname)
				if !f.IsValid() || !f.CanInterface() {
					continue
				}
				if h, ok := f.Interface().(http.Handler); ok {
					return h
				}
			}
		}
	}

	// 3) deps field
	if depsAny != nil {
		rv := reflect.ValueOf(depsAny)
		if rv.Kind() == reflect.Interface && !rv.IsNil() {
			rv = rv.Elem()
		}
		if rv.Kind() == reflect.Pointer {
			if rv.IsNil() {
				return nil
			}
			rv = rv.Elem()
		}
		if rv.Kind() == reflect.Struct {
			for _, fname := range depsFieldNames {
				fname = strings.TrimSpace(fname)
				if fname == "" {
					continue
				}
				f := rv.FieldByName(fname)
				if !f.IsValid() || !f.CanInterface() {
					continue
				}
				if h, ok := f.Interface().(http.Handler); ok {
					return h
				}
			}
		}
	}

	return nil
}
