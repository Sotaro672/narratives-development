// backend/internal/platform/di/sns_container.go
package di

import (
	"net/http"

	snshttp "narratives/internal/adapters/in/http/sns"
	snshandler "narratives/internal/adapters/in/http/sns/handler"
	usecase "narratives/internal/application/usecase"
)

// SNSDeps is a buyer-facing (sns) HTTP dependency set.
// Keep it minimal and independent from console/admin routing.
type SNSDeps struct {
	// Handlers
	List      http.Handler
	Inventory http.Handler
}

// NewSNSDeps wires SNS handlers.
//
// SNS は companyId 境界が無い（公開）ため、console 用 query は使わない。
func NewSNSDeps(
	listUC *usecase.ListUsecase,
	invUC *usecase.InventoryUsecase,
) SNSDeps {
	var listHandler http.Handler
	var invHandler http.Handler

	if listUC != nil {
		listHandler = snshandler.NewSNSListHandler(listUC)
	}

	if invUC != nil {
		invHandler = snshandler.NewSNSInventoryHandler(invUC)
	}

	return SNSDeps{
		List:      listHandler,
		Inventory: invHandler,
	}
}

// RegisterSNSRoutes registers buyer-facing routes onto mux.
func RegisterSNSRoutes(mux *http.ServeMux, deps SNSDeps) {
	if mux == nil {
		return
	}
	snshttp.Register(mux, snshttp.Deps{
		List:      deps.List,
		Inventory: deps.Inventory,
	})
}
