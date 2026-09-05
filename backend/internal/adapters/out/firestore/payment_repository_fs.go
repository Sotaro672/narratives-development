// backend/internal/adapters/out/firestore/payment_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	usecase "narratives/internal/application/usecase"
	paymentdom "narratives/internal/domain/payment"
)

var (
	_ paymentdom.RepositoryPort            = (*PaymentRepositoryFS)(nil)
	_ usecase.StripePaymentEventRepository = (*PaymentRepositoryFS)(nil)
)

// PaymentRepositoryFS is the Firestore-based implementation of:
//
// - payment.RepositoryPort
// - usecase.StripePaymentEventRepository
//
// Firestore design:
//
//	payments/{paymentId}
//	paymentStripeEvents/{stripeEventId}
//
// Payment document rules:
//
// - payment document ID = paymentId
// - paymentId must be the same value as order.ID
// - paymentId is not stored as a document field
// - stripePaymentIntentId is required for every status
// - transferGroup is required for every status
// - stripeChargeId is optional until Stripe has created a Charge
// - postPaidTriggeredAt is an internal exactly-once claim marker
//
// Stripe event rules:
//
//   - Stripe event ID is used as the event document ID
//   - duplicate event IDs are successful no-ops
//   - event marker creation, Payment status update, and post-paid marker
//     acquisition occur in one Firestore Transaction
type PaymentRepositoryFS struct {
	Client *firestore.Client
}

func NewPaymentRepositoryFS(client *firestore.Client) *PaymentRepositoryFS {
	return &PaymentRepositoryFS{Client: client}
}

func (r *PaymentRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("payments")
}

func (r *PaymentRepositoryFS) stripeEventCol() *firestore.CollectionRef {
	return r.Client.Collection("paymentStripeEvents")
}

// ============================================================
// payment.RepositoryPort
// ============================================================

func (r *PaymentRepositoryFS) GetByPaymentID(ctx context.Context, paymentID string) (*paymentdom.Payment, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	paymentID = strings.TrimSpace(paymentID)
	if paymentID == "" {
		return nil, paymentdom.ErrNotFound
	}

	snapshot, err := r.col().Doc(paymentID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, paymentdom.ErrNotFound
		}
		return nil, err
	}

	payment, err := docToPayment(snapshot)
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

func (r *PaymentRepositoryFS) Create(ctx context.Context, in paymentdom.CreatePaymentInput) (*paymentdom.Payment, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	in.PaymentID = strings.TrimSpace(in.PaymentID)
	in.PaymentMethodID = strings.TrimSpace(in.PaymentMethodID)
	in.StripeCustomerID = strings.TrimSpace(in.StripeCustomerID)
	in.StripePaymentMethodID = strings.TrimSpace(in.StripePaymentMethodID)
	in.StripePaymentIntentID = strings.TrimSpace(in.StripePaymentIntentID)
	in.StripeChargeID = strings.TrimSpace(in.StripeChargeID)
	in.TransferGroup = strings.TrimSpace(in.TransferGroup)

	if in.PaymentID == "" {
		return nil, paymentdom.ErrInvalidPaymentID
	}

	createdAt := time.Now().UTC()

	// Validate the complete Domain entity before writing anything.
	payment, err := paymentdom.New(
		in.PaymentID,
		in.PaymentMethodID,
		in.StripeCustomerID,
		in.StripePaymentMethodID,
		in.StripePaymentIntentID,
		in.StripeChargeID,
		in.TransferGroup,
		in.Amount,
		in.Status,
		normalizeOptionalString(in.ErrorType),
		normalizeOptionalString(in.ErrorCode),
		normalizeOptionalString(in.ErrorMsg),
		createdAt,
	)
	if err != nil {
		return nil, err
	}

	documentReference := r.col().Doc(payment.PaymentID)
	data := paymentToCreateData(payment)

	// When a Payment is initially created as succeeded, PaymentUsecase.Create
	// executes the post-paid processing. Store the claim marker in the same
	// transaction as the Payment creation so that a later succeeded webhook
	// does not execute it again.
	if payment.Status == paymentdom.StatusSucceeded {
		data["postPaidTriggeredAt"] = createdAt
	}

	err = r.Client.RunTransaction(ctx, func(ctx context.Context, transaction *firestore.Transaction) error {
		return transaction.Create(documentReference, data)
	})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil, paymentdom.ErrConflict
		}
		return nil, err
	}

	return &payment, nil
}

