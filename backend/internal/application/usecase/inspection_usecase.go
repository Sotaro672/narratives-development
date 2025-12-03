// backend/internal/application/usecase/inspection_usecase.go
package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	modeldom "narratives/internal/domain/model"
	productdom "narratives/internal/domain/product"
)

// ------------------------------------------------------------
// Ports (Repository Interfaces)
// ------------------------------------------------------------

// ProductInspectionRepo は products テーブル側の inspectionResult を更新するための
// 最小限のポートです。
type ProductInspectionRepo interface {
	// 指定 productId の inspectionResult を更新する
	UpdateInspectionResult(
		ctx context.Context,
		productID string,
		result productdom.InspectionResult,
	) error
}

// InspectionRepo インターフェース自体は print_usecase.go 側で定義済み。

// ModelNumberRepo は modelID から ModelVariation を取得するためのポート。
type ModelNumberRepo interface {
	GetModelVariationByID(ctx context.Context, modelID string) (modeldom.ModelVariation, error)
}

// ------------------------------------------------------------
// Usecase
// ------------------------------------------------------------

type InspectionUsecase struct {
	inspectionRepo InspectionRepo
	productRepo    ProductInspectionRepo
	modelRepo      ModelNumberRepo
}

func NewInspectionUsecase(
	inspectionRepo InspectionRepo,
	productRepo ProductInspectionRepo,
	modelRepo ModelNumberRepo,
) *InspectionUsecase {
	return &InspectionUsecase{
		inspectionRepo: inspectionRepo,
		productRepo:    productRepo,
		modelRepo:      modelRepo,
	}
}

// ------------------------------------------------------------
// private helper: modelId → modelNumber の解決
// ------------------------------------------------------------

func (u *InspectionUsecase) fillModelNumbers(
	ctx context.Context,
	batch productdom.InspectionBatch,
) productdom.InspectionBatch {

	if u.modelRepo == nil {
		return batch
	}

	// 同じ modelId に対する解決結果はキャッシュする
	cache := make(map[string]string)

	for i := range batch.Inspections {
		mid := strings.TrimSpace(batch.Inspections[i].ModelID)
		if mid == "" {
			continue
		}

		// 既にキャッシュ済みならそれを使う
		if num, ok := cache[mid]; ok {
			n := num
			batch.Inspections[i].ModelNumber = &n
			continue
		}

		// modelId から ModelVariation を取得
		mv, err := u.modelRepo.GetModelVariationByID(ctx, mid)
		if err != nil {
			continue
		}

		num := strings.TrimSpace(mv.ModelNumber)
		if num == "" {
			// ModelNumber が未設定ならスキップ
			continue
		}

		// キャッシュしてから項目に反映
		cache[mid] = num
		n := num
		batch.Inspections[i].ModelNumber = &n
	}

	return batch
}

// ★ 追加: productionId から inspections バッチをそのまま返す
func (u *InspectionUsecase) GetBatchByProductionID(
	ctx context.Context,
	productionID string,
) (productdom.InspectionBatch, error) {

	if u.inspectionRepo == nil {
		return productdom.InspectionBatch{}, fmt.Errorf("inspectionRepo is nil")
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionProductionID
	}

	batch, err := u.inspectionRepo.GetByProductionID(ctx, pid)
	if err != nil {
		return productdom.InspectionBatch{}, err
	}

	// modelId → modelNumber を埋めてから返却
	batch = u.fillModelNumbers(ctx, batch)
	return batch, nil
}

