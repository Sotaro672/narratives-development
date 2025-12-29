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

	// ✅ name resolver endpoints (for NameResolver)
	Company http.Handler
	Brand   http.Handler

	// ✅ auth entry (cart empty ok)
	SignIn http.Handler

	// ✅ auth onboarding resources
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
// - GET /sns/companies/{id}
// - GET /sns/brands/{id}
//
// ✅ Auth entry (cart empty ok)
// - POST /sns/sign-in
//
// ✅ Auth onboarding (buyer-facing)
// - POST/GET/PATCH/DELETE /sns/users/{id?}
// - POST/GET/PATCH/DELETE /sns/shipping-addresses/{id?}
// - POST/GET/PATCH/DELETE /sns/billing-addresses/{id?}
// - POST/GET/PATCH/DELETE /sns/avatars/{id?}
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
		mux.Handle("/sns/catalog", deps.Catalog)
		mux.Handle("/sns/catalog/", deps.Catalog)
	}

	// token blueprints
	// NOTE: only patch is required now: /sns/token-blueprints/{id}/patch
	if deps.TokenBlueprint != nil {
		mux.Handle("/sns/token-blueprints", deps.TokenBlueprint)
		mux.Handle("/sns/token-blueprints/", deps.TokenBlueprint)
	}

	// companies
	// NOTE: only detail is required now: /sns/companies/{id}
	if deps.Company != nil {
		mux.Handle("/sns/companies", deps.Company)
		mux.Handle("/sns/companies/", deps.Company)
	}

	// brands
	// NOTE: only detail is required now: /sns/brands/{id}
	if deps.Brand != nil {
		mux.Handle("/sns/brands", deps.Brand)
		mux.Handle("/sns/brands/", deps.Brand)
	}

	// ✅ sign-in (cart empty ok)
	if deps.SignIn != nil {
		mux.Handle("/sns/sign-in", deps.SignIn)
		mux.Handle("/sns/sign-in/", deps.SignIn)
	}

	// users
	if deps.User != nil {
		mux.Handle("/sns/users", deps.User)
		mux.Handle("/sns/users/", deps.User)
	}

	// shipping addresses
	if deps.ShippingAddress != nil {
		mux.Handle("/sns/shipping-addresses", deps.ShippingAddress)
		mux.Handle("/sns/shipping-addresses/", deps.ShippingAddress)
	}

	// billing addresses
	if deps.BillingAddress != nil {
		mux.Handle("/sns/billing-addresses", deps.BillingAddress)
		mux.Handle("/sns/billing-addresses/", deps.BillingAddress)
	}

	// avatars
	if deps.Avatar != nil {
		mux.Handle("/sns/avatars", deps.Avatar)
		mux.Handle("/sns/avatars/", deps.Avatar)
	}
}
