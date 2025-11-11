// backend/internal/adapters/out/firestore/tracking_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	trdom "narratives/internal/domain/tracking"
)

// =====================================================
// Firestore Tracking Repository
// implements usecase.TrackingRepo (minimal port)
// + additional helper methods for legacy usages
// =====================================================

type TrackingRepositoryFS struct {
	Client *firestore.Client
}

func NewTrackingRepositoryFS(client *firestore.Client) *TrackingRepositoryFS {
	return &TrackingRepositoryFS{Client: client}
}

func (r *TrackingRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("trackings")
}

// =====================================================
// Minimal TrackingRepo implementation
// (backend/internal/application/usecase/tracking_usecase.go)
// =====================================================

// GetByID implements TrackingRepo.GetByID.
func (r *TrackingRepositoryFS) GetByID(ctx context.Context, id string) (trdom.Tracking, error) {
	if r.Client == nil {
		return trdom.Tracking{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return trdom.Tracking{}, status.Error(codes.InvalidArgument, "tracking id is required")
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return trdom.Tracking{}, status.Error(codes.NotFound, "tracking not found")
		}
		return trdom.Tracking{}, err
	}

	t, err := docToTracking(snap)
	if err != nil {
		return trdom.Tracking{}, err
	}
	return t, nil
}

// Exists implements TrackingRepo.Exists.
func (r *TrackingRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
	if r.Client == nil {
		return false, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}

	_, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Create implements TrackingRepo.Create.
// If v.ID is empty, a new document ID is generated and set back to v.ID.
func (r *TrackingRepositoryFS) Create(ctx context.Context, v trdom.Tracking) (trdom.Tracking, error) {
	if r.Client == nil {
		return trdom.Tracking{}, errors.New("firestore client is nil")
	}

	id := strings.TrimSpace(v.ID)
	var ref *firestore.DocumentRef
	if id == "" {
		ref = r.col().NewDoc()
		v.ID = ref.ID
	} else {
		ref = r.col().Doc(id)
	}

	now := time.Now().UTC()
	if v.CreatedAt.IsZero() {
		v.CreatedAt = now
	}
	v.UpdatedAt = now

	data := map[string]any{
		"orderId":        strings.TrimSpace(v.OrderID),
		"carrier":        strings.TrimSpace(v.Carrier),
		"trackingNumber": strings.TrimSpace(v.TrackingNumber),
		"createdAt":      v.CreatedAt,
		"updatedAt":      v.UpdatedAt,
	}

	if v.SpecialInstructions != nil {
		if s := strings.TrimSpace(*v.SpecialInstructions); s != "" {
			data["specialInstructions"] = s
		}
	}

	if _, err := ref.Create(ctx, data); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return trdom.Tracking{}, status.Error(codes.AlreadyExists, "tracking already exists")
		}
		return trdom.Tracking{}, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return trdom.Tracking{}, err
	}

	created, err := docToTracking(snap)
	if err != nil {
		return trdom.Tracking{}, err
	}
	return created, nil
}

// Save implements TrackingRepo.Save as an upsert.
func (r *TrackingRepositoryFS) Save(ctx context.Context, v trdom.Tracking) (trdom.Tracking, error) {
	if r.Client == nil {
		return trdom.Tracking{}, errors.New("firestore client is nil")
	}

	id := strings.TrimSpace(v.ID)
	var ref *firestore.DocumentRef
	if id == "" {
		ref = r.col().NewDoc()
		v.ID = ref.ID
	} else {
		ref = r.col().Doc(id)
	}

	now := time.Now().UTC()
	if v.CreatedAt.IsZero() {
		v.CreatedAt = now
	}
	v.UpdatedAt = now

	data := map[string]any{
		"orderId":        strings.TrimSpace(v.OrderID),
		"carrier":        strings.TrimSpace(v.Carrier),
		"trackingNumber": strings.TrimSpace(v.TrackingNumber),
		"createdAt":      v.CreatedAt,
		"updatedAt":      v.UpdatedAt,
	}

	if v.SpecialInstructions != nil {
		if s := strings.TrimSpace(*v.SpecialInstructions); s != "" {
			data["specialInstructions"] = s
		} else {
			data["specialInstructions"] = firestore.Delete
		}
	}

	if _, err := ref.Set(ctx, data); err != nil {
		return trdom.Tracking{}, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return trdom.Tracking{}, err
	}
	saved, err := docToTracking(snap)
	if err != nil {
		return trdom.Tracking{}, err
	}
	return saved, nil
}