func (r *PaymentRepositoryFS) UpdateByPaymentID(ctx context.Context, paymentID string, patch paymentdom.UpdatePaymentInput) (*paymentdom.Payment, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	paymentID = strings.TrimSpace(paymentID)
	if paymentID == "" {
		return nil, paymentdom.ErrNotFound
	}

	// Stripe-originated status updates must use
	// ApplyStripePaymentEvent so event deduplication, transition validation,
	// and post-paid marker acquisition remain atomic.
	if patch.Status != nil {
		return nil, usecase.ErrPaymentStatusUpdateRequiresStripeEvent
	}

	if hasPaymentRefundPatch(patch) {
		if hasPaymentNonRefundPatch(patch) {
			return nil, paymentdom.ErrInvalidRefundState
		}
		return r.updateRefundStateByPaymentID(ctx, paymentID, patch)
	}

	documentReference := r.col().Doc(paymentID)
	updates := make([]firestore.Update, 0, 10)

	if patch.PaymentMethodID != nil {
		value := strings.TrimSpace(*patch.PaymentMethodID)
		if value == "" {
			return nil, paymentdom.ErrInvalidPaymentMethodID
		}
		updates = append(updates, firestore.Update{Path: "paymentMethodId", Value: value})
	}

	if patch.StripeCustomerID != nil {
		value := strings.TrimSpace(*patch.StripeCustomerID)
		if value == "" {
			return nil, paymentdom.ErrInvalidStripeCustomerID
		}
		updates = append(updates, firestore.Update{Path: "stripeCustomerId", Value: value})
	}

	if patch.StripePaymentMethodID != nil {
		value := strings.TrimSpace(*patch.StripePaymentMethodID)
		if value == "" {
			return nil, paymentdom.ErrInvalidStripePaymentMethod
		}
		updates = append(updates, firestore.Update{Path: "stripePaymentMethodId", Value: value})
	}

	if patch.StripePaymentIntentID != nil {
		value := strings.TrimSpace(*patch.StripePaymentIntentID)
		if value == "" {
			return nil, paymentdom.ErrInvalidStripePaymentIntent
		}
		updates = append(updates, firestore.Update{Path: "stripePaymentIntentId", Value: value})
	}

	if patch.StripeChargeID != nil {
		value := strings.TrimSpace(*patch.StripeChargeID)
		if value == "" {
			return nil, paymentdom.ErrInvalidStripeChargeID
		}
		updates = append(updates, firestore.Update{Path: "stripeChargeId", Value: value})
	}

	if patch.TransferGroup != nil {
		value := strings.TrimSpace(*patch.TransferGroup)
		if value == "" {
			return nil, paymentdom.ErrInvalidTransferGroup
		}
		updates = append(updates, firestore.Update{Path: "transferGroup", Value: value})
	}

	if patch.Amount != nil {
		if *patch.Amount < paymentdom.MinAmount || (paymentdom.MaxAmount > 0 && *patch.Amount > paymentdom.MaxAmount) {
			return nil, paymentdom.ErrInvalidAmount
		}
		updates = append(updates, firestore.Update{Path: "amount", Value: *patch.Amount})
	}

	if patch.ErrorType != nil {
		updates = appendOptionalStringUpdate(updates, "errorType", patch.ErrorType)
	}
	if patch.ErrorCode != nil {
		updates = appendOptionalStringUpdate(updates, "errorCode", patch.ErrorCode)
	}
	if patch.ErrorMsg != nil {
		updates = appendOptionalStringUpdate(updates, "errorMsg", patch.ErrorMsg)
	}

	if len(updates) == 0 {
		return r.GetByPaymentID(ctx, paymentID)
	}

	updates = append(updates, firestore.Update{Path: "updatedAt", Value: time.Now().UTC()})

	_, err := documentReference.Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, paymentdom.ErrNotFound
		}
		return nil, err
	}

	return r.GetByPaymentID(ctx, paymentID)
}

