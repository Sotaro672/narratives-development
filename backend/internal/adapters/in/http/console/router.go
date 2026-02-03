// backend/internal/adapters/in/http/console/router.go
package httpin

import (
	"net/http"

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
	AuthBootstrap   http.Handler
	Accounts        http.Handler
	Announcements   http.Handler
	Permissions     http.Handler
	Brands          http.Handler
	Companies       http.Handler
	Inquiries       http.Handler
	Inventories     http.Handler
	Lists           http.Handler
	ProductsPrint   http.Handler
	ProductBP       http.Handler
	TokenBP         http.Handler
	Messages        http.Handler
	Orders          http.Handler
	Wallets         http.Handler
	Members         http.Handler
	Productions     http.Handler
	Models          http.Handler
	Inspector       http.Handler
	Mint            http.Handler
	OwnerResolve    http.Handler
	MintDebugHandle http.HandlerFunc // optional（/mint/debug）
}

func NewRouter(deps RouterDeps) http.Handler {
	mux := http.NewServeMux()

	withAuth := func(h http.Handler) http.Handler {
		if deps.AuthMw == nil {
			return h
		}
		return deps.AuthMw.Handler(h)
	}
	withBootstrap := func(h http.Handler) http.Handler {
		if deps.BootstrapMw == nil {
			return h
		}
		return deps.BootstrapMw.Handler(h)
	}

	// ================================
	// /auth/bootstrap
	// ================================
	if deps.AuthBootstrap != nil {
		mux.Handle("/auth/bootstrap", withBootstrap(deps.AuthBootstrap))
	}

	// ================================
	// Accounts
	// ================================
	if deps.Accounts != nil {
		h := withAuth(deps.Accounts)
		mux.Handle("/accounts", h)
		mux.Handle("/accounts/", h)
	}

	// ================================
	// Announcements
	// ================================
	if deps.Announcements != nil {
		h := withAuth(deps.Announcements)
		mux.Handle("/announcements", h)
		mux.Handle("/announcements/", h)
	}

	// ================================
	// Permissions
	// ================================
	if deps.Permissions != nil {
		h := withAuth(deps.Permissions)
		mux.Handle("/permissions", h)
		mux.Handle("/permissions/", h)
	}

	// ================================
	// Brands
	// ================================
	if deps.Brands != nil {
		h := withAuth(deps.Brands)
		mux.Handle("/brands", h)
		mux.Handle("/brands/", h)
	}

	// ================================
	// Companies
	// ================================
	if deps.Companies != nil {
		h := withAuth(deps.Companies)
		mux.Handle("/companies", h)
		mux.Handle("/companies/", h)
	}

	// ================================
	// Inquiries
	// ================================
	if deps.Inquiries != nil {
		h := withAuth(deps.Inquiries)
		mux.Handle("/inquiries", h)
		mux.Handle("/inquiries/", h)
	}

	// ================================
	// Inventories
	// ================================
	if deps.Inventories != nil {
		h := withAuth(deps.Inventories)

		mux.Handle("/inventories", h)
		mux.Handle("/inventories/", h)

		mux.Handle("/inventory", h)
		mux.Handle("/inventory/", h)
	}

	// ================================
	// Lists
	// ================================
	if deps.Lists != nil {
		h := withAuth(deps.Lists)
		mux.Handle("/lists", h)
		mux.Handle("/lists/", h)
	}

	// ================================
	// Products（印刷系）
	// ================================
	if deps.ProductsPrint != nil {
		h := withAuth(deps.ProductsPrint)
		mux.Handle("/products", h)
		mux.Handle("/products/", h)
		mux.Handle("/products/print-logs", h)
	}

	// ================================
	// Product Blueprints
	// ================================
	if deps.ProductBP != nil {
		h := withAuth(deps.ProductBP)
		mux.Handle("/product-blueprints", h)
		mux.Handle("/product-blueprints/", h)
	}

	// ================================
	// Token Blueprints
	// ================================
	if deps.TokenBP != nil {
		h := withAuth(deps.TokenBP)
		mux.Handle("/token-blueprints", h)
		mux.Handle("/token-blueprints/", h)
	}

	// ================================
	// Messages
	// ================================
	if deps.Messages != nil {
		h := withAuth(deps.Messages)
		mux.Handle("/messages", h)
		mux.Handle("/messages/", h)
	}

	// ================================
	// Orders
	// ================================
	if deps.Orders != nil {
		h := withAuth(deps.Orders)
		mux.Handle("/orders", h)
		mux.Handle("/orders/", h)
	}

	// ================================
	// Wallets
	// ================================
	if deps.Wallets != nil {
		h := withAuth(deps.Wallets)
		mux.Handle("/wallets", h)
		mux.Handle("/wallets/", h)
	}

	// ================================
	// Members
	// ================================
	if deps.Members != nil {
		h := withAuth(deps.Members)
		mux.Handle("/members", h)
		mux.Handle("/members/", h)
	}

	// ================================
	// Productions
	// ================================
	if deps.Productions != nil {
		h := withAuth(deps.Productions)
		mux.Handle("/productions", h)
		mux.Handle("/productions/", h)
	}

	// ================================
	// Models
	// ================================
	if deps.Models != nil {
		h := withAuth(deps.Models)
		mux.Handle("/models", h)
		mux.Handle("/models/", h)
	}

	// ================================
	// Inspector
	// ================================
	if deps.Inspector != nil {
		h := withAuth(deps.Inspector)
		mux.Handle("/inspector/products/", h)
		mux.Handle("/products/inspections", h)
		mux.Handle("/products/inspections/", h)
	}

	// ================================
	// Mint
	// ================================
	if deps.Mint != nil {
		h := withAuth(deps.Mint)
		mux.Handle("/mint/", h)

		// /mint/debug（任意）
		if deps.MintDebugHandle != nil {
			mux.HandleFunc("/mint/debug", deps.MintDebugHandle)
		}
	}

	// ================================
	// Owner resolve (walletAddress/toAddress -> avatarId or brandId)
	// ================================
	if deps.OwnerResolve != nil {
		h := withAuth(deps.OwnerResolve)
		mux.Handle("/owners/resolve", h)
		mux.Handle("/owners/resolve/", h)
	}

	return mux
}
