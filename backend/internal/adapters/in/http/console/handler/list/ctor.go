// backend/internal/adapters/in/http/console/handler/list/ctor.go
//
// Responsibility:
// - ListHandler の生成（DI）を担当する。
// - 入口は NewListHandler のみ（唯一の出入り口）とし、依存の束ねをここへ集中させる。
// - ListUsecase / ListManagementQuery / ListDetailQuery を ListHandler に注入する。
//
// Firebase Storage migration policy:
// - backend は GCS signed URL / GCS upload adapter / GCS delete adapter を持たない
// - frontend が Firebase Storage へ直接 upload する
// - backend は /lists/{listId}/images/{imageId} の Firestore record を保存・取得・削除する
// - handler は画像アップロード実体を扱わず、usecase.SaveImage / DeleteImage / SetPrimaryImage に委譲する
package list

import (
	"net/http"

	listdetailquery "narratives/internal/application/query/console/list/detail"
	listmanagementquery "narratives/internal/application/query/console/list/management"
	listuc "narratives/internal/application/usecase/list"
)

// NewListHandlerParams
// - router/DI から受け取る依存をこの struct に集約する。
// - optional は nil を許容する。
type NewListHandlerParams struct {
	UC *listuc.ListUsecase

	QMgmt   *listmanagementquery.ListManagementQuery
	QDetail *listdetailquery.ListDetailQuery
}

// SINGLE ENTRYPOINT
func NewListHandler(p NewListHandlerParams) http.Handler {
	return &ListHandler{
		uc:      p.UC,
		qMgmt:   p.QMgmt,
		qDetail: p.QDetail,
	}
}
