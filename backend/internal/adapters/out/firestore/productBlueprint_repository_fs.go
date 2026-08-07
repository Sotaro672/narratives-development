// backend/internal/adapters/out/firestore/productBlueprint_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbdom "narratives/internal/domain/productBlueprint"
	categorydom "narratives/internal/domain/productBlueprintCategory"
)

const maxModelsPerDeleteTransaction = 498

// ProductBlueprintRepositoryFS implements pbdom.Repository using Firestore.
//
// このRepositoryは通常の作成・参照・更新・物理削除を担当します。
type ProductBlueprintRepositoryFS struct {
	Client *firestore.Client
}

func NewProductBlueprintRepositoryFS(client *firestore.Client) *ProductBlueprintRepositoryFS {
	return &ProductBlueprintRepositoryFS{Client: client}
}

func (r *ProductBlueprintRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("product_blueprints")
}

func (r *ProductBlueprintRepositoryFS) modelsCol() *firestore.CollectionRef {
	return r.Client.Collection("models")
}

// Compile-time check: ensure this satisfies domain port.
var _ pbdom.Repository = (*ProductBlueprintRepositoryFS)(nil)

// Create inserts a new ProductBlueprint (no upsert) from domain CreateInput.
func (r *ProductBlueprintRepositoryFS) Create(
	ctx context.Context,
	in pbdom.CreateInput,
) (pbdom.ProductBlueprint, error) {
	if r == nil || r.Client == nil {
		return pbdom.ProductBlueprint{}, errors.New("firestore client is nil")
	}

	id := in.ID
	if id == "" {
		return pbdom.ProductBlueprint{}, pbdom.ErrInvalidID
	}

	now := time.Now().UTC()
	createdAt := now

	if in.CreatedAt != nil && !in.CreatedAt.IsZero() {
		createdAt = in.CreatedAt.UTC()
	}

	productBlueprint, err := pbdom.New(
		id,
		in.ProductName,
		in.Description,
		in.BrandID,
		in.ProductBlueprintCategory,
		in.CategoryFields,
		in.ProductIdTag,
		in.AssigneeID,
		in.CreatedBy,
		createdAt,
		in.CompanyID,
		validateProductBlueprintCategoryFields,
	)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}

	if len(in.ModelRefs) > 0 {
		refs, err := sanitizeModelRefs(in.ModelRefs)
		if err != nil {
			return pbdom.ProductBlueprint{}, err
		}
		productBlueprint.ModelRefs = refs
	}

	if err := productBlueprint.ValidateCategoryFields(
		validateProductBlueprintCategoryFields,
	); err != nil {
		return pbdom.ProductBlueprint{}, err
	}

	documentReference := r.col().Doc(productBlueprint.ID)

	document, err := productBlueprintToDoc(
		productBlueprint,
		productBlueprint.CreatedAt,
		productBlueprint.UpdatedAt,
	)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}

	document["id"] = productBlueprint.ID

	if _, err := documentReference.Create(ctx, document); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return pbdom.ProductBlueprint{}, pbdom.ErrConflict
		}
		return pbdom.ProductBlueprint{}, err
	}

	snapshot, err := documentReference.Get(ctx)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}

	return docToProductBlueprint(snapshot)
}

// GetByID returns a ProductBlueprint by ID.
func (r *ProductBlueprintRepositoryFS) GetByID(
	ctx context.Context,
	id string,
) (pbdom.ProductBlueprint, error) {
	if r == nil || r.Client == nil {
		return pbdom.ProductBlueprint{}, errors.New("firestore client is nil")
	}

	if id == "" {
		return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
	}

	snapshot, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}

	return docToProductBlueprint(snapshot)
}

// GetIDByModelID returns productBlueprintID and modelRefs for the
// ProductBlueprint that owns the given modelID.
//
// 方針:
//   - models/{modelID}.productBlueprintIdを正としてProductBlueprintを特定する。
//   - productBlueprintIdだけが必要なcallerは第1戻り値を使う。
//   - displayOrderが必要なcallerは第2戻り値のmodelRefsから対象modelIdを探す。
func (r *ProductBlueprintRepositoryFS) GetIDByModelID(
	ctx context.Context,
	modelID string,
) (string, []pbdom.ModelRef, error) {
	if r == nil || r.Client == nil {
		return "", nil, errors.New("firestore client is nil")
	}

	if modelID == "" {
		return "", nil, pbdom.ErrNotFound
	}

	document, err := r.modelsCol().Doc(modelID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", nil, pbdom.ErrNotFound
		}
		return "", nil, err
	}

	data := document.Data()
	if data == nil {
		return "", nil, pbdom.ErrNotFound
	}

	productBlueprintID, ok := data["productBlueprintId"].(string)
	if !ok || productBlueprintID == "" {
		return "", nil, pbdom.ErrNotFound
	}

	productBlueprint, err := r.GetByID(ctx, productBlueprintID)
	if err != nil {
		return "", nil, err
	}

	return productBlueprintID, cloneModelRefs(productBlueprint.ModelRefs), nil
}