func hasPaymentRefundPatch(patch paymentdom.UpdatePaymentInput) bool {
	return patch.StripeRefundID != nil || patch.RefundStatus != nil || patch.RefundedAmount != nil || patch.RefundedAt != nil
}

func hasPaymentNonRefundPatch(patch paymentdom.UpdatePaymentInput) bool {
	return patch.PaymentMethodID != nil ||
		patch.StripeCustomerID != nil ||
		patch.StripePaymentMethodID != nil ||
		patch.StripePaymentIntentID != nil ||
		patch.StripeChargeID != nil ||
		patch.TransferGroup != nil ||
		patch.Amount != nil ||
		patch.ErrorType != nil ||
		patch.ErrorCode != nil ||
		patch.ErrorMsg != nil
}

func (r *PaymentRepositoryFS) updateRefundStateByPaymentID(ctx context.Context, paymentID string, patch paymentdom.UpdatePaymentInput) (*paymentdom.Payment, error) {
	if patch.StripeRefundID == nil || patch.RefundStatus == nil || patch.RefundedAmount == nil {
		return nil, paymentdom.ErrInvalidRefundState
	}

	stripeRefundID := strings.TrimSpace(*patch.StripeRefundID)
	if stripeRefundID == "" {
		return nil, paymentdom.ErrInvalidStripeRefundID
	}

	documentReference := r.col().Doc(paymentID)
	var updated paymentdom.Payment

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, transaction *firestore.Transaction) error {
		snapshot, err := transaction.Get(documentReference)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return paymentdom.ErrNotFound
			}
			return err
		}

		current, err := docToPayment(snapshot)
		if err != nil {
			return err
		}

		next := current
		if err := next.SetRefundState(stripeRefundID, *patch.RefundStatus, *patch.RefundedAmount, patch.RefundedAt); err != nil {
			return err
		}

		updates := []firestore.Update{
			{Path: "stripeRefundId", Value: next.StripeRefundID},
			{Path: "refundStatus", Value: string(next.RefundStatus)},
			{Path: "refundedAmount", Value: next.RefundedAmount},
		}

		if next.RefundedAt == nil {
			updates = append(updates, firestore.Update{Path: "refundedAt", Value: firestore.Delete})
		} else {
			updates = append(updates, firestore.Update{Path: "refundedAt", Value: next.RefundedAt.UTC()})
		}

		updates = append(updates, firestore.Update{Path: "updatedAt", Value: time.Now().UTC()})

		if err := transaction.Update(documentReference, updates); err != nil {
			return err
		}

		updated = next
		return nil
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, paymentdom.ErrNotFound
		}
		return nil, err
	}

	return &updated, nil
}

// ============================================================
// usecase.StripePaymentEventRepository
// ============================================================

