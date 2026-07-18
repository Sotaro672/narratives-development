package mall

import (
	"net/http"
	"os"

	mallhttp "narratives/internal/adapters/in/http/mall"
	mallhandler "narratives/internal/adapters/in/http/mall/handler"
	mallwebhook "narratives/internal/adapters/in/http/mall/webhook"
	"narratives/internal/adapters/in/http/middleware"
	firestoreOut "narratives/internal/adapters/out/firestore"
	mailadp "narratives/internal/adapters/out/mail"
	"narratives/internal/application/usecase"
	tokenBlueprint "narratives/internal/domain/tokenBlueprint"
)

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
		userAuthMW = &middleware.UserAuthMiddleware{FirebaseAuth: nil}
	}

	// ------------------------------------------------------------
	// Avatar context middleware (uid -> avatarId (+walletAddress))
	// ------------------------------------------------------------
	var avatarCtxMW *middleware.AvatarContextMiddleware
	{
		if cont.MeAvatarResolver != nil {
			avatarCtxMW = &middleware.AvatarContextMiddleware{
				Resolver: cont.MeAvatarResolver,
			}
		} else {
			avatarCtxMW = &middleware.AvatarContextMiddleware{
				Resolver: nil,
			}
		}
	}

	// ------------------------------------------------------------
	// Local repositories recreated from Firestore when needed
	// ------------------------------------------------------------
	var tokenBlueprintRepo tokenBlueprint.RepositoryPort
	{
		hasFS := cont.Infra != nil && cont.Infra.Firestore != nil
		if hasFS {
			repo := firestoreOut.NewTokenBlueprintRepositoryFS(
				cont.Infra.Firestore,
			)
			tokenBlueprintRepo = repo
		}
	}

	// ----------------------------
	// Handlers (construct only)
	// ----------------------------
	listH := notImplemented("List")
	invH := notImplemented("Inventory")
	pbH := notImplemented("ProductBlueprint")
	catalogH := notImplemented("Catalog")
	tbH := notImplemented("TokenBlueprint")

	pbReviewH := notImplemented("ProductBlueprintReview")
	tbReviewH := notImplemented("TokenBlueprintReview")

	companyH := notImplemented("Company")
	brandH := notImplemented("Brand")

	authH := notImplemented("Auth")

	userH := notImplemented("User")
	shipH := notImplemented("ShippingAddress")
	paymentMethodH := notImplemented("PaymentMethod")
	avatarH := notImplemented("Avatar")
	walletH := notImplemented("Wallet")
	meWalletH := notImplemented("MeWallet")
	cartH := notImplemented("Cart")
	payH := notImplemented("Payment")
	orderH := notImplemented("Order")
	inquiryH := notImplemented("Inquiry")
	meAvatarsH := notImplemented("MeAvatars")
	announcementH := notImplemented("Announcement")

	marketH := notImplemented("Market")
	resaleH := notImplemented("Resale")

	previewPublicH := notImplemented("PreviewPublic")
	previewMeH := notImplemented("PreviewMe")

	orderScanVerifyH := notImplemented("OrderScanVerify")
	orderScanTransferH := transferUsecaseNotConfiguredHandler()

	setupStatusH := notImplemented("SetupStatus")

	// Auth email verification
	if cont.Infra != nil && cont.Infra.FirebaseAuth != nil {
		authH = mallhandler.NewAuthHandler(
			cont.Infra.FirebaseAuth,
			mailadp.NewAuthMailerWithResend(),
			os.Getenv("AUTH_ACTION_BASE_URL"),
		)
	}

	// Lists (public)
	if cont.ListQ != nil {
		listH = mallhandler.NewMallListHandler(cont.ListQ)
	}

	// Catalog (public)
	if cont.CatalogQ != nil {
		catalogH = mallhandler.NewMallCatalogHandler(cont.CatalogQ)
	}

	// ProductBlueprintReview wiring (catalog composite)
	if cont.ProductBlueprintReviewUC != nil {
		pbReviewH = mallhandler.NewProductBlueprintReviewHandler(
			cont.ProductBlueprintReviewUC,
		)
		catalogH = newCatalogCompositeHandler(catalogH, pbReviewH)
	}

	// Brand
	if cont.BrandQ != nil {
		brandH = mallhandler.NewMallBrandHandler(cont.BrandQ)
	}

	// Avatar
	if cont.AvatarUC != nil {
		avatarH = mallhandler.NewAvatarHandler(cont.AvatarUC)
	}

	// TokenBlueprintReview wiring
	if cont.TokenBlueprintReviewRepo != nil {
		tbReviewUC := usecase.NewTokenBlueprintReviewUsecase(
			cont.TokenBlueprintReviewRepo,
			cont.AvatarRepo,
			tokenBlueprintRepo,
			cont.BrandRepo,
		)

		tbReviewH = mallhandler.NewTokenBlueprintReviewHandler(tbReviewUC)
	}

	if tbReviewH != nil && tbReviewH != http.NotFoundHandler() {
		tbH = mallhandler.NewTokenBlueprintCompositeHandler(
			tbH,
			tbReviewH,
		)
	}

	// Core resources
	if cont.UserUC != nil {
		userH = mallhandler.NewUserHandler(cont.UserUC)
	}

	if cont.ShippingAddressUC != nil {
		shipH = mallhandler.NewShippingAddressHandler(
			cont.ShippingAddressUC,
		)
	}

	if cont.PaymentMethodUC != nil {
		paymentMethodH = mallhandler.NewPaymentMethodHandler(
			cont.PaymentMethodUC,
		)
	}

	// Wallet (me)
	if cont.WalletUC != nil {
		meWalletH = mallhandler.NewMallMeWalletHandler(cont.WalletUC)
	}

	// /mall/me/avatars
	if cont.MeAvatarResolver != nil && cont.AvatarUC != nil {
		meAvatarsH = mallhandler.NewMeAvatarHandler(
			cont.MeAvatarResolver,
			cont.AvatarUC,
		)
	}

	// /mall/me/announcements
	if cont.MeAvatarResolver != nil &&
		cont.AnnouncementUC != nil &&
		cont.AnnouncementQ != nil {

		announcementH = mallhandler.NewMeAnnouncementHandler(
			cont.MeAvatarResolver,
			cont.AnnouncementUC,
			cont.AnnouncementQ,
		)
	}

	// /mall/market/resales
	if cont.MarketQ != nil {
		marketH = mallhandler.NewMarketHandler(
			mallhandler.NewMarketHandlerParams{
				MarketQ: cont.MarketQ,
			},
		)
	}

	// /mall/me/resales
	if cont.ResaleUC != nil && cont.ResaleQ != nil {
		resaleH = mallhandler.NewResaleHandler(
			mallhandler.NewResaleHandlerParams{
				UC:    cont.ResaleUC,
				Query: cont.ResaleQ,
			},
		)
	}

	// ------------------------------------------------------------
	// setup-status wiring
	// ------------------------------------------------------------
	if cont.SetupUC != nil {
		setupStatusH = mallhandler.NewSetupStatusHandler(cont.SetupUC)
	}

	// Cart
	if cont.CartUC != nil {
		cartH = mallhandler.NewCartHandler(cont.CartUC, cont.CartQ)
	}

	// Payment
	if cont.PaymentUC != nil {
		payH = mallhandler.NewPaymentHandler(
			cont.PaymentUC,
			cont.OrderQ,
			cont.PaymentFlowUC,
		)
	}

	// Order
	if cont.OrderUC != nil {
		orderH = mallhandler.NewOrderHandler(
			cont.OrderUC,
			cont.HistoryQ,
		)
	}

	// Inquiry
	if cont.InquiryUC != nil && cont.InquiryQ != nil {
		inquiryH = mallhandler.NewInquiryHandler(
			cont.InquiryUC,
			cont.InquiryQ,
		)
	}

	// Preview
	if cont.PreviewQ != nil {
		opts := []mallhandler.PreviewHandlerOption{}

		if cont.OwnerResolveQ != nil {
			opts = append(
				opts,
				mallhandler.WithOwnerResolveQuery(cont.OwnerResolveQ),
			)
		}

		if cont.NameResolver != nil {
			opts = append(
				opts,
				mallhandler.WithNameResolver(cont.NameResolver),
			)
		}

		previewPublicH = mallhandler.NewPreviewHandler(
			cont.PreviewQ,
			opts...,
		)

		previewMeH = mallhandler.NewPreviewMeHandler(
			cont.PreviewQ,
			cont.OwnerResolveQ,
			nil,
			cont.NameResolver,
		)
	}

	// Order scan verify
	if cont.PreviewQ != nil {
		orderScanVerifyH = mallhandler.NewOrderScanVerifyHandler(
			cont.PreviewQ,
		)
	}

	// Order scan transfer
	if cont.TransferUC != nil {
		orderScanTransferH = mallhandler.NewTransferHandler(
			cont.TransferUC,
		)
	}

	// SignIn: keep a stable no-op endpoint
	signInH := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.WriteHeader(http.StatusNoContent)
	})

	// ----------------------------
	// Router deps
	// ----------------------------
	deps := mallhttp.Deps{
		List: listH,

		Inventory:        invH,
		ProductBlueprint: pbH,
		Catalog:          catalogH,
		TokenBlueprint:   tbH,

		ProductBlueprintReview: pbReviewH,
		TokenBlueprintReview:   tbReviewH,

		Company: companyH,
		Brand:   brandH,

		SignIn: signInH,

		Auth: authH,

		User:            userH,
		ShippingAddress: shipH,
		PaymentMethod:   paymentMethodH,
		Avatar:          avatarH,

		MeAvatar: meAvatarsH,
		Wallet:   walletH,
		MeWallet: meWalletH,
		Cart:     cartH,

		Market: marketH,
		Resale: resaleH,

		Preview:   previewPublicH,
		PreviewMe: previewMeH,

		OrderScanVerify:   orderScanVerifyH,
		OrderScanTransfer: orderScanTransferH,
		OwnerResolve: notImplemented(
			"OwnerResolve(endpoint_disabled)",
		),
		Payment:      payH,
		Order:        orderH,
		Inquiry:      inquiryH,
		Announcement: announcementH,

		SetupStatus: setupStatusH,
	}

	mallhttp.Register(
		mux,
		deps,
		userAuthMW.Handler,
		avatarCtxMW.Handler,
	)

	// ----------------------------
	// Webhooks (no auth)
	// ----------------------------
	if cont.PaymentUC != nil {
		secret := os.Getenv("STRIPE_WEBHOOK_SECRET")
		if secret == "" {
			return
		}

		stripeWH := mallwebhook.NewStripeWebhookHandler(
			cont.PaymentUC,
			secret,
		)
		mux.Handle(StripeWebhookPath, stripeWH)
		mux.Handle(StripeWebhookPath+"/", stripeWH)
	}
}

func transferUsecaseNotConfiguredHandler() http.Handler {
	return http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write(
			[]byte(`{"error":"transfer_usecase_not_configured"}`),
		)
	})
}
