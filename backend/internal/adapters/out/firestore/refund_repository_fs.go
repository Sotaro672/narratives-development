// backend/internal/adapters/out/firestore/refund_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	refunddom "narratives/internal/domain/refund"
)

const (
	refundCollectionName           = "refunds"
	refundInquiryKeyCollectionName = "refundInquiryKeys"
)

var _ refunddom.RepositoryPort = (*RefundRepositoryFS)(nil)

// ============================================================
// RefundRepositoryFS
// ============================================================

// RefundRepositoryFS persists item-level purchaser refunds and the associated
// seller-side Transfer Reversal state.
//
// Firestore:
//
//	refunds/{refundId}
//
// Refund ID is deterministic:
//
//	{orderId}_{orderItemIndex}
//
// Inquiry uniqueness is protected by an internal key document:
//
//	refundInquiryKeys/{inquiryId}
//
// The key document and Refund document are created in the same Firestore
// transaction so concurrent receive-return requests cannot create two Refunds
// for the same Inquiry.
type RefundRepositoryFS struct {
	Client *firestore.Client
}

func NewRefundRepositoryFS(
	client *firestore.Client,
) *RefundRepositoryFS {
	return &RefundRepositoryFS{
		Client: client,
	}
}

func (r *RefundRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection(refundCollectionName)
}

func (r *RefundRepositoryFS) inquiryKeyCol() *firestore.CollectionRef {
	return r.Client.Collection(refundInquiryKeyCollectionName)
}

// ============================================================
// Get
// ============================================================

func (r *RefundRepositoryFS) GetByID(
	ctx context.Context,
	refundID string,
) (*refunddom.Refund, error) {
	if err := r.validateReady(); err != nil {
		return nil, err
	}

	if refundID == "" || strings.Contains(refundID, "/") {
		return nil, refunddom.ErrInvalidID
	}

	snapshot, err := r.col().Doc(refundID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, refunddom.ErrNotFound
		}

		return nil, err
	}

	refund, err := docToRefund(snapshot)
	if err != nil {
		return nil, err
	}

	return &refund, nil
}

func (r *RefundRepositoryFS) GetByInquiryID(
	ctx context.Context,
	inquiryID string,
) (*refunddom.Refund, error) {
	if err := r.validateReady(); err != nil {
		return nil, err
	}

	if inquiryID == "" || strings.Contains(inquiryID, "/") {
		return nil, refunddom.ErrInvalidInquiryID
	}

	keySnapshot, err := r.inquiryKeyCol().Doc(inquiryID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, refunddom.ErrNotFound
		}

		return nil, err
	}

	var key refundInquiryKeyDocument

	if err := keySnapshot.DataTo(&key); err != nil {
		return nil, err
	}

	if key.RefundID == "" || key.InquiryID != inquiryID {
		return nil, refunddom.ErrConflict
	}

	refund, err := r.GetByID(ctx, key.RefundID)
	if err != nil {
		if errors.Is(err, refunddom.ErrNotFound) {
			return nil, refunddom.ErrConflict
		}

		return nil, err
	}

	if refund.InquiryID != inquiryID {
		return nil, refunddom.ErrConflict
	}

	return refund, nil
}

func (r *RefundRepositoryFS) GetByOrderItem(
	ctx context.Context,
	orderID string,
	orderItemIndex int,
) (*refunddom.Refund, error) {
	if err := r.validateReady(); err != nil {
		return nil, err
	}

	refundID, err := refunddom.NewID(
		orderID,
		orderItemIndex,
	)
	if err != nil {
		return nil, err
	}

	refund, err := r.GetByID(
		ctx,
		refundID,
	)
	if err != nil {
		return nil, err
	}

	if refund.OrderID != orderID ||
		refund.OrderItemIndex != orderItemIndex {
		return nil, refunddom.ErrConflict
	}

	return refund, nil
}

// ============================================================
// List
// ============================================================

