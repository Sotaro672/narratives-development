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
	Catalog          http.Handler
	TokenBlueprint   http.Handler // patch

	// tokenBlueprint reviews
	TokenBlueprintReview http.Handler

	// ProductBlueprint reviews (catalog + me/catalog)
	// - public: GET /mall/catalog/product-blueprints/{pbId}/reviews
	// - me:     GET/POST /mall/me/catalog/product-blueprints/{pbId}/reviews
	ProductBlueprintReview http.Handler

	Company http.Handler
	Brand   http.Handler

	SignIn http.Handler

	// auth actions
	// - POST /auth/email-verification/send
	Auth http.Handler

	User            http.Handler
	ShippingAddress http.Handler
	PaymentMethod   http.Handler

	// /mall/avatars (POST create) + /mall/avatars/{id} (GET/PATCH/DELETE)
	Avatar http.Handler

	// /mall/me/avatar (resolve avatarId by current user uid)
	MeAvatar http.Handler

	AvatarState http.Handler

	// me: /mall/me/messages
	Message http.Handler

	// public: /mall/wallets
	Wallet http.Handler

	// me: /mall/me/wallets
	MeWallet http.Handler

	Cart    http.Handler
	Payment http.Handler

	Preview   http.Handler
	PreviewMe http.Handler

	OrderScanVerify   http.Handler
	OrderScanTransfer http.Handler

	// share transfer (me)
	// POST /mall/me/contents/share
	ShareTransfer http.Handler

	OwnerResolve http.Handler

	Order http.Handler

	// market resales (public)
	// - GET /mall/market/resales
	// - GET /mall/market/resales/cursor
	// - GET /mall/market/resales/{id}
	Market http.Handler

	// resales (me)
	// - POST /mall/me/resales
	// - GET  /mall/me/resales
	// - GET/PUT/DELETE /mall/me/resales/{id}
	// - GET  /mall/me/resales/{id}/images
	// - POST /mall/me/resales/{id}/images
	// - DELETE /mall/me/resales/{id}/images/{imageId}
	// - PUT /mall/me/resales/{id}/primary-image
	Resale http.Handler

	// inquiries (me)
	// - POST /mall/me/inquiries
	// - GET  /mall/me/inquiries/{id}
	// - POST /mall/me/inquiries/{id}/reply
	// - POST /mall/me/inquiries/{id}/close
	Inquiry http.Handler

	// me announcements
	// - GET  /mall/me/announcement
	// - POST /mall/me/announcement/{announcementId}/read
	Announcement http.Handler

	// /mall/me/setup-status (existence checks for redirect)
	SetupStatus http.Handler
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
	if h == nil {
		log.Printf("[mall.router] WARN: nil handler: %s pattern=%s (registering NotFoundHandler)", name, pattern)
		h = http.NotFoundHandler()
	}

	if auth == nil {
		log.Printf("[mall.router] WARN: nil auth middleware: %s pattern=%s (registering WITHOUT auth)", name, pattern)
		handleSafe(mux, pattern, h, name)
		return
	}
	handleSafe(mux, pattern, auth(h), name)
}

// handleSafeAuthAvatar registers pattern with auth + avatarContext wrapped handler.
//
// IMPORTANT (order):
// - UserAuthMiddleware must run BEFORE AvatarContextMiddleware.
// - In net/http middleware chain, the OUTER wrapper runs first.
// - Therefore: auth(avatar(Handler))
func handleSafeAuthAvatar(
	mux *http.ServeMux,
	pattern string,
	h http.Handler,
	name string,
	auth func(http.Handler) http.Handler,
	avatar func(http.Handler) http.Handler,
) {
	if h == nil {
		log.Printf("[mall.router] WARN: nil handler: %s pattern=%s (registering NotFoundHandler)", name, pattern)
		h = http.NotFoundHandler()
	}

	if auth == nil {
		log.Printf("[mall.router] WARN: nil auth middleware: %s pattern=%s (registering WITHOUT auth+avatar)", name, pattern)
		handleSafe(mux, pattern, h, name)
		return
	}

	if avatar == nil {
		log.Printf("[mall.router] WARN: nil avatar context middleware: %s pattern=%s (registering WITHOUT avatar context)", name, pattern)
		handleSafe(mux, pattern, auth(h), name)
		return
	}

	handleSafe(mux, pattern, auth(avatar(h)), name)
}

