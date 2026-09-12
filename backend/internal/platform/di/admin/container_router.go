// backend/internal/platform/di/admin/container_router.go
package admin

import (
	"net/http"

	adminhttp "narratives/internal/adapters/in/http/admin"
	adminhandler "narratives/internal/adapters/in/http/admin/handler"
	"narratives/internal/adapters/in/http/middleware"
)

// Register registers Admin routes onto mux.
func Register(mux *http.ServeMux, cont *Container) {
	if mux == nil || cont == nil {
		return
	}

	var authMw *middleware.AdminAuthMiddleware
	if cont.Infra != nil && cont.Infra.FirebaseAuth != nil {
		authMw = &middleware.AdminAuthMiddleware{
			FirebaseAuth: cont.Infra.FirebaseAuth,
			AllowedUID:   cont.adminFirebaseUID,
			AllowedEmail: cont.adminEmail,
		}
	}

	meHandler := adminhandler.NewMeHandler()
	contactHandler := adminhandler.NewContactHandler(cont.contactUsecase)
	companyHandler := adminhandler.NewCompanyHandler(
		cont.companyRepo,
		cont.memberRepo,
		cont.contractDetailQuery,
		cont.contractAnnouncementDetailQuery,
		cont.contractListQuery,
		cont.contractTokenBlueprintQuery,
		cont.contractProductBlueprintQuery,
		cont.contractTokenBlueprintReviewQuery,
		cont.contractProductBlueprintReviewQuery,
	)

	avatarHandler := adminhandler.NewAvatarHandler(
		cont.avatarRepo,
		cont.userRepo,
		cont.reportRepo,
		cont.resaleRepo,
	)

	resaleHandler := adminhandler.NewResaleHandler(
		cont.resaleRepo,
		cont.resaleImageRepo,
		cont.reportRepo,
		cont.productBlueprintRepo,
		cont.tokenBlueprintRepo,
	)

	resaleReviewHandler := adminhandler.NewResaleReviewHandler(
		cont.resaleRepo,
		cont.resaleReviewUsecase,
		cont.reportRepo,
	)

	resaleTradeHandler := adminhandler.NewResaleTradeHandler(
		cont.resaleTradeQuery,
	)

	tradeMessageHandler := adminhandler.NewTradeMessageHandler(
		cont.tradeMessageQuery,
	)

	gasHandler := adminhandler.NewGasHandler(cont.gasBalanceQuery)
	mintHandler := adminhandler.NewMintHandler(
		cont.mintListQuery,
		cont.mintDetailQuery,
	)
	newsHandler := adminhandler.NewNewsHandler(
		cont.newsUsecase,
		cont.newsQuery,
	)
	reportHandler := adminhandler.NewReportHandler(
		cont.reportUsecase,
		cont.reportNameQuery,
	)

	router := adminhttp.NewRouter(adminhttp.RouterDeps{
		AuthMw:        authMw,
		Me:            meHandler,
		Contacts:      contactHandler,
		Companies:     companyHandler,
		Avatars:       avatarHandler,
		Resales:       resaleHandler,
		ResaleReviews: resaleReviewHandler,
		ResaleTrades:  resaleTradeHandler,
		TradeMessages: tradeMessageHandler,
		Gas:           gasHandler,
		Mints:         mintHandler,
		News:          newsHandler,
		Reports:       reportHandler,
	})

	mux.Handle("/admin/", router)
	mux.Handle(
		"/admin",
		http.RedirectHandler(
			"/admin/",
			http.StatusPermanentRedirect,
		),
	)
}
