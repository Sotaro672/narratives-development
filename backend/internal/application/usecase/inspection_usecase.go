// backend/internal/application/usecase/inspection_usecase.go
package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	productdom "narratives/internal/domain/product"
)

// ------------------------------------------------------------
// Ports (Repository Interfaces)
// ------------------------------------------------------------

// ProductInspectionRepo は products テーブル側の inspectionResult を更新するための
// 最小限のポートです。
// 具体的な実装（Firestore など）は adapters 層で用意します。
type ProductInspectionRepo interface {
	// 指定 productId の inspectionResult を更新する
	UpdateInspectionResult(
		ctx context.Context,
		productID string,
		result productdom.InspectionResult,
	) error
}

// InspectionRepo インターフェース自体は既存のものを利用する想定。
//   - GetByProductionID(ctx, productionID)
//   - Save(ctx, batch)
// の 2 つを使います。
// （定義は別ファイルに既に存在している前提です）

// ------------------------------------------------------------
// Usecase
// ------------------------------------------------------------

// InspectionUsecase は検品アプリ（inspector）専用のユースケースをまとめる。
// - inspections テーブルの 1 productId 分の検品結果更新
// - 検品完了処理（一括クローズ）
// に加えて、
// - products テーブル側の inspectionResult も同じタイミングで更新する。
type InspectionUsecase struct {
	inspectionRepo InspectionRepo
	productRepo    ProductInspectionRepo
}

func NewInspectionUsecase(
	inspectionRepo InspectionRepo,
	productRepo ProductInspectionRepo,
) *InspectionUsecase {
	return &InspectionUsecase{
		inspectionRepo: inspectionRepo,
		productRepo:    productRepo,
	}
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

	return u.inspectionRepo.GetByProductionID(ctx, pid)
}

// ★ inspections 内の 1 productId 分を更新する
//
// もともと ProductUsecase.UpdateInspectionForProduct にあった処理を
// inspections テーブル専用に抜き出したものをベースに、
//
// さらに products テーブルの inspectionResult も同時に更新するように拡張。
//
// Flutter 側からは PATCH /products/inspections を 1 回叩くだけで、
// - inspections コレクション
// - products コレクション
// の両方が更新される想定。
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

	// 4) inspections テーブル側を保存
	updated, err := u.inspectionRepo.Save(ctx, batch)
	if err != nil {
		return productdom.InspectionBatch{}, err
	}

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
//
// もともと ProductUsecase.CompleteInspectionForProduction にあった処理を
// inspections テーブル専用に抜き出したものをベースに、
//
// さらに「完了後の各 productId の inspectionResult を products テーブル側にも反映」
// するように拡張。
//
// Flutter 側からは PATCH /products/inspections/complete を 1 回叩くだけで、
// - inspections コレクション側の status/inspectionResult
// - products コレクション側の inspectionResult
// が同期される想定。
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
	//    - inspectionResult == "notYet" → "notManufactured"
	//    - status → "completed"（など）
	if err := batch.Complete(by, at); err != nil {
		return productdom.InspectionBatch{}, err
	}

	// 3) inspections テーブル側を保存
	updated, err := u.inspectionRepo.Save(ctx, batch)
	if err != nil {
		return productdom.InspectionBatch{}, err
	}

	// 4) products テーブル側の inspectionResult も同期
	//    各 InspectionItem の InspectionResult をそのまま products に反映する。
	if u.productRepo != nil {
		for _, item := range updated.Inspections {
			if item.InspectionResult == nil {
				continue
			}
			result := *item.InspectionResult
			if !productdom.IsValidInspectionResult(result) {
				// 念のため検証
				return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionResult
			}
			if err := u.productRepo.UpdateInspectionResult(ctx, item.ProductID, result); err != nil {
				return productdom.InspectionBatch{}, err
			}
		}
	}

	return updated, nil
}
