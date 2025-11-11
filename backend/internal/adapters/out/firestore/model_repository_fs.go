// backend/internal/adapters/out/firestore/model_repository_fs.go
package firestore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
	modeldom "narratives/internal/domain/model"
)

// ModelRepositoryFS is a Firestore-based implementation of model.RepositoryPort.
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
	return r.Client.Collection("model_variations")
}

// WithTx executes fn "within a transaction".
// For Firestore, we use RunTransaction, but most methods work on the client
// directly, so this is a best-effort wrapper that only propagates context.
func (r *ModelRepositoryFS) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}
	return r.Client.RunTransaction(ctx, func(txCtx context.Context, _ *firestore.Transaction) error {
		return fn(txCtx)
	})
}

// ==========================
// Product-scoped model data
// ==========================

func (r *ModelRepositoryFS) GetModelData(ctx context.Context, productID string) (*modeldom.ModelData, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, modeldom.ErrNotFound
	}

	// Treat productID as document ID in model_sets.
	snap, err := r.modelSetsCol().Doc(productID).Get(ctx)
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

	productBlueprintID := ""
	if v, ok := data["productBlueprintId"].(string); ok {
		productBlueprintID = strings.TrimSpace(v)
	}
	if productBlueprintID == "" {
		if v, ok := data["product_blueprint_id"].(string); ok {
			productBlueprintID = strings.TrimSpace(v)
		}
	}
	if productBlueprintID == "" {
		return nil, fmt.Errorf("model_set missing productBlueprintId: %s", snap.Ref.ID)
	}

	var updatedAt time.Time
	if v, ok := data["updatedAt"].(time.Time); ok {
		updatedAt = v.UTC()
	} else if v, ok := data["updated_at"].(time.Time); ok {
		updatedAt = v.UTC()
	}

	vars, err := r.listVariationsByBlueprintID(ctx, productBlueprintID)
	if err != nil {
		return nil, err
	}

	return &modeldom.ModelData{
		ProductID:          productID,
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

	// Query model_sets by productBlueprintId (new field), then fallback to legacy.
	q := r.modelSetsCol().Where("productBlueprintId", "==", productBlueprintID).Limit(1)
	it := q.Documents(ctx)
	defer it.Stop()

	snap, err := it.Next()
	if err == iterator.Done {
		q2 := r.modelSetsCol().Where("product_blueprint_id", "==", productBlueprintID).Limit(1)
		it2 := q2.Documents(ctx)
		defer it2.Stop()
		snap, err = it2.Next()
		if err == iterator.Done {
			return nil, modeldom.ErrNotFound
		}
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	data := snap.Data()
	if data == nil {
		return nil, fmt.Errorf("empty model_set document: %s", snap.Ref.ID)
	}

	productID := strings.TrimSpace(snap.Ref.ID)

	var updatedAt time.Time
	if v, ok := data["updatedAt"].(time.Time); ok {
		updatedAt = v.UTC()
	} else if v, ok := data["updated_at"].(time.Time); ok {
		updatedAt = v.UTC()
	}

	vars, err := r.listVariationsByBlueprintID(ctx, productBlueprintID)
	if err != nil {
		return nil, err
	}

	return &modeldom.ModelData{
		ProductID:          productID,
		ProductBlueprintID: productBlueprintID,
		Variations:         vars,
		UpdatedAt:          updatedAt,
	}, nil
}

func (r *ModelRepositoryFS) UpdateModelData(ctx context.Context, productID string, updates modeldom.ModelDataUpdate) (*modeldom.ModelData, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, modeldom.ErrNotFound
	}

	docRef := r.modelSetsCol().Doc(productID)
	var fsUpdates []firestore.Update

	// Supported keys: productBlueprintID / product_blueprint_id
	if v, ok := updates["productBlueprintID"]; ok {
		if s, ok2 := v.(string); ok2 && strings.TrimSpace(s) != "" {
			fsUpdates = append(fsUpdates, firestore.Update{
				Path:  "productBlueprintId",
				Value: strings.TrimSpace(s),
			})
		}
	}
	if v, ok := updates["product_blueprint_id"]; ok {
		if s, ok2 := v.(string); ok2 && strings.TrimSpace(s) != "" {
			fsUpdates = append(fsUpdates, firestore.Update{
				Path:  "productBlueprintId",
				Value: strings.TrimSpace(s),
			})
		}
	}

	// Always bump updatedAt like the PG adapter.
	fsUpdates = append(fsUpdates, firestore.Update{
		Path:  "updatedAt",
		Value: time.Now().UTC(),
	})

	if len(fsUpdates) == 0 {
		// Nothing changed; just reload.
		return r.GetModelData(ctx, productID)
	}

	_, err := docRef.Update(ctx, fsUpdates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, modeldom.ErrNotFound
		}
		return nil, err
	}

	return r.GetModelData(ctx, productID)
}

