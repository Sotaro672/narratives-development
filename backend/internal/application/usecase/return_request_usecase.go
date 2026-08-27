// backend/internal/application/usecase/return_request_usecase.go
package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	inquirydom "narratives/internal/domain/inquiry"
	orderdom "narratives/internal/domain/order"
)

var (
	ErrReturnRequestUsecaseNotConfigured = errors.New(
		"return request usecase: not configured",
	)
)

// ReturnRequestOrderService is the minimum Order application service required
// by ReturnRequestUsecase.
//
// Order state is authoritative for return eligibility and server-observed
// transfer state. A return Inquiry must be confirmed before ReturnItem mutates
// the Order return state.
//
// An OrderDetail flow may explicitly declare an item opened without a prior
// token transfer. Scan-based opened flows still require Transferred or
// TokenTransferVerifiedAt as server-observed evidence.
type ReturnRequestOrderService interface {
	GetByID(
		ctx context.Context,
		id string,
	) (orderdom.Order, error)

	ReturnItem(
		ctx context.Context,
		in ReturnOrderItemInput,
	) (orderdom.Order, error)
}

// ReturnRequestInquiryCreator is the minimum Inquiry application service
// required for creating a return inquiry.
//
// Inquiry creation must go through InquiryUsecase so notification mail and
// other application-level behavior are not bypassed.
type ReturnRequestInquiryCreator interface {
	Create(
		ctx context.Context,
		inq inquirydom.Inquiry,
	) (inquirydom.Inquiry, error)
}

// ReturnRequestUsecase coordinates a return Inquiry and the corresponding
// Order return state.
//
// Responsibilities:
// - OrderDetail unopened return -> return_unopened
// - OrderDetail opened declaration -> return_opened
// - scanned/opened return -> return_opened
// - valid scan during return_unopened -> promote Inquiry, then persist opened kind
// - prevent duplicate return inquiries for the same Order item
// - never mark an Order item as return-requested before the return Inquiry exists
//
// Financial operations are intentionally outside this usecase.
// Stripe Refund / Transfer Reversal remain the responsibility of RefundUsecase.
type ReturnRequestUsecase struct {
	orderService   ReturnRequestOrderService
	inquiryRepo    inquirydom.Repository
	inquiryCreator ReturnRequestInquiryCreator

	now func() time.Time
}

func NewReturnRequestUsecase(
	orderService ReturnRequestOrderService,
	inquiryRepo inquirydom.Repository,
	inquiryCreator ReturnRequestInquiryCreator,
) *ReturnRequestUsecase {
	return &ReturnRequestUsecase{
		orderService:   orderService,
		inquiryRepo:    inquiryRepo,
		inquiryCreator: inquiryCreator,
		now:            time.Now,
	}
}

// SetNowFunc replaces the current time function for tests.
func (uc *ReturnRequestUsecase) SetNowFunc(
	now func() time.Time,
) {
	if uc == nil || now == nil {
		return
	}

	uc.now = now
}

// ReturnRequestResult is returned from all return request transitions.
//
// InquiryCreated:
// - true only when this call created a new Inquiry.
//
// InquiryPromoted:
//   - true only when this call changed
//     return_unopened -> return_opened.
type ReturnRequestResult struct {
	Order   orderdom.Order
	Inquiry inquirydom.Inquiry

	InquiryCreated  bool
	InquiryPromoted bool
}

// RequestUnopenedReturnInput identifies an unopened return request.
//
// This flow is intended for:
//
//	OrderDetail
//	-> return button
//	-> backend validates unopened state
//	-> create or confirm return_unopened Inquiry
//	-> persist Order return_unopened state
//
// Reason is required for this flow and is stored as Inquiry.Content.
type RequestUnopenedReturnInput struct {
	OrderID   string
	AvatarID  string
	ItemIndex int
	Reason    string
}

// RequestOpenedFromOrderDetailInput identifies an opened return declared by
// the purchaser from OrderDetail.
//
// Unlike RequestOpened, this flow does not require a prior token transfer or
// tokenTransferVerifiedAt. The opened state is an explicit purchaser
// declaration, and Reason is required.
type RequestOpenedFromOrderDetailInput struct {
	OrderID   string
	AvatarID  string
	ItemIndex int
	Reason    string
}

// RequestOpenedReturnInput identifies an opened return request.
//
// This flow is intended for:
//
//	valid product scan
//	-> token transfer verified
//	-> InquiryCreatePage
//	-> backend validates opened state
//	-> create, confirm, or promote return_opened Inquiry
//	-> persist Order return_opened state
//
// ProductID is required because the scan flow has already identified the
// physical product.
type RequestOpenedReturnInput struct {
	OrderID   string
	AvatarID  string
	ItemIndex int
	ProductID string

	Content string
	Images  []inquirydom.ImageFile
}

