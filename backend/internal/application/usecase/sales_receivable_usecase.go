// backend/internal/application/usecase/sales_receivable_usecase.go
package usecase

import (
	"context"
	"errors"
	"time"

	salesreceivabledom "narratives/internal/domain/salesReceivable"
)

var (
	ErrSalesReceivableRepositoryMissing = errors.New("salesReceivable: repository is not configured")
	ErrSalesReceivableClockMissing      = errors.New("salesReceivable: clock is not configured")
	ErrSalesReceivableExistingMismatch  = errors.New("salesReceivable: existing receivable does not match requested allocation")
	ErrSalesReceivableNotAvailable      = errors.New("salesReceivable: receivable cannot become available")
	ErrSalesReceivableCannotCancel      = errors.New("salesReceivable: receivable cannot be canceled")
	ErrSalesReceivableInvalidLimit      = errors.New("salesReceivable: invalid list limit")
)

// EnsureSalesReceivableInput represents one item-level resale allocation created
// after a successful buyer payment.
//
// One resale Order item must create exactly one SalesReceivable. Multiple resale
// items belonging to the same seller must remain independent receivables.
//
// The resulting document ID is deterministic:
//
//	PaymentID + "_resale_item_" + OrderItemIndex
//
// This operation is idempotent. If the deterministic document already exists,
// the existing entity is returned only when its immutable item allocation
// matches exactly.
type EnsureSalesReceivableInput struct {
	OrderID        string
	PaymentID      string
	OrderItemIndex int
	ResaleID       string

	AvatarID        string
	UserID          string
	PayoutAccountID string

	GrossAmount       int
	PlatformFeeAmount int
	ReceivableAmount  int

	Currency string
}

// SalesReceivableUsecase manages item-level resale proceeds owed by AMOL to Mall
// resale sellers.
//
// Responsibilities:
//   - create one pending receivable for each successfully paid resale Order item
//   - provide authoritative receivable reads
//   - make a pending receivable available after that exact item's fulfillment
//   - cancel an unpaid pending/available receivable
//
// Bank payout reservation and completion intentionally do not belong here.
// Those operations must later be coordinated with BankPayout creation and
// completion through a shared transactional boundary.
type SalesReceivableUsecase struct {
	repo salesreceivabledom.Repository
	now  func() time.Time
}

func NewSalesReceivableUsecase(repo salesreceivabledom.Repository) *SalesReceivableUsecase {
	return &SalesReceivableUsecase{
		repo: repo,
		now:  time.Now,
	}
}

// GetByID returns one persisted SalesReceivable.
func (u *SalesReceivableUsecase) GetByID(
	ctx context.Context,
	receivableID string,
) (*salesreceivabledom.SalesReceivable, error) {
	if err := u.validateRepositoryReady(); err != nil {
		return nil, err
	}
	if receivableID == "" {
		return nil, salesreceivabledom.ErrInvalidID
	}

	receivable, err := u.repo.GetByID(ctx, receivableID)
	if err != nil {
		return nil, err
	}
	if receivable.ID != receivableID {
		return nil, salesreceivabledom.ErrInvalidID
	}
	if err := receivable.Validate(); err != nil {
		return nil, err
	}

	return &receivable, nil
}

// ListByPaymentID returns every item-level resale receivable associated with one
// Payment.
func (u *SalesReceivableUsecase) ListByPaymentID(
	ctx context.Context,
	paymentID string,
) ([]salesreceivabledom.SalesReceivable, error) {
	if err := u.validateRepositoryReady(); err != nil {
		return nil, err
	}
	if paymentID == "" {
		return nil, salesreceivabledom.ErrInvalidPaymentID
	}

	return u.repo.ListByPaymentID(ctx, paymentID)
}

// ListByOrderID returns every item-level resale receivable associated with one
// Order.
func (u *SalesReceivableUsecase) ListByOrderID(
	ctx context.Context,
	orderID string,
) ([]salesreceivabledom.SalesReceivable, error) {
	if err := u.validateRepositoryReady(); err != nil {
		return nil, err
	}
	if orderID == "" {
		return nil, salesreceivabledom.ErrInvalidOrderID
	}

	return u.repo.ListByOrderID(ctx, orderID)
}

// ListByAvatarID returns item-level receivables belonging to one resale seller
// Avatar.
func (u *SalesReceivableUsecase) ListByAvatarID(
	ctx context.Context,
	avatarID string,
) ([]salesreceivabledom.SalesReceivable, error) {
	if err := u.validateRepositoryReady(); err != nil {
		return nil, err
	}
	if avatarID == "" {
		return nil, salesreceivabledom.ErrInvalidAvatarID
	}

	return u.repo.ListByAvatarID(ctx, avatarID)
}

