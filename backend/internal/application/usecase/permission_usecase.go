// backend/internal/application/usecase/permission_usecase.go
package usecase

import (
	"context"
	"strings"

	permissiondom "narratives/internal/domain/permission"
)

// PermissionRepo defines the persistence port used by PermissionUsecase.
// backend/internal/adapters/out/firestore/permission_repository_fs.go を正とし、
// 読み取り専用の Repository（List / GetByID）のみを前提とする。
type PermissionRepo interface {
	List(ctx context.Context, filter permissiondom.Filter, sort permissiondom.Sort, page permissiondom.Page) (permissiondom.PageResult[permissiondom.Permission], error)
	GetByID(ctx context.Context, id string) (permissiondom.Permission, error)
}

// PermissionUsecase orchestrates permission read operations.
type PermissionUsecase struct {
	repo PermissionRepo
}

func NewPermissionUsecase(repo PermissionRepo) *PermissionUsecase {
	return &PermissionUsecase{repo: repo}
}

// ========================================
// Queries（閲覧のみ）
// ========================================

// List はフィルタ + ソート + ページング付きで権限一覧を取得する。
func (u *PermissionUsecase) List(
	ctx context.Context,
	filter permissiondom.Filter,
	sort permissiondom.Sort,
	page permissiondom.Page,
) (permissiondom.PageResult[permissiondom.Permission], error) {
	return u.repo.List(ctx, filter, sort, page)
}

// GetByID は ID を元に単一 Permission を取得する。
func (u *PermissionUsecase) GetByID(ctx context.Context, id string) (permissiondom.Permission, error) {
	return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

// ※ Permission は「閲覧のみ」の前提のため、
//    Exists / Create / Save / Delete といった変更系ユースケースは定義しない。