// PromoteUnopenedToOpenedInput is used when a valid product scan occurs while
// an unopened return is already in progress.
//
// The token transfer itself must remain blocked. This transition only records
// that the product can no longer be treated as unopened.
type PromoteUnopenedToOpenedInput struct {
	OrderID   string
	AvatarID  string
	ItemIndex int
	ProductID string
}

// RequestUnopened records an unopened return request.
//
// The return Inquiry is created or confirmed before Order.ReturnItem is called.
// If Inquiry persistence fails, the Order return state is not changed.
func (uc *ReturnRequestUsecase) RequestUnopened(
	ctx context.Context,
	in RequestUnopenedReturnInput,
) (ReturnRequestResult, error) {
	if err := uc.validateConfigured(); err != nil {
		return ReturnRequestResult{}, err
	}

	orderID := in.OrderID
	if orderID == "" {
		return ReturnRequestResult{},
			orderdom.ErrInvalidID
	}

	avatarID := in.AvatarID
	if avatarID == "" {
		return ReturnRequestResult{},
			orderdom.ErrInvalidAvatarID
	}

	reason := in.Reason
	if reason == "" {
		return ReturnRequestResult{},
			inquirydom.ErrInvalidContent
	}

	if in.ItemIndex < 0 {
		return ReturnRequestResult{},
			orderdom.ErrInvalidItems
	}

	order, targetItem, err :=
		uc.getReturnTarget(
			ctx,
			orderID,
			avatarID,
			in.ItemIndex,
		)
	if err != nil {
		return ReturnRequestResult{}, err
	}

	if targetItem.IsCancelled ||
		!targetItem.IsDispatched ||
		targetItem.IsReturnCompleted ||
		targetItem.Transferred ||
		targetItem.TokenTransferVerifiedAt != nil {
		return ReturnRequestResult{},
			orderdom.ErrConflict
	}

	inquiry, inquiryCreated, err :=
		uc.ensureUnopenedInquiry(
			ctx,
			orderID,
			avatarID,
			in.ItemIndex,
			targetItem.ProductID,
			reason,
		)
	if err != nil {
		return ReturnRequestResult{
			Order: order,
		}, err
	}

	// Re-read after Inquiry persistence and before mutating Order.
	// A scan may have been verified while the Inquiry was being created.
	latestOrder, latestItem, err :=
		uc.getReturnTarget(
			ctx,
			orderID,
			avatarID,
			in.ItemIndex,
		)
	if err != nil {
		return ReturnRequestResult{
			Order:          order,
			Inquiry:        inquiry,
			InquiryCreated: inquiryCreated,
		}, err
	}

	if latestItem.IsCancelled ||
		!latestItem.IsDispatched ||
		latestItem.IsReturnCompleted {
		return ReturnRequestResult{
			Order:          latestOrder,
			Inquiry:        inquiry,
			InquiryCreated: inquiryCreated,
		}, orderdom.ErrConflict
	}

	desiredKind :=
		orderdom.ReturnRequestKindUnopened
	inquiryPromoted := false

	if isServerObservedOpenedReturnItem(latestItem) {
		promoted, err :=
			uc.promoteInquiryToOpened(
				ctx,
				inquiry,
				latestItem.ProductID,
			)
		if err != nil {
			return ReturnRequestResult{
				Order:          latestOrder,
				Inquiry:        inquiry,
				InquiryCreated: inquiryCreated,
			}, err
		}

		inquiryPromoted =
			inquiry.InquiryType !=
				inquirydom.InquiryTypeReturnOpened
		inquiry = promoted
		desiredKind =
			orderdom.ReturnRequestKindOpened
	}

	updatedOrder, err :=
		uc.orderService.ReturnItem(
			ctx,
			ReturnOrderItemInput{
				ID:        orderID,
				AvatarID:  avatarID,
				ItemIndex: in.ItemIndex,
				Kind:      desiredKind,
			},
		)
	if err != nil {
		return ReturnRequestResult{
			Order:           latestOrder,
			Inquiry:         inquiry,
			InquiryCreated:  inquiryCreated,
			InquiryPromoted: inquiryPromoted,
		}, err
	}

	latestOrder, latestItem, err =
		uc.getReturnTarget(
			ctx,
			orderID,
			avatarID,
			in.ItemIndex,
		)
	if err != nil {
		return ReturnRequestResult{
			Order:           updatedOrder,
			Inquiry:         inquiry,
			InquiryCreated:  inquiryCreated,
			InquiryPromoted: inquiryPromoted,
		}, err
	}

	if latestItem.IsCancelled ||
		!latestItem.IsDispatched ||
		latestItem.IsReturnCompleted ||
		!latestItem.IsReturnRequested ||
		latestItem.ReturnRequestKind != desiredKind {
		return ReturnRequestResult{
			Order:           latestOrder,
			Inquiry:         inquiry,
			InquiryCreated:  inquiryCreated,
			InquiryPromoted: inquiryPromoted,
		}, orderdom.ErrConflict
	}

	return ReturnRequestResult{
		Order:           latestOrder,
		Inquiry:         inquiry,
		InquiryCreated:  inquiryCreated,
		InquiryPromoted: inquiryPromoted,
	}, nil
}

