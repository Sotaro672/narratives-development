// backend/internal/platform/di/sns_container.go
package di

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"

	snshttp "narratives/internal/adapters/in/http/sns"
	snshandler "narratives/internal/adapters/in/http/sns/handler"
	snsquery "narratives/internal/application/query/sns"
	snsdto "narratives/internal/application/query/sns/dto"
	appresolver "narratives/internal/application/resolver"
	usecase "narratives/internal/application/usecase"
	pbdom "narratives/internal/domain/productBlueprint"
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
	//
	// 重要:
	// - ここでは「確実に存在する ctor」だけを呼ぶ（= NewSNSTokenBlueprintHandler）。
	// - brand/company 名解決や iconUrl 補完は、handler にフィールドがあれば reflection で注入する。
	if tokenBlueprintUC != nil {
		tokenBlueprintHandler = snshandler.NewSNSTokenBlueprintHandler(tokenBlueprintUC)

		// brand/company name resolver（handler 側にフィールドがあれば入る）
		if nameResolver != nil {
			setOptionalResolverField(tokenBlueprintHandler, "BrandNameResolver", nameResolver)
			setOptionalResolverField(tokenBlueprintHandler, "CompanyNameResolver", nameResolver)
			setOptionalResolverField(tokenBlueprintHandler, "NameResolver", nameResolver) // 将来用
		}

		// icon url resolver（handler 側にフィールドがあれば入る）
		// - env: TOKEN_ICON_PUBLIC_BUCKET を使う（resolver 実装に従う）
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

// RegisterSNSFromContainer registers SNS routes using *Container.
// RouterDeps 型に依存しないため、main.go 側が SNS の依存増減を意識しなくてよい。
func RegisterSNSFromContainer(mux *http.ServeMux, cont *Container) {
	if mux == nil || cont == nil {
		return
	}

	// cont.RouterDeps() の戻り値が「無名struct」でもここでは受けられる（型名不要）
	deps := cont.RouterDeps()

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

	// ✅ SNS name resolver (brandName/companyName)
	// - prefer: SNSNameResolver()
	// - fallback: GetSNSNameResolver()
	// - fallback: cont.NameResolver field
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

		// final fallback: field (di package can access)
		if nameResolver == nil && cont.NameResolver != nil {
			nameResolver = cont.NameResolver
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

	snsDeps := NewSNSDepsWithNameResolverAndOrgHandlers(
		deps.ListUC,
		deps.InventoryUC,
		deps.ProductBlueprintUC,
		deps.ModelUC,
		deps.TokenBlueprintUC,
		companyUC,
		brandUC,
		nameResolver,
		catalogQ,
	)
	RegisterSNSRoutes(mux, snsDeps)
}

// RegisterSNSRoutes registers buyer-facing routes onto mux.
func RegisterSNSRoutes(mux *http.ServeMux, deps SNSDeps) {
	if mux == nil {
		return
	}
	snshttp.Register(mux, snshttp.Deps{
		List:             deps.List,
		Inventory:        deps.Inventory,
		ProductBlueprint: deps.ProductBlueprint,
		Model:            deps.Model,
		Catalog:          deps.Catalog,

		TokenBlueprint: deps.TokenBlueprint,

		Company: deps.Company,
		Brand:   deps.Brand,
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

// ============================================================
// sns catalog adapters (DI-only helpers)
// - compile-time で inventory domain 型に依存しないため、reflection で吸収する
// - moved from container.go
// ============================================================

type snsCatalogInventoryRepoAdapter struct {
	repo any
}

func (a *snsCatalogInventoryRepoAdapter) GetByID(ctx context.Context, id string) (*snsdto.SNSCatalogInventoryDTO, error) {
	if a == nil || a.repo == nil {
		return nil, errors.New("sns catalog inventory repo: repo is nil")
	}
	v, err := callRepo(a.repo, []string{"GetByID", "GetById"}, ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	return toSNSCatalogInventoryDTO(v)
}

func (a *snsCatalogInventoryRepoAdapter) GetByProductAndTokenBlueprintID(ctx context.Context, productBlueprintID, tokenBlueprintID string) (*snsdto.SNSCatalogInventoryDTO, error) {
	if a == nil || a.repo == nil {
		return nil, errors.New("sns catalog inventory repo: repo is nil")
	}
	pb := strings.TrimSpace(productBlueprintID)
	tb := strings.TrimSpace(tokenBlueprintID)

	methods := []string{
		"GetByProductAndTokenBlueprintID",
		"GetByProductAndTokenBlueprintId",
		"GetByProductAndTokenBlueprintIDs",
		"GetByProductAndTokenBlueprintIds",
	}
	v, err := callRepo(a.repo, methods, ctx, pb, tb)
	if err != nil {
		return nil, err
	}
	return toSNSCatalogInventoryDTO(v)
}

type snsCatalogProductBlueprintRepoAdapter struct {
	repo any
}

func (a *snsCatalogProductBlueprintRepoAdapter) GetByID(ctx context.Context, id string) (*pbdom.ProductBlueprint, error) {
	if a == nil || a.repo == nil {
		return nil, errors.New("sns catalog product repo: repo is nil")
	}
	v, err := callRepo(a.repo, []string{"GetByID", "GetById"}, ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, errors.New("productBlueprint is nil")
	}
	if pb, ok := v.(*pbdom.ProductBlueprint); ok {
		return pb, nil
	}
	if pb, ok := v.(pbdom.ProductBlueprint); ok {
		cp := pb
		return &cp, nil
	}

	rv := reflect.ValueOf(v)
	if rv.IsValid() && rv.Kind() == reflect.Pointer && !rv.IsNil() {
		if x, ok := rv.Interface().(*pbdom.ProductBlueprint); ok {
			return x, nil
		}
	}
	return nil, errors.New("unexpected productBlueprint type")
}

func callRepo(repo any, methodNames []string, args ...any) (any, error) {
	rv := reflect.ValueOf(repo)
	if !rv.IsValid() {
		return nil, errors.New("repo is invalid")
	}

	for _, name := range methodNames {
		m := rv.MethodByName(name)
		if !m.IsValid() {
			continue
		}

		in := make([]reflect.Value, 0, len(args))
		for _, a := range args {
			in = append(in, reflect.ValueOf(a))
		}

		out := m.Call(in)
		if len(out) == 0 {
			return nil, nil
		}

		if len(out) >= 2 {
			if e, ok := out[len(out)-1].Interface().(error); ok && e != nil {
				return nil, e
			}
		}
		return out[0].Interface(), nil
	}

	return nil, errors.New("method not found on repo")
}

func toSNSCatalogInventoryDTO(v any) (*snsdto.SNSCatalogInventoryDTO, error) {
	if v == nil {
		return nil, errors.New("inventory is nil")
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return nil, errors.New("inventory is invalid")
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, errors.New("inventory is nil")
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, errors.New("inventory is not struct")
	}

	getStr := func(names ...string) string {
		for _, n := range names {
			f := rv.FieldByName(n)
			if !f.IsValid() {
				continue
			}
			if f.Kind() == reflect.String {
				return strings.TrimSpace(f.String())
			}
		}
		return ""
	}

	id := getStr("ID", "Id", "InventoryID", "InventoryId")
	pbID := getStr("ProductBlueprintID", "ProductBlueprintId", "productBlueprintId")
	tbID := getStr("TokenBlueprintID", "TokenBlueprintId", "tokenBlueprintId")

	stock := map[string]snsdto.SNSCatalogInventoryModelStockDTO{}

	var sf reflect.Value
	for _, n := range []string{"Stock", "Stocks", "stock"} {
		f := rv.FieldByName(n)
		if f.IsValid() {
			sf = f
			break
		}
	}

	if sf.IsValid() {
		if sf.Kind() == reflect.Pointer {
			if !sf.IsNil() {
				sf = sf.Elem()
			}
		}

		switch sf.Kind() {
		case reflect.Map:
			if sf.Type().Key().Kind() == reflect.String {
				iter := sf.MapRange()
				for iter.Next() {
					modelID := strings.TrimSpace(iter.Key().String())
					if modelID == "" {
						continue
					}

					ids := extractProductIDsFromStockValue(iter.Value())
					products := make(map[string]bool, len(ids))
					for _, pid := range ids {
						pid = strings.TrimSpace(pid)
						if pid == "" {
							continue
						}
						products[pid] = true
					}

					stock[modelID] = snsdto.SNSCatalogInventoryModelStockDTO{
						Products: products,
					}
				}
			}

		case reflect.Slice, reflect.Array:
			for i := 0; i < sf.Len(); i++ {
				it := sf.Index(i)
				if it.Kind() == reflect.Interface && !it.IsNil() {
					it = it.Elem()
				}
				if it.Kind() == reflect.Pointer && !it.IsNil() {
					it = it.Elem()
				}
				if !it.IsValid() {
					continue
				}
				if it.Kind() == reflect.String {
					modelID := strings.TrimSpace(it.String())
					if modelID == "" {
						continue
					}
					if _, ok := stock[modelID]; !ok {
						stock[modelID] = snsdto.SNSCatalogInventoryModelStockDTO{Products: map[string]bool{}}
					}
				}
			}
		}
	}

	return &snsdto.SNSCatalogInventoryDTO{
		ID:                 id,
		ProductBlueprintID: pbID,
		TokenBlueprintID:   tbID,
		Stock:              stock,
	}, nil
}

// ------------------------------------------------------------
// stock reflection helpers (modelId -> products)
// ------------------------------------------------------------

func extractProductIDsFromStockValue(v reflect.Value) []string {
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Interface && !v.IsNil() {
		v = v.Elem()
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		pf := v.FieldByName("Products")
		if pf.IsValid() {
			return extractStringIDs(pf)
		}
	}

	return extractStringIDs(v)
}

func extractStringIDs(v reflect.Value) []string {
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Interface && !v.IsNil() {
		v = v.Elem()
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		out := make([]string, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			e := v.Index(i)
			if e.Kind() == reflect.Interface && !e.IsNil() {
				e = e.Elem()
			}
			if e.Kind() == reflect.Pointer {
				if e.IsNil() {
					continue
				}
				e = e.Elem()
			}
			if e.Kind() == reflect.String {
				s := strings.TrimSpace(e.String())
				if s != "" {
					out = append(out, s)
				}
			}
		}
		return out

	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			return nil
		}
		out := make([]string, 0, v.Len())
		iter := v.MapRange()
		for iter.Next() {
			k := iter.Key()
			if k.Kind() != reflect.String {
				continue
			}
			s := strings.TrimSpace(k.String())
			if s != "" {
				out = append(out, s)
			}
		}
		return out

	default:
		return nil
	}
}
