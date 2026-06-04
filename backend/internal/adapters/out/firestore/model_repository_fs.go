// backend/internal/adapters/out/firestore/model_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	modeldom "narratives/internal/domain/model"
)

// ------------------------------------------------------------
// Repository struct
// ------------------------------------------------------------

type ModelRepositoryFS struct {
	Client *firestore.Client
}

func NewModelRepositoryFS(client *firestore.Client) *ModelRepositoryFS {
	return &ModelRepositoryFS{Client: client}
}

func (r *ModelRepositoryFS) variationsCol() *firestore.CollectionRef {
	return r.Client.Collection("models")
}

// ------------------------------------------------------------
// Variation CRUD（ライブの models コレクション）
// ------------------------------------------------------------

func (r *ModelRepositoryFS) GetModelVariationByID(ctx context.Context, variationID string) (modeldom.ModelVariation, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	if variationID == "" {
		return nil, modeldom.ErrNotFound
	}

	snap, err := r.variationsCol().Doc(variationID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, modeldom.ErrNotFound
		}
		return nil, err
	}

	return docToModelVariation(snap)
}

func (r *ModelRepositoryFS) CreateModelVariation(ctx context.Context, variation modeldom.NewModelVariation) (modeldom.ModelVariation, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	if err := variation.Validate(); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	docRef := r.variationsCol().NewDoc()

	mv, err := newModelVariationToDomain(docRef.ID, variation, now)
	if err != nil {
		return nil, err
	}

	doc, err := modelVariationToDoc(mv)
	if err != nil {
		return nil, err
	}

	if _, err := docRef.Create(ctx, doc); err != nil {
		return nil, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return nil, err
	}

	return docToModelVariation(snap)
}

func (r *ModelRepositoryFS) UpdateModelVariation(ctx context.Context, variationID string, updates modeldom.ModelVariationUpdate) (modeldom.ModelVariation, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	if variationID == "" {
		return nil, modeldom.ErrNotFound
	}

	docRef := r.variationsCol().Doc(variationID)

	fsUpdates := make([]firestore.Update, 0, 6)

	if updates.Size != nil {
		fsUpdates = append(fsUpdates, firestore.Update{Path: "size", Value: *updates.Size})
	}
	if updates.Color != nil {
		fsUpdates = append(fsUpdates, firestore.Update{
			Path: "color",
			Value: map[string]any{
				"name": updates.Color.Name,
				"rgb":  updates.Color.RGB,
			},
		})
	}
	if updates.ModelNumber != nil {
		fsUpdates = append(fsUpdates, firestore.Update{Path: "modelNumber", Value: *updates.ModelNumber})
	}
	if updates.Measurements != nil {
		fsUpdates = append(fsUpdates, firestore.Update{Path: "measurements", Value: updates.Measurements})
	}
	if updates.Volume != nil {
		fsUpdates = append(fsUpdates, firestore.Update{
			Path: "volume",
			Value: map[string]any{
				"value": updates.Volume.Value,
				"unit":  updates.Volume.Unit,
			},
		})
	}

	fsUpdates = append(fsUpdates, firestore.Update{
		Path:  "updatedAt",
		Value: time.Now().UTC(),
	})

	if _, err := docRef.Update(ctx, fsUpdates); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, modeldom.ErrNotFound
		}
		return nil, err
	}

	return r.GetModelVariationByID(ctx, variationID)
}

func (r *ModelRepositoryFS) DeleteModelVariation(ctx context.Context, variationID string) (modeldom.ModelVariation, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	if variationID == "" {
		return nil, modeldom.ErrNotFound
	}

	docRef := r.variationsCol().Doc(variationID)

	snap, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, modeldom.ErrNotFound
		}
		return nil, err
	}

	v, err := docToModelVariation(snap)
	if err != nil {
		return nil, err
	}

	if _, err := docRef.Delete(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, modeldom.ErrNotFound
		}
		return nil, err
	}

	return v, nil
}

// ------------------------------------------------------------
// RepositoryPort implementation
// ------------------------------------------------------------

