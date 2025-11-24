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
	ModelUC            *usecase.ModelUsecase // ★ これを使う
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

	AuthBootstrap     *authuc.BootstrapService
	InvitationQuery   usecase.InvitationQueryPort
	InvitationCommand usecase.InvitationCommandPort

	FirebaseAuth *firebaseauth.Client
	MemberRepo   memdom.Repository

	MessageRepo *msgrepo.MessageRepositoryFS
}

func NewRouter(deps RouterDeps) http.Handler {
	mux := http.NewServeMux()

	// healthz, debug/sendgrid などは省略（そのまま）

	var authMw *middleware.AuthMiddleware
	if deps.FirebaseAuth != nil && deps.MemberRepo != nil {
		authMw = &middleware.AuthMiddleware{
			FirebaseAuth: deps.FirebaseAuth,
			MemberRepo:   deps.MemberRepo,
		}
	}

	// ... bootstrap, invitation, members 設定は現状のまま ...

	// ============================================================
	// 既存ドメインの登録（そのまま）
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

	// ============================================================
	// ★ Models エンドポイントを追加
	//   - POST /models/{productID}/variations
	//   - GET  /models/{id}
	// ============================================================
	if deps.ModelUC != nil {
		modelH := handlers.NewModelHandler(deps.ModelUC)

		var securedModelHandler http.Handler = modelH
		if authMw != nil {
			securedModelHandler = authMw.Handler(securedModelHandler)
		}

		mux.Handle("/models", securedModelHandler)
		mux.Handle("/models/", securedModelHandler)
	}

	return mux
}
