// backend/internal/adapters/out/firestore/inspection_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"

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

func NewInspectionRepositoryFS(
	client *firestore.Client,
) *InspectionRepositoryFS {
	return &InspectionRepositoryFS{
		Client: client,
	}
}

func (r *InspectionRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("inspections")
}

type inspectionWriteMode uint8

const (
	inspectionWriteModeCreate inspectionWriteMode = iota
	inspectionWriteModeUpsert
)

// Create: inspections/{productionId} を新規作成
//
// NOTE:
// inspection.Repository の主ポートは GetByProductionID / Update です。
// Create は既存呼び出し互換のために残していますが、
// 通常の更新・upsert は Update を使用します。
func (r *InspectionRepositoryFS) Create(
	ctx context.Context,
	v inspectiondom.InspectionBatch,
) (inspectiondom.InspectionBatch, error) {
	return r.write(
		ctx,
		v,
		inspectionWriteModeCreate,
	)
}

// GetByProductionID: inspections/{productionId} を取得
func (r *InspectionRepositoryFS) GetByProductionID(
	ctx context.Context,
	productionID string,
) (inspectiondom.InspectionBatch, error) {
	if r == nil || r.Client == nil {
		return inspectiondom.InspectionBatch{},
			errors.New("firestore client is nil")
	}

	if productionID == "" {
		return inspectiondom.InspectionBatch{},
			inspectiondom.ErrInvalidInspectionProductionID
	}

	snap, err := r.col().
		Doc(productionID).
		Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return inspectiondom.InspectionBatch{},
				inspectiondom.ErrNotFound
		}

		return inspectiondom.InspectionBatch{}, err
	}

	return docToInspectionBatch(snap)
}

// Update: inspections/{productionId} をUpsert
//
// productionとinspectionは常にdocIdが一致するため、
// batch.ProductionIDをinspections/{productionId}のdocIdとして扱います。
func (r *InspectionRepositoryFS) Update(
	ctx context.Context,
	v inspectiondom.InspectionBatch,
) (inspectiondom.InspectionBatch, error) {
	return r.write(
		ctx,
		v,
		inspectionWriteModeUpsert,
	)
}

// writeはCreateとUpdateで共通する以下の処理を担当します。
// - Firestore clientの検証
// - InspectionBatchの検証
// - productionIdの検証
// - Firestore保存データへの変換
// - 保存後のドキュメント取得
// - Domain entityへの変換
func (r *InspectionRepositoryFS) write(
	ctx context.Context,
	v inspectiondom.InspectionBatch,
	mode inspectionWriteMode,
) (inspectiondom.InspectionBatch, error) {
	if r == nil || r.Client == nil {
		return inspectiondom.InspectionBatch{},
			errors.New("firestore client is nil")
	}

	if err := v.Validate(); err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	productionID := v.ProductionID
	if productionID == "" {
		return inspectiondom.InspectionBatch{},
			inspectiondom.ErrInvalidInspectionProductionID
	}

	docRef := r.col().Doc(productionID)
	data := inspectionBatchToDoc(v)

	if err := writeInspectionBatchDocument(
		ctx,
		docRef,
		data,
		mode,
	); err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return inspectiondom.InspectionBatch{},
				inspectiondom.ErrNotFound
		}

		return inspectiondom.InspectionBatch{}, err
	}

	return docToInspectionBatch(snap)
}

func writeInspectionBatchDocument(
	ctx context.Context,
	docRef *firestore.DocumentRef,
	data map[string]any,
	mode inspectionWriteMode,
) error {
	switch mode {
	case inspectionWriteModeCreate:
		_, err := docRef.Create(ctx, data)
		if err != nil {
			return err
		}

		return nil

	case inspectionWriteModeUpsert:
		_, err := docRef.Set(
			ctx,
			data,
			firestore.MergeAll,
		)
		if err != nil {
			return err
		}

		return nil

	default:
		return errors.New(
			"invalid inspection write mode",
		)
	}
}