// ★ inspections 内の 1 productId 分を更新する
func (u *InspectionUsecase) UpdateInspectionForProduct(
	ctx context.Context,
	productionID string,
	productID string,
	result *productdom.InspectionResult,
	inspectedBy *string,
	inspectedAt *time.Time,
	status *productdom.InspectionStatus,
) (productdom.InspectionBatch, error) {

	if u.inspectionRepo == nil {
		return productdom.InspectionBatch{}, fmt.Errorf("inspectionRepo is nil")
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionProductionID
	}
	pdID := strings.TrimSpace(productID)
	if pdID == "" {
		return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionProductIDs
	}

	// 1) 現在のバッチを取得（inspections/{productionId}）
	batch, err := u.inspectionRepo.GetByProductionID(ctx, pid)
	if err != nil {
		return productdom.InspectionBatch{}, err
	}

	// 2) 対象 productId の InspectionItem を探す
	found := false
	for i := range batch.Inspections {
		if strings.TrimSpace(batch.Inspections[i].ProductID) != pdID {
			continue
		}
		found = true

		item := &batch.Inspections[i]

		// inspectionResult の更新
		if result != nil {
			if !productdom.IsValidInspectionResult(*result) {
				return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionResult
			}
			r := *result
			item.InspectionResult = &r
		}

		// inspectedBy の更新
		if inspectedBy != nil {
			v := strings.TrimSpace(*inspectedBy)
			if v == "" {
				return productdom.InspectionBatch{}, productdom.ErrInvalidInspectedBy
			}
			item.InspectedBy = &v
		}

		// inspectedAt の更新
		if inspectedAt != nil {
			at := inspectedAt.UTC()
			if at.IsZero() {
				return productdom.InspectionBatch{}, productdom.ErrInvalidInspectedAt
			}
			item.InspectedAt = &at
		}

		break
	}

	if !found {
		return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionProductIDs
	}

	// 3) status の更新（任意）
	if status != nil {
		if !productdom.IsValidInspectionStatus(*status) {
			return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionStatus
		}
		batch.Status = *status
	}

	// 3.5) ★ totalPassed を再集計
	passedCount := 0
	for _, ins := range batch.Inspections {
		if ins.InspectionResult != nil && *ins.InspectionResult == productdom.InspectionPassed {
			passedCount++
		}
	}
	batch.TotalPassed = passedCount

	// 4) inspections テーブル側を保存
	updated, err := u.inspectionRepo.Save(ctx, batch)
	if err != nil {
		return productdom.InspectionBatch{}, err
	}

	// 4.5) ★ modelNumber の埋め込み
	updated = u.fillModelNumbers(ctx, updated)

	// 5) products テーブル側の inspectionResult も同期
	//    （result が指定されている場合のみ）
	if result != nil && u.productRepo != nil {
		if err := u.productRepo.UpdateInspectionResult(ctx, pdID, *result); err != nil {
			return productdom.InspectionBatch{}, err
		}
	}

	return updated, nil
}

// ★ 検品完了（未検品を notManufactured にし、ステータスを completed にする）
func (u *InspectionUsecase) CompleteInspectionForProduction(
	ctx context.Context,
	productionID string,
	by string,
	at time.Time,
) (productdom.InspectionBatch, error) {

	if u.inspectionRepo == nil {
		return productdom.InspectionBatch{}, fmt.Errorf("inspectionRepo is nil")
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionProductionID
	}

	// 1) 現在のバッチを取得
	batch, err := u.inspectionRepo.GetByProductionID(ctx, pid)
	if err != nil {
		return productdom.InspectionBatch{}, err
	}

	// 2) ドメイン側の Complete を利用して一括更新
	if err := batch.Complete(by, at); err != nil {
		return productdom.InspectionBatch{}, err
	}

	// 2.5) ★ 一括更新後に totalPassed を再集計
	passedCount := 0
	for _, ins := range batch.Inspections {
		if ins.InspectionResult != nil && *ins.InspectionResult == productdom.InspectionPassed {
			passedCount++
		}
	}
	batch.TotalPassed = passedCount

	// 3) inspections テーブル側を保存
	updated, err := u.inspectionRepo.Save(ctx, batch)
	if err != nil {
		return productdom.InspectionBatch{}, err
	}

	// 3.5) ★ modelNumber の埋め込み
	updated = u.fillModelNumbers(ctx, updated)

	// 4) products テーブル側の inspectionResult も同期
	//    - notManufactured になった行は「物理的な商品が存在しない」ケースがあるためスキップ
	//    - productId が空の行もスキップ
	if u.productRepo != nil {
		for _, item := range updated.Inspections {
			if item.InspectionResult == nil {
				continue
			}
			result := *item.InspectionResult
			if !productdom.IsValidInspectionResult(result) {
				return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionResult
			}

			// 物理商品が存在しない可能性が高いので、notManufactured は products へ同期しない
			if result == productdom.InspectionNotManufactured {
				continue
			}

			pid := strings.TrimSpace(item.ProductID)
			if pid == "" {
				// product ドキュメントが存在しないので同期対象外
				continue
			}

			if err := u.productRepo.UpdateInspectionResult(ctx, pid, result); err != nil {
				return productdom.InspectionBatch{}, err
			}
		}
	}

	return updated, nil
}
