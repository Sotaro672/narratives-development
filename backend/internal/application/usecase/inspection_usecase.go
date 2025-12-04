// backend/internal/application/usecase/inspection_usecase.go
package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	inspectiondom "narratives/internal/domain/inspection"
	modeldom "narratives/internal/domain/model"
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
		result inspectiondom.InspectionResult,
	) error
}

// InspectionRepo インターフェース自体は print_usecase.go 側で定義済み。
//   type InspectionRepo interface {
//       Create(ctx context.Context, batch inspectiondom.InspectionBatch) (inspectiondom.InspectionBatch, error)
//       GetByProductionID(ctx context.Context, productionID string) (inspectiondom.InspectionBatch, error)
//       Save(ctx context.Context, batch inspectiondom.InspectionBatch) (inspectiondom.InspectionBatch, error)
//   }

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
	batch inspectiondom.InspectionBatch,
) inspectiondom.InspectionBatch {

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

// ------------------------------------------------------------
// Queries
// ------------------------------------------------------------

// ★ productionId から inspections バッチをそのまま返す
func (u *InspectionUsecase) GetBatchByProductionID(
	ctx context.Context,
	productionID string,
) (inspectiondom.InspectionBatch, error) {

	if u.inspectionRepo == nil {
		return inspectiondom.InspectionBatch{}, fmt.Errorf("inspectionRepo is nil")
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductionID
	}

	batch, err := u.inspectionRepo.GetByProductionID(ctx, pid)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	// modelId → modelNumber を埋めてから返却
	batch = u.fillModelNumbers(ctx, batch)
	return batch, nil
}

// ★ 互換用エイリアス: ListByProductionID
//
//	既存コードで「ListByProductionID」を呼んでいる箇所があっても
//	GetBatchByProductionID と同じ挙動で動くようにするためのラッパー。
func (u *InspectionUsecase) ListByProductionID(
	ctx context.Context,
	productionID string,
) (inspectiondom.InspectionBatch, error) {
	return u.GetBatchByProductionID(ctx, productionID)
}

// ------------------------------------------------------------
// Commands
// ------------------------------------------------------------

// ★ inspections 内の 1 productId 分を更新する
func (u *InspectionUsecase) UpdateInspectionForProduct(
	ctx context.Context,
	productionID string,
	productID string,
	result *inspectiondom.InspectionResult,
	inspectedBy *string,
	inspectedAt *time.Time,
	status *inspectiondom.InspectionStatus,
) (inspectiondom.InspectionBatch, error) {

	if u.inspectionRepo == nil {
		return inspectiondom.InspectionBatch{}, fmt.Errorf("inspectionRepo is nil")
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductionID
	}
	pdID := strings.TrimSpace(productID)
	if pdID == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductIDs
	}

	// 1) 現在のバッチを取得（inspections/{productionId}）
	batch, err := u.inspectionRepo.GetByProductionID(ctx, pid)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
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
			if !inspectiondom.IsValidInspectionResult(*result) {
				return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionResult
			}
			r := *result
			item.InspectionResult = &r
		}

		// inspectedBy の更新
		if inspectedBy != nil {
			v := strings.TrimSpace(*inspectedBy)
			if v == "" {
				return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectedBy
			}
			item.InspectedBy = &v
		}

		// inspectedAt の更新
		if inspectedAt != nil {
			atUTC := inspectedAt.UTC()
			if atUTC.IsZero() {
				return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectedAt
			}
			item.InspectedAt = &atUTC
		}

		break
	}

	if !found {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductIDs
	}

	// 3) status の更新（任意）
	if status != nil {
		if !inspectiondom.IsValidInspectionStatus(*status) {
			return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionStatus
		}
		batch.Status = *status
	}

	// 3.5) ★ totalPassed を再集計
	passedCount := 0
	for _, ins := range batch.Inspections {
		if ins.InspectionResult != nil && *ins.InspectionResult == inspectiondom.InspectionPassed {
			passedCount++
		}
	}
	batch.TotalPassed = passedCount

	// 4) inspections テーブル側を保存
	updated, err := u.inspectionRepo.Save(ctx, batch)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	// 4.5) ★ modelNumber の埋め込み
	updated = u.fillModelNumbers(ctx, updated)

	// 5) products テーブル側の inspectionResult も同期
	//    （result が指定されている場合のみ）
	if result != nil && u.productRepo != nil {
		if err := u.productRepo.UpdateInspectionResult(ctx, pdID, *result); err != nil {
			return inspectiondom.InspectionBatch{}, err
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
) (inspectiondom.InspectionBatch, error) {

	if u.inspectionRepo == nil {
		return inspectiondom.InspectionBatch{}, fmt.Errorf("inspectionRepo is nil")
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductionID
	}

	// 1) 現在のバッチを取得
	batch, err := u.inspectionRepo.GetByProductionID(ctx, pid)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	// 2) ドメイン側の Complete を利用して一括更新
	if err := batch.Complete(by, at); err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	// 2.5) ★ 一括更新後に totalPassed を再集計
	passedCount := 0
	for _, ins := range batch.Inspections {
		if ins.InspectionResult != nil && *ins.InspectionResult == inspectiondom.InspectionPassed {
			passedCount++
		}
	}
	batch.TotalPassed = passedCount

	// 3) inspections テーブル側を保存
	updated, err := u.inspectionRepo.Save(ctx, batch)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
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
			if !inspectiondom.IsValidInspectionResult(result) {
				return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionResult
			}

			// 物理商品が存在しない可能性が高いので、notManufactured は products へ同期しない
			if result == inspectiondom.InspectionNotManufactured {
				continue
			}

			pid := strings.TrimSpace(item.ProductID)
			if pid == "" {
				// product ドキュメントが存在しないので同期対象外
				continue
			}

			if err := u.productRepo.UpdateInspectionResult(ctx, pid, result); err != nil {
				return inspectiondom.InspectionBatch{}, err
			}
		}
	}

	return updated, nil
}
