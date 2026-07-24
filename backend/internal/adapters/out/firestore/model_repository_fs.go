// backend/internal/adapters/out/firestore/model_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	modeldom "narratives/internal/domain/model"
)

// Firestoreの原子的Replaceで使用する最大書込み数。
// 既存documentの削除数と新規documentの作成数の合計で判定する。
const maxAtomicModelReplaceWrites = 450

// ------------------------------------------------------------
// Repository struct
// ------------------------------------------------------------

type ModelRepositoryFS struct {
	Client *firestore.Client
}

func NewModelRepositoryFS(
	client *firestore.Client,
) *ModelRepositoryFS {
	return &ModelRepositoryFS{
		Client: client,
	}
}

func (r *ModelRepositoryFS) variationsCol() *firestore.CollectionRef {
	return r.Client.Collection("models")
}

// ------------------------------------------------------------
// RepositoryPort implementation
// ------------------------------------------------------------

func (r *ModelRepositoryFS) ListByProductBlueprintID(
	ctx context.Context,
	productBlueprintID string,
) ([]modeldom.ModelVariation, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	if productBlueprintID == "" {
		return nil, modeldom.ErrInvalidBlueprintID
	}

	variations, err := r.listVariationsByProductBlueprintID(
		ctx,
		productBlueprintID,
	)
	if err != nil {
		return nil, err
	}

	sortModelVariations(variations)

	return variations, nil
}

func (r *ModelRepositoryFS) GetByID(
	ctx context.Context,
	variationID string,
) (modeldom.ModelVariation, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	if variationID == "" {
		return nil, modeldom.ErrNotFound
	}

	snapshot, err := r.variationsCol().
		Doc(variationID).
		Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, modeldom.ErrNotFound
		}

		return nil, err
	}

	return docToModelVariation(snapshot)
}

func (r *ModelRepositoryFS) Create(
	ctx context.Context,
	variation modeldom.NewModelVariation,
) (modeldom.ModelVariation, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	if err := variation.Validate(); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	documentReference := r.variationsCol().NewDoc()

	modelVariation, err := newModelVariationToDomain(
		documentReference.ID,
		variation,
		now,
	)
	if err != nil {
		return nil, err
	}

	document, err := modelVariationToDoc(modelVariation)
	if err != nil {
		return nil, err
	}

	if _, err := documentReference.Create(
		ctx,
		document,
	); err != nil {
		return nil, err
	}

	snapshot, err := documentReference.Get(ctx)
	if err != nil {
		return nil, err
	}

	return docToModelVariation(snapshot)
}

func (r *ModelRepositoryFS) Update(
	ctx context.Context,
	variationID string,
	updates modeldom.ModelVariationUpdate,
) (modeldom.ModelVariation, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	if variationID == "" {
		return nil, modeldom.ErrNotFound
	}

	current, err := r.GetByID(ctx, variationID)
	if err != nil {
		return nil, err
	}

	if err := updates.Validate(current.GetKind()); err != nil {
		return nil, err
	}

	firestoreUpdates := make(
		[]firestore.Update,
		0,
		6,
	)

	if updates.Size != nil {
		firestoreUpdates = append(
			firestoreUpdates,
			firestore.Update{
				Path:  "size",
				Value: *updates.Size,
			},
		)
	}

	if updates.Color != nil {
		firestoreUpdates = append(
			firestoreUpdates,
			firestore.Update{
				Path: "color",
				Value: map[string]any{
					"name": updates.Color.Name,
					"rgb":  updates.Color.RGB,
				},
			},
		)
	}

	if updates.ModelNumber != nil {
		firestoreUpdates = append(
			firestoreUpdates,
			firestore.Update{
				Path:  "modelNumber",
				Value: *updates.ModelNumber,
			},
		)
	}

	if updates.Measurements != nil {
		firestoreUpdates = append(
			firestoreUpdates,
			firestore.Update{
				Path:  "measurements",
				Value: updates.Measurements.Clone(),
			},
		)
	}

	if updates.Volume != nil {
		firestoreUpdates = append(
			firestoreUpdates,
			firestore.Update{
				Path: "volume",
				Value: map[string]any{
					"value": updates.Volume.Value,
					"unit":  updates.Volume.Unit,
				},
			},
		)
	}

	firestoreUpdates = append(
		firestoreUpdates,
		firestore.Update{
			Path:  "updatedAt",
			Value: time.Now().UTC(),
		},
	)

	documentReference := r.variationsCol().Doc(variationID)

	if _, err := documentReference.Update(
		ctx,
		firestoreUpdates,
	); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, modeldom.ErrNotFound
		}

		return nil, err
	}

	return r.GetByID(ctx, variationID)
}

