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

// Firestore-based implementation of Payment repository.
//
// current design:
// - payment docId = paymentId
// - paymentId must be the same value as order.ID
// - paymentId is NOT stored as a document field
type PaymentRepositoryFS struct {
	Client *firestore.Client
}

func NewPaymentRepositoryFS(client *firestore.Client) *PaymentRepositoryFS {
	return &PaymentRepositoryFS{Client: client}
}

func (r *PaymentRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("payments")
}

// ============================================================
// payment.RepositoryPort
// ============================================================

func (r *PaymentRepositoryFS) GetByPaymentID(ctx context.Context, paymentID string) (*paymentdom.Payment, error) {
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

	p, err := docToPayment(snap)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (r *PaymentRepositoryFS) Create(ctx context.Context, in paymentdom.CreatePaymentInput) (*paymentdom.Payment, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	if in.PaymentID == "" {
		return nil, paymentdom.ErrInvalidPaymentID
	}

	now := time.Now().UTC()
	docRef := r.col().Doc(in.PaymentID)

	data := map[string]any{
		"amount":                in.Amount,
		"createdAt":             now,
		"paymentMethodId":       in.PaymentMethodID,
		"status":                string(in.Status),
		"stripeCustomerId":      in.StripeCustomerID,
		"stripePaymentIntentId": in.StripePaymentIntentID,
		"stripePaymentMethodId": in.StripePaymentMethodID,
	}

	if in.ErrorType != nil && *in.ErrorType != "" {
		data["errorType"] = *in.ErrorType
	}
	if in.ErrorCode != nil && *in.ErrorCode != "" {
		data["errorCode"] = *in.ErrorCode
	}
	if in.ErrorMsg != nil && *in.ErrorMsg != "" {
		data["errorMsg"] = *in.ErrorMsg
	}

	_, err := docRef.Create(ctx, data)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil, paymentdom.ErrConflict
		}
		return nil, err
	}

	p, err := paymentdom.New(
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

	return &p, nil
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
		if *patch.Amount < paymentdom.MinAmount || (paymentdom.MaxAmount > 0 && *patch.Amount > paymentdom.MaxAmount) {
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

	return r.GetByPaymentID(ctx, paymentID)
}

// ============================================================
// Helpers
// ============================================================

func docToPayment(doc *firestore.DocumentSnapshot) (paymentdom.Payment, error) {
	data := doc.Data()
	if data == nil {
		return paymentdom.Payment{}, fmt.Errorf("empty payment document: %s", doc.Ref.ID)
	}

	paymentID := doc.Ref.ID

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

	amount, err := paymentRequiredInt(data, "amount")
	if err != nil {
		return paymentdom.Payment{}, err
	}

	statusText, err := paymentRequiredString(data, "status")
	if err != nil {
		return paymentdom.Payment{}, err
	}

	createdAt, err := paymentRequiredTime(data, "createdAt")
	if err != nil {
		return paymentdom.Payment{}, err
	}

	errorType := paymentOptionalString(data, "errorType")
	errorCode := paymentOptionalString(data, "errorCode")
	errorMsg := paymentOptionalString(data, "errorMsg")

	p, err := paymentdom.New(
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

	return p, nil
}

func paymentRequiredString(m map[string]any, key string) (string, error) {
	v, ok := m[key]
	if !ok || v == nil {
		return "", fmt.Errorf("payment: missing %s", key)
	}

	s, ok := v.(string)
	if !ok || s == "" {
		return "", fmt.Errorf("payment: invalid %s", key)
	}

	return s, nil
}

func paymentOptionalString(m map[string]any, key string) *string {
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}

	s, ok := v.(string)
	if !ok || s == "" {
		return nil
	}

	return &s
}

func paymentRequiredInt(m map[string]any, key string) (int, error) {
	v, ok := m[key]
	if !ok || v == nil {
		return 0, fmt.Errorf("payment: missing %s", key)
	}

	n, ok := v.(int64)
	if !ok {
		return 0, fmt.Errorf("payment: invalid %s", key)
	}

	return int(n), nil
}

func paymentRequiredTime(m map[string]any, key string) (time.Time, error) {
	v, ok := m[key]
	if !ok || v == nil {
		return time.Time{}, fmt.Errorf("payment: missing %s", key)
	}

	t, ok := v.(time.Time)
	if !ok || t.IsZero() {
		return time.Time{}, fmt.Errorf("payment: invalid %s", key)
	}

	return t.UTC(), nil
}
