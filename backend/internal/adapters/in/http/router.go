// backend/internal/adapters/in/http/router.go
package httpin

import (
	"net/http"
	"strings"

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

	// ★ Bootstrap 用サービス（auth/bootstrap 用）
	AuthBootstrap *authuc.BootstrapService

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
	mux.Handle("/debug/sendgrid", http.HandlerFunc(DebugSendGridHandler))

	// ============================================================
	// Auth Middleware（通常エンドポイント用）
	// ============================================================
	var authMw *middleware.AuthMiddleware
	if deps.FirebaseAuth != nil && deps.MemberRepo != nil {
		authMw = &middleware.AuthMiddleware{
			FirebaseAuth: deps.FirebaseAuth,
			MemberRepo:   deps.MemberRepo,
		}
	}

	// ============================================================
	// Bootstrap 用 Auth Middleware（/auth/bootstrap 専用）
	//   - Firebase ID トークン検証のみ
	//   - MemberRepo は使わない
	// ============================================================
	var bootstrapAuthMw *middleware.BootstrapAuthMiddleware
	if deps.FirebaseAuth != nil {
		bootstrapAuthMw = &middleware.BootstrapAuthMiddleware{
			FirebaseAuth: deps.FirebaseAuth,
		}
	}

	// ============================================================
	// auth/bootstrap （サインアップ後の初期セットアップ）
	// ============================================================
	if deps.AuthBootstrap != nil {
		bootstrapH := handlers.NewAuthBootstrapHandler(deps.AuthBootstrap)

		var securedBootstrap http.Handler = bootstrapH
		if bootstrapAuthMw != nil {
			// ★ UID / email だけ検証するミドルウェア
			securedBootstrap = bootstrapAuthMw.Handler(securedBootstrap)
		}

		// フロントが叩いているパスに合わせる: POST /auth/bootstrap
		mux.Handle("/auth/bootstrap", securedBootstrap)
	}

	// ============================================================
	// Invitation (未ログインでアクセス可能)
	//   - GET /api/invitation?token=xxx
	//   - POST /api/invitation/validate
	//   - POST /api/invitation/complete
	//   を 1 つのハンドラでまとめて扱う
	// ============================================================
	if deps.InvitationQuery != nil {
		invH := handlers.NewInvitationHandler(deps.InvitationQuery)

		// クエリだけのパターン
		mux.Handle("/api/invitation", invH)
		// サブパス (/validate, /complete など) も同じハンドラで受ける
		mux.Handle("/api/invitation/", invH)
	}

	// ============================================================
	// Members（認証必須）
	//   - MemberHandler: /members, /members/{id}
	//   - MemberInvitationHandler: /members/{id}/invitation
	// ============================================================
	if deps.MemberUC != nil {
		// 通常の MemberHandler
		memberH := handlers.NewMemberHandler(
			deps.MemberUC,
		)

		var securedMemberHandler http.Handler = memberH
		if authMw != nil {
			securedMemberHandler = authMw.Handler(securedMemberHandler)
		}

		// POST /members, GET /members
		mux.Handle("/members", securedMemberHandler)

		// 招待メール用ハンドラ（InvitationCommandPort 経由）
		var securedInvitationHandler http.Handler
		if deps.InvitationCommand != nil {
			invH := handlers.NewMemberInvitationHandler(deps.InvitationCommand)
			securedInvitationHandler = invH
			if authMw != nil {
				securedInvitationHandler = authMw.Handler(securedInvitationHandler)
			}
		}

		// /members/... (id or invitation)
		mux.Handle("/members/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			// POST /members/{id}/invitation → MemberInvitationHandler
			if r.Method == http.MethodPost &&
				strings.HasPrefix(path, "/members/") &&
				strings.HasSuffix(path, "/invitation") &&
				securedInvitationHandler != nil {
				securedInvitationHandler.ServeHTTP(w, r)
				return
			}

			// GET /members/{id}, PATCH /members/{id} → MemberHandler
			securedMemberHandler.ServeHTTP(w, r)
		}))
	}

	// ============================================================
	// 他ドメインは変更なし
	// ============================================================
	mux.Handle("/accounts", handlers.NewAccountHandler(deps.AccountUC))
	mux.Handle("/accounts/", handlers.NewAccountHandler(deps.AccountUC))

	mux.Handle("/announcements", handlers.NewAnnouncementHandler(deps.AnnouncementUC))
	mux.Handle("/announcements/", handlers.NewAnnouncementHandler(deps.AnnouncementUC))

	mux.Handle("/permissions", handlers.NewPermissionHandler(deps.PermissionUC))
	mux.Handle("/permissions/", handlers.NewPermissionHandler(deps.PermissionUC))

	mux.Handle("/avatars", handlers.NewAvatarHandler(deps.AvatarUC))
	mux.Handle("/avatars/", handlers.NewAvatarHandler(deps.AvatarUC))

	mux.Handle("/brands", handlers.NewBrandHandler(deps.BrandUC))
	mux.Handle("/brands/", handlers.NewBrandHandler(deps.BrandUC))

	mux.Handle("/companies", handlers.NewCompanyHandler(deps.CompanyUC))
	mux.Handle("/companies/", handlers.NewCompanyHandler(deps.CompanyUC))

	mux.Handle("/inquiries", handlers.NewInquiryHandler(deps.InquiryUC))
	mux.Handle("/inquiries/", handlers.NewInquiryHandler(deps.InquiryUC))

	mux.Handle("/inventories", handlers.NewInventoryHandler(deps.InventoryUC))
	mux.Handle("/inventories/", handlers.NewInventoryHandler(deps.InventoryUC))

	mux.Handle("/tokens", handlers.NewTokenHandler(deps.TokenUC))
	mux.Handle("/tokens/", handlers.NewTokenHandler(deps.TokenUC))

	mux.Handle("/products", handlers.NewProductHandler(deps.ProductUC))
	mux.Handle("/products/", handlers.NewProductHandler(deps.ProductUC))

	mux.Handle("/product-blueprints", handlers.NewProductBlueprintHandler(deps.ProductBlueprintUC))
	mux.Handle("/product-blueprints/", handlers.NewProductBlueprintHandler(deps.ProductBlueprintUC))

	mux.Handle("/token-blueprints", handlers.NewTokenBlueprintHandler(deps.TokenBlueprintUC))
	mux.Handle("/token-blueprints/", handlers.NewTokenBlueprintHandler(deps.TokenBlueprintUC))

	mux.Handle("/messages", handlers.NewMessageHandler(deps.MessageUC, deps.MessageRepo))
	mux.Handle("/messages/", handlers.NewMessageHandler(deps.MessageUC, deps.MessageRepo))

	mux.Handle("/orders", handlers.NewOrderHandler(deps.OrderUC))
	mux.Handle("/orders/", handlers.NewOrderHandler(deps.OrderUC))

	mux.Handle("/wallets", handlers.NewWalletHandler(deps.WalletUC))
	mux.Handle("/wallets/", handlers.NewWalletHandler(deps.WalletUC))

	return mux
}
