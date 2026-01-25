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
