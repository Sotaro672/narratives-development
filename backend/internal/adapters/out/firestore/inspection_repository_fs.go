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

	if r.Client == nil {
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

	if r.Client == nil {
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

	if r.Client == nil {
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

	if r.Client == nil {
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
// ★ 追加: UpdateRequestInfo
// ------------------------------------------------------------
//
// MintUsecase.UpdateRequestInfo 用。
// requestedBy / requestedAt / tokenBlueprintId を更新する。
// 保存後は最新の InspectionBatch を返す。
// ------------------------------------------------------------
func (r *InspectionRepositoryFS) UpdateRequestInfo(
	ctx context.Context,
	productionID string,
	requestedBy string,
	requestedAt time.Time,
	tokenBlueprintID string,
) (inspectiondom.InspectionBatch, error) {

	if r.Client == nil {
		return inspectiondom.InspectionBatch{}, errors.New("firestore client is nil")
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductionID
	}

	// Firestore 更新内容
	update := map[string]any{
		"requestedBy":      strings.TrimSpace(requestedBy),
		"requestedAt":      requestedAt.UTC(),
		"tokenBlueprintId": strings.TrimSpace(tokenBlueprintID),
	}

	docRef := r.col().Doc(pid)

	if _, err := docRef.Set(ctx, update, firestore.MergeAll); err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	// 保存後に最新の状態を取得
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

	if r.Client == nil {
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

		if ins.ModelNumber != nil && strings.TrimSpace(*ins.ModelNumber) != "" {
			m["modelNumber"] = strings.TrimSpace(*ins.ModelNumber)
		} else {
			m["modelNumber"] = nil
		}

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

	if v.RequestedBy != nil && strings.TrimSpace(*v.RequestedBy) != "" {
		data["requestedBy"] = strings.TrimSpace(*v.RequestedBy)
	} else {
		data["requestedBy"] = nil
	}

	if v.RequestedAt != nil && !v.RequestedAt.IsZero() {
		data["requestedAt"] = v.RequestedAt.UTC()
	} else {
		data["requestedAt"] = nil
	}

	if v.MintedAt != nil && !v.MintedAt.IsZero() {
		data["mintedAt"] = v.MintedAt.UTC()
	} else {
		data["mintedAt"] = nil
	}

	if v.ScheduledBurnDate != nil && !v.ScheduledBurnDate.IsZero() {
		data["scheduledBurnDate"] = v.ScheduledBurnDate.UTC()
	} else {
		data["scheduledBurnDate"] = nil
	}

	if v.TokenBlueprintID != nil && strings.TrimSpace(*v.TokenBlueprintID) != "" {
		data["tokenBlueprintId"] = strings.TrimSpace(*v.TokenBlueprintID)
	} else {
		data["tokenBlueprintId"] = nil
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
	}

	if v, ok := data["quantity"]; ok {
		if n, ok := asInt(v); ok {
			batch.Quantity = n
		}
	}

	if v, ok := data["totalPassed"]; ok {
		if n, ok := asInt(v); ok {
			batch.TotalPassed = n
		}
	}

	if v, ok := data["requestedBy"].(string); ok {
		s := strings.TrimSpace(v)
		if s != "" {
			batch.RequestedBy = &s
		}
	}

	if t, ok := data["requestedAt"].(time.Time); ok {
		tt := t.UTC()
		batch.RequestedAt = &tt
	}

	if t, ok := data["mintedAt"].(time.Time); ok {
		tt := t.UTC()
		batch.MintedAt = &tt
	}

	if t, ok := data["scheduledBurnDate"].(time.Time); ok {
		tt := t.UTC()
		batch.ScheduledBurnDate = &tt
	}

	if v, ok := data["tokenBlueprintId"].(string); ok {
		s := strings.TrimSpace(v)
		if s != "" {
			batch.TokenBlueprintID = &s
		}
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

			if v, ok := m["modelNumber"].(string); ok {
				s := strings.TrimSpace(v)
				if s != "" {
					item.ModelNumber = &s
				}
			}

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

// Firestore number → int
func asInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case float32:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}
