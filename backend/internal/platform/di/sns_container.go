// backend/internal/platform/di/sns_container.go
package di

import (
	"net/http"

	snshttp "narratives/internal/adapters/in/http/sns"
	snshandler "narratives/internal/adapters/in/http/sns/handler"
	usecase "narratives/internal/application/usecase"
)

// SNSDeps is a buyer-facing (sns) HTTP dependency set.
type SNSDeps struct {
	// Handlers
	List             http.Handler
	Inventory        http.Handler
	ProductBlueprint http.Handler // ✅ NEW
}

// NewSNSDeps wires SNS handlers.
//
// SNS は companyId 境界が無い（公開）ため、console 用 query は使わない。
func NewSNSDeps(
	listUC *usecase.ListUsecase,
	invUC *usecase.InventoryUsecase,
	pbUC *usecase.ProductBlueprintUsecase, // ✅ NEW
) SNSDeps {
	var listHandler http.Handler
	var invHandler http.Handler
	var pbHandler http.Handler

	if listUC != nil {
		listHandler = snshandler.NewSNSListHandler(listUC)
	}

	if invUC != nil {
		invHandler = snshandler.NewSNSInventoryHandler(invUC)
	}

	if pbUC != nil {
		pbHandler = snshandler.NewSNSProductBlueprintHandler(pbUC) // ✅ NEW
	}

	return SNSDeps{
		List:             listHandler,
		Inventory:        invHandler,
		ProductBlueprint: pbHandler,
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

	snsDeps := NewSNSDeps(
		deps.ListUC,
		deps.InventoryUC,
		deps.ProductBlueprintUC,
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
		ProductBlueprint: deps.ProductBlueprint, // ✅ NEW
	})
}