func (r *ModelRepositoryFS) Delete(
	ctx context.Context,
	variationID string,
) error {
	if r == nil || r.Client == nil {
		return errors.New("firestore client is nil")
	}

	if variationID == "" {
		return modeldom.ErrNotFound
	}

	documentReference := r.variationsCol().Doc(variationID)

	if _, err := documentReference.Delete(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return modeldom.ErrNotFound
		}

		return err
	}

	return nil
}

// ReplaceByProductBlueprintIDは、指定されたProductBlueprintに属する
// 既存variationの削除と新規variationの作成を単一transactionで実行する。
//
// transaction上限を超える場合は、書込みを開始せず
// ErrAtomicReplaceLimitExceededを返す。
func (r *ModelRepositoryFS) ReplaceByProductBlueprintID(
	ctx context.Context,
	productBlueprintID string,
	variations []modeldom.NewModelVariation,
) ([]modeldom.ModelVariation, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	if productBlueprintID == "" {
		return nil, modeldom.ErrInvalidBlueprintID
	}

	now := time.Now().UTC()

	prepared := make(
		[]preparedModelVariation,
		0,
		len(variations),
	)

	for _, variation := range variations {
		if err := variation.Validate(); err != nil {
			return nil, err
		}

		if variation.ProductBlueprintID() != productBlueprintID {
			return nil, modeldom.ErrProductMismatch
		}

		documentReference := r.variationsCol().NewDoc()

		modelVariation, err := newModelVariationToDomain(
			documentReference.ID,
			variation,
			now,
		)
		if err != nil {
			return nil, err
		}

		document, err := modelVariationToDoc(modelVariation)
		if err != nil {
			return nil, err
		}

		prepared = append(
			prepared,
			preparedModelVariation{
				reference: documentReference,
				document:  document,
				variation: modelVariation,
			},
		)
	}

	query := r.variationsCol().Where(
		"productBlueprintId",
		"==",
		productBlueprintID,
	)

	err := r.Client.RunTransaction(
		ctx,
		func(
			_ context.Context,
			transaction *firestore.Transaction,
		) error {
			documentIterator := transaction.Documents(query)
			defer documentIterator.Stop()

			existingReferences := make(
				[]*firestore.DocumentRef,
				0,
			)

			for {
				snapshot, err := documentIterator.Next()
				if err != nil {
					if err == iterator.Done {
						break
					}

					return err
				}

				existingReferences = append(
					existingReferences,
					snapshot.Ref,
				)
			}

			writeCount := len(existingReferences) + len(prepared)
			if writeCount > maxAtomicModelReplaceWrites {
				return modeldom.ErrAtomicReplaceLimitExceeded
			}

			for _, reference := range existingReferences {
				if err := transaction.Delete(reference); err != nil {
					return err
				}
			}

			for _, item := range prepared {
				if err := transaction.Create(
					item.reference,
					item.document,
				); err != nil {
					return err
				}
			}

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	result := make(
		[]modeldom.ModelVariation,
		0,
		len(prepared),
	)

	for _, item := range prepared {
		result = append(
			result,
			item.variation,
		)
	}

	sortModelVariations(result)

	return result, nil
}

// ------------------------------------------------------------
// Internal types
// ------------------------------------------------------------

type preparedModelVariation struct {
	reference *firestore.DocumentRef
	document  map[string]any
	variation modeldom.ModelVariation
}

// ------------------------------------------------------------
// Query helpers
// ------------------------------------------------------------

func (r *ModelRepositoryFS) listVariationsByProductBlueprintID(
	ctx context.Context,
	productBlueprintID string,
) ([]modeldom.ModelVariation, error) {
	query := r.variationsCol().Where(
		"productBlueprintId",
		"==",
		productBlueprintID,
	)

	documentIterator := query.Documents(ctx)
	defer documentIterator.Stop()

	variations := make(
		[]modeldom.ModelVariation,
		0,
	)

	for {
		document, err := documentIterator.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}

			return nil, err
		}

		variation, err := docToModelVariation(document)
		if err != nil {
			return nil, err
		}

		variations = append(
			variations,
			variation,
		)
	}

	return variations, nil
}

