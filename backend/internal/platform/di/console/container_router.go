package console

import (
	"net/http"

	httpin "narratives/internal/adapters/in/http/console"

	consoleHandler "narratives/internal/adapters/in/http/console/handler"

	usecase "narratives/internal/application/usecase"

	"narratives/internal/adapters/in/http/middleware"
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

		messagesH    http.Handler
		ordersH      http.Handler
		walletsH     http.Handler
		membersH     http.Handler
		productionsH http.Handler
		modelsH      http.Handler
		usersH       http.Handler

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

	if c.BrandUC != nil &&
		c.BrandManagementQuery != nil &&
		c.BrandDetailQuery != nil {
		brandsH = consoleHandler.NewBrandHandler(
			c.BrandUC,
			c.BrandManagementQuery,
			c.BrandDetailQuery,
		)
	}

	if c.CompanyUC != nil {
		companiesH = consoleHandler.NewCompanyHandler(c.CompanyUC)
	}

	if c.InquiryUC != nil {
		inquiriesH = consoleHandler.NewInquiryHandler(c.InquiryUC)
	}

	if c.InventoryManagementQuery != nil &&
		c.InventoryDetailQuery != nil &&
		c.ListCreateQuery != nil {
		inventoriesH = consoleHandler.NewInventoryHandlerWithListCreateQuery(
			c.InventoryManagementQuery,
			c.InventoryDetailQuery,
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

	if c.PrintUC != nil && c.PrintQueryService != nil {
		productsPrintH = consoleHandler.NewPrintHandler(
			c.PrintUC,
			c.PrintQueryService,
		)
	}

	if c.ProductBlueprintUC != nil &&
		c.ProductBlueprintManagementQuery != nil &&
		c.ProductBlueprintDetailQuery != nil {
		productBPH = consoleHandler.NewProductBlueprintHandler(
			c.ProductBlueprintUC,
			c.ProductBlueprintManagementQuery,
			c.ProductBlueprintDetailQuery,
		)
	}

	if c.ProductBlueprintCategoryUC != nil {
		productBPCategoriesH = consoleHandler.NewProductBlueprintCategoryHandler(
			c.ProductBlueprintCategoryUC,
		)
	}

	if c.TokenBlueprintUC != nil &&
		c.TokenBlueprintManagementQuery != nil &&
		c.TokenBlueprintDetailQuery != nil {
		tokenBPH = consoleHandler.NewTokenBlueprintHandler(
			c.TokenBlueprintUC,
			c.TokenBlueprintDetailQuery,
			c.TokenBlueprintManagementQuery,
		)
	}

	if c.TokenBlueprintRepo != nil &&
		c.TokenBlueprintReviewRepo != nil &&
		c.BrandRepo != nil {
		tbReviewUC := usecase.NewTokenBlueprintReviewUsecase(
			c.TokenBlueprintReviewRepo,
			c.AvatarRepo,
			c.TokenBlueprintRepo,
			c.BrandRepo,
		)
		tokenBPReviewH = consoleHandler.NewTokenBlueprintReviewHandler(tbReviewUC)
	}

	if c.ProductBlueprintRepo != nil &&
		c.ProductBlueprintReviewRepo != nil &&
		c.BrandRepo != nil {
		var walletRepo usecase.WalletRepository = nil

		pbReviewUC := usecase.NewProductBlueprintReviewUsecase(
			c.ProductBlueprintReviewRepo,
			walletRepo,
			c.ProductBlueprintRepo,
			c.BrandRepo,
			c.MemberService,
			nil,
			nil,
			nil,
			nil,
			c.AvatarRepo,
			nil,
		)

		productBPReviewH = consoleHandler.NewProductBlueprintReviewHandler(pbReviewUC)
	}

	if c.OrderManagementQuery != nil || c.OrderDetailQuery != nil {
		ordersH = consoleHandler.NewOrderHandler(
			c.OrderManagementQuery,
			c.OrderDetailQuery,
		)
	}

	if c.WalletUC != nil {
		walletsH = consoleHandler.NewWalletHandler(c.WalletUC)
	}

	if c.MemberRepo != nil {
		membersH = consoleHandler.NewMemberHandler(c.MemberRepo)
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

	if c.InspectionUC != nil && c.InspectorQuery != nil {
		var pbGetter consoleHandler.ProductBlueprintModelRefGetter
		if c.ProductBlueprintRepo != nil {
			if g, ok := any(c.ProductBlueprintRepo).(consoleHandler.ProductBlueprintModelRefGetter); ok {
				pbGetter = g
			}
		}

		inspectorH = consoleHandler.NewInspectorHandler(
			c.InspectionUC,
			c.InspectorQuery,
			c.NameResolver,
			pbGetter,
		)
	}

	if c.MintUC != nil {
		mintH = consoleHandler.NewMintHandler(
			c.MintUC,
			c.MintRequestQueryService,
		)
	}

	if c.OwnerResolveQ != nil {
		ownerResolveH = consoleHandler.NewOwnerResolveHandler(c.OwnerResolveQ)
	}

	if c.InvitationUC != nil {
		invitationH = consoleHandler.NewInvitationHandler(
			c.InvitationUC,
			c.CompanyRepo,
			c.BrandRepo,
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

		Messages: messagesH,
		Orders:   ordersH,
		Wallets:  walletsH,
		Members:  membersH,

		Productions: productionsH,
		Models:      modelsH,
		Users:       usersH,
		Invitation:  invitationH,

		Inspector: inspectorH,
		Mint:      mintH,

		OwnerResolve: ownerResolveH,
	}
}
