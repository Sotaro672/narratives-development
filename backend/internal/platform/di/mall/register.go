// backend/internal/platform/di/mall/register.go
package mall

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	mallhttp "narratives/internal/adapters/in/http/mall"
	mallhandler "narratives/internal/adapters/in/http/mall/handler"
	mallwebhook "narratives/internal/adapters/in/http/mall/webhook"
	"narratives/internal/adapters/in/http/middleware"
	"narratives/internal/application/usecase"
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
	// Avatar context middleware (uid -> avatarId)
	// ------------------------------------------------------------
	var avatarCtxMW *middleware.AvatarContextMiddleware
	if cont.OrderQ != nil {
		avatarCtxMW = &middleware.AvatarContextMiddleware{
			Resolver:      cont.OrderQ,
			AllowExplicit: false,
		}
	} else {
		log.Printf("[mall.register] WARN: cont.OrderQ is nil (avatar context will return 503 on endpoints requiring avatarId)")
		avatarCtxMW = &middleware.AvatarContextMiddleware{Resolver: nil, AllowExplicit: false}
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
	meAvatarH := notImplemented("MeAvatar")

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
	// ------------------------------------------------------------
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

	// ------------------------------------------------------------
	// Order scan verify wiring
	// ------------------------------------------------------------
	if cont.OrderScanVerifyQ != nil {
		orderScanVerifyH = mallhandler.NewOrderScanVerifyHandler(cont.OrderScanVerifyQ)
		log.Printf("[mall.register] order scan verify handler wired")
	} else {
		log.Printf("[mall.register] WARN: OrderScanVerifyQ is nil (order scan verify will return 501)")
	}

	// ------------------------------------------------------------
	// Order scan transfer wiring
	// ------------------------------------------------------------
	// ✅ TransferUsecase は Container 側で生成して持たせる（register.go は配線だけ）
	if cont.TransferUC != nil {
		httpUC := transferHTTPAdapter{uc: cont.TransferUC}

		// anti-spoof を handler 内でもやりたいなら WithResolver
		if cont.OrderQ != nil {
			orderScanTransferH = mallhandler.NewTransferHandlerWithResolver(httpUC, cont.OrderQ)
		} else {
			orderScanTransferH = mallhandler.NewTransferHandler(httpUC)
		}
		log.Printf("[mall.register] order scan transfer handler wired (TransferUC=%t)", cont.TransferUC != nil)
	} else {
		log.Printf("[mall.register] WARN: TransferUC is nil (order scan transfer will return 501)")
	}

	// SignIn: keep a stable no-op endpoint
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
		MeAvatar:        meAvatarH,
		AvatarState:     avatarStateH,
		Wallet:          walletH,
		Cart:            cartH,

		Preview:   previewPublicH,
		PreviewMe: previewMeH,

		OrderScanVerify:   orderScanVerifyH,
		OrderScanTransfer: orderScanTransferH,
		OwnerResolve:      notImplemented("OwnerResolve(endpoint_disabled)"),
		Payment:           payH,
		Order:             orderH,
		Invoice:           invoiceH,
	}

	mallhttp.Register(mux, deps)
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