// ------------------------------------------------------------
// Domain conversion
// ------------------------------------------------------------

func newModelVariationToDomain(
	id string,
	input modeldom.NewModelVariation,
	now time.Time,
) (modeldom.ModelVariation, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	switch input.Kind {
	case modeldom.ModelVariationKindAlcohol:
		modelVariation := modeldom.AlcoholModelVariation{
			ID:                 id,
			ProductBlueprintID: input.Alcohol.ProductBlueprintID,
			ModelNumber:        input.Alcohol.ModelNumber,
			Volume:             input.Alcohol.Volume,
			CreatedAt:          now,
			UpdatedAt:          now,
		}

		if err := modelVariation.Validate(); err != nil {
			return nil, err
		}

		return modelVariation, nil

	case modeldom.ModelVariationKindApparel:
		modelVariation := modeldom.ApparelModelVariation{
			ID:                 id,
			ProductBlueprintID: input.Apparel.ProductBlueprintID,
			ModelNumber:        input.Apparel.ModelNumber,
			Size:               input.Apparel.Size,
			Color: modeldom.Color{
				Name: input.Apparel.Color.Name,
				RGB:  input.Apparel.Color.RGB,
			},
			Measurements: input.Apparel.Measurements.Clone(),
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if err := modelVariation.Validate(); err != nil {
			return nil, err
		}

		return modelVariation, nil

	default:
		return nil, modeldom.ErrInvalidKind
	}
}

func docToModelVariation(
	document *firestore.DocumentSnapshot,
) (modeldom.ModelVariation, error) {
	if document == nil {
		return nil, modeldom.ErrInvalid
	}

	data := document.Data()
	if data == nil {
		return nil, fmt.Errorf(
			"empty variation: %s",
			document.Ref.ID,
		)
	}

	kind, err := requiredModelString(
		data,
		"kind",
	)
	if err != nil {
		return nil, err
	}

	productBlueprintID, err := requiredModelString(
		data,
		"productBlueprintId",
	)
	if err != nil {
		return nil, err
	}

	modelNumber, err := requiredModelString(
		data,
		"modelNumber",
	)
	if err != nil {
		return nil, err
	}

	createdAt, _ := asTime(data["createdAt"])
	updatedAt, _ := asTime(data["updatedAt"])

	switch kind {
	case string(modeldom.ModelVariationKindAlcohol):
		volume, err := modelVolume(
			data,
			"volume",
		)
		if err != nil {
			return nil, err
		}

		modelVariation := modeldom.AlcoholModelVariation{
			ID:                 document.Ref.ID,
			ProductBlueprintID: productBlueprintID,
			ModelNumber:        modelNumber,
			Volume:             volume,
			CreatedAt:          normalizeModelTime(createdAt),
			CreatedBy: modelStringPtr(
				data,
				"createdBy",
			),
			UpdatedAt: normalizeModelTime(updatedAt),
			UpdatedBy: modelStringPtr(
				data,
				"updatedBy",
			),
		}

		if err := modelVariation.Validate(); err != nil {
			return nil, err
		}

		return modelVariation, nil

	case string(modeldom.ModelVariationKindApparel):
		size, err := requiredModelString(
			data,
			"size",
		)
		if err != nil {
			return nil, err
		}

		color, err := modelColor(
			data,
			"color",
		)
		if err != nil {
			return nil, err
		}

		measurements, err := modelMeasurements(
			data,
			"measurements",
		)
		if err != nil {
			return nil, err
		}

		modelVariation := modeldom.ApparelModelVariation{
			ID:                 document.Ref.ID,
			ProductBlueprintID: productBlueprintID,
			ModelNumber:        modelNumber,
			Size:               size,
			Color:              color,
			Measurements:       measurements,
			CreatedAt:          normalizeModelTime(createdAt),
			CreatedBy: modelStringPtr(
				data,
				"createdBy",
			),
			UpdatedAt: normalizeModelTime(updatedAt),
			UpdatedBy: modelStringPtr(
				data,
				"updatedBy",
			),
		}

		if err := modelVariation.Validate(); err != nil {
			return nil, err
		}

		return modelVariation, nil

	default:
		return nil, modeldom.ErrInvalidKind
	}
}

func modelVariationToDoc(
	variation modeldom.ModelVariation,
) (map[string]any, error) {
	if variation == nil {
		return nil, modeldom.ErrInvalid
	}

	if err := variation.Validate(); err != nil {
		return nil, err
	}

	switch modelVariation := variation.(type) {
	case modeldom.AlcoholModelVariation:
		return alcoholModelVariationToDoc(
			modelVariation,
		), nil

	case modeldom.ApparelModelVariation:
		return apparelModelVariationToDoc(
			modelVariation,
		), nil

	default:
		return nil, modeldom.ErrInvalidKind
	}
}

func apparelModelVariationToDoc(
	variation modeldom.ApparelModelVariation,
) map[string]any {
	document := map[string]any{
		"kind": string(
			modeldom.ModelVariationKindApparel,
		),
		"productBlueprintId": variation.ProductBlueprintID,
		"modelNumber":        variation.ModelNumber,
		"size":               variation.Size,
		"color": map[string]any{
			"name": variation.Color.Name,
			"rgb":  variation.Color.RGB,
		},
	}

	if variation.Measurements != nil {
		document["measurements"] = variation.Measurements.Clone()
	}

	if !variation.CreatedAt.IsZero() {
		document["createdAt"] = variation.CreatedAt
	}

	if variation.CreatedBy != nil {
		document["createdBy"] = *variation.CreatedBy
	}

	if !variation.UpdatedAt.IsZero() {
		document["updatedAt"] = variation.UpdatedAt
	}

	if variation.UpdatedBy != nil {
		document["updatedBy"] = *variation.UpdatedBy
	}

	return document
}

func alcoholModelVariationToDoc(
	variation modeldom.AlcoholModelVariation,
) map[string]any {
	document := map[string]any{
		"kind": string(
			modeldom.ModelVariationKindAlcohol,
		),
		"productBlueprintId": variation.ProductBlueprintID,
		"modelNumber":        variation.ModelNumber,
		"volume": map[string]any{
			"value": variation.Volume.Value,
			"unit":  variation.Volume.Unit,
		},
	}

	if !variation.CreatedAt.IsZero() {
		document["createdAt"] = variation.CreatedAt
	}

	if variation.CreatedBy != nil {
		document["createdBy"] = *variation.CreatedBy
	}

	if !variation.UpdatedAt.IsZero() {
		document["updatedAt"] = variation.UpdatedAt
	}

	if variation.UpdatedBy != nil {
		document["updatedBy"] = *variation.UpdatedBy
	}

	return document
}

// ------------------------------------------------------------
// Firestore decoding
// ------------------------------------------------------------

func requiredModelString(
	data map[string]any,
	key string,
) (string, error) {
	value, ok := data[key].(string)
	if !ok || value == "" {
		return "", modeldom.ErrInvalid
	}

	return value, nil
}

func modelStringPtr(
	data map[string]any,
	key string,
) *string {
	value, ok := data[key].(string)
	if !ok || value == "" {
		return nil
	}

	return &value
}

func modelColor(
	data map[string]any,
	key string,
) (modeldom.Color, error) {
	raw, ok := data[key].(map[string]any)
	if !ok || raw == nil {
		return modeldom.Color{}, modeldom.ErrInvalidColor
	}

	name, ok := raw["name"].(string)
	if !ok || name == "" {
		return modeldom.Color{}, modeldom.ErrInvalidColor
	}

	rgb, ok := strictFirestoreInt(raw["rgb"])
	if !ok {
		return modeldom.Color{}, modeldom.ErrInvalidColor
	}

	color := modeldom.Color{
		Name: name,
		RGB:  rgb,
	}

	if err := color.Validate(); err != nil {
		return modeldom.Color{}, err
	}

	return color, nil
}

func modelVolume(
	data map[string]any,
	key string,
) (modeldom.Volume, error) {
	raw, ok := data[key].(map[string]any)
	if !ok || raw == nil {
		return modeldom.Volume{}, modeldom.ErrInvalidVolume
	}

	value, ok := strictFirestoreInt(raw["value"])
	if !ok {
		return modeldom.Volume{}, modeldom.ErrInvalidVolume
	}

	unit, ok := raw["unit"].(string)
	if !ok || unit == "" {
		return modeldom.Volume{}, modeldom.ErrInvalidVolumeUnit
	}

	volume := modeldom.Volume{
		Value: value,
		Unit:  unit,
	}

	if err := volume.Validate(); err != nil {
		return modeldom.Volume{}, err
	}

	return volume, nil
}

func modelMeasurements(
	data map[string]any,
	key string,
) (modeldom.Measurements, error) {
	value, exists := data[key]
	if !exists || value == nil {
		return nil, nil
	}

	raw, ok := value.(map[string]any)
	if !ok {
		return nil, modeldom.ErrInvalidMeasurements
	}

	measurements := make(
		modeldom.Measurements,
		len(raw),
	)

	for measurementKey, rawValue := range raw {
		if measurementKey == "" {
			return nil, modeldom.ErrInvalidMeasurements
		}

		measurementValue, ok := strictFirestoreInt(rawValue)
		if !ok || measurementValue < 0 {
			return nil, modeldom.ErrInvalidMeasurements
		}

		measurements[measurementKey] = measurementValue
	}

	if err := measurements.Validate(); err != nil {
		return nil, err
	}

	return measurements, nil
}

func strictFirestoreInt(
	value any,
) (int, bool) {
	switch number := value.(type) {
	case int:
		return number, true

	case int64:
		converted := int(number)
		if int64(converted) != number {
			return 0, false
		}

		return converted, true

	default:
		return 0, false
	}
}

// ------------------------------------------------------------
// Sorting
// ------------------------------------------------------------

func sortModelVariations(
	variations []modeldom.ModelVariation,
) {
	sort.Slice(
		variations,
		func(i, j int) bool {
			firstUpdatedAt,
				firstCreatedAt,
				firstID := modelVariationSortValues(
				variations[i],
			)

			secondUpdatedAt,
				secondCreatedAt,
				secondID := modelVariationSortValues(
				variations[j],
			)

			if !firstUpdatedAt.Equal(secondUpdatedAt) {
				return firstUpdatedAt.After(secondUpdatedAt)
			}

			if !firstCreatedAt.Equal(secondCreatedAt) {
				return firstCreatedAt.After(secondCreatedAt)
			}

			return firstID < secondID
		},
	)
}

func modelVariationSortValues(
	variation modeldom.ModelVariation,
) (
	time.Time,
	time.Time,
	string,
) {
	switch modelVariation := variation.(type) {
	case modeldom.ApparelModelVariation:
		return modelVariation.UpdatedAt,
			modelVariation.CreatedAt,
			modelVariation.ID

	case modeldom.AlcoholModelVariation:
		return modelVariation.UpdatedAt,
			modelVariation.CreatedAt,
			modelVariation.ID

	default:
		if variation == nil {
			return time.Time{},
				time.Time{},
				""
		}

		return time.Time{},
			time.Time{},
			variation.GetID()
	}
}

func normalizeModelTime(
	value time.Time,
) time.Time {
	if value.IsZero() {
		return time.Time{}
	}

	return value.UTC()
}