// ApplyStripePaymentEvent atomically:
//
//  1. Deduplicates the Stripe event.
//  2. Reads and validates the current Payment.
//  3. Verifies the Stripe PaymentIntent ID.
//  4. Persists the latest Stripe Charge ID when available.
//  5. Applies a valid status transition.
//  6. Acquires the post-paid marker if this is the first succeeded state.
//  7. Records the Stripe event as processed.
func (r *PaymentRepositoryFS) ApplyStripePaymentEvent(ctx context.Context, in usecase.ApplyStripePaymentEventInput) (*usecase.ApplyStripePaymentEventResult, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	in.EventID = strings.TrimSpace(in.EventID)
	in.PaymentID = strings.TrimSpace(in.PaymentID)
	in.StripePaymentIntentID = strings.TrimSpace(in.StripePaymentIntentID)
	in.StripeChargeID = strings.TrimSpace(in.StripeChargeID)

	if in.EventID == "" {
		return nil, usecase.ErrPaymentStripeEventIDEmpty
	}
	if strings.Contains(in.EventID, "/") {
		return nil, fmt.Errorf("payment: invalid Stripe event id %q", in.EventID)
	}
	if in.PaymentID == "" {
		return nil, paymentdom.ErrInvalidPaymentID
	}
	if in.StripePaymentIntentID == "" {
		return nil, paymentdom.ErrInvalidStripePaymentIntent
	}
	if !paymentdom.IsValidStatus(in.Status) {
		return nil, paymentdom.ErrInvalidStatus
	}
	if in.OccurredAt.IsZero() {
		return nil, usecase.ErrPaymentStripeEventOccurredAtInvalid
	}

	in.OccurredAt = in.OccurredAt.UTC()
	in.ErrorType = normalizeOptionalString(in.ErrorType)
	in.ErrorCode = normalizeOptionalString(in.ErrorCode)
	in.ErrorMsg = normalizeOptionalString(in.ErrorMsg)

	paymentReference := r.col().Doc(in.PaymentID)
	eventReference := r.stripeEventCol().Doc(in.EventID)
	processedAt := time.Now().UTC()

	var result *usecase.ApplyStripePaymentEventResult

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, transaction *firestore.Transaction) error {
		// Read all required documents before any write.
		eventSnapshot, eventErr := transaction.Get(eventReference)
		if eventErr != nil && status.Code(eventErr) != codes.NotFound {
			return eventErr
		}

		paymentSnapshot, paymentErr := transaction.Get(paymentReference)
		if paymentErr != nil {
			if status.Code(paymentErr) == codes.NotFound {
				return paymentdom.ErrNotFound
			}
			return paymentErr
		}

		current, decodeErr := docToPayment(paymentSnapshot)
		if decodeErr != nil {
			return decodeErr
		}

		// Duplicate Stripe event: successful no-op.
		if eventErr == nil && eventSnapshot != nil && eventSnapshot.Exists() {
			result = &usecase.ApplyStripePaymentEventResult{
				Payment:          &current,
				EventApplied:     false,
				StatusChanged:    false,
				PostPaidRequired: false,
			}
			return nil
		}

		if current.StripePaymentIntentID != in.StripePaymentIntentID {
			return paymentdom.ErrInvalidStripePaymentIntent
		}

		transitionAllowed := paymentStatusTransitionAllowed(current.Status, in.Status)
		next := current
		statusChanged := false
		stripeChargeChanged := false

		if transitionAllowed {
			statusChanged = current.Status != in.Status

			if in.StripeChargeID != "" {
				stripeChargeChanged = current.StripeChargeID != in.StripeChargeID
				next.StripeChargeID = in.StripeChargeID
			}

			next.Status = in.Status

			switch in.Status {
			case paymentdom.StatusFailed, paymentdom.StatusCanceled:
				next.ErrorType = in.ErrorType
				next.ErrorCode = in.ErrorCode
				next.ErrorMsg = in.ErrorMsg
			default:
				// A non-error Stripe state clears stale error metadata.
				next.ErrorType = nil
				next.ErrorCode = nil
				next.ErrorMsg = nil
			}

			validated, validationErr := paymentdom.New(
				next.PaymentID,
				next.PaymentMethodID,
				next.StripeCustomerID,
				next.StripePaymentMethodID,
				next.StripePaymentIntentID,
				next.StripeChargeID,
				next.TransferGroup,
				next.Amount,
				next.Status,
				next.ErrorType,
				next.ErrorCode,
				next.ErrorMsg,
				next.CreatedAt,
			)
			if validationErr != nil {
				return validationErr
			}

			if validationErr := validated.SetRefundState(
				current.StripeRefundID,
				current.RefundStatus,
				current.RefundedAmount,
				current.RefundedAt,
			); validationErr != nil {
				return validationErr
			}

			next = validated
		}

		postPaidMarkerExists, markerErr := paymentPostPaidMarkerExists(paymentSnapshot.Data())
		if markerErr != nil {
			return markerErr
		}

		postPaidRequired := transitionAllowed &&
			next.Status == paymentdom.StatusSucceeded &&
			in.Status == paymentdom.StatusSucceeded &&
			!postPaidMarkerExists

		updates := make([]firestore.Update, 0, 7)

		if transitionAllowed {
			updates = append(
				updates,
				firestore.Update{Path: "status", Value: string(next.Status)},
				firestore.Update{Path: "updatedAt", Value: processedAt},
			)

			updates = appendOptionalStringUpdate(updates, "errorType", next.ErrorType)
			updates = appendOptionalStringUpdate(updates, "errorCode", next.ErrorCode)
			updates = appendOptionalStringUpdate(updates, "errorMsg", next.ErrorMsg)

			if stripeChargeChanged {
				updates = append(updates, firestore.Update{Path: "stripeChargeId", Value: next.StripeChargeID})
			}
		}

		if postPaidRequired {
			updates = append(updates, firestore.Update{Path: "postPaidTriggeredAt", Value: processedAt})
		}

		if len(updates) > 0 {
			if updateErr := transaction.Update(paymentReference, updates); updateErr != nil {
				return updateErr
			}
		}

		eventData := map[string]any{
			"eventId":               in.EventID,
			"paymentId":             in.PaymentID,
			"stripePaymentIntentId": in.StripePaymentIntentID,
			"requestedStatus":       string(in.Status),
			"appliedStatus":         string(next.Status),
			"transitionApplied":     transitionAllowed,
			"statusChanged":         statusChanged,
			"postPaidRequired":      postPaidRequired,
			"occurredAt":            in.OccurredAt,
			"processedAt":           processedAt,
		}

		if in.StripeChargeID != "" {
			eventData["stripeChargeId"] = in.StripeChargeID
		}
		setOptionalString(eventData, "errorType", in.ErrorType)
		setOptionalString(eventData, "errorCode", in.ErrorCode)
		setOptionalString(eventData, "errorMsg", in.ErrorMsg)

		if createErr := transaction.Create(eventReference, eventData); createErr != nil {
			return createErr
		}

		result = &usecase.ApplyStripePaymentEventResult{
			Payment:          &next,
			EventApplied:     true,
			StatusChanged:    statusChanged,
			PostPaidRequired: postPaidRequired,
		}
		return nil
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, paymentdom.ErrNotFound
		}
		return nil, err
	}

	if result == nil || result.Payment == nil {
		return nil, usecase.ErrPaymentStripeEventResultEmpty
	}

	return result, nil
}

