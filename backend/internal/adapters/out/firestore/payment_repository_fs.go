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
	_ paymentdom.RepositoryPort = (*PaymentRepositoryFS)(nil)

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

func NewPaymentRepositoryFS(
	client *firestore.Client,
) *PaymentRepositoryFS {
	return &PaymentRepositoryFS{
		Client: client,
	}
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

func (r *PaymentRepositoryFS) GetByPaymentID(
	ctx context.Context,
	paymentID string,
) (*paymentdom.Payment, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New(
			"firestore client is nil",
		)
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

func (r *PaymentRepositoryFS) Create(
	ctx context.Context,
	in paymentdom.CreatePaymentInput,
) (*paymentdom.Payment, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New(
			"firestore client is nil",
		)
	}

	in.PaymentID = strings.TrimSpace(in.PaymentID)
	in.PaymentMethodID = strings.TrimSpace(
		in.PaymentMethodID,
	)
	in.StripeCustomerID = strings.TrimSpace(
		in.StripeCustomerID,
	)
	in.StripePaymentMethodID = strings.TrimSpace(
		in.StripePaymentMethodID,
	)
	in.StripePaymentIntentID = strings.TrimSpace(
		in.StripePaymentIntentID,
	)

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
		in.Amount,
		in.Status,
		normalizePaymentOptionalString(in.ErrorType),
		normalizePaymentOptionalString(in.ErrorCode),
		normalizePaymentOptionalString(in.ErrorMsg),
		createdAt,
	)
	if err != nil {
		return nil, err
	}

	documentReference := r.col().Doc(
		payment.PaymentID,
	)

	data := paymentToCreateData(payment)

	// When a Payment is initially created as succeeded, PaymentUsecase.Create
	// executes the post-paid processing. Store the claim marker in the same
	// transaction as the Payment creation so that a later succeeded webhook
	// does not execute it again.
	if payment.Status == paymentdom.StatusSucceeded {
		data["postPaidTriggeredAt"] = createdAt
	}

	err = r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			transaction *firestore.Transaction,
		) error {
			return transaction.Create(
				documentReference,
				data,
			)
		},
	)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil, paymentdom.ErrConflict
		}

		return nil, err
	}

	return &payment, nil
}

func (r *PaymentRepositoryFS) UpdateByPaymentID(
	ctx context.Context,
	paymentID string,
	patch paymentdom.UpdatePaymentInput,
) (*paymentdom.Payment, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New(
			"firestore client is nil",
		)
	}

	paymentID = strings.TrimSpace(paymentID)
	if paymentID == "" {
		return nil, paymentdom.ErrNotFound
	}

	// Stripe-originated status updates must use
	// ApplyStripePaymentEvent so event deduplication, transition validation,
	// and post-paid marker acquisition remain atomic.
	if patch.Status != nil {
		return nil,
			usecase.ErrPaymentStatusUpdateRequiresStripeEvent
	}

	documentReference := r.col().Doc(paymentID)
	updates := make(
		[]firestore.Update,
		0,
		8,
	)

	if patch.PaymentMethodID != nil {
		value := strings.TrimSpace(
			*patch.PaymentMethodID,
		)
		if value == "" {
			return nil,
				paymentdom.ErrInvalidPaymentMethodID
		}

		updates = append(
			updates,
			firestore.Update{
				Path:  "paymentMethodId",
				Value: value,
			},
		)
	}

	if patch.StripeCustomerID != nil {
		value := strings.TrimSpace(
			*patch.StripeCustomerID,
		)
		if value == "" {
			return nil,
				paymentdom.ErrInvalidStripeCustomerID
		}

		updates = append(
			updates,
			firestore.Update{
				Path:  "stripeCustomerId",
				Value: value,
			},
		)
	}

	if patch.StripePaymentMethodID != nil {
		value := strings.TrimSpace(
			*patch.StripePaymentMethodID,
		)
		if value == "" {
			return nil,
				paymentdom.ErrInvalidStripePaymentMethod
		}

		updates = append(
			updates,
			firestore.Update{
				Path:  "stripePaymentMethodId",
				Value: value,
			},
		)
	}

	if patch.StripePaymentIntentID != nil {
		value := strings.TrimSpace(
			*patch.StripePaymentIntentID,
		)
		if value == "" {
			return nil,
				paymentdom.ErrInvalidStripePaymentIntent
		}

		updates = append(
			updates,
			firestore.Update{
				Path:  "stripePaymentIntentId",
				Value: value,
			},
		)
	}

	if patch.Amount != nil {
		if *patch.Amount < paymentdom.MinAmount ||
			(paymentdom.MaxAmount > 0 &&
				*patch.Amount > paymentdom.MaxAmount) {
			return nil, paymentdom.ErrInvalidAmount
		}

		updates = append(
			updates,
			firestore.Update{
				Path:  "amount",
				Value: *patch.Amount,
			},
		)
	}

	if patch.ErrorType != nil {
		updates = appendPaymentOptionalStringUpdate(
			updates,
			"errorType",
			patch.ErrorType,
		)
	}

	if patch.ErrorCode != nil {
		updates = appendPaymentOptionalStringUpdate(
			updates,
			"errorCode",
			patch.ErrorCode,
		)
	}

	if patch.ErrorMsg != nil {
		updates = appendPaymentOptionalStringUpdate(
			updates,
			"errorMsg",
			patch.ErrorMsg,
		)
	}

	if len(updates) == 0 {
		return r.GetByPaymentID(
			ctx,
			paymentID,
		)
	}

	updates = append(
		updates,
		firestore.Update{
			Path:  "updatedAt",
			Value: time.Now().UTC(),
		},
	)

	_, err := documentReference.Update(
		ctx,
		updates,
	)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, paymentdom.ErrNotFound
		}

		return nil, err
	}

	return r.GetByPaymentID(
		ctx,
		paymentID,
	)
}

