// backend/internal/adapters/in/http/mall/router.go
package mall

import (
	"log"
	"net/http"
)

// Deps is a buyer-facing (mall) handler set.
type Deps struct {
	List             http.Handler
	Inventory        http.Handler
	ProductBlueprint http.Handler
	Model            http.Handler
	Catalog          http.Handler
	TokenBlueprint   http.Handler // patch

	Company http.Handler
	Brand   http.Handler

	SignIn http.Handler

	User            http.Handler
	ShippingAddress http.Handler
	BillingAddress  http.Handler
	Avatar          http.Handler

	AvatarState http.Handler

	// ✅ mall only: /mall/wallets
	Wallet http.Handler

	Cart    http.Handler
	Post    http.Handler
	Payment http.Handler
	Preview http.Handler
	Order   http.Handler
}

// handleSafe registers pattern with h.
// If h is nil, it logs and registers NotFoundHandler instead (so Cloud Run won't crash).
func handleSafe(mux *http.ServeMux, pattern string, h http.Handler, name string) {
	if h == nil {
		log.Printf("[mall.router] WARN: nil handler: %s pattern=%s (registering NotFoundHandler)", name, pattern)
		h = http.NotFoundHandler()
	}
	mux.Handle(pattern, h)
}

// Register registers buyer-facing routes onto mux (mall only).
func Register(mux *http.ServeMux, deps Deps) {
	if mux == nil {
		return
	}

	// lists
	handleSafe(mux, "/mall/lists", deps.List, "List")
	handleSafe(mux, "/mall/lists/", deps.List, "List")

	// inventories
	handleSafe(mux, "/mall/inventories", deps.Inventory, "Inventory")
	handleSafe(mux, "/mall/inventories/", deps.Inventory, "Inventory")

	// product blueprints
	handleSafe(mux, "/mall/product-blueprints", deps.ProductBlueprint, "ProductBlueprint")
	handleSafe(mux, "/mall/product-blueprints/", deps.ProductBlueprint, "ProductBlueprint")

	// models
	handleSafe(mux, "/mall/models", deps.Model, "Model")
	handleSafe(mux, "/mall/models/", deps.Model, "Model")

	// catalog
	handleSafe(mux, "/mall/catalog", deps.Catalog, "Catalog")
	handleSafe(mux, "/mall/catalog/", deps.Catalog, "Catalog")

	// token blueprints
	handleSafe(mux, "/mall/token-blueprints", deps.TokenBlueprint, "TokenBlueprint")
	handleSafe(mux, "/mall/token-blueprints/", deps.TokenBlueprint, "TokenBlueprint")

	// companies / brands
	handleSafe(mux, "/mall/companies", deps.Company, "Company")
	handleSafe(mux, "/mall/companies/", deps.Company, "Company")
	handleSafe(mux, "/mall/brands", deps.Brand, "Brand")
	handleSafe(mux, "/mall/brands/", deps.Brand, "Brand")

	// sign-in
	handleSafe(mux, "/mall/sign-in", deps.SignIn, "SignIn")
	handleSafe(mux, "/mall/sign-in/", deps.SignIn, "SignIn")

	// users
	handleSafe(mux, "/mall/users", deps.User, "User")
	handleSafe(mux, "/mall/users/", deps.User, "User")

	// shipping addresses
	handleSafe(mux, "/mall/shipping-addresses", deps.ShippingAddress, "ShippingAddress")
	handleSafe(mux, "/mall/shipping-addresses/", deps.ShippingAddress, "ShippingAddress")

	// billing addresses
	handleSafe(mux, "/mall/billing-addresses", deps.BillingAddress, "BillingAddress")
	handleSafe(mux, "/mall/billing-addresses/", deps.BillingAddress, "BillingAddress")

	// avatars
	handleSafe(mux, "/mall/avatars", deps.Avatar, "Avatar")
	handleSafe(mux, "/mall/avatars/", deps.Avatar, "Avatar")

	// avatar states
	handleSafe(mux, "/mall/avatar-states", deps.AvatarState, "AvatarState")
	handleSafe(mux, "/mall/avatar-states/", deps.AvatarState, "AvatarState")

	// wallet (plural only)
	handleSafe(mux, "/mall/wallets", deps.Wallet, "Wallet")
	handleSafe(mux, "/mall/wallets/", deps.Wallet, "Wallet")

	// cart
	handleSafe(mux, "/mall/cart", deps.Cart, "Cart")
	handleSafe(mux, "/mall/cart/", deps.Cart, "Cart")

	// preview
	handleSafe(mux, "/mall/preview", deps.Preview, "Preview")
	handleSafe(mux, "/mall/preview/", deps.Preview, "Preview")

	// posts
	handleSafe(mux, "/mall/posts", deps.Post, "Post")
	handleSafe(mux, "/mall/posts/", deps.Post, "Post")

	// payment
	handleSafe(mux, "/mall/payment", deps.Payment, "Payment")
	handleSafe(mux, "/mall/payment/", deps.Payment, "Payment")
	handleSafe(mux, "/mall/payments", deps.Payment, "Payment")
	handleSafe(mux, "/mall/payments/", deps.Payment, "Payment")

	// orders
	handleSafe(mux, "/mall/orders", deps.Order, "Order")
	handleSafe(mux, "/mall/orders/", deps.Order, "Order")
}