// ListByUserID returns item-level receivables belonging to one Mall user.
func (u *SalesReceivableUsecase) ListByUserID(
	ctx context.Context,
	userID string,
) ([]salesreceivabledom.SalesReceivable, error) {
	if err := u.validateRepositoryReady(); err != nil {
		return nil, err
	}
	if userID == "" {
		return nil, salesreceivabledom.ErrInvalidUserID
	}

	return u.repo.ListByUserID(ctx, userID)
}

// ListByPayoutAccountID returns item-level receivables associated with one
// payout-account identity.
func (u *SalesReceivableUsecase) ListByPayoutAccountID(
	ctx context.Context,
	payoutAccountID string,
) ([]salesreceivabledom.SalesReceivable, error) {
	if err := u.validateRepositoryReady(); err != nil {
		return nil, err
	}
	if payoutAccountID == "" {
		return nil, salesreceivabledom.ErrInvalidPayoutAccountID
	}

	return u.repo.ListByPayoutAccountID(ctx, payoutAccountID)
}

// ListByBankPayoutID returns receivables already assigned to one BankPayout.
func (u *SalesReceivableUsecase) ListByBankPayoutID(
	ctx context.Context,
	bankPayoutID string,
) ([]salesreceivabledom.SalesReceivable, error) {
	if err := u.validateRepositoryReady(); err != nil {
		return nil, err
	}
	if bankPayoutID == "" {
		return nil, salesreceivabledom.ErrInvalidBankPayoutID
	}

	return u.repo.ListByBankPayoutID(ctx, bankPayoutID)
}

// ListAvailableByUserID returns item-level receivables eligible for inclusion in
// a future BankPayout.
//
// This only reads candidates. It must not reserve them.
func (u *SalesReceivableUsecase) ListAvailableByUserID(
	ctx context.Context,
	userID string,
	limit int,
) ([]salesreceivabledom.SalesReceivable, error) {
	if err := u.validateRepositoryReady(); err != nil {
		return nil, err
	}
	if userID == "" {
		return nil, salesreceivabledom.ErrInvalidUserID
	}
	if limit <= 0 {
		return nil, ErrSalesReceivableInvalidLimit
	}

	return u.repo.ListAvailableByUserID(
		ctx,
		salesreceivabledom.ListAvailableByUserIDInput{
			UserID: userID,
			Limit:  limit,
		},
	)
}

// EnsurePending creates the pending SalesReceivable corresponding to one
// successfully paid resale Order item.
//
// The deterministic PaymentID + OrderItemIndex ID makes the operation safe to
// retry after payment webhook or application retries.
//
// If the document already exists and its immutable item allocation is identical,
// the existing entity is returned even if it has subsequently advanced to
// available, reserved, paid, or canceled. This prevents a repeated payment
// success event from recreating or rewinding financial state.
func (u *SalesReceivableUsecase) EnsurePending(
	ctx context.Context,
	in EnsureSalesReceivableInput,
) (*salesreceivabledom.SalesReceivable, error) {
	if err := u.validateWriteReady(); err != nil {
		return nil, err
	}

	receivableID, err := salesreceivabledom.NewID(
		in.PaymentID,
		in.OrderItemIndex,
	)
	if err != nil {
		return nil, err
	}

	now := u.now().UTC()

	receivable, err := salesreceivabledom.New(
		receivableID,
		in.OrderID,
		in.PaymentID,
		in.OrderItemIndex,
		in.ResaleID,
		in.AvatarID,
		in.UserID,
		in.PayoutAccountID,
		in.GrossAmount,
		in.PlatformFeeAmount,
		in.ReceivableAmount,
		in.Currency,
		now,
	)
	if err != nil {
		return nil, err
	}

	created, err := u.repo.Create(ctx, receivable)
	if err == nil {
		if err := validateCreatedSalesReceivable(created, receivable); err != nil {
			return nil, err
		}
		return &created, nil
	}
	if !errors.Is(err, salesreceivabledom.ErrConflict) {
		return nil, err
	}

	existing, getErr := u.repo.GetByID(ctx, receivableID)
	if getErr != nil {
		return nil, getErr
	}
	if err := validateExistingSalesReceivableAllocation(existing, receivable); err != nil {
		return nil, err
	}

	return &existing, nil
}

