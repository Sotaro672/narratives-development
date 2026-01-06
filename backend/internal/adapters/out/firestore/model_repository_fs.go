// backend\internal\adapters\out\firestore\model_repository_fs.go
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
	productBlueprintID = strings.TrimSpace(productBlueprintID)
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

	// Firestore 側に productBlueprintId があればそれを正とする
	if v, ok := data["productBlueprintId"].(string); ok {
		productBlueprintID = strings.TrimSpace(v)
	}
	if productBlueprintID == "" {
		return nil, fmt.Errorf("model_set missing productBlueprintId: %s", snap.Ref.ID)
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
	productBlueprintID = strings.TrimSpace(productBlueprintID)
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

	productBlueprintID = strings.TrimSpace(productBlueprintID)
	if productBlueprintID == "" {
		return nil, modeldom.ErrNotFound
	}

	docRef := r.modelSetsCol().Doc(productBlueprintID)
	var fsUpdates []firestore.Update

	if v, ok := updates["productBlueprintID"]; ok {
		if s, ok2 := v.(string); ok2 {
			fsUpdates = append(fsUpdates, firestore.Update{
				Path:  "productBlueprintId",
				Value: strings.TrimSpace(s),
			})
		}
	}

	// updatedAt は必ず更新
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
	variationID = strings.TrimSpace(variationID)
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

func (r *ModelRepositoryFS) CreateModelVariation(ctx context.Context, variation modeldom.NewModelVariation) (*modeldom.ModelVariation, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	now := time.Now().UTC()
	docRef := r.variationsCol().NewDoc()

	mv := modeldom.ModelVariation{
		ID:                 docRef.ID,
		ProductBlueprintID: strings.TrimSpace(variation.ProductBlueprintID),
		ModelNumber:        strings.TrimSpace(variation.ModelNumber),
		Size:               strings.TrimSpace(variation.Size),
		Color: modeldom.Color{
			Name: strings.TrimSpace(variation.Color.Name),
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
	variationID = strings.TrimSpace(variationID)
	if variationID == "" {
		return nil, modeldom.ErrNotFound
	}

	log.Printf("[ModelRepositoryFS] UpdateModelVariation id=%s path=models/%s", variationID, variationID)

	docRef := r.variationsCol().Doc(variationID)
	var fsUpdates []firestore.Update

	if updates.Size != nil {
		fsUpdates = append(fsUpdates, firestore.Update{Path: "size", Value: strings.TrimSpace(*updates.Size)})
	}
	if updates.Color != nil {
		fsUpdates = append(fsUpdates, firestore.Update{
			Path: "color",
			Value: map[string]any{
				"name": strings.TrimSpace(updates.Color.Name),
				"rgb":  updates.Color.RGB,
			},
		})
	}
	if updates.ModelNumber != nil {
		fsUpdates = append(fsUpdates, firestore.Update{Path: "modelNumber", Value: strings.TrimSpace(*updates.ModelNumber)})
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

	variationID = strings.TrimSpace(variationID)
	if variationID == "" {
		return nil, modeldom.ErrNotFound
	}

	docRef := r.variationsCol().Doc(variationID)

	// 削除前の状態を取得して返す
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

	// 物理削除
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

func (r *ModelRepositoryFS) ReplaceModelVariations(ctx context.Context, vars []modeldom.NewModelVariation) ([]modeldom.ModelVariation, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	if len(vars) == 0 {
		return []modeldom.ModelVariation{}, nil
	}

	productBlueprintID := strings.TrimSpace(vars[0].ProductBlueprintID)
	if productBlueprintID == "" {
		return nil, modeldom.ErrInvalidBlueprintID
	}

	for _, v := range vars {
		if strings.TrimSpace(v.ProductBlueprintID) != productBlueprintID {
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
				ref := r.variationsCol().Doc(v.ID)
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

				mv := modeldom.ModelVariation{
					ID:                 docRef.ID,
					ProductBlueprintID: productBlueprintID,
					ModelNumber:        strings.TrimSpace(nv.ModelNumber),
					Size:               strings.TrimSpace(nv.Size),
					Color: modeldom.Color{
						Name: strings.TrimSpace(nv.Color.Name),
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

// ListVariations implements model.RepositoryPort.
// Firestore 側のクエリ制約を避けるため、まず blueprint 単位で取得 → in-memory filter → paginate.
func (r *ModelRepositoryFS) ListVariations(
	ctx context.Context,
	filter modeldom.VariationFilter,
	page modeldom.Page,
) (modeldom.VariationPageResult, error) {
	if r.Client == nil {
		return modeldom.VariationPageResult{}, errors.New("firestore client is nil")
	}

	// このFS実装は「ProductBlueprintID」を主キーとして扱う（ProductID は互換として受ける）
	pbID := strings.TrimSpace(filter.ProductBlueprintID)
	if pbID == "" {
		pbID = strings.TrimSpace(filter.ProductID)
	}
	if pbID == "" {
		return modeldom.VariationPageResult{}, modeldom.ErrInvalidBlueprintID
	}

	all, err := r.listVariationsByProductBlueprintID(ctx, pbID)
	if err != nil {
		return modeldom.VariationPageResult{}, err
	}

	// ---- in-memory filter ----
	inSet := func(s string, xs []string) bool {
		if len(xs) == 0 {
			return true
		}
		for _, x := range xs {
			if strings.EqualFold(strings.TrimSpace(x), s) {
				return true
			}
		}
		return false
	}

	q := strings.ToLower(strings.TrimSpace(filter.SearchQuery))

	var filtered []modeldom.ModelVariation
	for _, v := range all {
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

		// deleted はドメインから削除済み（常に非deleted扱い）
		if filter.Deleted != nil && *filter.Deleted {
			continue
		}

		if q != "" {
			hay := strings.ToLower(strings.TrimSpace(v.ModelNumber) + " " + strings.TrimSpace(v.Size) + " " + strings.TrimSpace(v.Color.Name))
			if !strings.Contains(hay, q) {
				continue
			}
		}

		filtered = append(filtered, v)
	}

	// sort: UpdatedAt desc, then CreatedAt desc, then ID
	sort.Slice(filtered, func(i, j int) bool {
		a, b := filtered[i], filtered[j]
		if !a.UpdatedAt.Equal(b.UpdatedAt) {
			return a.UpdatedAt.After(b.UpdatedAt)
		}
		if !a.CreatedAt.Equal(b.CreatedAt) {
			return a.CreatedAt.After(b.CreatedAt)
		}
		return a.ID < b.ID
	})

	// paginate
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

// GetModelVariations implements model.RepositoryPort.
// このFS実装では productID も productBlueprintID として扱う（互換）。
func (r *ModelRepositoryFS) GetModelVariations(ctx context.Context, productID string) ([]modeldom.ModelVariation, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, modeldom.ErrInvalidProductID
	}
	return r.listVariationsByProductBlueprintID(ctx, productID)
}

// GetSizeVariations implements model.RepositoryPort.
// 型定義の差異に強いように reflect で値を詰める（struct/alias/string どれでもOK）。
func (r *ModelRepositoryFS) GetSizeVariations(ctx context.Context, productID string) ([]modeldom.SizeVariation, error) {
	vars, err := r.GetModelVariations(ctx, productID)
	if err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	var sizes []string
	for _, v := range vars {
		s := strings.TrimSpace(v.Size)
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

// GetModelNumbers implements model.RepositoryPort.
// 型定義の差異に強いように reflect で値を詰める（struct/alias/string どれでもOK）。
func (r *ModelRepositoryFS) GetModelNumbers(ctx context.Context, productID string) ([]modeldom.ModelNumber, error) {
	vars, err := r.GetModelVariations(ctx, productID)
	if err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	var nums []string
	for _, v := range vars {
		s := strings.TrimSpace(v.ModelNumber)
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
// 互換: 以前の usecase / 旧インターフェース用
// ------------------------------------------------------------

func (r *ModelRepositoryFS) ListModelVariationsByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]modeldom.ModelVariation, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	productBlueprintID = strings.TrimSpace(productBlueprintID)
	if productBlueprintID == "" {
		return nil, modeldom.ErrInvalidBlueprintID
	}
	return r.listVariationsByProductBlueprintID(ctx, productBlueprintID)
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

func (r *ModelRepositoryFS) listVariationsByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]modeldom.ModelVariation, error) {
	q := r.variationsCol().
		Where("productBlueprintId", "==", productBlueprintID)

	it := q.Documents(ctx)
	defer it.Stop()

	var out []modeldom.ModelVariation
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
		return modeldom.ModelVariation{}, fmt.Errorf("empty variation: %s", doc.Ref.ID)
	}

	getStr := func(k string) string {
		if v, ok := data[k].(string); ok {
			return strings.TrimSpace(v)
		}
		return ""
	}

	var color modeldom.Color
	if raw, ok := data["color"]; ok && raw != nil {
		if v, ok := raw.(map[string]any); ok {
			if n, ok2 := v["name"].(string); ok2 {
				color.Name = strings.TrimSpace(n)
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
	if v, ok := data["createdBy"].(string); ok && strings.TrimSpace(v) != "" {
		s := strings.TrimSpace(v)
		createdBy = &s
	}

	var updatedBy *string
	if v, ok := data["updatedBy"].(string); ok && strings.TrimSpace(v) != "" {
		s := strings.TrimSpace(v)
		updatedBy = &s
	}

	return modeldom.ModelVariation{
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

func modelVariationToDoc(v modeldom.ModelVariation) map[string]any {
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

// ListModelIDsByProductBlueprintID returns model variation IDs for a product blueprint.
// This is used by ListCreateQuery to build PriceRows independent of inventory stock.
func (r *ModelRepositoryFS) ListModelIDsByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]string, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	pbID := strings.TrimSpace(productBlueprintID)
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
		id := strings.TrimSpace(v.ID)
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
