// backend/internal/application/usecase/order_usecase.go
package usecase

import (
	"context"
	"time"

	applicationport "narratives/internal/application/port"
	accountdom "narratives/internal/domain/account"
	avatardom "narratives/internal/domain/avatar"
	branddom "narratives/internal/domain/brand"
	cartdom "narratives/internal/domain/cart"
	inventorydom "narratives/internal/domain/inventory"
	listdom "narratives/internal/domain/list"
	orderdom "narratives/internal/domain/order"
	paymentmethoddom "narratives/internal/domain/paymentMethod"
	productblueprintdom "narratives/internal/domain/productBlueprint"
	resaledom "narratives/internal/domain/resale"
	shippingaddressdom "narratives/internal/domain/shippingAddress"
)

type OrderConfirmationMailerPort interface {
	SendOrderConfirmation(
		ctx context.Context,
		from string,
		to string,
		order orderdom.Order,
	) error
}

type OrderCancellationMailerPort interface {
	SendOrderCancellationReceipt(
		ctx context.Context,
		toEmail string,
		orderID string,
		itemIndex int,
	) error
}

type ResaleOrderNotificationMailerPort interface {
	SendResaleOrderNotification(
		ctx context.Context,
		toEmail string,
		orderID string,
		itemIndex int,
		resaleID string,
		price int,
	) error

	SendResaleOrderCancellationNotification(
		ctx context.Context,
		toEmail string,
		orderID string,
		itemIndex int,
		resaleID string,
	) error
}

// OrderUsecase orchestrates order operations.
//
// - /mall/me/orders は Order の取得・作成を担当する
// - Invoice の作成は /mall/me/invoices の責務
// - Payment の作成は /mall/me/payments の責務
// - Trade は resale Order 起票時に作成する
// - ReturnItem は返品申請のみを記録し、返金実行は担当しない
// - Stripe Refund / Transfer Reversal は RefundUsecase の責務
type OrderUsecase struct {
	repo                 orderdom.Repository
	cartRepo             cartdom.Repository
	listRepo             listdom.Repository
	inventoryRepo        inventorydom.RepositoryPort
	productBlueprintRepo productblueprintdom.Repository
	brandRepo            branddom.Repository
	accountRepo          accountdom.Repository
	resaleRepo           resaledom.Repository
	avatarRepo           avatardom.Repository
	payoutAccountUC      *PayoutAccountUsecase
	paymentMethodRepo    paymentmethoddom.RepositoryPort
	shippingAddressRepo  shippingaddressdom.RepositoryPort
	shippingQuoteUC      *ShippingQuoteUsecase
	tradeUC              *TradeUsecase

	authUserReader                applicationport.AuthUserReader
	orderConfirmationMailer       OrderConfirmationMailerPort
	orderConfirmationMailFrom     string
	cancellationMailer            OrderCancellationMailerPort
	resaleOrderNotificationMailer ResaleOrderNotificationMailerPort

	now func() time.Time
}

func NewOrderUsecase(
	repo orderdom.Repository,
	listRepo listdom.Repository,
	inventoryRepo inventorydom.RepositoryPort,
	productBlueprintRepo productblueprintdom.Repository,
	resaleRepo resaledom.Repository,
	paymentMethodRepo paymentmethoddom.RepositoryPort,
	shippingAddressRepo shippingaddressdom.RepositoryPort,
	shippingQuoteUC *ShippingQuoteUsecase,
) *OrderUsecase {
	return &OrderUsecase{
		repo:                 repo,
		listRepo:             listRepo,
		inventoryRepo:        inventoryRepo,
		productBlueprintRepo: productBlueprintRepo,
		resaleRepo:           resaleRepo,
		paymentMethodRepo:    paymentMethodRepo,
		shippingAddressRepo:  shippingAddressRepo,
		shippingQuoteUC:      shippingQuoteUC,
		now:                  time.Now,
	}
}

func (u *OrderUsecase) WithCartRepository(
	cartRepo cartdom.Repository,
) *OrderUsecase {
	if u == nil {
		return u
	}

	u.cartRepo = cartRepo
	return u
}

func (u *OrderUsecase) WithSellerRepositories(
	brandRepo branddom.Repository,
	accountRepo accountdom.Repository,
) *OrderUsecase {
	if u == nil {
		return u
	}

	u.brandRepo = brandRepo
	u.accountRepo = accountRepo
	return u
}

func (u *OrderUsecase) WithResaleSellerRepositories(
	avatarRepo avatardom.Repository,
	payoutAccountUC *PayoutAccountUsecase,
) *OrderUsecase {
	if u == nil {
		return u
	}

	u.avatarRepo = avatarRepo
	u.payoutAccountUC = payoutAccountUC
	return u
}

func (u *OrderUsecase) WithTradeUsecase(
	tradeUC *TradeUsecase,
) *OrderUsecase {
	if u == nil {
		return u
	}

	u.tradeUC = tradeUC
	return u
}

func (u *OrderUsecase) WithOrderConfirmationNotification(
	authUserReader applicationport.AuthUserReader,
	mailer OrderConfirmationMailerPort,
	from string,
) *OrderUsecase {
	if u == nil {
		return u
	}

	u.authUserReader = authUserReader
	u.orderConfirmationMailer = mailer
	u.orderConfirmationMailFrom = from
	return u
}

func (u *OrderUsecase) WithCancellationNotification(
	authUserReader applicationport.AuthUserReader,
	mailer OrderCancellationMailerPort,
) *OrderUsecase {
	if u == nil {
		return u
	}

	u.authUserReader = authUserReader
	u.cancellationMailer = mailer
	return u
}

func (u *OrderUsecase) WithResaleOrderNotification(
	authUserReader applicationport.AuthUserReader,
	mailer ResaleOrderNotificationMailerPort,
) *OrderUsecase {
	if u == nil {
		return u
	}

	u.authUserReader = authUserReader
	u.resaleOrderNotificationMailer = mailer
	return u
}