// RequestOpenedFromOrderDetail records an opened return declaration from
// OrderDetail and creates or promotes the associated return_opened Inquiry.
//
// This flow intentionally differs from RequestOpened:
//   - RequestOpened requires server-observed opened state from a valid scan.
//   - RequestOpenedFromOrderDetail accepts the purchaser's explicit opened
//     declaration even when transferred == false and tokenTransferVerifiedAt == nil.
//
// The return_opened Inquiry is confirmed before Order.ReturnItem is called.
// Financial refund operations remain outside this usecase.
func (uc *ReturnRequestUsecase) RequestOpenedFromOrderDetail(
	ctx context.Context,
	in RequestOpenedFromOrderDetailInput,
) (ReturnRequestResult, error) {
	if err := uc.validateConfigured(); err != nil {
		return ReturnRequestResult{}, err
	}

	orderID := in.OrderID
	if orderID == "" {
		return ReturnRequestResult{},
			orderdom.ErrInvalidID
	}

	avatarID := in.AvatarID
	if avatarID == "" {
		return ReturnRequestResult{},
			orderdom.ErrInvalidAvatarID
	}

	reason := in.Reason
	if reason == "" {
		return ReturnRequestResult{},
			inquirydom.ErrInvalidContent
	}

	if in.ItemIndex < 0 {
		return ReturnRequestResult{},
			orderdom.ErrInvalidItems
	}

	order, targetItem, err :=
		uc.getReturnTarget(
			ctx,
			orderID,
			avatarID,
			in.ItemIndex,
		)
	if err != nil {
		return ReturnRequestResult{}, err
	}

	if targetItem.IsCancelled ||
		!targetItem.IsDispatched ||
		targetItem.IsReturnCompleted {
		return ReturnRequestResult{},
			orderdom.ErrConflict
	}

	inquiry,
		inquiryCreated,
		inquiryPromoted,
		err := uc.ensureDeclaredOpenedInquiry(
		ctx,
		orderID,
		avatarID,
		in.ItemIndex,
		targetItem.ProductID,
		reason,
	)
	if err != nil {
		return ReturnRequestResult{
			Order: order,
		}, err
	}

	updatedOrder, err :=
		uc.orderService.ReturnItem(
			ctx,
			ReturnOrderItemInput{
				ID:        orderID,
				AvatarID:  avatarID,
				ItemIndex: in.ItemIndex,
				Kind:      orderdom.ReturnRequestKindOpened,
			},
		)
	if err != nil {
		return ReturnRequestResult{
			Order:           order,
			Inquiry:         inquiry,
			InquiryCreated:  inquiryCreated,
			InquiryPromoted: inquiryPromoted,
		}, err
	}

	latestOrder, latestItem, err :=
		uc.getReturnTarget(
			ctx,
			orderID,
			avatarID,
			in.ItemIndex,
		)
	if err != nil {
		return ReturnRequestResult{
			Order:           updatedOrder,
			Inquiry:         inquiry,
			InquiryCreated:  inquiryCreated,
			InquiryPromoted: inquiryPromoted,
		}, err
	}

	if latestItem.IsCancelled ||
		!latestItem.IsDispatched ||
		latestItem.IsReturnCompleted ||
		!latestItem.IsReturnRequested ||
		latestItem.ReturnRequestKind !=
			orderdom.ReturnRequestKindOpened {
		return ReturnRequestResult{
			Order:           latestOrder,
			Inquiry:         inquiry,
			InquiryCreated:  inquiryCreated,
			InquiryPromoted: inquiryPromoted,
		}, orderdom.ErrConflict
	}

	return ReturnRequestResult{
		Order:           latestOrder,
		Inquiry:         inquiry,
		InquiryCreated:  inquiryCreated,
		InquiryPromoted: inquiryPromoted,
	}, nil
}

