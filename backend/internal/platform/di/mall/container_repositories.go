// backend/internal/platform/di/mall/container_repositories.go
package mall

import (
	"cloud.google.com/go/firestore"

	fs "narratives/internal/adapters/out/firestore"
)

type mallRepositories struct {
	accountRepo                      *fs.AccountRepositoryFS
	announcementRepo                 *fs.AnnouncementRepositoryFS
	announcementAvatarRepo           *fs.AnnouncementAvatarRepositoryFS
	announcementAttachmentRepo       *fs.AnnouncementAttachmentRepositoryFS
	avatarRepo                       *fs.AvatarRepositoryFS
	avatarReviewRepo                 *fs.AvatarReviewRepositoryFS
	bankPayoutRepo                   *fs.BankPayoutRepositoryFS
	brandRepo                        *fs.BrandRepositoryFS
	brandFeeSettlementRepo           *fs.BrandFeeSettlementRepositoryFS
	cartRepo                         *fs.CartRepositoryFS
	companyRepo                      *fs.CompanyRepositoryFS
	inquiryRepo                      *fs.InquiryRepositoryFS
	inquiryReplyRepo                 *fs.InquiryReplyRepositoryFS
	inventoryRepo                    *fs.InventoryRepositoryFS
	likeRepo                         *fs.LikeRepositoryFS
	listRepoFS                       *fs.ListRepositoryFS
	listImageRecordRepo              *fs.ListImageRepositoryFS
	memberRepo                       *fs.MemberRepositoryFS
	modelRepoFS                      *fs.ModelRepositoryFS
	orderRepo                        *fs.OrderRepositoryFS
	orderTransferItemRepo            *fs.OrderRepoForTransferFS
	paymentRepo                      *fs.PaymentRepositoryFS
	paymentMethodRepo                *fs.PaymentMethodRepositoryFS
	payoutAccountRepo                *fs.PayoutAccountRepositoryFS
	productRepo                      *fs.ProductRepositoryFS
	productBlueprintRepoFS           *fs.ProductBlueprintRepositoryFS
	productBlueprintReviewRepo       *fs.ProductBlueprintReviewRepositoryFS
	refundRepo                       *fs.RefundRepositoryFS
	refundCompletionNotificationRepo *fs.RefundCompletionNotificationRepositoryFS
	resalePayoutNotificationRepo     *fs.ResalePayoutNotificationRepositoryFS
	resaleRepo                       *fs.ResaleRepositoryFS
	resaleImageRepo                  *fs.ResaleImageRepositoryFS
	resaleReviewRepo                 *fs.ResaleReviewRepositoryFS
	salesReceivableRepo              *fs.SalesReceivableRepositoryFS
	settlementRepo                   *fs.SettlementRepositoryFS
	shippingAddressRepo              *fs.ShippingAddressRepositoryFS
	tokenBlueprintRepo               *fs.TokenBlueprintRepositoryFS
	tokenBlueprintReviewRepo         *fs.TokenBlueprintReviewRepositoryFS
	tokenOwnerUpdater                *fs.TokenOwnerUpdaterFS
	tokenReader                      *fs.TokenReaderFS
	tradeRepo                        *fs.TradeRepositoryFS
	tradeMessageRepo                 *fs.TradeMessageRepositoryFS
	transferRepo                     *fs.TransferRepositoryFS
	transportationRepo               *fs.TransportationRepositoryFS
	userRepo                         *fs.UserRepositoryFS
	walletRepo                       *fs.WalletRepositoryFS
	walletResolverRepo               *fs.WalletResolverRepoFS
}

