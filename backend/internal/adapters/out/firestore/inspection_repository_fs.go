// backend/internal/adapters/out/firestore/inspection_repository_fs.go
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

	fscommon "narratives/internal/adapters/out/firestore/common"
	inspectiondom "narratives/internal/domain/inspection"
)

// ------------------------------------------------------------
// InspectionRepositoryFS
// ------------------------------------------------------------

type InspectionRepositoryFS struct {
	Client *firestore.Client
}

func NewInspectionRepositoryFS(client *firestore.Client) *InspectionRepositoryFS {
	return &InspectionRepositoryFS{Client: client}
}

func (r *InspectionRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("inspections")
}

// Create: inspections/{productionId} を新規作成
func (r *InspectionRepositoryFS) Create(
	ctx context.Context,
	v inspectiondom.InspectionBatch,
) (inspectiondom.InspectionBatch, error) {

	if r == nil || r.Client == nil {
		return inspectiondom.InspectionBatch{}, errors.New("firestore client is nil")
	}

	pid := strings.TrimSpace(v.ProductionID)
	if pid == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductionID
	}

	docRef := r.col().Doc(pid)
	data := inspectionBatchToDoc(v)

	_, err := docRef.Create(ctx, data)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return inspectiondom.InspectionBatch{}, err
		}
		return inspectiondom.InspectionBatch{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}
	return docToInspectionBatch(snap)
}

// GetByProductionID: inspections/{productionId} を取得
func (r *InspectionRepositoryFS) GetByProductionID(
	ctx context.Context,
	productionID string,
) (inspectiondom.InspectionBatch, error) {

	if r == nil || r.Client == nil {
		return inspectiondom.InspectionBatch{}, errors.New("firestore client is nil")
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductionID
	}

	snap, err := r.col().Doc(pid).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return inspectiondom.InspectionBatch{}, inspectiondom.ErrNotFound
		}
		return inspectiondom.InspectionBatch{}, err
	}

	return docToInspectionBatch(snap)
}

// ListByProductionID: 複数 ID の inspections をまとめて取得
func (r *InspectionRepositoryFS) ListByProductionID(
	ctx context.Context,
	productionIDs []string,
) ([]inspectiondom.InspectionBatch, error) {

	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	if len(productionIDs) == 0 {
		return []inspectiondom.InspectionBatch{}, nil
	}

	uniq := make(map[string]struct{}, len(productionIDs))
	ids := make([]string, 0, len(productionIDs))
	for _, id := range productionIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := uniq[id]; ok {
			continue
		}
		uniq[id] = struct{}{}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return []inspectiondom.InspectionBatch{}, nil
	}

	batches := make([]inspectiondom.InspectionBatch, 0, len(ids))

	for _, pid := range ids {
		snap, err := r.col().Doc(pid).Get(ctx)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				continue
			}
			return nil, err
		}

		batch, err := docToInspectionBatch(snap)
		if err != nil {
			return nil, err
		}
		batches = append(batches, batch)
	}

	return batches, nil
}

// Save: Upsert
func (r *InspectionRepositoryFS) Save(
	ctx context.Context,
	v inspectiondom.InspectionBatch,
) (inspectiondom.InspectionBatch, error) {

	if r == nil || r.Client == nil {
		return inspectiondom.InspectionBatch{}, errors.New("firestore client is nil")
	}

	pid := strings.TrimSpace(v.ProductionID)
	if pid == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductionID
	}

	docRef := r.col().Doc(pid)
	data := inspectionBatchToDoc(v)

	_, err := docRef.Set(ctx, data, firestore.MergeAll)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	return docToInspectionBatch(snap)
}

// ------------------------------------------------------------
// ★ 追加: UpdateMintID
// ------------------------------------------------------------
//
// Mint 申請/作成後に inspections ドキュメントへ mintId を記録する。
// - mintID != nil : mintId を保存
// - mintID == nil : mintId フィールドを削除（未申請状態へ戻すなど）
// 保存後は最新の InspectionBatch を返す。
// ------------------------------------------------------------
func (r *InspectionRepositoryFS) UpdateMintID(
	ctx context.Context,
	productionID string,
	mintID *string,
) (inspectiondom.InspectionBatch, error) {

	if r == nil || r.Client == nil {
		return inspectiondom.InspectionBatch{}, errors.New("firestore client is nil")
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductionID
	}

	docRef := r.col().Doc(pid)

	update := map[string]any{}
	if mintID == nil {
		// フィールドごと削除（null を残さない）
		update["mintId"] = firestore.Delete
	} else {
		mid := strings.TrimSpace(*mintID)
		if mid == "" {
			return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionMintID
		}
		update["mintId"] = mid
	}

	if _, err := docRef.Set(ctx, update, firestore.MergeAll); err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	return docToInspectionBatch(snap)
}

