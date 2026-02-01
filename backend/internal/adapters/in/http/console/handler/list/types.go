// backend/internal/adapters/in/http/console/handler/list/types.go
//
// Responsibility:
// - ListHandler の構造（依存）と、ListImage の注入用インターフェースを定義する。
// - ここには「型定義のみ」を置き、HTTP の分岐や処理は置かない。
// - ListUsecase は usecase/list パッケージへ移動済みのため import を追従する。
package list

import (
	"context"

	query "narratives/internal/application/query/console"
	listuc "narratives/internal/application/usecase/list"
	listimgdom "narratives/internal/domain/listImage"
)

type ListImageUploader interface {
	Upload(ctx context.Context, in listimgdom.UploadImageInput) (*listimgdom.ListImage, error)
}

type ListImageDeleter interface {
	Delete(ctx context.Context, imageID string) error
}

type ListHandler struct {
	uc *listuc.ListUsecase

	// split: listManagement.tsx / listCreate.tsx 向け
	qMgmt *query.ListManagementQuery

	// split: listDetail.tsx 向け
	qDetail *query.ListDetailQuery

	// ListImage (optional)
	imgUploader ListImageUploader
	imgDeleter  ListImageDeleter
}