// ListByCompanyID returns ProductBlueprints for the given companyID.
func (r *ProductBlueprintRepositoryFS) ListByCompanyID(
	ctx context.Context,
	companyID string,
) ([]pbdom.ProductBlueprint, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	if companyID == "" {
		return nil, pbdom.ErrInvalidCompanyID
	}

	documentIterator := r.col().
		Where("companyId", "==", companyID).
		Documents(ctx)
	defer documentIterator.Stop()

	productBlueprints := make([]pbdom.ProductBlueprint, 0, 64)

	for {
		snapshot, err := documentIterator.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}

		productBlueprint, err := docToProductBlueprint(snapshot)
		if err != nil {
			return nil, err
		}

		productBlueprints = append(productBlueprints, productBlueprint)
	}

	return productBlueprints, nil
}

// ListIDsByBrandID returns blueprint IDs for the given brandID.
func (r *ProductBlueprintRepositoryFS) ListIDsByBrandID(
	ctx context.Context,
	brandID string,
) ([]string, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	if brandID == "" {
		return nil, pbdom.ErrNotFound
	}

	documentIterator := r.col().
		Where("brandId", "==", brandID).
		Documents(ctx)
	defer documentIterator.Stop()

	ids := make([]string, 0)

	for {
		snapshot, err := documentIterator.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}

		productBlueprint, err := docToProductBlueprint(snapshot)
		if err != nil {
			return nil, err
		}

		ids = append(ids, productBlueprint.ID)
	}

	return ids, nil
}

// ReplaceModelRefsWithoutTouch replaces modelRefs only, without touching
// updatedAt/updatedBy.
//
//   - printed=falseの場合だけ更新する。
//   - updatedAt / updatedByを更新しない。
//   - modelRefsのみ部分更新する。
//   - refsはdisplayOrder昇順に正規化し、1..Nに再採番する。
//   - refsが空の場合はmodelRefsを空配列に置き換える。
func (r *ProductBlueprintRepositoryFS) ReplaceModelRefsWithoutTouch(
	ctx context.Context,
	id string,
	refs []pbdom.ModelRef,
) (pbdom.ProductBlueprint, error) {
	if r == nil || r.Client == nil {
		return pbdom.ProductBlueprint{}, errors.New("firestore client is nil")
	}

	if id == "" {
		return pbdom.ProductBlueprint{}, pbdom.ErrInvalidID
	}

	normalizedRefs, err := sanitizeModelRefs(refs)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}

	documentReference := r.col().Doc(id)
	var result pbdom.ProductBlueprint

	err = r.Client.RunTransaction(
		ctx,
		func(ctx context.Context, transaction *firestore.Transaction) error {
			snapshot, err := transaction.Get(documentReference)
			if err != nil {
				return mapFirestoreNotFound(err)
			}

			productBlueprint, err := docToProductBlueprint(snapshot)
			if err != nil {
				return err
			}

			if !productBlueprint.CanModify() {
				return pbdom.ErrForbidden
			}

			if err := productBlueprint.ReplaceModelRefsWithoutTouch(
				normalizedRefs,
			); err != nil {
				return err
			}

			modelRefsDocument := modelRefsToDoc(productBlueprint.ModelRefs)

			if err := transaction.Update(
				documentReference,
				[]firestore.Update{
					{
						Path:  "modelRefs",
						Value: modelRefsDocument,
					},
				},
			); err != nil {
				return mapFirestoreNotFound(err)
			}

			result = productBlueprint
			return nil
		},
	)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}

	return result, nil
}

