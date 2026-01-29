// backend/internal/application/productBlueprint/usecase/commands.go
package productBlueprintUsecase

import (
	"context"
	"strings"

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
	cid := strings.TrimSpace(usecase.CompanyIDFromContext(ctx))
	if cid == "" {
		return productbpdom.ProductBlueprint{}, productbpdom.ErrInvalidCompanyID
	}

	in := productbpdom.CreateInput{
		ProductName:      strings.TrimSpace(v.ProductName),
		BrandID:          strings.TrimSpace(v.BrandID),
		ItemType:         v.ItemType,
		Fit:              strings.TrimSpace(v.Fit),
		Material:         strings.TrimSpace(v.Material),
		Weight:           v.Weight,
		QualityAssurance: v.QualityAssurance,
		ProductIdTag:     v.ProductIdTag,
		AssigneeID:       strings.TrimSpace(v.AssigneeID),
		CompanyID:        cid,
		CreatedBy:        v.CreatedBy,
		CreatedAt:        nil, // repo may set if nil（必要なら v.CreatedAt を詰めても良い）
	}

	created, err := u.repo.Create(ctx, in)
	if err != nil {
		return productbpdom.ProductBlueprint{}, err
	}

	if u.historyRepo != nil {
		if err := u.historyRepo.SaveSnapshot(ctx, created); err != nil {
			return productbpdom.ProductBlueprint{}, err
		}
	}

	return created, nil
}

// ★ printed フラグを true（印刷済み）に更新するユースケース
// Handler から /product-blueprints/{id}/mark-printed などで呼ばれる想定。
func (u *ProductBlueprintUsecase) MarkPrinted(ctx context.Context, id string) (productbpdom.ProductBlueprint, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return productbpdom.ProductBlueprint{}, productbpdom.ErrInvalidID
	}

	updated, err := u.repo.MarkPrinted(ctx, id)
	if err != nil {
		return productbpdom.ProductBlueprint{}, err
	}

	if u.historyRepo != nil {
		if err := u.historyRepo.SaveSnapshot(ctx, updated); err != nil {
			return productbpdom.ProductBlueprint{}, err
		}
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
	id := strings.TrimSpace(v.ID)
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
	id := strings.TrimSpace(v.ID)
	if id == "" {
		return productbpdom.ProductBlueprint{}, productbpdom.ErrInvalidID
	}

	cid := strings.TrimSpace(usecase.CompanyIDFromContext(ctx))
	if cid == "" {
		return productbpdom.ProductBlueprint{}, productbpdom.ErrInvalidCompanyID
	}

	// 既存取得（越境更新の防止、NotFound 明確化、printed 等の参照にも使える）
	current, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return productbpdom.ProductBlueprint{}, err
	}
	if strings.TrimSpace(current.CompanyID) == "" || strings.TrimSpace(current.CompanyID) != cid {
		// company 境界違反（id を推測されても更新できないようにする）
		return productbpdom.ProductBlueprint{}, productbpdom.ErrForbidden
	}

	// Patch を組み立て（Update で変更したい項目のみ）
	name := strings.TrimSpace(v.ProductName)
	brandID := strings.TrimSpace(v.BrandID)
	itemType := v.ItemType
	fit := strings.TrimSpace(v.Fit)
	material := strings.TrimSpace(v.Material)
	weight := v.Weight

	qa := make([]string, 0, len(v.QualityAssurance))
	if v.QualityAssurance != nil {
		qa = append(qa, v.QualityAssurance...)
	}

	tag := v.ProductIdTag
	assigneeID := strings.TrimSpace(v.AssigneeID)

	patch := productbpdom.Patch{
		ProductName:      &name,
		BrandID:          &brandID,
		ItemType:         &itemType,
		Fit:              &fit,
		Material:         &material,
		Weight:           &weight,
		QualityAssurance: &qa,
		ProductIdTag:     &tag,
		AssigneeID:       &assigneeID,

		// NOTE:
		// - ModelRefs は Update API では受け取らない（変更しない）ため nil のまま
		// - BrandName / CompanyName 等の表示専用も永続化しないため nil のまま
	}

	updated, err := u.repo.Update(ctx, id, patch)
	if err != nil {
		return productbpdom.ProductBlueprint{}, err
	}

	if u.historyRepo != nil {
		if err := u.historyRepo.SaveSnapshot(ctx, updated); err != nil {
			return productbpdom.ProductBlueprint{}, err
		}
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
	for _, raw := range in {
		v := strings.TrimSpace(raw)
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
	id = strings.TrimSpace(id)
	if id == "" {
		return productbpdom.ProductBlueprint{}, productbpdom.ErrInvalidID
	}

	// 境界チェック（companyId が取れないなら弾く）
	cid := strings.TrimSpace(usecase.CompanyIDFromContext(ctx))
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
	if strings.TrimSpace(current.CompanyID) == "" || strings.TrimSpace(current.CompanyID) != cid {
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

	// ★ updatedAt/updatedBy を触らない repository API を呼ぶ（repo 側で担保）
	updated, err := u.repo.AppendModelRefsWithoutTouch(ctx, id, refs)
	if err != nil {
		return productbpdom.ProductBlueprint{}, err
	}

	if u.historyRepo != nil {
		if err := u.historyRepo.SaveSnapshot(ctx, updated); err != nil {
			return productbpdom.ProductBlueprint{}, err
		}
	}

	return updated, nil
}
