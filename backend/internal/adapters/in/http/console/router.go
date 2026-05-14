// backend\internal\adapters\in\http\console\router.go
package httpin

import (
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
)

// RouterDeps は「ルーティングに必要なもの」だけを受け取る。
// ここには Usecase / Repo / 外部クライアント等の“生成材料”は持たせず、
// 生成済みの http.Handler と Middleware だけを渡す。
type RouterDeps struct {
	// Middlewares（生成はDI側）
	AuthMw      *middleware.AuthMiddleware
	BootstrapMw *middleware.BootstrapAuthMiddleware

	// Handlers（生成はDI側）
	AuthBootstrap       http.Handler
	Accounts            http.Handler
	Announcements       http.Handler
	Permissions         http.Handler
	Brands              http.Handler
	Companies           http.Handler
	Inquiries           http.Handler
	Inventories         http.Handler
	Lists               http.Handler
	ProductsPrint       http.Handler
	ProductBP           http.Handler
	ProductBPCategories http.Handler
	TokenBP             http.Handler
	Messages            http.Handler
	Orders              http.Handler
	Wallets             http.Handler
	Members             http.Handler
	MemberInvitation    http.Handler
	Productions         http.Handler
	Models              http.Handler
	Inspector           http.Handler
	Mint                http.Handler
	OwnerResolve        http.Handler
	Users               http.Handler
	Invitation          http.Handler
	Sales               http.Handler

	TokenBPReview   http.Handler
	ProductBPReview http.Handler

	MintDebugHandle http.HandlerFunc
}

func NewRouter(deps RouterDeps) http.Handler {
	mux := http.NewServeMux()

	withAuth := func(h http.Handler) http.Handler {
		h = middleware.CORS(h)
		if deps.AuthMw == nil {
			return h
		}
		return deps.AuthMw.Handler(h)
	}

	withBootstrap := func(h http.Handler) http.Handler {
		h = middleware.CORS(h)
		if deps.BootstrapMw == nil {
			return h
		}
		return deps.BootstrapMw.Handler(h)
	}

	withPublic := func(h http.Handler) http.Handler {
		return middleware.CORS(h)
	}

	if deps.AuthBootstrap != nil {
		mux.Handle("/auth/bootstrap", withBootstrap(deps.AuthBootstrap))
	}

	if deps.Invitation != nil {
		h := withPublic(deps.Invitation)
		mux.Handle("/api/invitation", h)
		mux.Handle("/api/invitation/", h)
	}

	if deps.Accounts != nil {
		h := withAuth(deps.Accounts)
		mux.Handle("/accounts", h)
		mux.Handle("/accounts/", h)
	}

	if deps.Announcements != nil {
		h := withAuth(deps.Announcements)
		mux.Handle("/announcements", h)
		mux.Handle("/announcements/", h)
	}

	if deps.Permissions != nil {
		h := withAuth(deps.Permissions)
		mux.Handle("/permissions", h)
		mux.Handle("/permissions/", h)
	}

	if deps.Brands != nil {
		h := withAuth(deps.Brands)
		mux.Handle("/brands", h)
		mux.Handle("/brands/", h)
	}

	if deps.Companies != nil {
		h := withAuth(deps.Companies)
		mux.Handle("/companies", h)
		mux.Handle("/companies/", h)
	}

	if deps.Inquiries != nil {
		h := withAuth(deps.Inquiries)
		mux.Handle("/inquiries", h)
		mux.Handle("/inquiries/", h)
	}

	if deps.Inventories != nil {
		h := withAuth(deps.Inventories)
		mux.Handle("/inventories", h)
		mux.Handle("/inventories/", h)
		mux.Handle("/inventory", h)
		mux.Handle("/inventory/", h)
	}

	if deps.Lists != nil {
		h := withAuth(deps.Lists)
		mux.Handle("/lists", h)
		mux.Handle("/lists/", h)
	}

	if deps.ProductsPrint != nil {
		h := withAuth(deps.ProductsPrint)
		mux.Handle("/products", h)
		mux.Handle("/products/", h)
		mux.Handle("/products/print-logs", h)
	}

	if deps.ProductBP != nil {
		h := withAuth(deps.ProductBP)
		mux.Handle("/product-blueprints", h)
		mux.Handle("/product-blueprints/", h)
	}

	if deps.ProductBPCategories != nil {
		h := withAuth(deps.ProductBPCategories)
		mux.Handle("/console/product-blueprint-categories", h)
		mux.Handle("/console/product-blueprint-categories/", h)
	}

	if deps.TokenBP != nil {
		h := withAuth(deps.TokenBP)
		mux.Handle("/token-blueprints", h)
		mux.Handle("/token-blueprints/", h)
	}

	if deps.TokenBPReview != nil {
		h := withAuth(deps.TokenBPReview)
		mux.Handle("/token-blueprint-reviews", h)
		mux.Handle("/token-blueprint-reviews/", h)
	}

	if deps.ProductBPReview != nil {
		h := withAuth(deps.ProductBPReview)
		mux.Handle("/product-blueprint-reviews", h)
		mux.Handle("/product-blueprint-reviews/", h)
	}

	if deps.Messages != nil {
		h := withAuth(deps.Messages)
		mux.Handle("/messages", h)
		mux.Handle("/messages/", h)
	}

	if deps.Orders != nil {
		h := withAuth(deps.Orders)
		mux.Handle("/orders", h)
		mux.Handle("/orders/", h)
	}

	if deps.Wallets != nil {
		h := withAuth(deps.Wallets)
		mux.Handle("/wallets", h)
		mux.Handle("/wallets/", h)
	}

	// Members / MemberInvitation
	if deps.Members != nil || deps.MemberInvitation != nil {
		var membersRoot http.Handler
		if deps.Members != nil {
			membersRoot = withAuth(deps.Members)
			mux.Handle("/members", membersRoot)
		}

		membersSubtree := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(strings.TrimRight(r.URL.Path, "/"), "/invitation") && deps.MemberInvitation != nil {
				withAuth(deps.MemberInvitation).ServeHTTP(w, r)
				return
			}
			if deps.Members != nil {
				withAuth(deps.Members).ServeHTTP(w, r)
				return
			}
			http.NotFound(w, r)
		})

		mux.Handle("/members/", membersSubtree)
	}

	if deps.Productions != nil {
		h := withAuth(deps.Productions)
		mux.Handle("/productions", h)
		mux.Handle("/productions/", h)
	}

	if deps.Models != nil {
		h := withAuth(deps.Models)
		mux.Handle("/models", h)
		mux.Handle("/models/", h)
	}

	if deps.Inspector != nil {
		h := withAuth(deps.Inspector)
		mux.Handle("/inspector/products/", h)
		mux.Handle("/products/inspections", h)
		mux.Handle("/products/inspections/", h)
	}

	if deps.Mint != nil {
		h := withAuth(deps.Mint)
		mux.Handle("/mint", h)
		mux.Handle("/mint/", h)
		if deps.MintDebugHandle != nil {
			mux.HandleFunc("/mint/debug", deps.MintDebugHandle)
		}
	}

	if deps.OwnerResolve != nil {
		h := withAuth(deps.OwnerResolve)
		mux.Handle("/owners/resolve", h)
		mux.Handle("/owners/resolve/", h)
	}

	if deps.Sales != nil {
		h := withAuth(deps.Sales)
		mux.Handle("/sales", h)
		mux.Handle("/sales/", h)
	}

	return mux
}
