// backend/internal/application/usecase/mint_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"

	inspectiondom "narratives/internal/domain/inspection"
	proddom "narratives/internal/domain/production"
)

// ============================================================
// Ports (Repository interfaces for MintUsecase)
// ============================================================

// mintProductBlueprintRepo は companyId から productBlueprintId の一覧を取得するための最小ポート
type mintProductBlueprintRepo interface {
	// companyId に紐づく product_blueprints の ID 一覧を返す
	// 実装例: ProductBlueprintRepositoryFS.ListIDsByCompany
	ListIDsByCompany(ctx context.Context, companyID string) ([]string, error)
}

// mintProductionRepo は productBlueprintId 群から productions を取得するための最小ポート
type mintProductionRepo interface {
	// 指定された productBlueprintId 群のいずれかを持つ Production をすべて返す
	// 実装例: ProductionRepositoryFS.ListByProductBlueprintID
	ListByProductBlueprintID(ctx context.Context, productBlueprintIDs []string) ([]proddom.Production, error)
}

// mintInspectionRepo は productionId 群から inspections を取得するための最小ポート
type mintInspectionRepo interface {
	// 指定された productionId 群に紐づく InspectionBatch をすべて返す
	// 実装例: InspectionRepositoryFS.ListByProductionID
	ListByProductionID(ctx context.Context, productionIDs []string) ([]inspectiondom.InspectionBatch, error)
}

// ============================================================
// MintUsecase 本体
// ============================================================

type MintUsecase struct {
	pbRepo   mintProductBlueprintRepo
	prodRepo mintProductionRepo
	inspRepo mintInspectionRepo
}

// NewMintUsecase は MintUsecase のコンストラクタです。
// DI コンテナから ProductBlueprintRepositoryFS / ProductionRepositoryFS / InspectionRepositoryFS
// をそれぞれ満たす実装として渡してください。
func NewMintUsecase(
	pbRepo mintProductBlueprintRepo,
	prodRepo mintProductionRepo,
	inspRepo mintInspectionRepo,
) *MintUsecase {
	return &MintUsecase{
		pbRepo:   pbRepo,
		prodRepo: prodRepo,
		inspRepo: inspRepo,
	}
}

// ErrCompanyIDMissing は context から companyId が解決できない場合のエラーです。
var ErrCompanyIDMissing = errors.New("companyId not found in context")

// ============================================================
// Public API
// ============================================================

// ListInspectionsForCurrentCompany は、context に埋め込まれた companyId
// （middleware.AuthMiddleware → usecase.WithCompanyID）を基点に、
//
//  1. companyId → productBlueprintId の一覧を取得
//  2. productBlueprintId → productionId の一覧を取得
//  3. productionId → inspections の一覧を取得
//
// という一連のチェーンを実行し、最終的な InspectionBatch の配列を返します。
func (u *MintUsecase) ListInspectionsForCurrentCompany(
	ctx context.Context,
) ([]inspectiondom.InspectionBatch, error) {

	companyID := strings.TrimSpace(CompanyIDFromContext(ctx))
	if companyID == "" {
		return nil, ErrCompanyIDMissing
	}

	return u.ListInspectionsByCompanyID(ctx, companyID)
}

// ListInspectionsByCompanyID は、明示的に companyId を指定して
// 同じチェーンを実行するバリアントです。
func (u *MintUsecase) ListInspectionsByCompanyID(
	ctx context.Context,
	companyID string,
) ([]inspectiondom.InspectionBatch, error) {

	if u == nil {
		return nil, errors.New("mint usecase is nil")
	}

	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, ErrCompanyIDMissing
	}

	// 1) companyId → productBlueprintId 一覧
	pbIDs, err := u.pbRepo.ListIDsByCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if len(pbIDs) == 0 {
		// 該当 product_blueprints が無ければ空配列を返す
		return []inspectiondom.InspectionBatch{}, nil
	}

	// 2) productBlueprintId 群 → Production 一覧
	prods, err := u.prodRepo.ListByProductBlueprintID(ctx, pbIDs)
	if err != nil {
		return nil, err
	}
	if len(prods) == 0 {
		// 該当 Production が無ければ空配列を返す
		return []inspectiondom.InspectionBatch{}, nil
	}

	// Production から productionId 一覧を抽出（重複除去）
	prodIDSet := make(map[string]struct{}, len(prods))
	for _, p := range prods {
		id := strings.TrimSpace(p.ID)
		if id == "" {
			continue
		}
		prodIDSet[id] = struct{}{}
	}

	if len(prodIDSet) == 0 {
		return []inspectiondom.InspectionBatch{}, nil
	}

	prodIDs := make([]string, 0, len(prodIDSet))
	for id := range prodIDSet {
		prodIDs = append(prodIDs, id)
	}

	// 3) productionId 群 → InspectionBatch 一覧
	batches, err := u.inspRepo.ListByProductionID(ctx, prodIDs)
	if err != nil {
		return nil, err
	}

	// inspections が未作成の productionId については
	// Repository 側で NotFound をスキップしている想定。
	return batches, nil
}
