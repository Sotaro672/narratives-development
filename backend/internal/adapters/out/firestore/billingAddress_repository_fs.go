// backend/internal/adapters/out/firestore/billingAddress_repository_fs.go
package firestore

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	badom "narratives/internal/domain/billingAddress"
)

// BillingAddressRepositoryFS is the Firestore implementation of billingAddress.RepositoryPort.
type BillingAddressRepositoryFS struct {
	Client *firestore.Client
}

func NewBillingAddressRepositoryFS(client *firestore.Client) *BillingAddressRepositoryFS {
	return &BillingAddressRepositoryFS{Client: client}
}

func (r *BillingAddressRepositoryFS) col() *firestore.CollectionRef {
	// ✅ shippingAddresses と同様の命名に寄せる（camelCase）
	// 期待値: billingAddresses コレクション
	return r.Client.Collection("billingAddresses")
}

// Compile-time check
var _ badom.RepositoryPort = (*BillingAddressRepositoryFS)(nil)

// ========== Public API ==========

func (r *BillingAddressRepositoryFS) GetByID(ctx context.Context, id string) (*badom.BillingAddress, error) {
	if id == "" {
		return nil, badom.ErrNotFound
	}

	doc, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, badom.ErrNotFound
		}
		return nil, err
	}

	ba, err := r.docToDomain(doc)
	if err != nil {
		return nil, err
	}
	return &ba, nil
}

// ✅ userId フィールドで引く（1ユーザー複数レコード対応）
// ✅ sort/filter/page などは Port から削除したため、この関数のみ残す
func (r *BillingAddressRepositoryFS) GetByUser(ctx context.Context, userID string) ([]badom.BillingAddress, error) {
	uid := userID
	if uid == "" {
		return []badom.BillingAddress{}, nil
	}

	// updatedAt desc, docId desc（安定化）
	q := r.col().
		Where("userId", "==", uid).
		OrderBy("updatedAt", firestore.Desc).
		OrderBy(firestore.DocumentID, firestore.Desc)

	iter := q.Documents(ctx)
	defer iter.Stop()

	var items []badom.BillingAddress
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		ba, err := r.docToDomain(doc)
		if err != nil {
			return nil, err
		}
		items = append(items, ba)
	}
	return items, nil
}

// Create implements RepositoryPort.Create.
// ✅ docId はランダム
// ✅ userId は in.UserID（= middleware userAuth で確定した uid をハンドラがセットして渡す）
func (r *BillingAddressRepositoryFS) Create(ctx context.Context, in badom.CreateBillingAddressInput) (*badom.BillingAddress, error) {
	now := time.Now().UTC()

	uid := in.UserID
	if uid == "" {
		return nil, badom.ErrInvalidUserID
	}

	createdAt := now
	if in.CreatedAt != nil && !in.CreatedAt.IsZero() {
		createdAt = in.CreatedAt.UTC()
	}
	updatedAt := createdAt
	if in.UpdatedAt != nil && !in.UpdatedAt.IsZero() {
		updatedAt = in.UpdatedAt.UTC()
	}

	docID, err := newRandomDocID(24)
	if err != nil {
		return nil, err
	}

	ref := r.col().Doc(docID)

	ba := badom.BillingAddress{
		ID:             docID,
		UserID:         uid,
		CardNumber:     in.CardNumber,
		CardholderName: in.CardholderName,
		CVC:            in.CVC,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}

	if _, err := ref.Create(ctx, r.domainToDocData(ba)); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil, badom.ErrConflict
		}
		return nil, err
	}

	return &ba, nil
}

// Update: docId はそのまま、userId は更新しない（anti-spoof）
func (r *BillingAddressRepositoryFS) Update(ctx context.Context, id string, in badom.UpdateBillingAddressInput) (*badom.BillingAddress, error) {
	if id == "" {
		return nil, badom.ErrNotFound
	}

	ref := r.col().Doc(id)

	// 存在確認
	if _, err := ref.Get(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, badom.ErrNotFound
		}
		return nil, err
	}

	var updates []firestore.Update

	if in.CardNumber != nil {
		updates = append(updates, firestore.Update{
			Path:  "cardNumber",
			Value: *in.CardNumber,
		})
	}
	if in.CardholderName != nil {
		updates = append(updates, firestore.Update{
			Path:  "cardholderName",
			Value: *in.CardholderName,
		})
	}
	if in.CVC != nil {
		updates = append(updates, firestore.Update{
			Path:  "cvc",
			Value: *in.CVC,
		})
	}

	// updatedAt
	if in.UpdatedAt != nil && !in.UpdatedAt.IsZero() {
		updates = append(updates, firestore.Update{
			Path:  "updatedAt",
			Value: in.UpdatedAt.UTC(),
		})
	} else {
		updates = append(updates, firestore.Update{
			Path:  "updatedAt",
			Value: time.Now().UTC(),
		})
	}

	if _, err := ref.Update(ctx, updates); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, badom.ErrNotFound
		}
		return nil, err
	}

	doc, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, badom.ErrNotFound
		}
		return nil, err
	}

	ba, err := r.docToDomain(doc)
	if err != nil {
		return nil, err
	}
	return &ba, nil
}

func (r *BillingAddressRepositoryFS) Delete(ctx context.Context, id string) error {
	if id == "" {
		return badom.ErrNotFound
	}

	ref := r.col().Doc(id)
	if _, err := ref.Get(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return badom.ErrNotFound
		}
		return err
	}
	_, err := ref.Delete(ctx)
	return err
}

// ========== Helpers ==========

func (r *BillingAddressRepositoryFS) docToDomain(doc *firestore.DocumentSnapshot) (badom.BillingAddress, error) {
	var raw struct {
		UserID         string    `firestore:"userId"`
		CardNumber     string    `firestore:"cardNumber"`
		CardholderName string    `firestore:"cardholderName"`
		CVC            string    `firestore:"cvc"`
		CreatedAt      time.Time `firestore:"createdAt"`
		UpdatedAt      time.Time `firestore:"updatedAt"`
	}

	if err := doc.DataTo(&raw); err != nil {
		return badom.BillingAddress{}, err
	}

	return badom.BillingAddress{
		ID:             doc.Ref.ID,
		UserID:         raw.UserID,
		CardNumber:     raw.CardNumber,
		CardholderName: raw.CardholderName,
		CVC:            raw.CVC,
		CreatedAt:      raw.CreatedAt.UTC(),
		UpdatedAt:      raw.UpdatedAt.UTC(),
	}, nil
}

func (r *BillingAddressRepositoryFS) domainToDocData(ba badom.BillingAddress) map[string]any {
	return map[string]any{
		"userId":         ba.UserID,
		"cardNumber":     ba.CardNumber,
		"cardholderName": ba.CardholderName,
		"cvc":            ba.CVC,
		"createdAt":      ba.CreatedAt.UTC(),
		"updatedAt":      ba.UpdatedAt.UTC(),
	}
}

// newRandomDocID returns URL-safe random id (no padding) with nBytes entropy.
func newRandomDocID(nBytes int) (string, error) {
	if nBytes <= 0 {
		nBytes = 24
	}
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
