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
	// - 既存 Mint を更新します。
	// - status / minted / mintedAt / onChainTxSignature などの親 Mint 状態を更新します。
	// - 対象が存在しない場合は ErrNotFound を返す想定です。
	Update(ctx context.Context, m Mint) (Mint, error)
}

// ------------------------------------------------------
// Repository Port for MintProductTask
// ------------------------------------------------------
//
// MintProductTaskRepository は、productId 単位の mint task を永続化するポートです。
//
// 推奨 Firestore 構造:
//
//	mints/{mintID}/products/{productID}
//
// 1 product = 1 mint task とし、worker は必ず1件ずつ処理します。
// これにより、18件を一括mintせず、
// 「1件mint成功確認 → 次の1件mint」という順次実行を実現します。
type MintProductTaskRepository interface {
	// CreateTasks:
	// - mintID に紐づく productId ごとの MintProductTask を作成します。
	// - 既に同じ productID の task が存在する場合は、実装側で冪等に扱うことを推奨します。
	// - 初期 status は PENDING です。
	CreateTasks(
		ctx context.Context,
		mintID string,
		productIDs []string,
	) ([]MintProductTask, error)

	// GetByProductID:
	// - mintID + productID で task を1件取得します。
	// - 取得できない場合は ErrMintProductTaskNotFound を返す想定です。
	GetByProductID(
		ctx context.Context,
		mintID string,
		productID string,
	) (MintProductTask, error)

	// ListByMintID:
	// - mintID に紐づく task を全件取得します。
	// - 親 Mint の進捗計算、一覧表示、完了判定に使います。
	ListByMintID(
		ctx context.Context,
		mintID string,
	) ([]MintProductTask, error)

	// GetNextExecutableTask:
	// - mintID に紐づく次の実行可能 task を1件取得します。
	// - 原則として PENDING を優先し、必要に応じて FAILED_RETRYABLE も対象にします。
	// - 実装側では createdAt / productID などで安定した順序にしてください。
	// - 対象が存在しない場合は ErrMintProductTaskNotFound を返す想定です。
	GetNextExecutableTask(
		ctx context.Context,
		mintID string,
	) (MintProductTask, error)

	// MarkMinting:
	// - 対象 task を MINTING に更新します。
	// - attemptCount を増やし、mintingStartedAt / updatedAt を更新します。
	// - 既に MINTING / MINTED の場合は、実装側で排他またはエラーにしてください。
	MarkMinting(
		ctx context.Context,
		mintID string,
		productID string,
	) (MintProductTask, error)

	// MarkMinted:
	// - 対象 task を MINTED に更新します。
	// - mintAddress / signature / mintedAt / updatedAt を保存します。
	// - productId 単位で1件mintが成功したときに呼び出します。
	MarkMinted(
		ctx context.Context,
		mintID string,
		productID string,
		mintAddress string,
		signature string,
	) (MintProductTask, error)

	// MarkFailedRetryable:
	// - 対象 task を FAILED_RETRYABLE に更新します。
	// - RPC 429 / timeout / 一時的な外部依存エラーなど、再実行可能な失敗で使います。
	MarkFailedRetryable(
		ctx context.Context,
		mintID string,
		productID string,
		message string,
	) (MintProductTask, error)

	// MarkFailedFatal:
	// - 対象 task を FAILED_FATAL に更新します。
	// - metadataURI 不正、ToAddress 不正など、再実行しても成功しない可能性が高い失敗で使います。
	MarkFailedFatal(
		ctx context.Context,
		mintID string,
		productID string,
		message string,
	) (MintProductTask, error)

	// ResetRetryableToPending:
	// - FAILED_RETRYABLE の task を PENDING に戻します。
	// - 管理画面からの再実行、または worker の再開処理で使います。
	ResetRetryableToPending(
		ctx context.Context,
		mintID string,
		productID string,
	) (MintProductTask, error)
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
