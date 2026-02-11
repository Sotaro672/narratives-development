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

	// ✅ /mall/avatars (POST create) + /mall/avatars/{id} (GET/PATCH/DELETE)
	Avatar http.Handler

	// ✅ /mall/me/avatar (resolve avatarId by current user uid)
	MeAvatar http.Handler

	AvatarState http.Handler

	// ✅ mall only: /mall/me/wallets
	Wallet http.Handler

	Cart    http.Handler
	Payment http.Handler

	Preview   http.Handler
	PreviewMe http.Handler

	OrderScanVerify   http.Handler
	OrderScanTransfer http.Handler

	OwnerResolve http.Handler

	Order http.Handler

	Invoice http.Handler
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

// handleSafeAuth registers pattern with auth-wrapped handler.
// If auth is nil, it falls back to plain handleSafe (and warns) to avoid crash.
func handleSafeAuth(mux *http.ServeMux, pattern string, h http.Handler, name string, auth func(http.Handler) http.Handler) {
	if auth == nil {
		log.Printf("[mall.router] WARN: nil auth middleware: %s pattern=%s (registering WITHOUT auth)", name, pattern)
		handleSafe(mux, pattern, h, name)
		return
	}
	handleSafe(mux, pattern, auth(h), name)
}

// Register registers buyer-facing routes onto mux (mall only).
//
// auth:
//   - /mall/me/** のみ auth を必須にするための middleware wrapper
//   - 例: auth := userAuthMiddleware.Handler
func Register(mux *http.ServeMux, deps Deps, auth func(http.Handler) http.Handler) {
	if mux == nil {
		return
	}

	// ------------------------------------------------------------
	// Public routes (no auth)
	// ------------------------------------------------------------

	// lists (public)
	handleSafe(mux, "/mall/lists", deps.List, "List")
	handleSafe(mux, "/mall/lists/", deps.List, "List")

	// inventories (public)
	handleSafe(mux, "/mall/inventories", deps.Inventory, "Inventory")
	handleSafe(mux, "/mall/inventories/", deps.Inventory, "Inventory")

	// product blueprints (public)
	handleSafe(mux, "/mall/product-blueprints", deps.ProductBlueprint, "ProductBlueprint")
	handleSafe(mux, "/mall/product-blueprints/", deps.ProductBlueprint, "ProductBlueprint")

	// models (public)
	handleSafe(mux, "/mall/models", deps.Model, "Model")
	handleSafe(mux, "/mall/models/", deps.Model, "Model")

	// catalog (public)
	handleSafe(mux, "/mall/catalog", deps.Catalog, "Catalog")
	handleSafe(mux, "/mall/catalog/", deps.Catalog, "Catalog")

	// token blueprints (public)
	handleSafe(mux, "/mall/token-blueprints", deps.TokenBlueprint, "TokenBlueprint")
	handleSafe(mux, "/mall/token-blueprints/", deps.TokenBlueprint, "TokenBlueprint")

	// companies / brands (public)
	handleSafe(mux, "/mall/companies", deps.Company, "Company")
	handleSafe(mux, "/mall/companies/", deps.Company, "Company")

	handleSafe(mux, "/mall/brands", deps.Brand, "Brand")
	handleSafe(mux, "/mall/brands/", deps.Brand, "Brand")

	// sign-in (public)
	handleSafe(mux, "/mall/sign-in", deps.SignIn, "SignIn")
	handleSafe(mux, "/mall/sign-in/", deps.SignIn, "SignIn")

	// avatars (public)
	handleSafe(mux, "/mall/avatars", deps.Avatar, "Avatar")
	handleSafe(mux, "/mall/avatars/", deps.Avatar, "Avatar")

	// owner resolve (public OK)
	handleSafe(mux, "/mall/owners/resolve", deps.OwnerResolve, "OwnerResolve")
	handleSafe(mux, "/mall/owners/resolve/", deps.OwnerResolve, "OwnerResolve")

	// preview (public)
	handleSafe(mux, "/mall/preview", deps.Preview, "Preview")
	handleSafe(mux, "/mall/preview/", deps.Preview, "Preview")

	// ------------------------------------------------------------
	// Auth-required routes (/mall/me/**)
	// ------------------------------------------------------------

	// lists (me)
	handleSafeAuth(mux, "/mall/me/lists", deps.List, "List(me)", auth)
	handleSafeAuth(mux, "/mall/me/lists/", deps.List, "List(me)", auth)

	// inventories (me)
	handleSafeAuth(mux, "/mall/me/inventories", deps.Inventory, "Inventory(me)", auth)
	handleSafeAuth(mux, "/mall/me/inventories/", deps.Inventory, "Inventory(me)", auth)

	// product blueprints (me)
	handleSafeAuth(mux, "/mall/me/product-blueprints", deps.ProductBlueprint, "ProductBlueprint(me)", auth)
	handleSafeAuth(mux, "/mall/me/product-blueprints/", deps.ProductBlueprint, "ProductBlueprint(me)", auth)

	// models (me)
	handleSafeAuth(mux, "/mall/me/models", deps.Model, "Model(me)", auth)
	handleSafeAuth(mux, "/mall/me/models/", deps.Model, "Model(me)", auth)

	// catalog (me)
	handleSafeAuth(mux, "/mall/me/catalog", deps.Catalog, "Catalog(me)", auth)
	handleSafeAuth(mux, "/mall/me/catalog/", deps.Catalog, "Catalog(me)", auth)

	// token blueprints (me)
	handleSafeAuth(mux, "/mall/me/token-blueprints", deps.TokenBlueprint, "TokenBlueprint(me)", auth)
	handleSafeAuth(mux, "/mall/me/token-blueprints/", deps.TokenBlueprint, "TokenBlueprint(me)", auth)

	// companies / brands (me)
	handleSafeAuth(mux, "/mall/me/companies", deps.Company, "Company(me)", auth)
	handleSafeAuth(mux, "/mall/me/companies/", deps.Company, "Company(me)", auth)

	handleSafeAuth(mux, "/mall/me/brands", deps.Brand, "Brand(me)", auth)
	handleSafeAuth(mux, "/mall/me/brands/", deps.Brand, "Brand(me)", auth)

	// users ✅ A案: "me" のみ（token uid を必須にする）
	handleSafeAuth(mux, "/mall/me/users", deps.User, "User(me)", auth)
	handleSafeAuth(mux, "/mall/me/users/", deps.User, "User(me)", auth)

	// shipping addresses ✅ A案: "me" のみ
	handleSafeAuth(mux, "/mall/me/shipping-addresses", deps.ShippingAddress, "ShippingAddress(me)", auth)
	handleSafeAuth(mux, "/mall/me/shipping-addresses/", deps.ShippingAddress, "ShippingAddress(me)", auth)

	// billing addresses ✅ A案: "me" のみ
	handleSafeAuth(mux, "/mall/me/billing-addresses", deps.BillingAddress, "BillingAddress(me)", auth)
	handleSafeAuth(mux, "/mall/me/billing-addresses/", deps.BillingAddress, "BillingAddress(me)", auth)

	// me avatar
	handleSafeAuth(mux, "/mall/me/avatars", deps.MeAvatar, "MeAvatar", auth)
	handleSafeAuth(mux, "/mall/me/avatars/", deps.MeAvatar, "MeAvatar", auth)

	// avatar states (me)
	handleSafeAuth(mux, "/mall/me/avatar-states", deps.AvatarState, "AvatarState(me)", auth)
	handleSafeAuth(mux, "/mall/me/avatar-states/", deps.AvatarState, "AvatarState(me)", auth)

	// wallet (me)
	handleSafeAuth(mux, "/mall/me/wallets", deps.Wallet, "Wallet(me)", auth)
	handleSafeAuth(mux, "/mall/me/wallets/", deps.Wallet, "Wallet(me)", auth)

	// cart (me)
	handleSafeAuth(mux, "/mall/me/cart", deps.Cart, "Cart(me)", auth)
	handleSafeAuth(mux, "/mall/me/cart/", deps.Cart, "Cart(me)", auth)

	// preview (me)
	handleSafeAuth(mux, "/mall/me/preview", deps.PreviewMe, "Preview(me)", auth)
	handleSafeAuth(mux, "/mall/me/preview/", deps.PreviewMe, "Preview(me)", auth)

	// order scan verify (me)
	handleSafeAuth(mux, "/mall/me/orders/scan/verify", deps.OrderScanVerify, "OrderScanVerify(me)", auth)
	handleSafeAuth(mux, "/mall/me/orders/scan/verify/", deps.OrderScanVerify, "OrderScanVerify(me)", auth)

	// order scan transfer (me)
	handleSafeAuth(mux, "/mall/me/orders/scan/transfer", deps.OrderScanTransfer, "OrderScanTransfer(me)", auth)
	handleSafeAuth(mux, "/mall/me/orders/scan/transfer/", deps.OrderScanTransfer, "OrderScanTransfer(me)", auth)

	// payment (me)
	handleSafeAuth(mux, "/mall/me/payment", deps.Payment, "Payment(me)", auth)
	handleSafeAuth(mux, "/mall/me/payment/", deps.Payment, "Payment(me)", auth)
	handleSafeAuth(mux, "/mall/me/payments", deps.Payment, "Payment(me)", auth)
	handleSafeAuth(mux, "/mall/me/payments/", deps.Payment, "Payment(me)", auth)

	// invoices (me)
	handleSafeAuth(mux, "/mall/me/invoices", deps.Invoice, "Invoice(me)", auth)
	handleSafeAuth(mux, "/mall/me/invoices/", deps.Invoice, "Invoice(me)", auth)

	// orders (me)
	handleSafeAuth(mux, "/mall/me/orders", deps.Order, "Order(me)", auth)
	handleSafeAuth(mux, "/mall/me/orders/", deps.Order, "Order(me)", auth)
}