// ------------------------------------------------------------
// ListPassedProductIDsByProductionID
// ------------------------------------------------------------
//
// mint.PassedProductListerを満たすための実装。
//   - 指定されたproductionIDのInspectionBatchを1件取得
//   - inspections配列からinspectionResult == "passed"のproductIdを
//     重複なしで返す
func (r *InspectionRepositoryFS) ListPassedProductIDsByProductionID(
	ctx context.Context,
	productionID string,
) ([]string, error) {
	if r == nil || r.Client == nil {
		return nil,
			errors.New("firestore client is nil")
	}

	if productionID == "" {
		return nil,
			inspectiondom.ErrInvalidInspectionProductionID
	}

	batch, err := r.GetByProductionID(
		ctx,
		productionID,
	)
	if err != nil {
		return nil, err
	}

	out := make(
		[]string,
		0,
		len(batch.Inspections),
	)
	seen := make(
		map[string]struct{},
		len(batch.Inspections),
	)

	for _, item := range batch.Inspections {
		if item.InspectionResult == nil {
			continue
		}

		if *item.InspectionResult !=
			inspectiondom.InspectionPassed {
			continue
		}

		productID := item.ProductID
		if productID == "" {
			continue
		}

		if _, exists := seen[productID]; exists {
			continue
		}

		seen[productID] = struct{}{}
		out = append(out, productID)
	}

	// passedが0件でもエラーにはせず、空スライスを返す。
	// 最終判断はusecase側で行う。
	return out, nil
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

func inspectionBatchToDoc(
	v inspectiondom.InspectionBatch,
) map[string]any {
	items := make(
		[]map[string]any,
		0,
		len(v.Inspections),
	)

	for _, inspection := range v.Inspections {
		item := map[string]any{
			"productId": inspection.ProductID,
			"modelId":   inspection.ModelID,
		}

		// modelNumberはinspections collectionには記録しない。
		// 画面側でNameResolverにより解決する。
		if inspection.InspectionResult != nil {
			item["inspectionResult"] =
				string(*inspection.InspectionResult)
		} else {
			item["inspectionResult"] = nil
		}

		if inspection.InspectedBy != nil {
			item["inspectedBy"] =
				*inspection.InspectedBy
		} else {
			item["inspectedBy"] = nil
		}

		if inspection.InspectedAt != nil &&
			!inspection.InspectedAt.IsZero() {
			item["inspectedAt"] =
				inspection.InspectedAt.UTC()
		} else {
			item["inspectedAt"] = nil
		}

		items = append(items, item)
	}

	quantity := v.Quantity
	if quantity <= 0 {
		quantity = len(items)
	}

	return map[string]any{
		"productionId": v.ProductionID,
		"status":       string(v.Status),
		"inspections":  items,
		"quantity":     quantity,
		"totalPassed":  v.TotalPassed,
	}
}

func docToInspectionBatch(
	doc *firestore.DocumentSnapshot,
) (inspectiondom.InspectionBatch, error) {
	if doc == nil || doc.Ref == nil {
		return inspectiondom.InspectionBatch{},
			errors.New(
				"inspection document snapshot is nil",
			)
	}

	data := doc.Data()
	if data == nil {
		return inspectiondom.InspectionBatch{},
			fmt.Errorf(
				"empty inspection document: %s",
				doc.Ref.ID,
			)
	}

	productionID := asString(
		data["productionId"],
	)
	if productionID == "" {
		productionID = doc.Ref.ID
	}

	batch := inspectiondom.InspectionBatch{
		ProductionID: productionID,
		Status: inspectiondom.InspectionStatus(
			asString(data["status"]),
		),
		MintID: nil,
	}

	if value, ok := data["quantity"]; ok {
		batch.Quantity = asInt(value)
	}

	if value, ok := data["totalPassed"]; ok {
		batch.TotalPassed = asInt(value)
	}

	rawInspections, ok := data["inspections"]
	if !ok || rawInspections == nil {
		return inspectiondom.InspectionBatch{},
			inspectiondom.ErrInvalidInspectionProductIDs
	}

	switch items := rawInspections.(type) {
	case []any:
		for _, rawItem := range items {
			itemData, ok :=
				rawItem.(map[string]any)
			if !ok {
				continue
			}

			item := inspectiondom.InspectionItem{
				ProductID: asString(
					itemData["productId"],
				),
				ModelID: asString(
					itemData["modelId"],
				),
			}

			// modelNumberはFirestoreから読み取らない。
			// 画面側のNameResolverで解決する。
			inspectionResult := asString(
				itemData["inspectionResult"],
			)
			if inspectionResult != "" {
				result :=
					inspectiondom.InspectionResult(
						inspectionResult,
					)
				item.InspectionResult = &result
			}

			inspectedBy := asString(
				itemData["inspectedBy"],
			)
			if inspectedBy != "" {
				item.InspectedBy = &inspectedBy
			}

			inspectedAt, ok := asTime(
				itemData["inspectedAt"],
			)
			if ok && !inspectedAt.IsZero() {
				utc := inspectedAt.UTC()
				item.InspectedAt = &utc
			}

			batch.Inspections = append(
				batch.Inspections,
				item,
			)
		}
	}

	if batch.ProductionID == "" ||
		len(batch.Inspections) == 0 {
		return inspectiondom.InspectionBatch{},
			inspectiondom.ErrInvalidInspectionProductIDs
	}

	if batch.Quantity <= 0 {
		batch.Quantity =
			len(batch.Inspections)
	}

	return batch, nil
}
