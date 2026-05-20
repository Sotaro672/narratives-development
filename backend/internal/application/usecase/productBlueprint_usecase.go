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

// ProductBlueprintUsecase is the application service for productBlueprint.
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
type ProductBlueprintRepo interface {
	// Read (single)
	GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error)
	Exists(ctx context.Context, id string) (bool, error)

	// Read (multi) - company スコープ必須
	ListByCompanyID(ctx context.Context, companyID string) ([]productbpdom.ProductBlueprint, error)

	// companyId 単位で productBlueprint の ID 一覧を取得（MintRequest chain 等）
	ListIDsByCompany(ctx context.Context, companyID string) ([]string, error)

	// printed フラグを true（印刷済み）に更新する
	MarkPrinted(ctx context.Context, id string) (productbpdom.ProductBlueprint, error)

	// Write
	Create(ctx context.Context, in productbpdom.CreateInput) (productbpdom.ProductBlueprint, error)
	Update(ctx context.Context, id string, patch productbpdom.Patch) (productbpdom.ProductBlueprint, error)

	// Delete (physical)
	Delete(ctx context.Context, id string) error

	// 起票後に modelRefs を追記（updatedAt / updatedBy を更新しない）
	// - refs は displayOrder を含む（usecase 側で採番して渡す）
	// - repo 実装側で modelRefs フィールドのみ部分更新し、updatedAt/updatedBy を触らないこと
	AppendModelRefsWithoutTouch(ctx context.Context, id string, refs []productbpdom.ModelRef) (productbpdom.ProductBlueprint, error)
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
// Queries
// ------------------------------------------------------------

func (u *ProductBlueprintUsecase) GetByID(
	ctx context.Context,
	id string,
) (productbpdom.ProductBlueprint, error) {
	return u.repo.GetByID(ctx, id)
}

func (u *ProductBlueprintUsecase) Exists(
	ctx context.Context,
	id string,
) (bool, error) {
	return u.repo.Exists(ctx, id)
}

// ListByCompanyID は handler 側の GET /product-blueprints から利用される一覧取得。
// companyId を必須にする（companyId なしの List は廃止済み）。
// テナント境界は repo 側のクエリに委譲しつつ、usecase 側でも二重ガードする。
func (u *ProductBlueprintUsecase) ListByCompanyID(
	ctx context.Context,
) ([]productbpdom.ProductBlueprint, error) {
	cid := CompanyIDFromContext(ctx)
	if cid == "" {
		return nil, productbpdom.ErrInvalidCompanyID
	}

	rows, err := u.repo.ListByCompanyID(ctx, cid)
	if err != nil {
		return nil, err
	}

	// 念のため usecase 側でも companyId をガード
	filtered := make([]productbpdom.ProductBlueprint, 0, len(rows))
	for _, pb := range rows {
		if pb.CompanyID != cid {
			continue
		}
		filtered = append(filtered, pb)
	}

	return filtered, nil
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

		// create 時点では空でも良い（後段で append）
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

// ------------------------------------------------------------
// Append model refs (no touch updatedAt/updatedBy)
// ------------------------------------------------------------

// AppendModelRefs は productBlueprint 起票後に modelRefs を追記する。
// 要件:
// - 入力: modelIds（順序が displayOrder の採番元）
// - 追記時に updatedAt / updatedBy が更新されないこと（repo 側で担保する）
func (u *ProductBlueprintUsecase) AppendModelRefs(
	ctx context.Context,
	id string,
	modelIds []string,
) (productbpdom.ProductBlueprint, error) {
	if id == "" {
		return productbpdom.ProductBlueprint{}, productbpdom.ErrInvalidID
	}

	// 境界チェック（companyId が取れないなら弾く）
	cid := CompanyIDFromContext(ctx)
	if cid == "" {
		return productbpdom.ProductBlueprint{}, productbpdom.ErrInvalidCompanyID
	}

	modelIds = sanitizeModelIDs(modelIds)
	if len(modelIds) == 0 {
		return productbpdom.ProductBlueprint{}, productbpdom.WrapInvalid(nil, "modelIds is required")
	}

	// 存在確認 + 越境防止
	current, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return productbpdom.ProductBlueprint{}, err
	}
	if current.CompanyID == "" || current.CompanyID != cid {
		return productbpdom.ProductBlueprint{}, productbpdom.ErrForbidden
	}

	// displayOrder は usecase 側で採番（順序は保持）
	refs := make([]productbpdom.ModelRef, 0, len(modelIds))
	for i, mid := range modelIds {
		refs = append(refs, productbpdom.ModelRef{
			ModelID:      mid,
			DisplayOrder: i + 1,
		})
	}

	// updatedAt/updatedBy を触らない repository API を呼ぶ（repo 側で担保）
	updated, err := u.repo.AppendModelRefsWithoutTouch(ctx, id, refs)
	if err != nil {
		return productbpdom.ProductBlueprint{}, err
	}

	return updated, nil
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

// sanitizeModelIDs は入力 modelIds を正規化する。
// - 空は除外
// - 重複は除外（順序は保持）
func sanitizeModelIDs(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))

	for _, v := range in {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}

	return out
}

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
