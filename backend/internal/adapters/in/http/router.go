package httpin

import (
	"net/http"
	"strings"

	firebaseauth "firebase.google.com/go/v4/auth"

	usecase "narratives/internal/application/usecase"

	// ハンドラ群
	"narratives/internal/adapters/in/http/handlers"
	"narratives/internal/adapters/in/http/middleware"

	// MessageHandler 用 Repository
	msgrepo "narratives/internal/adapters/out/firestore"
	memdom "narratives/internal/domain/member"
)

// RouterDeps collects all usecases (and other dependencies) injected from main.go.
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
	ProductUC          *usecase.ProductUsecase
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

	// ★ 招待情報取得用 Usecase（InvitationQueryPort）
	InvitationQuery usecase.InvitationQueryPort
	// ★ 招待メール発行用 Usecase（InvitationCommandPort）
	InvitationCommand usecase.InvitationCommandPort

	// Firebase Auth + MemberRepo (認証)
	FirebaseAuth *firebaseauth.Client
	MemberRepo   memdom.Repository

	// MessageHandler 用
	MessageRepo *msgrepo.MessageRepositoryFS
}

// ============================================================================
// Router 本体
// ============================================================================
func NewRouter(deps RouterDeps) http.Handler {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// ============================================================
	// Auth Middleware
	// ============================================================
	var authMw *middleware.AuthMiddleware
	if deps.FirebaseAuth != nil && deps.MemberRepo != nil {
		authMw = &middleware.AuthMiddleware{
			FirebaseAuth: deps.FirebaseAuth,
			MemberRepo:   deps.MemberRepo,
		}
	}

	// ============================================================
	// Invitation (未ログインでアクセス可能)
	// GET /api/invitation?token=xxx
	// ============================================================
	if deps.InvitationQuery != nil {
		mux.Handle(
			"/api/invitation",
			handlers.NewInvitationHandler(deps.InvitationQuery),
		)
	}

	// ============================================================
	// Members（認証必須）
	// ============================================================
	if deps.MemberUC != nil {
		// MemberHandler（招待も内包）
		memberH := handlers.NewMemberHandler(
			deps.MemberUC,
			deps.InvitationCommand, // ★ sendInvitation 用
		)

		var securedMemberHandler http.Handler = memberH
		if authMw != nil {
			securedMemberHandler = authMw.Handler(securedMemberHandler)
		}

		// POST /members, GET /members
		mux.Handle("/members", securedMemberHandler)

		// /members/... (id, invitation 判定)
		mux.Handle("/members/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			// POST /members/{id}/invitation → sendInvitation (MemberHandler 内部)
			if r.Method == http.MethodPost &&
				strings.HasPrefix(path, "/members/") &&
				strings.HasSuffix(path, "/invitation") {
				securedMemberHandler.ServeHTTP(w, r)
				return
			}

			// GET /members/{id}, PATCH /members/{id}
			securedMemberHandler.ServeHTTP(w, r)
		}))
	}

	// ============================================================
	// 他ドメインは変更なし
	// ============================================================
	if deps.AccountUC != nil {
		mux.Handle("/accounts/", handlers.NewAccountHandler(deps.AccountUC))
	}

	if deps.AnnouncementUC != nil {
		mux.Handle("/announcements/", handlers.NewAnnouncementHandler(deps.AnnouncementUC))
	}

	if deps.PermissionUC != nil {
		mux.Handle("/permissions/", handlers.NewPermissionHandler(deps.PermissionUC))
	}

	if deps.AvatarUC != nil {
		mux.Handle("/avatars/", handlers.NewAvatarHandler(deps.AvatarUC))
	}

	if deps.BrandUC != nil {
		mux.Handle("/brands/", handlers.NewBrandHandler(deps.BrandUC))
	}

	if deps.CompanyUC != nil {
		mux.Handle("/companies/", handlers.NewCompanyHandler(deps.CompanyUC))
	}

	if deps.InquiryUC != nil {
		mux.Handle("/inquiries/", handlers.NewInquiryHandler(deps.InquiryUC))
	}

	if deps.InventoryUC != nil {
		mux.Handle("/inventories/", handlers.NewInventoryHandler(deps.InventoryUC))
	}

	if deps.TokenUC != nil {
		mux.Handle("/tokens/", handlers.NewTokenHandler(deps.TokenUC))
	}

	if deps.ProductUC != nil {
		mux.Handle("/products/", handlers.NewProductHandler(deps.ProductUC))
	}

	if deps.ProductBlueprintUC != nil {
		mux.Handle("/product-blueprints/", handlers.NewProductBlueprintHandler(deps.ProductBlueprintUC))
	}

	if deps.TokenBlueprintUC != nil {
		mux.Handle("/token-blueprints/", handlers.NewTokenBlueprintHandler(deps.TokenBlueprintUC))
	}

	if deps.MessageUC != nil && deps.MessageRepo != nil {
		mux.Handle(
			"/messages/",
			handlers.NewMessageHandler(deps.MessageUC, deps.MessageRepo),
		)
	}

	if deps.OrderUC != nil {
		mux.Handle("/orders/", handlers.NewOrderHandler(deps.OrderUC))
	}

	if deps.WalletUC != nil {
		mux.Handle("/wallets/", handlers.NewWalletHandler(deps.WalletUC))
	}

	return mux
}
