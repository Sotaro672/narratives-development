// backend/internal/platform/di/mall/register.go
package mall

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	mallhttp "narratives/internal/adapters/in/http/mall"
	mallhandler "narratives/internal/adapters/in/http/mall/handler"
	mallwebhook "narratives/internal/adapters/in/http/mall/webhook"
	"narratives/internal/adapters/in/http/middleware"
)

// notImplemented returns a non-nil handler (so deps are never nil) for endpoints
// that are not wired yet.
func notImplemented(name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "not_implemented",
			"name":  name,
		})
	})
}

// requireUserAuth wraps handler with UserAuthMiddleware (fail-closed).
// If middleware is not initialized, it returns 503 so the bug is obvious.
func requireUserAuth(mw *middleware.UserAuthMiddleware, h http.Handler, name string) http.Handler {
	if h == nil {
		h = http.NotFoundHandler()
	}
	if mw == nil || mw.FirebaseAuth == nil {
		log.Printf("[mall.register] ERROR: UserAuthMiddleware is not initialized (endpoint=%s). returning 503", name)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "user_auth_not_initialized",
				"name":  name,
			})
		})
	}
	return mw.Handler(h)
}

// Register registers mall routes onto mux.
// Pure DI: construct handlers and pass into mall router.Register.
// - No method/path branching here
// - deps must be non-nil for all handlers (no nil in deps)
// - UserAuthMiddleware is applied to ALL user-authenticated endpoints (user系全部)
func Register(mux *http.ServeMux, cont *Container) {
	if mux == nil || cont == nil {
		return
	}

	// ------------------------------------------------------------
	// Auth middleware (buyer/user side)
	// ------------------------------------------------------------
	var userAuthMW *middleware.UserAuthMiddleware
	if cont.Infra != nil && cont.Infra.FirebaseAuth != nil {
		userAuthMW = &middleware.UserAuthMiddleware{
			FirebaseAuth: cont.Infra.FirebaseAuth,
		}
	} else {
		// fail-closed in requireUserAuth
		log.Printf("[mall.register] WARN: cont.Infra or cont.Infra.FirebaseAuth is nil (user auth will return 503 on protected endpoints)")
		userAuthMW = &middleware.UserAuthMiddleware{FirebaseAuth: nil}
	}

	// ----------------------------
	// Handlers (construct only)
	// ----------------------------
	// default to non-nil for all handlers
	listH := notImplemented("List")
	invH := notImplemented("Inventory")
	pbH := notImplemented("ProductBlueprint")
	modelH := notImplemented("Model")
	catalogH := notImplemented("Catalog")
	tbH := notImplemented("TokenBlueprint")
	companyH := notImplemented("Company")
	brandH := notImplemented("Brand")

	// user-authenticated
	userH := notImplemented("User")
	shipH := notImplemented("ShippingAddress")
	billH := notImplemented("BillingAddress")
	avatarH := notImplemented("Avatar")
	avatarStateH := notImplemented("AvatarState")
	walletH := notImplemented("Wallet")
	cartH := notImplemented("Cart")
	payH := notImplemented("Payment")
	orderH := notImplemented("Order")
	invoiceH := notImplemented("Invoice")
	meAvatarH := notImplemented("MeAvatar")

	// Preview split:
	// - /mall/preview     : public (no auth)
	// - /mall/me/preview  : authenticated
	previewPublicH := notImplemented("PreviewPublic")
	previewMeH := notImplemented("PreviewMe")

	// Lists (public)
	if cont.ListUC != nil {
		listH = mallhandler.NewMallListHandler(cont.ListUC)
	}

	// Catalog (public)
	if cont.CatalogQ != nil {
		catalogH = mallhandler.NewMallCatalogHandler(cont.CatalogQ)
	}

	// Inventory (public read-only)
	if cont.InventoryUC != nil {
		invH = mallhandler.NewMallInventoryHandler(cont.InventoryUC)
	}

	// TokenBlueprint (public patch)
	if cont.TokenBlueprintRepo != nil {
		if cont.NameResolver != nil {
			if cont.TokenIconURLResolver != nil {
				tbH = mallhandler.NewMallTokenBlueprintHandlerWithNameAndImageResolver(
					cont.TokenBlueprintRepo,
					cont.NameResolver,
					cont.TokenIconURLResolver,
				)
			} else {
				tbH = mallhandler.NewMallTokenBlueprintHandlerWithNameResolver(
					cont.TokenBlueprintRepo,
					cont.NameResolver,
				)
			}
		} else {
			tbH = mallhandler.NewMallTokenBlueprintHandler(cont.TokenBlueprintRepo)
		}
	}

	// Core authenticated resources (user side)
	if cont.UserUC != nil {
		userH = mallhandler.NewUserHandler(cont.UserUC)
	}
	if cont.ShippingAddressUC != nil {
		shipH = mallhandler.NewShippingAddressHandler(cont.ShippingAddressUC)
	}
	if cont.BillingAddressUC != nil {
		billH = mallhandler.NewBillingAddressHandler(cont.BillingAddressUC)
	}

	// Avatar（/mall/avatars）
	if cont.AvatarUC != nil {
		avatarH = mallhandler.NewAvatarHandler(cont.AvatarUC)
	}

	// Wallet
	if cont.WalletUC != nil {
		walletH = mallhandler.NewMallWalletHandler(cont.WalletUC, cont.AvatarUC)
	}

	// /mall/me/avatar (uid -> avatarId)
	if cont.MeAvatarRepo != nil {
		meAvatarH = mallhandler.NewMeAvatarHandler(cont.MeAvatarRepo)
	}

	// Cart (authenticated)
	if cont.CartUC != nil {
		cartH = mallhandler.NewCartHandlerWithQueries(cont.CartUC, cont.CartQ)
	} else if cont.CartQ != nil {
		cartH = mallhandler.NewCartQueryHandler(cont.CartQ)
	}

	// Payment / Order (authenticated)
	if cont.PaymentUC != nil {
		payH = mallhandler.NewPaymentHandlerWithOrderQuery(cont.PaymentUC, cont.OrderQ)
	}
	if cont.OrderUC != nil {
		orderH = mallhandler.NewOrderHandler(cont.OrderUC)
	}

	// Invoice (authenticated)
	if cont.InvoiceUC != nil {
		invoiceH = mallhandler.NewInvoiceHandler(cont.InvoiceUC)
	}

	// ------------------------------------------------------------
	// Preview handler wiring (split)
	// 前提:
	// - cont.PreviewQ が存在
	// - handler.NewPreviewHandler(cont.PreviewQ) が /mall/preview 用
	// - handler.NewPreviewMeHandler(cont.PreviewQ) が /mall/me/preview 用
	// ------------------------------------------------------------
	if cont.PreviewQ != nil {
		previewPublicH = mallhandler.NewPreviewHandler(cont.PreviewQ)
		previewMeH = mallhandler.NewPreviewMeHandler(cont.PreviewQ)
	}

	// SignIn: keep a stable no-op endpoint (client convenience)
	// NOTE: 認証チェックは不要（ただの疎通・互換のため）
	signInH := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	// ------------------------------------------------------------
	// Apply UserAuthMiddleware to ALL user-authenticated handlers
	// ------------------------------------------------------------
	userH = requireUserAuth(userAuthMW, userH, "User")
	shipH = requireUserAuth(userAuthMW, shipH, "ShippingAddress")
	billH = requireUserAuth(userAuthMW, billH, "BillingAddress")
	avatarH = requireUserAuth(userAuthMW, avatarH, "Avatar")
	avatarStateH = requireUserAuth(userAuthMW, avatarStateH, "AvatarState")
	walletH = requireUserAuth(userAuthMW, walletH, "Wallet")
	meAvatarH = requireUserAuth(userAuthMW, meAvatarH, "MeAvatar")

	// cart は auth（/mall/cart も含めて auth にする運用）
	cartH = requireUserAuth(userAuthMW, cartH, "Cart")

	// /mall/me/preview は auth
	previewMeH = requireUserAuth(userAuthMW, previewMeH, "Preview(me)")

	payH = requireUserAuth(userAuthMW, payH, "Payment")
	orderH = requireUserAuth(userAuthMW, orderH, "Order")
	invoiceH = requireUserAuth(userAuthMW, invoiceH, "Invoice") // invoices

	// ----------------------------
	// Router deps
	// ----------------------------
	// ✅ 直し方A:
	// router.go 側に PreviewMe を追加し、/mall/me/preview は deps.PreviewMe に向ける。
	// これにより mux.Handle の二重登録（後勝ち差し替え）を完全に排除する。
	deps := mallhttp.Deps{
		// public
		List: listH,

		Inventory:        invH,
		ProductBlueprint: pbH,
		Model:            modelH,
		Catalog:          catalogH,
		TokenBlueprint:   tbH,

		Company: companyH,
		Brand:   brandH,

		SignIn: signInH,

		// authenticated (user系)
		User:            userH,
		ShippingAddress: shipH,
		BillingAddress:  billH,
		Avatar:          avatarH,
		MeAvatar:        meAvatarH,
		AvatarState:     avatarStateH,
		Wallet:          walletH,
		Cart:            cartH,

		//preview split
		Preview:   previewPublicH, // /mall/preview (public)
		PreviewMe: previewMeH,     // /mall/me/preview (auth)

		Payment: payH,
		Order:   orderH,
		Invoice: invoiceH,
	}

	mallhttp.Register(mux, deps)
	log.Printf("[boot] mall routes registered")

	// ----------------------------
	// Webhooks (no auth)
	// ----------------------------
	// Stripe webhook: PaymentUsecase + signing secret(string) が必要
	if cont.PaymentUC != nil {
		secret := strings.TrimSpace(os.Getenv("STRIPE_WEBHOOK_SECRET"))
		if secret == "" {
			log.Printf("[boot] mall stripe webhook NOT registered: STRIPE_WEBHOOK_SECRET is empty")
			return
		}

		stripeWH := mallwebhook.NewStripeWebhookHandler(cont.PaymentUC, secret)
		mux.Handle(StripeWebhookPath, stripeWH)
		mux.Handle(StripeWebhookPath+"/", stripeWH)
		log.Printf("[boot] mall stripe webhook registered path=%s", StripeWebhookPath)
	}
}