// Update updates an unprinted ProductBlueprint by patch.
func (r *ProductBlueprintRepositoryFS) Update(
	ctx context.Context,
	id string,
	patch pbdom.Patch,
) (pbdom.ProductBlueprint, error) {
	if r == nil || r.Client == nil {
		return pbdom.ProductBlueprint{}, errors.New("firestore client is nil")
	}

	if id == "" {
		return pbdom.ProductBlueprint{}, pbdom.ErrInvalidID
	}

	documentReference := r.col().Doc(id)
	var result pbdom.ProductBlueprint

	err := r.Client.RunTransaction(
		ctx,
		func(ctx context.Context, transaction *firestore.Transaction) error {
			snapshot, err := transaction.Get(documentReference)
			if err != nil {
				return mapFirestoreNotFound(err)
			}

			productBlueprint, err := docToProductBlueprint(snapshot)
			if err != nil {
				return err
			}

			if !productBlueprint.CanModify() {
				return pbdom.ErrForbidden
			}

			now := time.Now().UTC()

			if patch.ProductName != nil {
				if err := productBlueprint.UpdateProductName(
					*patch.ProductName,
					now,
					patch.UpdatedBy,
				); err != nil {
					return err
				}
			}

			if patch.Description != nil {
				if err := productBlueprint.UpdateDescription(
					*patch.Description,
					now,
					patch.UpdatedBy,
				); err != nil {
					return err
				}
			}

			if patch.BrandID != nil {
				if err := productBlueprint.UpdateBrand(
					*patch.BrandID,
					now,
					patch.UpdatedBy,
				); err != nil {
					return err
				}
			}

			if patch.CompanyID != nil {
				if *patch.CompanyID == "" {
					return pbdom.ErrInvalidCompanyID
				}

				productBlueprint.CompanyID = *patch.CompanyID
				productBlueprint.UpdatedAt = now
				productBlueprint.UpdatedBy = patch.UpdatedBy
			}

			switch {
			case patch.ProductBlueprintCategory != nil &&
				patch.CategoryFields != nil:
				if err := productBlueprint.UpdateCategoryAndFields(
					*patch.ProductBlueprintCategory,
					*patch.CategoryFields,
					validateProductBlueprintCategoryFields,
					now,
					patch.UpdatedBy,
				); err != nil {
					return err
				}

			case patch.ProductBlueprintCategory != nil:
				if err := productBlueprint.UpdateCategory(
					*patch.ProductBlueprintCategory,
					validateProductBlueprintCategoryFields,
					now,
					patch.UpdatedBy,
				); err != nil {
					return err
				}

			case patch.CategoryFields != nil:
				if err := productBlueprint.UpdateCategoryFields(
					*patch.CategoryFields,
					validateProductBlueprintCategoryFields,
					now,
					patch.UpdatedBy,
				); err != nil {
					return err
				}
			}

			if patch.ProductIdTag != nil {
				if err := productBlueprint.UpdateTag(
					*patch.ProductIdTag,
					now,
					patch.UpdatedBy,
				); err != nil {
					return err
				}
			}

			if patch.AssigneeID != nil {
				if err := productBlueprint.UpdateAssignee(
					*patch.AssigneeID,
					now,
					patch.UpdatedBy,
				); err != nil {
					return err
				}
			}

			if patch.ModelRefs != nil {
				normalizedRefs, err := sanitizeModelRefs(*patch.ModelRefs)
				if err != nil {
					return err
				}

				modelIDs := make([]string, 0, len(normalizedRefs))
				for _, modelRef := range normalizedRefs {
					modelIDs = append(modelIDs, modelRef.ModelID)
				}

				if err := productBlueprint.UpdateModelIDs(
					modelIDs,
					now,
					patch.UpdatedBy,
				); err != nil {
					return err
				}
			}

			if err := productBlueprint.ValidateCategoryFields(
				validateProductBlueprintCategoryFields,
			); err != nil {
				return err
			}

			document, err := productBlueprintToDoc(
				productBlueprint,
				productBlueprint.CreatedAt,
				productBlueprint.UpdatedAt,
			)
			if err != nil {
				return err
			}

			document["id"] = productBlueprint.ID

			if err := transaction.Set(
				documentReference,
				document,
			); err != nil {
				return mapFirestoreNotFound(err)
			}

			result = productBlueprint
			return nil
		},
	)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}

	return result, nil
}

