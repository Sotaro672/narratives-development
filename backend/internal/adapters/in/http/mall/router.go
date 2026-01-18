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
	// Contract (new only; legacy removed):
	// - GET  /mall/me/wallets        : return { wallets: [ wallet ] }
	// - POST /mall/me/wallets/sync   : sync on-chain -> persist -> return { wallets: [ wallet ] }
	//
	// NOTE:
	// - avatarId is derived from AvatarContextMiddleware (uid -> avatarId) and NOT from query/path.
	// - walletAddress is derived from wallet doc (docId=avatarId) and NOT from query/path.
	Wallet http.Handler

	Cart    http.Handler
	Payment http.Handler

	// ✅ Preview routes are intentionally split:
	// - /mall/preview     : public (no auth) for pre-login scanner
	// - /mall/me/preview  : authenticated for post-login scanner
	Preview   http.Handler
	PreviewMe http.Handler

	// ✅ NEW: order scan verify (authenticated)
	// - POST /mall/me/orders/scan/verify
	OrderScanVerify http.Handler

	// ✅ NEW: order scan transfer (authenticated)
	// - POST /mall/me/orders/scan/transfer
	OrderScanTransfer http.Handler

	// ✅ NEW: owner resolve (walletAddress/toAddress -> avatarId or brandId)
	// - /mall/owners/resolve : public OK (tokenの所有者表示など)
	OwnerResolve http.Handler

	Order http.Handler

	// ✅ invoices (buyer-facing)
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

