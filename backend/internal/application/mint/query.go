// backend/internal/application/mint/query.go
package mint

import (
	"context"
	"errors"
	"sort"

	appusecase "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	domcommon "narratives/internal/domain/common"
	inspectiondom "narratives/internal/domain/inspection"
	mintdom "narratives/internal/domain/mint"
	pbpdom "narratives/internal/domain/productBlueprint"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ErrCompanyIDMissing は context から companyId が解決できない場合のエラーです。
var ErrCompanyIDMissing = errors.New("companyId not found in context")

// ============================================================
// Query: mints を productionIds(docId) で取得
// ============================================================

// ListMintsByInspectionIDs は互換名として残す。
// inspectionIDs は productionIds と同一 docId として扱う。
func (u *MintUsecase) ListMintsByInspectionIDs(
	ctx context.Context,
	inspectionIDs []string,
) (map[string]mintdom.Mint, error) {
	return u.ListMintsByProductionIDs(ctx, inspectionIDs)
}

// ListMintsByProductionIDs は、productionIds（= mint docIds）に紐づく mints を
// productionId をキーにした map で返します。
func (u *MintUsecase) ListMintsByProductionIDs(
	ctx context.Context,
	productionIDs []string,
) (map[string]mintdom.Mint, error) {
	if u == nil {
		return nil, errors.New("mint usecase is nil")
	}
	if u.mintRepo == nil {
		return nil, errors.New("mint repo is nil")
	}

	seen := make(map[string]struct{}, len(productionIDs))
	ids := make([]string, 0, len(productionIDs))

	for _, id := range productionIDs {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return map[string]mintdom.Mint{}, nil
	}

	sort.Strings(ids)

	return u.mintRepo.ListByProductionID(ctx, ids)
}

// ============================================================
// Query: ProductBlueprint Patch 解決
// ============================================================

func (u *MintUsecase) GetProductBlueprintPatchByID(
	ctx context.Context,
	productBlueprintID string,
) (pbpdom.Patch, error) {
	if u == nil {
		return pbpdom.Patch{}, errors.New("mint usecase is nil")
	}
	if u.pbRepo == nil {
		return pbpdom.Patch{}, errors.New("productBlueprint repo is nil")
	}

	if productBlueprintID == "" {
		return pbpdom.Patch{}, errors.New("productBlueprintID is empty")
	}

	return u.pbRepo.GetPatchByID(ctx, productBlueprintID)
}

// ============================================================
// Query: Brand 一覧（current company）
// ============================================================

func (u *MintUsecase) ListBrandsForCurrentCompany(
	ctx context.Context,
	page branddom.Page,
) (branddom.PageResult[branddom.Brand], error) {
	var empty branddom.PageResult[branddom.Brand]

	if u == nil {
		return empty, errors.New("mint usecase is nil")
	}
	if u.brandSvc == nil {
		return empty, errors.New("brand service is nil")
	}

	companyID := appusecase.CompanyIDFromContext(ctx)
	if companyID == "" {
		return empty, ErrCompanyIDMissing
	}

	return u.brandSvc.ListByCompanyID(ctx, companyID, page)
}

// ============================================================
// Query: TokenBlueprint 一覧（brandId フィルタ）
// ============================================================

func (u *MintUsecase) ListTokenBlueprintsByBrand(
	ctx context.Context,
	brandID string,
	page domcommon.Page,
) (domcommon.PageResult[tbdom.TokenBlueprint], error) {
	var empty domcommon.PageResult[tbdom.TokenBlueprint]

	if u == nil {
		return empty, errors.New("mint usecase is nil")
	}
	if u.tbRepo == nil {
		return empty, errors.New("tokenBlueprint repo is nil")
	}

	if brandID == "" {
		return empty, errors.New("brandID is empty")
	}

	return tbdom.ListByBrandID(ctx, u.tbRepo, brandID, page)
}

// ============================================================
// Query: inspection batches by production docIds
// ============================================================

// ListInspectionBatchesByProductionIDs fetches inspection batches by production docIds.
func (u *MintUsecase) ListInspectionBatchesByProductionIDs(
	ctx context.Context,
	productionIDs []string,
) ([]inspectiondom.InspectionBatch, error) {
	if u == nil {
		return nil, errors.New("mint usecase is nil")
	}
	if u.inspRepo == nil {
		return nil, errors.New("inspection repo is nil")
	}

	seen := make(map[string]struct{}, len(productionIDs))
	ids := make([]string, 0, len(productionIDs))

	for _, id := range productionIDs {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return []inspectiondom.InspectionBatch{}, nil
	}

	sort.Strings(ids)

	return u.inspRepo.ListByProductionID(ctx, ids)
}
