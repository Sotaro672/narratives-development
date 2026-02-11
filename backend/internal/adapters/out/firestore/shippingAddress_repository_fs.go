// backend/internal/adapters/out/firestore/shippingAddress_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	shipaddrdom "narratives/internal/domain/shippingAddress"
)

// ============================================================
// Firestore ShippingAddress Repository
// ============================================================
//
// コレクション: shippingAddresses
// ドキュメントID: ShippingAddress.ID
// - repo は docID を自動採番しない（usecase が UUID を採番して渡す）
//
// 実装対象（usecase port 準拠）:
// - GetByID / Exists / ListByUserID / Create / Update / Delete
//
// Firestore fields（domain の json tag に合わせる）:
// - userId, zipCode, state, city, street, street2, country, createdAt, updatedAt
// ============================================================

type ShippingAddressRepositoryFS struct {
	Client *firestore.Client
}

func NewShippingAddressRepositoryFS(client *firestore.Client) *ShippingAddressRepositoryFS {
	return &ShippingAddressRepositoryFS{Client: client}
}

func (r *ShippingAddressRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("shippingAddresses")
}

// --------------------
// Read
// --------------------

func (r *ShippingAddressRepositoryFS) GetByID(ctx context.Context, id string) (*shipaddrdom.ShippingAddress, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, shipaddrdom.ErrInvalidID
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, shipaddrdom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	v, err := docToShippingAddress(snap)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *ShippingAddressRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
	if r == nil || r.Client == nil {
		return false, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}

	_, err := r.col().Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// ListByUserID returns all shipping addresses for the given userId.
// ✅ 設計2: docID は UUID のまま（1ユーザー=複数住所）なので、userId でクエリする。
func (r *ShippingAddressRepositoryFS) ListByUserID(ctx context.Context, userID string) ([]shipaddrdom.ShippingAddress, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, shipaddrdom.ErrInvalidUserID
	}

	// updatedAt の降順（新しい順）
	// ※ Where + OrderBy は環境によって複合インデックスが必要になる場合があります。
	q := r.col().
		Where("userId", "==", userID).
		OrderBy("updatedAt", firestore.Desc)

	snaps, err := q.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	res := make([]shipaddrdom.ShippingAddress, 0, len(snaps))
	for _, s := range snaps {
		v, derr := docToShippingAddress(s)
		if derr != nil {
			return nil, derr
		}
		res = append(res, v)
	}

	return res, nil
}

// --------------------
// Write
// --------------------

// Create creates shippingAddresses/{id}. If already exists -> ErrConflict.
func (r *ShippingAddressRepositoryFS) Create(ctx context.Context, v shipaddrdom.ShippingAddress) (*shipaddrdom.ShippingAddress, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	id := strings.TrimSpace(v.ID)
	if id == "" {
		return nil, shipaddrdom.ErrInvalidID
	}

	now := time.Now().UTC()
	if v.CreatedAt.IsZero() {
		v.CreatedAt = now
	}
	if v.UpdatedAt.IsZero() {
		v.UpdatedAt = v.CreatedAt
	}

	v.ID = id

	if strings.TrimSpace(v.Country) == "" {
		v.Country = "JP"
	}

	ref := r.col().Doc(id)
	if _, err := ref.Create(ctx, shippingAddressToDocData(v)); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil, shipaddrdom.ErrConflict
		}
		return nil, err
	}

	return r.GetByID(ctx, id)
}

// Update updates shippingAddresses/{id}. If not exists -> ErrNotFound.
// - createdAt は既存値を保持
// - updatedAt は v.UpdatedAt が妥当ならそれ、無ければ now
func (r *ShippingAddressRepositoryFS) Update(ctx context.Context, v shipaddrdom.ShippingAddress) (*shipaddrdom.ShippingAddress, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	id := strings.TrimSpace(v.ID)
	if id == "" {
		return nil, shipaddrdom.ErrInvalidID
	}

	ref := r.col().Doc(id)

	snap, err := ref.Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, shipaddrdom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	current, err := docToShippingAddress(snap)
	if err != nil {
		return nil, err
	}

	createdAt := current.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	updatedAt := time.Now().UTC()
	if !v.UpdatedAt.IsZero() && !v.UpdatedAt.Before(createdAt) {
		updatedAt = v.UpdatedAt.UTC()
	}

	country := strings.TrimSpace(v.Country)
	if country == "" {
		country = strings.TrimSpace(current.Country)
	}
	if country == "" {
		country = "JP"
	}

	next := shipaddrdom.ShippingAddress{
		ID:        id,
		UserID:    strings.TrimSpace(v.UserID),
		ZipCode:   strings.TrimSpace(v.ZipCode),
		State:     strings.TrimSpace(v.State),
		City:      strings.TrimSpace(v.City),
		Street:    strings.TrimSpace(v.Street),
		Street2:   strings.TrimSpace(v.Street2),
		Country:   country,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	if _, err := ref.Set(ctx, shippingAddressToDocData(next)); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, shipaddrdom.ErrNotFound
		}
		return nil, err
	}

	return r.GetByID(ctx, id)
}

func (r *ShippingAddressRepositoryFS) Delete(ctx context.Context, id string) error {
	if r == nil || r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return shipaddrdom.ErrInvalidID
	}

	ref := r.col().Doc(id)

	_, err := ref.Get(ctx)
	if status.Code(err) == codes.NotFound {
		return shipaddrdom.ErrNotFound
	}
	if err != nil {
		return err
	}

	_, err = ref.Delete(ctx)
	return err
}

// --------------------
// Mapping
// --------------------

func docToShippingAddress(doc *firestore.DocumentSnapshot) (shipaddrdom.ShippingAddress, error) {
	data := doc.Data()
	if data == nil {
		return shipaddrdom.ShippingAddress{}, shipaddrdom.ErrNotFound
	}

	getStr := func(key string) string {
		v, ok := data[key]
		if !ok {
			return ""
		}
		s, ok := v.(string)
		if !ok {
			return ""
		}
		return strings.TrimSpace(s)
	}

	getTime := func(key string) time.Time {
		v, ok := data[key]
		if !ok {
			return time.Time{}
		}
		t, ok := v.(time.Time)
		if !ok {
			return time.Time{}
		}
		return t.UTC()
	}

	return shipaddrdom.ShippingAddress{
		ID:        strings.TrimSpace(doc.Ref.ID),
		UserID:    getStr("userId"),
		ZipCode:   getStr("zipCode"),
		State:     getStr("state"),
		City:      getStr("city"),
		Street:    getStr("street"),
		Street2:   getStr("street2"),
		Country:   getStr("country"),
		CreatedAt: getTime("createdAt"),
		UpdatedAt: getTime("updatedAt"),
	}, nil
}

func shippingAddressToDocData(v shipaddrdom.ShippingAddress) map[string]any {
	data := map[string]any{
		"userId":    strings.TrimSpace(v.UserID),
		"zipCode":   strings.TrimSpace(v.ZipCode),
		"state":     strings.TrimSpace(v.State),
		"city":      strings.TrimSpace(v.City),
		"street":    strings.TrimSpace(v.Street),
		"street2":   strings.TrimSpace(v.Street2),
		"country":   strings.TrimSpace(v.Country),
		"createdAt": v.CreatedAt.UTC(),
		"updatedAt": v.UpdatedAt.UTC(),
	}

	if v.CreatedAt.IsZero() {
		delete(data, "createdAt")
	}
	if v.UpdatedAt.IsZero() {
		delete(data, "updatedAt")
	}

	return data
}