func (r *ModelRepositoryFS) ListVariations(
	ctx context.Context,
	filter modeldom.VariationFilter,
	page modeldom.Page,
) (modeldom.VariationPageResult, error) {
	if r.Client == nil {
		return modeldom.VariationPageResult{}, errors.New("firestore client is nil")
	}

	pbID := filter.ProductBlueprintID
	if pbID == "" {
		return modeldom.VariationPageResult{}, modeldom.ErrInvalidBlueprintID
	}

	all, err := r.listVariationsByProductBlueprintID(ctx, pbID)
	if err != nil {
		return modeldom.VariationPageResult{}, err
	}

	inSet := func(s string, xs []string) bool {
		if len(xs) == 0 {
			return true
		}
		for _, x := range xs {
			if x == s {
				return true
			}
		}
		return false
	}

	volumeInSet := func(v modeldom.Volume, xs []modeldom.Volume) bool {
		if len(xs) == 0 {
			return true
		}
		for _, x := range xs {
			if v.Value == x.Value && v.Unit == x.Unit {
				return true
			}
		}
		return false
	}

	q := strings.ToLower(filter.SearchQuery)

	filtered := make([]modeldom.ModelVariation, 0, len(all))
	for _, raw := range all {
		if raw == nil {
			continue
		}

		if apparel, ok := toApparelModelVariation(raw); ok {
			if !inSet(apparel.Size, filter.Sizes) {
				continue
			}
			if !inSet(apparel.Color.Name, filter.Colors) {
				continue
			}
			if !inSet(apparel.ModelNumber, filter.ModelNumbers) {
				continue
			}

			if filter.CreatedFrom != nil && !apparel.CreatedAt.IsZero() && apparel.CreatedAt.Before(filter.CreatedFrom.UTC()) {
				continue
			}
			if filter.CreatedTo != nil && !apparel.CreatedAt.IsZero() && apparel.CreatedAt.After(filter.CreatedTo.UTC()) {
				continue
			}
			if filter.UpdatedFrom != nil && !apparel.UpdatedAt.IsZero() && apparel.UpdatedAt.Before(filter.UpdatedFrom.UTC()) {
				continue
			}
			if filter.UpdatedTo != nil && !apparel.UpdatedAt.IsZero() && apparel.UpdatedAt.After(filter.UpdatedTo.UTC()) {
				continue
			}

			if q != "" {
				hay := strings.ToLower(apparel.ModelNumber + " " + apparel.Size + " " + apparel.Color.Name)
				if !strings.Contains(hay, q) {
					continue
				}
			}

			filtered = append(filtered, apparel)
			continue
		}

		if alcohol, ok := toAlcoholModelVariation(raw); ok {
			if len(filter.Sizes) > 0 || len(filter.Colors) > 0 {
				continue
			}
			if !inSet(alcohol.ModelNumber, filter.ModelNumbers) {
				continue
			}
			if !volumeInSet(alcohol.Volume, filter.Volumes) {
				continue
			}

			if filter.CreatedFrom != nil && !alcohol.CreatedAt.IsZero() && alcohol.CreatedAt.Before(filter.CreatedFrom.UTC()) {
				continue
			}
			if filter.CreatedTo != nil && !alcohol.CreatedAt.IsZero() && alcohol.CreatedAt.After(filter.CreatedTo.UTC()) {
				continue
			}
			if filter.UpdatedFrom != nil && !alcohol.UpdatedAt.IsZero() && alcohol.UpdatedAt.Before(filter.UpdatedFrom.UTC()) {
				continue
			}
			if filter.UpdatedTo != nil && !alcohol.UpdatedAt.IsZero() && alcohol.UpdatedAt.After(filter.UpdatedTo.UTC()) {
				continue
			}

			if q != "" {
				volumeText := fmt.Sprintf("%d%s", alcohol.Volume.Value, alcohol.Volume.Unit)
				hay := strings.ToLower(alcohol.ModelNumber + " " + volumeText)
				if !strings.Contains(hay, q) {
					continue
				}
			}

			filtered = append(filtered, alcohol)
			continue
		}

		return modeldom.VariationPageResult{}, modeldom.ErrInvalid
	}

	sort.Slice(filtered, func(i, j int) bool {
		aUpdatedAt, aCreatedAt, aID := modelVariationSortValues(filtered[i])
		bUpdatedAt, bCreatedAt, bID := modelVariationSortValues(filtered[j])

		if !aUpdatedAt.Equal(bUpdatedAt) {
			return aUpdatedAt.After(bUpdatedAt)
		}
		if !aCreatedAt.Equal(bCreatedAt) {
			return aCreatedAt.After(bCreatedAt)
		}
		return aID < bID
	})

	per := page.PerPage
	if per <= 0 {
		per = 50
	}
	num := page.Number
	if num <= 0 {
		num = 1
	}

	total := len(filtered)
	totalPages := (total + per - 1) / per
	start := (num - 1) * per
	if start > total {
		start = total
	}
	end := start + per
	if end > total {
		end = total
	}

	items := make([]modeldom.ModelVariation, 0, end-start)
	if start < end {
		items = append(items, filtered[start:end]...)
	}

	return modeldom.VariationPageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       num,
		PerPage:    per,
	}, nil
}