// ============================================================
// Stripe status transition policy
// ============================================================

// paymentStatusTransitionAllowed prevents stale or out-of-order Stripe
// webhook events from regressing a terminal Payment state.
//
// Invalid/stale transitions are recorded as processed events but do not
// update the Payment.
func paymentStatusTransitionAllowed(current paymentdom.PaymentStatus, next paymentdom.PaymentStatus) bool {
	if current == next {
		return true
	}

	switch current {
	case paymentdom.StatusPending:
		switch next {
		case paymentdom.StatusRequiresAction,
			paymentdom.StatusProcessing,
			paymentdom.StatusSucceeded,
			paymentdom.StatusFailed,
			paymentdom.StatusCanceled:
			return true
		}

	case paymentdom.StatusRequiresAction:
		switch next {
		case paymentdom.StatusPending,
			paymentdom.StatusProcessing,
			paymentdom.StatusSucceeded,
			paymentdom.StatusFailed,
			paymentdom.StatusCanceled:
			return true
		}

	case paymentdom.StatusProcessing:
		switch next {
		case paymentdom.StatusRequiresAction,
			paymentdom.StatusSucceeded,
			paymentdom.StatusFailed,
			paymentdom.StatusCanceled:
			return true
		}

	case paymentdom.StatusFailed:
		// A PaymentIntent can recover after a new payment method or another
		// confirmation attempt.
		switch next {
		case paymentdom.StatusPending,
			paymentdom.StatusRequiresAction,
			paymentdom.StatusProcessing,
			paymentdom.StatusSucceeded,
			paymentdom.StatusCanceled:
			return true
		}

	case paymentdom.StatusSucceeded:
		// succeeded is terminal.
		return false

	case paymentdom.StatusCanceled:
		// canceled is terminal.
		return false
	}

	return false
}

