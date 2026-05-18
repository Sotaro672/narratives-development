// backend/internal/platform/di/mall/register.go
package mall

import (
	"log"
	"net/http"
	"os"

	mallhttp "narratives/internal/adapters/in/http/mall"
	mallhandler "narratives/internal/adapters/in/http/mall/handler"
	avatarHandler "narratives/internal/adapters/in/http/mall/handler/avatar"
	mallwebhook "narratives/internal/adapters/in/http/mall/webhook"
	"narratives/internal/adapters/in/http/middleware"
	mallquery "narratives/internal/application/query/mall"
	"narratives/internal/application/usecase"
	tokenBlueprint "narratives/internal/domain/tokenBlueprint"

	// setup-status repo (Firestore)
	firestoreMall "narratives/internal/adapters/out/firestore/mall"

	// resolvedTokens repo (Firestore)
	firestoreOut "narratives/internal/adapters/out/firestore"
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
		log.Printf("[mall.register] WARN: cont.Infra or cont.Infra.FirebaseAuth is nil (user auth will return 503 on protected endpoints)")
		userAuthMW = &middleware.UserAuthMiddleware{FirebaseAuth: nil}
	}

	// ------------------------------------------------------------
	// Avatar context middleware (uid -> avatarId (+walletAddress))
	// ------------------------------------------------------------
	var avatarCtxMW *middleware.AvatarContextMiddleware
	{
		if cont.MeAvatarRepo != nil {
			avatarCtxMW = &middleware.AvatarContextMiddleware{
				Resolver: cont.MeAvatarRepo,
			}
		} else {
			log.Printf("[mall.register] WARN: cont.MeAvatarRepo is nil (avatar context will return 503 on endpoints requiring avatarId)")
			avatarCtxMW = &middleware.AvatarContextMiddleware{Resolver: nil}
		}
	}

	// ------------------------------------------------------------
	// Local repositories recreated from Firestore when needed
	// ------------------------------------------------------------
	var tokenBlueprintRepo tokenBlueprint.RepositoryPort
	var tokenBlueprintPatchRepo mallhandler.TokenBlueprintPatchReader
	{
		hasFS := cont.Infra != nil && cont.Infra.Firestore != nil
		if hasFS {
			repo := firestoreOut.NewTokenBlueprintRepositoryFS(cont.Infra.Firestore)

			tokenBlueprintRepo = repo
			tokenBlueprintPatchRepo = repo

			log.Printf("[mall.register] tokenBlueprint repo wired (Firestore)")
		} else {
			log.Printf("[mall.register] WARN: Firestore client is nil (tokenBlueprint repo unavailable)")
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

	pbReviewH := notImplemented("ProductBlueprintReview")
	tbReviewH := notImplemented("TokenBlueprintReview")

	companyH := notImplemented("Company")
	brandH := notImplemented("Brand")

	userH := notImplemented("User")
	shipH := notImplemented("ShippingAddress")
	paymentMethodH := notImplemented("PaymentMethod")
	avatarH := notImplemented("Avatar")
	avatarStateH := notImplemented("AvatarState")
	walletH := notImplemented("Wallet")
	meWalletH := notImplemented("MeWallet")
	cartH := notImplemented("Cart")
	payH := notImplemented("Payment")
	orderH := notImplemented("Order")
	meAvatarsH := notImplemented("MeAvatars")

	previewPublicH := notImplemented("PreviewPublic")
	previewMeH := notImplemented("PreviewMe")

	orderScanVerifyH := notImplemented("OrderScanVerify")
	orderScanTransferH := notImplemented("OrderScanTransfer")
	shareTransferH := notImplemented("ShareTransfer")

	setupStatusH := notImplemented("SetupStatus")

	// Lists (public)
	if cont.ListUC != nil {
		listH = mallhandler.NewMallListHandler(cont.ListUC)
	}

	// Catalog (public)
	if cont.CatalogQ != nil {
		catalogH = mallhandler.NewMallCatalogHandler(cont.CatalogQ)
	}

	// ProductBlueprintReview wiring (catalog composite)
	if cont.ProductBlueprintReviewUC != nil {
		pbReviewH = mallhandler.NewProductBlueprintReviewHandler(cont.ProductBlueprintReviewUC)
		catalogH = newCatalogCompositeHandler(catalogH, pbReviewH)
		log.Printf("[mall.register] productBlueprint review handler wired (catalog composite enabled)")
	} else {
		log.Printf("[mall.register] WARN: ProductBlueprintReviewUC is nil (productBlueprint review will return 501)")
	}

	// Inventory (public read-only)
	if cont.InventoryUC != nil {
		invH = mallhandler.NewMallInventoryHandler(cont.InventoryUC)
	}

	// TokenBlueprint (public patch)
	//
	// Firebase Storage migration policy:
	// - backend は GCS objectPath -> public URL resolver を使わない
	// - tokenBlueprint iconUrl / Patch.IconURL には Firebase Storage downloadURL が入る
	// - handler には NameResolver のみ渡す
	if tokenBlueprintRepo != nil {
		if cont.NameResolver != nil {
			tbH = mallhandler.NewMallTokenBlueprintHandlerWithNameResolver(
				tokenBlueprintRepo,
				cont.NameResolver,
			)
		} else {
			tbH = mallhandler.NewMallTokenBlueprintHandler(tokenBlueprintRepo)
		}
	}

	// Brand（/mall/brands/{id}）
	if cont.BrandQ != nil {
		brandH = mallhandler.NewMallBrandHandler(cont.BrandQ)
		log.Printf("[mall.register] brand handler wired")
	} else {
		log.Printf("[mall.register] WARN: BrandQ is nil (brand endpoint will return 501)")
	}

	// Avatar（/mall/avatars）
	if cont.AvatarUC != nil {
		avatarH = avatarHandler.NewAvatarHandler(cont.AvatarUC, cont.AvatarRepo)
	}

	// TokenBlueprintReview wiring
	//
	// Hexagonal architecture:
	// - handler は HTTP adapter
	// - usecase は application service
	// - repository / avatar / tokenBlueprintRepo / brand service は usecase に注入する
	if cont.TokenBlueprintReviewRepo != nil {
		if tokenBlueprintRepo == nil {
			log.Printf("[mall.register] WARN: tokenBlueprint repo is nil (tokenBlueprint review usecase will have nil tokenBlueprintRepo)")
		}

		tbReviewUC := usecase.NewTokenBlueprintReviewUsecase(
			cont.TokenBlueprintReviewRepo,
			cont.AvatarRepo,
			tokenBlueprintRepo,
			cont.BrandService,
		)

		tbReviewH = mallhandler.NewTokenBlueprintReviewHandler(tbReviewUC)

		log.Printf(
			"[mall.register] tokenBlueprint review handler wired via usecase (avatarRepo=%t tokenBlueprintRepo=%t brandService=%t)",
			cont.AvatarRepo != nil,
			tokenBlueprintRepo != nil,
			cont.BrandService != nil,
		)
	} else {
		log.Printf("[mall.register] WARN: TokenBlueprintReviewRepo is nil (tokenBlueprint review will return 501)")
	}

	// 方法A: TokenBlueprint handler 1つだけ登録し、内部で reviews へ振り分ける
	if tbReviewH != nil && tbReviewH != http.NotFoundHandler() {
		tbH = mallhandler.NewTokenBlueprintCompositeHandler(tbH, tbReviewH)
		log.Printf("[mall.register] tokenBlueprint composite handler enabled (tb + reviews)")
	} else {
		log.Printf("[mall.register] tokenBlueprint composite handler NOT enabled (reviews handler is nil)")
	}

	// Core resources
	if cont.UserUC != nil {
		userH = mallhandler.NewUserHandler(cont.UserUC)
	}
	if cont.ShippingAddressUC != nil {
		shipH = mallhandler.NewShippingAddressHandler(cont.ShippingAddressUC)
	}
	if cont.PaymentMethodUC != nil {
		paymentMethodH = mallhandler.NewPaymentMethodHandler(cont.PaymentMethodUC)
	}

	var resolvedRepo mallhandler.ResolvedTokenRepository
	{
		hasFS := cont.Infra != nil && cont.Infra.Firestore != nil
		if hasFS {
			resolvedRepo = firestoreOut.NewResolvedTokenRepositoryFS(cont.Infra.Firestore)
			log.Printf("[mall.register] resolvedTokens repo wired (Firestore)")
		} else {
			log.Printf("[mall.register] WARN: Firestore client is nil (resolvedTokens cache disabled)")
		}
	}

	// Wallet (public)
	if cont.WalletUC != nil {
		walletH = mallhandler.NewWalletHandler(cont.WalletUC, resolvedRepo)
		log.Printf("[mall.register] public wallet handler wired")
	}

	// Wallet (me)
	if cont.WalletUC != nil {
		meWalletH = mallhandler.NewMallMeWalletHandler(cont.WalletUC, resolvedRepo)
		log.Printf("[mall.register] me wallet handler wired")
	}

	// /mall/me/avatars
	if cont.MeAvatarRepo != nil &&
		cont.AvatarUC != nil &&
		cont.AvatarRepo != nil &&
		cont.Infra != nil &&
		cont.Infra.Firestore != nil {

		avatarStateRepo := firestoreOut.NewAvatarStateRepositoryFS(cont.Infra.Firestore)
		avatarStateQuery := mallquery.NewAvatarStateQuery(cont.AvatarRepo, avatarStateRepo)

		meAvatarsH = mallhandler.NewMeAvatarHandler(cont.MeAvatarRepo, cont.AvatarUC, avatarStateQuery)
		log.Printf("[mall.register] me avatars handler wired (repo+avatarUC+avatarStateQuery)")
	} else {
		log.Printf(
			"[mall.register] WARN: MeAvatars not wired (meAvatarRepo=%t avatarUC=%t avatarRepo=%t firestore=%t) (MeAvatars will return 501)",
			cont.MeAvatarRepo != nil,
			cont.AvatarUC != nil,
			cont.AvatarRepo != nil,
			cont.Infra != nil && cont.Infra.Firestore != nil,
		)
	}

	// ------------------------------------------------------------
	// setup-status wiring (Firestore)
	// ------------------------------------------------------------
	{
		hasFS := cont.Infra != nil && cont.Infra.Firestore != nil
		log.Printf("[mall.register] setup-status wiring start firestore=%t", hasFS)

		if hasFS {
			repo := firestoreMall.NewSetupStatusRepoFirestore(cont.Infra.Firestore)
			setupStatusH = mallhandler.NewSetupStatusHandler(repo)
			log.Printf("[mall.register] setup-status handler wired (Firestore)")
		} else {
			log.Printf("[mall.register] WARN: Firestore client is nil (setup-status will return 501)")
		}
	}

	// Cart
	if cont.CartUC != nil {
		cartH = mallhandler.NewCartHandlerWithQueries(cont.CartUC, cont.CartQ)
	} else if cont.CartQ != nil {
		cartH = mallhandler.NewCartQueryHandler(cont.CartQ)
	}

	// Payment / Order
	if cont.PaymentUC != nil {
		if cont.PaymentFlowUC != nil {
			payH = mallhandler.NewPaymentHandlerWithOrderQueryAndPaymentFlow(
				cont.PaymentUC,
				cont.OrderQ,
				cont.PaymentFlowUC,
			)
			log.Printf("[mall.register] payment handler wired with PaymentFlowUC")
		} else {
			payH = mallhandler.NewPaymentHandlerWithOrderQuery(cont.PaymentUC, cont.OrderQ)
			log.Printf("[mall.register] WARN: PaymentFlowUC is nil; payment handler uses legacy PaymentUsecase.Create path")
		}
	}

	if cont.OrderUC != nil {
		if cont.HistoryQ != nil {
			orderH = mallhandler.NewOrderHandlerWithHistoryQuery(cont.OrderUC, cont.HistoryQ)
			log.Printf("[mall.register] order handler wired with HistoryQ")
		} else {
			orderH = mallhandler.NewOrderHandler(cont.OrderUC)
			log.Printf("[mall.register] WARN: HistoryQ is nil; order handler uses plain OrderUsecase.List response")
		}
	}

	// Preview
	if cont.PreviewQ != nil {
		opts := []mallhandler.PreviewHandlerOption{}
		if cont.OwnerResolveQ != nil {
			opts = append(opts, mallhandler.WithOwnerResolveQuery(cont.OwnerResolveQ))
		}
		if tokenBlueprintPatchRepo != nil {
			opts = append(opts, mallhandler.WithTokenBlueprintPatchRepo(tokenBlueprintPatchRepo))
		}
		if cont.NameResolver != nil {
			opts = append(opts, mallhandler.WithNameResolver(cont.NameResolver))
		}

		previewPublicH = mallhandler.NewPreviewHandler(cont.PreviewQ, opts...)
		previewMeH = mallhandler.NewPreviewMeHandler(cont.PreviewQ, opts...)

		log.Printf(
			"[mall.register] preview handlers wired (public+me) ownerQ=%t tbRepo=%t nameResolver=%t",
			cont.OwnerResolveQ != nil,
			tokenBlueprintPatchRepo != nil,
			cont.NameResolver != nil,
		)
	}

	// Order scan verify
	if cont.OrderScanVerifyQ != nil {
		orderScanVerifyH = mallhandler.NewOrderScanVerifyHandler(cont.OrderScanVerifyQ)
		log.Printf("[mall.register] order scan verify handler wired")
	} else {
		log.Printf("[mall.register] WARN: OrderScanVerifyQ is nil (order scan verify will return 501)")
	}

	// Order scan transfer
	if cont.TransferUC != nil {
		orderScanTransferH = mallhandler.NewTransferHandler(cont.TransferUC)
		log.Printf("[mall.register] order scan transfer handler wired (TransferUC=%t)", cont.TransferUC != nil)
	} else {
		log.Printf("[mall.register] WARN: TransferUC is nil (order scan transfer will return 501)")
	}

	// Share transfer
	if cont.ShareTransferUC != nil {
		shareTransferH = mallhandler.NewShareTransferHandler(cont.ShareTransferUC)
		log.Printf("[mall.register] share transfer handler wired (ShareTransferUC=%t)", cont.ShareTransferUC != nil)
	} else {
		log.Printf("[mall.register] WARN: ShareTransferUC is nil (share transfer will return 501)")
	}

	// SignIn: keep a stable no-op endpoint
	signInH := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

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

		ProductBlueprintReview: pbReviewH,
		TokenBlueprintReview:   tbReviewH,

		Company: companyH,
		Brand:   brandH,

		SignIn: signInH,

		User:            userH,
		ShippingAddress: shipH,
		PaymentMethod:   paymentMethodH,
		Avatar:          avatarH,

		MeAvatar: meAvatarsH,

		AvatarState: avatarStateH,
		Wallet:      walletH,
		MeWallet:    meWalletH,
		Cart:        cartH,

		Preview:   previewPublicH,
		PreviewMe: previewMeH,

		OrderScanVerify:   orderScanVerifyH,
		OrderScanTransfer: orderScanTransferH,
		ShareTransfer:     shareTransferH,
		OwnerResolve:      notImplemented("OwnerResolve(endpoint_disabled)"),
		Payment:           payH,
		Order:             orderH,

		SetupStatus: setupStatusH,
	}

	mallhttp.Register(
		mux,
		deps,
		userAuthMW.Handler,
		avatarCtxMW.Handler,
	)

	log.Printf("[boot] mall routes registered")

	// ----------------------------
	// Webhooks (no auth)
	// ----------------------------
	if cont.PaymentUC != nil {
		secret := os.Getenv("STRIPE_WEBHOOK_SECRET")
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
