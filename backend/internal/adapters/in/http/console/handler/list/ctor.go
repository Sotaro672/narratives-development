// backend/internal/adapters/in/http/console/handler/list/ctor.go
//
// Responsibility:
// - ListHandler の生成（DI）を担当する。
// - 入口は NewListHandler のみ（唯一の出入り口）とし、依存の束ねをここへ集中させる。
// - ListUsecase / ListManagementQuery / ListDetailQuery / ListImageUploader を ListHandler に注入する。
// - DELETE API 廃止により ListImageDeleter は受け取らない（常に nil）。
package list

import (
	"net/http"

	listdetailquery "narratives/internal/application/query/console/list/detail"
	listmanagementquery "narratives/internal/application/query/console/list/management"
	listuc "narratives/internal/application/usecase/list"
)

// NewListHandlerParams
// - router/DI から受け取る依存をこの struct に集約する。
// - optional は nil を許容する（既存の endpoint を壊さないため）。
type NewListHandlerParams struct {
	UC *listuc.ListUsecase

	QMgmt   *listmanagementquery.ListManagementQuery
	QDetail *listdetailquery.ListDetailQuery

	// NOTE:
	// - signed-url PUT + SaveImageFromGCS 方式なら uploader は nil でもOK
	// - DELETE API は廃止 -> deleter は受け取らない
	ImgUploader ListImageUploader
}

// ✅ SINGLE ENTRYPOINT (唯一の出入り口)
func NewListHandler(p NewListHandlerParams) http.Handler {
	return &ListHandler{
		uc:          p.UC,
		qMgmt:       p.QMgmt,
		qDetail:     p.QDetail,
		imgUploader: p.ImgUploader,
	}
}