// ============================================================
// Document conversion
// ============================================================

func paymentToCreateData(payment paymentdom.Payment) map[string]any {
	data := map[string]any{
		"amount":                payment.Amount,
		"createdAt":             payment.CreatedAt,
		"paymentMethodId":       payment.PaymentMethodID,
		"status":                string(payment.Status),
		"stripeCustomerId":      payment.StripeCustomerID,
		"stripePaymentIntentId": payment.StripePaymentIntentID,
		"stripePaymentMethodId": payment.StripePaymentMethodID,
		"transferGroup":         payment.TransferGroup,
		"refundStatus":          string(payment.RefundStatus),
		"refundedAmount":        payment.RefundedAmount,
	}

	if payment.StripeChargeID != "" {
		data["stripeChargeId"] = payment.StripeChargeID
	}
	if payment.StripeRefundID != "" {
		data["stripeRefundId"] = payment.StripeRefundID
	}
	if payment.RefundedAt != nil {
		data["refundedAt"] = payment.RefundedAt.UTC()
	}

	setOptionalString(data, "errorType", payment.ErrorType)
	setOptionalString(data, "errorCode", payment.ErrorCode)
	setOptionalString(data, "errorMsg", payment.ErrorMsg)

	return data
}

func docToPayment(document *firestore.DocumentSnapshot) (paymentdom.Payment, error) {
	if document == nil {
		return paymentdom.Payment{}, errors.New("payment: document snapshot is nil")
	}

	data := document.Data()
	if data == nil {
		return paymentdom.Payment{}, fmt.Errorf("empty payment document: %s", document.Ref.ID)
	}

	paymentID := document.Ref.ID

	paymentMethodID, err := paymentRequiredString(data, "paymentMethodId")
	if err != nil {
		return paymentdom.Payment{}, err
	}

	stripeCustomerID, err := paymentRequiredString(data, "stripeCustomerId")
	if err != nil {
		return paymentdom.Payment{}, err
	}

	stripePaymentMethodID, err := paymentRequiredString(data, "stripePaymentMethodId")
	if err != nil {
		return paymentdom.Payment{}, err
	}

	stripePaymentIntentID, err := paymentRequiredString(data, "stripePaymentIntentId")
	if err != nil {
		return paymentdom.Payment{}, err
	}

	stripeChargeIDValue, err := firestoreOptionalString(data, "stripeChargeId")
	if err != nil {
		return paymentdom.Payment{}, err
	}
	stripeChargeID := ""
	if stripeChargeIDValue != nil {
		stripeChargeID = *stripeChargeIDValue
	}

	transferGroup, err := paymentRequiredString(data, "transferGroup")
	if err != nil {
		return paymentdom.Payment{}, err
	}

	amount, err := paymentRequiredInt(data, "amount")
	if err != nil {
		return paymentdom.Payment{}, err
	}

	statusText, err := paymentRequiredString(data, "status")
	if err != nil {
		return paymentdom.Payment{}, err
	}

	createdAt, err := firestoreRequiredTime(data, "createdAt")
	if err != nil {
		return paymentdom.Payment{}, err
	}

	errorType, err := firestoreOptionalString(data, "errorType")
	if err != nil {
		return paymentdom.Payment{}, err
	}
	errorCode, err := firestoreOptionalString(data, "errorCode")
	if err != nil {
		return paymentdom.Payment{}, err
	}
	errorMsg, err := firestoreOptionalString(data, "errorMsg")
	if err != nil {
		return paymentdom.Payment{}, err
	}

	payment, err := paymentdom.New(
		paymentID,
		paymentMethodID,
		stripeCustomerID,
		stripePaymentMethodID,
		stripePaymentIntentID,
		stripeChargeID,
		transferGroup,
		amount,
		paymentdom.PaymentStatus(statusText),
		errorType,
		errorCode,
		errorMsg,
		createdAt,
	)
	if err != nil {
		return paymentdom.Payment{}, err
	}

	stripeRefundIDValue, err := firestoreOptionalString(data, "stripeRefundId")
	if err != nil {
		return paymentdom.Payment{}, err
	}
	stripeRefundID := ""
	if stripeRefundIDValue != nil {
		stripeRefundID = *stripeRefundIDValue
	}

	refundStatus, err := paymentRefundStatus(data, "refundStatus")
	if err != nil {
		return paymentdom.Payment{}, err
	}

	refundedAmount, err := paymentRefundedAmount(data, "refundedAmount")
	if err != nil {
		return paymentdom.Payment{}, err
	}

	refundedAt, err := firestoreOptionalTime(data, "refundedAt")
	if err != nil {
		return paymentdom.Payment{}, err
	}

	if err := payment.SetRefundState(stripeRefundID, refundStatus, refundedAmount, refundedAt); err != nil {
		return paymentdom.Payment{}, err
	}

	return payment, nil
}