// ============================================================
// usecase.StripePaymentEventRepository
// ============================================================

// ApplyStripePaymentEvent atomically:
//
//  1. Deduplicates the Stripe event.
//  2. Reads and validates the current Payment.
//  3. Verifies the Stripe PaymentIntent ID.
//  4. Applies a valid status transition.
//  5. Acquires the post-paid marker if this is the first succeeded state.
//  6. Records the Stripe event as processed.
func (r *PaymentRepositoryFS) ApplyStripePaymentEvent(
	ctx context.Context,
	in usecase.ApplyStripePaymentEventInput,
) (*usecase.ApplyStripePaymentEventResult, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New(
			"firestore client is nil",
		)
	}

	in.EventID = strings.TrimSpace(in.EventID)
	in.PaymentID = strings.TrimSpace(in.PaymentID)
	in.StripePaymentIntentID = strings.TrimSpace(
		in.StripePaymentIntentID,
	)

	if in.EventID == "" {
		return nil,
			usecase.ErrPaymentStripeEventIDEmpty
	}

	if strings.Contains(in.EventID, "/") {
		return nil, fmt.Errorf(
			"payment: invalid Stripe event id %q",
			in.EventID,
		)
	}

	if in.PaymentID == "" {
		return nil, paymentdom.ErrInvalidPaymentID
	}

	if in.StripePaymentIntentID == "" {
		return nil,
			paymentdom.ErrInvalidStripePaymentIntent
	}

	if !paymentdom.IsValidStatus(in.Status) {
		return nil, paymentdom.ErrInvalidStatus
	}

	if in.OccurredAt.IsZero() {
		return nil,
			usecase.ErrPaymentStripeEventOccurredAtInvalid
	}

	in.OccurredAt = in.OccurredAt.UTC()
	in.ErrorType = normalizePaymentOptionalString(
		in.ErrorType,
	)
	in.ErrorCode = normalizePaymentOptionalString(
		in.ErrorCode,
	)
	in.ErrorMsg = normalizePaymentOptionalString(
		in.ErrorMsg,
	)

	paymentReference := r.col().Doc(
		in.PaymentID,
	)
	eventReference := r.stripeEventCol().Doc(
		in.EventID,
	)

	processedAt := time.Now().UTC()

	var result *usecase.ApplyStripePaymentEventResult

	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			transaction *firestore.Transaction,
		) error {
			// Read all required documents before any write.
			eventSnapshot, eventErr :=
				transaction.Get(eventReference)

			if eventErr != nil &&
				status.Code(eventErr) != codes.NotFound {
				return eventErr
			}

			paymentSnapshot, paymentErr :=
				transaction.Get(paymentReference)
			if paymentErr != nil {
				if status.Code(paymentErr) ==
					codes.NotFound {
					return paymentdom.ErrNotFound
				}

				return paymentErr
			}

			current, decodeErr :=
				docToPayment(paymentSnapshot)
			if decodeErr != nil {
				return decodeErr
			}

			// Duplicate Stripe event: successful no-op.
			if eventErr == nil &&
				eventSnapshot != nil &&
				eventSnapshot.Exists() {
				result =
					&usecase.ApplyStripePaymentEventResult{
						Payment:          &current,
						EventApplied:     false,
						StatusChanged:    false,
						PostPaidRequired: false,
					}

				return nil
			}

			if current.StripePaymentIntentID !=
				in.StripePaymentIntentID {
				return paymentdom.
					ErrInvalidStripePaymentIntent
			}

			transitionAllowed :=
				paymentStatusTransitionAllowed(
					current.Status,
					in.Status,
				)

			next := current
			statusChanged := false

			if transitionAllowed {
				statusChanged =
					current.Status != in.Status

				next.Status = in.Status

				switch in.Status {
				case paymentdom.StatusFailed,
					paymentdom.StatusCanceled:
					next.ErrorType = in.ErrorType
					next.ErrorCode = in.ErrorCode
					next.ErrorMsg = in.ErrorMsg

				default:
					// A non-error Stripe state clears stale error metadata.
					next.ErrorType = nil
					next.ErrorCode = nil
					next.ErrorMsg = nil
				}

				validated, validationErr :=
					paymentdom.New(
						next.PaymentID,
						next.PaymentMethodID,
						next.StripeCustomerID,
						next.StripePaymentMethodID,
						next.StripePaymentIntentID,
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

				next = validated
			}

			postPaidMarkerExists, markerErr :=
				paymentPostPaidMarkerExists(
					paymentSnapshot.Data(),
				)
			if markerErr != nil {
				return markerErr
			}

			postPaidRequired :=
				transitionAllowed &&
					next.Status ==
						paymentdom.StatusSucceeded &&
					in.Status ==
						paymentdom.StatusSucceeded &&
					!postPaidMarkerExists

			updates := make(
				[]firestore.Update,
				0,
				6,
			)

			if transitionAllowed {
				updates = append(
					updates,
					firestore.Update{
						Path:  "status",
						Value: string(next.Status),
					},
					firestore.Update{
						Path:  "updatedAt",
						Value: processedAt,
					},
				)

				updates =
					appendPaymentOptionalStringUpdate(
						updates,
						"errorType",
						next.ErrorType,
					)
				updates =
					appendPaymentOptionalStringUpdate(
						updates,
						"errorCode",
						next.ErrorCode,
					)
				updates =
					appendPaymentOptionalStringUpdate(
						updates,
						"errorMsg",
						next.ErrorMsg,
					)
			}

			if postPaidRequired {
				updates = append(
					updates,
					firestore.Update{
						Path:  "postPaidTriggeredAt",
						Value: processedAt,
					},
				)
			}

			if len(updates) > 0 {
				if updateErr := transaction.Update(
					paymentReference,
					updates,
				); updateErr != nil {
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

			if in.ErrorType != nil {
				eventData["errorType"] =
					*in.ErrorType
			}
			if in.ErrorCode != nil {
				eventData["errorCode"] =
					*in.ErrorCode
			}
			if in.ErrorMsg != nil {
				eventData["errorMsg"] =
					*in.ErrorMsg
			}

			if createErr := transaction.Create(
				eventReference,
				eventData,
			); createErr != nil {
				return createErr
			}

			result =
				&usecase.ApplyStripePaymentEventResult{
					Payment:          &next,
					EventApplied:     true,
					StatusChanged:    statusChanged,
					PostPaidRequired: postPaidRequired,
				}

			return nil
		},
	)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, paymentdom.ErrNotFound
		}

		return nil, err
	}

	if result == nil || result.Payment == nil {
		return nil,
			usecase.ErrPaymentStripeEventResultEmpty
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
func paymentStatusTransitionAllowed(
	current paymentdom.PaymentStatus,
	next paymentdom.PaymentStatus,
) bool {
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

func paymentToCreateData(
	payment paymentdom.Payment,
) map[string]any {
	data := map[string]any{
		"amount":                payment.Amount,
		"createdAt":             payment.CreatedAt,
		"paymentMethodId":       payment.PaymentMethodID,
		"status":                string(payment.Status),
		"stripeCustomerId":      payment.StripeCustomerID,
		"stripePaymentIntentId": payment.StripePaymentIntentID,
		"stripePaymentMethodId": payment.StripePaymentMethodID,
	}

	if payment.ErrorType != nil {
		data["errorType"] = *payment.ErrorType
	}

	if payment.ErrorCode != nil {
		data["errorCode"] = *payment.ErrorCode
	}

	if payment.ErrorMsg != nil {
		data["errorMsg"] = *payment.ErrorMsg
	}

	return data
}

func docToPayment(
	document *firestore.DocumentSnapshot,
) (paymentdom.Payment, error) {
	if document == nil {
		return paymentdom.Payment{},
			errors.New(
				"payment: document snapshot is nil",
			)
	}

	data := document.Data()
	if data == nil {
		return paymentdom.Payment{}, fmt.Errorf(
			"empty payment document: %s",
			document.Ref.ID,
		)
	}

	paymentID := document.Ref.ID

	paymentMethodID, err :=
		paymentRequiredString(
			data,
			"paymentMethodId",
		)
	if err != nil {
		return paymentdom.Payment{}, err
	}

	stripeCustomerID, err :=
		paymentRequiredString(
			data,
			"stripeCustomerId",
		)
	if err != nil {
		return paymentdom.Payment{}, err
	}

	stripePaymentMethodID, err :=
		paymentRequiredString(
			data,
			"stripePaymentMethodId",
		)
	if err != nil {
		return paymentdom.Payment{}, err
	}

	stripePaymentIntentID, err :=
		paymentRequiredString(
			data,
			"stripePaymentIntentId",
		)
	if err != nil {
		return paymentdom.Payment{}, err
	}

	amount, err := paymentRequiredInt(
		data,
		"amount",
	)
	if err != nil {
		return paymentdom.Payment{}, err
	}

	statusText, err :=
		paymentRequiredString(
			data,
			"status",
		)
	if err != nil {
		return paymentdom.Payment{}, err
	}

	createdAt, err :=
		paymentRequiredTime(
			data,
			"createdAt",
		)
	if err != nil {
		return paymentdom.Payment{}, err
	}

	errorType := paymentOptionalString(
		data,
		"errorType",
	)
	errorCode := paymentOptionalString(
		data,
		"errorCode",
	)
	errorMsg := paymentOptionalString(
		data,
		"errorMsg",
	)

	payment, err := paymentdom.New(
		paymentID,
		paymentMethodID,
		stripeCustomerID,
		stripePaymentMethodID,
		stripePaymentIntentID,
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

	return payment, nil
}

// ============================================================
// Firestore field helpers
// ============================================================

func paymentRequiredString(
	values map[string]any,
	key string,
) (string, error) {
	value, ok := values[key]
	if !ok || value == nil {
		return "", fmt.Errorf(
			"payment: missing %s",
			key,
		)
	}

	text, ok := value.(string)
	text = strings.TrimSpace(text)

	if !ok || text == "" {
		return "", fmt.Errorf(
			"payment: invalid %s",
			key,
		)
	}

	return text, nil
}

func paymentOptionalString(
	values map[string]any,
	key string,
) *string {
	value, ok := values[key]
	if !ok || value == nil {
		return nil
	}

	text, ok := value.(string)
	if !ok {
		return nil
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	return &text
}

func normalizePaymentOptionalString(
	value *string,
) *string {
	if value == nil {
		return nil
	}

	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return nil
	}

	return &normalized
}

func appendPaymentOptionalStringUpdate(
	updates []firestore.Update,
	path string,
	value *string,
) []firestore.Update {
	normalized :=
		normalizePaymentOptionalString(value)

	if normalized == nil {
		return append(
			updates,
			firestore.Update{
				Path:  path,
				Value: firestore.Delete,
			},
		)
	}

	return append(
		updates,
		firestore.Update{
			Path:  path,
			Value: *normalized,
		},
	)
}

func paymentRequiredInt(
	values map[string]any,
	key string,
) (int, error) {
	value, ok := values[key]
	if !ok || value == nil {
		return 0, fmt.Errorf(
			"payment: missing %s",
			key,
		)
	}

	number, ok := value.(int64)
	if !ok {
		return 0, fmt.Errorf(
			"payment: invalid %s",
			key,
		)
	}

	return int(number), nil
}

func paymentRequiredTime(
	values map[string]any,
	key string,
) (time.Time, error) {
	value, ok := values[key]
	if !ok || value == nil {
		return time.Time{}, fmt.Errorf(
			"payment: missing %s",
			key,
		)
	}

	timestamp, ok := value.(time.Time)
	if !ok || timestamp.IsZero() {
		return time.Time{}, fmt.Errorf(
			"payment: invalid %s",
			key,
		)
	}

	return timestamp.UTC(), nil
}

func paymentPostPaidMarkerExists(
	values map[string]any,
) (bool, error) {
	value, exists := values["postPaidTriggeredAt"]
	if !exists || value == nil {
		return false, nil
	}

	timestamp, ok := value.(time.Time)
	if !ok || timestamp.IsZero() {
		return false, errors.New(
			"payment: invalid postPaidTriggeredAt",
		)
	}

	return true, nil
}
