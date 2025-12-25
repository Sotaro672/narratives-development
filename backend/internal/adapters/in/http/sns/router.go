// backend\internal\adapters\in\http\sns\router.go
package sns

import "net/http"

// Deps is a buyer-facing (sns) handler set.
type Deps struct {
	List             http.Handler
	Inventory        http.Handler
	ProductBlueprint http.Handler // ✅ NEW
	Model            http.Handler // ✅ NEW
	Catalog          http.Handler // ✅ NEW
}

// Register registers buyer-facing routes onto mux.
//
// Routes:
// - GET /sns/lists
// - GET /sns/lists/{id}
// - GET /sns/inventories?productBlueprintId=&tokenBlueprintId=
// - GET /sns/inventories/{id}
// - GET /sns/product-blueprints/{id}
// - GET /sns/models?productBlueprintId=
// - GET /sns/models/{id}
// - GET /sns/catalog/{listId}                ✅ NEW
func Register(mux *http.ServeMux, deps Deps) {
	if mux == nil {
		return
	}

	// lists
	if deps.List != nil {
		mux.Handle("/sns/lists", deps.List)
		mux.Handle("/sns/lists/", deps.List)
	}

	// inventories
	if deps.Inventory != nil {
		mux.Handle("/sns/inventories", deps.Inventory)
		mux.Handle("/sns/inventories/", deps.Inventory)
	}

	// product blueprints
	if deps.ProductBlueprint != nil {
		mux.Handle("/sns/product-blueprints", deps.ProductBlueprint)
		mux.Handle("/sns/product-blueprints/", deps.ProductBlueprint)
	}

	// models
	if deps.Model != nil {
		mux.Handle("/sns/models", deps.Model)
		mux.Handle("/sns/models/", deps.Model)
	}

	// catalog ✅ NEW
	// NOTE: only detail is required now: /sns/catalog/{listId}
	if deps.Catalog != nil {
		mux.Handle("/sns/catalog/", deps.Catalog)
		// （必要なら将来 /sns/catalog を index に使う）
		mux.Handle("/sns/catalog", deps.Catalog)
	}
}
