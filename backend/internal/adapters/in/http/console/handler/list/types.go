// backend/internal/adapters/in/http/console/handler/list/types.go
//
// Responsibility:
// - ListHandler の構造を定義する。
// - ここには「型定義のみ」を置き、HTTP の分岐や処理は置かない。
// - ListUsecase は usecase/list パッケージへ移動済みのため import を追従する。
//
// Firebase Storage migration policy:
// - backend は GCS signed URL / GCS upload adapter / GCS delete adapter を持たない
// - frontend が Firebase Storage へ直接 upload する
// - backend は /lists/{listId}/images/{imageId} の Firestore record を保存・取得・削除する
// - handler は画像アップロード実体を扱わず、usecase.SaveImage / DeleteImage / SetPrimaryImage に委譲する
package list

import (
	listdetailquery "narratives/internal/application/query/console/list/detail"
	listmanagementquery "narratives/internal/application/query/console/list/management"
	listuc "narratives/internal/application/usecase/list"
)

type ListHandler struct {
	uc *listuc.ListUsecase

	// split: listManagement.tsx / listCreate.tsx 向け
	qMgmt *listmanagementquery.ListManagementQuery

	// split: listDetail.tsx 向け
	qDetail *listdetailquery.ListDetailQuery
}
