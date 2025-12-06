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

	// ★ 追加: companyId 単位で productBlueprint の ID 一覧を取得
	// （MintRequest 用のチェーン: companyId → productBlueprintId → production → mintRequest など）
	ListIDsByCompany(ctx context.Context, companyID string) ([]string, error)

	// ★ 追加: printed == "notYet" のみを対象に、指定 ID 群の Blueprint を取得
	// ListIDsByCompany → ListNotYetPrinted で 1 セットになる想定
	ListNotYetPrinted(ctx context.Context, ids []string) ([]productbpdom.ProductBlueprint, error)

	// ★ 追加: printed == "printed" のみを対象に、指定 ID 群の Blueprint を取得
	// ListIDsByCompany → ListPrinted で 1 セットになる想定
	ListPrinted(ctx context.Context, ids []string) ([]productbpdom.ProductBlueprint, error)

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

// ★ companyId ごとの printed:notYet 一覧
// ListIDsByCompany → ListNotYetPrinted で 1 セットのユースケース。
func (u *ProductBlueprintUsecase) ListNotYetPrintedByCompany(
	ctx context.Context,
) ([]productbpdom.ProductBlueprint, error) {
	cid := strings.TrimSpace(companyIDFromContext(ctx))
	if cid == "" {
		return nil, productbpdom.ErrInvalidCompanyID
	}

	// 1) companyId から対象 Blueprint ID 群を取得
	ids, err := u.repo.ListIDsByCompany(ctx, cid)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []productbpdom.ProductBlueprint{}, nil
	}

	// 2) その ID 群のうち、printed == "notYet" のものだけを取得
	rows, err := u.repo.ListNotYetPrinted(ctx, ids)
	if err != nil {
		return nil, err
	}

	// 念のため usecase 側でも printed 状態を確認し、論理削除は除外
	out := make([]productbpdom.ProductBlueprint, 0, len(rows))
	for _, pb := range rows {
		if pb.DeletedAt != nil {
			continue
		}
		if pb.Printed != productbpdom.PrintedStatusNotYet {
			continue
		}
		// companyId も念のためチェック
		if strings.TrimSpace(pb.CompanyID) != cid {
			continue
		}
		out = append(out, pb)
	}

	return out, nil
}

// ★ companyId ごとの printed:printed 一覧
// ListIDsByCompany → ListPrinted で 1 セットのユースケース。
func (u *ProductBlueprintUsecase) ListPrintedByCompany(
	ctx context.Context,
) ([]productbpdom.ProductBlueprint, error) {
	cid := strings.TrimSpace(companyIDFromContext(ctx))
	if cid == "" {
		return nil, productbpdom.ErrInvalidCompanyID
	}

	// 1) companyId から対象 Blueprint ID 群を取得
	ids, err := u.repo.ListIDsByCompany(ctx, cid)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []productbpdom.ProductBlueprint{}, nil
	}

	// 2) その ID 群のうち、printed == "printed" のものだけを取得
	rows, err := u.repo.ListPrinted(ctx, ids)
	if err != nil {
		return nil, err
	}

	// 念のため usecase 側でも printed 状態を確認し、論理削除は除外
	out := make([]productbpdom.ProductBlueprint, 0, len(rows))
	for _, pb := range rows {
		if pb.DeletedAt != nil {
			continue
		}
		if pb.Printed != productbpdom.PrintedStatusPrinted {
			continue
		}
		if strings.TrimSpace(pb.CompanyID) != cid {
			continue
		}
		out = append(out, pb)
	}

	return out, nil
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
// historyRepo から productBlueprintID ごとの履歴を取得する。
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

	created, err := u.repo.Create(ctx, v)
	if err != nil {
		return productbpdom.ProductBlueprint{}, err
	}

	// ★ 履歴スナップショット保存
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
// - 履歴スナップショットを保存する
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

	// まず存在確認のみ行う（存在しなければエラー）
	if _, err := u.repo.GetByID(ctx, id); err != nil {
		return productbpdom.ProductBlueprint{}, err
	}

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
