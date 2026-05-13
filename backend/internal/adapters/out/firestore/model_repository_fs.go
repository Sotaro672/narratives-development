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

func (r *ModelRepositoryFS) modelSetsCol() *firestore.CollectionRef {
	return r.Client.Collection("model_sets")
}

func (r *ModelRepositoryFS) variationsCol() *firestore.CollectionRef {
	return r.Client.Collection("models")
}

// ------------------------------------------------------------
// model_sets 取得（ライブ）
// ------------------------------------------------------------

func (r *ModelRepositoryFS) GetModelData(ctx context.Context, productBlueprintID string) (*modeldom.ModelData, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	if productBlueprintID == "" {
		return nil, modeldom.ErrNotFound
	}

	snap, err := r.modelSetsCol().Doc(productBlueprintID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, modeldom.ErrNotFound
		}
		return nil, err
	}

	data := snap.Data()
	if data == nil {
		return nil, fmt.Errorf("empty model_set document: %s", snap.Ref.ID)
	}

	var updatedAt time.Time
	if v, ok := data["updatedAt"].(time.Time); ok {
		updatedAt = v.UTC()
	}

	vars, err := r.listVariationsByProductBlueprintID(ctx, productBlueprintID)
	if err != nil {
		return nil, err
	}

	return &modeldom.ModelData{
		ProductBlueprintID: productBlueprintID,
		Variations:         vars,
		UpdatedAt:          updatedAt,
	}, nil
}

func (r *ModelRepositoryFS) GetModelDataByBlueprintID(ctx context.Context, productBlueprintID string) (*modeldom.ModelData, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	if productBlueprintID == "" {
		return nil, modeldom.ErrNotFound
	}

	q := r.modelSetsCol().Where("productBlueprintId", "==", productBlueprintID).Limit(1)
	it := q.Documents(ctx)
	defer it.Stop()

	snap, err := it.Next()
	if err != nil {
		if err == iterator.Done {
			return nil, modeldom.ErrNotFound
		}
		return nil, err
	}

	data := snap.Data()
	if data == nil {
		return nil, fmt.Errorf("empty model_set: %s", snap.Ref.ID)
	}

	var updatedAt time.Time
	if v, ok := data["updatedAt"].(time.Time); ok {
		updatedAt = v.UTC()
	}

	vars, err := r.listVariationsByProductBlueprintID(ctx, productBlueprintID)
	if err != nil {
		return nil, err
	}

	return &modeldom.ModelData{
		ProductBlueprintID: productBlueprintID,
		Variations:         vars,
		UpdatedAt:          updatedAt,
	}, nil
}

// ------------------------------------------------------------
// model_sets 更新（ライブ）
// ------------------------------------------------------------

func (r *ModelRepositoryFS) UpdateModelData(ctx context.Context, productBlueprintID string, updates modeldom.ModelDataUpdate) (*modeldom.ModelData, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	if productBlueprintID == "" {
		return nil, modeldom.ErrNotFound
	}

	docRef := r.modelSetsCol().Doc(productBlueprintID)
	var fsUpdates []firestore.Update

	if v, ok := updates["productBlueprintID"]; ok {
		if s, ok2 := v.(string); ok2 {
			fsUpdates = append(fsUpdates, firestore.Update{
				Path:  "productBlueprintId",
				Value: s,
			})
		}
	}

	fsUpdates = append(fsUpdates, firestore.Update{
		Path:  "updatedAt",
		Value: time.Now().UTC(),
	})

	if len(fsUpdates) == 0 {
		return r.GetModelData(ctx, productBlueprintID)
	}

	if _, err := docRef.Update(ctx, fsUpdates); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, modeldom.ErrNotFound
		}
		return nil, err
	}

	return r.GetModelData(ctx, productBlueprintID)
}

// ------------------------------------------------------------
// Variation CRUD（ライブの models コレクション）
// ------------------------------------------------------------

func (r *ModelRepositoryFS) GetModelVariationByID(ctx context.Context, variationID string) (*modeldom.ModelVariation, error) {
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

	v, err := docToModelVariation(snap)
	if err != nil {
		return nil, err
	}

	return &v, nil
}

