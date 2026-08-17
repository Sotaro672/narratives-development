// backend/internal/adapters/out/firestore/inspection_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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

type inspectionWriteMode uint8

const (
	inspectionWriteModeCreate inspectionWriteMode = iota
	inspectionWriteModeUpsert
)

// ------------------------------------------------------------
// Firestore record
// ------------------------------------------------------------

type inspectionItemRecord struct {
	ProductID        string     `firestore:"productId"`
	ModelID          string     `firestore:"modelId"`
	InspectionResult *string    `firestore:"inspectionResult"`
	InspectedBy      *string    `firestore:"inspectedBy"`
	InspectedAt      *time.Time `firestore:"inspectedAt"`
}

type inspectionRecord struct {
	ProductionID string                 `firestore:"productionId"`
	Status       string                 `firestore:"status"`
	Inspections  []inspectionItemRecord `firestore:"inspections"`
	Quantity     int                    `firestore:"quantity"`
	TotalPassed  int                    `firestore:"totalPassed"`
}

// Create: inspections/{productionId} を新規作成
//
// NOTE:
// inspection.Repository の主ポートは GetByProductionID / Update です。
// Create は既存呼び出し互換のために残していますが、
// 通常の更新・upsert は Update を使用します。
func (r *InspectionRepositoryFS) Create(ctx context.Context, v inspectiondom.InspectionBatch) (inspectiondom.InspectionBatch, error) {
	return r.write(ctx, v, inspectionWriteModeCreate)
}

// GetByProductionID: inspections/{productionId} を取得
func (r *InspectionRepositoryFS) GetByProductionID(ctx context.Context, productionID string) (inspectiondom.InspectionBatch, error) {
	if r == nil || r.Client == nil {
		return inspectiondom.InspectionBatch{}, errors.New("firestore client is nil")
	}
	if productionID == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductionID
	}

	snap, err := r.col().Doc(productionID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return inspectiondom.InspectionBatch{}, inspectiondom.ErrNotFound
		}
		return inspectiondom.InspectionBatch{}, err
	}

	return docToInspectionBatch(snap)
}

// Update: inspections/{productionId} をUpsert
//
// productionとinspectionは常にdocIdが一致するため、
// batch.ProductionIDをinspections/{productionId}のdocIdとして扱います。
func (r *InspectionRepositoryFS) Update(ctx context.Context, v inspectiondom.InspectionBatch) (inspectiondom.InspectionBatch, error) {
	return r.write(ctx, v, inspectionWriteModeUpsert)
}

// writeはCreateとUpdateで共通する以下の処理を担当します。
// - Firestore clientの検証
// - InspectionBatchの検証
// - productionIdの検証
// - Firestore保存データへの変換
// - 保存後のドキュメント取得
// - Domain entityへの変換
func (r *InspectionRepositoryFS) write(ctx context.Context, v inspectiondom.InspectionBatch, mode inspectionWriteMode) (inspectiondom.InspectionBatch, error) {
	if r == nil || r.Client == nil {
		return inspectiondom.InspectionBatch{}, errors.New("firestore client is nil")
	}
	if err := v.Validate(); err != nil {
		return inspectiondom.InspectionBatch{}, err
	}
	if v.ProductionID == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductionID
	}

	docRef := r.col().Doc(v.ProductionID)
	data := inspectionBatchToDoc(v)

	if err := writeInspectionBatchDocument(ctx, docRef, data, mode); err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return inspectiondom.InspectionBatch{}, inspectiondom.ErrNotFound
		}
		return inspectiondom.InspectionBatch{}, err
	}

	return docToInspectionBatch(snap)
}

func writeInspectionBatchDocument(ctx context.Context, docRef *firestore.DocumentRef, data map[string]any, mode inspectionWriteMode) error {
	switch mode {
	case inspectionWriteModeCreate:
		_, err := docRef.Create(ctx, data)
		return err

	case inspectionWriteModeUpsert:
		_, err := docRef.Set(ctx, data, firestore.MergeAll)
		return err

	default:
		return errors.New("invalid inspection write mode")
	}
}