func buildMallRepositories(fsClient *firestore.Client) *mallRepositories {
	avatarRepo := fs.NewAvatarRepositoryFS(fsClient)
	avatarReviewRepo := fs.NewAvatarReviewRepositoryFS(fsClient)
	shippingAddressRepo := fs.NewShippingAddressRepositoryFS(fsClient)
	paymentMethodRepo := fs.NewPaymentMethodRepositoryFS(fsClient)
	payoutAccountRepo := fs.NewPayoutAccountRepositoryFS(fsClient)
	bankPayoutRepo := fs.NewBankPayoutRepositoryFS(fsClient)
	userRepo := fs.NewUserRepositoryFS(fsClient)
	memberRepo := fs.NewMemberRepositoryFS(fsClient)
	walletRepo := fs.NewWalletRepositoryFS(fsClient)
	productRepo := fs.NewProductRepositoryFS(fsClient)
	brandRepo := fs.NewBrandRepositoryFS(fsClient)
	brandFeeSettlementRepo := fs.NewBrandFeeSettlementRepositoryFS(fsClient)
	accountRepo := fs.NewAccountRepositoryFS(fsClient)
	companyRepo := fs.NewCompanyRepositoryFS(fsClient)
	cartRepo := fs.NewCartRepositoryFS(fsClient)
	paymentRepo := fs.NewPaymentRepositoryFS(fsClient)
	settlementRepo := fs.NewSettlementRepositoryFS(fsClient)
	salesReceivableRepo := fs.NewSalesReceivableRepositoryFS(fsClient)
	refundRepo := fs.NewRefundRepositoryFS(fsClient)
	orderRepo := fs.NewOrderRepositoryFS(fsClient)
	orderTransferItemRepo := fs.NewOrderRepoForTransferFS(fsClient)
	inventoryRepo := fs.NewInventoryRepositoryFS(fsClient)
	tokenBlueprintRepo := fs.NewTokenBlueprintRepositoryFS(fsClient)
	productBlueprintRepoFS := fs.NewProductBlueprintRepositoryFS(fsClient)
	modelRepoFS := fs.NewModelRepositoryFS(fsClient)
	tokenReader := fs.NewTokenReaderFS(fsClient)
	inquiryRepo := fs.NewInquiryRepositoryFS(fsClient)
	inquiryReplyRepo := fs.NewInquiryReplyRepositoryFS(fsClient)
	announcementRepo := fs.NewAnnouncementRepositoryFS(fsClient)
	announcementAvatarRepo := fs.NewAnnouncementAvatarRepositoryFS(fsClient)
	announcementAttachmentRepo := fs.NewAnnouncementAttachmentRepositoryFS(fsClient)
	tokenBlueprintReviewRepo := fs.NewTokenBlueprintReviewRepositoryFS(fsClient)
	productBlueprintReviewRepo := fs.NewProductBlueprintReviewRepositoryFS(fsClient)
	likeRepo := fs.NewLikeRepositoryFS(fsClient)
	listRepoFS := fs.NewListRepositoryFS(fsClient)
	listImageRecordRepo := fs.NewListImageRepositoryFS(fsClient)
	resaleRepo := fs.NewResaleRepositoryFS(fsClient)
	resaleImageRepo := fs.NewResaleImageRepositoryFS(fsClient)
	resaleReviewRepo := fs.NewResaleReviewRepositoryFS(fsClient)
	transportationRepo := fs.NewTransportationRepositoryFS(fsClient)
	tokenOwnerUpdater := fs.NewTokenOwnerUpdaterFS(fsClient)
	tradeRepo := fs.NewTradeRepositoryFS(fsClient)
	tradeMessageRepo := fs.NewTradeMessageRepositoryFS(fsClient)
	transferRepo := fs.NewTransferRepositoryFS(fsClient)
	refundCompletionNotificationRepo := fs.NewRefundCompletionNotificationRepositoryFS(fsClient)
	resalePayoutNotificationRepo := fs.NewResalePayoutNotificationRepositoryFS(fsClient)

	walletResolverRepo := fs.NewWalletResolverRepoFS(
		brandRepo,
		walletRepo,
	)

	return &mallRepositories{
		accountRepo:                      accountRepo,
		announcementRepo:                 announcementRepo,
		announcementAvatarRepo:           announcementAvatarRepo,
		announcementAttachmentRepo:       announcementAttachmentRepo,
		avatarRepo:                       avatarRepo,
		avatarReviewRepo:                 avatarReviewRepo,
		bankPayoutRepo:                   bankPayoutRepo,
		brandRepo:                        brandRepo,
		brandFeeSettlementRepo:           brandFeeSettlementRepo,
		cartRepo:                         cartRepo,
		companyRepo:                      companyRepo,
		inquiryRepo:                      inquiryRepo,
		inquiryReplyRepo:                 inquiryReplyRepo,
		inventoryRepo:                    inventoryRepo,
		likeRepo:                         likeRepo,
		listRepoFS:                       listRepoFS,
		listImageRecordRepo:              listImageRecordRepo,
		memberRepo:                       memberRepo,
		modelRepoFS:                      modelRepoFS,
		orderRepo:                        orderRepo,
		orderTransferItemRepo:            orderTransferItemRepo,
		paymentRepo:                      paymentRepo,
		paymentMethodRepo:                paymentMethodRepo,
		payoutAccountRepo:                payoutAccountRepo,
		productRepo:                      productRepo,
		productBlueprintRepoFS:           productBlueprintRepoFS,
		productBlueprintReviewRepo:       productBlueprintReviewRepo,
		refundRepo:                       refundRepo,
		refundCompletionNotificationRepo: refundCompletionNotificationRepo,
		resalePayoutNotificationRepo:     resalePayoutNotificationRepo,
		resaleRepo:                       resaleRepo,
		resaleImageRepo:                  resaleImageRepo,
		resaleReviewRepo:                 resaleReviewRepo,
		salesReceivableRepo:              salesReceivableRepo,
		settlementRepo:                   settlementRepo,
		shippingAddressRepo:              shippingAddressRepo,
		tokenBlueprintRepo:               tokenBlueprintRepo,
		tokenBlueprintReviewRepo:         tokenBlueprintReviewRepo,
		tokenOwnerUpdater:                tokenOwnerUpdater,
		tokenReader:                      tokenReader,
		tradeRepo:                        tradeRepo,
		tradeMessageRepo:                 tradeMessageRepo,
		transferRepo:                     transferRepo,
		transportationRepo:               transportationRepo,
		userRepo:                         userRepo,
		walletRepo:                       walletRepo,
		walletResolverRepo:               walletResolverRepo,
	}
}
