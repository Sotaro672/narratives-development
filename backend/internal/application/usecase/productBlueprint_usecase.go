// backend/internal/application/usecase/productBlueprint_usecase.go
package usecase

import (
	"context"
	"strings"
	"time"

	productbpdom "narratives/internal/domain/productBlueprint"
)

// ProductBlueprintRepo defines the minimal persistence port needed by ProductBlueprintUsecase.
type ProductBlueprintRepo interface {
	GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error)
	Exists(ctx context.Context, id string) (bool, error)

	// 一覧取得用（companyId による絞り込みは repository 側の実装に委譲）
	List(ctx context.Context) ([]productbpdom.ProductBlueprint, error)

	Create(ctx context.Context, v productbpdom.ProductBlueprint) (productbpdom.ProductBlueprint, error)
	Save(ctx context.Context, v productbpdom.ProductBlueprint) (productbpdom.ProductBlueprint, error)
	Delete(ctx context.Context, id string) error
}

// ProductBlueprintUsecase orchestrates productBlueprint operations.
type ProductBlueprintUsecase struct {
	repo ProductBlueprintRepo
}

func NewProductBlueprintUsecase(repo ProductBlueprintRepo) *ProductBlueprintUsecase {
	return &ProductBlueprintUsecase{repo: repo}
}

// ------------------------------------------------------------
// Queries
// ------------------------------------------------------------

func (u *ProductBlueprintUsecase) GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error) {
	return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *ProductBlueprintUsecase) Exists(ctx context.Context, id string) (bool, error) {
	return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// List
// handler 側の GET /product-blueprints から利用される一覧取得。
// companyId でのテナント絞り込みは、現状は repository 実装に委譲する形にしています。
// （もし usecase 層で companyId を強制したい場合は、BrandUsecase 同様に
//
//	Filter 型を導入していく想定）
func (u *ProductBlueprintUsecase) List(ctx context.Context) ([]productbpdom.ProductBlueprint, error) {
	rows, err := u.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	// ★ 論理削除済み（DeletedAt != nil）は一覧から除外する
	filtered := make([]productbpdom.ProductBlueprint, 0, len(rows))
	for _, pb := range rows {
		if pb.DeletedAt != nil {
			// 削除済みは管理画面の一覧には出さない
			continue
		}
		filtered = append(filtered, pb)
	}

	return filtered, nil
}

// ★ 新規追加:
// DeletedAt が null ではない（論理削除済み）の ProductBlueprint のみを一覧で返す。
// 管理画面側で「ゴミ箱一覧」「復元候補一覧」などに利用する想定。
func (u *ProductBlueprintUsecase) ListDeleted(ctx context.Context) ([]productbpdom.ProductBlueprint, error) {
	rows, err := u.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	deleted := make([]productbpdom.ProductBlueprint, 0, len(rows))
	for _, pb := range rows {
		if pb.DeletedAt != nil {
			deleted = append(deleted, pb)
		}
	}
	return deleted, nil
}

// ------------------------------------------------------------
// Commands (単体)
// ------------------------------------------------------------

func (u *ProductBlueprintUsecase) Create(
	ctx context.Context,
	v productbpdom.ProductBlueprint,
) (productbpdom.ProductBlueprint, error) {
	// ★ BrandUsecase と同様:
	//   context の companyId を優先して強制適用
	if cid := companyIDFromContext(ctx); cid != "" {
		v.CompanyID = strings.TrimSpace(cid)
	}
	return u.repo.Create(ctx, v)
}

func (u *ProductBlueprintUsecase) Save(
	ctx context.Context,
	v productbpdom.ProductBlueprint,
) (productbpdom.ProductBlueprint, error) {
	// ★ BrandUsecase と同様:
	//   context の companyId を優先して強制適用
	if cid := companyIDFromContext(ctx); cid != "" {
		v.CompanyID = strings.TrimSpace(cid)
	}
	return u.repo.Save(ctx, v)
}

// Update: 既存 ID を前提とした更新用ユースケース
// - ID が空の場合は ErrInvalidID を返す
// - companyId は context 側を優先
func (u *ProductBlueprintUsecase) Update(
	ctx context.Context,
	v productbpdom.ProductBlueprint,
) (productbpdom.ProductBlueprint, error) {
	if strings.TrimSpace(v.ID) == "" {
		return productbpdom.ProductBlueprint{}, productbpdom.ErrInvalidID
	}

	if cid := companyIDFromContext(ctx); cid != "" {
		v.CompanyID = strings.TrimSpace(cid)
	}

	// repository では Save を更新にも利用する前提
	return u.repo.Save(ctx, v)
}

// 旧・物理削除用ユースケース（将来的には使用停止予定）
func (u *ProductBlueprintUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, strings.TrimSpace(id))
}

// ------------------------------------------------------------
// SoftDelete / Restore (withModels 用のエントリポイント)
// ------------------------------------------------------------

// SoftDeleteWithModels は ProductBlueprint を論理削除するためのユースケースです。
// 現時点では product_blueprints ドキュメントの DeletedAt / DeletedBy を更新する
// 実装に留め、models へのカスケードは今後の拡張ポイントとして残しています。
func (u *ProductBlueprintUsecase) SoftDeleteWithModels(
	ctx context.Context,
	id string,
	deletedBy *string,
) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return productbpdom.ErrInvalidID
	}

	// 対象 Blueprint を取得
	pb, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	// ★ ドメインメソッドで SoftDelete（DeletedAt / ExpireAt / Updated 系をまとめて更新）
	const softDeleteTTL = 90 * 24 * time.Hour
	pb.SoftDelete(now, deletedBy, softDeleteTTL)

	// companyId は context を優先
	if cid := companyIDFromContext(ctx); cid != "" {
		pb.CompanyID = strings.TrimSpace(cid)
	}

	_, err = u.repo.Save(ctx, pb)
	if err != nil {
		return err
	}

	// TODO: models 側の論理削除カスケードを ModelUsecase / ModelRepo と連携して実装
	return nil
}

// RestoreWithModels は論理削除された ProductBlueprint を復元するためのユースケースです。
// 現時点では Blueprint の DeletedAt/DeletedBy をクリアする実装に留めています。
func (u *ProductBlueprintUsecase) RestoreWithModels(
	ctx context.Context,
	id string,
	restoredBy *string,
) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return productbpdom.ErrInvalidID
	}

	pb, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	// ★ ドメインメソッドで復元（Deleted/Expire をクリアして Updated 系も更新）
	pb.Restore(now, restoredBy)

	// companyId は context を優先
	if cid := companyIDFromContext(ctx); cid != "" {
		pb.CompanyID = strings.TrimSpace(cid)
	}

	_, err = u.repo.Save(ctx, pb)
	if err != nil {
		return err
	}

	// TODO: models 側の復元カスケードを ModelUsecase / ModelRepo と連携して実装
	return nil
}
