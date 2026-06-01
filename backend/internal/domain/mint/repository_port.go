// backend/internal/domain/mint/repository_port.go
package mint

import (
	"context"

	inspectiondom "narratives/internal/domain/inspection"
	pbpdom "narratives/internal/domain/productBlueprint"
)

// ------------------------------------------------------
// Repository Port for Mint (mints テーブル)
// ------------------------------------------------------
//
// Hexagonal Architecture における「出力ポート」。
// Firestore などの具体的な永続化実装は adapters/out 側で実装し、
// ドメイン層からはこのインターフェースのみを参照します。

// MintRepository は mints テーブルへの永続化を担当するリポジトリポートです。
type MintRepository interface {
	// Create:
	// - 新しい Mint エンティティを保存します。
	// - AMOL/Narratives では docId (= mintID = productionID) として保存します。
	Create(ctx context.Context, m Mint) (Mint, error)

	// GetByID:
	// - docId (= mintID = productionID) で取得します。
	// - 取得できない場合は ErrNotFound を返す想定です。
	GetByID(ctx context.Context, id string) (Mint, error)

	// Update:
	// - 既存 Mint を更新します（minted/mintedAt/onChainTxSignature 更新など）。
	// - 対象が存在しない場合は ErrNotFound を返す想定です。
	Update(ctx context.Context, m Mint) (Mint, error)
}

// ============================================================
// 他ドメイン由来のリポジトリポート
// （MintUsecase から利用される最小限のインターフェース群）
// ============================================================

// MintProductBlueprintRepo は productBlueprintId から ProductBlueprint を解決するための最小ポートです。
type MintProductBlueprintRepo interface {
	// GetByID:
	// - productBlueprintId から ProductBlueprint を取得します。
	// - ProductName / BrandID などの基本情報は取得結果から参照します。
	GetByID(ctx context.Context, id string) (pbpdom.ProductBlueprint, error)
}

// MintProductionRepo は production / productBlueprint の関連情報を取得するための最小ポートです。
type MintProductionRepo interface {
	// GetProductBlueprintIDByProductionID:
	// - productionId から productBlueprintId を取得します。
	//   この正規ポートを経由します。
	//
	// 実装例:
	// - ProductionRepositoryFS.GetProductBlueprintIDByProductionID
	// - 内部で GetByID(ctx, productionID) を呼び、
	//   Production.ProductBlueprintID を返します。
	GetProductBlueprintIDByProductionID(ctx context.Context, productionID string) (string, error)
}

// MintInspectionRepo は、mint 処理で必要な InspectionBatch を productionID キーで取得するための最小ポートです。
//
// AMOL/Narratives では production / inspection / mint の docId は同一値として扱います。
// そのため、MintUsecase から inspection を参照する場合も inspectionID ではなく productionID に統一します。
// 取得対象は InspectionBatch ですが、呼び出し側のキー名は productionID を正とします。
//
// mint は申請後に即時実行されるため、InspectionBatch.mintId を別途更新する通常フローは持ちません。
type MintInspectionRepo interface {
	// GetByProductionID:
	// - productionID と同一 docId の InspectionBatch を1件取得します。
	// - MintUsecase で単一 productionID から検品結果を参照する正規ルートです。
	GetByProductionID(ctx context.Context, productionID string) (inspectiondom.InspectionBatch, error)
}

// ------------------------------------------------------
// Inspection 由来のデータ取得ポート
// ------------------------------------------------------
//
// inspections テーブルから、inspectionResult: "passed" の productId 一覧を
// ミント処理用に取得するためのポートです。
// 実装は inspection モジュール側の Firestore リポジトリなどが担当します。

// PassedProductLister は、検査結果が "passed" の productId 一覧を取得するためのポートです。
type PassedProductLister interface {
	// ListPassedProductIDsByProductionID:
	// - productionID を受け取り、
	//   inspectionResult == "passed" の InspectionItem の productId を全件返します。
	// - 対象が存在しない場合は ErrNotFound を返すのが望ましいです。
	ListPassedProductIDsByProductionID(
		ctx context.Context,
		productionID string,
	) ([]string, error)
}

// ------------------------------------------------------
// Behavior (Mint のドメイン振る舞い)
// ------------------------------------------------------

// Validate はエンティティの一貫性チェックを公開します。
// 実体は entity.go 側の m.validate() に委譲します。
func (m Mint) Validate() error {
	return m.validate()
}