// ==========================
// Variations CRUD + listing
// ==========================

func (r *ModelRepositoryFS) ListVariations(
	ctx context.Context,
	filter modeldom.VariationFilter,
	sort modeldom.VariationSort,
	page modeldom.Page,
) (modeldom.VariationPageResult, error) {
	if r.Client == nil {
		return modeldom.VariationPageResult{}, errors.New("firestore client is nil")
	}

	pageNum, perPage, offset := fscommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	q := r.variationsCol().Query
	q = applyVariationSort(q, sort)

	// We fetch and then apply the full VariationFilter in-memory
	// (some conditions are not directly expressible in a single Firestore query).
	it := q.Documents(ctx)
	defer it.Stop()

	var all []modeldom.ModelVariation
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return modeldom.VariationPageResult{}, err
		}
		v, err := docToModelVariation(doc)
		if err != nil {
			return modeldom.VariationPageResult{}, err
		}
		if matchVariationFilter(v, filter) {
			all = append(all, v)
		}
	}

	total := len(all)
	if total == 0 {
		return modeldom.VariationPageResult{
			Items:      []modeldom.ModelVariation{},
			TotalCount: 0,
			TotalPages: 0,
			Page:       pageNum,
			PerPage:    perPage,
		}, nil
	}

	if offset > total {
		offset = total
	}
	end := offset + perPage
	if end > total {
		end = total
	}
	items := all[offset:end]

	return modeldom.VariationPageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: fscommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *ModelRepositoryFS) CountVariations(ctx context.Context, filter modeldom.VariationFilter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("firestore client is nil")
	}

	it := r.variationsCol().Documents(ctx)
	defer it.Stop()

	total := 0
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		v, err := docToModelVariation(doc)
		if err != nil {
			return 0, err
		}
		if matchVariationFilter(v, filter) {
			total++
		}
	}
	return total, nil
}

func (r *ModelRepositoryFS) GetModelVariations(ctx context.Context, productID string) ([]modeldom.ModelVariation, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, modeldom.ErrNotFound
	}

	// Resolve blueprint via model_sets (doc ID = productID).
	snap, err := r.modelSetsCol().Doc(productID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, modeldom.ErrNotFound
		}
		return nil, err
	}

	data := snap.Data()
	var blueprintID string
	if v, ok := data["productBlueprintId"].(string); ok {
		blueprintID = strings.TrimSpace(v)
	}
	if blueprintID == "" {
		if v, ok := data["product_blueprint_id"].(string); ok {
			blueprintID = strings.TrimSpace(v)
		}
	}
	if blueprintID == "" {
		return nil, fmt.Errorf("model_set missing productBlueprintId: %s", snap.Ref.ID)
	}

	return r.listVariationsByBlueprintID(ctx, blueprintID)
}

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