// RequestOpened records an opened return request and creates or updates the
// associated return Inquiry.
//
// A scan-based opened return requires server-observed opened evidence:
//
//	Transferred == true
//
// OR:
//
//	TokenTransferVerifiedAt != nil
//
// The return_opened Inquiry is created, confirmed, or promoted before the
// Order return state is persisted.
func (uc *ReturnRequestUsecase) RequestOpened(
	ctx context.Context,
	in RequestOpenedReturnInput,
) (ReturnRequestResult, error) {
	if err := uc.validateConfigured(); err != nil {
		return ReturnRequestResult{}, err
	}

	orderID := in.OrderID
	if orderID == "" {
		return ReturnRequestResult{},
			orderdom.ErrInvalidID
	}

	avatarID := in.AvatarID
	if avatarID == "" {
		return ReturnRequestResult{},
			orderdom.ErrInvalidAvatarID
	}

	productID := in.ProductID
	if productID == "" {
		return ReturnRequestResult{},
			inquirydom.ErrInvalidProductID
	}

	if in.ItemIndex < 0 {
		return ReturnRequestResult{},
			orderdom.ErrInvalidItems
	}

	order, targetItem, err :=
		uc.getReturnTarget(
			ctx,
			orderID,
			avatarID,
			in.ItemIndex,
		)
	if err != nil {
		return ReturnRequestResult{}, err
	}

	if targetItem.IsCancelled ||
		!targetItem.IsDispatched ||
		targetItem.IsReturnCompleted {
		return ReturnRequestResult{},
			orderdom.ErrConflict
	}

	if !isServerObservedOpenedReturnItem(targetItem) {
		return ReturnRequestResult{},
			orderdom.ErrConflict
	}

	if targetItem.ProductID != "" &&
		targetItem.ProductID != productID {
		return ReturnRequestResult{},
			orderdom.ErrNotFound
	}

	inquiry,
		inquiryCreated,
		inquiryPromoted,
		err := uc.ensureScannedOpenedInquiry(
		ctx,
		orderID,
		avatarID,
		in.ItemIndex,
		productID,
		in.Content,
		in.Images,
	)
	if err != nil {
		return ReturnRequestResult{
			Order: order,
		}, err
	}

	updatedOrder, err :=
		uc.orderService.ReturnItem(
			ctx,
			ReturnOrderItemInput{
				ID:        orderID,
				AvatarID:  avatarID,
				ItemIndex: in.ItemIndex,
				Kind:      orderdom.ReturnRequestKindOpened,
			},
		)
	if err != nil {
		return ReturnRequestResult{
			Order:           order,
			Inquiry:         inquiry,
			InquiryCreated:  inquiryCreated,
			InquiryPromoted: inquiryPromoted,
		}, err
	}

	latestOrder, latestItem, err :=
		uc.getReturnTarget(
			ctx,
			orderID,
			avatarID,
			in.ItemIndex,
		)
	if err != nil {
		return ReturnRequestResult{
			Order:           updatedOrder,
			Inquiry:         inquiry,
			InquiryCreated:  inquiryCreated,
			InquiryPromoted: inquiryPromoted,
		}, err
	}

	if latestItem.IsCancelled ||
		!latestItem.IsDispatched ||
		latestItem.IsReturnCompleted ||
		!latestItem.IsReturnRequested ||
		latestItem.ReturnRequestKind !=
			orderdom.ReturnRequestKindOpened {
		return ReturnRequestResult{
			Order:           latestOrder,
			Inquiry:         inquiry,
			InquiryCreated:  inquiryCreated,
			InquiryPromoted: inquiryPromoted,
		}, orderdom.ErrConflict
	}

	return ReturnRequestResult{
		Order:           latestOrder,
		Inquiry:         inquiry,
		InquiryCreated:  inquiryCreated,
		InquiryPromoted: inquiryPromoted,
	}, nil
}

