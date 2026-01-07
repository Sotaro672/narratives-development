// backend/internal/adapters/in/http/mall/router.go
package mall

import (
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

// Register registers buyer-facing routes onto mux (mall only).
func Register(mux *http.ServeMux, deps Deps) {
	if mux == nil {
		return
	}

	// ✅ NOTE:
	// 多くの既存ハンドラは /sns/* 前提（NewSNS*Handler）なので、
	// ルーティングは /mall/* のまま、内部で /sns/* に rewrite して渡す。
	// Cart/Preview/Payment/Order などは /mall 対応済みのことが多いので rewrite しない。

	// lists
	mux.Handle("/mall/lists", deps.List)
	mux.Handle("/mall/lists/", deps.List)

	// inventories
	mux.Handle("/mall/inventories", deps.Inventory)
	mux.Handle("/mall/inventories/", deps.Inventory)

	// product blueprints
	mux.Handle("/mall/product-blueprints", deps.ProductBlueprint)
	mux.Handle("/mall/product-blueprints/", deps.ProductBlueprint)

	// models（Mall対応ハンドラの可能性が高いので rewrite しない）
	mux.Handle("/mall/models", deps.Model)
	mux.Handle("/mall/models/", deps.Model)

	// catalog
	mux.Handle("/mall/catalog", deps.Catalog)
	mux.Handle("/mall/catalog/", deps.Catalog)

	// token blueprints
	mux.Handle("/mall/token-blueprints", deps.TokenBlueprint)
	mux.Handle("/mall/token-blueprints/", deps.TokenBlueprint)

	// companies / brands
	mux.Handle("/mall/companies", deps.Company)
	mux.Handle("/mall/companies/", deps.Company)
	mux.Handle("/mall/brands", deps.Brand)
	mux.Handle("/mall/brands/", deps.Brand)

	// sign-in
	mux.Handle("/mall/sign-in", deps.SignIn)
	mux.Handle("/mall/sign-in/", deps.SignIn)

	// users
	mux.Handle("/mall/users", deps.User)
	mux.Handle("/mall/users/", deps.User)

	// shipping addresses
	mux.Handle("/mall/shipping-addresses", deps.ShippingAddress)
	mux.Handle("/mall/shipping-addresses/", deps.ShippingAddress)

	// billing addresses
	mux.Handle("/mall/billing-addresses", deps.BillingAddress)
	mux.Handle("/mall/billing-addresses/", deps.BillingAddress)

	// avatars
	mux.Handle("/mall/avatars", deps.Avatar)
	mux.Handle("/mall/avatars/", deps.Avatar)

	// avatar states
	mux.Handle("/mall/avatar-states", deps.AvatarState)
	mux.Handle("/mall/avatar-states/", deps.AvatarState)

	// ✅ wallet (plural only)
	mux.Handle("/mall/wallets", deps.Wallet)
	mux.Handle("/mall/wallets/", deps.Wallet)

	// cart（/mall 前提のラッパを DI 側で組んでいるので rewrite しない）
	mux.Handle("/mall/cart", deps.Cart)
	mux.Handle("/mall/cart/", deps.Cart)

	// preview（同上）
	mux.Handle("/mall/preview", deps.Preview)
	mux.Handle("/mall/preview/", deps.Preview)

	// posts
	mux.Handle("/mall/posts", deps.Post)
	mux.Handle("/mall/posts/", deps.Post)

	// ✅ payment（PaymentHandler が /mall を自前で normalize するので rewrite しない）
	mux.Handle("/mall/payment", deps.Payment)
	mux.Handle("/mall/payment/", deps.Payment)
	mux.Handle("/mall/payments", deps.Payment)
	mux.Handle("/mall/payments/", deps.Payment)

	// orders（Mall対応ハンドラの可能性が高いので rewrite しない）
	mux.Handle("/mall/orders", deps.Order)
	mux.Handle("/mall/orders/", deps.Order)
}
