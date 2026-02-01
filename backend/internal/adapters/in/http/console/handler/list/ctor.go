// backend/internal/adapters/in/http/console/handler/list/ctor.go
//
// Responsibility:
// - ListHandler の生成（DI）を担当する。
// - router から渡される依存（usecase/query/uploader/deleter）を構造体へ束ねるだけ。
// - ListUsecase が usecase/list パッケージへ移動したため、import を追従する。
package list

import (
	"net/http"

	query "narratives/internal/application/query/console"
	listuc "narratives/internal/application/usecase/list"
)

func NewListHandler(uc *listuc.ListUsecase) http.Handler {
	return &ListHandler{uc: uc, qMgmt: nil, qDetail: nil, imgUploader: nil, imgDeleter: nil}
}

func NewListHandlerWithQueries(
	uc *listuc.ListUsecase,
	qMgmt *query.ListManagementQuery,
	qDetail *query.ListDetailQuery,
) http.Handler {
	return &ListHandler{uc: uc, qMgmt: qMgmt, qDetail: qDetail, imgUploader: nil, imgDeleter: nil}
}

func NewListHandlerWithQueriesAndListImage(
	uc *listuc.ListUsecase,
	qMgmt *query.ListManagementQuery,
	qDetail *query.ListDetailQuery,
	uploader ListImageUploader,
	deleter ListImageDeleter,
) http.Handler {
	return &ListHandler{
		uc:          uc,
		qMgmt:       qMgmt,
		qDetail:     qDetail,
		imgUploader: uploader,
		imgDeleter:  deleter,
	}
}
