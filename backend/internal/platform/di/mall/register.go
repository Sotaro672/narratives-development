// backend/internal/platform/di/mall/register.go
package mall

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	mallhttp "narratives/internal/adapters/in/http/mall"
	mallhandler "narratives/internal/adapters/in/http/mall/handler"
	avatarHandler "narratives/internal/adapters/in/http/mall/handler/avatar"
	mallwebhook "narratives/internal/adapters/in/http/mall/webhook"
	"narratives/internal/adapters/in/http/middleware"
	"narratives/internal/application/usecase"
	avatardom "narratives/internal/domain/avatar"
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

// requireAvatarContext wraps handler with AvatarContextMiddleware (fail-closed).
// This middleware resolves uid -> avatarId and stores it into request context.
// If resolver is not initialized, it returns 503 so the bug is obvious.
func requireAvatarContext(mw *middleware.AvatarContextMiddleware, h http.Handler, name string) http.Handler {
	if h == nil {
		h = http.NotFoundHandler()
	}
	// NOTE:
	// AvatarContextMiddleware now requires Resolver (AvatarResolver) to be non-nil.
	if mw == nil || mw.Resolver == nil {
		log.Printf("[mall.register] ERROR: AvatarContextMiddleware is not initialized (endpoint=%s). returning 503", name)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "avatar_context_not_initialized",
				"name":  name,
			})
		})
	}
	return mw.Handler(h)
}

// transferHTTPAdapter adapts application/usecase.TransferUsecase to HTTP handler's required interface.
type transferHTTPAdapter struct {
	uc *usecase.TransferUsecase
}

func (a transferHTTPAdapter) TransferByScanPurchasedByAvatarID(ctx context.Context, avatarID, productID string) (*mallhandler.ScanTransferResult, error) {
	if a.uc == nil {
		return nil, usecase.ErrTransferNotConfigured
	}

	out, err := a.uc.TransferToAvatarByVerifiedScan(ctx, usecase.TransferByVerifiedScanInput{
		AvatarID:  avatarID,
		ProductID: productID,
	})
	if err != nil {
		// matched=false を 200 で返したい場合はここで吸収
		if err == usecase.ErrTransferNotMatched {
			return &mallhandler.ScanTransferResult{
				AvatarID:  avatarID,
				ProductID: productID,
				Matched:   false,
			}, nil
		}
		return nil, err
	}

	return &mallhandler.ScanTransferResult{
		AvatarID:         avatarID,
		ProductID:        productID,
		Matched:          true,
		TxSignature:      strings.TrimSpace(out.TxSignature),
		FromWallet:       strings.TrimSpace(out.FromWallet),
		ToWallet:         strings.TrimSpace(out.ToWallet),
		UpdatedToAddress: true, // TransferUsecase 内で UpdateToAddressByProductID を実行済み想定
	}, nil
}

// ------------------------------------------------------------
// ✅ MeAvatars DI adapter (NO legacy repo)
// ------------------------------------------------------------
//
// Frontend contract adopts ONLY:
//   - GET   /mall/me/avatars
//   - PATCH /mall/me/avatars
//
// This adapter resolves uid -> avatarId(+walletAddress) and then (best-effort)
// fetches/updates avatar entity via AvatarUsecase (anti-spoof).
type meAvatarExtendedRepo interface {
	ResolveAvatarByUID(ctx context.Context, uid string) (avatarId string, walletAddress string, err error)
	ResolveAvatarIDByUID(ctx context.Context, uid string) (string, error)
}

// Avatar usecase port (Get + Update). Required for PATCH(/mall/me/avatars).
type avatarUsecasePort interface {
	GetByID(ctx context.Context, id string) (avatardom.Avatar, error)
	Update(ctx context.Context, id string, patch avatardom.AvatarPatch) (avatardom.Avatar, error)
}

// adapter that always satisfies mallhandler.MeAvatarService expected by NewMeAvatarHandler
// (handler is expected to be routed to /mall/me/avatars by mallhttp.Register)
type meAvatarServiceAdapter struct {
	extended meAvatarExtendedRepo
	avatarUC avatarUsecasePort // optional for GET; required for PATCH
}

func trimPtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
}

func strPtrTrim(s string) *string {
	t := strings.TrimSpace(s)
	if t == "" {
		return nil
	}
	return &t
}