// MarkPrinted sets printed=true on a ProductBlueprint and returns it.
func (r *ProductBlueprintRepositoryFS) MarkPrinted(
	ctx context.Context,
	id string,
) (pbdom.ProductBlueprint, error) {
	if r == nil || r.Client == nil {
		return pbdom.ProductBlueprint{}, errors.New("firestore client is nil")
	}

	if id == "" {
		return pbdom.ProductBlueprint{}, pbdom.ErrInvalidID
	}

	documentReference := r.col().Doc(id)
	var result pbdom.ProductBlueprint

	err := r.Client.RunTransaction(
		ctx,
		func(ctx context.Context, transaction *firestore.Transaction) error {
			snapshot, err := transaction.Get(documentReference)
			if err != nil {
				return mapFirestoreNotFound(err)
			}

			productBlueprint, err := docToProductBlueprint(snapshot)
			if err != nil {
				return err
			}

			if productBlueprint.Printed {
				result = productBlueprint
				return nil
			}

			now := time.Now().UTC()

			if err := productBlueprint.MarkPrinted(
				now,
				productBlueprint.UpdatedBy,
				validateProductBlueprintCategoryFields,
			); err != nil {
				return err
			}

			updates := []firestore.Update{
				{
					Path:  "printed",
					Value: true,
				},
				{
					Path:  "updatedAt",
					Value: productBlueprint.UpdatedAt,
				},
			}

			updates = appendOptionalStringUpdate(
				updates,
				"updatedBy",
				productBlueprint.UpdatedBy,
			)

			if err := transaction.Update(
				documentReference,
				updates,
			); err != nil {
				return mapFirestoreNotFound(err)
			}

			result = productBlueprint
			return nil
		},
	)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}

	return result, nil
}

// Delete physically deletes an unprinted ProductBlueprint and all of its models.
//
// models collectionのproductBlueprintId == idを正として配下Modelを取得し、
// 同一Transaction内でModelを先に削除した後、
// productBlueprintReviewAggregates/{id}とProductBlueprint本体を削除します。
func (r *ProductBlueprintRepositoryFS) Delete(
	ctx context.Context,
	id string,
	companyID string,
) error {
	if r == nil || r.Client == nil {
		return errors.New("firestore client is nil")
	}

	if id == "" {
		return pbdom.ErrInvalidID
	}

	if companyID == "" {
		return pbdom.ErrInvalidCompanyID
	}

	documentReference := r.col().Doc(id)
	reviewAggregateReference := r.Client.
		Collection("productBlueprintReviewAggregates").
		Doc(id)

	return r.Client.RunTransaction(
		ctx,
		func(ctx context.Context, transaction *firestore.Transaction) error {
			snapshot, err := transaction.Get(documentReference)
			if err != nil {
				return mapFirestoreNotFound(err)
			}

			productBlueprint, err := docToProductBlueprint(snapshot)
			if err != nil {
				return err
			}

			if productBlueprint.CompanyID == "" ||
				productBlueprint.CompanyID != companyID {
				return pbdom.ErrForbidden
			}

			if productBlueprint.Printed {
				return pbdom.ErrForbidden
			}

			modelSnapshots, err := r.listModelSnapshotsInTransaction(
				transaction,
				id,
			)
			if err != nil {
				return err
			}

			if len(modelSnapshots) > maxModelsPerDeleteTransaction {
				return pbdom.WrapConflict(
					nil,
					"too many models for one delete transaction",
				)
			}

			for _, modelSnapshot := range modelSnapshots {
				if modelSnapshot == nil || modelSnapshot.Ref == nil {
					continue
				}

				if err := transaction.Delete(modelSnapshot.Ref); err != nil {
					return err
				}
			}

			if err := transaction.Delete(reviewAggregateReference); err != nil {
				return err
			}

			if err := transaction.Delete(documentReference); err != nil {
				return err
			}

			return nil
		},
	)
}