func (r *ModelRepositoryFS) GetModelVariations(ctx context.Context, productBlueprintID string) ([]modeldom.ModelVariation, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	if productBlueprintID == "" {
		return nil, modeldom.ErrInvalidBlueprintID
	}

	return r.listVariationsByProductBlueprintID(ctx, productBlueprintID)
}

func (r *ModelRepositoryFS) GetSizeVariations(ctx context.Context, productBlueprintID string) ([]modeldom.SizeVariation, error) {
	vars, err := r.GetModelVariations(ctx, productBlueprintID)
	if err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	out := make([]modeldom.SizeVariation, 0)

	for _, raw := range vars {
		v, ok := toApparelModelVariation(raw)
		if !ok {
			continue
		}

		if v.Size == "" {
			continue
		}
		if _, ok := seen[v.Size]; ok {
			continue
		}
		seen[v.Size] = struct{}{}

		out = append(out, modeldom.SizeVariation{
			ID:           v.ID,
			Size:         v.Size,
			Measurements: cloneMeasurements(v.Measurements),
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Size < out[j].Size
	})

	return out, nil
}

func (r *ModelRepositoryFS) GetModelNumbers(ctx context.Context, productBlueprintID string) ([]modeldom.ModelNumber, error) {
	vars, err := r.GetModelVariations(ctx, productBlueprintID)
	if err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	out := make([]modeldom.ModelNumber, 0)

	for _, raw := range vars {
		if raw == nil {
			continue
		}

		modelNumber := raw.GetModelNumber()
		if modelNumber == "" {
			continue
		}
		if _, ok := seen[modelNumber]; ok {
			continue
		}
		seen[modelNumber] = struct{}{}

		if apparel, ok := toApparelModelVariation(raw); ok {
			out = append(out, modeldom.ModelNumber{
				Size:        apparel.Size,
				Color:       apparel.Color.Name,
				ModelNumber: apparel.ModelNumber,
			})
			continue
		}

		if alcohol, ok := toAlcoholModelVariation(raw); ok {
			out = append(out, modeldom.ModelNumber{
				ModelNumber: alcohol.ModelNumber,
			})
			continue
		}

		return nil, modeldom.ErrInvalid
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].ModelNumber < out[j].ModelNumber
	})

	return out, nil
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

func (r *ModelRepositoryFS) listVariationsByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]modeldom.ModelVariation, error) {
	q := r.variationsCol().
		Where("productBlueprintId", "==", productBlueprintID)

	it := q.Documents(ctx)
	defer it.Stop()

	out := make([]modeldom.ModelVariation, 0)
	for {
		doc, err := it.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, err
		}

		v, err := docToModelVariation(doc)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}

	return out, nil
}

