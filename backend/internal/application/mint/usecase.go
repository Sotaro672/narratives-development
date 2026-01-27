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
	// 互換のため残しているが、company -> pb -> production の探索にはもう使わない方針
	pbRepo    mintdom.MintProductBlueprintRepo
	prodRepo  mintdom.MintProductionRepo
	inspRepo  mintdom.MintInspectionRepo
	modelRepo mintdom.MintModelRepo

	// ★ 追加: mint パッケージ側の Port 経由で tokenBlueprint を操作する（直 import しない）
	tbBucketEnsurer   TokenBlueprintBucketEnsurer
	tbMetadataEnsurer TokenBlueprintMetadataEnsurer

	// TokenBlueprint の minted 状態や一覧を扱うためのリポジトリ（既存）
	tbRepo tbdom.RepositoryPort

	// Brand 一覧取得用
	brandSvc *branddom.Service

	// mints テーブル用リポジトリ
	mintRepo mintdom.MintRepository

	// mintRepo の互換吸収（GetByID/Get）を隔離する Adapter
	mintRepoAdapter *MintRequestRepositoryAdapter

	// 署名/アドレス等のフィールド揺れ吸収を隔離する Mapper
	mintResultMapper *MintResultMapper

	// inspections → passed productId 一覧を取得するためのポート
	passedProductLister mintdom.PassedProductLister

	// チェーンミント実行用ポート（TokenUsecase を想定）
	tokenMinter TokenMintPort

	// inventories への反映（modelId 単位）
	inventoryUC InventoryUpserter

	// createdBy(memberId) → 氏名 を解決するため
	// 既存DIを壊さないため、Setterで後から差し込む
	nameResolver *resolver.NameResolver
}

// NewMintUsecase は MintUsecase のコンストラクタです。
// NameResolver / InventoryUC / TokenBlueprint Ensurers は任意依存（Setterで後から差し込む）とする。
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
		mintRepoAdapter:     NewMintRequestRepositoryAdapter(mintRepo),
		mintResultMapper:    NewMintResultMapper(),
		passedProductLister: passedProductLister,
		tokenMinter:         tokenMinter,

		// ★ 後注入（任意依存）
		tbBucketEnsurer:   nil,
		tbMetadataEnsurer: nil,

		inventoryUC:  nil,
		nameResolver: nil,
	}
}

// ============================================================
// Setters (DI 後注入用)
// ============================================================

// DI 側で NameResolver を後から注入できるようにする
func (u *MintUsecase) SetNameResolver(r *resolver.NameResolver) {
	if u == nil {
		return
	}
	u.nameResolver = r
}

// ★ DI 側で InventoryUsecase（または互換の Upserter）を後から注入できるようにする
// ※ *usecase.InventoryUsecase が UpsertFromMintByModel を実装している前提
func (u *MintUsecase) SetInventoryUsecase(uc *appusecase.InventoryUsecase) {
	if u == nil {
		return
	}
	// コンパイル時に interface 実装を保証したいので代入時点でチェック
	var _ InventoryUpserter = uc
	u.inventoryUC = uc
}

// 互換: interface 注入したいケース用
func (u *MintUsecase) SetInventoryUpserter(up InventoryUpserter) {
	if u == nil {
		return
	}
	u.inventoryUC = up
}

// ★ 追加: tokenBlueprint bucket ensurer を後注入
func (u *MintUsecase) SetTokenBlueprintBucketEnsurer(e TokenBlueprintBucketEnsurer) {
	if u == nil {
		return
	}
	u.tbBucketEnsurer = e
}

// ★ 追加: tokenBlueprint metadata ensurer を後注入
func (u *MintUsecase) SetTokenBlueprintMetadataEnsurer(e TokenBlueprintMetadataEnsurer) {
	if u == nil {
		return
	}
	u.tbMetadataEnsurer = e
}
