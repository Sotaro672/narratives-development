// backend/internal/platform/di/console/router_di.go
package console

import (
	"encoding/json"
	"net/http"
	"strings"

	httpin "narratives/internal/adapters/in/http/console"

	// handlers
	consoleHandler "narratives/internal/adapters/in/http/console/handler"
	inspectionHandler "narratives/internal/adapters/in/http/console/handler/inspection"
	inventoryHandler "narratives/internal/adapters/in/http/console/handler/inventory"
	listHandler "narratives/internal/adapters/in/http/console/handler/list"
	modelHandler "narratives/internal/adapters/in/http/console/handler/model"
	productBlueprintHandler "narratives/internal/adapters/in/http/console/handler/productBlueprint"
	productionHandler "narratives/internal/adapters/in/http/console/handler/production"

	// middlewares
	"narratives/internal/adapters/in/http/middleware"

	// queries
	sharedquery "narratives/internal/application/query/shared"
)

// BuildConsoleRouterDeps は “新しい httpin.RouterDeps（handler束）” を組み立てて返す。
// ※注意：このファイルでは Container.RouterDeps を定義しない（container_router.go と衝突するため）。
func BuildConsoleRouterDeps(c *Container) httpin.RouterDeps {
	// =========================================
	// ListImage uploader fallback wiring
	// =========================================
	uploader := c.ListImageUploader
	if c.ListUC != nil && uploader == nil {
		if up, ok := any(c.ListUC).(listHandler.ListImageUploader); ok {
			uploader = up
		}
	}
	// deleter は常に nil（DELETE API 廃止）
	// handler 側は imgDeleter == nil の場合 501 を返す想定

	// =========================================
	// Middlewares (built in DI)
	// =========================================
	var authMw *middleware.AuthMiddleware
	if c.Infra.FirebaseAuth != nil && c.MemberRepo != nil {
		authMw = &middleware.AuthMiddleware{
			FirebaseAuth: c.Infra.FirebaseAuth,
			MemberRepo:   c.MemberRepo,
		}
	}

	var bootstrapMw *middleware.BootstrapAuthMiddleware
	if c.Infra.FirebaseAuth != nil {
		bootstrapMw = &middleware.BootstrapAuthMiddleware{
			FirebaseAuth: c.Infra.FirebaseAuth,
		}
	}

	// =========================================
	// Handlers (built in DI)
	// =========================================
	var (
		authBootstrapH http.Handler

		accountsH      http.Handler
		announcementsH http.Handler
		permissionsH   http.Handler
		brandsH        http.Handler
		companiesH     http.Handler
		inquiriesH     http.Handler
		inventoriesH   http.Handler
		listsH         http.Handler

		productsPrintH http.Handler
		productBPH     http.Handler
		tokenBPH       http.Handler

		messagesH    http.Handler
		ordersH      http.Handler
		walletsH     http.Handler
		membersH     http.Handler
		productionsH http.Handler
		modelsH      http.Handler

		inspectorH http.Handler
		mintH      http.Handler

		ownerResolveH http.Handler

		mintDebug http.HandlerFunc
	)

	// /auth/bootstrap
	if c.AuthBootstrap != nil && bootstrapMw != nil {
		authBootstrapH = consoleHandler.NewAuthBootstrapHandler(c.AuthBootstrap)
	}

	// Accounts
	if c.AccountUC != nil {
		accountsH = consoleHandler.NewAccountHandler(c.AccountUC)
	}

	// Announcements
	if c.AnnouncementUC != nil {
		announcementsH = consoleHandler.NewAnnouncementHandler(c.AnnouncementUC)
	}

	// Permissions
	if c.PermissionUC != nil {
		permissionsH = consoleHandler.NewPermissionHandler(c.PermissionUC)
	}

	// Brands
	if c.BrandUC != nil {
		brandsH = consoleHandler.NewBrandHandler(c.BrandUC)
	}

	// Companies
	if c.CompanyUC != nil {
		companiesH = consoleHandler.NewCompanyHandler(c.CompanyUC)
	}

	// Inquiries
	if c.InquiryUC != nil {
		inquiriesH = consoleHandler.NewInquiryHandler(c.InquiryUC)
	}

	// Inventories
	if c.InventoryUC != nil {
		inventoriesH = inventoryHandler.NewInventoryHandlerWithListCreateQuery(
			c.InventoryUC,
			c.InventoryQuery,
			c.ListCreateQuery,
		)
	}

	// Lists
	if c.ListUC != nil {
		listsH = listHandler.NewListHandlerWithQueriesAndListImage(
			c.ListUC,
			c.ListManagementQuery,
			c.ListDetailQuery,
			uploader,
			nil,
		)
	}

	// Products（印刷系）
	if c.PrintUC != nil {
		productsPrintH = consoleHandler.NewPrintHandler(
			c.PrintUC,
			c.ProductionUC,
			c.ModelUC,
			c.NameResolver,
		)
	}

	// Product Blueprints
	if c.ProductBlueprintUC != nil {
		productBPH = productBlueprintHandler.NewProductBlueprintHandler(
			c.ProductBlueprintUC,
			c.BrandService,
			c.MemberService,
		)
	}

	// Token Blueprints
	if c.TokenBlueprintUC != nil {
		tokenBPH = consoleHandler.NewTokenBlueprintHandler(
			c.TokenBlueprintUC,
			c.TokenBlueprintQueryUC,
			c.BrandService,
		)
	}

	// Messages
	if c.MessageUC != nil && c.MessageRepo != nil {
		messagesH = consoleHandler.NewMessageHandler(c.MessageUC, c.MessageRepo)
	}

	// Orders
	if c.OrderUC != nil {
		ordersH = consoleHandler.NewOrderHandler(c.OrderUC)
	}

	// Wallets
	if c.WalletUC != nil {
		walletsH = consoleHandler.NewWalletHandler(c.WalletUC)
	}

	// Members
	if c.MemberUC != nil && c.MemberRepo != nil {
		membersH = consoleHandler.NewMemberHandler(c.MemberUC, c.MemberRepo)
	}

	// Productions
	if c.ProductionUC != nil && c.CompanyProductionQueryService != nil {
		productionsH = productionHandler.NewProductionHandler(
			c.CompanyProductionQueryService,
			c.ProductionUC,
		)
	}

	// Models
	if c.ModelUC != nil {
		modelsH = modelHandler.NewModelHandler(c.ModelUC)
	}

	// Inspector
	if c.ProductUC != nil && c.InspectionUC != nil {
		var pbGetter inspectionHandler.ProductBlueprintModelRefGetter
		if c.ProductBlueprintUC != nil {
			if g, ok := any(c.ProductBlueprintUC).(inspectionHandler.ProductBlueprintModelRefGetter); ok {
				pbGetter = g
			}
		}

		inspectorH = inspectionHandler.NewInspectorHandler(
			c.ProductUC,
			c.InspectionUC,
			c.NameResolver,
			pbGetter,
		)
	}

	// Mint
	if c.MintUC != nil {
		mintHandler := consoleHandler.NewMintHandler(
			c.MintUC,
			c.NameResolver,
			c.ProductionUC,
			c.MintRequestQueryService,
		)
		mintH = mintHandler

		// /mint/debug（任意）
		if mh, ok := mintHandler.(*consoleHandler.MintHandler); ok {
			mintDebug = mh.HandleDebug
		}
	}

	// Owner resolve
	if c.OwnerResolveQ != nil {
		// 同一パッケージ内の既存 ownerResolveHandler と衝突しないように別名にする
		ownerResolveH = &consoleOwnerResolveHandler{q: c.OwnerResolveQ}
	}

	return httpin.RouterDeps{
		AuthMw:      authMw,
		BootstrapMw: bootstrapMw,

		AuthBootstrap: authBootstrapH,

		Accounts:      accountsH,
		Announcements: announcementsH,
		Permissions:   permissionsH,
		Brands:        brandsH,
		Companies:     companiesH,
		Inquiries:     inquiriesH,
		Inventories:   inventoriesH,
		Lists:         listsH,

		ProductsPrint: productsPrintH,
		ProductBP:     productBPH,
		TokenBP:       tokenBPH,

		Messages: messagesH,
		Orders:   ordersH,
		Wallets:  walletsH,
		Members:  membersH,

		Productions: productionsH,
		Models:      modelsH,

		Inspector: inspectorH,
		Mint:      mintH,

		OwnerResolve: ownerResolveH,

		// optional
		MintDebugHandle: mintDebug,
	}
}

// consoleOwnerResolveHandler は owner resolve の小さなHTTPハンドラ実装。
// ※container_router.go の ownerResolveHandler と型名が衝突しないよう別名にしている。
type consoleOwnerResolveHandler struct {
	q *sharedquery.OwnerResolveQuery
}

func (h *consoleOwnerResolveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	addr := strings.TrimSpace(q.Get("walletAddress"))
	if addr == "" {
		addr = strings.TrimSpace(q.Get("toAddress"))
	}
	if addr == "" {
		addr = strings.TrimSpace(q.Get("address"))
	}
	if addr == "" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": "walletAddress (or toAddress/address) is required",
		})
		return
	}

	res, err := h.q.Resolve(r.Context(), addr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		switch err {
		case sharedquery.ErrInvalidWalletAddress:
			w.WriteHeader(http.StatusBadRequest)
		case sharedquery.ErrOwnerNotFound:
			w.WriteHeader(http.StatusNotFound)
		case sharedquery.ErrOwnerResolveNotConfigured:
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": res,
	})
}