func (r *ModelRepositoryFS) CreateModelVariation(ctx context.Context, productID string, variation modeldom.NewModelVariation) (*modeldom.ModelVariation, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, modeldom.ErrNotFound
	}

	// Resolve blueprint via model_sets.
	snap, err := r.modelSetsCol().Doc(productID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, modeldom.ErrNotFound
		}
		return nil, err
	}
	data := snap.Data()
	var blueprintID string
	if v, ok := data["productBlueprintId"].(string); ok {
		blueprintID = strings.TrimSpace(v)
	}
	if blueprintID == "" {
		if v, ok := data["product_blueprint_id"].(string); ok {
			blueprintID = strings.TrimSpace(v)
		}
	}
	if blueprintID == "" {
		return nil, fmt.Errorf("model_set missing productBlueprintId: %s", snap.Ref.ID)
	}

	now := time.Now().UTC()
	docRef := r.variationsCol().NewDoc()

	v := modeldom.ModelVariation{
		ID:                 docRef.ID,
		ProductBlueprintID: blueprintID,
		ModelNumber:        strings.TrimSpace(variation.ModelNumber),
		Size:               strings.TrimSpace(variation.Size),
		Color:              strings.TrimSpace(variation.Color),
		Measurements:       variation.Measurements,
		CreatedAt:          &now,
		UpdatedAt:          &now,
	}

	dataMap := modelVariationToDoc(v)

	if _, err := docRef.Create(ctx, dataMap); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil, modeldom.ErrConflict
		}
		return nil, err
	}

	savedSnap, err := docRef.Get(ctx)
	if err != nil {
		return nil, err
	}
	saved, err := docToModelVariation(savedSnap)
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

	docRef := r.variationsCol().Doc(variationID)
	var fsUpdates []firestore.Update

	if updates.Size != nil {
		fsUpdates = append(fsUpdates, firestore.Update{
			Path:  "size",
			Value: strings.TrimSpace(*updates.Size),
		})
	}
	if updates.Color != nil {
		fsUpdates = append(fsUpdates, firestore.Update{
			Path:  "color",
			Value: strings.TrimSpace(*updates.Color),
		})
	}
	if updates.ModelNumber != nil {
		fsUpdates = append(fsUpdates, firestore.Update{
			Path:  "modelNumber",
			Value: strings.TrimSpace(*updates.ModelNumber),
		})
	}
	// Measurements is a map alias; nil means "no update".
	if updates.Measurements != nil {
		fsUpdates = append(fsUpdates, firestore.Update{
			Path:  "measurements",
			Value: updates.Measurements,
		})
	}

	// Touch updatedAt
	fsUpdates = append(fsUpdates, firestore.Update{
		Path:  "updatedAt",
		Value: time.Now().UTC(),
	})

	if len(fsUpdates) == 0 {
		return r.GetModelVariationByID(ctx, variationID)
	}

	_, err := docRef.Update(ctx, fsUpdates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, modeldom.ErrNotFound
		}
		if status.Code(err) == codes.AlreadyExists {
			return nil, modeldom.ErrConflict
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

	// Load first (to return deleted entity).
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

	if _, err := snap.Ref.Delete(ctx); err != nil {
		return nil, err
	}

	return &v, nil
}