// ============================================================
// Firestore field helpers
// ============================================================

func paymentRefundStatus(values map[string]any, key string) (paymentdom.RefundStatus, error) {
	text, err := firestoreOptionalString(values, key)
	if err != nil {
		return "", err
	}
	if text == nil {
		return paymentdom.DefaultRefundStatus, nil
	}

	refundStatus := paymentdom.RefundStatus(*text)
	if !paymentdom.IsValidRefundStatus(refundStatus) {
		return "", paymentdom.ErrInvalidRefundStatus
	}

	return refundStatus, nil
}

func paymentRefundedAmount(values map[string]any, key string) (int, error) {
	value, exists := values[key]
	if !exists || value == nil {
		return 0, nil
	}

	number, ok := value.(int64)
	if !ok {
		return 0, fmt.Errorf("payment: invalid %s", key)
	}

	return int(number), nil
}

func paymentRequiredString(values map[string]any, key string) (string, error) {
	value, ok := values[key]
	if !ok || value == nil {
		return "", fmt.Errorf("payment: missing %s", key)
	}

	text, ok := value.(string)
	text = strings.TrimSpace(text)
	if !ok || text == "" {
		return "", fmt.Errorf("payment: invalid %s", key)
	}

	return text, nil
}

func paymentRequiredInt(values map[string]any, key string) (int, error) {
	value, ok := values[key]
	if !ok || value == nil {
		return 0, fmt.Errorf("payment: missing %s", key)
	}

	number, ok := value.(int64)
	if !ok {
		return 0, fmt.Errorf("payment: invalid %s", key)
	}

	return int(number), nil
}

func paymentPostPaidMarkerExists(values map[string]any) (bool, error) {
	timestamp, err := firestoreOptionalTime(values, "postPaidTriggeredAt")
	if err != nil {
		return false, err
	}
	return timestamp != nil, nil
}