func (r *ProductBlueprintRepositoryFS) listModelSnapshotsInTransaction(
	transaction *firestore.Transaction,
	productBlueprintID string,
) ([]*firestore.DocumentSnapshot, error) {
	documentIterator := transaction.Documents(
		r.modelsCol().Where(
			"productBlueprintId",
			"==",
			productBlueprintID,
		),
	)
	defer documentIterator.Stop()

	snapshots := make([]*firestore.DocumentSnapshot, 0)

	for {
		snapshot, err := documentIterator.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}

		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

type productBlueprintCategorySnapshotDoc struct {
	ID     string   `firestore:"productBlueprintCategoryId"`
	Code   string   `firestore:"productBlueprintCategoryCode"`
	NameJa string   `firestore:"productBlueprintCategoryNameJa"`
	NameEn string   `firestore:"productBlueprintCategoryNameEn"`
	Kind   string   `firestore:"productBlueprintCategoryKind"`
	Path   []string `firestore:"productBlueprintCategoryPath"`
}

func docToProductBlueprint(
	document *firestore.DocumentSnapshot,
) (pbdom.ProductBlueprint, error) {
	if document == nil {
		return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
	}

	data := document.Data()
	if data == nil {
		return pbdom.ProductBlueprint{},
			fmt.Errorf(
				"empty product_blueprints document: %s",
				document.Ref.ID,
			)
	}

	getString := func(key string) string {
		value, _ := data[key].(string)
		return value
	}

	getStringPointer := func(key string) *string {
		value, ok := data[key].(string)
		if !ok || value == "" {
			return nil
		}
		return &value
	}

	getTime := func(key string) time.Time {
		value, ok := data[key].(time.Time)
		if !ok || value.IsZero() {
			return time.Time{}
		}
		return value.UTC()
	}

	printed, _ := data["printed"].(bool)

	category, err := docToProductBlueprintCategorySnapshot(document)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}

	categoryFields, err := docValueToCategoryFields(data["categoryFields"])
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}

	modelRefs, err := docToModelRefs(data["modelRefs"])
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}

	id := getString("id")
	if id == "" {
		id = document.Ref.ID
	}

	productBlueprint := pbdom.ProductBlueprint{
		ID:          id,
		ProductName: getString("productName"),
		Description: getString("description"),
		BrandID:     getString("brandId"),
		CompanyID:   getString("companyId"),

		ProductBlueprintCategory: category,
		CategoryFields:           categoryFields,

		ProductIdTag: pbdom.ProductIDTag{
			Type: pbdom.ProductIDTagType(
				getString("productIdTagType"),
			),
		},

		AssigneeID: getString("assigneeId"),
		ModelRefs:  modelRefs,
		Printed:    printed,

		CreatedBy: getStringPointer("createdBy"),
		CreatedAt: getTime("createdAt"),
		UpdatedBy: getStringPointer("updatedBy"),
		UpdatedAt: getTime("updatedAt"),
	}

	if err := productBlueprint.ValidateCategoryFields(
		validateProductBlueprintCategoryFields,
	); err != nil {
		return pbdom.ProductBlueprint{}, err
	}

	return productBlueprint, nil
}

func docToProductBlueprintCategorySnapshot(
	document *firestore.DocumentSnapshot,
) (pbdom.ProductBlueprintCategorySnapshot, error) {
	if document == nil {
		return pbdom.ProductBlueprintCategorySnapshot{}, pbdom.ErrNotFound
	}

	var categoryDocument productBlueprintCategorySnapshotDoc
	if err := document.DataTo(&categoryDocument); err != nil {
		return pbdom.ProductBlueprintCategorySnapshot{}, err
	}

	category := pbdom.ProductBlueprintCategorySnapshot{
		ID:     categoryDocument.ID,
		Code:   categoryDocument.Code,
		NameJa: categoryDocument.NameJa,
		NameEn: categoryDocument.NameEn,
		Kind:   categorydom.CategoryKind(categoryDocument.Kind),
		Path:   append([]string(nil), categoryDocument.Path...),
	}

	if err := category.Validate(); err != nil {
		return pbdom.ProductBlueprintCategorySnapshot{}, err
	}

	return category, nil
}

func docToModelRefs(raw any) ([]pbdom.ModelRef, error) {
	if raw == nil {
		return nil, nil
	}

	items, ok := raw.([]any)
	if !ok {
		return nil,
			pbdom.WrapInvalid(
				nil,
				"modelRefs must be []interface{}",
			)
	}

	modelRefs := make([]pbdom.ModelRef, 0, len(items))

	for _, item := range items {
		record, ok := item.(map[string]any)
		if !ok || record == nil {
			return nil,
				pbdom.WrapInvalid(
					nil,
					"modelRefs item must be map[string]interface{}",
				)
		}

		modelID, ok := record["modelId"].(string)
		if !ok || modelID == "" {
			return nil,
				pbdom.WrapInvalid(
					nil,
					"modelRefs.modelId must be string",
				)
		}

		displayOrder64, ok := record["displayOrder"].(int64)
		if !ok || displayOrder64 <= 0 {
			return nil,
				pbdom.WrapInvalid(
					nil,
					"modelRefs.displayOrder must be int64",
				)
		}

		displayOrder := int(displayOrder64)
		if int64(displayOrder) != displayOrder64 {
			return nil,
				pbdom.WrapInvalid(
					nil,
					"modelRefs.displayOrder is out of int range",
				)
		}

		modelRefs = append(
			modelRefs,
			pbdom.ModelRef{
				ModelID:      modelID,
				DisplayOrder: displayOrder,
			},
		)
	}

	sort.SliceStable(
		modelRefs,
		func(i, j int) bool {
			return modelRefs[i].DisplayOrder <
				modelRefs[j].DisplayOrder
		},
	)

	return sanitizeModelRefs(modelRefs)
}

