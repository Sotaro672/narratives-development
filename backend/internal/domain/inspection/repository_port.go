// backend/internal/domain/inspection/repository_port.go
package inspection

import "context"

// Repository は inspection ドメインの永続化ポートです。
// 「1生産ロット = 1 InspectionBatch」という前提で、
// get / list / update（upsert）のみを扱います。
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

	// ListByProductionIDs は、複数の productionID に対応する
	// InspectionBatch をまとめて取得します。
	//
	// - 引数の productionIDs は normalizeIDList 等で正規化される前提です。
	// - 見つからない ID があってもエラーとはせず、存在するものだけを返す実装を推奨します。
	ListByProductionID(
		ctx context.Context,
		productionIDs []string,
	) ([]InspectionBatch, error)

	// Save は InspectionBatch の更新（Upsert）を行います。
	//
	// - 新規作成と更新を同一メソッドで扱う想定です。
	// - 戻り値には、永続化後（サーバタイムスタンプ等を反映済み）の
	//   InspectionBatch を返すことを推奨します。
	// - 保存前に batch.Validate()（= InspectionBatch.Validate） を呼ぶのが望ましいです。
	Save(
		ctx context.Context,
		batch InspectionBatch,
	) (InspectionBatch, error)
}
