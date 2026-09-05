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
	)

	router := adminhttp.NewRouter(adminhttp.RouterDeps{
		AuthMw:    authMw,
		Me:        meHandler,
		Contacts:  contactHandler,
		Companies: companyHandler,
	})

	mux.Handle("/admin/", router)
	mux.Handle("/admin", http.RedirectHandler("/admin/", http.StatusPermanentRedirect))
}
