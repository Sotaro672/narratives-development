package firestore

import (
	"context"
	"errors"
	"fmt"
	"log"
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
// model_sets 取得
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

	// ★ ここを := ではなく = に変更（引数を上書きして使う）
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
// model_sets 更新
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

	return r.GetModelData(ctx, productBlueprintID)
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
	// ★ 論理削除済みのものは「存在しない」とみなす
	if v.DeletedAt != nil {
		return nil, modeldom.ErrNotFound
	}
	return &v, nil
}

// CreateModelVariation（productBlueprintID は NewModelVariation から利用）
func (r *ModelRepositoryFS) CreateModelVariation(
	ctx context.Context,
	variation modeldom.NewModelVariation,
) (*modeldom.ModelVariation, error) {
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

	// どのパスを更新しようとしているかログ出力
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

	// updatedAt は必ず更新
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

// ★ DeleteModelVariation: 物理削除 → 論理削除に変更
func (r *ModelRepositoryFS) DeleteModelVariation(ctx context.Context, variationID string) (*modeldom.ModelVariation, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	variationID = strings.TrimSpace(variationID)
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

	// すでに論理削除されている場合はそのまま返す
	if v.DeletedAt != nil {
		return &v, nil
	}

	now := time.Now().UTC()
	v.DeletedAt = &now
	// DeletedBy は、現状 context からの取得はしていないので nil のまま
	// 必要になればここで actor を context から取り出してセットする

	// Firestore 上では deletedAt / updatedAt を更新
	if _, err := docRef.Update(ctx, []firestore.Update{
		{Path: "deletedAt", Value: now},
		{Path: "updatedAt", Value: now},
	}); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, modeldom.ErrNotFound
		}
		return nil, err
	}

	return &v, nil
}

// ------------------------------------------------------------
// ReplaceModelVariations（大量更新）
// ------------------------------------------------------------

func (r *ModelRepositoryFS) ReplaceModelVariations(
	ctx context.Context,
	vars []modeldom.NewModelVariation,
) ([]modeldom.ModelVariation, error) {

	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	// 空なら何もしない（必要であればここをエラーに変えてもよい）
	if len(vars) == 0 {
		return []modeldom.ModelVariation{}, nil
	}

	// NewModelVariation 側の ProductBlueprintID から紐付けキーを解決
	productBlueprintID := strings.TrimSpace(vars[0].ProductBlueprintID)
	if productBlueprintID == "" {
		return nil, modeldom.ErrInvalidBlueprintID
	}

	// 安全のため、全要素が同じ ProductBlueprintID を持っているか確認
	for _, v := range vars {
		if strings.TrimSpace(v.ProductBlueprintID) != productBlueprintID {
			return nil, fmt.Errorf("ReplaceModelVariations: mixed ProductBlueprintID is not allowed")
		}
	}

	// 既存 variations を削除（productBlueprint 単位で）
	// ※ここは依然として物理削除。履歴を残したい場合は同様に論理削除へ変更する。
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

		batch := r.Client.Batch()
		for _, v := range chunk {
			ref := r.variationsCol().Doc(v.ID)
			batch.Delete(ref)
		}
		if _, err := batch.Commit(ctx); err != nil {
			return nil, err
		}
	}

	// 新規 variations を挿入
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

	// 挿入後の最新 variations を返す
	return r.listVariationsByProductBlueprintID(ctx, productBlueprintID)
}

// ------------------------------------------------------------
// ModelRepo interface の追加メソッド実装
// ------------------------------------------------------------

// 与えられた productBlueprintID に紐づく ModelVariation 一覧を返す
func (r *ModelRepositoryFS) ListModelVariationsByProductBlueprintID(
	ctx context.Context,
	productBlueprintID string,
) ([]modeldom.ModelVariation, error) {
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
		// ★ 論理削除済み（deletedAt が入っている）ものは一覧から除外
		if v.DeletedAt != nil {
			continue
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

	// Color は { name, rgb } として保存されている前提
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

	// measurements: map[string]int として扱う
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

	var deletedAt *time.Time
	if v, ok := data["deletedAt"].(time.Time); ok {
		t := v.UTC()
		deletedAt = &t
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

	var deletedBy *string
	if v, ok := data["deletedBy"].(string); ok && strings.TrimSpace(v) != "" {
		s := strings.TrimSpace(v)
		deletedBy = &s
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
		DeletedAt:          deletedAt,
		DeletedBy:          deletedBy,
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
	if v.DeletedAt != nil {
		m["deletedAt"] = *v.DeletedAt
	}
	if v.DeletedBy != nil {
		m["deletedBy"] = *v.DeletedBy
	}
	return m
}
