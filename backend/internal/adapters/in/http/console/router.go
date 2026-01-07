// backend/internal/adapters/in/http/console/router.go
package httpin

import (
	"net/http"

	firebaseauth "firebase.google.com/go/v4/auth"

	// ★ MintUsecase 移動先
	mintapp "narratives/internal/application/mint"

	// ★ InspectionUsecase 移動先
	inspectionapp "narratives/internal/application/inspection"

	usecase "narratives/internal/application/usecase"
	authuc "narratives/internal/application/usecase/auth"

	// ★ new: Production 用の新パッケージ
	productionapp "narratives/internal/application/production"

	// ★ new: Query services
	companyquery "narratives/internal/application/query/console"

	// ✅ console handlers（正）
	consoleHandler "narratives/internal/adapters/in/http/console/handler"

	"narratives/internal/adapters/in/http/middleware"

	resolver "narratives/internal/application/resolver"

	// MessageHandler 用 Repository
	msgrepo "narratives/internal/adapters/out/firestore"

	// ドメインサービス
	branddom "narratives/internal/domain/brand"
	memdom "narratives/internal/domain/member"
)

type RouterDeps struct {
	AccountUC        *usecase.AccountUsecase
	AnnouncementUC   *usecase.AnnouncementUsecase
	AvatarUC         *usecase.AvatarUsecase
	BillingAddressUC *usecase.BillingAddressUsecase
	BrandUC          *usecase.BrandUsecase
	CampaignUC       *usecase.CampaignUsecase
	CompanyUC        *usecase.CompanyUsecase
	InquiryUC        *usecase.InquiryUsecase
	InventoryUC      *usecase.InventoryUsecase
	InvoiceUC        *usecase.InvoiceUsecase
	ListUC           *usecase.ListUsecase
	MemberUC         *usecase.MemberUsecase
	MessageUC        *usecase.MessageUsecase
	ModelUC          *usecase.ModelUsecase
	OrderUC          *usecase.OrderUsecase
	PaymentUC        *usecase.PaymentUsecase
	PermissionUC     *usecase.PermissionUsecase
	PrintUC          *usecase.PrintUsecase
	TokenUC          *usecase.TokenUsecase

	// ★ここだけ型を新パッケージに変更
	ProductionUC       *productionapp.ProductionUsecase
	ProductBlueprintUC *usecase.ProductBlueprintUsecase
	ShippingAddressUC  *usecase.ShippingAddressUsecase
	TokenBlueprintUC   *usecase.TokenBlueprintUsecase
	TokenOperationUC   *usecase.TokenOperationUsecase
	TrackingUC         *usecase.TrackingUsecase
	UserUC             *usecase.UserUsecase
	WalletUC           *usecase.WalletUsecase

	// ★ 追加: Company → ProductBlueprintIds → Productions の Query 専用（GET一覧）
	CompanyProductionQueryService *companyquery.CompanyProductionQueryService

	// ★ NEW: Inventory detail の read-model assembler（/inventory/...）
	InventoryQuery *companyquery.InventoryQuery

	// ✅ NEW: listCreate DTO assembler
	ListCreateQuery *companyquery.ListCreateQuery

	// ✅ NEW: Lists の read-model assembler
	ListManagementQuery *companyquery.ListManagementQuery

	// ✅ NEW: List detail DTO assembler
	ListDetailQuery *companyquery.ListDetailQuery

	// ✅ NEW: ListImage uploader/deleter
	ListImageUploader consoleHandler.ListImageUploader
	ListImageDeleter  consoleHandler.ListImageDeleter

	// ★ NameResolver（ID→名前/型番解決）
	NameResolver *resolver.NameResolver

	// ⭐ Inspector 用 ProductUsecase
	ProductUC *usecase.ProductUsecase

	// ⭐ 検品専用 Usecase（★ moved）
	InspectionUC *inspectionapp.InspectionUsecase

	// ⭐ Mint 用 Usecase
	MintUC *mintapp.MintUsecase

	// 認証・招待まわり
	AuthBootstrap     *authuc.BootstrapService
	InvitationQuery   usecase.InvitationQueryPort
	InvitationCommand usecase.InvitationCommandPort

	// Firebase / MemberRepo
	FirebaseAuth *firebaseauth.Client
	MemberRepo   memdom.Repository

	// ★ member.Service（表示名解決用）
	MemberService *memdom.Service

	// ★ brand.Service（ブランド名解決用）
	BrandService *branddom.Service

	// Message 用の Firestore Repository
	MessageRepo *msgrepo.MessageRepositoryFS

	// ★ MintRequest の query
	MintRequestQueryService consoleHandler.MintRequestQueryService
}

