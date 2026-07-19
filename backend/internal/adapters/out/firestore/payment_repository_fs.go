// backend/internal/adapters/out/firestore/payment_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	paymentdom "narratives/internal/domain/payment"
)

// PaymentRepositoryFS is the Firestore-based implementation of
// payment.RepositoryPort.
//
// Current design:
// - payment document ID = paymentId
// - paymentId must be the same value as order.ID
// - paymentId is not stored as a document field
// - stripePaymentIntentId is required for every status, including pending
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

// ============================================================
// payment.RepositoryPort
// ============================================================

func (r *PaymentRepositoryFS) GetByPaymentID(
	ctx context.Context,
	paymentID string,
) (*paymentdom.Payment, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	if paymentID == "" {
		return nil, paymentdom.ErrNotFound
	}

	snap, err := r.col().Doc(paymentID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, paymentdom.ErrNotFound
		}

		return nil, err
	}

	payment, err := docToPayment(snap)
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
		return nil, errors.New("firestore client is nil")
	}

	if in.PaymentID == "" {
		return nil, paymentdom.ErrInvalidPaymentID
	}

	// Validate the complete domain entity before writing anything to
	// Firestore.
	//
	// In particular, StripePaymentIntentID must be non-empty for every
	// status, including pending. Performing this validation first prevents
	// an invalid document from being persisted before an error is returned.
	now := time.Now().UTC()

	payment, err := paymentdom.New(
		in.PaymentID,
		in.PaymentMethodID,
		in.StripeCustomerID,
		in.StripePaymentMethodID,
		in.StripePaymentIntentID,
		in.Amount,
		in.Status,
		in.ErrorType,
		in.ErrorCode,
		in.ErrorMsg,
		now,
	)
	if err != nil {
		return nil, err
	}

	docRef := r.col().Doc(payment.PaymentID)

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

	_, err = docRef.Create(ctx, data)
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
		return nil, errors.New("firestore client is nil")
	}

	if paymentID == "" {
		return nil, paymentdom.ErrNotFound
	}

	docRef := r.col().Doc(paymentID)
	updates := make([]firestore.Update, 0, 9)

	if patch.PaymentMethodID != nil {
		if *patch.PaymentMethodID == "" {
			return nil, paymentdom.ErrInvalidPaymentMethodID
		}

		updates = append(updates, firestore.Update{
			Path:  "paymentMethodId",
			Value: *patch.PaymentMethodID,
		})
	}

	if patch.StripeCustomerID != nil {
		if *patch.StripeCustomerID == "" {
			return nil, paymentdom.ErrInvalidStripeCustomerID
		}

		updates = append(updates, firestore.Update{
			Path:  "stripeCustomerId",
			Value: *patch.StripeCustomerID,
		})
	}

	if patch.StripePaymentMethodID != nil {
		if *patch.StripePaymentMethodID == "" {
			return nil, paymentdom.ErrInvalidStripePaymentMethod
		}

		updates = append(updates, firestore.Update{
			Path:  "stripePaymentMethodId",
			Value: *patch.StripePaymentMethodID,
		})
	}

	// nil means "not updated". A specified value must never be empty.
	if patch.StripePaymentIntentID != nil {
		if *patch.StripePaymentIntentID == "" {
			return nil, paymentdom.ErrInvalidStripePaymentIntent
		}

		updates = append(updates, firestore.Update{
			Path:  "stripePaymentIntentId",
			Value: *patch.StripePaymentIntentID,
		})
	}

	if patch.Amount != nil {
		if *patch.Amount < paymentdom.MinAmount ||
			(paymentdom.MaxAmount > 0 &&
				*patch.Amount > paymentdom.MaxAmount) {
			return nil, paymentdom.ErrInvalidAmount
		}

		updates = append(updates, firestore.Update{
			Path:  "amount",
			Value: *patch.Amount,
		})
	}

	if patch.Status != nil {
		if !paymentdom.IsValidStatus(*patch.Status) {
			return nil, paymentdom.ErrInvalidStatus
		}

		updates = append(updates, firestore.Update{
			Path:  "status",
			Value: string(*patch.Status),
		})
	}

	if patch.ErrorType != nil {
		if *patch.ErrorType == "" {
			updates = append(updates, firestore.Update{
				Path:  "errorType",
				Value: firestore.Delete,
			})
		} else {
			updates = append(updates, firestore.Update{
				Path:  "errorType",
				Value: *patch.ErrorType,
			})
		}
	}

	if patch.ErrorCode != nil {
		if *patch.ErrorCode == "" {
			updates = append(updates, firestore.Update{
				Path:  "errorCode",
				Value: firestore.Delete,
			})
		} else {
			updates = append(updates, firestore.Update{
				Path:  "errorCode",
				Value: *patch.ErrorCode,
			})
		}
	}

	if patch.ErrorMsg != nil {
		if *patch.ErrorMsg == "" {
			updates = append(updates, firestore.Update{
				Path:  "errorMsg",
				Value: firestore.Delete,
			})
		} else {
			updates = append(updates, firestore.Update{
				Path:  "errorMsg",
				Value: *patch.ErrorMsg,
			})
		}
	}

	if len(updates) == 0 {
		return r.GetByPaymentID(ctx, paymentID)
	}

	_, err := docRef.Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, paymentdom.ErrNotFound
		}

		return nil, err
	}

	// Reading through docToPayment validates the complete document after
	// applying the partial update.
	return r.GetByPaymentID(ctx, paymentID)
}

// ============================================================
// Helpers
// ============================================================

func docToPayment(
	doc *firestore.DocumentSnapshot,
) (paymentdom.Payment, error) {
	data := doc.Data()
	if data == nil {
		return paymentdom.Payment{}, fmt.Errorf(
			"empty payment document: %s",
			doc.Ref.ID,
		)
	}

	paymentID := doc.Ref.ID

	paymentMethodID, err := paymentRequiredString(
		data,
		"paymentMethodId",
	)
	if err != nil {
		return paymentdom.Payment{}, err
	}

	stripeCustomerID, err := paymentRequiredString(
		data,
		"stripeCustomerId",
	)
	if err != nil {
		return paymentdom.Payment{}, err
	}

	stripePaymentMethodID, err := paymentRequiredString(
		data,
		"stripePaymentMethodId",
	)
	if err != nil {
		return paymentdom.Payment{}, err
	}

	// stripePaymentIntentId is required regardless of payment status.
	stripePaymentIntentID, err := paymentRequiredString(
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

	statusText, err := paymentRequiredString(
		data,
		"status",
	)
	if err != nil {
		return paymentdom.Payment{}, err
	}

	createdAt, err := paymentRequiredTime(
		data,
		"createdAt",
	)
	if err != nil {
		return paymentdom.Payment{}, err
	}

	// Error information remains optional. These fields are unrelated to
	// the required stripePaymentIntentId invariant.
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
	if !ok || text == "" {
		return nil
	}

	return &text
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