func (r *ModelRepositoryFS) CreateModelVariation(ctx context.Context, variation modeldom.NewApparelModelVariation) (*modeldom.ModelVariation, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	now := time.Now().UTC()
	docRef := r.variationsCol().NewDoc()

	mv := modeldom.ApparelModelVariation{
		ID:                 docRef.ID,
		ProductBlueprintID: variation.ProductBlueprintID,
		ModelNumber:        variation.ModelNumber,
		Size:               variation.Size,
		Color: modeldom.Color{
			Name: variation.Color.Name,
			RGB:  variation.Color.RGB,
		},
		Measurements: variation.Measurements,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if _, err := docRef.Create(ctx, modelVariationToDoc(mv)); err != nil {
		return nil, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return nil, err
	}

	saved, err := docToModelVariation(snap)
	if err != nil {
		return nil, err
	}

	return &saved, nil
}

func (r *ModelRepositoryFS) UpdateModelVariation(ctx context.Context, variationID string, updates modeldom.ModelVariationUpdate) (*modeldom.ModelVariation, error) {
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

func (r *ModelRepositoryFS) DeleteModelVariation(ctx context.Context, variationID string) (*modeldom.ModelVariation, error) {
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

	return &v, nil
}

// ------------------------------------------------------------
// ReplaceModelVariations（大量更新、ライブ）
// ------------------------------------------------------------

func (r *ModelRepositoryFS) ReplaceModelVariations(ctx context.Context, vars []modeldom.NewApparelModelVariation) ([]modeldom.ModelVariation, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	if len(vars) == 0 {
		return []modeldom.ModelVariation{}, nil
	}

	productBlueprintID := vars[0].ProductBlueprintID
	if productBlueprintID == "" {
		return nil, modeldom.ErrInvalidBlueprintID
	}

	for _, v := range vars {
		if v.ProductBlueprintID != productBlueprintID {
			return nil, fmt.Errorf("ReplaceModelVariations: mixed ProductBlueprintID is not allowed")
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

				mv := modeldom.ApparelModelVariation{
					ID:                 docRef.ID,
					ProductBlueprintID: productBlueprintID,
					ModelNumber:        nv.ModelNumber,
					Size:               nv.Size,
					Color: modeldom.Color{
						Name: nv.Color.Name,
						RGB:  nv.Color.RGB,
					},
					Measurements: nv.Measurements,
					CreatedAt:    now,
					UpdatedAt:    now,
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
// RepositoryPort 追加（不足メソッド）
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

	q := strings.ToLower(filter.SearchQuery)

	filtered := make([]modeldom.ModelVariation, 0, len(all))
	for _, raw := range all {
		v, ok := toApparelModelVariation(raw)
		if !ok {
			continue
		}

		if !inSet(v.Size, filter.Sizes) {
			continue
		}
		if !inSet(v.Color.Name, filter.Colors) {
			continue
		}
		if !inSet(v.ModelNumber, filter.ModelNumbers) {
			continue
		}

		if filter.CreatedFrom != nil && !v.CreatedAt.IsZero() && v.CreatedAt.Before(filter.CreatedFrom.UTC()) {
			continue
		}
		if filter.CreatedTo != nil && !v.CreatedAt.IsZero() && v.CreatedAt.After(filter.CreatedTo.UTC()) {
			continue
		}
		if filter.UpdatedFrom != nil && !v.UpdatedAt.IsZero() && v.UpdatedAt.Before(filter.UpdatedFrom.UTC()) {
			continue
		}
		if filter.UpdatedTo != nil && !v.UpdatedAt.IsZero() && v.UpdatedAt.After(filter.UpdatedTo.UTC()) {
			continue
		}

		if filter.Deleted != nil && *filter.Deleted {
			continue
		}

		if q != "" {
			hay := strings.ToLower(v.ModelNumber + " " + v.Size + " " + v.Color.Name)
			if !strings.Contains(hay, q) {
				continue
			}
		}

		filtered = append(filtered, v)
	}

	sort.Slice(filtered, func(i, j int) bool {
		a, _ := toApparelModelVariation(filtered[i])
		b, _ := toApparelModelVariation(filtered[j])

		if !a.UpdatedAt.Equal(b.UpdatedAt) {
			return a.UpdatedAt.After(b.UpdatedAt)
		}
		if !a.CreatedAt.Equal(b.CreatedAt) {
			return a.CreatedAt.After(b.CreatedAt)
		}
		return a.ID < b.ID
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
		v, ok := toApparelModelVariation(raw)
		if !ok {
			continue
		}

		s := v.ModelNumber
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

	var color modeldom.Color
	if raw, ok := data["color"]; ok && raw != nil {
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
	}

	getMeasurements := func() modeldom.Measurements {
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

	var createdAt, updatedAt time.Time
	if v, ok := data["createdAt"].(time.Time); ok {
		createdAt = v.UTC()
	}
	if v, ok := data["updatedAt"].(time.Time); ok {
		updatedAt = v.UTC()
	}

	var createdBy *string
	if v, ok := data["createdBy"].(string); ok && v != "" {
		s := v
		createdBy = &s
	}

	var updatedBy *string
	if v, ok := data["updatedBy"].(string); ok && v != "" {
		s := v
		updatedBy = &s
	}

	return modeldom.ApparelModelVariation{
		ID:                 doc.Ref.ID,
		ProductBlueprintID: getStr("productBlueprintId"),
		ModelNumber:        getStr("modelNumber"),
		Size:               getStr("size"),
		Color:              color,
		Measurements:       getMeasurements(),
		CreatedAt:          createdAt,
		CreatedBy:          createdBy,
		UpdatedAt:          updatedAt,
		UpdatedBy:          updatedBy,
	}, nil
}

func modelVariationToDoc(v modeldom.ApparelModelVariation) map[string]any {
	m := map[string]any{
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
