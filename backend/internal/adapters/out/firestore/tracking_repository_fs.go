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
// implements tracking.RepositoryPort
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
// RepositoryPort 実装
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

// GetTrackingByID: ID で1件取得
func (r *TrackingRepositoryFS) GetTrackingByID(ctx context.Context, id string) (*trdom.Tracking, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "tracking id is required")
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, status.Error(codes.NotFound, "tracking not found")
	}
	if err != nil {
		return nil, err
	}

	t, err := docToTracking(snap)
	if err != nil {
		return nil, err
	}
	return &t, nil
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

// CreateTracking: CreateTrackingInput から新規作成
func (r *TrackingRepositoryFS) CreateTracking(ctx context.Context, in trdom.CreateTrackingInput) (*trdom.Tracking, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	now := time.Now().UTC()

	ref := r.col().NewDoc()

	data := map[string]any{
		"orderId":        strings.TrimSpace(in.OrderID),
		"carrier":        strings.TrimSpace(in.Carrier),
		"trackingNumber": strings.TrimSpace(in.TrackingNumber),
		"createdAt":      now,
		"updatedAt":      now,
	}

	if in.SpecialInstructions != nil {
		if s := strings.TrimSpace(*in.SpecialInstructions); s != "" {
			data["specialInstructions"] = s
		}
	}

	if _, err := ref.Create(ctx, data); err != nil {
		// 呼び出し側で一意制約相当を気にする場合は status.Code(err) を見る想定
		if status.Code(err) == codes.AlreadyExists {
			return nil, err
		}
		return nil, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return nil, err
	}

	t, err := docToTracking(snap)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// UpdateTracking: 差分更新
func (r *TrackingRepositoryFS) UpdateTracking(ctx context.Context, id string, in trdom.UpdateTrackingInput) (*trdom.Tracking, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "tracking id is required")
	}

	ref := r.col().Doc(id)

	// 存在確認
	if _, err := ref.Get(ctx); status.Code(err) == codes.NotFound {
		return nil, status.Error(codes.NotFound, "tracking not found")
	} else if err != nil {
		return nil, err
	}

	var updates []firestore.Update

	if in.Carrier != nil {
		updates = append(updates, firestore.Update{
			Path:  "carrier",
			Value: strings.TrimSpace(*in.Carrier),
		})
	}
	if in.TrackingNumber != nil {
		updates = append(updates, firestore.Update{
			Path:  "trackingNumber",
			Value: strings.TrimSpace(*in.TrackingNumber),
		})
	}
	if in.SpecialInstructions != nil {
		v := strings.TrimSpace(*in.SpecialInstructions)
		if v == "" {
			updates = append(updates, firestore.Update{
				Path:  "specialInstructions",
				Value: firestore.Delete,
			})
		} else {
			updates = append(updates, firestore.Update{
				Path:  "specialInstructions",
				Value: v,
			})
		}
	}

	// 変更がない場合はそのまま返す
	if len(updates) == 0 {
		return r.GetTrackingByID(ctx, id)
	}

	// 常に updatedAt は更新
	updates = append(updates, firestore.Update{
		Path:  "updatedAt",
		Value: time.Now().UTC(),
	})

	if _, err := ref.Update(ctx, updates); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, status.Error(codes.NotFound, "tracking not found")
		}
		return nil, err
	}

	return r.GetTrackingByID(ctx, id)
}

// DeleteTracking: 1件削除
func (r *TrackingRepositoryFS) DeleteTracking(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return status.Error(codes.InvalidArgument, "tracking id is required")
	}

	ref := r.col().Doc(id)
	if _, err := ref.Get(ctx); status.Code(err) == codes.NotFound {
		return status.Error(codes.NotFound, "tracking not found")
	} else if err != nil {
		return err
	}

	if _, err := ref.Delete(ctx); err != nil {
		return err
	}
	return nil
}

// ResetTrackings: 全削除 (開発/テスト用途)
func (r *TrackingRepositoryFS) ResetTrackings(ctx context.Context) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	batch := r.Client.Batch()
	count := 0

	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		batch.Delete(doc.Ref)
		count++
		if count%400 == 0 {
			if _, err := batch.Commit(ctx); err != nil {
				return err
			}
			batch = r.Client.Batch()
		}
	}
	if count > 0 {
		if _, err := batch.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

// WithTx: Firestore用の簡易トランザクションヘルパー。
// （必要であれば RunTransaction に差し替え可能）
func (r *TrackingRepositoryFS) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}
	// 現状はそのまま実行（複数ドキュメントTxが必要になったら拡張）
	return fn(ctx)
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