func productBlueprintToDoc(
	productBlueprint pbdom.ProductBlueprint,
	createdAt time.Time,
	updatedAt time.Time,
) (map[string]any, error) {
	if err := productBlueprint.ValidateCategoryFields(
		validateProductBlueprintCategoryFields,
	); err != nil {
		return nil, err
	}

	category := productBlueprint.ProductBlueprintCategory

	document := map[string]any{
		"productName": productBlueprint.ProductName,
		"description": productBlueprint.Description,
		"brandId":     productBlueprint.BrandID,
		"companyId":   productBlueprint.CompanyID,

		"productBlueprintCategoryId":     category.ID,
		"productBlueprintCategoryCode":   category.Code,
		"productBlueprintCategoryNameJa": category.NameJa,
		"productBlueprintCategoryNameEn": category.NameEn,
		"productBlueprintCategoryKind":   string(category.Kind),
		"productBlueprintCategoryPath":   append([]string(nil), category.Path...),

		"assigneeId": productBlueprint.AssigneeID,
		"createdAt":  createdAt.UTC(),
		"updatedAt":  updatedAt.UTC(),
		"printed":    productBlueprint.Printed,
	}

	if productBlueprint.CategoryFields != nil {
		categoryFieldsDocument, err := categoryFieldsToDoc(
			productBlueprint.CategoryFields,
		)
		if err != nil {
			return nil, err
		}

		document["categoryFields"] = categoryFieldsDocument
	}

	if productBlueprint.ProductIdTag.Type != "" {
		document["productIdTagType"] =
			string(productBlueprint.ProductIdTag.Type)
	}

	if productBlueprint.ModelRefs != nil {
		normalizedRefs, err := sanitizeModelRefs(productBlueprint.ModelRefs)
		if err != nil {
			return nil, err
		}

		modelRefsDocument := make(
			[]map[string]any,
			0,
			len(normalizedRefs),
		)

		for _, modelRef := range normalizedRefs {
			modelRefsDocument = append(
				modelRefsDocument,
				map[string]any{
					"modelId":      modelRef.ModelID,
					"displayOrder": modelRef.DisplayOrder,
				},
			)
		}

		document["modelRefs"] = modelRefsDocument
	}

	if productBlueprint.CreatedBy != nil &&
		*productBlueprint.CreatedBy != "" {
		document["createdBy"] = *productBlueprint.CreatedBy
	}

	if productBlueprint.UpdatedBy != nil &&
		*productBlueprint.UpdatedBy != "" {
		document["updatedBy"] = *productBlueprint.UpdatedBy
	}

	return document, nil
}

func mapFirestoreNotFound(err error) error {
	if status.Code(err) == codes.NotFound {
		return pbdom.ErrNotFound
	}
	return err
}

func appendOptionalStringUpdate(
	updates []firestore.Update,
	path string,
	value *string,
) []firestore.Update {
	if value == nil || *value == "" {
		return append(
			updates,
			firestore.Update{
				Path:  path,
				Value: firestore.Delete,
			},
		)
	}

	return append(
		updates,
		firestore.Update{
			Path:  path,
			Value: *value,
		},
	)
}

func modelRefsToDoc(
	modelRefs []pbdom.ModelRef,
) []map[string]any {
	documents := make(
		[]map[string]any,
		0,
		len(modelRefs),
	)

	for _, modelRef := range modelRefs {
		documents = append(
			documents,
			map[string]any{
				"modelId":      modelRef.ModelID,
				"displayOrder": modelRef.DisplayOrder,
			},
		)
	}

	return documents
}