// ------------------------------------------------------------
// ★ 追加: ListPassedProductIDsByProductionID
// ------------------------------------------------------------
//
// mint.PassedProductLister を満たすための実装。
// - 指定された productionID の InspectionBatch を 1 件取得
// - inspections 配列の中から inspectionResult == "passed" の productId を重複なしで返す
// ------------------------------------------------------------
func (r *InspectionRepositoryFS) ListPassedProductIDsByProductionID(
	ctx context.Context,
	productionID string,
) ([]string, error) {

	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return nil, inspectiondom.ErrInvalidInspectionProductionID
	}

	// 単一バッチ取得を再利用
	batch, err := r.GetByProductionID(ctx, pid)
	if err != nil {
		return nil, err
	}

	out := make([]string, 0, len(batch.Inspections))
	seen := make(map[string]struct{}, len(batch.Inspections))

	for _, item := range batch.Inspections {
		if item.InspectionResult == nil {
			continue
		}
		if *item.InspectionResult != inspectiondom.InspectionPassed {
			continue
		}
		p := strings.TrimSpace(item.ProductID)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}

	// passed が 0 件でもエラーにはせず、空スライスを返す
	// （最終判断は usecase 側で行う）
	return out, nil
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

func inspectionBatchToDoc(v inspectiondom.InspectionBatch) map[string]any {
	items := make([]map[string]any, 0, len(v.Inspections))
	for _, ins := range v.Inspections {
		m := map[string]any{
			"productId": strings.TrimSpace(ins.ProductID),
			"modelId":   strings.TrimSpace(ins.ModelID),
		}

		// modelNumber は inspections テーブルには記録しない
		// （画面側で NameResolver により解決する方針）
		// そのため m["modelNumber"] 自体を持たせない

		if ins.InspectionResult != nil {
			m["inspectionResult"] = string(*ins.InspectionResult)
		} else {
			m["inspectionResult"] = nil
		}

		if ins.InspectedBy != nil {
			m["inspectedBy"] = strings.TrimSpace(*ins.InspectedBy)
		} else {
			m["inspectedBy"] = nil
		}

		if ins.InspectedAt != nil && !ins.InspectedAt.IsZero() {
			m["inspectedAt"] = ins.InspectedAt.UTC()
		} else {
			m["inspectedAt"] = nil
		}

		items = append(items, m)
	}

	qty := v.Quantity
	if qty <= 0 {
		qty = len(items)
	}

	data := map[string]any{
		"productionId": strings.TrimSpace(v.ProductionID),
		"status":       string(v.Status),
		"inspections":  items,
		"quantity":     qty,
		"totalPassed":  v.TotalPassed,
	}

	// mintId（任意）
	if v.MintID != nil {
		mid := strings.TrimSpace(*v.MintID)
		if mid != "" {
			data["mintId"] = mid
		} else {
			// 空文字は保存しない（ドメイン側の validate でも弾く想定）
			data["mintId"] = nil
		}
	} else {
		data["mintId"] = nil
	}

	return data
}

func docToInspectionBatch(
	doc *firestore.DocumentSnapshot,
) (inspectiondom.InspectionBatch, error) {

	data := doc.Data()
	if data == nil {
		return inspectiondom.InspectionBatch{}, fmt.Errorf("empty inspection document: %s", doc.Ref.ID)
	}

	batch := inspectiondom.InspectionBatch{
		ProductionID: strings.TrimSpace(fscommon.AsString(data["productionId"])),
		Status:       inspectiondom.InspectionStatus(strings.TrimSpace(fscommon.AsString(data["status"]))),
		MintID:       nil,
	}

	// mintId（任意）
	if v, ok := data["mintId"]; ok && v != nil {
		if s, ok := v.(string); ok {
			mid := strings.TrimSpace(s)
			if mid != "" {
				batch.MintID = &mid
			}
		}
	}

	// quantity / totalPassed は helper_repository_fs.go の asInt(v any) int を利用
	if v, ok := data["quantity"]; ok {
		batch.Quantity = asInt(v)
	}
	if v, ok := data["totalPassed"]; ok {
		batch.TotalPassed = asInt(v)
	}

	raw, ok := data["inspections"]
	if !ok || raw == nil {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductIDs
	}

	switch vv := raw.(type) {
	case []interface{}:
		for _, e := range vv {
			m, ok := e.(map[string]interface{})
			if !ok {
				continue
			}

			item := inspectiondom.InspectionItem{}

			if v, ok := m["productId"].(string); ok {
				item.ProductID = strings.TrimSpace(v)
			}

			if v, ok := m["modelId"].(string); ok {
				item.ModelID = strings.TrimSpace(v)
			}

			// modelNumber は DB からは読み取らない（スキップ）
			// 画面側 NameResolver で解決

			if v, ok := m["inspectionResult"].(string); ok {
				r := inspectiondom.InspectionResult(strings.TrimSpace(v))
				item.InspectionResult = &r
			}

			if v, ok := m["inspectedBy"].(string); ok {
				s := strings.TrimSpace(v)
				item.InspectedBy = &s
			}

			if t, ok := m["inspectedAt"].(time.Time); ok {
				tt := t.UTC()
				item.InspectedAt = &tt
			}

			batch.Inspections = append(batch.Inspections, item)
		}
	}

	if batch.ProductionID == "" || len(batch.Inspections) == 0 {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductIDs
	}

	if batch.Quantity <= 0 {
		batch.Quantity = len(batch.Inspections)
	}

	return batch, nil
}
