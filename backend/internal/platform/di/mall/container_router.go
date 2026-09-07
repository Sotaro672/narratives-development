// backend/internal/platform/di/mall/container_router.go
package mall

import (
	"net/http"

	internalHandler "narratives/internal/adapters/in/http/handler"
	mallhttp "narratives/internal/adapters/in/http/mall"
	mallhandler "narratives/internal/adapters/in/http/mall/handler"
	mallwebhook "narratives/internal/adapters/in/http/mall/webhook"
	"narratives/internal/adapters/in/http/middleware"
	mailadp "narratives/internal/adapters/out/mail"
)

// Register registers mall routes onto mux.
func Register(mux *http.ServeMux, cont *Container) {
	if mux == nil || cont == nil {
		return
	}

	cfg := cont.config

	// ------------------------------------------------------------
	// Auth middleware (buyer/user side)
	// ------------------------------------------------------------
	userAuthMW := &middleware.UserAuthMiddleware{}
	if cont.Infra != nil {
		userAuthMW.FirebaseAuth = cont.Infra.FirebaseAuth
	}

	// ------------------------------------------------------------
	// Avatar context middleware (uid -> avatarId (+walletAddress))
	// ------------------------------------------------------------
	avatarCtxMW := &middleware.AvatarContextMiddleware{
		Resolver: cont.MeAvatarResolver,
	}

	// ----------------------------
	// Handlers (construct only)
	// ----------------------------
	var listH http.Handler
	var catalogH http.Handler
	var pbReviewH http.Handler
	var brandH http.Handler
	var authH http.Handler
	var userH http.Handler
	var shipH http.Handler
	var shippingQuoteH http.Handler
	var paymentMethodH http.Handler
	var payoutAccountH http.Handler
	var avatarH http.Handler
	var meWalletH http.Handler
	var likeH http.Handler
	var cartH http.Handler
	var payH http.Handler
	var orderH http.Handler
	var tradeH http.Handler
	var avatarReviewH http.Handler
	var inquiryH http.Handler
	var meAvatarsH http.Handler
	var announcementH http.Handler
	var reportDecisionNotificationH http.Handler
	var marketH http.Handler
	var resaleH http.Handler
	var previewPublicH http.Handler
	var previewMeH http.Handler
	var setupStatusH http.Handler

	// TokenBlueprint の base handler。
	// report/review のいずれにも該当しない path は 404 とする。
	var tbH http.Handler = http.NotFoundHandler()

	orderScanTransferH := transferUsecaseNotConfiguredHandler()

	// Auth email verification
	if cont.Infra != nil && cont.Infra.FirebaseAuth != nil {
		authMailer := mailadp.NewAuthMailer(
			mailadp.NewResendClient(cfg.ResendAPIKey),
			cfg.ResendFrom,
		)

		authH = mallhandler.NewAuthHandler(
			cont.Infra.FirebaseAuth,
			authMailer,
			cfg.AuthActionBaseURL,
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

	// ProductBlueprintReview
	if cont.ProductBlueprintReviewUC != nil {
		pbReviewH = mallhandler.NewProductBlueprintReviewHandler(
			cont.ProductBlueprintReviewUC,
			cont.ReportUC,
		)
	}

	// Brand
	if cont.BrandQ != nil {
		brandH = mallhandler.NewMallBrandHandler(cont.BrandQ)
	}

	// Avatar
	if cont.AvatarUC != nil && cont.AvatarRegistrationUC != nil {
		avatarH = mallhandler.NewAvatarHandler(
			cont.AvatarUC,
			cont.AvatarRegistrationUC,
		)
	}

	// TokenBlueprint report
	// POST /mall/me/token-blueprints/{tokenBlueprintId}/reports
	if cont.ReportUC != nil {
		tbH = mallhandler.NewTokenBlueprintReportHandler(cont.ReportUC)
	}

	// TokenBlueprintReview
	if cont.TokenBlueprintReviewUC != nil {
		tbReviewH := mallhandler.NewTokenBlueprintReviewHandler(
			cont.TokenBlueprintReviewUC,
			cont.ReportUC,
		)
		tbH = mallhandler.NewTokenBlueprintCompositeHandler(tbH, tbReviewH)
	}

	// Core resources
	if cont.UserUC != nil {
		userH = mallhandler.NewUserHandler(cont.UserUC)
	}

	if cont.ShippingAddressUC != nil {
		shipH = mallhandler.NewShippingAddressHandler(cont.ShippingAddressUC)
	}

	if cont.ShippingQuoteUC != nil {
		shippingQuoteH = mallhandler.NewShippingQuoteHandler(cont.ShippingQuoteUC)
	}

	if cont.PaymentMethodUC != nil && cont.Infra != nil {
		paymentMethodH = mallhandler.NewPaymentMethodHandler(
			cont.PaymentMethodUC,
			cont.Infra,
		)
	}

	if cont.PayoutAccountUC != nil {
		payoutAccountH = mallhandler.NewPayoutAccountHandler(cont.PayoutAccountUC)
	}

	// Wallet (me)
	if cont.WalletUC != nil {
		meWalletH = mallhandler.NewMallMeWalletHandler(cont.WalletUC)
	}

	// Likes (me)
	if cont.LikeUC != nil {
		likeH = mallhandler.NewLikeHandler(cont.LikeUC)
	}

	// /mall/me/avatars
	if cont.MeAvatarResolver != nil &&
		cont.AvatarUC != nil &&
		cont.ReportUC != nil {

		meAvatarsH = mallhandler.NewMeAvatarHandler(
			cont.MeAvatarResolver,
			cont.AvatarUC,
			cont.ReportUC,
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

	// /mall/me/report-decision-notifications
	if cont.ReportUC != nil {
		reportDecisionNotificationH =
			mallhandler.NewReportDecisionNotificationHandler(
				cont.ReportUC,
			)
	}

	// /mall/market/resales
	if cont.MarketQ != nil {
		marketH = mallhandler.NewMarketHandler(
			mallhandler.NewMarketHandlerParams{
				MarketQ:        cont.MarketQ,
				ResaleReviewUC: cont.ResaleReviewUC,
			},
		)
	}

	// /mall/me/resales
	if cont.ResaleUC != nil && cont.ResaleQ != nil {
		resaleH = mallhandler.NewResaleHandler(
			mallhandler.NewResaleHandlerParams{
				UC:             cont.ResaleUC,
				Query:          cont.ResaleQ,
				ResaleReviewUC: cont.ResaleReviewUC,
			},
		)
	}

	// setup-status
	if cont.SetupUC != nil {
		setupStatusH = mallhandler.NewSetupStatusHandler(cont.SetupUC)
	}

	// Cart
	if cont.CartUC != nil {
		cartH = mallhandler.NewCartHandler(
			cont.CartUC,
			cont.CartQ,
		)
	}

	// Payment
	if cont.PaymentUC != nil {
		payH = mallhandler.NewPaymentHandler(
			cont.OrderQ,
			cont.PaymentFlowUC,
		)
	}

	// Order
	if cont.OrderUC != nil {
		orderH = mallhandler.NewOrderHandler(
			cont.OrderUC,
			cont.ReturnRequestUC,
			cont.HistoryQ,
			cont.OrderDetailQ,
		)
	}

	// Trade
	if cont.TradeQ != nil &&
		cont.TradeMessageUC != nil &&
		cont.ResaleTradeDispatchUC != nil &&
		cont.ResaleTradeReturnReceiptUC != nil {

		tradeH = mallhandler.NewTradeHandler(
			cont.TradeQ,
			cont.TradeMessageUC,
			cont.ResaleTradeDispatchUC,
			cont.ResaleTradeReturnReceiptUC,
		)
	}

	// Avatar review
	if cont.AvatarReviewUC != nil {
		avatarReviewH = mallhandler.NewAvatarReviewHandler(cont.AvatarReviewUC)
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

	// Order scan transfer
	if cont.TransferUC != nil {
		orderScanTransferH = mallhandler.NewTransferHandler(cont.TransferUC)
	}

	// SignIn: keep a stable no-op endpoint
	signInH := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	// ----------------------------
	// Router deps
	// ----------------------------
	deps := mallhttp.Deps{
		List:                   listH,
		Catalog:                catalogH,
		TokenBlueprint:         tbH,
		ProductBlueprintReview: pbReviewH,
		Brand:                  brandH,
		SignIn:                 signInH,
		Auth:                   authH,

		User:            userH,
		ShippingAddress: shipH,
		ShippingQuote:   shippingQuoteH,
		PaymentMethod:   paymentMethodH,
		PayoutAccount:   payoutAccountH,
		Avatar:          avatarH,

		MeAvatar: meAvatarsH,
		MeWallet: meWalletH,
		Like:     likeH,
		Cart:     cartH,

		Market: marketH,
		Resale: resaleH,

		Preview:   previewPublicH,
		PreviewMe: previewMeH,

		OrderScanTransfer: orderScanTransferH,

		Payment:                    payH,
		Order:                      orderH,
		Trade:                      tradeH,
		AvatarReview:               avatarReviewH,
		Inquiry:                    inquiryH,
		Announcement:               announcementH,
		ReportDecisionNotification: reportDecisionNotificationH,

		SetupStatus: setupStatusH,
	}

	mallhttp.Register(
		mux,
		deps,
		userAuthMW.Handler,
		avatarCtxMW.Handler,
	)

	// ----------------------------
	// Internal financial tasks
	// ----------------------------
	if cont.BrandFeeSettlementTransferUC != nil &&
		cont.BrandFeeSettlementQueue != nil {

		brandFeeSettlementTaskHandler :=
			internalHandler.NewBrandFeeSettlementTaskHandler(
				cont.BrandFeeSettlementTransferUC,
				cont.BrandFeeSettlementQueue,
			)

		mux.Handle(
			"/internal/brand-fee-settlements/process",
			http.HandlerFunc(brandFeeSettlementTaskHandler.Process),
		)

		mux.Handle(
			"/internal/brand-fee-settlements/dispatch-due",
			http.HandlerFunc(brandFeeSettlementTaskHandler.DispatchDue),
		)
	}

	if cont.ResalePayoutNotificationUC != nil {
		resalePayoutNotificationHandler :=
			internalHandler.NewResalePayoutNotificationHandler(
				cont.ResalePayoutNotificationUC,
			)

		mux.Handle(
			"/internal/resale-payout-notifications/process",
			http.HandlerFunc(resalePayoutNotificationHandler.Process),
		)

		mux.Handle(
			"/internal/resale-payout-notifications/dispatch-due",
			http.HandlerFunc(resalePayoutNotificationHandler.DispatchDue),
		)
	}

	// ----------------------------
	// Webhooks (no auth)
	// ----------------------------
	if cont.PaymentUC != nil {
		secret := cfg.StripeWebhookSecret
		if secret == "" {
			return
		}

		stripeWH := mallwebhook.NewStripeWebhookHandler(
			cont.PaymentUC,
			cont.OrderUC,
			cont.SettlementUC,
			cont.RefundUC,
			cont.ItemRefundUC,
			cont.RefundRepo,
			cont.RefundCompletionNotificationUC,
			secret,
		)

		mux.Handle(StripeWebhookPath, stripeWH)
		mux.Handle(StripeWebhookPath+"/", stripeWH)
	}
}

func transferUsecaseNotConfiguredHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"transfer_usecase_not_configured"}`))
	})
}
