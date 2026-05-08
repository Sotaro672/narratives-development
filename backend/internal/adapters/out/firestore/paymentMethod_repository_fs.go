// backend/internal/adapters/out/firestore/paymentMethod_repository_fs.go
package firestore

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	pm "narratives/internal/domain/paymentMethod"

	"cloud.google.com/go/firestore"

	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PaymentMethodRepositoryFS is the Firestore implementation of paymentMethod.RepositoryPort.
type PaymentMethodRepositoryFS struct {
	Client *firestore.Client
}

func NewPaymentMethodRepositoryFS(client *firestore.Client) *PaymentMethodRepositoryFS {
	return &PaymentMethodRepositoryFS{Client: client}
}

func (r *PaymentMethodRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("paymentMethods")
}

// customerCol stores userId -> stripeCustomerId mapping for setup-intent flow.
func (r *PaymentMethodRepositoryFS) customerCol() *firestore.CollectionRef {
	return r.Client.Collection("paymentMethodCustomers")
}

// Compile-time check
var _ pm.RepositoryPort = (*PaymentMethodRepositoryFS)(nil)

// ========== Public API ==========

func (r *PaymentMethodRepositoryFS) GetByID(ctx context.Context, id string) (*pm.PaymentMethod, error) {
	if id == "" {
		return nil, pm.ErrNotFound
	}

	doc, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, pm.ErrNotFound
		}
		return nil, err
	}

	item, err := r.docToDomain(doc)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// GetByUser returns all payment methods for the user.
// 並び順: isDefault desc, updatedAt desc, docId desc
func (r *PaymentMethodRepositoryFS) GetByUser(ctx context.Context, userID string) ([]pm.PaymentMethod, error) {
	if userID == "" {
		return []pm.PaymentMethod{}, nil
	}

	q := r.col().
		Where("userId", "==", userID).
		OrderBy("isDefault", firestore.Desc).
		OrderBy("updatedAt", firestore.Desc).
		OrderBy(firestore.DocumentID, firestore.Desc)

	iter := q.Documents(ctx)
	defer iter.Stop()

	var items []pm.PaymentMethod
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}

		item, err := r.docToDomain(doc)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

func (r *PaymentMethodRepositoryFS) GetDefaultByUser(ctx context.Context, userID string) (*pm.PaymentMethod, error) {
	if userID == "" {
		return nil, pm.ErrNotFound
	}

	iter := r.col().
		Where("userId", "==", userID).
		Where("isDefault", "==", true).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return nil, pm.ErrNotFound
		}
		return nil, err
	}

	item, err := r.docToDomain(doc)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *PaymentMethodRepositoryFS) GetByStripePaymentMethodID(ctx context.Context, stripePaymentMethodID string) (*pm.PaymentMethod, error) {
	if stripePaymentMethodID == "" {
		return nil, pm.ErrNotFound
	}

	iter := r.col().
		Where("stripePaymentMethodId", "==", stripePaymentMethodID).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return nil, pm.ErrNotFound
		}
		return nil, err
	}

	item, err := r.docToDomain(doc)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// GetStripeCustomerIDByUser returns mapped stripe customer id for the user.
func (r *PaymentMethodRepositoryFS) GetStripeCustomerIDByUser(ctx context.Context, userID string) (string, error) {
	if userID == "" {
		return "", pm.ErrInvalidUserID
	}

	doc, err := r.customerCol().Doc(userID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", pm.ErrNotFound
		}
		return "", err
	}

	var raw struct {
		UserID           string    `firestore:"userId"`
		StripeCustomerID string    `firestore:"stripeCustomerId"`
		CreatedAt        time.Time `firestore:"createdAt"`
		UpdatedAt        time.Time `firestore:"updatedAt"`
	}
	if err := doc.DataTo(&raw); err != nil {
		return "", err
	}
	if raw.UserID == "" || raw.StripeCustomerID == "" {
		return "", pm.ErrNotFound
	}
	return raw.StripeCustomerID, nil
}

// SaveStripeCustomerIDByUser upserts userId -> stripeCustomerId mapping.
func (r *PaymentMethodRepositoryFS) SaveStripeCustomerIDByUser(ctx context.Context, userID string, stripeCustomerID string) error {
	if userID == "" {
		return pm.ErrInvalidUserID
	}
	if stripeCustomerID == "" {
		return pm.ErrInvalidStripeCustomerID
	}

	now := time.Now().UTC()
	ref := r.customerCol().Doc(userID)

	_, err := ref.Set(ctx, map[string]any{
		"userId":           userID,
		"stripeCustomerId": stripeCustomerID,
		"updatedAt":        now,
		"createdAt":        now,
	}, firestore.MergeAll)
	return err
}

// Create implements RepositoryPort.Create.
// docId はランダム。
// isDefault=true の作成時は、同 user の既定 paymentMethod を先に解除します。
func (r *PaymentMethodRepositoryFS) Create(ctx context.Context, in pm.CreatePaymentMethodInput) (*pm.PaymentMethod, error) {
	now := time.Now().UTC()

	if in.UserID == "" {
		return nil, pm.ErrInvalidUserID
	}

	createdAt := now
	if in.CreatedAt != nil && !in.CreatedAt.IsZero() {
		createdAt = in.CreatedAt.UTC()
	}

	updatedAt := createdAt
	if in.UpdatedAt != nil && !in.UpdatedAt.IsZero() {
		updatedAt = in.UpdatedAt.UTC()
	}

	docID, err := newRandomPaymentMethodDocID(24)
	if err != nil {
		return nil, err
	}

	item, err := pm.New(
		docID,
		in.UserID,
		in.StripeCustomerID,
		in.StripePaymentMethodID,
		in.Brand,
		in.Last4,
		in.ExpMonth,
		in.ExpYear,
		in.CardholderName,
		in.IsDefault,
		createdAt,
		updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if in.IsDefault {
		if err := r.ClearDefaultByUser(ctx, in.UserID); err != nil {
			return nil, err
		}
	}

	ref := r.col().Doc(docID)
	if _, err := ref.Create(ctx, r.domainToDocData(item)); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil, pm.ErrConflict
		}
		return nil, err
	}

	if err := r.SaveStripeCustomerIDByUser(ctx, in.UserID, in.StripeCustomerID); err != nil {
		return nil, err
	}

	return &item, nil
}