// ResolveAvatarPatchByUID returns avatarId + AvatarPatch (payload for /mall/me/avatars GET).
func (a meAvatarServiceAdapter) ResolveAvatarPatchByUID(ctx context.Context, uid string) (string, avatardom.AvatarPatch, error) {
	if a.extended == nil {
		return "", avatardom.AvatarPatch{}, usecase.ErrTransferNotConfigured
	}

	avatarId, walletAddress, err := a.extended.ResolveAvatarByUID(ctx, uid)
	if err != nil {
		return "", avatardom.AvatarPatch{}, err
	}

	avatarId = strings.TrimSpace(avatarId)
	walletAddress = strings.TrimSpace(walletAddress)

	// Base patch (walletAddress is required by current frontend policy)
	base := avatardom.AvatarPatch{
		UserID:        "", // filled if avatarUC is available
		AvatarName:    nil,
		AvatarIcon:    nil,
		WalletAddress: strPtrTrim(walletAddress),
		Profile:       nil,
		ExternalLink:  nil,
		DeletedAt:     nil,
	}

	// If we cannot fetch avatar, still return base patch.
	if a.avatarUC == nil || avatarId == "" {
		base.Sanitize()
		return avatarId, base, nil
	}

	av, gerr := a.avatarUC.GetByID(ctx, avatarId)
	if gerr != nil {
		// best-effort: still return base patch + avatarId
		base.Sanitize()
		return avatarId, base, nil
	}

	patch := avatardom.AvatarPatch{
		UserID:        strings.TrimSpace(av.UserID),
		AvatarName:    strPtrTrim(av.AvatarName),
		AvatarIcon:    trimPtr(av.AvatarIcon),
		WalletAddress: trimPtr(av.WalletAddress),
		Profile:       trimPtr(av.Profile),
		ExternalLink:  trimPtr(av.ExternalLink),
		DeletedAt:     av.DeletedAt,
	}

	// Ensure walletAddress is at least the resolved one (server truth)
	if patch.WalletAddress == nil {
		patch.WalletAddress = strPtrTrim(walletAddress)
	}

	patch.Sanitize()
	return avatarId, patch, nil
}

// UpdateAvatarPatchByUID applies patch to "me" avatar resolved from uid (anti-spoof).
// (used by /mall/me/avatars PATCH)
func (a meAvatarServiceAdapter) UpdateAvatarPatchByUID(ctx context.Context, uid string, patch avatardom.AvatarPatch) (string, avatardom.AvatarPatch, error) {
	if a.extended == nil {
		return "", avatardom.AvatarPatch{}, usecase.ErrTransferNotConfigured
	}
	if a.avatarUC == nil {
		return "", avatardom.AvatarPatch{}, errors.New("avatar usecase not configured")
	}

	avatarId, walletAddress, err := a.extended.ResolveAvatarByUID(ctx, uid)
	if err != nil {
		return "", avatardom.AvatarPatch{}, err
	}

	avatarId = strings.TrimSpace(avatarId)
	walletAddress = strings.TrimSpace(walletAddress)
	if avatarId == "" {
		return "", avatardom.AvatarPatch{}, avatardom.ErrInvalidID
	}

	// Anti-spoof: client cannot update these via /mall/me/avatars
	patch.UserID = ""
	patch.WalletAddress = nil
	patch.DeletedAt = nil

	patch.Sanitize()

	updated, uerr := a.avatarUC.Update(ctx, avatarId, patch)
	if uerr != nil {
		return "", avatardom.AvatarPatch{}, uerr
	}

	out := avatardom.AvatarPatch{
		UserID:        strings.TrimSpace(updated.UserID),
		AvatarName:    strPtrTrim(updated.AvatarName),
		AvatarIcon:    trimPtr(updated.AvatarIcon),
		WalletAddress: trimPtr(updated.WalletAddress),
		Profile:       trimPtr(updated.Profile),
		ExternalLink:  trimPtr(updated.ExternalLink),
		DeletedAt:     updated.DeletedAt,
	}

	// Ensure walletAddress is at least the resolved one (server truth)
	if out.WalletAddress == nil {
		out.WalletAddress = strPtrTrim(walletAddress)
	}

	out.Sanitize()
	return avatarId, out, nil
}

func (a meAvatarServiceAdapter) ResolveAvatarByUID(ctx context.Context, uid string) (string, string, error) {
	if a.extended == nil {
		return "", "", usecase.ErrTransferNotConfigured
	}
	return a.extended.ResolveAvatarByUID(ctx, uid)
}

func (a meAvatarServiceAdapter) ResolveAvatarIDByUID(ctx context.Context, uid string) (string, error) {
	if a.extended == nil {
		return "", usecase.ErrTransferNotConfigured
	}
	return a.extended.ResolveAvatarIDByUID(ctx, uid)
}

