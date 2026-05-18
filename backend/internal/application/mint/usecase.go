// backend/internal/application/mint/usecase.go
package mint

import (
	resolver "narratives/internal/application/resolver"
	appusecase "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	mintdom "narratives/internal/domain/mint"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ============================================================
// MintUsecase 本体（DI/構造体）
// ============================================================

type MintUsecase struct {
	// productBlueprint / production / inspection / model 参照用リポジトリ
	pbRepo    mintdom.MintProductBlueprintRepo
	prodRepo  mintdom.MintProductionRepo
	inspRepo  mintdom.MintInspectionRepo
	modelRepo mintdom.MintModelRepo

	// tokenBlueprint の bucket / metadata URI 操作用ポート
	tbBucketEnsurer   TokenBlueprintBucketEnsurer
	tbMetadataEnsurer TokenBlueprintMetadataEnsurer

	// TokenBlueprint の minted 状態や一覧を扱うためのリポジトリ
	tbRepo tbdom.RepositoryPort

	// Brand 一覧取得用
	brandSvc *branddom.Service

	// mints テーブル用リポジトリ
	mintRepo mintdom.MintRepository

	// mint 結果を Mint entity へ反映する Mapper
	mintResultMapper *MintResultMapper

	// inspections → passed productId 一覧を取得するためのポート
	passedProductLister mintdom.PassedProductLister

	// チェーンミント実行用ポート
	tokenMinter TokenMintPort

	// inventories への反映（modelId 単位）
	inventoryUC InventoryUpserter

	// createdBy(memberId) → 氏名 を解決するため
	nameResolver *resolver.NameResolver
}

// NewMintUsecase は MintUsecase のコンストラクタです。
func NewMintUsecase(
	pbRepo mintdom.MintProductBlueprintRepo,
	prodRepo mintdom.MintProductionRepo,
	inspRepo mintdom.MintInspectionRepo,
	modelRepo mintdom.MintModelRepo,
	tbRepo tbdom.RepositoryPort,
	brandSvc *branddom.Service,
	mintRepo mintdom.MintRepository,
	passedProductLister mintdom.PassedProductLister,
	tokenMinter TokenMintPort,
) *MintUsecase {
	return &MintUsecase{
		pbRepo:              pbRepo,
		prodRepo:            prodRepo,
		inspRepo:            inspRepo,
		modelRepo:           modelRepo,
		tbRepo:              tbRepo,
		brandSvc:            brandSvc,
		mintRepo:            mintRepo,
		mintResultMapper:    NewMintResultMapper(),
		passedProductLister: passedProductLister,
		tokenMinter:         tokenMinter,

		tbBucketEnsurer:   nil,
		tbMetadataEnsurer: nil,

		inventoryUC:  nil,
		nameResolver: nil,
	}
}

// ============================================================
// Setters (DI 後注入用)
// ============================================================

func (u *MintUsecase) SetNameResolver(r *resolver.NameResolver) {
	if u == nil {
		return
	}
	u.nameResolver = r
}

func (u *MintUsecase) SetInventoryUsecase(uc *appusecase.InventoryUsecase) {
	if u == nil {
		return
	}

	var _ InventoryUpserter = uc
	u.inventoryUC = uc
}

func (u *MintUsecase) SetTokenBlueprintMetadataEnsurer(e TokenBlueprintMetadataEnsurer) {
	if u == nil {
		return
	}
	u.tbMetadataEnsurer = e
}