// Delete implements TrackingRepo.Delete.
func (r *TrackingRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return status.Error(codes.InvalidArgument, "tracking id is required")
	}

	ref := r.col().Doc(id)
	_, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return status.Error(codes.NotFound, "tracking not found")
		}
		return err
	}

	if _, err := ref.Delete(ctx); err != nil {
		return err
	}
	return nil
}

// =====================================================
// Additional helpers (legacy/administration)
// =====================================================

// GetAllTrackings: 全件取得（管理/一覧用途）
func (r *TrackingRepositoryFS) GetAllTrackings(ctx context.Context) ([]*trdom.Tracking, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	it := r.col().
		OrderBy("updatedAt", firestore.Desc).
		OrderBy(firestore.DocumentID, firestore.Desc).
		Documents(ctx)
	defer it.Stop()

	var out []*trdom.Tracking
	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		t, err := docToTracking(snap)
		if err != nil {
			return nil, err
		}
		tt := t
		out = append(out, &tt)
	}
	return out, nil
}

// GetTrackingsByOrderID: 注文IDで複数件取得
func (r *TrackingRepositoryFS) GetTrackingsByOrderID(ctx context.Context, orderID string) ([]*trdom.Tracking, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return []*trdom.Tracking{}, nil
	}

	q := r.col().
		Where("orderId", "==", orderID).
		OrderBy("updatedAt", firestore.Desc).
		OrderBy(firestore.DocumentID, firestore.Desc)

	it := q.Documents(ctx)
	defer it.Stop()

	var out []*trdom.Tracking
	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		t, err := docToTracking(snap)
		if err != nil {
			return nil, err
		}
		tt := t
		out = append(out, &tt)
	}
	return out, nil
}

// ResetTrackings: 全削除 (開発/テスト用途)
func (r *TrackingRepositoryFS) ResetTrackings(ctx context.Context) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	var snaps []*firestore.DocumentSnapshot
	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		snaps = append(snaps, snap)
	}

	const chunkSize = 400
	for i := 0; i < len(snaps); i += chunkSize {
		end := i + chunkSize
		if end > len(snaps) {
			end = len(snaps)
		}
		if err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			for _, s := range snaps[i:end] {
				if err := tx.Delete(s.Ref); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

// WithTx: Firestore用の簡易トランザクションヘルパー。
func (r *TrackingRepositoryFS) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}
	return r.Client.RunTransaction(ctx, func(txCtx context.Context, _ *firestore.Transaction) error {
		return fn(txCtx)
	})
}

// =====================================================
// Helpers (Firestore -> Domain)
// =====================================================

func docToTracking(doc *firestore.DocumentSnapshot) (trdom.Tracking, error) {
	data := doc.Data()
	if data == nil {
		return trdom.Tracking{}, status.Error(codes.NotFound, "tracking not found")
	}

	getStr := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := data[k].(string); ok {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}
	getOptStrPtr := func(keys ...string) *string {
		for _, k := range keys {
			if v, ok := data[k].(string); ok {
				s := strings.TrimSpace(v)
				if s != "" {
					return &s
				}
				return nil
			}
		}
		return nil
	}
	getTime := func(keys ...string) time.Time {
		for _, k := range keys {
			if v, ok := data[k].(time.Time); ok {
				return v.UTC()
			}
		}
		return time.Time{}
	}

	return trdom.Tracking{
		ID:                  strings.TrimSpace(doc.Ref.ID),
		OrderID:             getStr("orderId", "order_id"),
		Carrier:             getStr("carrier"),
		TrackingNumber:      getStr("trackingNumber", "tracking_number"),
		SpecialInstructions: getOptStrPtr("specialInstructions", "special_instructions"),
		CreatedAt:           getTime("createdAt", "created_at"),
		UpdatedAt:           getTime("updatedAt", "updated_at"),
	}, nil
}
