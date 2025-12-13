// backend/internal/application/inspection/usecase.go
package inspection

import (
	"context"
	"fmt"
	"strings"
	"time"

	inspectiondto "narratives/internal/application/inspection/dto"
	inspectiondom "narratives/internal/domain/inspection"
	mintdom "narratives/internal/domain/mint"
)

// ------------------------------------------------------------
// Ports (Repository Interfaces)
// ------------------------------------------------------------
//
// inspections 永続化ポートは domain 側（inspection.Repository）へ移譲しました。
// ここでは inspection 以外の境界（products / mints）に関する最小ポートのみ定義します。

// ProductInspectionRepo は products テーブル側の inspectionResult を更新するための
// 最小限のポートです。
type ProductInspectionRepo interface {
	UpdateInspectionResult(
		ctx context.Context,
		productID string,
		result inspectiondom.InspectionResult,
	) error
}

// inspectionId (= productionId 扱い) から mint を 1 件取得するための最小ポート
type InspectionMintGetter interface {
	GetByInspectionID(ctx context.Context, inspectionID string) (mintdom.Mint, error)
}

// ------------------------------------------------------------
// Usecase
// ------------------------------------------------------------

type InspectionUsecase struct {
	inspectionRepo inspectiondom.Repository
	productRepo    ProductInspectionRepo
	mintRepo       InspectionMintGetter // nil 許容
}

func NewInspectionUsecase(
	inspectionRepo inspectiondom.Repository,
	productRepo ProductInspectionRepo,
) *InspectionUsecase {
	return &InspectionUsecase{
		inspectionRepo: inspectionRepo,
		productRepo:    productRepo,
	}
}

func NewInspectionUsecaseWithMint(
	inspectionRepo inspectiondom.Repository,
	productRepo ProductInspectionRepo,
	mintRepo InspectionMintGetter,
) *InspectionUsecase {
	u := NewInspectionUsecase(inspectionRepo, productRepo)
	u.mintRepo = mintRepo
	return u
}

// ------------------------------------------------------------
// Queries
// ------------------------------------------------------------

// productionId から inspections バッチをそのまま返す
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

	return batch, nil
}

// 互換用エイリアス
func (u *InspectionUsecase) ListByProductionID(
	ctx context.Context,
	productionID string,
) (inspectiondom.InspectionBatch, error) {
	return u.GetBatchByProductionID(ctx, productionID)
}

// inspectionId に紐づく mint を 1 件取得する
func (u *InspectionUsecase) GetMintByInspectionID(
	ctx context.Context,
	inspectionID string,
) (mintdom.Mint, error) {

	if u.mintRepo == nil {
		return mintdom.Mint{}, fmt.Errorf("mintRepo is nil")
	}

	iid := strings.TrimSpace(inspectionID)
	if iid == "" {
		return mintdom.Mint{}, inspectiondom.ErrInvalidInspectionProductionID
	}

	return u.mintRepo.GetByInspectionID(ctx, iid)
}

func (u *InspectionUsecase) GetMintByProductionID(
	ctx context.Context,
	productionID string,
) (mintdom.Mint, error) {
	return u.GetMintByInspectionID(ctx, productionID)
}

// ★ 追加：画面用 DTO（InspectionBatch + Mint を結合して返す）
func (u *InspectionUsecase) GetBatchForScreenByProductionID(
	ctx context.Context,
	productionID string,
) (inspectiondto.InspectionBatchForScreenDTO, error) {

	batch, err := u.GetBatchByProductionID(ctx, productionID)
	if err != nil {
		return inspectiondto.InspectionBatchForScreenDTO{}, err
	}

	// batch.MintID は *string なので nil 安全に扱う
	var mintDTO *inspectiondto.MintDTO
	if u.mintRepo != nil && batch.MintID != nil {
		mintID := strings.TrimSpace(*batch.MintID)
		if mintID != "" {
			m, err := u.mintRepo.GetByInspectionID(ctx, mintID)
			if err == nil {
				dto := inspectiondto.NewMintDTO(m, batch.ProductionID)
				mintDTO = &dto
			} else if err != mintdom.ErrNotFound {
				return inspectiondto.InspectionBatchForScreenDTO{}, err
			}
		}
	}

	return inspectiondto.NewInspectionBatchForScreenDTO(batch, mintDTO), nil
}

// ------------------------------------------------------------
// Commands
// ------------------------------------------------------------

// inspections 内の 1 productId 分を更新する
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

	batch, err := u.inspectionRepo.GetByProductionID(ctx, pid)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	found := false
	for i := range batch.Inspections {
		if strings.TrimSpace(batch.Inspections[i].ProductID) != pdID {
			continue
		}
		found = true

		item := &batch.Inspections[i]

		if result != nil {
			if !inspectiondom.IsValidInspectionResult(*result) {
				return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionResult
			}
			r := *result
			item.InspectionResult = &r
		}

		if inspectedBy != nil {
			v := strings.TrimSpace(*inspectedBy)
			if v == "" {
				return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectedBy
			}
			item.InspectedBy = &v
		}

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

	if status != nil {
		if !inspectiondom.IsValidInspectionStatus(*status) {
			return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionStatus
		}
		batch.Status = *status
	}

	passedCount := 0
	for _, ins := range batch.Inspections {
		if ins.InspectionResult != nil && *ins.InspectionResult == inspectiondom.InspectionPassed {
			passedCount++
		}
	}
	batch.TotalPassed = passedCount

	updated, err := u.inspectionRepo.Save(ctx, batch)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	if result != nil && u.productRepo != nil {
		if err := u.productRepo.UpdateInspectionResult(ctx, pdID, *result); err != nil {
			return inspectiondom.InspectionBatch{}, err
		}
	}

	return updated, nil
}

// 検品完了
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

	batch, err := u.inspectionRepo.GetByProductionID(ctx, pid)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	if err := batch.Complete(by, at); err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	passedCount := 0
	for _, ins := range batch.Inspections {
		if ins.InspectionResult != nil && *ins.InspectionResult == inspectiondom.InspectionPassed {
			passedCount++
		}
	}
	batch.TotalPassed = passedCount

	updated, err := u.inspectionRepo.Save(ctx, batch)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	if u.productRepo != nil {
		for _, item := range updated.Inspections {
			if item.InspectionResult == nil {
				continue
			}
			result := *item.InspectionResult
			if !inspectiondom.IsValidInspectionResult(result) {
				return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionResult
			}

			pid := strings.TrimSpace(item.ProductID)
			if pid == "" {
				continue
			}

			if err := u.productRepo.UpdateInspectionResult(ctx, pid, result); err != nil {
				return inspectiondom.InspectionBatch{}, err
			}
		}
	}

	return updated, nil
}