func (r *ModelRepositoryFS) ReplaceModelVariations(ctx context.Context, productID string, variations []modeldom.NewModelVariation) ([]modeldom.ModelVariation, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, modeldom.ErrNotFound
	}

	// Resolve blueprint.
	snap, err := r.modelSetsCol().Doc(productID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, modeldom.ErrNotFound
		}
		return nil, err
	}
	data := snap.Data()
	var blueprintID string
	if v, ok := data["productBlueprintId"].(string); ok {
		blueprintID = strings.TrimSpace(v)
	}
	if blueprintID == "" {
		if v, ok := data["product_blueprint_id"].(string); ok {
			blueprintID = strings.TrimSpace(v)
		}
	}
	if blueprintID == "" {
		return nil, fmt.Errorf("model_set missing productBlueprintId: %s", snap.Ref.ID)
	}

	// Delete existing variations for blueprintID.
	var toDelete []*firestore.DocumentRef
	it := r.variationsCol().
		Where("productBlueprintId", "==", blueprintID).
		Documents(ctx)
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		toDelete = append(toDelete, doc.Ref)
	}
	if len(toDelete) > 0 {
		b := r.Client.Batch()
		for i, ref := range toDelete {
			b.Delete(ref)
			if (i+1)%400 == 0 {
				if _, err := b.Commit(ctx); err != nil {
					return nil, err
				}
				b = r.Client.Batch()
			}
		}
		if _, err := b.Commit(ctx); err != nil {
			return nil, err
		}
	}

	// Insert new variations.
	now := time.Now().UTC()
	var out []modeldom.ModelVariation
	if len(variations) > 0 {
		b := r.Client.Batch()
		count := 0
		for _, nv := range variations {
			docRef := r.variationsCol().NewDoc()
			mv := modeldom.ModelVariation{
				ID:                 docRef.ID,
				ProductBlueprintID: blueprintID,
				ModelNumber:        strings.TrimSpace(nv.ModelNumber),
				Size:               strings.TrimSpace(nv.Size),
				Color:              strings.TrimSpace(nv.Color),
				Measurements:       nv.Measurements,
				CreatedAt:          &now,
				UpdatedAt:          &now,
			}
			b.Set(docRef, modelVariationToDoc(mv))
			out = append(out, mv)
			count++
			if count%400 == 0 {
				if _, err := b.Commit(ctx); err != nil {
					return nil, err
				}
				b = r.Client.Batch()
			}
		}
		if _, err := b.Commit(ctx); err != nil {
			return nil, err
		}
	}

	// Reload from Firestore for canonical state.
	return r.listVariationsByBlueprintID(ctx, blueprintID)
}

func (r *ModelRepositoryFS) GetSizeVariations(ctx context.Context, productID string) ([]modeldom.SizeVariation, error) {
	vars, err := r.GetModelVariations(ctx, productID)
	if err != nil {
		return nil, err
	}
	out := make([]modeldom.SizeVariation, 0, len(vars))
	for _, v := range vars {
		out = append(out, modeldom.SizeVariation{
			ID:           v.ID,
			Size:         v.Size,
			Measurements: v.Measurements,
		})
	}
	return out, nil
}

// Note: Without a production quantities source, return 0 quantities.
func (r *ModelRepositoryFS) GetProductionQuantities(ctx context.Context, productID string) ([]modeldom.ProductionQuantity, error) {
	_ = ctx
	_ = productID
	return []modeldom.ProductionQuantity{}, nil
}

func (r *ModelRepositoryFS) GetModelNumbers(ctx context.Context, productID string) ([]modeldom.ModelNumber, error) {
	vars, err := r.GetModelVariations(ctx, productID)
	if err != nil {
		return nil, err
	}
	out := make([]modeldom.ModelNumber, 0, len(vars))
	for _, v := range vars {
		out = append(out, modeldom.ModelNumber{
			Size:        v.Size,
			Color:       v.Color,
			ModelNumber: v.ModelNumber,
		})
	}
	return out, nil
}

func (r *ModelRepositoryFS) GetModelVariationsWithQuantity(ctx context.Context, productID string) ([]modeldom.ModelVariationWithQuantity, error) {
	vars, err := r.GetModelVariations(ctx, productID)
	if err != nil {
		return nil, err
	}
	out := make([]modeldom.ModelVariationWithQuantity, 0, len(vars))
	for _, v := range vars {
		out = append(out, modeldom.ModelVariationWithQuantity{
			ModelVariation: v,
			Quantity:       0,
		})
	}
	return out, nil
}

// ==========================
// Helpers
// ==========================

