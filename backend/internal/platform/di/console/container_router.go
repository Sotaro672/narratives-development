// backend/internal/platform/di/console/container_router.go
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

func (c *Container) RouterDeps() httpin.RouterDeps {
	// ✅ Fallback wiring:
	// If container fields are nil, but ListUC implements the handler ports,
	// use ListUC directly so endpoints don't become 501 by DI omission.
	uploader := c.ListImageUploader

	// DELETE API 廃止のため、deleter は常に nil（RouterDeps にも渡さない）
	// （handler 側は imgDeleter == nil の場合 501 を返す想定）
	if c.ListUC != nil {
		if uploader == nil {
			if up, ok := any(c.ListUC).(listHandler.ListImageUploader); ok {
				uploader = up
			}
		}
	}

	// ================================
	// Middlewares（生成はDI側）
	// ================================
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

	// ================================
	// Handlers（生成はDI側）
	// ================================
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

	// Lists（画像アップロードは uploader が nil でもOK / deleterは常にnil）
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
		// ✅ ProductBlueprintUC が displayOrder 解決用インターフェースを満たすなら渡す（満たさない場合は nil）
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

		// /mint/debug（任意。露出したくないなら mintDebug = nil のままでOK）
		if mh, ok := mintHandler.(*consoleHandler.MintHandler); ok {
			mintDebug = mh.HandleDebug
		}
	}

	// Owner resolve
	if c.OwnerResolveQ != nil {
		ownerResolveH = &ownerResolveHandler{q: c.OwnerResolveQ}
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

// ownerResolveHandler は router.go から分離した「小さなHTTPハンドラ実装」。
// （入力正規化・エラー→HTTP変換・JSONレスポンス）
type ownerResolveHandler struct {
	q *sharedquery.OwnerResolveQuery
}

func (h *ownerResolveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