// ------------------------------------------------------------
// ListPassedProductIDsByProductionID
// ------------------------------------------------------------
//
// mint.PassedProductListerを満たすための実装。
// 指定されたproductionIDのInspectionBatchを取得し、
// Domainが確定したpassedのproductIdを返します。
func (r *InspectionRepositoryFS) ListPassedProductIDsByProductionID(ctx context.Context, productionID string) ([]string, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	if productionID == "" {
		return nil, inspectiondom.ErrInvalidInspectionProductionID
	}

	batch, err := r.GetByProductionID(ctx, productionID)
	if err != nil {
		return nil, err
	}

	return batch.MintTargetProductIDs(), nil
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

func inspectionBatchToDoc(v inspectiondom.InspectionBatch) map[string]any {
	items := make([]map[string]any, 0, len(v.Inspections))

	for _, inspection := range v.Inspections {
		item := map[string]any{
			"productId":        inspection.ProductID,
			"modelId":          inspection.ModelID,
			"inspectionResult": string(*inspection.InspectionResult),
			"inspectedBy":      nil,
			"inspectedAt":      nil,
		}

		// modelNumberはinspections collectionには記録しない。
		// 画面側でNameResolverにより解決する。
		if inspection.InspectedBy != nil {
			item["inspectedBy"] = *inspection.InspectedBy
		}
		if inspection.InspectedAt != nil {
			item["inspectedAt"] = inspection.InspectedAt.UTC()
		}

		items = append(items, item)
	}

	return map[string]any{
		"productionId": v.ProductionID,
		"status":       string(v.Status),
		"inspections":  items,
		"quantity":     v.Quantity,
		"totalPassed":  v.TotalPassed,
	}
}

func docToInspectionBatch(doc *firestore.DocumentSnapshot) (inspectiondom.InspectionBatch, error) {
	if doc == nil || doc.Ref == nil {
		return inspectiondom.InspectionBatch{}, errors.New("inspection document snapshot is nil")
	}

	var rec inspectionRecord
	if err := doc.DataTo(&rec); err != nil {
		return inspectiondom.InspectionBatch{}, fmt.Errorf("decode inspection document %s: %w", doc.Ref.ID, err)
	}

	if rec.ProductionID == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductionID
	}
	if rec.ProductionID != doc.Ref.ID {
		return inspectiondom.InspectionBatch{}, fmt.Errorf(
			"inspection document id mismatch: docId=%q productionId=%q",
			doc.Ref.ID,
			rec.ProductionID,
		)
	}

	inspections := make([]inspectiondom.InspectionItem, 0, len(rec.Inspections))
	for _, raw := range rec.Inspections {
		item := inspectiondom.InspectionItem{
			ProductID: raw.ProductID,
			ModelID:   raw.ModelID,
		}

		if raw.InspectionResult != nil {
			result := inspectiondom.InspectionResult(*raw.InspectionResult)
			item.InspectionResult = &result
		}

		if raw.InspectedBy != nil {
			inspectedBy := *raw.InspectedBy
			item.InspectedBy = &inspectedBy
		}

		if raw.InspectedAt != nil {
			inspectedAt := *raw.InspectedAt
			item.InspectedAt = &inspectedAt
		}

		inspections = append(inspections, item)
	}

	batch := inspectiondom.InspectionBatch{
		ProductionID: rec.ProductionID,
		Status:       inspectiondom.InspectionStatus(rec.Status),
		MintID:       nil,
		Quantity:     rec.Quantity,
		TotalPassed:  rec.TotalPassed,
		Inspections:  inspections,
	}

	if err := batch.Validate(); err != nil {
		return inspectiondom.InspectionBatch{}, fmt.Errorf("invalid inspection document %s: %w", doc.Ref.ID, err)
	}

	return batch, nil
}