func (r *RefundRepositoryFS) ListByOrderID(
	ctx context.Context,
	orderID string,
) ([]refunddom.Refund, error) {
	if err := r.validateReady(); err != nil {
		return nil, err
	}

	if orderID == "" {
		return nil, refunddom.ErrInvalidOrderID
	}

	return r.listByField(
		ctx,
		"orderId",
		orderID,
	)
}

func (r *RefundRepositoryFS) ListByPaymentID(
	ctx context.Context,
	paymentID string,
) ([]refunddom.Refund, error) {
	if err := r.validateReady(); err != nil {
		return nil, err
	}

	if paymentID == "" {
		return nil, refunddom.ErrInvalidPaymentID
	}

	return r.listByField(
		ctx,
		"paymentId",
		paymentID,
	)
}

func (r *RefundRepositoryFS) ListBySettlementID(
	ctx context.Context,
	settlementID string,
) ([]refunddom.Refund, error) {
	if err := r.validateReady(); err != nil {
		return nil, err
	}

	if settlementID == "" {
		return nil, refunddom.ErrInvalidSettlementID
	}

	return r.listByField(
		ctx,
		"settlementId",
		settlementID,
	)
}

func (r *RefundRepositoryFS) ListByCompanyID(
	ctx context.Context,
	companyID string,
) ([]refunddom.Refund, error) {
	if err := r.validateReady(); err != nil {
		return nil, err
	}

	if companyID == "" {
		return nil, refunddom.ErrInvalidCompanyID
	}

	return r.listByField(
		ctx,
		"companyId",
		companyID,
	)
}

func (r *RefundRepositoryFS) listByField(
	ctx context.Context,
	field string,
	value any,
) ([]refunddom.Refund, error) {
	iter := r.col().
		Where(
			field,
			"==",
			value,
		).
		Documents(ctx)
	defer iter.Stop()

	result := make([]refunddom.Refund, 0)

	for {
		snapshot, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		refund, err := docToRefund(snapshot)
		if err != nil {
			return nil, err
		}

		result = append(result, refund)
	}

	// Firestore OrderBy is intentionally avoided here so these single-field
	// queries do not require extra composite indexes. Return ordering remains
	// deterministic in application memory.
	sortRefunds(result)

	return result, nil
}

// ============================================================
// Create
// ============================================================