// Register registers buyer-facing routes onto mux (mall only).
func Register(mux *http.ServeMux, deps Deps) {
	if mux == nil {
		return
	}

	// lists
	handleSafe(mux, "/mall/lists", deps.List, "List")
	handleSafe(mux, "/mall/lists/", deps.List, "List")
	handleSafe(mux, "/mall/me/lists", deps.List, "List(me)")
	handleSafe(mux, "/mall/me/lists/", deps.List, "List(me)")

	// inventories
	handleSafe(mux, "/mall/inventories", deps.Inventory, "Inventory")
	handleSafe(mux, "/mall/inventories/", deps.Inventory, "Inventory")
	handleSafe(mux, "/mall/me/inventories", deps.Inventory, "Inventory(me)")
	handleSafe(mux, "/mall/me/inventories/", deps.Inventory, "Inventory(me)")

	// product blueprints
	handleSafe(mux, "/mall/product-blueprints", deps.ProductBlueprint, "ProductBlueprint")
	handleSafe(mux, "/mall/product-blueprints/", deps.ProductBlueprint, "ProductBlueprint")
	handleSafe(mux, "/mall/me/product-blueprints", deps.ProductBlueprint, "ProductBlueprint(me)")
	handleSafe(mux, "/mall/me/product-blueprints/", deps.ProductBlueprint, "ProductBlueprint(me)")

	// models
	handleSafe(mux, "/mall/models", deps.Model, "Model")
	handleSafe(mux, "/mall/models/", deps.Model, "Model")
	handleSafe(mux, "/mall/me/models", deps.Model, "Model(me)")
	handleSafe(mux, "/mall/me/models/", deps.Model, "Model(me)")

	// catalog
	handleSafe(mux, "/mall/catalog", deps.Catalog, "Catalog")
	handleSafe(mux, "/mall/catalog/", deps.Catalog, "Catalog")
	handleSafe(mux, "/mall/me/catalog", deps.Catalog, "Catalog(me)")
	handleSafe(mux, "/mall/me/catalog/", deps.Catalog, "Catalog(me)")

	// token blueprints
	handleSafe(mux, "/mall/token-blueprints", deps.TokenBlueprint, "TokenBlueprint")
	handleSafe(mux, "/mall/token-blueprints/", deps.TokenBlueprint, "TokenBlueprint")
	handleSafe(mux, "/mall/me/token-blueprints", deps.TokenBlueprint, "TokenBlueprint(me)")
	handleSafe(mux, "/mall/me/token-blueprints/", deps.TokenBlueprint, "TokenBlueprint(me)")

	// companies / brands
	handleSafe(mux, "/mall/companies", deps.Company, "Company")
	handleSafe(mux, "/mall/companies/", deps.Company, "Company")
	handleSafe(mux, "/mall/me/companies", deps.Company, "Company(me)")
	handleSafe(mux, "/mall/me/companies/", deps.Company, "Company(me)")

	handleSafe(mux, "/mall/brands", deps.Brand, "Brand")
	handleSafe(mux, "/mall/brands/", deps.Brand, "Brand")
	handleSafe(mux, "/mall/me/brands", deps.Brand, "Brand(me)")
	handleSafe(mux, "/mall/me/brands/", deps.Brand, "Brand(me)")

	// sign-in
	handleSafe(mux, "/mall/sign-in", deps.SignIn, "SignIn")
	handleSafe(mux, "/mall/sign-in/", deps.SignIn, "SignIn")
	handleSafe(mux, "/mall/me/sign-in", deps.SignIn, "SignIn(me)")
	handleSafe(mux, "/mall/me/sign-in/", deps.SignIn, "SignIn(me)")

	// users
	handleSafe(mux, "/mall/users", deps.User, "User")
	handleSafe(mux, "/mall/users/", deps.User, "User")
	handleSafe(mux, "/mall/me/users", deps.User, "User(me)")
	handleSafe(mux, "/mall/me/users/", deps.User, "User(me)")

	// shipping addresses
	handleSafe(mux, "/mall/shipping-addresses", deps.ShippingAddress, "ShippingAddress")
	handleSafe(mux, "/mall/shipping-addresses/", deps.ShippingAddress, "ShippingAddress")
	handleSafe(mux, "/mall/me/shipping-addresses", deps.ShippingAddress, "ShippingAddress(me)")
	handleSafe(mux, "/mall/me/shipping-addresses/", deps.ShippingAddress, "ShippingAddress(me)")

	// billing addresses
	handleSafe(mux, "/mall/billing-addresses", deps.BillingAddress, "BillingAddress")
	handleSafe(mux, "/mall/billing-addresses/", deps.BillingAddress, "BillingAddress")
	handleSafe(mux, "/mall/me/billing-addresses", deps.BillingAddress, "BillingAddress(me)")
	handleSafe(mux, "/mall/me/billing-addresses/", deps.BillingAddress, "BillingAddress(me)")

	// avatars
	handleSafe(mux, "/mall/avatars", deps.Avatar, "Avatar")
	handleSafe(mux, "/mall/avatars/", deps.Avatar, "Avatar")
	handleSafe(mux, "/mall/me/avatars", deps.Avatar, "Avatar(me)")
	handleSafe(mux, "/mall/me/avatars/", deps.Avatar, "Avatar(me)")

	// me avatar (single endpoint)
	handleSafe(mux, "/mall/me/avatar", deps.MeAvatar, "MeAvatar")
	handleSafe(mux, "/mall/me/avatar/", deps.MeAvatar, "MeAvatar")

	// avatar states
	handleSafe(mux, "/mall/me/avatar-states", deps.AvatarState, "AvatarState(me)")
	handleSafe(mux, "/mall/me/avatar-states/", deps.AvatarState, "AvatarState(me)")

	// ------------------------------------------------------------
	// ✅ Wallet (new contract only; legacy removed)
	// - GET  /mall/me/wallets
	// - POST /mall/me/wallets/sync
	//
	// NOTE:
	// - We intentionally register only the base path and the trailing-slash
	//   prefix. /mall/me/wallets/sync is matched by the "/mall/me/wallets/"
	//   pattern and handled inside the Wallet handler.
	// ------------------------------------------------------------
	handleSafe(mux, "/mall/me/wallets", deps.Wallet, "Wallet(me)")
	handleSafe(mux, "/mall/me/wallets/", deps.Wallet, "Wallet(me)")

	// cart
	handleSafe(mux, "/mall/cart", deps.Cart, "Cart")
	handleSafe(mux, "/mall/cart/", deps.Cart, "Cart")
	handleSafe(mux, "/mall/me/cart", deps.Cart, "Cart(me)")
	handleSafe(mux, "/mall/me/cart/", deps.Cart, "Cart(me)")

	// ✅ preview
	handleSafe(mux, "/mall/preview", deps.Preview, "Preview")
	handleSafe(mux, "/mall/preview/", deps.Preview, "Preview")
	handleSafe(mux, "/mall/me/preview", deps.PreviewMe, "Preview(me)")
	handleSafe(mux, "/mall/me/preview/", deps.PreviewMe, "Preview(me)")

	// ✅ NEW: order scan verify (authenticated)
	// - POST /mall/me/orders/scan/verify
	handleSafe(mux, "/mall/me/orders/scan/verify", deps.OrderScanVerify, "OrderScanVerify(me)")
	handleSafe(mux, "/mall/me/orders/scan/verify/", deps.OrderScanVerify, "OrderScanVerify(me)")

	// ✅ NEW: order scan transfer (authenticated)
	// - POST /mall/me/orders/scan/transfer
	handleSafe(mux, "/mall/me/orders/scan/transfer", deps.OrderScanTransfer, "OrderScanTransfer(me)")
	handleSafe(mux, "/mall/me/orders/scan/transfer/", deps.OrderScanTransfer, "OrderScanTransfer(me)")

	// ✅ NEW: owner resolve (walletAddress/toAddress -> avatarId or brandId)
	handleSafe(mux, "/mall/owners/resolve", deps.OwnerResolve, "OwnerResolve")
	handleSafe(mux, "/mall/owners/resolve/", deps.OwnerResolve, "OwnerResolve")

	// payment
	handleSafe(mux, "/mall/me/payment", deps.Payment, "Payment(me)")
	handleSafe(mux, "/mall/me/payment/", deps.Payment, "Payment(me)")
	handleSafe(mux, "/mall/me/payments", deps.Payment, "Payment(me)")
	handleSafe(mux, "/mall/me/payments/", deps.Payment, "Payment(me)")

	// invoices
	handleSafe(mux, "/mall/me/invoices", deps.Invoice, "Invoice(me)")
	handleSafe(mux, "/mall/me/invoices/", deps.Invoice, "Invoice(me)")

	// orders
	handleSafe(mux, "/mall/me/orders", deps.Order, "Order(me)")
	handleSafe(mux, "/mall/me/orders/", deps.Order, "Order(me)")
}