// MarkAvailable transitions a pending item-level resale receivable to available
// after its exact resale Order item has crossed the fulfillment boundary.
//
// Repeated execution is idempotent once the receivable has reached available or
// a later payout state. A canceled receivable cannot be reopened.
func (u *SalesReceivableUsecase) MarkAvailable(
	ctx context.Context,
	receivableID string,
) (*salesreceivabledom.SalesReceivable, error) {
	if err := u.validateWriteReady(); err != nil {
		return nil, err
	}

	current, err := u.repo.GetByID(ctx, receivableID)
	if err != nil {
		return nil, err
	}

	switch current.Status {
	case salesreceivabledom.StatusPending:
		if err := current.MarkAvailable(u.now().UTC()); err != nil {
			return nil, err
		}

		updated, err := u.repo.Update(ctx, current)
		if err != nil {
			return nil, err
		}
		if err := updated.Validate(); err != nil {
			return nil, err
		}

		return &updated, nil

	case salesreceivabledom.StatusAvailable,
		salesreceivabledom.StatusReserved,
		salesreceivabledom.StatusPaid:
		if err := current.Validate(); err != nil {
			return nil, err
		}
		return &current, nil

	case salesreceivabledom.StatusCanceled:
		return nil, ErrSalesReceivableNotAvailable

	default:
		return nil, salesreceivabledom.ErrInvalidStatus
	}
}

// Cancel cancels a receivable that has not been assigned to a BankPayout.
//
// pending and available may become canceled.
//
// A repeated cancellation is idempotent. reserved and paid cannot be canceled
// through this usecase because payout coordination is required first.
func (u *SalesReceivableUsecase) Cancel(
	ctx context.Context,
	receivableID string,
) (*salesreceivabledom.SalesReceivable, error) {
	if err := u.validateWriteReady(); err != nil {
		return nil, err
	}

	current, err := u.repo.GetByID(ctx, receivableID)
	if err != nil {
		return nil, err
	}

	switch current.Status {
	case salesreceivabledom.StatusPending,
		salesreceivabledom.StatusAvailable:
		if err := current.Cancel(u.now().UTC()); err != nil {
			return nil, err
		}

		updated, err := u.repo.Update(ctx, current)
		if err != nil {
			return nil, err
		}
		if err := updated.Validate(); err != nil {
			return nil, err
		}

		return &updated, nil

	case salesreceivabledom.StatusCanceled:
		if err := current.Validate(); err != nil {
			return nil, err
		}
		return &current, nil

	case salesreceivabledom.StatusReserved,
		salesreceivabledom.StatusPaid:
		return nil, ErrSalesReceivableCannotCancel

	default:
		return nil, salesreceivabledom.ErrInvalidStatus
	}
}

func validateCreatedSalesReceivable(
	actual salesreceivabledom.SalesReceivable,
	expected salesreceivabledom.SalesReceivable,
) error {
	if err := actual.Validate(); err != nil {
		return err
	}
	if actual.Status != salesreceivabledom.StatusPending {
		return ErrSalesReceivableExistingMismatch
	}

	return validateExistingSalesReceivableAllocation(actual, expected)
}

func validateExistingSalesReceivableAllocation(
	actual salesreceivabledom.SalesReceivable,
	expected salesreceivabledom.SalesReceivable,
) error {
	if err := actual.Validate(); err != nil {
		return err
	}
	if err := expected.Validate(); err != nil {
		return err
	}

	if actual.ID != expected.ID ||
		actual.OrderID != expected.OrderID ||
		actual.PaymentID != expected.PaymentID ||
		actual.OrderItemIndex != expected.OrderItemIndex ||
		actual.ResaleID != expected.ResaleID ||
		actual.AvatarID != expected.AvatarID ||
		actual.UserID != expected.UserID ||
		actual.PayoutAccountID != expected.PayoutAccountID ||
		actual.GrossAmount != expected.GrossAmount ||
		actual.PlatformFeeAmount != expected.PlatformFeeAmount ||
		actual.ReceivableAmount != expected.ReceivableAmount ||
		actual.Currency != expected.Currency {
		return ErrSalesReceivableExistingMismatch
	}

	expectedID, err := salesreceivabledom.NewID(
		actual.PaymentID,
		actual.OrderItemIndex,
	)
	if err != nil {
		return err
	}
	if actual.ID != expectedID {
		return ErrSalesReceivableExistingMismatch
	}

	return nil
}

func (u *SalesReceivableUsecase) validateRepositoryReady() error {
	if u == nil || u.repo == nil {
		return ErrSalesReceivableRepositoryMissing
	}
	return nil
}

func (u *SalesReceivableUsecase) validateWriteReady() error {
	if err := u.validateRepositoryReady(); err != nil {
		return err
	}
	if u.now == nil {
		return ErrSalesReceivableClockMissing
	}
	return nil
}