// PromoteUnopenedToOpened changes an existing return_unopened Inquiry to
// return_opened after a valid product scan.
//
// This method does not transfer the token.
//
// Expected flow:
//
//	return_unopened
//	-> valid scan
//	-> tokenTransferVerifiedAt recorded
//	-> promote or confirm return_opened Inquiry
//	-> persist Order opened return kind
//	-> token transfer blocked
func (uc *ReturnRequestUsecase) PromoteUnopenedToOpened(
	ctx context.Context,
	in PromoteUnopenedToOpenedInput,
) (ReturnRequestResult, error) {
	if err := uc.validateConfigured(); err != nil {
		return ReturnRequestResult{}, err
	}

	orderID := in.OrderID
	if orderID == "" {
		return ReturnRequestResult{},
			orderdom.ErrInvalidID
	}

	avatarID := in.AvatarID
	if avatarID == "" {
		return ReturnRequestResult{},
			orderdom.ErrInvalidAvatarID
	}

	productID := in.ProductID
	if productID == "" {
		return ReturnRequestResult{},
			inquirydom.ErrInvalidProductID
	}

	if in.ItemIndex < 0 {
		return ReturnRequestResult{},
			orderdom.ErrInvalidItems
	}

	order, targetItem, err :=
		uc.getReturnTarget(
			ctx,
			orderID,
			avatarID,
			in.ItemIndex,
		)
	if err != nil {
		return ReturnRequestResult{}, err
	}

	if targetItem.IsCancelled ||
		!targetItem.IsDispatched ||
		targetItem.IsReturnCompleted ||
		!targetItem.IsReturnRequested {
		return ReturnRequestResult{},
			orderdom.ErrConflict
	}

	if !isServerObservedOpenedReturnItem(targetItem) {
		return ReturnRequestResult{},
			orderdom.ErrConflict
	}

	if targetItem.ProductID != "" &&
		targetItem.ProductID != productID {
		return ReturnRequestResult{},
			orderdom.ErrNotFound
	}

	existing, found, err :=
		uc.findReturnInquiry(
			ctx,
			avatarID,
			orderID,
			in.ItemIndex,
		)
	if err != nil {
		return ReturnRequestResult{
			Order: order,
		}, err
	}

	if !found {
		return ReturnRequestResult{
			Order: order,
		}, inquirydom.ErrNotFound
	}

	inquiry := existing
	inquiryPromoted := false

	switch existing.InquiryType {
	case inquirydom.InquiryTypeReturnOpened:
		if err := validateReturnInquiryProductID(
			existing,
			productID,
		); err != nil {
			return ReturnRequestResult{
				Order:   order,
				Inquiry: existing,
			}, err
		}

		if existing.ProductID == "" {
			updatedInquiry, err :=
				uc.setReturnInquiryProductID(
					ctx,
					existing,
					productID,
				)
			if err != nil {
				return ReturnRequestResult{
					Order:   order,
					Inquiry: existing,
				}, err
			}

			inquiry = updatedInquiry
		}

	case inquirydom.InquiryTypeReturnUnopened:
		promoted, err :=
			uc.promoteInquiryToOpened(
				ctx,
				existing,
				productID,
			)
		if err != nil {
			return ReturnRequestResult{
				Order:   order,
				Inquiry: existing,
			}, err
		}

		inquiry = promoted
		inquiryPromoted = true

	default:
		return ReturnRequestResult{
			Order:   order,
			Inquiry: existing,
		}, inquirydom.ErrConflict
	}

	updatedOrder, err :=
		uc.orderService.ReturnItem(
			ctx,
			ReturnOrderItemInput{
				ID:        orderID,
				AvatarID:  avatarID,
				ItemIndex: in.ItemIndex,
				Kind:      orderdom.ReturnRequestKindOpened,
			},
		)
	if err != nil {
		return ReturnRequestResult{
			Order:           order,
			Inquiry:         inquiry,
			InquiryPromoted: inquiryPromoted,
		}, err
	}

	latestOrder, latestItem, err :=
		uc.getReturnTarget(
			ctx,
			orderID,
			avatarID,
			in.ItemIndex,
		)
	if err != nil {
		return ReturnRequestResult{
			Order:           updatedOrder,
			Inquiry:         inquiry,
			InquiryPromoted: inquiryPromoted,
		}, err
	}

	if latestItem.IsCancelled ||
		!latestItem.IsDispatched ||
		latestItem.IsReturnCompleted ||
		!latestItem.IsReturnRequested ||
		latestItem.ReturnRequestKind !=
			orderdom.ReturnRequestKindOpened {
		return ReturnRequestResult{
			Order:           latestOrder,
			Inquiry:         inquiry,
			InquiryPromoted: inquiryPromoted,
		}, orderdom.ErrConflict
	}

	return ReturnRequestResult{
		Order:           latestOrder,
		Inquiry:         inquiry,
		InquiryPromoted: inquiryPromoted,
	}, nil
}

