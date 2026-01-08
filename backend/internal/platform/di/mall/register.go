// backend/internal/platform/di/mall/register.go
package mall

import (
	"log"
	"net/http"

	mallhttp "narratives/internal/adapters/in/http/mall"
	mallhandler "narratives/internal/adapters/in/http/mall/handler"
	mallwebhook "narratives/internal/adapters/in/http/mall/webhook"
)

// Register registers mall routes onto mux.
// Pure DI: construct handlers and pass into mall router.Register.
// - No method/path branching here
// - No auth middleware wrapping here
// - Nil handlers are OK: mall router will register NotFoundHandler for nil deps
func Register(mux *http.ServeMux, cont *Container) {
	if mux == nil || cont == nil {
		return
	}

	// ----------------------------
	// Handlers (construct only)
	// ----------------------------
	var (
		listH   http.Handler
		userH   http.Handler
		shipH   http.Handler
		billH   http.Handler
		walletH http.Handler
		cartH   http.Handler
		prevH   http.Handler
		payH    http.Handler
		orderH  http.Handler
	)

	// ✅ Lists: this is the one that was 501 when not wired
	if cont.ListUC != nil {
		listH = mallhandler.NewMallListHandler(cont.ListUC)
	}

	if cont.UserUC != nil {
		userH = mallhandler.NewUserHandler(cont.UserUC)
	}
	if cont.ShippingAddressUC != nil {
		shipH = mallhandler.NewShippingAddressHandler(cont.ShippingAddressUC)
	}
	if cont.BillingAddressUC != nil {
		billH = mallhandler.NewBillingAddressHandler(cont.BillingAddressUC)
	}
	if cont.WalletUC != nil {
		walletH = mallhandler.NewWalletHandler(cont.WalletUC)
	}

	// Cart / Preview
	if cont.CartUC != nil {
		// this handler is expected to support both cart + preview behaviors via injected queries
		cartH = mallhandler.NewCartHandlerWithQueries(cont.CartUC, cont.CartQ, cont.PreviewQ)
		prevH = cartH
	} else if cont.CartQ != nil {
		// read-only fallback
		cartH = mallhandler.NewCartQueryHandler(cont.CartQ)
		// preview remains nil
	}

	// Payment / Order
	if cont.PaymentUC != nil {
		payH = mallhandler.NewPaymentHandlerWithOrderQuery(cont.PaymentUC, cont.OrderQ)
	}
	if cont.OrderUC != nil {
		orderH = mallhandler.NewOrderHandler(cont.OrderUC)
	}

	// SignIn: keep a stable no-op endpoint (client convenience)
	signInH := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	// ----------------------------
	// Router deps
	// ----------------------------
	deps := mallhttp.Deps{
		// public browsing
		List: listH,

		// keep nil unless you wire them
		Inventory:        nil,
		ProductBlueprint: nil,
		Model:            nil,
		Catalog:          nil,
		TokenBlueprint:   nil,

		Company: nil,
		Brand:   nil,

		SignIn: signInH,

		User:            userH,
		ShippingAddress: shipH,
		BillingAddress:  billH,
		Avatar:          nil,
		AvatarState:     nil,
		Wallet:          walletH,

		Cart:    cartH,
		Preview: prevH,

		Post:    nil,
		Payment: payH,
		Order:   orderH,
	}

	mallhttp.Register(mux, deps)
	log.Printf("[boot] mall routes registered")

	// ----------------------------
	// Webhooks (no auth)
	// ----------------------------
	if cont.InvoiceUC != nil && cont.PaymentUC != nil {
		stripeWH := mallwebhook.NewStripeWebhookHandler(cont.InvoiceUC, cont.PaymentUC)
		mux.Handle(StripeWebhookPath, stripeWH)
		mux.Handle(StripeWebhookPath+"/", stripeWH)
		log.Printf("[boot] mall stripe webhook registered path=%s", StripeWebhookPath)
	}
}