// Register registers buyer-facing routes onto mux (mall only).
//
// auth:
//   - /mall/me/** routes requiring user auth
//
// avatar:
//   - /mall/me/** routes requiring avatar context
func Register(mux *http.ServeMux, deps Deps, auth func(http.Handler) http.Handler, avatar func(http.Handler) http.Handler) {
	if mux == nil {
		return
	}

	// ------------------------------------------------------------
	// Public routes (no auth)
	// ------------------------------------------------------------

	// lists (public)
	handleSafe(mux, "/mall/lists", deps.List, "List")
	handleSafe(mux, "/mall/lists/", deps.List, "List")

	// product blueprints (public)
	handleSafe(mux, "/mall/product-blueprints", deps.ProductBlueprint, "ProductBlueprint")
	handleSafe(mux, "/mall/product-blueprints/", deps.ProductBlueprint, "ProductBlueprint")

	// catalog (public)
	handleSafe(mux, "/mall/catalog", deps.Catalog, "Catalog")
	handleSafe(mux, "/mall/catalog/", deps.Catalog, "Catalog")

	// productBlueprint reviews (public catalog)
	handleSafe(mux, "/mall/catalog/product-blueprints", deps.ProductBlueprintReview, "ProductBlueprintReview(catalog)")
	handleSafe(mux, "/mall/catalog/product-blueprints/", deps.ProductBlueprintReview, "ProductBlueprintReview(catalog)")

	// token blueprints (public)
	handleSafe(mux, "/mall/token-blueprints", deps.TokenBlueprint, "TokenBlueprint")
	handleSafe(mux, "/mall/token-blueprints/", deps.TokenBlueprint, "TokenBlueprint")

	handleSafe(mux, "/mall/brands", deps.Brand, "Brand")
	handleSafe(mux, "/mall/brands/", deps.Brand, "Brand")

	// sign-in (public)
	handleSafe(mux, "/mall/sign-in", deps.SignIn, "SignIn")
	handleSafe(mux, "/mall/sign-in/", deps.SignIn, "SignIn")

	// stripe config (public publishable key)
	handleSafe(mux, "/mall/config/stripe", deps.PaymentMethod, "PaymentMethod(stripe.config)")
	handleSafe(mux, "/mall/config/stripe/", deps.PaymentMethod, "PaymentMethod(stripe.config)")

	// avatars (public)
	handleSafe(mux, "/mall/avatars", deps.Avatar, "Avatar")
	handleSafe(mux, "/mall/avatars/", deps.Avatar, "Avatar")

	// wallets (public)
	handleSafe(mux, "/mall/wallets", deps.Wallet, "Wallet")
	handleSafe(mux, "/mall/wallets/", deps.Wallet, "Wallet")

	// owner resolve (public OK)
	handleSafe(mux, "/mall/owners/resolve", deps.OwnerResolve, "OwnerResolve")
	handleSafe(mux, "/mall/owners/resolve/", deps.OwnerResolve, "OwnerResolve")

	// preview (public)
	handleSafe(mux, "/mall/preview", deps.Preview, "Preview")
	handleSafe(mux, "/mall/preview/", deps.Preview, "Preview")

	// market resales (public)
	handleSafe(mux, "/mall/market/resales", deps.Market, "Market")
	handleSafe(mux, "/mall/market/resales/", deps.Market, "Market")

	// ------------------------------------------------------------
	// Auth-required routes outside /mall/me
	// ------------------------------------------------------------

	// auth email verification - auth only
	handleSafeAuth(mux, "/auth/email-verification/send", deps.Auth, "Auth(emailVerification)", auth)
	handleSafeAuth(mux, "/auth/email-verification/send/", deps.Auth, "Auth(emailVerification)", auth)

	// ------------------------------------------------------------
	// Auth-required routes (/mall/me/**)
	// setup-status / users / shipping-addresses / payment-methods are auth-only.
	// ------------------------------------------------------------

	// setup status (me) - auth only
	handleSafeAuth(mux, "/mall/me/setup-status", deps.SetupStatus, "SetupStatus(me)", auth)
	handleSafeAuth(mux, "/mall/me/setup-status/", deps.SetupStatus, "SetupStatus(me)", auth)

	// users (me) - auth only
	handleSafeAuth(mux, "/mall/me/users", deps.User, "User(me)", auth)
	handleSafeAuth(mux, "/mall/me/users/", deps.User, "User(me)", auth)

	// shipping addresses (me) - auth only
	handleSafeAuth(mux, "/mall/me/shipping-addresses", deps.ShippingAddress, "ShippingAddress(me)", auth)
	handleSafeAuth(mux, "/mall/me/shipping-addresses/", deps.ShippingAddress, "ShippingAddress(me)", auth)

	// payment methods (me) - auth only
	handleSafeAuth(mux, "/mall/me/payment-methods", deps.PaymentMethod, "PaymentMethod(me)", auth)
	handleSafeAuth(mux, "/mall/me/payment-methods/", deps.PaymentMethod, "PaymentMethod(me)", auth)

	// ------------------------------------------------------------
	// Auth+Avatar-required routes (/mall/me/**)
	// ------------------------------------------------------------

	// catalog (me)
	handleSafeAuthAvatar(mux, "/mall/me/catalog", deps.Catalog, "Catalog(me)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/catalog/", deps.Catalog, "Catalog(me)", auth, avatar)

	// productBlueprint reviews (me catalog)
	handleSafeAuthAvatar(mux, "/mall/me/catalog/product-blueprints", deps.ProductBlueprintReview, "ProductBlueprintReview(me.catalog)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/catalog/product-blueprints/", deps.ProductBlueprintReview, "ProductBlueprintReview(me.catalog)", auth, avatar)

	// token blueprints (me)
	handleSafeAuthAvatar(mux, "/mall/me/token-blueprints", deps.TokenBlueprint, "TokenBlueprint(me)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/token-blueprints/", deps.TokenBlueprint, "TokenBlueprint(me)", auth, avatar)

	handleSafeAuthAvatar(mux, "/mall/me/brands", deps.Brand, "Brand(me)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/brands/", deps.Brand, "Brand(me)", auth, avatar)

	// me avatar
	handleSafeAuthAvatar(mux, "/mall/me/avatars", deps.MeAvatar, "MeAvatar", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/avatars/", deps.MeAvatar, "MeAvatar", auth, avatar)

	// avatar states (me)
	handleSafeAuthAvatar(mux, "/mall/me/avatar-states", deps.AvatarState, "AvatarState(me)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/avatar-states/", deps.AvatarState, "AvatarState(me)", auth, avatar)

	// messages (me)
	handleSafeAuthAvatar(mux, "/mall/me/messages", deps.Message, "Message(me)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/messages/", deps.Message, "Message(me)", auth, avatar)

	// wallet (me)
	handleSafeAuthAvatar(mux, "/mall/me/wallets", deps.MeWallet, "MeWallet", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/wallets/", deps.MeWallet, "MeWallet", auth, avatar)

	// cart (me)
	handleSafeAuthAvatar(mux, "/mall/me/cart", deps.Cart, "Cart(me)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/cart/", deps.Cart, "Cart(me)", auth, avatar)

	// preview (me)
	handleSafeAuthAvatar(mux, "/mall/me/preview", deps.PreviewMe, "Preview(me)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/preview/", deps.PreviewMe, "Preview(me)", auth, avatar)

	// order scan verify (me)
	handleSafeAuthAvatar(mux, "/mall/me/orders/scan/verify", deps.OrderScanVerify, "OrderScanVerify(me)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/orders/scan/verify/", deps.OrderScanVerify, "OrderScanVerify(me)", auth, avatar)

	// order scan transfer (me)
	handleSafeAuthAvatar(mux, "/mall/me/orders/scan/transfer", deps.OrderScanTransfer, "OrderScanTransfer(me)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/orders/scan/transfer/", deps.OrderScanTransfer, "OrderScanTransfer(me)", auth, avatar)

	// share transfer (me)
	handleSafeAuthAvatar(mux, "/mall/me/contents/share", deps.ShareTransfer, "ShareTransfer(me)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/contents/share/", deps.ShareTransfer, "ShareTransfer(me)", auth, avatar)

	// announcements (me)
	handleSafeAuthAvatar(mux, "/mall/me/announcement", deps.Announcement, "Announcement(me)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/announcement/", deps.Announcement, "Announcement(me)", auth, avatar)

	// payment (me)
	handleSafeAuthAvatar(mux, "/mall/me/payments", deps.Payment, "Payment(me)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/payments/", deps.Payment, "Payment(me)", auth, avatar)

	// orders (me)
	handleSafeAuthAvatar(mux, "/mall/me/orders", deps.Order, "Order(me)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/orders/", deps.Order, "Order(me)", auth, avatar)

	// resales (me)
	handleSafeAuthAvatar(mux, "/mall/me/resales", deps.Resale, "Resale(me)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/resales/", deps.Resale, "Resale(me)", auth, avatar)

	// inquiries (me)
	handleSafeAuthAvatar(mux, "/mall/me/inquiries", deps.Inquiry, "Inquiry(me)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/inquiries/", deps.Inquiry, "Inquiry(me)", auth, avatar)
}