func (r *PaymentMethodRepositoryFS) Delete(ctx context.Context, id string) error {
	if id == "" {
		return pm.ErrNotFound
	}

	ref := r.col().Doc(id)
	if _, err := ref.Get(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return pm.ErrNotFound
		}
		return err
	}

	_, err := ref.Delete(ctx)
	return err
}

func (r *PaymentMethodRepositoryFS) ClearDefaultByUser(ctx context.Context, userID string) error {
	if userID == "" {
		return pm.ErrInvalidUserID
	}

	iter := r.col().
		Where("userId", "==", userID).
		Where("isDefault", "==", true).
		Documents(ctx)
	defer iter.Stop()

	batch := r.Client.Batch()
	count := 0
	now := time.Now().UTC()

	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}

		batch.Update(doc.Ref, []firestore.Update{
			{Path: "isDefault", Value: false},
			{Path: "updatedAt", Value: now},
		})
		count++
	}

	if count == 0 {
		return nil
	}

	_, err := batch.Commit(ctx)
	return err
}

func (r *PaymentMethodRepositoryFS) SetDefault(ctx context.Context, id string, userID string, updatedAt time.Time) (*pm.PaymentMethod, error) {
	if id == "" {
		return nil, pm.ErrNotFound
	}
	if userID == "" {
		return nil, pm.ErrInvalidUserID
	}

	ref := r.col().Doc(id)

	doc, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, pm.ErrNotFound
		}
		return nil, err
	}

	current, err := r.docToDomain(doc)
	if err != nil {
		return nil, err
	}
	if current.UserID != userID {
		return nil, pm.ErrNotFound
	}

	now := updatedAt.UTC()
	if updatedAt.IsZero() {
		now = time.Now().UTC()
	}

	if err := r.ClearDefaultByUser(ctx, userID); err != nil {
		return nil, err
	}

	if _, err := ref.Update(ctx, []firestore.Update{
		{Path: "isDefault", Value: true},
		{Path: "updatedAt", Value: now},
	}); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, pm.ErrNotFound
		}
		return nil, err
	}

	updatedDoc, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, pm.ErrNotFound
		}
		return nil, err
	}

	item, err := r.docToDomain(updatedDoc)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// ========== Helpers ==========

func (r *PaymentMethodRepositoryFS) docToDomain(doc *firestore.DocumentSnapshot) (pm.PaymentMethod, error) {
	var raw struct {
		UserID                string    `firestore:"userId"`
		StripeCustomerID      string    `firestore:"stripeCustomerId"`
		StripePaymentMethodID string    `firestore:"stripePaymentMethodId"`
		Brand                 string    `firestore:"brand"`
		Last4                 string    `firestore:"last4"`
		ExpMonth              int       `firestore:"expMonth"`
		ExpYear               int       `firestore:"expYear"`
		CardholderName        string    `firestore:"cardholderName"`
		IsDefault             bool      `firestore:"isDefault"`
		CreatedAt             time.Time `firestore:"createdAt"`
		UpdatedAt             time.Time `firestore:"updatedAt"`
	}

	if err := doc.DataTo(&raw); err != nil {
		return pm.PaymentMethod{}, err
	}

	createdAt := raw.CreatedAt.UTC()
	updatedAt := raw.UpdatedAt.UTC()

	if createdAt.IsZero() {
		createdAt = updatedAt
	}
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}
	if createdAt.IsZero() && updatedAt.IsZero() {
		now := time.Now().UTC()
		createdAt = now
		updatedAt = now
	}
	if updatedAt.Before(createdAt) {
		updatedAt = createdAt
	}

	return pm.New(
		doc.Ref.ID,
		raw.UserID,
		raw.StripeCustomerID,
		raw.StripePaymentMethodID,
		raw.Brand,
		raw.Last4,
		raw.ExpMonth,
		raw.ExpYear,
		raw.CardholderName,
		raw.IsDefault,
		createdAt,
		updatedAt,
	)
}

func (r *PaymentMethodRepositoryFS) domainToDocData(item pm.PaymentMethod) map[string]any {
	return map[string]any{
		"userId":                item.UserID,
		"stripeCustomerId":      item.StripeCustomerID,
		"stripePaymentMethodId": item.StripePaymentMethodID,
		"brand":                 item.Brand,
		"last4":                 item.Last4,
		"expMonth":              item.ExpMonth,
		"expYear":               item.ExpYear,
		"cardholderName":        item.CardholderName,
		"isDefault":             item.IsDefault,
		"createdAt":             item.CreatedAt.UTC(),
		"updatedAt":             item.UpdatedAt.UTC(),
	}
}

// newRandomPaymentMethodDocID returns URL-safe random id (no padding) with nBytes entropy.
func newRandomPaymentMethodDocID(nBytes int) (string, error) {
	if nBytes <= 0 {
		nBytes = 24
	}
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