func (r *ModelRepositoryFS) listVariationsByBlueprintID(ctx context.Context, blueprintID string) ([]modeldom.ModelVariation, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	blueprintID = strings.TrimSpace(blueprintID)
	if blueprintID == "" {
		return []modeldom.ModelVariation{}, nil
	}

	q := r.variationsCol().
		Where("productBlueprintId", "==", blueprintID).
		OrderBy("modelNumber", firestore.Asc).
		OrderBy("size", firestore.Asc).
		OrderBy("color", firestore.Asc)

	it := q.Documents(ctx)
	defer it.Stop()

	var out []modeldom.ModelVariation
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
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
		return modeldom.ModelVariation{}, fmt.Errorf("empty model_variation document: %s", doc.Ref.ID)
	}

	getStr := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := data[k].(string); ok {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}
	getStrPtr := func(keys ...string) *string {
		for _, k := range keys {
			if v, ok := data[k].(string); ok {
				s := strings.TrimSpace(v)
				if s != "" {
					return &s
				}
			}
		}
		return nil
	}
	getTimePtr := func(keys ...string) *time.Time {
		for _, k := range keys {
			if v, ok := data[k].(time.Time); ok && !v.IsZero() {
				t := v.UTC()
				return &t
			}
		}
		return nil
	}
	getMeasurements := func() modeldom.Measurements {
		if raw, ok := data["measurements"]; ok && raw != nil {
			switch vv := raw.(type) {
			case map[string]any:
				out := make(modeldom.Measurements, len(vv))
				for k, v := range vv {
					switch n := v.(type) {
					case float64:
						out[k] = n
					case int64:
						out[k] = float64(n)
					case int:
						out[k] = float64(n)
					}
				}
				return out
			case string:
				if vv == "" {
					return nil
				}
				var m modeldom.Measurements
				if err := json.Unmarshal([]byte(vv), &m); err == nil {
					return m
				}
			}
		}
		return nil
	}

	return modeldom.ModelVariation{
		ID:                 doc.Ref.ID,
		ProductBlueprintID: getStr("productBlueprintId", "product_blueprint_id"),
		ModelNumber:        getStr("modelNumber", "model_number"),
		Size:               getStr("size"),
		Color:              getStr("color"),
		Measurements:       getMeasurements(),
		CreatedAt:          getTimePtr("createdAt", "created_at"),
		CreatedBy:          getStrPtr("createdBy", "created_by"),
		UpdatedAt:          getTimePtr("updatedAt", "updated_at"),
		UpdatedBy:          getStrPtr("updatedBy", "updated_by"),
		DeletedAt:          getTimePtr("deletedAt", "deleted_at"),
		DeletedBy:          getStrPtr("deletedBy", "deleted_by"),
	}, nil
}

func modelVariationToDoc(v modeldom.ModelVariation) map[string]any {
	m := map[string]any{
		"productBlueprintId": strings.TrimSpace(v.ProductBlueprintID),
		"modelNumber":        strings.TrimSpace(v.ModelNumber),
		"size":               strings.TrimSpace(v.Size),
		"color":              strings.TrimSpace(v.Color),
	}

	if v.Measurements != nil {
		m["measurements"] = v.Measurements
	}

	if v.CreatedAt != nil && !v.CreatedAt.IsZero() {
		m["createdAt"] = v.CreatedAt.UTC()
	}
	if v.CreatedBy != nil {
		if s := strings.TrimSpace(*v.CreatedBy); s != "" {
			m["createdBy"] = s
		}
	}
	if v.UpdatedAt != nil && !v.UpdatedAt.IsZero() {
		m["updatedAt"] = v.UpdatedAt.UTC()
	}
	if v.UpdatedBy != nil {
		if s := strings.TrimSpace(*v.UpdatedBy); s != "" {
			m["updatedBy"] = s
		}
	}
	if v.DeletedAt != nil && !v.DeletedAt.IsZero() {
		m["deletedAt"] = v.DeletedAt.UTC()
	}
	if v.DeletedBy != nil {
		if s := strings.TrimSpace(*v.DeletedBy); s != "" {
			m["deletedBy"] = s
		}
	}

	return m
}

