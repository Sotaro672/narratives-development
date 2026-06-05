// backend/internal/application/usecase/model_usecase.go
package usecase

import (
	"context"
	"sort"

	modeldom "narratives/internal/domain/model"
	productbpdom "narratives/internal/domain/productBlueprint"
)

// ------------------------------------------------------------
// ProductBlueprintModelRefPort
// ------------------------------------------------------------
//
// ModelUsecase が models collection を正として
// productBlueprint.modelRefs を同期するための port。
//
// NOTE:
// - ProductBlueprint の通常更新責務は持たない。
// - modelRefs の置き換えだけを扱う。
// - ReplaceModelRefsWithoutTouch は updatedAt / updatedBy を触らない。
type ProductBlueprintModelRefPort interface {
	GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error)

	ReplaceModelRefsWithoutTouch(
		ctx context.Context,
		id string,
		refs []productbpdom.ModelRef,
	) (productbpdom.ProductBlueprint, error)
}

// ------------------------------------------------------------
// ModelUsecase
// ------------------------------------------------------------
//
// Mall handler から model.RepositoryPort として渡したいので、
// ModelUsecase 自体が modeldom.RepositoryPort を実装する（= repo に委譲する）。
//
// NOTE:
// Product-level metadata は productBlueprint.CategoryFields に集約する方針。
//
// この usecase は category-specific model variation の操作を担当する。
// apparel では size / color / measurements を扱う。
// alcohol では volume のみを扱う。
// どの category で model variation を作成するかは、
// productBlueprintCategory/input_schema.go の schema を application/usecase 側で参照して判断する。
//
// modelRefs は ProductBlueprintUsecase ではなく、ModelUsecase 側で
// models collection を正として同期する。
// ------------------------------------------------------------

type ModelUsecase struct {
	repo modeldom.RepositoryPort

	productBlueprintRefs ProductBlueprintModelRefPort
}

func NewModelUsecase(
	repo modeldom.RepositoryPort,
	productBlueprintRefs ProductBlueprintModelRefPort,
) *ModelUsecase {
	return &ModelUsecase{
		repo:                 repo,
		productBlueprintRefs: productBlueprintRefs,
	}
}

// ------------------------------------------------------------
// RepositoryPort implementation
// ------------------------------------------------------------

func (u *ModelUsecase) ListByProductBlueprintID(
	ctx context.Context,
	productBlueprintID string,
) ([]modeldom.ModelVariation, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}
	if productBlueprintID == "" {
		return nil, modeldom.ErrInvalidBlueprintID
	}

	return u.repo.ListByProductBlueprintID(ctx, productBlueprintID)
}

// GetByID returns a category-specific ModelVariation by variation ID.
//
// NOTE:
//   - MintRequest detail / InspectionResultCard などで、
//     modelId から modelNumber / size / color / volume を単体解決する用途。
//   - 永続化の正は repository なので、usecase では repo.GetByID に委譲する。
//   - productBlueprint.modelRefs の同期は不要。
func (u *ModelUsecase) GetByID(
	ctx context.Context,
	variationID string,
) (modeldom.ModelVariation, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}
	if variationID == "" {
		return nil, modeldom.ErrInvalidID
	}

	return u.repo.GetByID(ctx, variationID)
}

// Create creates a category-specific ModelVariation.
//
// NOTE:
//   - apparel では NewModelVariation.Apparel を使う。
//   - alcohol では NewModelVariation.Alcohol を使う。
//   - apparel.outerwear / apparel.shoes では Measurements は nil / 空でもよい。
//   - alcohol では Volume のみを variation field として扱う。
//   - measurements 必須カテゴリかどうかは usecase 側で category schema を参照して判定する。
//   - 作成後、models collection を正として productBlueprint.modelRefs を同期する。
func (u *ModelUsecase) Create(
	ctx context.Context,
	v modeldom.NewModelVariation,
) (modeldom.ModelVariation, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}
	if err := v.Validate(); err != nil {
		return nil, err
	}

	created, err := u.repo.Create(ctx, v)
	if err != nil {
		return nil, err
	}

	if err := u.syncProductBlueprintModelRefs(ctx, v.ProductBlueprintID()); err != nil {
		return nil, err
	}

	return created, nil
}

// Update updates a category-specific ModelVariation.
//
// NOTE:
//   - apparel では size / color / measurements 更新に対応する。
//   - alcohol では volume 更新に対応する。
//   - 一括差し替えは行わない。
//   - 削除が必要な variation は Delete を個別に呼び出す。
//   - 履歴は保存しない。
//   - modelId 自体は変わらないため、modelRefs の同期は行わない。
func (u *ModelUsecase) Update(
	ctx context.Context,
	variationID string,
	updates modeldom.ModelVariationUpdate,
) (modeldom.ModelVariation, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}
	if variationID == "" {
		return nil, modeldom.ErrInvalidID
	}

	updated, err := u.repo.Update(ctx, variationID, updates)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