func validateProductBlueprintCategoryFields(
	category pbdom.ProductBlueprintCategorySnapshot,
	fields pbdom.CategoryFields,
) error {
	schema, ok := categorydom.GetCategoryInputSchema(category.Code)
	if !ok {
		return pbdom.WrapInvalid(
			pbdom.ErrInvalidCategoryFields,
			"category input schema is not registered",
		)
	}

	if schema.CategoryKind != string(category.Kind) {
		return pbdom.WrapInvalid(
			pbdom.ErrInvalidCategoryFields,
			"category input schema kind mismatch",
		)
	}

	definitions := make(
		map[string]categorydom.CategoryInputFieldDefinition,
		len(schema.ProductBlueprintFields),
	)

	for _, definition := range schema.ProductBlueprintFields {
		if isCommonProductBlueprintField(definition.Key) {
			continue
		}

		definitions[definition.Key] = definition
	}

	for key := range fields {
		if key == "" {
			return pbdom.WrapInvalid(
				pbdom.ErrInvalidCategoryFields,
				"categoryFields key is empty",
			)
		}

		if _, exists := definitions[key]; !exists {
			return pbdom.WrapInvalid(
				pbdom.ErrInvalidCategoryFields,
				fmt.Sprintf(
					"categoryFields.%s is not allowed for category %s",
					key,
					category.Code,
				),
			)
		}
	}

	for key, definition := range definitions {
		value, exists := fields[key]
		if !exists || value == nil {
			if definition.Required {
				return pbdom.WrapInvalid(
					pbdom.ErrInvalidCategoryFields,
					fmt.Sprintf(
						"categoryFields.%s is required",
						key,
					),
				)
			}
			continue
		}

		if err := validateCategoryFieldValue(
			key,
			value,
			definition,
		); err != nil {
			return err
		}
	}

	return nil
}

func isCommonProductBlueprintField(key string) bool {
	switch key {
	case "brandId",
		"productName",
		"productIdTagType",
		"description":
		return true
	default:
		return false
	}
}

func validateCategoryFieldValue(
	key string,
	value any,
	definition categorydom.CategoryInputFieldDefinition,
) error {
	switch definition.Type {
	case categorydom.InputFieldTypeText,
		categorydom.InputFieldTypeTextarea,
		categorydom.InputFieldTypeSelect,
		categorydom.InputFieldTypeDate:
		text, ok := value.(string)
		if !ok {
			return invalidCategoryFieldType(
				key,
				string(definition.Type),
			)
		}

		if definition.Required && text == "" {
			return pbdom.WrapInvalid(
				pbdom.ErrInvalidCategoryFields,
				fmt.Sprintf(
					"categoryFields.%s is required",
					key,
				),
			)
		}

	case categorydom.InputFieldTypeNumber:
		number, ok := categoryFieldNumber(value)
		if !ok ||
			math.IsNaN(number) ||
			math.IsInf(number, 0) {
			return invalidCategoryFieldType(
				key,
				string(definition.Type),
			)
		}

		switch key {
		case "weight":
			if number < 0 {
				return invalidCategoryFieldValue(
					key,
					"must be >= 0",
				)
			}

		case "alcoholContent":
			if number < 0 || number > 100 {
				return invalidCategoryFieldValue(
					key,
					"must be between 0 and 100",
				)
			}
		}

	case categorydom.InputFieldTypeMultiSelect:
		if !isStringSlice(value) {
			return invalidCategoryFieldType(
				key,
				string(definition.Type),
			)
		}

	case categorydom.InputFieldTypeBoolean:
		if _, ok := value.(bool); !ok {
			return invalidCategoryFieldType(
				key,
				string(definition.Type),
			)
		}

	default:
		return pbdom.WrapInvalid(
			pbdom.ErrInvalidCategoryFields,
			fmt.Sprintf(
				"categoryFields.%s has unsupported schema type %s",
				key,
				definition.Type,
			),
		)
	}

	return nil
}

func categoryFieldNumber(value any) (float64, bool) {
	switch number := value.(type) {
	case int:
		return float64(number), true

	case int64:
		return float64(number), true

	case float64:
		return number, true

	default:
		return 0, false
	}
}

func isStringSlice(value any) bool {
	switch values := value.(type) {
	case []string:
		return true

	case []any:
		for _, item := range values {
			if _, ok := item.(string); !ok {
				return false
			}
		}
		return true

	default:
		return false
	}
}

