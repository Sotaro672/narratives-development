// backend/internal/domain/inspection/repository_port.go
package inspection

import (
	"context"
)

// Repository は inspection ドメインの永続化ポートです。
//
// 「1生産ロット = 1 InspectionBatch」という前提で、
// get / update（upsert）のみを扱います。
//
// production と inspection は常に docId が一致し、1 対 1 の関係であるため、
// 複数 productionID による一覧取得は扱いません。
type Repository interface {
	// GetByProductionID は、指定した productionID に対応する
	// InspectionBatch を 1 件取得します。
	//
	// - 見つからない場合は ErrNotFound を返す想定です。
	// - productionID が不正な場合は ErrInvalidInspectionProductionID を返す想定です。
	GetByProductionID(
		ctx context.Context,
		productionID string,
	) (InspectionBatch, error)

	// Update は InspectionBatch の更新（Upsert）を行います。
	//
	// production と inspection は常に docId が一致するため、
	// batch.ProductionID を inspections/{productionId} の docId として扱います。
	//
	// - 新規作成と更新を同一メソッドで扱う想定です。
	// - 戻り値には、永続化後（サーバタイムスタンプ等を反映済み）の
	//   InspectionBatch を返すことを推奨します。
	// - 保存前に batch.Validate()（= InspectionBatch.Validate） を呼ぶのが望ましいです。
	Update(
		ctx context.Context,
		batch InspectionBatch,
	) (InspectionBatch, error)
}