func (r *RefundRepositoryFS) Create(
	ctx context.Context,
	in refunddom.CreateRefundInput,
) (*refunddom.Refund, error) {
	if err := r.validateReady(); err != nil {
		return nil, err
	}

	expectedRefundID, err := refunddom.NewID(
		in.OrderID,
		in.OrderItemIndex,
	)
	if err != nil {
		return nil, err
	}

	if in.RefundID == "" ||
		in.RefundID != expectedRefundID {
		return nil, refunddom.ErrInvalidID
	}

	entity, err := refunddom.New(
		in.RefundID,
		in.InquiryID,
		in.OrderID,
		in.PaymentID,
		in.OrderItemIndex,
		in.CompanyID,
		in.AccountID,
		in.SettlementID,
		in.MerchandiseAmount,
		in.MerchandiseTaxAmount,
		in.TransferReversalAmount,
		in.Currency,
		in.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	refundReference := r.col().Doc(entity.ID)
	inquiryKeyReference := r.inquiryKeyCol().Doc(entity.InquiryID)

	err = r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			transaction *firestore.Transaction,
		) error {
			refundSnapshot, err := transaction.Get(refundReference)
			if err == nil {
				current, decodeErr := docToRefund(refundSnapshot)
				if decodeErr != nil {
					return refunddom.ErrConflict
				}

				if current.OrderID == entity.OrderID &&
					current.OrderItemIndex == entity.OrderItemIndex {
					return refunddom.ErrDuplicateOrderItem
				}

				return refunddom.ErrConflict
			}
			if status.Code(err) != codes.NotFound {
				return err
			}

			keySnapshot, err := transaction.Get(inquiryKeyReference)
			if err == nil {
				var currentKey refundInquiryKeyDocument

				if decodeErr := keySnapshot.DataTo(&currentKey); decodeErr != nil {
					return refunddom.ErrConflict
				}

				return refunddom.ErrDuplicateInquiry
			}
			if status.Code(err) != codes.NotFound {
				return err
			}

			if err := transaction.Set(
				refundReference,
				refundToData(entity),
			); err != nil {
				return err
			}

			if err := transaction.Set(
				inquiryKeyReference,
				refundInquiryKeyDocument{
					RefundID:       entity.ID,
					InquiryID:      entity.InquiryID,
					OrderID:        entity.OrderID,
					OrderItemIndex: entity.OrderItemIndex,
					CreatedAt:      entity.CreatedAt,
				},
			); err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	created := entity
	return &created, nil
}

// ============================================================
// Update
// ============================================================

// UpdateByID applies one validated Refund domain transition atomically.
//
// Financial fields are never patched directly. The current Refund is loaded
// inside a Firestore transaction, the corresponding domain behavior is
// executed, the resulting complete aggregate is validated, and only then is it
// persisted.
func (r *RefundRepositoryFS) UpdateByID(
	ctx context.Context,
	refundID string,
	in refunddom.UpdateRefundInput,
) (*refunddom.Refund, error) {
	if err := r.validateReady(); err != nil {
		return nil, err
	}

	if refundID == "" || strings.Contains(refundID, "/") {
		return nil, refunddom.ErrInvalidID
	}

	if !refunddom.IsValidUpdateOperation(in.Operation) {
		return nil, refunddom.ErrInvalidUpdateOperation
	}

	if in.UpdatedAt.IsZero() {
		return nil, refunddom.ErrInvalidUpdatedAt
	}

	refundReference := r.col().Doc(refundID)

	var result refunddom.Refund

	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			transaction *firestore.Transaction,
		) error {
			snapshot, err := transaction.Get(refundReference)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return refunddom.ErrNotFound
				}

				return err
			}

			current, err := docToRefund(snapshot)
			if err != nil {
				return err
			}

			if current.ID != refundID {
				return refunddom.ErrConflict
			}

			if err := applyRefundUpdate(
				&current,
				in,
			); err != nil {
				return err
			}

			if err := current.Validate(); err != nil {
				return err
			}

			if err := transaction.Set(
				refundReference,
				refundToData(current),
			); err != nil {
				return err
			}

			result = current
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func applyRefundUpdate(
	current *refunddom.Refund,
	in refunddom.UpdateRefundInput,
) error {
	if current == nil {
		return refunddom.ErrConflict
	}

	switch in.Operation {
	case refunddom.UpdateOperationApplyStripeRefund:
		if in.StripeTransferReversalID != "" ||
			in.TransferReversedAt != nil {
			return refunddom.ErrInvalidUpdateOperation
		}

		return current.ApplyStripeRefund(
			in.StripeRefundID,
			in.RefundStatus,
			in.RefundedAt,
			in.UpdatedAt,
		)

	case refunddom.UpdateOperationMarkTransferReversalPending:
		if hasStripeRefundUpdateFields(in) ||
			in.StripeTransferReversalID != "" ||
			in.TransferReversedAt != nil {
			return refunddom.ErrInvalidUpdateOperation
		}

		return current.MarkTransferReversalPending(in.UpdatedAt)

	case refunddom.UpdateOperationMarkTransferReversalSucceeded:
		if hasStripeRefundUpdateFields(in) ||
			in.StripeTransferReversalID == "" ||
			in.TransferReversedAt == nil ||
			in.TransferReversedAt.IsZero() {
			return refunddom.ErrInvalidUpdateOperation
		}

		return current.MarkTransferReversalSucceeded(
			in.StripeTransferReversalID,
			*in.TransferReversedAt,
			in.UpdatedAt,
		)

	case refunddom.UpdateOperationMarkTransferReversalFailedRetryable:
		if hasStripeRefundUpdateFields(in) ||
			in.StripeTransferReversalID != "" ||
			in.TransferReversedAt != nil {
			return refunddom.ErrInvalidUpdateOperation
		}

		return current.MarkTransferReversalFailedRetryable(in.UpdatedAt)

	case refunddom.UpdateOperationMarkTransferReversalFailed:
		if hasStripeRefundUpdateFields(in) ||
			in.StripeTransferReversalID != "" ||
			in.TransferReversedAt != nil {
			return refunddom.ErrInvalidUpdateOperation
		}

		return current.MarkTransferReversalFailed(in.UpdatedAt)

	default:
		return refunddom.ErrInvalidUpdateOperation
	}
}

