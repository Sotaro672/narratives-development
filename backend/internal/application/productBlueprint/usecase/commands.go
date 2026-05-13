// backend/internal/application/productBlueprint/usecase/commands.go
package productBlueprintUsecase

import (
	"context"
	"time"

	usecase "narratives/internal/application/usecase"
	productbpdom "narratives/internal/domain/productBlueprint"
)

// ------------------------------------------------------------
// Commands
// ------------------------------------------------------------

// Create creates a ProductBlueprint.
// NOTE: usecase の公開APIは引き続き ProductBlueprint を受け取るが、repo には CreateInput を渡す。
func (u *ProductBlueprintUsecase) Create(
	ctx context.Context,
	v productbpdom.ProductBlueprint,
) (productbpdom.ProductBlueprint, error) {
	cid := usecase.CompanyIDFromContext(ctx)
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

// printed フラグを true（印刷済み）に更新するユースケース
// Handler から /product-blueprints/{id}/mark-printed などで呼ばれる想定。
func (u *ProductBlueprintUsecase) MarkPrinted(ctx context.Context, id string) (productbpdom.ProductBlueprint, error) {
	if id == "" {
		return productbpdom.ProductBlueprint{}, productbpdom.ErrInvalidID
	}

	updated, err := u.repo.MarkPrinted(ctx, id)
	if err != nil {
		return productbpdom.ProductBlueprint{}, err
	}

	return updated, nil
}

// Save is kept for backward compatibility at the usecase layer.
// - id が空なら Create として扱う
// - id があれば Update(Patch) に委譲する
//
// NOTE: repo port から Save が消えたため、ここでは repo.Save は呼ばない。
func (u *ProductBlueprintUsecase) Save(
	ctx context.Context,
	v productbpdom.ProductBlueprint,
) (productbpdom.ProductBlueprint, error) {
	id := v.ID
	if id == "" {
		return u.Create(ctx, v)
	}
	return u.Update(ctx, v)
}

// Update updates a ProductBlueprint using Patch.
// - companyId 境界は usecase でチェック（id が漏れても越境更新しない）
// - Update API では modelRefs を受け取らない方針のため、Patch には modelRefs を入れない（= 変更しない）
//
// NOTE: repo port から Save が消えたため、repo.Update を呼ぶ。
func (u *ProductBlueprintUsecase) Update(
	ctx context.Context,
	v productbpdom.ProductBlueprint,
) (productbpdom.ProductBlueprint, error) {
	id := v.ID
	if id == "" {
		return productbpdom.ProductBlueprint{}, productbpdom.ErrInvalidID
	}

	cid := usecase.CompanyIDFromContext(ctx)
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

// sanitizeModelIDs は入力 modelIds を正規化する。
// - trim
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
	cid := usecase.CompanyIDFromContext(ctx)
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