func invalidCategoryFieldType(
	key string,
	expected string,
) error {
	return pbdom.WrapInvalid(
		pbdom.ErrInvalidCategoryFields,
		fmt.Sprintf(
			"categoryFields.%s must be %s",
			key,
			expected,
		),
	)
}

func invalidCategoryFieldValue(
	key string,
	requirement string,
) error {
	return pbdom.WrapInvalid(
		pbdom.ErrInvalidCategoryFields,
		fmt.Sprintf(
			"categoryFields.%s %s",
			key,
			requirement,
		),
	)
}

func sanitizeModelRefs(
	input []pbdom.ModelRef,
) ([]pbdom.ModelRef, error) {
	type indexedModelRef struct {
		ref   pbdom.ModelRef
		index int
	}

	indexedRefs := make(
		[]indexedModelRef,
		0,
		len(input),
	)

	for index, modelRef := range input {
		indexedRefs = append(
			indexedRefs,
			indexedModelRef{
				ref:   modelRef,
				index: index,
			},
		)
	}

	sort.SliceStable(
		indexedRefs,
		func(i, j int) bool {
			if indexedRefs[i].ref.DisplayOrder ==
				indexedRefs[j].ref.DisplayOrder {
				return indexedRefs[i].index <
					indexedRefs[j].index
			}

			return indexedRefs[i].ref.DisplayOrder <
				indexedRefs[j].ref.DisplayOrder
		},
	)

	seen := make(
		map[string]struct{},
		len(indexedRefs),
	)

	modelIDs := make(
		[]string,
		0,
		len(indexedRefs),
	)

	for _, indexedRef := range indexedRefs {
		modelID := indexedRef.ref.ModelID
		if modelID == "" {
			continue
		}

		if _, exists := seen[modelID]; exists {
			continue
		}

		seen[modelID] = struct{}{}
		modelIDs = append(modelIDs, modelID)
	}

	normalized := make(
		[]pbdom.ModelRef,
		0,
		len(modelIDs),
	)

	for index, modelID := range modelIDs {
		normalized = append(
			normalized,
			pbdom.ModelRef{
				ModelID:      modelID,
				DisplayOrder: index + 1,
			},
		)
	}

	return normalized, nil
}

func cloneModelRefs(input []pbdom.ModelRef) []pbdom.ModelRef {
	if input == nil {
		return nil
	}

	output := make([]pbdom.ModelRef, len(input))
	copy(output, input)

	return output
}

func cloneCategoryFields(
	input pbdom.CategoryFields,
) pbdom.CategoryFields {
	if input == nil {
		return nil
	}

	output := make(
		pbdom.CategoryFields,
		len(input),
	)

	for key, value := range input {
		if key == "" {
			continue
		}

		output[key] = cloneCategoryFieldValue(value)
	}

	if len(output) == 0 {
		return nil
	}

	return output
}

func cloneCategoryFieldValue(value any) any {
	switch typedValue := value.(type) {
	case []string:
		return append([]string(nil), typedValue...)

	case []any:
		return append([]any(nil), typedValue...)

	default:
		return value
	}
}

func docValueToCategoryFields(
	raw any,
) (pbdom.CategoryFields, error) {
	if raw == nil {
		return nil, nil
	}

	record, ok := raw.(map[string]any)
	if !ok {
		return nil,
			pbdom.WrapInvalid(
				pbdom.ErrInvalidCategoryFields,
				"categoryFields must be map[string]interface{}",
			)
	}

	output := make(
		pbdom.CategoryFields,
		len(record),
	)

	for key, value := range record {
		if key == "" {
			return nil,
				pbdom.WrapInvalid(
					pbdom.ErrInvalidCategoryFields,
					"categoryFields key is empty",
				)
		}

		output[key] = cloneCategoryFieldValue(value)
	}

	if len(output) == 0 {
		return nil, nil
	}

	return output, nil
}

func categoryFieldsToDoc(
	input pbdom.CategoryFields,
) (map[string]any, error) {
	if input == nil {
		return nil, nil
	}

	output := make(
		map[string]any,
		len(input),
	)

	for key, value := range input {
		if key == "" {
			return nil,
				pbdom.WrapInvalid(
					pbdom.ErrInvalidCategoryFields,
					"categoryFields key is empty",
				)
		}

		output[key] = cloneCategoryFieldValue(value)
	}

	return output, nil
}