func hasStripeRefundUpdateFields(
	in refunddom.UpdateRefundInput,
) bool {
	return in.StripeRefundID != "" ||
		in.RefundStatus != "" ||
		in.RefundedAt != nil
}

// ============================================================
// Firestore Mapping
// ============================================================

type refundDocument struct {
	ID string `firestore:"id"`

	InquiryID string `firestore:"inquiryId"`

	OrderID        string `firestore:"orderId"`
	PaymentID      string `firestore:"paymentId"`
	OrderItemIndex int    `firestore:"orderItemIndex"`

	CompanyID string `firestore:"companyId"`
	AccountID string `firestore:"accountId"`

	SettlementID string `firestore:"settlementId"`

	Policy string `firestore:"policy,omitempty"`

	MerchandiseAmount    int `firestore:"merchandiseAmount"`
	MerchandiseTaxAmount int `firestore:"merchandiseTaxAmount"`

	OutboundShippingAmount    int `firestore:"outboundShippingAmount,omitempty"`
	OutboundShippingTaxAmount int `firestore:"outboundShippingTaxAmount,omitempty"`

	ReturnShippingAmount    int `firestore:"returnShippingAmount,omitempty"`
	ReturnShippingTaxAmount int `firestore:"returnShippingTaxAmount,omitempty"`

	RefundAmount int `firestore:"refundAmount"`

	Currency string `firestore:"currency"`

	StripeRefundID string     `firestore:"stripeRefundId"`
	Status         string     `firestore:"status"`
	RefundedAt     *time.Time `firestore:"refundedAt,omitempty"`

	TransferReversalAmount int `firestore:"transferReversalAmount"`

	StripeTransferReversalID string     `firestore:"stripeTransferReversalId"`
	TransferReversalStatus   string     `firestore:"transferReversalStatus"`
	TransferReversedAt       *time.Time `firestore:"transferReversedAt,omitempty"`

	CreatedAt time.Time `firestore:"createdAt"`
	UpdatedAt time.Time `firestore:"updatedAt"`
}

type refundInquiryKeyDocument struct {
	RefundID string `firestore:"refundId"`

	InquiryID string `firestore:"inquiryId"`

	OrderID        string `firestore:"orderId"`
	OrderItemIndex int    `firestore:"orderItemIndex"`

	CreatedAt time.Time `firestore:"createdAt"`
}

