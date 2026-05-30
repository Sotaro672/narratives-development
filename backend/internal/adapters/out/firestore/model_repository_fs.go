// backend/internal/adapters/out/firestore/model_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
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

	if _, err := docRef.Create(ctx, modelVariationToDoc(mv)); err != nil {
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

	log.Printf("[ModelRepositoryFS] UpdateModelVariation id=%s path=models/%s", variationID, variationID)

	docRef := r.variationsCol().Doc(variationID)
	var fsUpdates []firestore.Update

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

	if len(fsUpdates) == 0 {
		return r.GetModelVariationByID(ctx, variationID)
	}

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
// ReplaceModelVariations（大量更新、ライブ）
// ------------------------------------------------------------

func (r *ModelRepositoryFS) ReplaceModelVariations(ctx context.Context, vars []modeldom.NewModelVariation) ([]modeldom.ModelVariation, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	if len(vars) == 0 {
		return []modeldom.ModelVariation{}, nil
	}

	productBlueprintID := vars[0].ProductBlueprintID()
	if productBlueprintID == "" {
		return nil, modeldom.ErrInvalidBlueprintID
	}

	for _, v := range vars {
		if err := v.Validate(); err != nil {
			return nil, err
		}
		if v.ProductBlueprintID() != productBlueprintID {
			return nil, modeldom.ErrProductMismatch
		}
	}

	const chunkSize = 400

	existing, err := r.listVariationsByProductBlueprintID(ctx, productBlueprintID)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(existing); i += chunkSize {
		end := i + chunkSize
		if end > len(existing) {
			end = len(existing)
		}
		chunk := existing[i:end]

		err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			for _, v := range chunk {
				id := v.GetID()
				if id == "" {
					continue
				}
				ref := r.variationsCol().Doc(id)
				if err := tx.Delete(ref); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	for i := 0; i < len(vars); i += chunkSize {
		end := i + chunkSize
		if end > len(vars) {
			end = len(vars)
		}
		chunk := vars[i:end]
		now := time.Now().UTC()

		err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			for _, nv := range chunk {
				docRef := r.variationsCol().NewDoc()

				mv, err := newModelVariationToDomain(docRef.ID, nv, now)
				if err != nil {
					return err
				}

				if err := tx.Set(docRef, modelVariationToDoc(mv)); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return r.listVariationsByProductBlueprintID(ctx, productBlueprintID)
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
			if strings.EqualFold(x, s) {
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
			if v.Value == x.Value && strings.EqualFold(v.Unit, x.Unit) {
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

			if filter.Deleted != nil && *filter.Deleted {
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

			if filter.Deleted != nil && *filter.Deleted {
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
		}
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
	var sizes []string

	for _, raw := range vars {
		v, ok := toApparelModelVariation(raw)
		if !ok {
			continue
		}

		s := v.Size
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		sizes = append(sizes, s)
	}

	sort.Strings(sizes)

	makeSizeVariation := func(size string) modeldom.SizeVariation {
		var out modeldom.SizeVariation
		rv := reflect.ValueOf(&out).Elem()
		switch rv.Kind() {
		case reflect.String:
			rv.SetString(size)
		case reflect.Struct:
			for _, fn := range []string{"Size", "Name", "Value"} {
				f := rv.FieldByName(fn)
				if f.IsValid() && f.CanSet() && f.Kind() == reflect.String {
					f.SetString(size)
					break
				}
			}
		}
		return out
	}

	res := make([]modeldom.SizeVariation, 0, len(sizes))
	for _, s := range sizes {
		res = append(res, makeSizeVariation(s))
	}

	return res, nil
}

func (r *ModelRepositoryFS) GetModelNumbers(ctx context.Context, productBlueprintID string) ([]modeldom.ModelNumber, error) {
	vars, err := r.GetModelVariations(ctx, productBlueprintID)
	if err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	var nums []string

	for _, raw := range vars {
		if raw == nil {
			continue
		}

		s := raw.GetModelNumber()
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		nums = append(nums, s)
	}

	sort.Strings(nums)

	makeModelNumber := func(mn string) modeldom.ModelNumber {
		var out modeldom.ModelNumber
		rv := reflect.ValueOf(&out).Elem()
		switch rv.Kind() {
		case reflect.String:
			rv.SetString(mn)
		case reflect.Struct:
			for _, fn := range []string{"ModelNumber", "Number", "Name", "Value"} {
				f := rv.FieldByName(fn)
				if f.IsValid() && f.CanSet() && f.Kind() == reflect.String {
					f.SetString(mn)
					break
				}
			}
		}
		return out
	}

	res := make([]modeldom.ModelNumber, 0, len(nums))
	for _, s := range nums {
		res = append(res, makeModelNumber(s))
	}

	return res, nil
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
			Measurements: input.Apparel.Measurements,
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

	getStr := func(k string) string {
		if v, ok := data[k].(string); ok {
			return v
		}
		return ""
	}

	getTime := func(k string) time.Time {
		if v, ok := data[k].(time.Time); ok {
			return v.UTC()
		}
		return time.Time{}
	}

	getStringPtr := func(k string) *string {
		if v, ok := data[k].(string); ok && v != "" {
			s := v
			return &s
		}
		return nil
	}

	kind := getStr("kind")
	if kind == "" {
		kind = string(modeldom.ModelVariationKindApparel)
	}

	if kind == string(modeldom.ModelVariationKindAlcohol) {
		return modeldom.AlcoholModelVariation{
			ID:                 doc.Ref.ID,
			ProductBlueprintID: getStr("productBlueprintId"),
			ModelNumber:        getStr("modelNumber"),
			Volume:             getVolumeFromDoc(data),
			CreatedAt:          getTime("createdAt"),
			CreatedBy:          getStringPtr("createdBy"),
			UpdatedAt:          getTime("updatedAt"),
			UpdatedBy:          getStringPtr("updatedBy"),
		}, nil
	}

	return modeldom.ApparelModelVariation{
		ID:                 doc.Ref.ID,
		ProductBlueprintID: getStr("productBlueprintId"),
		ModelNumber:        getStr("modelNumber"),
		Size:               getStr("size"),
		Color:              getColorFromDoc(data),
		Measurements:       getMeasurementsFromDoc(data),
		CreatedAt:          getTime("createdAt"),
		CreatedBy:          getStringPtr("createdBy"),
		UpdatedAt:          getTime("updatedAt"),
		UpdatedBy:          getStringPtr("updatedBy"),
	}, nil
}

func modelVariationToDoc(v modeldom.ModelVariation) map[string]any {
	switch mv := v.(type) {
	case modeldom.AlcoholModelVariation:
		return alcoholModelVariationToDoc(mv)
	case *modeldom.AlcoholModelVariation:
		if mv == nil {
			return map[string]any{}
		}
		return alcoholModelVariationToDoc(*mv)
	case modeldom.ApparelModelVariation:
		return apparelModelVariationToDoc(mv)
	case *modeldom.ApparelModelVariation:
		if mv == nil {
			return map[string]any{}
		}
		return apparelModelVariationToDoc(*mv)
	default:
		return map[string]any{
			"productBlueprintId": v.GetProductBlueprintID(),
			"modelNumber":        v.GetModelNumber(),
		}
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
		m["measurements"] = v.Measurements
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

func getColorFromDoc(data map[string]any) modeldom.Color {
	var color modeldom.Color
	raw, ok := data["color"]
	if !ok || raw == nil {
		return color
	}

	if v, ok := raw.(map[string]any); ok {
		if n, ok2 := v["name"].(string); ok2 {
			color.Name = n
		}
		switch rv := v["rgb"].(type) {
		case int64:
			color.RGB = int(rv)
		case int:
			color.RGB = rv
		case float64:
			color.RGB = int(rv)
		}
	}

	return color
}

func getVolumeFromDoc(data map[string]any) modeldom.Volume {
	var volume modeldom.Volume
	raw, ok := data["volume"]
	if !ok || raw == nil {
		return volume
	}

	if v, ok := raw.(map[string]any); ok {
		switch rv := v["value"].(type) {
		case int64:
			volume.Value = int(rv)
		case int:
			volume.Value = rv
		case float64:
			volume.Value = int(rv)
		}
		if unit, ok := v["unit"].(string); ok {
			volume.Unit = unit
		}
	}

	return volume
}

func getMeasurementsFromDoc(data map[string]any) modeldom.Measurements {
	raw, ok := data["measurements"]
	if !ok || raw == nil {
		return nil
	}

	out := make(modeldom.Measurements)

	switch vv := raw.(type) {
	case map[string]any:
		for k, v := range vv {
			switch n := v.(type) {
			case int64:
				out[k] = int(n)
			case int:
				out[k] = n
			case float64:
				out[k] = int(n)
			}
		}
	case map[string]int:
		for k, v := range vv {
			out[k] = v
		}
	case map[string]int64:
		for k, v := range vv {
			out[k] = int(v)
		}
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

	switch x := v.(type) {
	case modeldom.ApparelModelVariation:
		return x, true
	case *modeldom.ApparelModelVariation:
		if x == nil {
			return modeldom.ApparelModelVariation{}, false
		}
		return *x, true
	default:
		return modeldom.ApparelModelVariation{}, false
	}
}

func toAlcoholModelVariation(v modeldom.ModelVariation) (modeldom.AlcoholModelVariation, bool) {
	if v == nil {
		return modeldom.AlcoholModelVariation{}, false
	}

	switch x := v.(type) {
	case modeldom.AlcoholModelVariation:
		return x, true
	case *modeldom.AlcoholModelVariation:
		if x == nil {
			return modeldom.AlcoholModelVariation{}, false
		}
		return *x, true
	default:
		return modeldom.AlcoholModelVariation{}, false
	}
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

// ListModelIDsByProductBlueprintID returns model variation IDs for a product blueprint.
// This is used by ListCreateQuery to build PriceRows independent of inventory stock.
func (r *ModelRepositoryFS) ListModelIDsByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]string, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	pbID := productBlueprintID
	if pbID == "" {
		return nil, modeldom.ErrInvalidBlueprintID
	}

	vars, err := r.listVariationsByProductBlueprintID(ctx, pbID)
	if err != nil {
		return nil, err
	}
	if len(vars) == 0 {
		return []string{}, nil
	}

	seen := map[string]struct{}{}
	out := make([]string, 0, len(vars))

	for _, v := range vars {
		if v == nil {
			continue
		}

		id := v.GetID()
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}

		seen[id] = struct{}{}
		out = append(out, id)
	}

	sort.Strings(out)

	return out, nil
}
