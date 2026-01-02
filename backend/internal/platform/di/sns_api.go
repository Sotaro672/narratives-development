// backend/internal/platform/di/sns_api.go
package di

import (
	"log"
	"net/http"
	"strings"

	firebaseauth "firebase.google.com/go/v4/auth"

	"narratives/internal/adapters/in/http/middleware"
	snshttp "narratives/internal/adapters/in/http/sns"
)

// SNSAPI is a fixed (non best-effort, non-reflection) wiring surface for buyer-facing (sns) routes.
//   - All dependencies are explicit fields.
//   - No Container introspection here. (No reflect, no "try multiple names").
//   - Optional handlers can be nil (route simply won't be registered by snshttp.Register if it guards nil,
//     or will be registered as nil and panic if snshttp doesn't guard; prefer setting a no-op handler if needed).
type SNSAPI struct {
	// Core public endpoints
	List             http.Handler
	Inventory        http.Handler
	ProductBlueprint http.Handler
	Model            http.Handler
	Catalog          http.Handler
	TokenBlueprint   http.Handler

	// Org/name resolver endpoints
	Company http.Handler
	Brand   http.Handler

	// Auth entry (should remain public; DO NOT wrap with buyer-auth)
	SignIn http.Handler

	// Auth onboarding resources (buyer-auth required)
	User            http.Handler
	ShippingAddress http.Handler
	BillingAddress  http.Handler
	Avatar          http.Handler

	// Buyer-auth required
	AvatarState http.Handler
	Wallet      http.Handler

	// Cart / Preview
	// - CartWrite handles POST/PUT/DELETE (write-model).
	// - CartQuery handles GET /sns/cart, /sns/cart/, /sns/cart/query* (read-model).
	// - Preview is typically a separate handler (e.g. /sns/preview), but can be same as CartWrite if designed so.
	CartWrite http.Handler
	CartQuery http.Handler
	Preview   http.Handler

	// Posts
	Post http.Handler

	// Payment / Checkout (buyer-auth required)
	Payment http.Handler

	// Middleware dependency
	FirebaseAuth *firebaseauth.Client

	// Optional: set true to emit wiring logs
	Debug bool
}

// Register registers SNS routes onto mux using fixed wiring.
// This is the only place that composes:
//   - cart query/write split handler
//   - buyer-auth middleware wrapping
//   - final mux registration
func (a SNSAPI) Register(mux *http.ServeMux) {
	if mux == nil {
		return
	}

	// 1) Build merged handlers (cart split, preview fallback)
	cartH := a.buildCartHandler()
	previewH := a.Preview
	if previewH == nil {
		// If your implementation allows CartWrite to serve /sns/preview, keep it as fallback.
		previewH = a.CartWrite
	}

	// 2) Apply buyer-auth middleware (only for protected endpoints)
	userAuth := a.newUserAuthMiddleware()
	if userAuth == nil {
		if a.Debug {
			log.Printf("[sns_api] WARN: user_auth middleware is not available (firebase auth client missing). protected routes may 401/500 depending on handlers.")
		}
	} else {
		wrap := func(h http.Handler) http.Handler {
			if h == nil {
				return nil
			}
			return userAuth.Handler(h)
		}

		// NOTE: SignIn must remain public (no token required).
		a.User = wrap(a.User)
		a.ShippingAddress = wrap(a.ShippingAddress)
		a.BillingAddress = wrap(a.BillingAddress)
		a.Avatar = wrap(a.Avatar)

		a.AvatarState = wrap(a.AvatarState)
		a.Wallet = wrap(a.Wallet)
		cartH = wrap(cartH)
		previewH = wrap(previewH)
		a.Post = wrap(a.Post)
		a.Payment = wrap(a.Payment)
	}

	// 3) Register into mux
	snshttp.Register(mux, snshttp.Deps{
		List:             a.List,
		Inventory:        a.Inventory,
		ProductBlueprint: a.ProductBlueprint,
		Model:            a.Model,
		Catalog:          a.Catalog,

		TokenBlueprint: a.TokenBlueprint,

		Company: a.Company,
		Brand:   a.Brand,

		SignIn: a.SignIn,

		User:            a.User,
		ShippingAddress: a.ShippingAddress,
		BillingAddress:  a.BillingAddress,
		Avatar:          a.Avatar,

		AvatarState: a.AvatarState,
		Wallet:      a.Wallet,
		Cart:        cartH,
		Preview:     previewH,
		Post:        a.Post,

		Payment: a.Payment,
	})

	if a.Debug {
		log.Printf("[sns_api] registered: signIn=%t cartWrite=%t cartQuery=%t payment=%t fbAuth=%t",
			a.SignIn != nil,
			a.CartWrite != nil,
			a.CartQuery != nil,
			a.Payment != nil,
			a.FirebaseAuth != nil,
		)
	}
}

// buildCartHandler composes CartWrite and CartQuery into one handler.
// Policy (same intent as your current sns_container.go):
// - GET /sns/cart, /sns/cart/, /sns/cart/query*  => CartQuery
// - otherwise                                   => CartWrite
func (a SNSAPI) buildCartHandler() http.Handler {
	// If there is no split, prefer CartWrite as a single handler.
	if a.CartQuery == nil {
		return a.CartWrite
	}
	if a.CartWrite == nil {
		// Read-only mode (still valid for /sns/cart GET)
		return a.CartQuery
	}

	qh := a.CartQuery
	wh := a.CartWrite

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil {
			wh.ServeHTTP(w, r)
			return
		}

		if r.Method == http.MethodGet {
			p := r.URL.Path
			if p == "/sns/cart" || p == "/sns/cart/" || strings.HasPrefix(p, "/sns/cart/query") {
				qh.ServeHTTP(w, r)
				return
			}
		}

		wh.ServeHTTP(w, r)
	})
}

func (a SNSAPI) newUserAuthMiddleware() *middleware.UserAuthMiddleware {
	if a.FirebaseAuth == nil {
		return nil
	}
	return &middleware.UserAuthMiddleware{FirebaseAuth: a.FirebaseAuth}
}

// SNSProvider is an optional compile-time contract for Container (or another module) to expose SNSAPI.
// Implement this interface and then your router layer can do:
//
//	var p di.SNSProvider = cont
//	p.SNS().Register(mux)
type SNSProvider interface {
	SNS() SNSAPI
}
