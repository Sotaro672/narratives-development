// backend\internal\adapters\in\http\mall\router.go
package mall

import (
	"log"
	"net/http"
)

// Deps is a buyer-facing (mall) handler set.
type Deps struct {
	List           http.Handler
	Catalog        http.Handler
	TokenBlueprint http.Handler // patch

	// ProductBlueprint reviews (catalog + me/catalog)
	// - public: GET /mall/catalog/product-blueprints/{pbId}/reviews
	// - me:     GET/POST /mall/me/catalog/product-blueprints/{pbId}/reviews
	ProductBlueprintReview http.Handler
	Brand                  http.Handler

	SignIn http.Handler

	// auth actions
	// - POST /auth/email-verification/send
	Auth http.Handler

	User            http.Handler
	ShippingAddress http.Handler
	ShippingQuote   http.Handler
	PaymentMethod   http.Handler
	PayoutAccount   http.Handler

	// /mall/avatars (POST create) + /mall/avatars/{id} (GET/PATCH/DELETE)
	Avatar http.Handler

	// /mall/me/avatar (resolve avatarId by current user uid)
	MeAvatar http.Handler

	// me: /mall/me/wallets
	MeWallet http.Handler

	// me: /mall/me/likes
	// - GET    /mall/me/likes
	// - GET    /mall/me/likes/{list|resale}/{id}
	// - PUT    /mall/me/likes/{list|resale}/{id}
	// - DELETE /mall/me/likes/{list|resale}/{id}
	Like http.Handler

	Cart    http.Handler
	Payment http.Handler

	Preview   http.Handler
	PreviewMe http.Handler

	OrderScanTransfer http.Handler
	Order             http.Handler

	// trades (me)
	// - GET  /mall/me/trades/order-items/{orderId}/{itemIndex}
	// - GET  /mall/me/trades/{tradeId}
	// - POST /mall/me/trades/{tradeId}/messages
	// - POST /mall/me/trades/{tradeId}/read
	// - GET  /mall/me/trades/{tradeId}/unread-count
	Trade http.Handler

	// avatar reviews (me)
	// - POST /mall/me/avatar-reviews
	AvatarReview http.Handler

	// market resales (auth + avatar required)
	// - GET /mall/market/resales
	// - GET /mall/market/resales/cursor
	// - GET /mall/market/resales/{id}
	Market http.Handler

	// resales
	// public:
	// - GET /mall/resales/avatar/{avatarId}
	// - GET /mall/resales/{id}
	// - GET /mall/resales/{id}/images
	//
	// me:
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

	// review report decision notifications (me)
	// - GET  /mall/me/review-report-decision-notifications
	// - POST /mall/me/review-report-decision-notifications/{notificationId}/read
	ReviewReportDecisionNotification http.Handler

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

func mallRouterUnavailableHandler(errorCode string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"` + errorCode + `"}`))
	})
}

// handleSafeAuth registers pattern with auth-wrapped handler.
// Missing auth middleware fails closed with Service Unavailable.
func handleSafeAuth(
	mux *http.ServeMux,
	pattern string,
	h http.Handler,
	name string,
	auth func(http.Handler) http.Handler,
) {
	if h == nil {
		log.Printf("[mall.router] WARN: nil handler: %s pattern=%s (registering NotFoundHandler)", name, pattern)
		h = http.NotFoundHandler()
	}
	if auth == nil {
		log.Printf("[mall.router] ERROR: nil auth middleware: %s pattern=%s (failing closed)", name, pattern)
		handleSafe(mux, pattern, mallRouterUnavailableHandler("auth_middleware_not_configured"), name)
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
		log.Printf("[mall.router] ERROR: nil auth middleware: %s pattern=%s (failing closed)", name, pattern)
		handleSafe(mux, pattern, mallRouterUnavailableHandler("auth_middleware_not_configured"), name)
		return
	}
	if avatar == nil {
		log.Printf("[mall.router] ERROR: nil avatar context middleware: %s pattern=%s (failing closed)", name, pattern)
		handleSafe(mux, pattern, mallRouterUnavailableHandler("avatar_context_middleware_not_configured"), name)
		return
	}
	handleSafe(mux, pattern, auth(avatar(h)), name)
}

// avatarPublicHandler keeps public avatar reads available while requiring
// user authentication for avatar creation.
// If auth middleware is unavailable, avatar creation fails closed.
func avatarPublicHandler(h http.Handler, auth func(http.Handler) http.Handler) http.Handler {
	if h == nil {
		return nil
	}
	if auth == nil {
		log.Printf("[mall.router] ERROR: nil auth middleware: Avatar(create) (failing closed for POST)")
		unavailable := mallRouterUnavailableHandler("auth_middleware_not_configured")
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost && (r.URL.Path == "/mall/avatars" || r.URL.Path == "/mall/avatars/") {
				unavailable.ServeHTTP(w, r)
				return
			}
			h.ServeHTTP(w, r)
		})
	}
	authed := auth(h)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && (r.URL.Path == "/mall/avatars" || r.URL.Path == "/mall/avatars/") {
			authed.ServeHTTP(w, r)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// paymentReadOnlyHandler limits the buyer-facing payment endpoint to reads.
//
// Payment creation is performed only from the Console dispatch flow.
// Mall clients must never be able to create or confirm a Stripe payment.
func paymentReadOnlyHandler(h http.Handler) http.Handler {
	if h == nil {
		return nil
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodOptions {
			h.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Allow", "GET, OPTIONS")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write([]byte(`{"error":"payment_creation_disabled"}`))
	})
}

// Register registers buyer-facing routes onto mux (mall only).
//
// auth:
//   - /mall/market/resales** and /mall/me/** routes requiring user auth
//
// avatar:
//   - /mall/market/resales** and /mall/me/** routes requiring avatar context
func Register(
	mux *http.ServeMux,
	deps Deps,
	auth func(http.Handler) http.Handler,
	avatar func(http.Handler) http.Handler,
) {
	if mux == nil {
		return
	}

	// ------------------------------------------------------------
	// Public routes (no auth)
	// ------------------------------------------------------------

	// lists (public)
	handleSafe(mux, "/mall/lists", deps.List, "List")
	handleSafe(mux, "/mall/lists/", deps.List, "List")

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

	// avatars
	// - POST /mall/avatars: auth required
	// - GET  /mall/avatars/{id}: public
	avatarHandler := avatarPublicHandler(deps.Avatar, auth)
	handleSafe(mux, "/mall/avatars", avatarHandler, "Avatar")
	handleSafe(mux, "/mall/avatars/", avatarHandler, "Avatar")

	// preview (public)
	handleSafe(mux, "/mall/preview", deps.Preview, "Preview")
	handleSafe(mux, "/mall/preview/", deps.Preview, "Preview")

	// resales by public avatar
	handleSafe(mux, "/mall/resales", deps.Resale, "Resale(public)")
	handleSafe(mux, "/mall/resales/", deps.Resale, "Resale(public)")

	// ------------------------------------------------------------
	// Auth-required routes outside /mall/me
	// ------------------------------------------------------------

	// auth email verification - auth only
	handleSafeAuth(mux, "/auth/email-verification/send", deps.Auth, "Auth(emailVerification)", auth)
	handleSafeAuth(mux, "/auth/email-verification/send/", deps.Auth, "Auth(emailVerification)", auth)

	// ------------------------------------------------------------
	// Auth+Avatar-required routes outside /mall/me
	// ------------------------------------------------------------

	// market resales
	handleSafeAuthAvatar(mux, "/mall/market/resales", deps.Market, "Market", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/market/resales/", deps.Market, "Market", auth, avatar)

	// ------------------------------------------------------------
	// Auth-required routes (/mall/me/**)
	// setup-status / users / shipping-addresses / payment-methods / payout-account are auth-only.
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

	// payout account (me) - auth only
	handleSafeAuth(mux, "/mall/me/payout-account", deps.PayoutAccount, "PayoutAccount(me)", auth)
	handleSafeAuth(mux, "/mall/me/payout-account/", deps.PayoutAccount, "PayoutAccount(me)", auth)

	// ------------------------------------------------------------
	// Auth+Avatar-required routes (/mall/me/**)
	// ------------------------------------------------------------

	// catalog (me)
	handleSafeAuthAvatar(mux, "/mall/me/catalog", deps.Catalog, "Catalog(me)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/catalog/", deps.Catalog, "Catalog(me)", auth, avatar)

	// productBlueprint reviews (me catalog)
	handleSafeAuthAvatar(
		mux,
		"/mall/me/catalog/product-blueprints",
		deps.ProductBlueprintReview,
		"ProductBlueprintReview(me.catalog)",
		auth,
		avatar,
	)
	handleSafeAuthAvatar(
		mux,
		"/mall/me/catalog/product-blueprints/",
		deps.ProductBlueprintReview,
		"ProductBlueprintReview(me.catalog)",
		auth,
		avatar,
	)

	// token blueprints (me)
	handleSafeAuthAvatar(mux, "/mall/me/token-blueprints", deps.TokenBlueprint, "TokenBlueprint(me)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/token-blueprints/", deps.TokenBlueprint, "TokenBlueprint(me)", auth, avatar)

	// me avatar
	handleSafeAuthAvatar(mux, "/mall/me/avatars", deps.MeAvatar, "MeAvatar", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/avatars/", deps.MeAvatar, "MeAvatar", auth, avatar)

	// wallet (me)
	handleSafeAuthAvatar(mux, "/mall/me/wallets", deps.MeWallet, "MeWallet", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/wallets/", deps.MeWallet, "MeWallet", auth, avatar)

	// likes (me)
	handleSafeAuthAvatar(mux, "/mall/me/likes", deps.Like, "Like(me)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/likes/", deps.Like, "Like(me)", auth, avatar)

	// cart (me)
	handleSafeAuthAvatar(mux, "/mall/me/cart", deps.Cart, "Cart(me)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/cart/", deps.Cart, "Cart(me)", auth, avatar)

	// preview (me)
	handleSafeAuthAvatar(mux, "/mall/me/preview", deps.PreviewMe, "Preview(me)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/preview/", deps.PreviewMe, "Preview(me)", auth, avatar)

	// order scan transfer (me)
	handleSafeAuthAvatar(mux, "/mall/me/orders/scan/transfer", deps.OrderScanTransfer, "OrderScanTransfer(me)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/orders/scan/transfer/", deps.OrderScanTransfer, "OrderScanTransfer(me)", auth, avatar)

	// announcements (me)
	handleSafeAuthAvatar(mux, "/mall/me/announcement", deps.Announcement, "Announcement(me)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/announcement/", deps.Announcement, "Announcement(me)", auth, avatar)

	// review report decision notifications (me)
	handleSafeAuthAvatar(
		mux,
		"/mall/me/review-report-decision-notifications",
		deps.ReviewReportDecisionNotification,
		"ReviewReportDecisionNotification(me)",
		auth,
		avatar,
	)
	handleSafeAuthAvatar(
		mux,
		"/mall/me/review-report-decision-notifications/",
		deps.ReviewReportDecisionNotification,
		"ReviewReportDecisionNotification(me)",
		auth,
		avatar,
	)

	// shipping quote (me)
	handleSafeAuthAvatar(mux, "/mall/me/shipping-quotes", deps.ShippingQuote, "ShippingQuote(me)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/shipping-quotes/", deps.ShippingQuote, "ShippingQuote(me)", auth, avatar)

	// payment context (me) - GET only
	paymentHandler := paymentReadOnlyHandler(deps.Payment)
	handleSafeAuthAvatar(mux, "/mall/me/payments", paymentHandler, "Payment(me.readOnly)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/payments/", paymentHandler, "Payment(me.readOnly)", auth, avatar)

	// orders (me)
	handleSafeAuthAvatar(mux, "/mall/me/orders", deps.Order, "Order(me)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/orders/", deps.Order, "Order(me)", auth, avatar)

	// trades (me)
	handleSafeAuthAvatar(mux, "/mall/me/trades", deps.Trade, "Trade(me)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/trades/", deps.Trade, "Trade(me)", auth, avatar)

	// avatar reviews (me)
	handleSafeAuthAvatar(mux, "/mall/me/avatar-reviews", deps.AvatarReview, "AvatarReview(me)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/avatar-reviews/", deps.AvatarReview, "AvatarReview(me)", auth, avatar)

	// resales (me)
	handleSafeAuthAvatar(mux, "/mall/me/resales", deps.Resale, "Resale(me)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/resales/", deps.Resale, "Resale(me)", auth, avatar)

	// inquiries (me)
	handleSafeAuthAvatar(mux, "/mall/me/inquiries", deps.Inquiry, "Resale(me)", auth, avatar)
	handleSafeAuthAvatar(mux, "/mall/me/inquiries/", deps.Inquiry, "Resale(me)", auth, avatar)
}
