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
	return r.Client.Collection("model_variations")
}

// ------------------------------------------------------------
// model_sets 取得
// ------------------------------------------------------------

func (r *ModelRepositoryFS) GetModelData(ctx context.Context, productID string) (*modeldom.ModelData, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, modeldom.ErrNotFound
	}

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

	blueprintID := ""
	if v, ok := data["productBlueprintId"].(string); ok {
		blueprintID = strings.TrimSpace(v)
	}
	if blueprintID == "" {
		return nil, fmt.Errorf("model_set missing productBlueprintId: %s", snap.Ref.ID)
	}

	var updatedAt time.Time
	if v, ok := data["updatedAt"].(time.Time); ok {
		updatedAt = v.UTC()
	}

	vars, err := r.listVariationsByBlueprintID(ctx, blueprintID)
	if err != nil {
		return nil, err
	}

	return &modeldom.ModelData{
		ProductID:          productID,
		ProductBlueprintID: blueprintID,
		Variations:         vars,
		UpdatedAt:          updatedAt,
	}, nil
}

func (r *ModelRepositoryFS) GetModelDataByBlueprintID(ctx context.Context, blueprintID string) (*modeldom.ModelData, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	blueprintID = strings.TrimSpace(blueprintID)
	if blueprintID == "" {
		return nil, modeldom.ErrNotFound
	}

	q := r.modelSetsCol().Where("productBlueprintId", "==", blueprintID).Limit(1)
	it := q.Documents(ctx)
	defer it.Stop()

	snap, err := it.Next()
	if err == iterator.Done {
		return nil, modeldom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	data := snap.Data()
	if data == nil {
		return nil, fmt.Errorf("empty model_set: %s", snap.Ref.ID)
	}

	productID := strings.TrimSpace(snap.Ref.ID)

	var updatedAt time.Time
	if v, ok := data["updatedAt"].(time.Time); ok {
		updatedAt = v.UTC()
	}

	vars, err := r.listVariationsByBlueprintID(ctx, blueprintID)
	if err != nil {
		return nil, err
	}

	return &modeldom.ModelData{
		ProductID:          productID,
		ProductBlueprintID: blueprintID,
		Variations:         vars,
		UpdatedAt:          updatedAt,
	}, nil
}

// ------------------------------------------------------------
// model_sets 更新（残す）
// ------------------------------------------------------------

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

	if v, ok := updates["productBlueprintID"]; ok {
		if s, ok2 := v.(string); ok2 {
			fsUpdates = append(fsUpdates, firestore.Update{
				Path:  "productBlueprintId",
				Value: strings.TrimSpace(s),
			})
		}
	}

	// updatedAt 必ず更新
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

	return r.GetModelData(ctx, productID)
}

// ------------------------------------------------------------
// Variation CRUD
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

func (r *ModelRepositoryFS) CreateModelVariation(ctx context.Context, productID string, variation modeldom.NewModelVariation) (*modeldom.ModelVariation, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, modeldom.ErrNotFound
	}

	// resolve blueprint
	snap, err := r.modelSetsCol().Doc(productID).Get(ctx)
	if err != nil {
		return nil, modeldom.ErrNotFound
	}

	data := snap.Data()
	blueprintID, _ := data["productBlueprintId"].(string)
	blueprintID = strings.TrimSpace(blueprintID)
	if blueprintID == "" {
		return nil, fmt.Errorf("model_set missing productBlueprintId")
	}

	docRef := r.variationsCol().NewDoc()

	v := modeldom.ModelVariation{
		ID:                 docRef.ID,
		ProductBlueprintID: blueprintID,
		ModelNumber:        strings.TrimSpace(variation.ModelNumber),
		Size:               strings.TrimSpace(variation.Size),
		Color:              strings.TrimSpace(variation.Color),
		Measurements:       variation.Measurements,
	}

	if _, err := docRef.Create(ctx, modelVariationToDoc(v)); err != nil {
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

	docRef := r.variationsCol().Doc(variationID)
	var fsUpdates []firestore.Update

	if updates.Size != nil {
		fsUpdates = append(fsUpdates, firestore.Update{Path: "size", Value: *updates.Size})
	}
	if updates.Color != nil {
		fsUpdates = append(fsUpdates, firestore.Update{Path: "color", Value: *updates.Color})
	}
	if updates.ModelNumber != nil {
		fsUpdates = append(fsUpdates, firestore.Update{Path: "modelNumber", Value: *updates.ModelNumber})
	}
	if updates.Measurements != nil {
		fsUpdates = append(fsUpdates, firestore.Update{Path: "measurements", Value: updates.Measurements})
	}

	if len(fsUpdates) == 0 {
		return r.GetModelVariationByID(ctx, variationID)
	}

	if _, err := docRef.Update(ctx, fsUpdates); err != nil {
		return nil, err
	}

	return r.GetModelVariationByID(ctx, variationID)
}

