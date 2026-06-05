// backend/internal/application/usecase/productBlueprint_usecase.go
package usecase

import (
	"context"
	"time"

	productbpdom "narratives/internal/domain/productBlueprint"
)

// ------------------------------------------------------------
// Usecase
// ------------------------------------------------------------

// ProductBlueprintUsecase is the command application service for productBlueprint.
type ProductBlueprintUsecase struct {
	repo ProductBlueprintRepo

	// ProductBlueprint 起票時に productBlueprintReview 側も初期化するためのポート
	// NOTE: NewProductBlueprintUsecase が唯一の入口となるよう、外から With で差し込まない。
	reviewInit ProductBlueprintReviewInitializer
}

// NewProductBlueprintUsecase を唯一の出入り口にするため、reviewInit をコンストラクタ引数にする。
// - reviewInit が nil の場合は初期化をスキップ（既存互換）
// - 「必ず作りたい」場合は DI 側で non-nil を渡す
func NewProductBlueprintUsecase(
	repo ProductBlueprintRepo,
	reviewInit ProductBlueprintReviewInitializer,
) *ProductBlueprintUsecase {
	return &ProductBlueprintUsecase{
		repo:       repo,
		reviewInit: reviewInit,
	}
}

// ------------------------------------------------------------
// Ports
// ------------------------------------------------------------

// ProductBlueprintRepo defines the minimal persistence port needed by ProductBlueprintUsecase.
// NOTE:
// ProductBlueprintUsecase は command 専用に寄せる。
// 一覧/詳細など画面構築用 read model は application/query/console 側へ分離する。
// modelRefs の同期は ModelUsecase 側で models collection を正として行う。
type ProductBlueprintRepo interface {
	// Read for command-side existence/company-boundary checks.
	GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error)

	// printed フラグを true（印刷済み）に更新する
	MarkPrinted(ctx context.Context, id string) (productbpdom.ProductBlueprint, error)

	// Write
	Create(ctx context.Context, in productbpdom.CreateInput) (productbpdom.ProductBlueprint, error)
	Update(ctx context.Context, id string, patch productbpdom.Patch) (productbpdom.ProductBlueprint, error)

	// Delete (physical)
	Delete(ctx context.Context, id string) error
}

// ProductBlueprintReviewInitializer は ProductBlueprint 起票時に、
// レビュー側の「商品単位初期化ドキュメント」等を作成するためのポート。
type ProductBlueprintReviewInitializer interface {
	InitForProductBlueprint(
		ctx context.Context,
		productBlueprintID string,
		companyID string,
		createdAt time.Time,
		createdBy *string,
	) error
}

// ------------------------------------------------------------
// Commands
// ------------------------------------------------------------

// Create creates a ProductBlueprint.
// NOTE: usecase の公開APIは ProductBlueprint を受け取り、repo には CreateInput を渡す。
func (u *ProductBlueprintUsecase) Create(
	ctx context.Context,
	v productbpdom.ProductBlueprint,
) (productbpdom.ProductBlueprint, error) {
	cid := CompanyIDFromContext(ctx)
	if cid == "" {
		return productbpdom.ProductBlueprint{}, productbpdom.ErrInvalidCompanyID
	}

	// Create時に usecase でID生成 + CreatedAtセット（domain.validate が必須）
	id := productbpdom.NewID()
	now := time.Now().UTC()
	createdAt := &now

	in := productbpdom.CreateInput{
		ID:          id,
		ProductName: v.ProductName,
		Description: v.Description,

		BrandID: v.BrandID,

		// productBlueprintCategories の正データから生成済みの denormalized snapshot
		ProductBlueprintCategory: v.ProductBlueprintCategory,

		// fit / material / weight / qualityAssurance などカテゴリ依存項目は
		// ProductBlueprint 直下ではなく CategoryFields に集約する。
		CategoryFields: cloneCategoryFields(v.CategoryFields),

		ProductIdTag: v.ProductIdTag,
		AssigneeID:   v.AssigneeID,

		// NOTE: companyId は auth context を正とする（越境防止）
		CompanyID: cid,

		CreatedBy: v.CreatedBy,

		// domain.validate が CreatedAt 必須なので必ず埋める
		CreatedAt: createdAt,

		// modelRefs は ModelUsecase 側で models collection を正として同期する
		ModelRefs: nil,
	}

	created, err := u.repo.Create(ctx, in)
	if err != nil {
		return productbpdom.ProductBlueprint{}, err
	}

	// ------------------------------------------------------------
	// productBlueprintReview 側の初期化（起票）
	// ------------------------------------------------------------
	if u.reviewInit != nil {
		if err := u.reviewInit.InitForProductBlueprint(
			ctx,
			created.ID,
			created.CompanyID,
			created.CreatedAt,
			created.CreatedBy,
		); err != nil {
			// 方針: 二重起票の整合を優先して失敗扱い
			return productbpdom.ProductBlueprint{}, err
		}
	}

	return created, nil
}

