// backend/internal/domain/mint/repository_port.go
package mint

import (
	"context"
	"time"

	inspectiondom "narratives/internal/domain/inspection"
	modeldom "narratives/internal/domain/model"
	pbpdom "narratives/internal/domain/productBlueprint"
	proddom "narratives/internal/domain/production"
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
	Create(ctx context.Context, m Mint) (Mint, error)

	// GetByID:
	// - docId (= mintID = productionID) で取得します。
	// - 取得できない場合は ErrNotFound を返す想定です。
	GetByID(ctx context.Context, id string) (Mint, error)

	// Update:
	// - 既存 Mint を更新します（minted/mintedAt 更新など）。
	// - 対象が存在しない場合は ErrNotFound を返す想定です。
	Update(ctx context.Context, m Mint) (Mint, error)

	// ListByProductionID:
	// - production の docId 群（= mint の docId 群）から mint を取得します。
	// - mint が存在しない productionID はスキップしてよい（一覧用途）。
	// - 戻り値は join しやすいよう productionID をキーにした map を推奨します。
	ListByProductionID(ctx context.Context, productionIDs []string) (map[string]Mint, error)
}

// ============================================================
// 他ドメイン由来のリポジトリポート
// （MintUsecase から利用される最小限のインターフェース群）
// ============================================================

// MintProductBlueprintRepo は productBlueprintId から productName / Patch を解決するための最小ポートです。
type MintProductBlueprintRepo interface {
	// GetProductNameByID:
	// - productBlueprintId から productName だけを取得します。
	// - 実装例: ProductBlueprintRepositoryFS.GetProductNameByID
	GetProductNameByID(ctx context.Context, id string) (string, error)

	// GetPatchByID:
	// - productBlueprintId から Patch 全体を取得します。
	// - mintRequestDetail 画面の ProductBlueprintCard 表示用です。
	GetPatchByID(ctx context.Context, id string) (pbpdom.Patch, error)
}

// MintProductionRepo は production / productBlueprint の関連情報を取得するための最小ポートです。
type MintProductionRepo interface {
	// ListByProductBlueprintID:
	// - 指定された productBlueprintId 群のいずれかを持つ Production をすべて返します。
	// - 実装例: ProductionRepositoryFS.ListByProductBlueprintID
	ListByProductBlueprintID(ctx context.Context, productBlueprintIDs []string) ([]proddom.Production, error)

	// GetProductBlueprintIDByProductionID:
	// - productionId から productBlueprintId を取得します。
	// - MintUsecase 側では productionID から productBlueprintID を推測・reflect 取得せず、
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

	// ListByProductionID:
	// - 複数 productionID に対応する InspectionBatch をまとめて取得します。
	// - query.go の一覧取得で使用します。
	// - 見つからない ID があってもエラーにせず、存在するものだけを返す想定です。
	ListByProductionID(ctx context.Context, productionIDs []string) ([]inspectiondom.InspectionBatch, error)
}

// MintModelRepo は modelId(variationID) から size / color / rgb などの情報を解決するための最小ポートです。
type MintModelRepo interface {
	// GetModelVariationByID:
	// - variationID から ModelVariation を取得します。
	// - 実装例: ModelRepositoryFS.GetModelVariationByID
	GetModelVariationByID(ctx context.Context, variationID string) (*modeldom.ModelVariation, error)
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

// MarkMinted はミント完了を表現するドメイン操作です。
// - at がゼロ時刻の場合は ErrInvalidMintedAt を返します。
func (m *Mint) MarkMinted(at time.Time) error {
	if at.IsZero() {
		return ErrInvalidMintedAt
	}
	atUTC := at.UTC()

	m.Minted = true
	m.MintedAt = &atUTC

	return m.validate()
}

// Validate はエンティティの一貫性チェックを公開します。
// 実体は entity.go 側の m.validate() に委譲します。
func (m Mint) Validate() error {
	return m.validate()
}
