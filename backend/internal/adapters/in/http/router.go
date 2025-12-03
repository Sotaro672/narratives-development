// backend/internal/adapters/in/http/router.go
package httpin

import (
	"net/http"

	firebaseauth "firebase.google.com/go/v4/auth"

	usecase "narratives/internal/application/usecase"
	authuc "narratives/internal/application/usecase/auth"

	// ハンドラ群
	"narratives/internal/adapters/in/http/handlers"
	"narratives/internal/adapters/in/http/middleware"

	// MessageHandler 用 Repository
	msgrepo "narratives/internal/adapters/out/firestore"
	memdom "narratives/internal/domain/member"
)

type RouterDeps struct {
	AccountUC          *usecase.AccountUsecase
	AnnouncementUC     *usecase.AnnouncementUsecase
	AvatarUC           *usecase.AvatarUsecase
	BillingAddressUC   *usecase.BillingAddressUsecase
	BrandUC            *usecase.BrandUsecase
	CampaignUC         *usecase.CampaignUsecase
	CompanyUC          *usecase.CompanyUsecase
	DiscountUC         *usecase.DiscountUsecase
	FulfillmentUC      *usecase.FulfillmentUsecase
	InquiryUC          *usecase.InquiryUsecase
	InventoryUC        *usecase.InventoryUsecase
	InvoiceUC          *usecase.InvoiceUsecase
	ListUC             *usecase.ListUsecase
	MemberUC           *usecase.MemberUsecase
	MessageUC          *usecase.MessageUsecase
	MintRequestUC      *usecase.MintRequestUsecase
	ModelUC            *usecase.ModelUsecase
	OrderUC            *usecase.OrderUsecase
	PaymentUC          *usecase.PaymentUsecase
	PermissionUC       *usecase.PermissionUsecase
	PrintUC            *usecase.PrintUsecase
	ProductionUC       *usecase.ProductionUsecase
	ProductBlueprintUC *usecase.ProductBlueprintUsecase
	SaleUC             *usecase.SaleUsecase
	ShippingAddressUC  *usecase.ShippingAddressUsecase
	TokenUC            *usecase.TokenUsecase
	TokenBlueprintUC   *usecase.TokenBlueprintUsecase
	TokenOperationUC   *usecase.TokenOperationUsecase
	TrackingUC         *usecase.TrackingUsecase
	UserUC             *usecase.UserUsecase
	WalletUC           *usecase.WalletUsecase

	// ⭐ Inspector 用 ProductUsecase（/inspector/products/{id}）
	ProductUC *usecase.ProductUsecase

	// ⭐ 検品専用 Usecase
	InspectionUC *usecase.InspectionUsecase

	// 認証・招待まわり
	AuthBootstrap     *authuc.BootstrapService
	InvitationQuery   usecase.InvitationQueryPort
	InvitationCommand usecase.InvitationCommandPort

	// Firebase / MemberRepo
	FirebaseAuth *firebaseauth.Client
	MemberRepo   memdom.Repository

	// Message 用の Firestore Repository
	MessageRepo *msgrepo.MessageRepositoryFS
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
		bootstrapHandler := handlers.NewAuthBootstrapHandler(deps.AuthBootstrap)
		var h http.Handler = bootstrapHandler
		h = bootstrapMw.Handler(h)
		mux.Handle("/auth/bootstrap", h)
	}

	// ================================
	// Accounts
	// ================================
	if deps.AccountUC != nil {
		accountH := handlers.NewAccountHandler(deps.AccountUC)
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
		announcementH := handlers.NewAnnouncementHandler(deps.AnnouncementUC)
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
		permissionH := handlers.NewPermissionHandler(deps.PermissionUC)
		var h http.Handler = permissionH
		if authMw != nil {
			h = authMw.Handler(h)
		}
		mux.Handle("/permissions", h)
		mux.Handle("/permissions/", h)
	}

	// ================================
	// Avatars
	// ================================
	if deps.AvatarUC != nil {
		avatarH := handlers.NewAvatarHandler(deps.AvatarUC)
		var h http.Handler = avatarH
		if authMw != nil {
			h = authMw.Handler(h)
		}
		mux.Handle("/avatars", h)
		mux.Handle("/avatars/", h)
	}

	// ================================
	// Brands
	// ================================
	if deps.BrandUC != nil {
		brandH := handlers.NewBrandHandler(deps.BrandUC)
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
		companyH := handlers.NewCompanyHandler(deps.CompanyUC)
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
		inquiryH := handlers.NewInquiryHandler(deps.InquiryUC)
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
		inventoryH := handlers.NewInventoryHandler(deps.InventoryUC)
		var h http.Handler = inventoryH
		if authMw != nil {
			h = authMw.Handler(h)
		}
		mux.Handle("/inventories", h)
		mux.Handle("/inventories/", h)
	}

	// ================================
	// Tokens
	// ================================
	if deps.TokenUC != nil {
		tokenH := handlers.NewTokenHandler(deps.TokenUC)
		var h http.Handler = tokenH
		if authMw != nil {
			h = authMw.Handler(h)
		}
		mux.Handle("/tokens", h)
		mux.Handle("/tokens/", h)
	}

	// ================================
	// Products
	// ================================
	if deps.PrintUC != nil {
		printH := handlers.NewPrintHandler(deps.PrintUC, deps.ProductionUC, deps.ModelUC)

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
		pbH := handlers.NewProductBlueprintHandler(deps.ProductBlueprintUC, nil)

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
		tbH := handlers.NewTokenBlueprintHandler(deps.TokenBlueprintUC)

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
		messageH := handlers.NewMessageHandler(deps.MessageUC, deps.MessageRepo)

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
		orderH := handlers.NewOrderHandler(deps.OrderUC)

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
		walletH := handlers.NewWalletHandler(deps.WalletUC)

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
	if deps.MemberUC != nil {
		memberH := handlers.NewMemberHandler(deps.MemberUC)

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
	if deps.ProductionUC != nil {
		productionH := handlers.NewProductionHandler(deps.ProductionUC)

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
		modelH := handlers.NewModelHandler(deps.ModelUC)

		var h http.Handler = modelH
		if authMw != nil {
			h = authMw.Handler(h)
		}

		mux.Handle("/models", h)
		mux.Handle("/models/", h)
	}

	// ================================
	// Inspector Products (検品アプリ用詳細)
	//   GET /inspector/products/{id}
	//   → ProductUsecase + ProductHandler
	// ================================
	if deps.ProductUC != nil {
		inspectorProductH := handlers.NewProductHandler(deps.ProductUC)

		var h http.Handler = inspectorProductH
		if authMw != nil {
			h = authMw.Handler(h)
		}

		// /inspector/products/{id} をこのハンドラに紐付け
		mux.Handle("/inspector/products/", h)
	}

	// ================================
	// ⭐ 検品 API（Inspector 用）
	//   GET  /products/inspections
	//   PATCH /products/inspections
	//   PATCH /products/inspections/complete
	// ================================
	if deps.PrintUC != nil && deps.InspectionUC != nil {
		inspectorH := handlers.NewInspectorHandler(deps.PrintUC, deps.InspectionUC)

		var h http.Handler = inspectorH
		// Flutter inspector アプリは Firebase Auth を使っており認証必須
		if authMw != nil {
			h = authMw.Handler(h)
		}

		mux.Handle("/products/inspections", h)
		mux.Handle("/products/inspections/", h) // ← /complete などもこのハンドラに流す
	}

	return mux
}
