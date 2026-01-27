// backend/internal/application/mint/ports.go
package mint

import (
	"context"

	invdom "narratives/internal/domain/inventory"
	tokendom "narratives/internal/domain/token"
)

// ============================================================
// チェーンミント起動用ポート
// ============================================================

// TokenMintPort は、MintUsecase から見た「オンチェーンミントを起動するための」ポートです。
// TokenUsecase がこのインターフェースを実装する想定です。
type TokenMintPort interface {
	MintFromMintRequest(ctx context.Context, mintID string) (*tokendom.MintResult, error)
}

// ============================================================
// Inventory Upsert Port（modelId 単位）
// ============================================================

// InventoryUpserter は inventories の upsert を行うための最小インターフェースです。
// inventories の docId を modelId_tokenBlueprintId にする方針のため、modelID を必須にする。
type InventoryUpserter interface {
	UpsertFromMintByModel(
		ctx context.Context,
		tokenBlueprintID string,
		productBlueprintID string,
		modelID string,
		productIDs []string,
	) (invdom.Mint, error)
}

// ============================================================
// TokenBlueprint bucket ensure port（mint 直前に .keep を保証）
// ============================================================

type TokenBlueprintBucketEnsurer interface {
	// icon/contents 両 bucket の "{tokenBlueprintId}/.keep" を保証する（既存なら成功）
	EnsureKeepObjects(ctx context.Context, tokenBlueprintID string) error
}

// ============================================================
// TokenBlueprint metadata ensure port（mint 直前に metadataUri を確定）
// ============================================================

type TokenBlueprintMetadataEnsurer interface {
	// 必要なら生成・永続化して、確定した metadataUri を返す
	EnsureMetadataURIByTokenBlueprintID(ctx context.Context, tokenBlueprintID string, actorID string) (string, error)
}
