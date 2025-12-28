// backend/internal/adapters/in/http/sns/router.go
package sns

import "net/http"

// Deps is a buyer-facing (sns) handler set.
type Deps struct {
	List             http.Handler
	Inventory        http.Handler
	ProductBlueprint http.Handler
	Model            http.Handler
	Catalog          http.Handler
	TokenBlueprint   http.Handler // patch

	// ✅ NEW: name resolver endpoints (for NameResolver)
	Company http.Handler
	Brand   http.Handler

	// ✅ NEW: auth onboarding resources
	User            http.Handler
	ShippingAddress http.Handler
	BillingAddress  http.Handler
	Avatar          http.Handler
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
// - GET /sns/catalog/{listId}
// - GET /sns/token-blueprints/{id}/patch
// - GET /sns/companies/{id}     ✅ NEW (name resolver)
// - GET /sns/brands/{id}        ✅ NEW (name resolver)
//
// ✅ NEW (auth onboarding; buyer-facing)
// - POST/GET /sns/users/{id?}
// - POST/GET /sns/shipping-addresses/{id?}
// - POST/GET /sns/billing-addresses/{id?}
// - POST/GET /sns/avatars/{id?}
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

	// catalog
	// NOTE: only detail is required now: /sns/catalog/{listId}
	if deps.Catalog != nil {
		mux.Handle("/sns/catalog/", deps.Catalog)
		mux.Handle("/sns/catalog", deps.Catalog)
	}

	// token blueprints
	// NOTE: only patch is required now: /sns/token-blueprints/{id}/patch
	if deps.TokenBlueprint != nil {
		mux.Handle("/sns/token-blueprints/", deps.TokenBlueprint)
		mux.Handle("/sns/token-blueprints", deps.TokenBlueprint)
	}

	// companies ✅ NEW
	// NOTE: only detail is required now: /sns/companies/{id}
	if deps.Company != nil {
		mux.Handle("/sns/companies/", deps.Company)
		mux.Handle("/sns/companies", deps.Company)
	}

	// brands ✅ NEW
	// NOTE: only detail is required now: /sns/brands/{id}
	if deps.Brand != nil {
		mux.Handle("/sns/brands/", deps.Brand)
		mux.Handle("/sns/brands", deps.Brand)
	}

	// users ✅ NEW
	if deps.User != nil {
		mux.Handle("/sns/users", deps.User)
		mux.Handle("/sns/users/", deps.User)
	}

	// shipping addresses ✅ NEW
	if deps.ShippingAddress != nil {
		mux.Handle("/sns/shipping-addresses", deps.ShippingAddress)
		mux.Handle("/sns/shipping-addresses/", deps.ShippingAddress)
	}

	// billing addresses ✅ NEW
	if deps.BillingAddress != nil {
		mux.Handle("/sns/billing-addresses", deps.BillingAddress)
		mux.Handle("/sns/billing-addresses/", deps.BillingAddress)
	}

	// avatars ✅ NEW
	if deps.Avatar != nil {
		mux.Handle("/sns/avatars", deps.Avatar)
		mux.Handle("/sns/avatars/", deps.Avatar)
	}
}
