// backend/internal/domain/mint/repository_port.go
package mint

import (
	"context"
	"strings"
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
	// - m.ID が空文字の場合、実装側で採番して返却しても構いません。
	// - 戻り値 Mint には、保存後の ID / CreatedAt などが反映されていることを期待します。
	Create(ctx context.Context, m Mint) (Mint, error)
}

// ============================================================
// 他ドメイン由来のリポジトリポート
// （MintUsecase から利用される最小限のインターフェース群）
// ============================================================

// MintProductBlueprintRepo は companyId から productBlueprintId の一覧を取得したり、
// productBlueprintId から productName / Patch を解決するための最小ポートです。
type MintProductBlueprintRepo interface {
	// companyId に紐づく product_blueprints の ID 一覧を返す
	// 実装例: ProductBlueprintRepositoryFS.ListIDsByCompany
	ListIDsByCompany(ctx context.Context, companyID string) ([]string, error)

	// productBlueprintId から productName だけを取得するヘルパ
	// 実装例: ProductBlueprintRepositoryFS.GetProductNameByID
	GetProductNameByID(ctx context.Context, id string) (string, error)

	// productBlueprintId から Patch 全体を取得するヘルパ
	// mintRequestDetail 画面の ProductBlueprintCard 表示用
	GetPatchByID(ctx context.Context, id string) (pbpdom.Patch, error)
}

// MintProductionRepo は productBlueprintId 群から productions を取得するための最小ポートです。
type MintProductionRepo interface {
	// 指定された productBlueprintId 群のいずれかを持つ Production をすべて返す
	// 実装例: ProductionRepositoryFS.ListByProductBlueprintID
	ListByProductBlueprintID(ctx context.Context, productBlueprintIDs []string) ([]proddom.Production, error)
}

// MintInspectionRepo は productionId 群から inspections を取得・更新するための最小ポートです。
type MintInspectionRepo interface {
	// 指定された productionId 群に紐づく InspectionBatch をすべて返す
	// 実装例: InspectionRepositoryFS.ListByProductionID
	ListByProductionID(ctx context.Context, productionIDs []string) ([]inspectiondom.InspectionBatch, error)

	// requested フラグを更新するための専用メソッド
	// - ミント申請が実行された productionID に紐づく InspectionBatch.requested を true にする用途を想定
	UpdateRequestedFlag(
		ctx context.Context,
		productionID string,
		requested bool,
	) (inspectiondom.InspectionBatch, error)
}

// MintModelRepo は modelId(variationID) から size / color / rgb などの情報を解決するための最小ポートです。
type MintModelRepo interface {
	// 実装例: ModelRepositoryFS.GetModelVariationByID
	GetModelVariationByID(ctx context.Context, variationID string) (*modeldom.ModelVariation, error)
}

// ------------------------------------------------------
// Inspection 由来のデータ取得ポート
// ------------------------------------------------------
//
// inspections テーブルから、inspectionResult: "passed" の productId 一覧を
// ミント処理用に取得するためのポートです。
// （実装は inspection モジュール側の Firestore リポジトリなどが担当）

// PassedProductLister は、検査結果が "passed" の productId 一覧を取得するためのポートです。
type PassedProductLister interface {
	// ListPassedProductIDsByProductionID:
	// - productionId を受け取り、
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

// ResetMinted はミント状態を未ミントへ戻します（再ミントなどのケース想定）。
func (m *Mint) ResetMinted() {
	m.Minted = false
	m.MintedAt = nil
}

// Validate はエンティティの一貫性チェックを公開します。
func (m Mint) Validate() error {
	return m.validate()
}

// ------------------------------------------------------
// internal validation
// ------------------------------------------------------

func (m Mint) validate() error {
	// ID は必須扱いにはしない（リポジトリ層で採番するケースを許容）
	if strings.TrimSpace(m.BrandID) == "" {
		return ErrInvalidBrandID
	}
	if m.TokenBlueprintID == "" {
		return ErrInvalidTokenBlueprintID
	}
	if strings.TrimSpace(m.CreatedBy) == "" {
		return ErrInvalidCreatedBy
	}
	if m.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}

	if len(m.Products) == 0 {
		return ErrInvalidProducts
	}
	for _, pid := range m.Products {
		if strings.TrimSpace(pid) == "" {
			return ErrInvalidProducts
		}
	}

	// minted / mintedAt の整合性チェック
	if m.Minted {
		if m.MintedAt == nil || m.MintedAt.IsZero() {
			return ErrInconsistentMintedStatus
		}
	} else {
		// minted=false のとき mintedAt が入っていたら矛盾として扱う
		if m.MintedAt != nil && !m.MintedAt.IsZero() {
			return ErrInconsistentMintedStatus
		}
	}

	return nil
}