func NewRouter(deps RouterDeps) http.Handler {
	mux := http.NewServeMux()

	// ================================
	// 共通 Auth ミドルウェア
	// ================================
	var authMw *middleware.AuthMiddleware
	if deps.FirebaseAuth != nil && deps.MemberRepo != nil {
		authMw = &middleware.AuthMiddleware{
			FirebaseAuth: deps.FirebaseAuth,
			MemberRepo:   deps.MemberRepo,
		}
	}

	// ================================
	// /auth/bootstrap 専用
	// ================================
	var bootstrapMw *middleware.BootstrapAuthMiddleware
	if deps.FirebaseAuth != nil {
		bootstrapMw = &middleware.BootstrapAuthMiddleware{
			FirebaseAuth: deps.FirebaseAuth,
		}
	}

	// ================================
	// /auth/bootstrap
	// ================================
	if deps.AuthBootstrap != nil && bootstrapMw != nil {
		bootstrapHandler := consoleHandler.NewAuthBootstrapHandler(deps.AuthBootstrap)
		var h http.Handler = bootstrapHandler
		h = bootstrapMw.Handler(h)
		mux.Handle("/auth/bootstrap", h)
	}

	// ================================
	// Accounts
	// ================================
	if deps.AccountUC != nil {
		accountH := consoleHandler.NewAccountHandler(deps.AccountUC)
		var h http.Handler = accountH
		if authMw != nil {
			h = authMw.Handler(h)
		}
		mux.Handle("/accounts", h)
		mux.Handle("/accounts/", h)
	}

	// ================================
	// Announcements
	// ================================
	if deps.AnnouncementUC != nil {
		announcementH := consoleHandler.NewAnnouncementHandler(deps.AnnouncementUC)
		var h http.Handler = announcementH
		if authMw != nil {
			h = authMw.Handler(h)
		}
		mux.Handle("/announcements", h)
		mux.Handle("/announcements/", h)
	}

	// ================================
	// Permissions
	// ================================
	if deps.PermissionUC != nil {
		permissionH := consoleHandler.NewPermissionHandler(deps.PermissionUC)
		var h http.Handler = permissionH
		if authMw != nil {
			h = authMw.Handler(h)
		}
		mux.Handle("/permissions", h)
		mux.Handle("/permissions/", h)
	}

	// ================================
	// Brands
	// ================================
	if deps.BrandUC != nil {
		brandH := consoleHandler.NewBrandHandler(deps.BrandUC)
		var h http.Handler = brandH
		if authMw != nil {
			h = authMw.Handler(h)
		}
		mux.Handle("/brands", h)
		mux.Handle("/brands/", h)
	}

	// ================================
	// Companies
	// ================================
	if deps.CompanyUC != nil {
		companyH := consoleHandler.NewCompanyHandler(deps.CompanyUC)
		var h http.Handler = companyH
		if authMw != nil {
			h = authMw.Handler(h)
		}
		mux.Handle("/companies", h)
		mux.Handle("/companies/", h)
	}

	// ================================
	// Inquiries
	// ================================
	if deps.InquiryUC != nil {
		inquiryH := consoleHandler.NewInquiryHandler(deps.InquiryUC)
		var h http.Handler = inquiryH
		if authMw != nil {
			h = authMw.Handler(h)
		}
		mux.Handle("/inquiries", h)
		mux.Handle("/inquiries/", h)
	}

	// ================================
	// Inventories
	// ================================
	if deps.InventoryUC != nil {
		inventoryH := consoleHandler.NewInventoryHandlerWithListCreateQuery(
			deps.InventoryUC,
			deps.InventoryQuery,
			deps.ListCreateQuery,
		)

		var h http.Handler = inventoryH
		if authMw != nil {
			h = authMw.Handler(h)
		}

		mux.Handle("/inventories", h)
		mux.Handle("/inventories/", h)

		mux.Handle("/inventory", h)
		mux.Handle("/inventory/", h)
	}

	// ================================
	// Lists
	// ================================
	if deps.ListUC != nil {
		var listH http.Handler

		if deps.ListImageUploader != nil || deps.ListImageDeleter != nil {
			listH = consoleHandler.NewListHandlerWithQueriesAndListImage(
				deps.ListUC,
				deps.ListManagementQuery,
				deps.ListDetailQuery,
				deps.ListImageUploader,
				deps.ListImageDeleter,
			)
		} else if deps.ListManagementQuery != nil || deps.ListDetailQuery != nil {
			listH = consoleHandler.NewListHandlerWithQueries(
				deps.ListUC,
				deps.ListManagementQuery,
				deps.ListDetailQuery,
			)
		} else {
			listH = consoleHandler.NewListHandler(deps.ListUC)
		}

		var h http.Handler = listH
		if authMw != nil {
			h = authMw.Handler(h)
		}
		mux.Handle("/lists", h)
		mux.Handle("/lists/", h)
	}

	// ================================
	// Products（印刷系）
	// ================================
	if deps.PrintUC != nil {
		printH := consoleHandler.NewPrintHandler(
			deps.PrintUC,
			deps.ProductionUC,
			deps.ModelUC,
			deps.NameResolver,
		)

		var h http.Handler = printH
		if authMw != nil {
			h = authMw.Handler(h)
		}

		mux.Handle("/products", h)
		mux.Handle("/products/", h)
		mux.Handle("/products/print-logs", h)
	}

	// ================================
	// Product Blueprints
	// ================================
	if deps.ProductBlueprintUC != nil {
		pbH := consoleHandler.NewProductBlueprintHandler(
			deps.ProductBlueprintUC,
			deps.BrandService,
			deps.MemberService,
		)

		var h http.Handler = pbH
		if authMw != nil {
			h = authMw.Handler(h)
		}

		mux.Handle("/product-blueprints", h)
		mux.Handle("/product-blueprints/", h)
	}

	// ================================
	// Token Blueprints
	// ================================
	if deps.TokenBlueprintUC != nil {
		tbH := consoleHandler.NewTokenBlueprintHandler(
			deps.TokenBlueprintUC,
			deps.MemberService,
			deps.BrandService,
		)

		var h http.Handler = tbH
		if authMw != nil {
			h = authMw.Handler(h)
		}

		mux.Handle("/token-blueprints", h)
		mux.Handle("/token-blueprints/", h)
	}

	// ================================
	// Messages
	// ================================
	if deps.MessageUC != nil && deps.MessageRepo != nil {
		messageH := consoleHandler.NewMessageHandler(deps.MessageUC, deps.MessageRepo)

		var h http.Handler = messageH
		if authMw != nil {
			h = authMw.Handler(h)
		}

		mux.Handle("/messages", h)
		mux.Handle("/messages/", h)
	}

	// ================================
	// Orders
	// ================================
	if deps.OrderUC != nil {
		orderH := consoleHandler.NewOrderHandler(deps.OrderUC)

		var h http.Handler = orderH
		if authMw != nil {
			h = authMw.Handler(h)
		}

		mux.Handle("/orders", h)
		mux.Handle("/orders/", h)
	}

	// ================================
	// Wallets
	// ================================
	if deps.WalletUC != nil {
		walletH := consoleHandler.NewWalletHandler(deps.WalletUC)

		var h http.Handler = walletH
		if authMw != nil {
			h = authMw.Handler(h)
		}

		mux.Handle("/wallets", h)
		mux.Handle("/wallets/", h)
	}

	// ================================
	// Members
	// ================================
	if deps.MemberUC != nil && deps.MemberRepo != nil {
		memberH := consoleHandler.NewMemberHandler(deps.MemberUC, deps.MemberRepo)

		var h http.Handler = memberH
		if authMw != nil {
			h = authMw.Handler(h)
		}

		mux.Handle("/members", h)
		mux.Handle("/members/", h)
	}

	// ================================
	// Productions
	// ================================
	if deps.ProductionUC != nil && deps.CompanyProductionQueryService != nil {
		productionH := consoleHandler.NewProductionHandler(
			deps.CompanyProductionQueryService,
			deps.ProductionUC,
		)

		var h http.Handler = productionH
		if authMw != nil {
			h = authMw.Handler(h)
		}

		mux.Handle("/productions", h)
		mux.Handle("/productions/", h)
	}

	// ================================
	// Models
	// ================================
	if deps.ModelUC != nil {
		modelH := consoleHandler.NewModelHandler(deps.ModelUC)

		var h http.Handler = modelH
		if authMw != nil {
			h = authMw.Handler(h)
		}
		mux.Handle("/models", h)
		mux.Handle("/models/", h)
	}

	// ================================
	// Inspector
	// ================================
	if deps.ProductUC != nil && deps.InspectionUC != nil {
		inspectorH := consoleHandler.NewInspectorHandler(deps.ProductUC, deps.InspectionUC)

		var h http.Handler = inspectorH
		if authMw != nil {
			h = authMw.Handler(h)
		}

		mux.Handle("/inspector/products/", h)
		mux.Handle("/products/inspections", h)
		mux.Handle("/products/inspections/", h)
	}

	// ================================
	// Mint
	// ================================
	if deps.MintUC != nil {
		mintH := consoleHandler.NewMintHandler(
			deps.MintUC,
			deps.TokenUC,
			deps.NameResolver,
			deps.ProductionUC,
			deps.MintRequestQueryService,
			nil,
		)

		if mh, ok := mintH.(*consoleHandler.MintHandler); ok {
			mux.HandleFunc("/mint/debug", mh.HandleDebug)
		}

		var h http.Handler = mintH
		if authMw != nil {
			h = authMw.Handler(h)
		}

		mux.Handle("/mint/", h)
	}

	return mux
}