func (uc *ReturnRequestUsecase) ensureUnopenedInquiry(
	ctx context.Context,
	orderID string,
	avatarID string,
	itemIndex int,
	productID string,
	reason string,
) (
	inquirydom.Inquiry,
	bool,
	error,
) {
	existing, found, err :=
		uc.findReturnInquiry(
			ctx,
			avatarID,
			orderID,
			itemIndex,
		)
	if err != nil {
		return inquirydom.Inquiry{},
			false,
			err
	}

	if found {
		if existing.InquiryType !=
			inquirydom.InquiryTypeReturnUnopened {
			return inquirydom.Inquiry{},
				false,
				inquirydom.ErrConflict
		}

		return existing,
			false,
			nil
	}

	now := uc.nowUTC()

	inquiry, err :=
		inquirydom.NewReturnUnopened(
			returnInquiryID(
				orderID,
				itemIndex,
			),
			productID,
			orderID,
			itemIndex,
			avatarID,
			reason,
			now,
			now,
		)
	if err != nil {
		return inquirydom.Inquiry{},
			false,
			err
	}

	inquiry.Images =
		[]inquirydom.ImageFile{}

	confirmed, created, err :=
		uc.createAndConfirmReturnInquiry(
			ctx,
			inquiry,
			avatarID,
			orderID,
			itemIndex,
		)
	if err != nil {
		return inquirydom.Inquiry{},
			false,
			err
	}

	if confirmed.InquiryType !=
		inquirydom.InquiryTypeReturnUnopened {
		return inquirydom.Inquiry{},
			false,
			inquirydom.ErrConflict
	}

	return confirmed,
		created,
		nil
}

func (uc *ReturnRequestUsecase) ensureDeclaredOpenedInquiry(
	ctx context.Context,
	orderID string,
	avatarID string,
	itemIndex int,
	productID string,
	reason string,
) (
	inquirydom.Inquiry,
	bool,
	bool,
	error,
) {
	existing, found, err :=
		uc.findReturnInquiry(
			ctx,
			avatarID,
			orderID,
			itemIndex,
		)
	if err != nil {
		return inquirydom.Inquiry{},
			false,
			false,
			err
	}

	if !found {
		now := uc.nowUTC()

		inquiry, err :=
			inquirydom.NewReturnOpened(
				returnInquiryID(
					orderID,
					itemIndex,
				),
				productID,
				orderID,
				itemIndex,
				avatarID,
				reason,
				now,
				now,
			)
		if err != nil {
			return inquirydom.Inquiry{},
				false,
				false,
				err
		}

		inquiry.Images =
			[]inquirydom.ImageFile{}

		confirmed, created, err :=
			uc.createAndConfirmReturnInquiry(
				ctx,
				inquiry,
				avatarID,
				orderID,
				itemIndex,
			)
		if err != nil {
			return inquirydom.Inquiry{},
				false,
				false,
				err
		}

		existing = confirmed
		found = true

		if created &&
			existing.InquiryType ==
				inquirydom.InquiryTypeReturnOpened {
			return existing,
				true,
				false,
				nil
		}
	}

	if !found {
		return inquirydom.Inquiry{},
			false,
			false,
			inquirydom.ErrNotFound
	}

	switch existing.InquiryType {
	case inquirydom.InquiryTypeReturnOpened:
		if productID != "" {
			if err := validateReturnInquiryProductID(
				existing,
				productID,
			); err != nil {
				return inquirydom.Inquiry{},
					false,
					false,
					err
			}

			if existing.ProductID == "" {
				updatedInquiry, err :=
					uc.setReturnInquiryProductID(
						ctx,
						existing,
						productID,
					)
				if err != nil {
					return inquirydom.Inquiry{},
						false,
						false,
						err
				}

				existing = updatedInquiry
			}
		}

		return existing,
			false,
			false,
			nil

	case inquirydom.InquiryTypeReturnUnopened:
		promoted, err :=
			uc.promoteInquiryToOpenedWithContent(
				ctx,
				existing,
				productID,
				reason,
			)
		if err != nil {
			return inquirydom.Inquiry{},
				false,
				false,
				err
		}

		return promoted,
			false,
			true,
			nil

	default:
		return inquirydom.Inquiry{},
			false,
			false,
			inquirydom.ErrConflict
	}
}

