// backend/internal/application/port/list_sales_summary_reader.go
package port

import "context"

// ListSalesSummary は、1つの List に紐づく累計注文実績を表す。
//
// TotalOrderCount:
// - Paid == true の Order のみ対象
// - type == "list" の item のみ対象
// - IsCancelled == false の item のみ対象
// - 同一 Order 内に同一 List の item が複数存在しても 1 注文として数える
//
// TotalSalesAmount:
// - 対象 item の Price * Qty の累計
// - 消費税は含めない
// - 配送料は含めない
type ListSalesSummary struct {
	TotalOrderCount  int
	TotalSalesAmount int64
}

// ListSalesSummaryReader は、List ごとの累計注文実績を取得する read port。
//
// listIDs:
// - 集計対象の List ID
//
// allowedInventoryIDs:
// - current company が参照可能な inventory ID の集合
// - Order item の InventoryID がこの集合に含まれる場合のみ集計対象とする
//
// 戻り値:
// - key: listID
// - value: ListSalesSummary
// - 注文実績が存在しない List は map に存在しないことを許容する
// - 呼び出し側は未存在の場合 0 件 / 0 円として扱う
type ListSalesSummaryReader interface {
	ListByListIDs(
		ctx context.Context,
		listIDs []string,
		allowedInventoryIDs map[string]struct{},
	) (
		map[string]ListSalesSummary,
		error,
	)
}
