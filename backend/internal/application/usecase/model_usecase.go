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

// ProductBlueprintModelRefPortは、models collectionを正として
// ProductBlueprintのmodelRefsを同期するためのportです。
//
// ProductBlueprintの通常更新は扱わず、modelRefsの置換だけを扱います。
// ReplaceModelRefsWithoutTouchはupdatedAtとupdatedByを変更しません。
type ProductBlueprintModelRefPort interface {
	GetByID(
		ctx context.Context,
		id string,
	) (productbpdom.ProductBlueprint, error)

	ReplaceModelRefsWithoutTouch(
		ctx context.Context,
		id string,
		refs []productbpdom.ModelRef,
	) (productbpdom.ProductBlueprint, error)
}

// ------------------------------------------------------------
// ModelUsecase
// ------------------------------------------------------------

// ModelUsecaseはcategory-specificなModel variationの操作を担当します。
//
// Product-level metadataはProductBlueprint.CategoryFieldsを正とします。
// Apparelではsize、color、measurementsを扱います。
// Alcoholではvolumeだけを扱います。
//
// models collectionを正として、変更後に
// ProductBlueprint.modelRefsを同期します。
type ModelUsecase struct {
	repo modeldom.RepositoryPort

	productBlueprintRefs ProductBlueprintModelRefPort
}

var _ modeldom.RepositoryPort = (*ModelUsecase)(nil)

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
	if u == nil || u.repo == nil {
		return nil, modeldom.ErrNotFound
	}

	if productBlueprintID == "" {
		return nil, modeldom.ErrInvalidBlueprintID
	}

	return u.repo.ListByProductBlueprintID(
		ctx,
		productBlueprintID,
	)
}

// GetByIDはvariation IDからcategory-specificなModel variationを取得します。
func (u *ModelUsecase) GetByID(
	ctx context.Context,
	variationID string,
) (modeldom.ModelVariation, error) {
	if u == nil || u.repo == nil {
		return nil, modeldom.ErrNotFound
	}

	if variationID == "" {
		return nil, modeldom.ErrInvalidID
	}

	return u.repo.GetByID(
		ctx,
		variationID,
	)
}

// Createはcategory-specificなModel variationを作成します。
//
// 作成後、models collectionを正として
// ProductBlueprint.modelRefsを同期します。
func (u *ModelUsecase) Create(
	ctx context.Context,
	variation modeldom.NewModelVariation,
) (modeldom.ModelVariation, error) {
	if u == nil || u.repo == nil {
		return nil, modeldom.ErrNotFound
	}

	if err := variation.Validate(); err != nil {
		return nil, err
	}

	productBlueprintID := variation.ProductBlueprintID()
	if productBlueprintID == "" {
		return nil, modeldom.ErrInvalidBlueprintID
	}

	created, err := u.repo.Create(
		ctx,
		variation,
	)
	if err != nil {
		return nil, err
	}

	if err := u.syncProductBlueprintModelRefs(
		ctx,
		productBlueprintID,
	); err != nil {
		return nil, err
	}

	return created, nil
}

// Updateはcategory-specificなModel variationを更新します。
//
// 既存variationを取得してkindを確定した後、
// kindに対して更新内容が有効であることを検証します。
// Model IDは変わらないためmodelRefsの同期は行いません。
func (u *ModelUsecase) Update(
	ctx context.Context,
	variationID string,
	updates modeldom.ModelVariationUpdate,
) (modeldom.ModelVariation, error) {
	if u == nil || u.repo == nil {
		return nil, modeldom.ErrNotFound
	}

	if variationID == "" {
		return nil, modeldom.ErrInvalidID
	}

	current, err := u.repo.GetByID(
		ctx,
		variationID,
	)
	if err != nil {
		return nil, err
	}

	if current == nil {
		return nil, modeldom.ErrNotFound
	}

	if err := updates.Validate(
		current.GetKind(),
	); err != nil {
		return nil, err
	}

	return u.repo.Update(
		ctx,
		variationID,
		updates,
	)
}

// Deleteはcategory-specificなModel variationを物理削除します。
//
// 削除前にProductBlueprint IDを取得し、削除後に
// ProductBlueprint.modelRefsを同期します。
func (u *ModelUsecase) Delete(
	ctx context.Context,
	variationID string,
) error {
	if u == nil || u.repo == nil {
		return modeldom.ErrNotFound
	}

	if variationID == "" {
		return modeldom.ErrInvalidID
	}

	current, err := u.repo.GetByID(
		ctx,
		variationID,
	)
	if err != nil {
		return err
	}

	if current == nil {
		return modeldom.ErrNotFound
	}

	productBlueprintID := current.GetProductBlueprintID()
	if productBlueprintID == "" {
		return modeldom.ErrInvalidBlueprintID
	}

	if err := u.repo.Delete(
		ctx,
		variationID,
	); err != nil {
		return err
	}

	return u.syncProductBlueprintModelRefs(
		ctx,
		productBlueprintID,
	)
}