// ------------------------------------------------------------
// ✅ AvatarResolver adapter
// ------------------------------------------------------------
type avatarResolverAdapter struct {
	me meAvatarExtendedRepo
}

func (a avatarResolverAdapter) ResolveAvatarByUID(ctx context.Context, uid string) (string, string, error) {
	if a.me == nil {
		return "", "", usecase.ErrTransferNotConfigured
	}
	return a.me.ResolveAvatarByUID(ctx, uid)
}

// Register registers mall routes onto mux.
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
		log.Printf("[mall.register] WARN: cont.Infra or cont.Infra.FirebaseAuth is nil (user auth will return 503 on protected endpoints)")
		userAuthMW = &middleware.UserAuthMiddleware{FirebaseAuth: nil}
	}

	// ------------------------------------------------------------
	// Avatar context middleware (uid -> avatarId (+walletAddress))
	// ------------------------------------------------------------
	var avatarCtxMW *middleware.AvatarContextMiddleware
	{
		var ext meAvatarExtendedRepo
		if cont.MeAvatarRepo != nil {
			if v, ok := any(cont.MeAvatarRepo).(meAvatarExtendedRepo); ok {
				ext = v
			}
		}

		if ext != nil {
			avatarCtxMW = &middleware.AvatarContextMiddleware{
				Resolver: avatarResolverAdapter{me: ext},
			}
		} else {
			log.Printf("[mall.register] WARN: cont.MeAvatarRepo does not implement meAvatarExtendedRepo (avatar context will return 503 on endpoints requiring avatarId)")
			avatarCtxMW = &middleware.AvatarContextMiddleware{Resolver: nil}
		}
	}

	// ----------------------------
	// Handlers (construct only)
	// ----------------------------
	listH := notImplemented("List")
	invH := notImplemented("Inventory")
	pbH := notImplemented("ProductBlueprint")
	modelH := notImplemented("Model")
	catalogH := notImplemented("Catalog")
	tbH := notImplemented("TokenBlueprint")
	companyH := notImplemented("Company")
	brandH := notImplemented("Brand")

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
	meAvatarsH := notImplemented("MeAvatars")

	previewPublicH := notImplemented("PreviewPublic")
	previewMeH := notImplemented("PreviewMe")

	orderScanVerifyH := notImplemented("OrderScanVerify")
	orderScanTransferH := notImplemented("OrderScanTransfer")

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
		avatarH = avatarHandler.NewAvatarHandler(cont.AvatarUC)
	}

	// Wallet（AvatarUC は渡さない）
	if cont.WalletUC != nil {
		walletH = mallhandler.NewMallWalletHandler(cont.WalletUC)
	}

	// /mall/me/avatars (uid -> avatarId + avatar patch, and PATCH update)
	if cont.MeAvatarRepo != nil {
		var ext meAvatarExtendedRepo
		if v, ok := any(cont.MeAvatarRepo).(meAvatarExtendedRepo); ok {
			ext = v
		}
		if ext != nil {
			var avuc avatarUsecasePort
			if cont.AvatarUC != nil {
				avuc = cont.AvatarUC
			}

			// NOTE:
			// - GET は avatarUC が nil でも動く（best-effort）
			// - PATCH は avatarUC.Update が必要
			if avuc != nil {
				meAvatarsH = mallhandler.NewMeAvatarHandler(meAvatarServiceAdapter{
					extended: ext,
					avatarUC: avuc,
				})
			} else {
				log.Printf("[mall.register] WARN: cont.AvatarUC is nil (MeAvatars PATCH requires AvatarUC.Update; MeAvatars will return 501)")
			}
		} else {
			log.Printf("[mall.register] WARN: cont.MeAvatarRepo does not implement meAvatarExtendedRepo (MeAvatars will return 501)")
		}
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

	// Preview handler wiring (split)
	if cont.PreviewQ != nil {
		if cont.OwnerResolveQ != nil {
			previewPublicH = mallhandler.NewPreviewHandlerWithOwner(cont.PreviewQ, cont.OwnerResolveQ)
			previewMeH = mallhandler.NewPreviewMeHandlerWithOwner(cont.PreviewQ, cont.OwnerResolveQ)
			log.Printf("[mall.register] preview handlers wired WITH owner-resolve query")
		} else {
			previewPublicH = mallhandler.NewPreviewHandler(cont.PreviewQ)
			previewMeH = mallhandler.NewPreviewMeHandler(cont.PreviewQ)
			log.Printf("[mall.register] preview handlers wired WITHOUT owner-resolve query (OwnerResolveQ is nil)")
		}
	}

	// Order scan verify wiring
	if cont.OrderScanVerifyQ != nil {
		orderScanVerifyH = mallhandler.NewOrderScanVerifyHandler(cont.OrderScanVerifyQ)
		log.Printf("[mall.register] order scan verify handler wired")
	} else {
		log.Printf("[mall.register] WARN: OrderScanVerifyQ is nil (order scan verify will return 501)")
	}

	// Order scan transfer wiring
	if cont.TransferUC != nil {
		httpUC := transferHTTPAdapter{uc: cont.TransferUC}
		orderScanTransferH = mallhandler.NewTransferHandler(httpUC)
		log.Printf("[mall.register] order scan transfer handler wired (TransferUC=%t)", cont.TransferUC != nil)
	} else {
		log.Printf("[mall.register] WARN: TransferUC is nil (order scan transfer will return 501)")
	}

	// SignIn: keep a stable no-op endpoint
	signInH := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	// ------------------------------------------------------------
	// Apply UserAuthMiddleware / AvatarContextMiddleware
	// ------------------------------------------------------------
	userH = requireUserAuth(userAuthMW, userH, "User")
	shipH = requireUserAuth(userAuthMW, shipH, "ShippingAddress")
	billH = requireUserAuth(userAuthMW, billH, "BillingAddress")
	avatarH = requireUserAuth(userAuthMW, avatarH, "Avatar")
	avatarStateH = requireUserAuth(userAuthMW, avatarStateH, "AvatarState")

	// Wallet は同期APIで avatarId が必須なので AvatarContext を通す
	walletH = requireAvatarContext(avatarCtxMW, walletH, "Wallet:AvatarContext")
	walletH = requireUserAuth(userAuthMW, walletH, "Wallet")

	// /mall/me/avatars は uid だけで動く想定（AvatarContext は不要）
	meAvatarsH = requireUserAuth(userAuthMW, meAvatarsH, "MeAvatars")
	cartH = requireUserAuth(userAuthMW, cartH, "Cart")

	previewMeH = requireAvatarContext(avatarCtxMW, previewMeH, "Preview(me):AvatarContext")
	previewMeH = requireUserAuth(userAuthMW, previewMeH, "Preview(me)")

	orderScanVerifyH = requireAvatarContext(avatarCtxMW, orderScanVerifyH, "OrderScanVerify(me):AvatarContext")
	orderScanVerifyH = requireUserAuth(userAuthMW, orderScanVerifyH, "OrderScanVerify(me)")

	orderScanTransferH = requireAvatarContext(avatarCtxMW, orderScanTransferH, "OrderScanTransfer(me):AvatarContext")
	orderScanTransferH = requireUserAuth(userAuthMW, orderScanTransferH, "OrderScanTransfer(me)")

	payH = requireUserAuth(userAuthMW, payH, "Payment")
	orderH = requireUserAuth(userAuthMW, orderH, "Order")
	invoiceH = requireUserAuth(userAuthMW, invoiceH, "Invoice")

	// ----------------------------
	// Router deps
	// ----------------------------
	deps := mallhttp.Deps{
		List: listH,

		Inventory:        invH,
		ProductBlueprint: pbH,
		Model:            modelH,
		Catalog:          catalogH,
		TokenBlueprint:   tbH,

		Company: companyH,
		Brand:   brandH,

		SignIn: signInH,

		User:            userH,
		ShippingAddress: shipH,
		BillingAddress:  billH,
		Avatar:          avatarH,

		// ✅ deps のフィールド名は MeAvatar のままでもOK（ルーティング側で /mall/me/avatars に割り当てる）
		MeAvatar: meAvatarsH,

		AvatarState: avatarStateH,
		Wallet:      walletH,
		Cart:        cartH,

		Preview:   previewPublicH,
		PreviewMe: previewMeH,

		OrderScanVerify:   orderScanVerifyH,
		OrderScanTransfer: orderScanTransferH,
		OwnerResolve:      notImplemented("OwnerResolve(endpoint_disabled)"),
		Payment:           payH,
		Order:             orderH,
		Invoice:           invoiceH,
	}

	// ✅ ここは DI だけなので “そのまま” でOK
	mallhttp.Register(mux, deps, userAuthMW.Handler)

	log.Printf("[boot] mall routes registered")

	// ----------------------------
	// Webhooks (no auth)
	// ----------------------------
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

// NOTE:
// This file already imports time; keep it even if not referenced elsewhere.
// Some builds may trim it if unused in future edits.
var _ = time.RFC3339
