// backend/internal/platform/di/sns_container.go
package di

import (
	"net/http"

	snshttp "narratives/internal/adapters/in/http/sns"
	snshandler "narratives/internal/adapters/in/http/sns/handler"
	snsquery "narratives/internal/application/query/sns"
	usecase "narratives/internal/application/usecase"
)

// SNSDeps is a buyer-facing (sns) HTTP dependency set.
type SNSDeps struct {
	// Handlers
	List             http.Handler
	Inventory        http.Handler
	ProductBlueprint http.Handler // ✅ NEW
	Model            http.Handler // ✅ NEW
	Catalog          http.Handler // ✅ NEW
}

// NewSNSDeps wires SNS handlers.
//
// SNS は companyId 境界が無い（公開）ため、console 用 query は使わない。
func NewSNSDeps(
	listUC *usecase.ListUsecase,
	invUC *usecase.InventoryUsecase,
	pbUC *usecase.ProductBlueprintUsecase, // ✅ NEW
	modelUC *usecase.ModelUsecase, // ✅ NEW

	// ✅ NEW: catalog query
	catalogQ *snsquery.SNSCatalogQuery,
) SNSDeps {
	var listHandler http.Handler
	var invHandler http.Handler
	var pbHandler http.Handler
	var modelHandler http.Handler
	var catalogHandler http.Handler

	if listUC != nil {
		listHandler = snshandler.NewSNSListHandler(listUC)
	}

	if invUC != nil {
		invHandler = snshandler.NewSNSInventoryHandler(invUC)
	}

	if pbUC != nil {
		pbHandler = snshandler.NewSNSProductBlueprintHandler(pbUC) // ✅ NEW
	}

	if modelUC != nil {
		modelHandler = snshandler.NewSNSModelHandler(modelUC) // ✅ NEW
	}

	if catalogQ != nil {
		catalogHandler = snshandler.NewSNSCatalogHandler(catalogQ) // ✅ NEW
	}

	return SNSDeps{
		List:             listHandler,
		Inventory:        invHandler,
		ProductBlueprint: pbHandler,
		Model:            modelHandler,
		Catalog:          catalogHandler,
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

	// ✅ NEW: try to obtain catalog query from Container without touching RouterDeps fields.
	// （RouterDeps に ListRepo/ModelRepo 等が無いので、ここで作れないため）
	var catalogQ *snsquery.SNSCatalogQuery
	{
		// Prefer: func (c *Container) SNSCatalogQuery() *snsquery.SNSCatalogQuery
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

	snsDeps := NewSNSDeps(
		deps.ListUC,
		deps.InventoryUC,
		deps.ProductBlueprintUC,
		deps.ModelUC, // ✅ NEW
		catalogQ,     // ✅ NEW
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
		Catalog:          deps.Catalog, // ✅ NEW
	})
}
