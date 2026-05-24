package console

import (
	"encoding/json"
	"net/http"

	httpin "narratives/internal/adapters/in/http/console"

	consoleHandler "narratives/internal/adapters/in/http/console/handler"

	usecase "narratives/internal/application/usecase"

	"narratives/internal/adapters/in/http/middleware"

	sharedquery "narratives/internal/application/query/shared"
)

func (c *Container) RouterDeps() httpin.RouterDeps {
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
		salesH         http.Handler

		productsPrintH       http.Handler
		productBPH           http.Handler
		productBPCategoriesH http.Handler
		tokenBPH             http.Handler

		tokenBPReviewH   http.Handler
		productBPReviewH http.Handler

		messagesH         http.Handler
		ordersH           http.Handler
		walletsH          http.Handler
		membersH          http.Handler
		memberInvitationH http.Handler
		productionsH      http.Handler
		modelsH           http.Handler
		usersH            http.Handler

		inspectorH  http.Handler
		mintH       http.Handler
		invitationH http.Handler

		ownerResolveH http.Handler
	)

	if c.AuthBootstrap != nil && bootstrapMw != nil {
		authBootstrapH = consoleHandler.NewAuthBootstrapHandler(c.AuthBootstrap)
	}

	if c.AccountUC != nil {
		accountsH = consoleHandler.NewAccountHandler(c.AccountUC)
	}

	if c.AnnouncementUC != nil {
		announcementsH = consoleHandler.NewAnnouncementHandler(c.AnnouncementUC)
	}

	if c.PermissionUC != nil {
		permissionsH = consoleHandler.NewPermissionHandler(c.PermissionUC)
	}

	if c.BrandUC != nil {
		brandsH = consoleHandler.NewBrandHandler(c.BrandUC)
	}

	if c.CompanyUC != nil {
		companiesH = consoleHandler.NewCompanyHandler(c.CompanyUC)
	}

	if c.InquiryUC != nil {
		inquiriesH = consoleHandler.NewInquiryHandler(c.InquiryUC)
	}

	if c.InventoryUC != nil {
		inventoriesH = consoleHandler.NewInventoryHandlerWithListCreateQuery(
			c.InventoryUC,
			c.InventoryQuery,
			c.ListCreateQuery,
		)
	}

	if c.ListUC != nil {
		listsH = consoleHandler.NewListHandler(consoleHandler.NewListHandlerParams{
			UC: c.ListUC,

			QMgmt:   c.ListManagementQuery,
			QDetail: c.ListDetailQuery,
		})
	}

	if c.SalesQuery != nil {
		salesH = &consoleHandler.SalesHandler{
			SalesQuery: c.SalesQuery,
		}
	}

	if c.PrintUC != nil {
		productsPrintH = consoleHandler.NewPrintHandler(
			c.PrintUC,
			c.ProductionUC,
			c.ModelUC,
			c.NameResolver,
		)
	}

	if c.ProductBlueprintUC != nil {
		productBPH = consoleHandler.NewProductBlueprintHandler(
			c.ProductBlueprintUC,
			c.BrandService,
			c.MemberService,
		)
	}

	if c.ProductBlueprintCategoryUC != nil {
		productBPCategoriesH = consoleHandler.NewHandler(
			c.ProductBlueprintCategoryUC,
		)
	}

	if c.TokenBlueprintUC != nil {
		tokenBPH = consoleHandler.NewTokenBlueprintHandler(
			c.TokenBlueprintUC,
			c.TokenBlueprintQueryUC,
			c.BrandService,
		)
	}

	if c.TokenBlueprintRepo != nil && c.TokenBlueprintReviewRepo != nil {
		tbReviewUC := usecase.NewTokenBlueprintReviewUsecase(
			c.TokenBlueprintReviewRepo,
			c.AvatarRepo,
			c.TokenBlueprintRepo,
			c.BrandService,
		)
		tokenBPReviewH = consoleHandler.NewTokenBlueprintReviewHandler(tbReviewUC)
	}

	if c.ProductBlueprintRepo != nil && c.ProductBlueprintReviewRepo != nil {
		var walletRepo usecase.WalletRepository = nil

		pbReviewUC := usecase.NewProductBlueprintReviewUsecase(
			c.ProductBlueprintReviewRepo,
			walletRepo,
		).
			WithProductBlueprintRepo(c.ProductBlueprintRepo).
			WithBrandService(c.BrandService).
			WithMemberService(c.MemberService)

		if c.AvatarRepo != nil {
			pbReviewUC = pbReviewUC.WithAvatarRepo(c.AvatarRepo)
		}

		productBPReviewH = consoleHandler.NewProductBlueprintReviewHandler(pbReviewUC)
	}

	if c.OrderUC != nil && c.OrderManagementQuery != nil {
		var invBlueprint consoleHandler.InventoryBlueprintResolver
		if c.InventoryUC != nil {
			if r, ok := any(c.InventoryUC).(consoleHandler.InventoryBlueprintResolver); ok {
				invBlueprint = r
			}
		}

		var pbName consoleHandler.ProductBlueprintNameResolver
		if c.ProductBlueprintUC != nil {
			if r, ok := any(c.ProductBlueprintUC).(consoleHandler.ProductBlueprintNameResolver); ok {
				pbName = r
			}
		}

		var tbName consoleHandler.TokenBlueprintNameResolver
		if c.TokenBlueprintUC != nil {
			if r, ok := any(c.TokenBlueprintUC).(consoleHandler.TokenBlueprintNameResolver); ok {
				tbName = r
			}
		}

		var avatarName consoleHandler.AvatarNameResolver
		if c.AvatarUC != nil {
			if r, ok := any(c.AvatarUC).(consoleHandler.AvatarNameResolver); ok {
				avatarName = r
			}
		}

		var userName consoleHandler.UserNameResolver
		if c.UserUC != nil {
			if r, ok := any(c.UserUC).(consoleHandler.UserNameResolver); ok {
				userName = r
			}
		}

		var modelResolver consoleHandler.ModelResolver
		if c.NameResolver != nil {
			modelResolver = c.NameResolver
		}

		ordersH = consoleHandler.NewOrderHandler(
			c.OrderUC,
			c.OrderManagementQuery,
			invBlueprint,
			pbName,
			tbName,
			avatarName,
			userName,
			modelResolver,
		)
	}

	if c.WalletUC != nil {
		walletsH = consoleHandler.NewWalletHandler(c.WalletUC)
	}

	if c.MemberUC != nil && c.MemberRepo != nil {
		membersH = consoleHandler.NewMemberHandler(c.MemberUC, c.MemberRepo)
	}

	if c.InvitationCommand != nil {
		memberInvitationH = consoleHandler.NewMemberInvitationHandler(c.InvitationCommand)
	}

	if c.ProductionUC != nil && c.CompanyProductionQueryService != nil {
		productionsH = consoleHandler.NewProductionHandler(
			c.CompanyProductionQueryService,
			c.ProductionUC,
		)
	}

	if c.ModelUC != nil {
		modelsH = consoleHandler.NewModelHandler(c.ModelUC)
	}

	if c.ProductUC != nil && c.InspectionUC != nil {
		var pbGetter consoleHandler.ProductBlueprintModelRefGetter
		if c.ProductBlueprintRepo != nil {
			if g, ok := any(c.ProductBlueprintRepo).(consoleHandler.ProductBlueprintModelRefGetter); ok {
				pbGetter = g
			}
		}

		inspectorH = consoleHandler.NewInspectorHandler(
			c.ProductUC,
			c.InspectionUC,
			c.NameResolver,
			pbGetter,
		)
	}

	if c.MintUC != nil {
		mintH = consoleHandler.NewMintHandler(
			c.MintUC,
			c.NameResolver,
			c.ProductionUC,
			c.MintRequestQueryService,
		)
	}

	if c.OwnerResolveQ != nil {
		ownerResolveH = &ownerResolveHandler{q: c.OwnerResolveQ}
	}

	if c.InvitationQuery != nil && c.InvitationComplete != nil {
		invitationH = consoleHandler.NewInvitationHandler(
			c.InvitationQuery,
			c.InvitationComplete,
			c.CompanyService,
			c.BrandService,
		)
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
		Sales:         salesH,

		ProductsPrint:       productsPrintH,
		ProductBP:           productBPH,
		ProductBPCategories: productBPCategoriesH,
		TokenBP:             tokenBPH,

		TokenBPReview:   tokenBPReviewH,
		ProductBPReview: productBPReviewH,

		Messages:         messagesH,
		Orders:           ordersH,
		Wallets:          walletsH,
		Members:          membersH,
		MemberInvitation: memberInvitationH,

		Productions: productionsH,
		Models:      modelsH,
		Users:       usersH,
		Invitation:  invitationH,

		Inspector: inspectorH,
		Mint:      mintH,

		OwnerResolve: ownerResolveH,
	}
}

type ownerResolveHandler struct {
	q *sharedquery.OwnerResolveQuery
}

func (h *ownerResolveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	addr := q.Get("walletAddress")
	if addr == "" {
		addr = q.Get("toAddress")
	}
	if addr == "" {
		addr = q.Get("address")
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
