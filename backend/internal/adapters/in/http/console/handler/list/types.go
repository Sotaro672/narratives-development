// backend/internal/adapters/in/http/console/handler/list/types.go
//
// Responsibility:
// - ListHandler の構造（依存）と、ListImage の注入用インターフェースを定義する。
// - ここには「型定義のみ」を置き、HTTP の分岐や処理は置かない。
// - ListUsecase は usecase/list パッケージへ移動済みのため import を追従する。
//
// Policy:
// - 画像削除は handler ではなく usecase 側に寄せる（handler から imgDeleter を撤去）。
package list

import (
	"context"

	listdetailquery "narratives/internal/application/query/console/list/detail"
	listmanagementquery "narratives/internal/application/query/console/list/management"
	listuc "narratives/internal/application/usecase/list"
	listimgdom "narratives/internal/domain/listImage"
)

type ListImageUploader interface {
	Upload(ctx context.Context, in listimgdom.UploadImageInput) (*listimgdom.ListImage, error)
}

type ListHandler struct {
	uc *listuc.ListUsecase

	// split: listManagement.tsx / listCreate.tsx 向け
	qMgmt *listmanagementquery.ListManagementQuery

	// split: listDetail.tsx 向け
	qDetail *listdetailquery.ListDetailQuery

	// ListImage
	//
	// - Upload は「direct upload」を使う場合のみ必要なので optional のまま。
	// - Delete は usecase に寄せたため、handler には持たない。
	imgUploader ListImageUploader
}