// MarkPrinted は printed フラグを true（印刷済み）に更新するユースケース。
// Handler から /product-blueprints/{id}/mark-printed などで呼ばれる想定。
func (u *ProductBlueprintUsecase) MarkPrinted(
	ctx context.Context,
	id string,
) (productbpdom.ProductBlueprint, error) {
	if id == "" {
		return productbpdom.ProductBlueprint{}, productbpdom.ErrInvalidID
	}

	updated, err := u.repo.MarkPrinted(ctx, id)
	if err != nil {
		return productbpdom.ProductBlueprint{}, err
	}

	return updated, nil
}

// Update updates a ProductBlueprint using Patch.
// - companyId 境界は usecase でチェック（id が漏れても越境更新しない）
// - Update API では modelRefs を受け取らない方針のため、Patch には modelRefs を入れない（= 変更しない）
func (u *ProductBlueprintUsecase) Update(
	ctx context.Context,
	v productbpdom.ProductBlueprint,
) (productbpdom.ProductBlueprint, error) {
	id := v.ID
	if id == "" {
		return productbpdom.ProductBlueprint{}, productbpdom.ErrInvalidID
	}

	cid := CompanyIDFromContext(ctx)
	if cid == "" {
		return productbpdom.ProductBlueprint{}, productbpdom.ErrInvalidCompanyID
	}

	// 既存取得（越境更新の防止、NotFound 明確化、printed 等の参照にも使える）
	current, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return productbpdom.ProductBlueprint{}, err
	}
	if current.CompanyID == "" || current.CompanyID != cid {
		// company 境界違反（id を推測されても更新できないようにする）
		return productbpdom.ProductBlueprint{}, productbpdom.ErrForbidden
	}

	// Patch を組み立て（Update で変更したい項目のみ）
	name := v.ProductName
	description := v.Description
	brandID := v.BrandID
	category := v.ProductBlueprintCategory
	categoryFields := cloneCategoryFields(v.CategoryFields)
	tag := v.ProductIdTag
	assigneeID := v.AssigneeID

	var categoryFieldsPtr *productbpdom.CategoryFields
	if categoryFields != nil {
		categoryFieldsPtr = &categoryFields
	}

	patch := productbpdom.Patch{
		ProductName: &name,
		Description: &description,

		BrandID: &brandID,

		// productBlueprintCategories の正データから生成済みの denormalized snapshot
		ProductBlueprintCategory: &category,

		// fit / material / weight / qualityAssurance などカテゴリ依存項目は
		// ProductBlueprint 直下ではなく CategoryFields に集約する。
		CategoryFields: categoryFieldsPtr,

		ProductIdTag: &tag,
		AssigneeID:   &assigneeID,
	}

	updated, err := u.repo.Update(ctx, id, patch)
	if err != nil {
		return productbpdom.ProductBlueprint{}, err
	}

	return updated, nil
}

// Delete physically deletes a ProductBlueprint.
func (u *ProductBlueprintUsecase) Delete(
	ctx context.Context,
	id string,
) error {
	if id == "" {
		return productbpdom.ErrInvalidID
	}

	cid := CompanyIDFromContext(ctx)
	if cid == "" {
		return productbpdom.ErrInvalidCompanyID
	}

	current, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if current.CompanyID == "" || current.CompanyID != cid {
		return productbpdom.ErrForbidden
	}

	return u.repo.Delete(ctx, id)
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

func cloneCategoryFields(in productbpdom.CategoryFields) productbpdom.CategoryFields {
	if len(in) == 0 {
		return nil
	}

	out := make(productbpdom.CategoryFields, len(in))
	for key, value := range in {
		if key == "" {
			continue
		}
		out[key] = value
	}

	if len(out) == 0 {
		return nil
	}

	return out
}
