package httpin

import (
	"net/http"

	usecase "narratives/internal/application/usecase"

	// ハンドラ群
	"narratives/internal/adapters/in/http/handlers"

	// MessageHandler だけは repo も必要なので、router で受け取って渡す
	msgrepo "narratives/internal/adapters/out/firestore"
	// ↑ MessageRepositoryPG がいるパッケージを alias import
	// もし MessageRepositoryPG が firestore/message_repository_pg.go にあり、
	// パッケージ宣言が `package firestore` ならこの import は
	//    "narratives/internal/adapters/out/firestore"
	// でOKです。上の msgrepo は読みやすさのための別名です。
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

	// ★ 追加: MessageHandler が直接必要とする Repository
	//
	// interface で受けたいなら
	//   MessageRepo message.Repository
	// みたいにしてもいいですが、
	// 現状は concrete でも問題ないので PG 実装をそのまま持たせる。
	MessageRepo *msgrepo.MessageRepositoryPG
}

// NewRouter sets up HTTP routing for all domain endpoints.
func NewRouter(deps RouterDeps) http.Handler {
	mux := http.NewServeMux()

	// Health check (always on)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// 以降、Usecase が存在するものだけマウントする
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

	if deps.BillingAddressUC != nil {
		mux.Handle("/billing-addresses/", handlers.NewBillingAddressHandler(deps.BillingAddressUC))
	}

	if deps.BrandUC != nil {
		mux.Handle("/brands/", handlers.NewBrandHandler(deps.BrandUC))
	}

	if deps.CampaignUC != nil {
		mux.Handle("/campaigns/", handlers.NewCampaignHandler(deps.CampaignUC))
	}

	if deps.CompanyUC != nil {
		mux.Handle("/companies/", handlers.NewCompanyHandler(deps.CompanyUC))
	}

	if deps.DiscountUC != nil {
		mux.Handle("/discounts/", handlers.NewDiscountHandler(deps.DiscountUC))
	}

	if deps.FulfillmentUC != nil {
		mux.Handle("/fulfillments/", handlers.NewFulfillmentHandler(deps.FulfillmentUC))
	}

	if deps.InquiryUC != nil {
		mux.Handle("/inquiries/", handlers.NewInquiryHandler(deps.InquiryUC))
	}

	if deps.InventoryUC != nil {
		mux.Handle("/inventories/", handlers.NewInventoryHandler(deps.InventoryUC))
	}

	if deps.InvoiceUC != nil {
		mux.Handle("/invoices/", handlers.NewInvoiceHandler(deps.InvoiceUC))
	}

	if deps.ListUC != nil {
		mux.Handle("/lists/", handlers.NewListHandler(deps.ListUC))
	}

	if deps.MemberUC != nil {
		mux.Handle("/members/", handlers.NewMemberHandler(deps.MemberUC))
	}

	// ←★ここが今回のエラー箇所
	// MessageHandler は2つ必要:
	//   1. MessageUsecase
	//   2. Repository (threadsや低レベル操作用)
	if deps.MessageUC != nil && deps.MessageRepo != nil {
		mux.Handle("/messages/", handlers.NewMessageHandler(deps.MessageUC, deps.MessageRepo))
	}

	if deps.MintRequestUC != nil {
		mux.Handle("/mint-requests/", handlers.NewMintRequestHandler(deps.MintRequestUC))
	}

	if deps.ModelUC != nil {
		mux.Handle("/models/", handlers.NewModelHandler(deps.ModelUC))
	}

	if deps.OrderUC != nil {
		mux.Handle("/orders/", handlers.NewOrderHandler(deps.OrderUC))
	}

	if deps.PaymentUC != nil {
		mux.Handle("/payments/", handlers.NewPaymentHandler(deps.PaymentUC))
	}

	if deps.ProductUC != nil {
		mux.Handle("/products/", handlers.NewProductHandler(deps.ProductUC))
	}

	if deps.ProductionUC != nil {
		mux.Handle("/productions/", handlers.NewProductionHandler(deps.ProductionUC))
	}

	if deps.ProductBlueprintUC != nil {
		mux.Handle("/product-blueprints/", handlers.NewProductBlueprintHandler(deps.ProductBlueprintUC))
	}

	if deps.SaleUC != nil {
		mux.Handle("/sales/", handlers.NewSaleHandler(deps.SaleUC))
	}

	if deps.ShippingAddressUC != nil {
		mux.Handle("/shipping-addresses/", handlers.NewShippingAddressHandler(deps.ShippingAddressUC))
	}

	if deps.TokenUC != nil {
		mux.Handle("/tokens/", handlers.NewTokenHandler(deps.TokenUC))
	}

	if deps.TokenBlueprintUC != nil {
		mux.Handle("/token-blueprints/", handlers.NewTokenBlueprintHandler(deps.TokenBlueprintUC))
	}

	if deps.TokenOperationUC != nil {
		mux.Handle("/token-operations/", handlers.NewTokenOperationHandler(deps.TokenOperationUC))
	}

	if deps.TrackingUC != nil {
		mux.Handle("/trackings/", handlers.NewTrackingHandler(deps.TrackingUC))
	}

	if deps.UserUC != nil {
		mux.Handle("/users/", handlers.NewUserHandler(deps.UserUC))
	}

	if deps.WalletUC != nil {
		mux.Handle("/wallets/", handlers.NewWalletHandler(deps.WalletUC))
	}

	return mux
}
