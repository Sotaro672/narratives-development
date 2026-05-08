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
// usecase.PaymentRepo
// ============================================================

func (r *PaymentRepositoryFS) GetByID(ctx context.Context, id string) (*paymentdom.Payment, error) {
	return r.GetByPaymentID(ctx, id)
}

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
		"paymentMethodId":       in.PaymentMethodID,
		"stripeCustomerId":      in.StripeCustomerID,
		"stripePaymentMethodId": in.StripePaymentMethodID,
		"stripePaymentIntentId": in.StripePaymentIntentID,
		"amount":                in.Amount,
		"status":                string(in.Status),
		"createdAt":             now,
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

func (r *PaymentRepositoryFS) Update(ctx context.Context, id string, patch paymentdom.UpdatePaymentInput) (*paymentdom.Payment, error) {
	return r.UpdateByPaymentID(ctx, id, patch)
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
	updates := make([]firestore.Update, 0, 8)

	setStringPtr := func(path string, p *string) {
		if p == nil {
			return
		}
		if *p == "" {
			updates = append(updates, firestore.Update{Path: path, Value: firestore.Delete})
			return
		}
		updates = append(updates, firestore.Update{Path: path, Value: *p})
	}

	setStatusPtr := func(path string, p *paymentdom.PaymentStatus) {
		if p == nil {
			return
		}
		updates = append(updates, firestore.Update{Path: path, Value: string(*p)})
	}

	if patch.PaymentMethodID != nil {
		setStringPtr("paymentMethodId", patch.PaymentMethodID)
	}
	if patch.StripeCustomerID != nil {
		setStringPtr("stripeCustomerId", patch.StripeCustomerID)
	}
	if patch.StripePaymentMethodID != nil {
		setStringPtr("stripePaymentMethodId", patch.StripePaymentMethodID)
	}
	if patch.StripePaymentIntentID != nil {
		setStringPtr("stripePaymentIntentId", patch.StripePaymentIntentID)
	}
	if patch.Amount != nil {
		updates = append(updates, firestore.Update{Path: "amount", Value: *patch.Amount})
	}
	if patch.Status != nil {
		setStatusPtr("status", patch.Status)
	}
	if patch.ErrorType != nil {
		setStringPtr("errorType", patch.ErrorType)
	}
	if patch.ErrorCode != nil {
		setStringPtr("errorCode", patch.ErrorCode)
	}
	if patch.ErrorMsg != nil {
		setStringPtr("errorMsg", patch.ErrorMsg)
	}

	if len(updates) == 0 {
		return r.GetByPaymentID(ctx, paymentID)
	}

	_, err := docRef.Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, paymentdom.ErrNotFound
		}
		if status.Code(err) == codes.AlreadyExists {
			return nil, paymentdom.ErrConflict
		}
		return nil, err
	}

	return r.GetByPaymentID(ctx, paymentID)
}

func (r *PaymentRepositoryFS) Delete(ctx context.Context, id string) error {
	return r.DeleteByPaymentID(ctx, id)
}

func (r *PaymentRepositoryFS) DeleteByPaymentID(ctx context.Context, paymentID string) error {
	if r == nil || r.Client == nil {
		return errors.New("firestore client is nil")
	}
	if paymentID == "" {
		return paymentdom.ErrNotFound
	}

	_, err := r.col().Doc(paymentID).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return paymentdom.ErrNotFound
		}
		return err
	}
	return nil
}

// Save stores the whole payment entity.
//
// entity.PaymentID is used as the Firestore payment document ID.
// paymentId itself is not stored as a document field.
func (r *PaymentRepositoryFS) Save(
	ctx context.Context,
	entity paymentdom.Payment,
	opts *paymentdom.SaveOptions,
) (paymentdom.Payment, error) {
	if r == nil || r.Client == nil {
		return paymentdom.Payment{}, errors.New("firestore client is nil")
	}
	if entity.PaymentID == "" {
		return paymentdom.Payment{}, paymentdom.ErrInvalidPaymentID
	}

	data := map[string]any{
		"paymentMethodId":       entity.PaymentMethodID,
		"stripeCustomerId":      entity.StripeCustomerID,
		"stripePaymentMethodId": entity.StripePaymentMethodID,
		"stripePaymentIntentId": entity.StripePaymentIntentID,
		"amount":                entity.Amount,
		"status":                string(entity.Status),
		"createdAt":             entity.CreatedAt.UTC(),
	}

	if entity.ErrorType != nil && *entity.ErrorType != "" {
		data["errorType"] = *entity.ErrorType
	}
	if entity.ErrorCode != nil && *entity.ErrorCode != "" {
		data["errorCode"] = *entity.ErrorCode
	}
	if entity.ErrorMsg != nil && *entity.ErrorMsg != "" {
		data["errorMsg"] = *entity.ErrorMsg
	}

	_, err := r.col().Doc(entity.PaymentID).Set(ctx, data)
	if err != nil {
		return paymentdom.Payment{}, err
	}

	return entity, nil
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
	paymentMethodID := getString(data, "paymentMethodId")
	stripeCustomerID := getString(data, "stripeCustomerId")
	stripePaymentMethodID := getString(data, "stripePaymentMethodId")
	stripePaymentIntentID := getString(data, "stripePaymentIntentId")
	amount := getInt(data, "amount")
	statusText := getString(data, "status")
	createdAt := getTime(data, "createdAt")
	errorType := getOptionalString(data, "errorType")
	errorCode := getOptionalString(data, "errorCode")
	errorMsg := getOptionalString(data, "errorMsg")

	if createdAt.IsZero() && !doc.CreateTime.IsZero() {
		createdAt = doc.CreateTime.UTC()
	}

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

func getString(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func getOptionalString(m map[string]any, key string) *string {
	s := getString(m, key)
	if s == "" {
		return nil
	}
	v := s
	return &v
}

func getInt(m map[string]any, key string) int {
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}

	switch n := v.(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

func getTime(m map[string]any, key string) time.Time {
	v, ok := m[key]
	if !ok || v == nil {
		return time.Time{}
	}

	t, ok := v.(time.Time)
	if !ok || t.IsZero() {
		return time.Time{}
	}
	return t.UTC()
}