func (uc *ReturnRequestUsecase) ensureScannedOpenedInquiry(
	ctx context.Context,
	orderID string,
	avatarID string,
	itemIndex int,
	productID string,
	content string,
	images []inquirydom.ImageFile,
) (
	inquirydom.Inquiry,
	bool,
	bool,
	error,
) {
	existing, found, err :=
		uc.findReturnInquiry(
			ctx,
			avatarID,
			orderID,
			itemIndex,
		)
	if err != nil {
		return inquirydom.Inquiry{},
			false,
			false,
			err
	}

	if !found {
		now := uc.nowUTC()

		inquiry := inquirydom.Inquiry{
			ID: returnInquiryID(
				orderID,
				itemIndex,
			),
			ProductID: productID,
			OrderID:   orderID,
			OrderItemIndex: intPointer(
				itemIndex,
			),
			AvatarID:    avatarID,
			Content:     content,
			Status:      inquirydom.InquiryStatusOpen,
			InquiryType: inquirydom.InquiryTypeReturnOpened,
			IsRead:      false,
			Images:      images,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		confirmed, created, err :=
			uc.createAndConfirmReturnInquiry(
				ctx,
				inquiry,
				avatarID,
				orderID,
				itemIndex,
			)
		if err != nil {
			return inquirydom.Inquiry{},
				false,
				false,
				err
		}

		existing = confirmed
		found = true

		if created &&
			existing.InquiryType ==
				inquirydom.InquiryTypeReturnOpened {
			return existing,
				true,
				false,
				nil
		}
	}

	if !found {
		return inquirydom.Inquiry{},
			false,
			false,
			inquirydom.ErrNotFound
	}

	switch existing.InquiryType {
	case inquirydom.InquiryTypeReturnOpened:
		if err := validateReturnInquiryProductID(
			existing,
			productID,
		); err != nil {
			return inquirydom.Inquiry{},
				false,
				false,
				err
		}

		if existing.ProductID == "" {
			updatedInquiry, err :=
				uc.setReturnInquiryProductID(
					ctx,
					existing,
					productID,
				)
			if err != nil {
				return inquirydom.Inquiry{},
					false,
					false,
					err
			}

			existing = updatedInquiry
		}

		return existing,
			false,
			false,
			nil

	case inquirydom.InquiryTypeReturnUnopened:
		promoted, err :=
			uc.promoteInquiryToOpened(
				ctx,
				existing,
				productID,
			)
		if err != nil {
			return inquirydom.Inquiry{},
				false,
				false,
				err
		}

		return promoted,
			false,
			true,
			nil

	default:
		return inquirydom.Inquiry{},
			false,
			false,
			inquirydom.ErrConflict
	}
}

// createAndConfirmReturnInquiry creates an Inquiry and confirms persistence
// before any Order return-state mutation is allowed.
//
// InquiryUsecase.Create can return a persisted Inquiry together with a
// secondary error, for example a notification-mail failure. A non-empty
// created.ID therefore means the Inquiry itself exists and the return flow may
// continue.
//
// A deterministic Inquiry ID makes concurrent creates converge on the same
// Inquiry. When Create returns ErrConflict, the persisted Inquiry is read back
// and treated as the confirmed result.
func (uc *ReturnRequestUsecase) createAndConfirmReturnInquiry(
	ctx context.Context,
	inquiry inquirydom.Inquiry,
	avatarID string,
	orderID string,
	itemIndex int,
) (
	inquirydom.Inquiry,
	bool,
	error,
) {
	created, err :=
		uc.inquiryCreator.Create(
			ctx,
			inquiry,
		)

	if created.ID != "" {
		return created,
			true,
			nil
	}

	if err != nil &&
		errors.Is(
			err,
			inquirydom.ErrConflict,
		) {
		existing, found, findErr :=
			uc.findReturnInquiry(
				ctx,
				avatarID,
				orderID,
				itemIndex,
			)
		if findErr != nil {
			return inquirydom.Inquiry{},
				false,
				findErr
		}

		if found {
			return existing,
				false,
				nil
		}
	}

	if err != nil {
		return inquirydom.Inquiry{},
			false,
			err
	}

	return inquirydom.Inquiry{},
		false,
		fmt.Errorf(
			"return request usecase: inquiry was not persisted",
		)
}

func (uc *ReturnRequestUsecase) getReturnTarget(
	ctx context.Context,
	orderID string,
	avatarID string,
	itemIndex int,
) (
	orderdom.Order,
	orderdom.OrderItemSnapshot,
	error,
) {
	order, err :=
		uc.orderService.GetByID(
			ctx,
			orderID,
		)
	if err != nil {
		return orderdom.Order{},
			orderdom.OrderItemSnapshot{},
			err
	}

	if order.AvatarID != avatarID {
		return orderdom.Order{},
			orderdom.OrderItemSnapshot{},
			orderdom.ErrNotFound
	}

	if itemIndex < 0 ||
		itemIndex >= len(order.Items) {
		return orderdom.Order{},
			orderdom.OrderItemSnapshot{},
			orderdom.ErrNotFound
	}

	return order,
		order.Items[itemIndex],
		nil
}

func (uc *ReturnRequestUsecase) findReturnInquiry(
	ctx context.Context,
	avatarID string,
	orderID string,
	itemIndex int,
) (
	inquirydom.Inquiry,
	bool,
	error,
) {
	notDeleted := false

	result, err :=
		uc.inquiryRepo.ListByAvatarID(
			ctx,
			avatarID,
			inquirydom.Filter{
				OrderID:        &orderID,
				OrderItemIndex: &itemIndex,
				Deleted:        &notDeleted,
			},
			inquirydom.Sort{},
			inquirydom.Page{
				Number:  1,
				PerPage: 200,
			},
		)
	if err != nil {
		return inquirydom.Inquiry{},
			false,
			err
	}

	var found *inquirydom.Inquiry

	for index := range result.Items {
		inquiry := result.Items[index]

		switch inquiry.InquiryType {
		case inquirydom.InquiryTypeReturnUnopened,
			inquirydom.InquiryTypeReturnOpened:
		default:
			continue
		}

		if found != nil {
			// More than one active return inquiry for the same Order item means
			// the persisted state has violated the one-return-inquiry invariant.
			return inquirydom.Inquiry{},
				false,
				inquirydom.ErrConflict
		}

		copy := inquiry
		found = &copy
	}

	if found == nil {
		return inquirydom.Inquiry{},
			false,
			nil
	}

	return *found,
		true,
		nil
}

func (uc *ReturnRequestUsecase) promoteInquiryToOpened(
	ctx context.Context,
	inquiry inquirydom.Inquiry,
	productID string,
) (inquirydom.Inquiry, error) {
	return uc.promoteInquiryToOpenedWithContent(
		ctx,
		inquiry,
		productID,
		"",
	)
}

func (uc *ReturnRequestUsecase) promoteInquiryToOpenedWithContent(
	ctx context.Context,
	inquiry inquirydom.Inquiry,
	productID string,
	content string,
) (inquirydom.Inquiry, error) {
	if productID != "" {
		if err := validateReturnInquiryProductID(
			inquiry,
			productID,
		); err != nil {
			return inquirydom.Inquiry{}, err
		}
	}

	now := uc.nowUTC()

	if err := inquiry.PromoteReturnOpened(
		now,
	); err != nil {
		return inquirydom.Inquiry{}, err
	}

	inquiryType :=
		inquirydom.InquiryTypeReturnOpened

	patch := inquirydom.InquiryPatch{
		InquiryType: &inquiryType,
		UpdatedAt:   &now,
	}

	if inquiry.ProductID == "" &&
		productID != "" {
		patch.ProductID = &productID
	}

	if content != "" {
		patch.Content = &content
	}

	return uc.inquiryRepo.Update(
		ctx,
		inquiry.ID,
		patch,
	)
}

func (uc *ReturnRequestUsecase) setReturnInquiryProductID(
	ctx context.Context,
	inquiry inquirydom.Inquiry,
	productID string,
) (inquirydom.Inquiry, error) {
	if inquiry.ProductID != "" {
		if inquiry.ProductID != productID {
			return inquirydom.Inquiry{},
				inquirydom.ErrConflict
		}

		return inquiry, nil
	}

	now := uc.nowUTC()

	return uc.inquiryRepo.Update(
		ctx,
		inquiry.ID,
		inquirydom.InquiryPatch{
			ProductID: &productID,
			UpdatedAt: &now,
		},
	)
}

func isServerObservedOpenedReturnItem(
	item orderdom.OrderItemSnapshot,
) bool {
	return item.Transferred ||
		item.TokenTransferVerifiedAt != nil
}

func validateReturnInquiryProductID(
	inquiry inquirydom.Inquiry,
	productID string,
) error {
	if productID == "" {
		return inquirydom.ErrInvalidProductID
	}

	if inquiry.ProductID != "" &&
		inquiry.ProductID != productID {
		return inquirydom.ErrConflict
	}

	return nil
}

func (uc *ReturnRequestUsecase) validateConfigured() error {
	if uc == nil ||
		uc.orderService == nil ||
		uc.inquiryRepo == nil ||
		uc.inquiryCreator == nil {
		return ErrReturnRequestUsecaseNotConfigured
	}

	return nil
}

func (uc *ReturnRequestUsecase) nowUTC() time.Time {
	if uc == nil || uc.now == nil {
		return time.Now().UTC()
	}

	return uc.now().UTC()
}

// returnInquiryID generates one stable Inquiry ID per Order item.
//
// A deterministic ID prevents two concurrent requests from creating separate
// return inquiries for the same Order item.
func returnInquiryID(
	orderID string,
	itemIndex int,
) string {
	source := fmt.Sprintf(
		"%s:%d",
		orderID,
		itemIndex,
	)

	sum := sha256.Sum256(
		[]byte(source),
	)

	return "inq_return_" +
		hex.EncodeToString(
			sum[:16],
		)
}

func intPointer(value int) *int {
	v := value
	return &v
}
