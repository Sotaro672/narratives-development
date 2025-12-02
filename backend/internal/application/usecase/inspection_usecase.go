// backend\internal\application\usecase\inspection_usecase.go
package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	productdom "narratives/internal/domain/product"
)

// InspectorUsecase は検品アプリ（inspector）専用のユースケースをまとめる。
// - inspections テーブルの 1 productId 分の検品結果更新
// - 検品完了処理（一括クローズ）
type InspectionUsecase struct {
	inspectionRepo InspectionRepo
}

func NewInspectionUsecase(
	inspectionRepo InspectionRepo,
) *InspectionUsecase {
	return &InspectionUsecase{
		inspectionRepo: inspectionRepo,
	}
}

// ★ inspections 内の 1 productId 分を更新する
//
// もともと ProductUsecase.UpdateInspectionForProduct にあった処理を
// inspections テーブル専用に抜き出したもの。
//
// PATCH /products/inspections 用
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

	// 現在のバッチを取得
	batch, err := u.inspectionRepo.GetByProductionID(ctx, pid)
	if err != nil {
		return productdom.InspectionBatch{}, err
	}

	// 対象 productId の InspectionItem を探す
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

	// status の更新（任意）
	if status != nil {
		if !productdom.IsValidInspectionStatus(*status) {
			return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionStatus
		}
		batch.Status = *status
	}

	// 保存（InspectionRepo.Save 側で Firestore に反映）
	updated, err := u.inspectionRepo.Save(ctx, batch)
	if err != nil {
		return productdom.InspectionBatch{}, err
	}

	return updated, nil
}

// ★ 検品完了（未検品を notManufactured にし、ステータスを completed にする）
//
// もともと ProductUsecase.CompleteInspectionForProduction にあった処理を
// inspections テーブル専用に抜き出したもの。
//
// PATCH /products/inspections/complete 用
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

	// 現在のバッチを取得
	batch, err := u.inspectionRepo.GetByProductionID(ctx, pid)
	if err != nil {
		return productdom.InspectionBatch{}, err
	}

	// ドメイン側の Complete を利用して一括更新
	if err := batch.Complete(by, at); err != nil {
		return productdom.InspectionBatch{}, err
	}

	// 保存
	updated, err := u.inspectionRepo.Save(ctx, batch)
	if err != nil {
		return productdom.InspectionBatch{}, err
	}

	return updated, nil
}