// Delete physically deletes a category-specific ModelVariation.
//
// NOTE:
//   - repository は対象 document を物理削除する。
//   - 削除前に productBlueprintID を保持する。
//   - 削除後、models collection を正として productBlueprint.modelRefs を同期する。
func (u *ModelUsecase) Delete(
	ctx context.Context,
	variationID string,
) error {
	if u.repo == nil {
		return modeldom.ErrNotFound
	}
	if variationID == "" {
		return modeldom.ErrInvalidID
	}

	current, err := u.repo.GetByID(ctx, variationID)
	if err != nil {
		return err
	}

	productBlueprintID := modelVariationProductBlueprintID(current)
	if productBlueprintID == "" {
		return modeldom.ErrInvalidBlueprintID
	}

	if err := u.repo.Delete(ctx, variationID); err != nil {
		return err
	}

	return u.syncProductBlueprintModelRefs(ctx, productBlueprintID)
}

// ------------------------------------------------------------
// productBlueprint.modelRefs sync
// ------------------------------------------------------------

func (u *ModelUsecase) syncProductBlueprintModelRefs(
	ctx context.Context,
	productBlueprintID string,
) error {
	if u.repo == nil {
		return modeldom.ErrNotFound
	}
	if u.productBlueprintRefs == nil {
		return nil
	}
	if productBlueprintID == "" {
		return modeldom.ErrInvalidBlueprintID
	}

	pb, err := u.productBlueprintRefs.GetByID(ctx, productBlueprintID)
	if err != nil {
		return err
	}

	models, err := u.repo.ListByProductBlueprintID(ctx, productBlueprintID)
	if err != nil {
		return err
	}

	refs := rebuildModelRefsPreservingOrder(pb.ModelRefs, models)

	_, err = u.productBlueprintRefs.ReplaceModelRefsWithoutTouch(
		ctx,
		productBlueprintID,
		refs,
	)
	if err != nil {
		return err
	}

	return nil
}

func rebuildModelRefsPreservingOrder(
	current []productbpdom.ModelRef,
	models []modeldom.ModelVariation,
) []productbpdom.ModelRef {
	existingModelIDs := make(map[string]struct{}, len(models))
	modelIDsInListOrder := make([]string, 0, len(models))

	for _, model := range models {
		modelID := modelVariationID(model)
		if modelID == "" {
			continue
		}
		if _, ok := existingModelIDs[modelID]; ok {
			continue
		}

		existingModelIDs[modelID] = struct{}{}
		modelIDsInListOrder = append(modelIDsInListOrder, modelID)
	}

	used := make(map[string]struct{}, len(modelIDsInListOrder))
	orderedIDs := make([]string, 0, len(modelIDsInListOrder))

	currentCopy := append([]productbpdom.ModelRef(nil), current...)
	sort.SliceStable(currentCopy, func(i, j int) bool {
		return currentCopy[i].DisplayOrder < currentCopy[j].DisplayOrder
	})

	for _, ref := range currentCopy {
		modelID := ref.ModelID
		if modelID == "" {
			continue
		}
		if _, ok := existingModelIDs[modelID]; !ok {
			continue
		}
		if _, ok := used[modelID]; ok {
			continue
		}

		used[modelID] = struct{}{}
		orderedIDs = append(orderedIDs, modelID)
	}

	for _, modelID := range modelIDsInListOrder {
		if modelID == "" {
			continue
		}
		if _, ok := used[modelID]; ok {
			continue
		}

		used[modelID] = struct{}{}
		orderedIDs = append(orderedIDs, modelID)
	}

	refs := make([]productbpdom.ModelRef, 0, len(orderedIDs))
	for i, modelID := range orderedIDs {
		refs = append(refs, productbpdom.ModelRef{
			ModelID:      modelID,
			DisplayOrder: i + 1,
		})
	}

	return refs
}

func modelVariationID(v modeldom.ModelVariation) string {
	if v == nil {
		return ""
	}

	return v.GetID()
}

func modelVariationProductBlueprintID(v modeldom.ModelVariation) string {
	switch x := v.(type) {
	case modeldom.ApparelModelVariation:
		return x.ProductBlueprintID
	case modeldom.AlcoholModelVariation:
		return x.ProductBlueprintID
	default:
		return ""
	}
}
