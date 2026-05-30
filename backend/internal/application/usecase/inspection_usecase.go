// backend/internal/application/usecase/inspection_usecase.go
package usecase

import (
	"context"
	"fmt"
	"time"

	inspectiondom "narratives/internal/domain/inspection"
)

// ------------------------------------------------------------
// Ports
// ------------------------------------------------------------
//
// inspections 永続化ポートは domain 側（inspection.Repository）へ移譲しています。
// ここでは inspection 以外の境界（products）に関する最小ポートのみ定義します。

// ProductInspectionRepo は products テーブル側の inspectionResult を更新するための
// 最小限のポートです。
type ProductInspectionRepo interface {
	UpdateInspectionResult(
		ctx context.Context,
		productID string,
		result inspectiondom.InspectionResult,
	) error
}

// ------------------------------------------------------------
// Usecase
// ------------------------------------------------------------
//
// InspectionUsecase は inspection の command 専用 usecase です。
// 画面表示用 query / DTO 組み立ては application/query/inspector に分離します。

type InspectionUsecase struct {
	inspectionRepo inspectiondom.Repository
	productRepo    ProductInspectionRepo
}

// NewInspectionUsecase を唯一の出入り口にするため、必要な依存はすべてここで受け取る。
func NewInspectionUsecase(
	inspectionRepo inspectiondom.Repository,
	productRepo ProductInspectionRepo,
) *InspectionUsecase {
	return &InspectionUsecase{
		inspectionRepo: inspectionRepo,
		productRepo:    productRepo,
	}
}

// ------------------------------------------------------------
// Commands
// ------------------------------------------------------------

// CompleteInspectionForProduction は検品を完了します。
//
// ネガティブ制では、failed / notManufactured として明示的に入力されなかった
// notYet の productId を Complete 時に passed として確定します。
func (u *InspectionUsecase) CompleteInspectionForProduction(
	ctx context.Context,
	productionID string,
	by string,
	at time.Time,
) (inspectiondom.InspectionBatch, error) {
	if u.inspectionRepo == nil {
		return inspectiondom.InspectionBatch{}, fmt.Errorf("inspectionRepo is nil")
	}

	pid := productionID
	if pid == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductionID
	}

	batch, err := u.inspectionRepo.GetByProductionID(ctx, pid)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	if err := batch.Complete(by, at); err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	updated, err := u.inspectionRepo.Update(ctx, batch)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	if u.productRepo != nil {
		for _, item := range updated.Inspections {
			if item.InspectionResult == nil {
				return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionResult
			}

			result := *item.InspectionResult
			if !inspectiondom.IsValidInspectionResult(result) {
				return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionResult
			}

			pdID := item.ProductID
			if pdID == "" {
				return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductIDs
			}

			if err := u.productRepo.UpdateInspectionResult(ctx, pdID, result); err != nil {
				return inspectiondom.InspectionBatch{}, err
			}
		}
	}

	return updated, nil
}

// UpdateInspectionForProduct は inspections 内の 1 productId 分を更新します。
//
// ネガティブ制では、通常は failed / notManufactured を明示的に入力します。
// ただし、誤って failed / notManufactured にした productId を戻すため、
// 修正操作として passed への更新も許可します。
func (u *InspectionUsecase) UpdateInspectionForProduct(
	ctx context.Context,
	productionID string,
	productID string,
	result *inspectiondom.InspectionResult,
	inspectedBy *string,
	inspectedAt *time.Time,
) (inspectiondom.InspectionBatch, error) {
	if u.inspectionRepo == nil {
		return inspectiondom.InspectionBatch{}, fmt.Errorf("inspectionRepo is nil")
	}

	pid := productionID
	if pid == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductionID
	}

	pdID := productID
	if pdID == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductIDs
	}

	if result == nil {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionResult
	}

	if *result != inspectiondom.InspectionPassed &&
		*result != inspectiondom.InspectionFailed &&
		*result != inspectiondom.InspectionNotManufactured {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionResult
	}

	if inspectedBy == nil || *inspectedBy == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectedBy
	}

	if inspectedAt == nil {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectedAt
	}

	atUTC := inspectedAt.UTC()
	if atUTC.IsZero() {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectedAt
	}

	batch, err := u.inspectionRepo.GetByProductionID(ctx, pid)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	switch *result {
	case inspectiondom.InspectionPassed:
		if err := batch.MarkPassed(pdID, *inspectedBy, atUTC); err != nil {
			return inspectiondom.InspectionBatch{}, err
		}

	case inspectiondom.InspectionFailed:
		if err := batch.MarkFailed(pdID, *inspectedBy, atUTC); err != nil {
			return inspectiondom.InspectionBatch{}, err
		}

	case inspectiondom.InspectionNotManufactured:
		if err := batch.MarkNotManufactured(pdID, *inspectedBy, atUTC); err != nil {
			return inspectiondom.InspectionBatch{}, err
		}

	default:
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionResult
	}

	updated, err := u.inspectionRepo.Update(ctx, batch)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	if u.productRepo != nil {
		if err := u.productRepo.UpdateInspectionResult(ctx, pdID, *result); err != nil {
			return inspectiondom.InspectionBatch{}, err
		}
	}

	return updated, nil
}
