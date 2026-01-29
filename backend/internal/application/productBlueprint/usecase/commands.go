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

func (u *ProductBlueprintUsecase) Create(ctx context.Context, v productbpdom.ProductBlueprint) (productbpdom.ProductBlueprint, error) {
	cid := strings.TrimSpace(usecase.CompanyIDFromContext(ctx))
	if cid == "" {
		return productbpdom.ProductBlueprint{}, productbpdom.ErrInvalidCompanyID
	}
	v.CompanyID = cid

	created, err := u.repo.Create(ctx, v)
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

func (u *ProductBlueprintUsecase) Save(ctx context.Context, v productbpdom.ProductBlueprint) (productbpdom.ProductBlueprint, error) {
	cid := strings.TrimSpace(usecase.CompanyIDFromContext(ctx))
	if cid == "" {
		return productbpdom.ProductBlueprint{}, productbpdom.ErrInvalidCompanyID
	}
	v.CompanyID = cid

	return u.repo.Save(ctx, v)
}

func (u *ProductBlueprintUsecase) Update(ctx context.Context, v productbpdom.ProductBlueprint) (productbpdom.ProductBlueprint, error) {
	id := strings.TrimSpace(v.ID)
	if id == "" {
		return productbpdom.ProductBlueprint{}, productbpdom.ErrInvalidID
	}

	cid := strings.TrimSpace(usecase.CompanyIDFromContext(ctx))
	if cid == "" {
		return productbpdom.ProductBlueprint{}, productbpdom.ErrInvalidCompanyID
	}
	v.CompanyID = cid

	// ★重要：既存レコードを取得して、モデル参照などの「Update 入力に含まれないフィールド」を引き継ぐ
	current, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return productbpdom.ProductBlueprint{}, err
	}

	// Update API では modelRefs を受け取らない設計のため、ここで必ず引き継ぐ
	v.ModelRefs = current.ModelRefs

	// printed は Update で変更しない方針なら、ここでも引き継いでおく（安全）
	v.Printed = current.Printed

	updated, err := u.repo.Save(ctx, v)
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

// AppendModelRefs は productBlueprint 起票後に modelRefs を追記する（案1）。
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
	_ = cid // repo 実装側で company 境界を確認する場合に備え、ここでは未使用を回避

	modelIds = sanitizeModelIDs(modelIds)
	if len(modelIds) == 0 {
		return productbpdom.ProductBlueprint{}, productbpdom.WrapInvalid(nil, "modelIds is required")
	}

	// 存在確認（NotFound を明確化）
	if _, err := u.repo.GetByID(ctx, id); err != nil {
		return productbpdom.ProductBlueprint{}, err
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
