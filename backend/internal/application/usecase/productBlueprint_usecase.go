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

	// ★ 追加: 論理削除済みのみ取得する専用メソッド
	ListDeleted(ctx context.Context) ([]productbpdom.ProductBlueprint, error)

	Create(ctx context.Context, v productbpdom.ProductBlueprint) (productbpdom.ProductBlueprint, error)
	Save(ctx context.Context, v productbpdom.ProductBlueprint) (productbpdom.ProductBlueprint, error)
	Delete(ctx context.Context, id string) error
}

// ProductBlueprintUsecase orchestrates productBlueprint operations.
type ProductBlueprintUsecase struct {
	repo        ProductBlueprintRepo
	historyRepo productbpdom.ProductBlueprintHistoryRepo
}

func NewProductBlueprintUsecase(
	repo ProductBlueprintRepo,
	historyRepo productbpdom.ProductBlueprintHistoryRepo,
) *ProductBlueprintUsecase {
	return &ProductBlueprintUsecase{
		repo:        repo,
		historyRepo: historyRepo,
	}
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

// ★ 論理削除済みのみの一覧
// DeletedAt が null ではない ProductBlueprint のみを返す。
// Repository 側の ListDeleted は Firestore クエリで deletedAt / companyId を絞り込む。
func (u *ProductBlueprintUsecase) ListDeleted(ctx context.Context) ([]productbpdom.ProductBlueprint, error) {
	rows, err := u.repo.ListDeleted(ctx)
	if err != nil {
		return nil, err
	}

	// 念のため usecase 側でも DeletedAt != nil を保証しておく
	deleted := make([]productbpdom.ProductBlueprint, 0, len(rows))
	for _, pb := range rows {
		if pb.DeletedAt == nil {
			continue
		}
		deleted = append(deleted, pb)
	}
	return deleted, nil
}

// ★ 履歴一覧取得（LogCard 用）
// historyRepo から productBlueprintID ごとのバージョン履歴を取得する。
func (u *ProductBlueprintUsecase) ListHistory(
	ctx context.Context,
	productBlueprintID string,
) ([]productbpdom.ProductBlueprint, error) {
	productBlueprintID = strings.TrimSpace(productBlueprintID)
	if productBlueprintID == "" {
		return nil, productbpdom.ErrInvalidID
	}
	if u.historyRepo == nil {
		return nil, productbpdom.ErrInternal
	}
	return u.historyRepo.ListByProductBlueprintID(ctx, productBlueprintID)
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

	// ★ 新規作成時は version=1 を保証
	if v.Version <= 0 {
		v.Version = 1
	}

	created, err := u.repo.Create(ctx, v)
	if err != nil {
		return productbpdom.ProductBlueprint{}, err
	}

	// ★ 履歴スナップショット保存（version=1）
	if u.historyRepo != nil {
		if err := u.historyRepo.SaveSnapshot(ctx, created); err != nil {
			return productbpdom.ProductBlueprint{}, err
		}
	}

	return created, nil
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
// - companyId は context を優先
// - version は更新ごとに +1 し、そのスナップショットを履歴に保存する
func (u *ProductBlueprintUsecase) Update(
	ctx context.Context,
	v productbpdom.ProductBlueprint,
) (productbpdom.ProductBlueprint, error) {
	id := strings.TrimSpace(v.ID)
	if id == "" {
		return productbpdom.ProductBlueprint{}, productbpdom.ErrInvalidID
	}

	// companyId は context を優先
	if cid := companyIDFromContext(ctx); cid != "" {
		v.CompanyID = strings.TrimSpace(cid)
	}

	// 現在の version を取得して nextVersion を決定
	current, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return productbpdom.ProductBlueprint{}, err
	}

	var nextVersion int64
	if current.Version <= 0 {
		// 旧データなど、version 未設定の場合は 1 から開始
		nextVersion = 1
	} else {
		nextVersion = current.Version + 1
	}
	v.Version = nextVersion

	// repository では Save を更新にも利用する前提
	updated, err := u.repo.Save(ctx, v)
	if err != nil {
		return productbpdom.ProductBlueprint{}, err
	}

	// ★ 更新後スナップショットを履歴に保存
	if u.historyRepo != nil {
		if err := u.historyRepo.SaveSnapshot(ctx, updated); err != nil {
			return productbpdom.ProductBlueprint{}, err
		}
	}

	return updated, nil
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

	// ★ SoftDelete も履歴の一種としてバージョンを進める
	var nextVersion int64
	if pb.Version <= 0 {
		nextVersion = 1
	} else {
		nextVersion = pb.Version + 1
	}
	pb.Version = nextVersion

	saved, err := u.repo.Save(ctx, pb)
	if err != nil {
		return err
	}

	// ★ 論理削除後のスナップショットを履歴に保存
	if u.historyRepo != nil {
		if err := u.historyRepo.SaveSnapshot(ctx, saved); err != nil {
			return err
		}
	}

	// TODO: models 側の論理削除カスケードを ModelUsecase / ModelRepo と連携して実装
	return nil
}

// RestoreWithModels は論理削除された ProductBlueprint を復元するためのユースケースです。
// deletedAt / deletedBy / expireAt をクリアし、updatedAt / updatedBy を更新する。
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

	// ------------------------------------------------
	// ★ 復旧ロジック:
	//   - deletedAt / deletedBy / expireAt を null にする
	//   - updatedAt / updatedBy を更新する
	// ------------------------------------------------
	pb.DeletedAt = nil
	pb.DeletedBy = nil
	pb.ExpireAt = nil

	pb.UpdatedAt = now
	if restoredBy != nil {
		if trimmed := strings.TrimSpace(*restoredBy); trimmed != "" {
			// 新しいポインタを作成して UpdatedBy にセット
			rb := trimmed
			pb.UpdatedBy = &rb
		} else {
			// 空文字の場合は UpdatedBy をクリア
			pb.UpdatedBy = nil
		}
	}

	// companyId は context を優先
	if cid := companyIDFromContext(ctx); cid != "" {
		pb.CompanyID = strings.TrimSpace(cid)
	}

	// ★ 復旧も履歴イベントとしてバージョンを進める
	var nextVersion int64
	if pb.Version <= 0 {
		nextVersion = 1
	} else {
		nextVersion = pb.Version + 1
	}
	pb.Version = nextVersion

	saved, err := u.repo.Save(ctx, pb)
	if err != nil {
		return err
	}

	// ★ 復旧後スナップショットを履歴に保存
	if u.historyRepo != nil {
		if err := u.historyRepo.SaveSnapshot(ctx, saved); err != nil {
			return err
		}
	}

	// TODO: models 側の復元カスケードを ModelUsecase / ModelRepo と連携して実装
	return nil
}