// matchVariationFilter applies VariationFilter in-memory
// (Firestore-friendly mirror of the PG implementation).
func matchVariationFilter(v modeldom.ModelVariation, f modeldom.VariationFilter) bool {
	// ProductID: in PG this is via join; here, callers that rely on ProductID
	// should generally go through GetModelVariations/ReplaceModelVariations which
	// resolve via model_sets. For List/Count with ProductID set, this helper
	// currently does not resolve joins.
	if pid := strings.TrimSpace(f.ProductID); pid != "" {
		// Cannot verify without an extra lookup; reject by default to avoid
		// returning incorrect rows.
		return false
	}

	// ProductBlueprintID
	if pb := strings.TrimSpace(f.ProductBlueprintID); pb != "" {
		if strings.TrimSpace(v.ProductBlueprintID) != pb {
			return false
		}
	}

	// Sizes
	if len(f.Sizes) > 0 {
		ok := false
		for _, s := range f.Sizes {
			if strings.TrimSpace(s) == v.Size {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	// Colors
	if len(f.Colors) > 0 {
		ok := false
		for _, c := range f.Colors {
			if strings.TrimSpace(c) == v.Color {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	// ModelNumbers
	if len(f.ModelNumbers) > 0 {
		ok := false
		for _, mn := range f.ModelNumbers {
			if strings.TrimSpace(mn) == v.ModelNumber {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	// Free text search
	if q := strings.TrimSpace(f.SearchQuery); q != "" {
		lq := strings.ToLower(q)
		haystack := strings.ToLower(v.ModelNumber + " " + v.Size + " " + v.Color)
		if !strings.Contains(haystack, lq) {
			return false
		}
	}

	// Time ranges
	if f.CreatedFrom != nil {
		if v.CreatedAt == nil || v.CreatedAt.Before(f.CreatedFrom.UTC()) {
			return false
		}
	}
	if f.CreatedTo != nil {
		if v.CreatedAt == nil || !v.CreatedAt.Before(f.CreatedTo.UTC()) {
			return false
		}
	}
	if f.UpdatedFrom != nil {
		if v.UpdatedAt == nil || v.UpdatedAt.Before(f.UpdatedFrom.UTC()) {
			return false
		}
	}
	if f.UpdatedTo != nil {
		if v.UpdatedAt == nil || !v.UpdatedAt.Before(f.UpdatedTo.UTC()) {
			return false
		}
	}

	// Deletion filter
	if f.Deleted != nil {
		if *f.Deleted {
			if v.DeletedAt == nil {
				return false
			}
		} else {
			if v.DeletedAt != nil {
				return false
			}
		}
	}

	return true
}

// applyVariationSort maps VariationSort to Firestore orderBy.
func applyVariationSort(q firestore.Query, sort modeldom.VariationSort) firestore.Query {
	col := strings.ToLower(strings.TrimSpace(string(sort.Column)))
	var field string

	switch col {
	case "modelnumber", "model_number":
		field = "modelNumber"
	case "size":
		field = "size"
	case "color":
		field = "color"
	case "createdat", "created_at":
		field = "createdAt"
	case "updatedat", "updated_at":
		field = "updatedAt"
	default:
		// Default ordering
		return q.OrderBy("modelNumber", firestore.Asc).
			OrderBy("size", firestore.Asc).
			OrderBy("color", firestore.Asc)
	}

	dir := firestore.Asc
	if strings.EqualFold(string(sort.Order), "desc") {
		dir = firestore.Desc
	}

	// Stable secondary sort by document ID.
	return q.OrderBy(field, dir).OrderBy(firestore.DocumentID, dir)
}
