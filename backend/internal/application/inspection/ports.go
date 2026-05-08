// backend/internal/application/inspection/ports.go
package inspection

import (
	"context"

	inspectiondom "narratives/internal/domain/inspection"
	mintdom "narratives/internal/domain/mint"
	modeldom "narratives/internal/domain/model"
)

// ------------------------------------------------------------
// Ports (Repository Interfaces)
// ------------------------------------------------------------
//
// inspections 永続化ポートは domain 側（inspection.Repository）へ移譲しました。
// ここでは inspection 以外の境界（products / mints / models）に関する最小ポートのみ定義します。

// ProductInspectionRepo は products テーブル側の inspectionResult を更新するための
// 最小限のポートです。
type ProductInspectionRepo interface {
	UpdateInspectionResult(
		ctx context.Context,
		productID string,
		result inspectiondom.InspectionResult,
	) error
}

// inspectionId から mint を 1 件取得するための最小ポート
type InspectionMintGetter interface {
	GetByInspectionID(ctx context.Context, inspectionID string) (mintdom.Mint, error)
}

// modelId から ModelVariation を 1 件取得するための最小ポート
type ModelVariationGetter interface {
	GetModelVariationByID(ctx context.Context, variationID string) (*modeldom.ModelVariation, error)
}
