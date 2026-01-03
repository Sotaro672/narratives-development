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

	// ✅ NEW: avatar state (follower/following/post counts)
	AvatarState http.Handler

	// ✅ NEW: wallet (tokens)
	Wallet http.Handler

	// ✅ cart (GET is read-model internally; write routes handled by same handler)
	// NOTE: CartQuery route is removed; Cart handler must internally dispatch GET /sns/cart to cart_query DTO.
	Cart http.Handler

	// ✅ NEW: posts
	Post http.Handler

	// ✅ NEW: payment (order context / checkout)
	Payment http.Handler

	// ✅ NEW: preview
	Preview http.Handler
}

// Register registers buyer-facing routes onto mux.
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
	if deps.Catalog != nil {
		mux.Handle("/sns/catalog", deps.Catalog)
		mux.Handle("/sns/catalog/", deps.Catalog)
	}

	// token blueprints
	if deps.TokenBlueprint != nil {
		mux.Handle("/sns/token-blueprints", deps.TokenBlueprint)
		mux.Handle("/sns/token-blueprints/", deps.TokenBlueprint)
	}

	// companies
	if deps.Company != nil {
		mux.Handle("/sns/companies", deps.Company)
		mux.Handle("/sns/companies/", deps.Company)
	}

	// brands
	if deps.Brand != nil {
		mux.Handle("/sns/brands", deps.Brand)
		mux.Handle("/sns/brands/", deps.Brand)
	}

	// sign-in
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

	// avatarStates
	if deps.AvatarState != nil {
		mux.Handle("/sns/avatar-states", deps.AvatarState)
		mux.Handle("/sns/avatar-states/", deps.AvatarState)
	}

	// wallet
	if deps.Wallet != nil {
		mux.Handle("/sns/wallet", deps.Wallet)
		mux.Handle("/sns/wallet/", deps.Wallet)
	}

	// ✅ cart (single handler; GET /sns/cart returns read-model DTO internally)
	if deps.Cart != nil {
		mux.Handle("/sns/cart", deps.Cart)
		mux.Handle("/sns/cart/", deps.Cart)
	}

	// preview
	if deps.Preview != nil {
		mux.Handle("/sns/preview", deps.Preview)
		mux.Handle("/sns/preview/", deps.Preview)
	}

	// posts
	if deps.Post != nil {
		mux.Handle("/sns/posts", deps.Post)
		mux.Handle("/sns/posts/", deps.Post)
	}

	// payment
	if deps.Payment != nil {
		mux.Handle("/sns/payment", deps.Payment)
		mux.Handle("/sns/payment/", deps.Payment)
	}
}