// ReplaceByProductBlueprintIDは、指定されたProductBlueprintに属する
// Model variationを一括置換します。
//
// 既存variationの削除と新規variationの作成は、Repository側の
// 単一transaction内で実行されます。
//
// 置換後、作成されたModel variationを正として
// ProductBlueprint.modelRefsを同期します。
func (u *ModelUsecase) ReplaceByProductBlueprintID(
	ctx context.Context,
	productBlueprintID string,
	variations []modeldom.NewModelVariation,
) ([]modeldom.ModelVariation, error) {
	if u == nil || u.repo == nil {
		return nil, modeldom.ErrNotFound
	}

	if productBlueprintID == "" {
		return nil, modeldom.ErrInvalidBlueprintID
	}

	for _, variation := range variations {
		if err := variation.Validate(); err != nil {
			return nil, err
		}

		if variation.ProductBlueprintID() != productBlueprintID {
			return nil, modeldom.ErrProductMismatch
		}
	}

	replaced, err := u.repo.ReplaceByProductBlueprintID(
		ctx,
		productBlueprintID,
		variations,
	)
	if err != nil {
		return nil, err
	}

	if err := u.syncProductBlueprintModelRefsFromVariations(
		ctx,
		productBlueprintID,
		replaced,
	); err != nil {
		return nil, err
	}

	return replaced, nil
}

// ------------------------------------------------------------
// ProductBlueprint.modelRefs sync
// ------------------------------------------------------------

func (u *ModelUsecase) syncProductBlueprintModelRefs(
	ctx context.Context,
	productBlueprintID string,
) error {
	if u == nil || u.repo == nil {
		return modeldom.ErrNotFound
	}

	if u.productBlueprintRefs == nil {
		return nil
	}

	if productBlueprintID == "" {
		return modeldom.ErrInvalidBlueprintID
	}

	variations, err := u.repo.ListByProductBlueprintID(
		ctx,
		productBlueprintID,
	)
	if err != nil {
		return err
	}

	return u.syncProductBlueprintModelRefsFromVariations(
		ctx,
		productBlueprintID,
		variations,
	)
}

func (u *ModelUsecase) syncProductBlueprintModelRefsFromVariations(
	ctx context.Context,
	productBlueprintID string,
	variations []modeldom.ModelVariation,
) error {
	if u == nil {
		return modeldom.ErrNotFound
	}

	if u.productBlueprintRefs == nil {
		return nil
	}

	if productBlueprintID == "" {
		return modeldom.ErrInvalidBlueprintID
	}

	productBlueprint, err := u.productBlueprintRefs.GetByID(
		ctx,
		productBlueprintID,
	)
	if err != nil {
		return err
	}

	for _, variation := range variations {
		if variation == nil {
			return modeldom.ErrInvalid
		}

		if variation.GetProductBlueprintID() != productBlueprintID {
			return modeldom.ErrProductMismatch
		}

		if err := variation.Validate(); err != nil {
			return err
		}
	}

	refs := rebuildModelRefsPreservingOrder(
		productBlueprint.ModelRefs,
		variations,
	)

	_, err = u.productBlueprintRefs.ReplaceModelRefsWithoutTouch(
		ctx,
		productBlueprintID,
		refs,
	)

	return err
}

func rebuildModelRefsPreservingOrder(
	current []productbpdom.ModelRef,
	variations []modeldom.ModelVariation,
) []productbpdom.ModelRef {
	existingModelIDs := make(
		map[string]struct{},
		len(variations),
	)

	modelIDsInListOrder := make(
		[]string,
		0,
		len(variations),
	)

	for _, variation := range variations {
		if variation == nil {
			continue
		}

		modelID := variation.GetID()
		if modelID == "" {
			continue
		}

		if _, exists := existingModelIDs[modelID]; exists {
			continue
		}

		existingModelIDs[modelID] = struct{}{}
		modelIDsInListOrder = append(
			modelIDsInListOrder,
			modelID,
		)
	}

	usedModelIDs := make(
		map[string]struct{},
		len(modelIDsInListOrder),
	)

	orderedModelIDs := make(
		[]string,
		0,
		len(modelIDsInListOrder),
	)

	currentCopy := append(
		[]productbpdom.ModelRef(nil),
		current...,
	)

	sort.SliceStable(
		currentCopy,
		func(i, j int) bool {
			return currentCopy[i].DisplayOrder <
				currentCopy[j].DisplayOrder
		},
	)

	for _, ref := range currentCopy {
		modelID := ref.ModelID
		if modelID == "" {
			continue
		}

		if _, exists := existingModelIDs[modelID]; !exists {
			continue
		}

		if _, alreadyUsed := usedModelIDs[modelID]; alreadyUsed {
			continue
		}

		usedModelIDs[modelID] = struct{}{}
		orderedModelIDs = append(
			orderedModelIDs,
			modelID,
		)
	}

	for _, modelID := range modelIDsInListOrder {
		if modelID == "" {
			continue
		}

		if _, alreadyUsed := usedModelIDs[modelID]; alreadyUsed {
			continue
		}

		usedModelIDs[modelID] = struct{}{}
		orderedModelIDs = append(
			orderedModelIDs,
			modelID,
		)
	}

	refs := make(
		[]productbpdom.ModelRef,
		0,
		len(orderedModelIDs),
	)

	for index, modelID := range orderedModelIDs {
		refs = append(
			refs,
			productbpdom.ModelRef{
				ModelID:      modelID,
				DisplayOrder: index + 1,
			},
		)
	}

	return refs
}