func (r *ModelRepositoryFS) DeleteModelVariation(ctx context.Context, variationID string) (*modeldom.ModelVariation, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	snap, err := r.variationsCol().Doc(variationID).Get(ctx)
	if err != nil {
		return nil, modeldom.ErrNotFound
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

// ------------------------------------------------------------
// ReplaceModelVariations（大量更新）
// ------------------------------------------------------------
func (r *ModelRepositoryFS) ReplaceModelVariations(
	ctx context.Context,
	productID string,
	variations []modeldom.NewModelVariation,
) ([]modeldom.ModelVariation, error) {

	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, modeldom.ErrNotFound
	}

	// 1. モデルセットから blueprintID を取得
	snap, err := r.modelSetsCol().Doc(productID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, modeldom.ErrNotFound
		}
		return nil, err
	}

	data := snap.Data()
	blueprintID, _ := data["productBlueprintId"].(string)
	blueprintID = strings.TrimSpace(blueprintID)
	if blueprintID == "" {
		return nil, fmt.Errorf("model_set missing productBlueprintId")
	}

	// 2. 古い variations を全取得
	it := r.variationsCol().
		Where("productBlueprintId", "==", blueprintID).
		Documents(ctx)

	var toDelete []*firestore.DocumentRef
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

	// Delete & Insert は chunk に分けて安全に処理
	const chunkSize = 400

	// ------------------------------------------------------------
	// 4. variations の新規挿入（トランザクション使用）
	// ------------------------------------------------------------
	for i := 0; i < len(variations); i += chunkSize {
		end := i + chunkSize
		if end > len(variations) {
			end = len(variations)
		}
		chunk := variations[i:end]

		err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			for _, nv := range chunk {
				docRef := r.variationsCol().NewDoc()

				mv := modeldom.ModelVariation{
					ID:                 docRef.ID,
					ProductBlueprintID: blueprintID,
					ModelNumber:        strings.TrimSpace(nv.ModelNumber),
					Size:               strings.TrimSpace(nv.Size),
					Color:              strings.TrimSpace(nv.Color),
					Measurements:       nv.Measurements,
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

	// 5. 挿入後の最新 variations を返す
	return r.listVariationsByBlueprintID(ctx, blueprintID)
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

func (r *ModelRepositoryFS) listVariationsByBlueprintID(ctx context.Context, blueprintID string) ([]modeldom.ModelVariation, error) {
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

	getMeasurements := func() modeldom.Measurements {
		raw, ok := data["measurements"]
		if !ok || raw == nil {
			return nil
		}
		switch vv := raw.(type) {
		case map[string]any:
			out := make(modeldom.Measurements)
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
			_ = json.Unmarshal([]byte(vv), &m)
			return m
		}
		return nil
	}

	return modeldom.ModelVariation{
		ID:                 doc.Ref.ID,
		ProductBlueprintID: getStr("productBlueprintId"),
		ModelNumber:        getStr("modelNumber"),
		Size:               getStr("size"),
		Color:              getStr("color"),
		Measurements:       getMeasurements(),
	}, nil
}

func modelVariationToDoc(v modeldom.ModelVariation) map[string]any {
	m := map[string]any{
		"productBlueprintId": v.ProductBlueprintID,
		"modelNumber":        v.ModelNumber,
		"size":               v.Size,
		"color":              v.Color,
	}

	if v.Measurements != nil {
		m["measurements"] = v.Measurements
	}

	return m
}