func newModelVariationToDomain(id string, input modeldom.NewModelVariation, now time.Time) (modeldom.ModelVariation, error) {
	switch input.Kind {
	case modeldom.ModelVariationKindAlcohol:
		if input.Alcohol == nil {
			return nil, modeldom.ErrInvalid
		}

		mv := modeldom.AlcoholModelVariation{
			ID:                 id,
			ProductBlueprintID: input.Alcohol.ProductBlueprintID,
			ModelNumber:        input.Alcohol.ModelNumber,
			Volume:             input.Alcohol.Volume,
			CreatedAt:          now,
			UpdatedAt:          now,
		}

		if err := mv.Validate(); err != nil {
			return nil, err
		}

		return mv, nil

	case modeldom.ModelVariationKindApparel:
		if input.Apparel == nil {
			return nil, modeldom.ErrInvalid
		}

		mv := modeldom.ApparelModelVariation{
			ID:                 id,
			ProductBlueprintID: input.Apparel.ProductBlueprintID,
			ModelNumber:        input.Apparel.ModelNumber,
			Size:               input.Apparel.Size,
			Color: modeldom.Color{
				Name: input.Apparel.Color.Name,
				RGB:  input.Apparel.Color.RGB,
			},
			Measurements: cloneMeasurements(input.Apparel.Measurements),
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if err := mv.Validate(); err != nil {
			return nil, err
		}

		return mv, nil

	default:
		return nil, modeldom.ErrInvalid
	}
}

func docToModelVariation(doc *firestore.DocumentSnapshot) (modeldom.ModelVariation, error) {
	data := doc.Data()
	if data == nil {
		return nil, fmt.Errorf("empty variation: %s", doc.Ref.ID)
	}

	kind, err := requiredModelString(data, "kind")
	if err != nil {
		return nil, err
	}

	switch kind {
	case string(modeldom.ModelVariationKindAlcohol):
		mv := modeldom.AlcoholModelVariation{
			ID:                 doc.Ref.ID,
			ProductBlueprintID: modelString(data, "productBlueprintId"),
			ModelNumber:        modelString(data, "modelNumber"),
			Volume:             modelVolume(data, "volume"),
			CreatedAt:          modelTime(data, "createdAt"),
			CreatedBy:          modelStringPtr(data, "createdBy"),
			UpdatedAt:          modelTime(data, "updatedAt"),
			UpdatedBy:          modelStringPtr(data, "updatedBy"),
		}

		if err := mv.Validate(); err != nil {
			return nil, err
		}

		return mv, nil

	case string(modeldom.ModelVariationKindApparel):
		mv := modeldom.ApparelModelVariation{
			ID:                 doc.Ref.ID,
			ProductBlueprintID: modelString(data, "productBlueprintId"),
			ModelNumber:        modelString(data, "modelNumber"),
			Size:               modelString(data, "size"),
			Color:              modelColor(data, "color"),
			Measurements:       modelMeasurements(data, "measurements"),
			CreatedAt:          modelTime(data, "createdAt"),
			CreatedBy:          modelStringPtr(data, "createdBy"),
			UpdatedAt:          modelTime(data, "updatedAt"),
			UpdatedBy:          modelStringPtr(data, "updatedBy"),
		}

		if err := mv.Validate(); err != nil {
			return nil, err
		}

		return mv, nil

	default:
		return nil, modeldom.ErrInvalid
	}
}

func modelVariationToDoc(v modeldom.ModelVariation) (map[string]any, error) {
	switch mv := v.(type) {
	case modeldom.AlcoholModelVariation:
		return alcoholModelVariationToDoc(mv), nil

	case modeldom.ApparelModelVariation:
		return apparelModelVariationToDoc(mv), nil

	default:
		return nil, modeldom.ErrInvalid
	}
}

func apparelModelVariationToDoc(v modeldom.ApparelModelVariation) map[string]any {
	m := map[string]any{
		"kind":               string(modeldom.ModelVariationKindApparel),
		"productBlueprintId": v.ProductBlueprintID,
		"modelNumber":        v.ModelNumber,
		"size":               v.Size,
		"color": map[string]any{
			"name": v.Color.Name,
			"rgb":  v.Color.RGB,
		},
	}

	if v.Measurements != nil {
		m["measurements"] = cloneMeasurements(v.Measurements)
	}
	if !v.CreatedAt.IsZero() {
		m["createdAt"] = v.CreatedAt
	}
	if v.CreatedBy != nil {
		m["createdBy"] = *v.CreatedBy
	}
	if !v.UpdatedAt.IsZero() {
		m["updatedAt"] = v.UpdatedAt
	}
	if v.UpdatedBy != nil {
		m["updatedBy"] = *v.UpdatedBy
	}

	return m
}

func alcoholModelVariationToDoc(v modeldom.AlcoholModelVariation) map[string]any {
	m := map[string]any{
		"kind":               string(modeldom.ModelVariationKindAlcohol),
		"productBlueprintId": v.ProductBlueprintID,
		"modelNumber":        v.ModelNumber,
		"volume": map[string]any{
			"value": v.Volume.Value,
			"unit":  v.Volume.Unit,
		},
	}

	if !v.CreatedAt.IsZero() {
		m["createdAt"] = v.CreatedAt
	}
	if v.CreatedBy != nil {
		m["createdBy"] = *v.CreatedBy
	}
	if !v.UpdatedAt.IsZero() {
		m["updatedAt"] = v.UpdatedAt
	}
	if v.UpdatedBy != nil {
		m["updatedBy"] = *v.UpdatedBy
	}

	return m
}

func requiredModelString(data map[string]any, key string) (string, error) {
	v, ok := data[key].(string)
	if !ok || v == "" {
		return "", modeldom.ErrInvalid
	}
	return v, nil
}

func modelString(data map[string]any, key string) string {
	v, _ := data[key].(string)
	return v
}

func modelStringPtr(data map[string]any, key string) *string {
	v, _ := data[key].(string)
	if v == "" {
		return nil
	}
	return &v
}

func modelTime(data map[string]any, key string) time.Time {
	v, _ := data[key].(time.Time)
	if v.IsZero() {
		return time.Time{}
	}
	return v.UTC()
}

func modelColor(data map[string]any, key string) modeldom.Color {
	raw, _ := data[key].(map[string]any)

	name, _ := raw["name"].(string)
	rgb, _ := raw["rgb"].(int64)

	return modeldom.Color{
		Name: name,
		RGB:  int(rgb),
	}
}

func modelVolume(data map[string]any, key string) modeldom.Volume {
	raw, _ := data[key].(map[string]any)

	value, _ := raw["value"].(int64)
	unit, _ := raw["unit"].(string)

	return modeldom.Volume{
		Value: int(value),
		Unit:  unit,
	}
}

func modelMeasurements(data map[string]any, key string) modeldom.Measurements {
	raw, ok := data[key].(map[string]any)
	if !ok || raw == nil {
		return nil
	}

	out := make(modeldom.Measurements, len(raw))
	for k, v := range raw {
		n, ok := v.(int64)
		if !ok {
			return nil
		}
		out[k] = int(n)
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

func toApparelModelVariation(v modeldom.ModelVariation) (modeldom.ApparelModelVariation, bool) {
	if v == nil {
		return modeldom.ApparelModelVariation{}, false
	}

	x, ok := v.(modeldom.ApparelModelVariation)
	return x, ok
}

func toAlcoholModelVariation(v modeldom.ModelVariation) (modeldom.AlcoholModelVariation, bool) {
	if v == nil {
		return modeldom.AlcoholModelVariation{}, false
	}

	x, ok := v.(modeldom.AlcoholModelVariation)
	return x, ok
}

func modelVariationSortValues(v modeldom.ModelVariation) (time.Time, time.Time, string) {
	if apparel, ok := toApparelModelVariation(v); ok {
		return apparel.UpdatedAt, apparel.CreatedAt, apparel.ID
	}
	if alcohol, ok := toAlcoholModelVariation(v); ok {
		return alcohol.UpdatedAt, alcohol.CreatedAt, alcohol.ID
	}
	if v == nil {
		return time.Time{}, time.Time{}, ""
	}
	return time.Time{}, time.Time{}, v.GetID()
}

func cloneMeasurements(in modeldom.Measurements) modeldom.Measurements {
	if in == nil {
		return nil
	}

	out := make(modeldom.Measurements, len(in))
	for k, v := range in {
		out[k] = v
	}

	return out
}