func refundToData(
	refund refunddom.Refund,
) refundDocument {
	return refundDocument{
		ID: refund.ID,

		InquiryID: refund.InquiryID,

		OrderID:        refund.OrderID,
		PaymentID:      refund.PaymentID,
		OrderItemIndex: refund.OrderItemIndex,

		CompanyID: refund.CompanyID,
		AccountID: refund.AccountID,

		SettlementID: refund.SettlementID,

		Policy: string(refund.Policy),

		MerchandiseAmount:    refund.MerchandiseAmount,
		MerchandiseTaxAmount: refund.MerchandiseTaxAmount,

		OutboundShippingAmount:    refund.OutboundShippingAmount,
		OutboundShippingTaxAmount: refund.OutboundShippingTaxAmount,

		ReturnShippingAmount:    refund.ReturnShippingAmount,
		ReturnShippingTaxAmount: refund.ReturnShippingTaxAmount,

		RefundAmount: refund.RefundAmount,

		Currency: refund.Currency,

		StripeRefundID: refund.StripeRefundID,
		Status:         string(refund.Status),
		RefundedAt:     cloneTimePointer(refund.RefundedAt),

		TransferReversalAmount: refund.TransferReversalAmount,

		StripeTransferReversalID: refund.StripeTransferReversalID,
		TransferReversalStatus:   string(refund.TransferReversalStatus),
		TransferReversedAt:       cloneTimePointer(refund.TransferReversedAt),

		CreatedAt: refund.CreatedAt.UTC(),
		UpdatedAt: refund.UpdatedAt.UTC(),
	}
}

func docToRefund(
	snapshot *firestore.DocumentSnapshot,
) (refunddom.Refund, error) {
	if snapshot == nil {
		return refunddom.Refund{}, refunddom.ErrNotFound
	}

	var document refundDocument

	if err := snapshot.DataTo(&document); err != nil {
		return refunddom.Refund{}, err
	}

	if document.ID == "" {
		document.ID = snapshot.Ref.ID
	}

	refund := refunddom.Refund{
		ID: document.ID,

		InquiryID: document.InquiryID,

		OrderID:        document.OrderID,
		PaymentID:      document.PaymentID,
		OrderItemIndex: document.OrderItemIndex,

		CompanyID: document.CompanyID,
		AccountID: document.AccountID,

		SettlementID: document.SettlementID,

		Policy: refunddom.OpenedReturnRefundPolicy(document.Policy),

		MerchandiseAmount:    document.MerchandiseAmount,
		MerchandiseTaxAmount: document.MerchandiseTaxAmount,

		OutboundShippingAmount:    document.OutboundShippingAmount,
		OutboundShippingTaxAmount: document.OutboundShippingTaxAmount,

		ReturnShippingAmount:    document.ReturnShippingAmount,
		ReturnShippingTaxAmount: document.ReturnShippingTaxAmount,

		RefundAmount: document.RefundAmount,

		Currency: document.Currency,

		StripeRefundID: document.StripeRefundID,
		Status: refunddom.RefundStatus(
			document.Status,
		),
		RefundedAt: cloneTimePointer(document.RefundedAt),

		TransferReversalAmount: document.TransferReversalAmount,

		StripeTransferReversalID: document.StripeTransferReversalID,
		TransferReversalStatus: refunddom.TransferReversalStatus(
			document.TransferReversalStatus,
		),
		TransferReversedAt: cloneTimePointer(document.TransferReversedAt),

		CreatedAt: document.CreatedAt.UTC(),
		UpdatedAt: document.UpdatedAt.UTC(),
	}

	if err := refund.Validate(); err != nil {
		return refunddom.Refund{}, err
	}

	return refund, nil
}

func cloneTimePointer(
	value *time.Time,
) *time.Time {
	if value == nil {
		return nil
	}

	cloned := value.UTC()
	return &cloned
}

// ============================================================
// Helpers
// ============================================================

func sortRefunds(
	refunds []refunddom.Refund,
) {
	sort.Slice(
		refunds,
		func(i, j int) bool {
			if refunds[i].OrderItemIndex ==
				refunds[j].OrderItemIndex {
				return refunds[i].ID <
					refunds[j].ID
			}

			return refunds[i].OrderItemIndex <
				refunds[j].OrderItemIndex
		},
	)
}

func (r *RefundRepositoryFS) validateReady() error {
	if r == nil ||
		r.Client == nil {
		return errors.New(
			"refund: firestore client is nil",
		)
	}

	return nil
}
