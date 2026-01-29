// backend/internal/adapters/out/firestore/productBlueprint_repository_fs.go
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

	pbdom "narratives/internal/domain/productBlueprint"
)

// ProductBlueprintRepositoryFS implements pbdom.Repository using Firestore.
type ProductBlueprintRepositoryFS struct {
	Client *firestore.Client
}

func NewProductBlueprintRepositoryFS(client *firestore.Client) *ProductBlueprintRepositoryFS {
	return &ProductBlueprintRepositoryFS{Client: client}
}

func (r *ProductBlueprintRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("product_blueprints")
}

// history コレクション: product_blueprints_history/{blueprintId}/versions/{version}
func (r *ProductBlueprintRepositoryFS) historyCol(blueprintID string) *firestore.CollectionRef {
	return r.Client.Collection("product_blueprints_history").
		Doc(blueprintID).
		Collection("versions")
}

// Compile-time check: ensure this satisfies domain port
var (
	_ pbdom.Repository = (*ProductBlueprintRepositoryFS)(nil)
)

// ========================
// Core methods (pbdom.Repository)
// ========================

// GetByID returns a single ProductBlueprint by ID.
func (r *ProductBlueprintRepositoryFS) GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return pbdom.ProductBlueprint{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}

	return docToProductBlueprint(snap)
}

// GetBrandIDByID returns brandId only.
func (r *ProductBlueprintRepositoryFS) GetBrandIDByID(ctx context.Context, id string) (string, error) {
	if r.Client == nil {
		return "", errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return "", pbdom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", pbdom.ErrNotFound
		}
		return "", err
	}

	data := snap.Data()
	if data != nil {
		if v, ok := data["brandId"].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v), nil
		}
	}

	pb, err := docToProductBlueprint(snap)
	if err != nil {
		return "", err
	}
	brandID := strings.TrimSpace(pb.BrandID)
	if brandID == "" {
		return "", pbdom.ErrNotFound
	}
	return brandID, nil
}

// GetProductNameByID returns productName only.
func (r *ProductBlueprintRepositoryFS) GetProductNameByID(ctx context.Context, id string) (string, error) {
	if r.Client == nil {
		return "", errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return "", pbdom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", pbdom.ErrNotFound
		}
		return "", err
	}

	data := snap.Data()
	if data != nil {
		if v, ok := data["productName"].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v), nil
		}
	}

	pb, err := docToProductBlueprint(snap)
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(pb.ProductName)
	if name == "" {
		return "", pbdom.ErrNotFound
	}
	return name, nil
}

// GetModelRefsByModelID gets modelRefs (displayOrder included) by modelID (best-effort).
// 実装方針:
// 1) 互換: product_blueprints 側に legacy フィールド "modelId" がある場合はそこから辿る
// 2) models / model_variations 側に productBlueprintId を持っているケース（docID=modelID 想定）で辿る
func (r *ProductBlueprintRepositoryFS) GetModelRefsByModelID(ctx context.Context, modelID string) ([]pbdom.ModelRef, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil, pbdom.ErrNotFound
	}

	// 1) legacy: product_blueprints 側に modelId を持っているケース
	iter := r.col().Where("modelId", "==", modelID).Limit(1).Documents(ctx)
	snap, err := iter.Next()
	iter.Stop()

	if err == nil && snap != nil {
		pb, err2 := docToProductBlueprint(snap)
		if err2 != nil {
			return nil, err2
		}
		// pb.ModelRefs が空でも「見つかった」ので空で返す（呼び出し側で判断）
		return cloneModelRefs(pb.ModelRefs), nil
	}
	if err != nil && err != iterator.Done {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
	}

	// 2) models / model_variations 側に productBlueprintId を持っているケース（docID=modelID 想定）
	collections := []string{"models", "model_variations", "modelVariations"}

	for _, col := range collections {
		doc, err := r.Client.Collection(col).Doc(modelID).Get(ctx)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				continue
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil, err
			}
			continue
		}
		data := doc.Data()
		if data == nil {
			continue
		}
		if v, ok := data["productBlueprintId"]; ok && v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				blueprintID := strings.TrimSpace(s)
				pb, err := r.GetByID(ctx, blueprintID)
				if err != nil {
					return nil, err
				}
				return cloneModelRefs(pb.ModelRefs), nil
			}
		}
	}

	return nil, pbdom.ErrNotFound
}

// GetPatchByID returns patch for mint/read-model usecases (display fields are not filled here).
func (r *ProductBlueprintRepositoryFS) GetPatchByID(ctx context.Context, id string) (pbdom.Patch, error) {
	if r.Client == nil {
		return pbdom.Patch{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return pbdom.Patch{}, pbdom.ErrNotFound
	}

	pb, err := r.GetByID(ctx, id)
	if err != nil {
		return pbdom.Patch{}, err
	}

	name := pb.ProductName
	brandID := pb.BrandID
	itemType := pb.ItemType
	fit := pb.Fit
	material := pb.Material
	weight := pb.Weight
	qa := make([]string, len(pb.QualityAssurance))
	copy(qa, pb.QualityAssurance)
	productIdTag := pb.ProductIdTag
	assigneeID := pb.AssigneeID

	var refsPtr *[]pbdom.ModelRef
	if pb.ModelRefs != nil {
		refs := cloneModelRefs(pb.ModelRefs)
		refsPtr = &refs
	}

	return pbdom.Patch{
		ProductName:      &name,
		BrandID:          &brandID,
		ItemType:         &itemType,
		Fit:              &fit,
		Material:         &material,
		Weight:           &weight,
		QualityAssurance: &qa,
		ProductIdTag:     &productIdTag,
		AssigneeID:       &assigneeID,
		ModelRefs:        refsPtr,
	}, nil
}

// ListIDsByCompany returns blueprint IDs for given companyID.
func (r *ProductBlueprintRepositoryFS) ListIDsByCompany(ctx context.Context, companyID string) ([]string, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, pbdom.ErrInvalidCompanyID
	}

	iter := r.col().
		Where("companyId", "==", companyID).
		Documents(ctx)
	defer iter.Stop()

	var ids []string
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		ids = append(ids, snap.Ref.ID)
	}
	return ids, nil
}

// Exists reports whether a ProductBlueprint with given ID exists.
func (r *ProductBlueprintRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
	if r.Client == nil {
		return false, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}

	_, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Create inserts a new ProductBlueprint (no upsert) from domain CreateInput.
func (r *ProductBlueprintRepositoryFS) Create(ctx context.Context, in pbdom.CreateInput) (pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return pbdom.ProductBlueprint{}, errors.New("firestore client is nil")
	}

	now := time.Now().UTC()

	createdAt := now
	if in.CreatedAt != nil && !in.CreatedAt.IsZero() {
		createdAt = in.CreatedAt.UTC()
	}

	pb, err := pbdom.New(
		"", // ID は Firestore 採番
		in.ProductName,
		in.BrandID,
		in.ItemType,
		in.Fit,
		in.Material,
		in.Weight,
		in.QualityAssurance,
		in.ProductIdTag,
		in.AssigneeID,
		in.CreatedBy,
		createdAt,
		in.CompanyID,
	)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}

	// modelRefs（任意）
	if len(in.ModelRefs) > 0 {
		refs, err := sanitizeModelRefs(in.ModelRefs)
		if err != nil {
			return pbdom.ProductBlueprint{}, err
		}
		pb.ModelRefs = refs
	}

	docRef := r.col().NewDoc()
	pb.ID = docRef.ID

	data, err := productBlueprintToDoc(pb, pb.CreatedAt, pb.UpdatedAt)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}
	data["id"] = pb.ID

	if _, err := docRef.Create(ctx, data); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return pbdom.ProductBlueprint{}, pbdom.ErrConflict
		}
		return pbdom.ProductBlueprint{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}
	return docToProductBlueprint(snap)
}

// Update updates a ProductBlueprint by patch (touches updatedAt; updatedBy is best-effort).
func (r *ProductBlueprintRepositoryFS) Update(ctx context.Context, id string, patch pbdom.Patch) (pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return pbdom.ProductBlueprint{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return pbdom.ProductBlueprint{}, pbdom.ErrInvalidID
	}

	docRef := r.col().Doc(id)

	snap, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}

	// printed guard
	if v, ok := snap.Data()["printed"].(bool); ok && v {
		return pbdom.ProductBlueprint{}, pbdom.ErrForbidden
	}

	pb, err := docToProductBlueprint(snap)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}

	// apply patch
	if patch.ProductName != nil {
		v := strings.TrimSpace(*patch.ProductName)
		if v == "" {
			return pbdom.ProductBlueprint{}, pbdom.ErrInvalidProduct
		}
		pb.ProductName = v
	}
	if patch.BrandID != nil {
		v := strings.TrimSpace(*patch.BrandID)
		if v == "" {
			return pbdom.ProductBlueprint{}, pbdom.ErrInvalidBrand
		}
		pb.BrandID = v
	}
	if patch.CompanyID != nil {
		// NOTE: contract says display-only; do not persist companyId changes via Patch
		// so intentionally ignored
	}
	if patch.ItemType != nil {
		if !pbdom.IsValidItemType(*patch.ItemType) {
			return pbdom.ProductBlueprint{}, pbdom.ErrInvalidItemType
		}
		pb.ItemType = *patch.ItemType
	}
	if patch.Fit != nil {
		pb.Fit = strings.TrimSpace(*patch.Fit)
	}
	if patch.Material != nil {
		pb.Material = strings.TrimSpace(*patch.Material)
	}
	if patch.Weight != nil {
		if *patch.Weight < 0 {
			return pbdom.ProductBlueprint{}, pbdom.ErrInvalidWeight
		}
		pb.Weight = *patch.Weight
	}
	if patch.QualityAssurance != nil {
		pb.QualityAssurance = dedupTrimStrings(*patch.QualityAssurance)
	}
	if patch.ProductIdTag != nil {
		if !pbdom.IsValidTagType(patch.ProductIdTag.Type) {
			return pbdom.ProductBlueprint{}, pbdom.ErrInvalidTagType
		}
		pb.ProductIdTag = *patch.ProductIdTag
	}
	if patch.AssigneeID != nil {
		v := strings.TrimSpace(*patch.AssigneeID)
		if v == "" {
			return pbdom.ProductBlueprint{}, pbdom.ErrInvalidAssignee
		}
		pb.AssigneeID = v
	}
	if patch.ModelRefs != nil {
		refs, err := sanitizeModelRefs(*patch.ModelRefs)
		if err != nil {
			return pbdom.ProductBlueprint{}, err
		}
		pb.ModelRefs = refs
	}

	// touch updatedAt (updatedBy は Patch にないため現状維持)
	pb.UpdatedAt = time.Now().UTC()

	data, err := productBlueprintToDoc(pb, pb.CreatedAt, pb.UpdatedAt)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}
	data["id"] = pb.ID

	if _, err := docRef.Set(ctx, data); err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}

	snap, err = docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}
	return docToProductBlueprint(snap)
}

// Delete removes a ProductBlueprint by ID (physical delete).
func (r *ProductBlueprintRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return pbdom.ErrNotFound
	}

	_, err := r.col().Doc(id).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ErrNotFound
		}
		return err
	}
	return nil
}

// AppendModelRefsWithoutTouch updates modelRefs only, without touching updatedAt/updatedBy.
// 要件:
// - updatedAt / updatedBy を更新しない（このメソッドでは一切書き換えない）
// - modelRefs のみ部分更新する
// - printed=true の場合は更新不可（ドメイン制約に合わせて ErrForbidden）
// 実装:
// - 既存 + 追加入力をマージし、重複排除しつつ displayOrder を 1..N で採番し直す
func (r *ProductBlueprintRepositoryFS) AppendModelRefsWithoutTouch(
	ctx context.Context,
	id string,
	refs []pbdom.ModelRef,
) (pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return pbdom.ProductBlueprint{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return pbdom.ProductBlueprint{}, pbdom.ErrInvalidID
	}

	docRef := r.col().Doc(id)

	// existence + printed guard
	snap, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}
	if v, ok := snap.Data()["printed"].(bool); ok && v {
		return pbdom.ProductBlueprint{}, pbdom.ErrForbidden
	}

	currentPB, err := docToProductBlueprint(snap)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}

	appendRefs, err := sanitizeModelRefs(refs)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}
	if len(appendRefs) == 0 {
		return pbdom.ProductBlueprint{}, pbdom.WrapInvalid(nil, "modelRefs has no valid items")
	}

	merged := mergeAndRenumberModelRefs(currentPB.ModelRefs, appendRefs)

	modelRefsDoc := make([]map[string]any, 0, len(merged))
	for _, mr := range merged {
		modelRefsDoc = append(modelRefsDoc, map[string]any{
			"modelId":      mr.ModelID,
			"displayOrder": mr.DisplayOrder,
		})
	}

	// IMPORTANT: do NOT update updatedAt/updatedBy here
	if _, err := docRef.Update(ctx, []firestore.Update{
		{Path: "modelRefs", Value: modelRefsDoc},
	}); err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}

	// re-fetch
	snap, err = docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}
	return docToProductBlueprint(snap)
}

// MarkPrinted sets printed=true and returns updated blueprint.
func (r *ProductBlueprintRepositoryFS) MarkPrinted(ctx context.Context, id string) (pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return pbdom.ProductBlueprint{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return pbdom.ProductBlueprint{}, pbdom.ErrInvalidID
	}

	docRef := r.col().Doc(id)

	snap, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}

	// printed は bool のみ
	if v, ok := snap.Data()["printed"].(bool); ok && v {
		return pbdom.ProductBlueprint{}, pbdom.ErrForbidden
	}

	now := time.Now().UTC()
	if _, err := docRef.Update(ctx, []firestore.Update{
		{Path: "printed", Value: true},
		{Path: "updatedAt", Value: now},
	}); err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}

	snap, err = docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}

	return docToProductBlueprint(snap)
}

// ========================
// History (snapshot, versioned)
// ========================

func (r *ProductBlueprintRepositoryFS) SaveHistorySnapshot(ctx context.Context, blueprintID string, h pbdom.HistoryRecord) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	blueprintID = strings.TrimSpace(blueprintID)
	if blueprintID == "" {
		return pbdom.ErrInvalidID
	}

	if strings.TrimSpace(h.Blueprint.ID) == "" || h.Blueprint.ID != blueprintID {
		h.Blueprint.ID = blueprintID
	}

	if h.UpdatedAt.IsZero() {
		h.UpdatedAt = h.Blueprint.UpdatedAt
	}
	if h.UpdatedAt.IsZero() {
		h.UpdatedAt = time.Now().UTC()
	}

	docID := fmt.Sprintf("%d", h.Version)
	docRef := r.historyCol(blueprintID).Doc(docID)

	data, err := productBlueprintToDoc(h.Blueprint, h.Blueprint.CreatedAt, h.Blueprint.UpdatedAt)
	if err != nil {
		return err
	}
	data["id"] = blueprintID
	data["version"] = h.Version
	data["historyUpdatedAt"] = h.UpdatedAt.UTC()
	if h.UpdatedBy != nil {
		if s := strings.TrimSpace(*h.UpdatedBy); s != "" {
			data["historyUpdatedBy"] = s
		}
	}

	if _, err := docRef.Set(ctx, data); err != nil {
		return err
	}
	return nil
}

func (r *ProductBlueprintRepositoryFS) ListHistory(ctx context.Context, blueprintID string) ([]pbdom.HistoryRecord, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	blueprintID = strings.TrimSpace(blueprintID)
	if blueprintID == "" {
		return nil, pbdom.ErrInvalidID
	}

	q := r.historyCol(blueprintID).OrderBy("version", firestore.Desc)
	snaps, err := q.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	out := make([]pbdom.HistoryRecord, 0, len(snaps))
	for _, snap := range snaps {
		data := snap.Data()
		if data == nil {
			continue
		}

		pb, err := docToProductBlueprint(snap)
		if err != nil {
			return nil, err
		}

		var version int64
		if v, ok := data["version"]; ok {
			switch x := v.(type) {
			case int64:
				version = x
			case int:
				version = int64(x)
			case float64:
				version = int64(x)
			}
		}

		var histUpdatedAt time.Time
		if v, ok := data["historyUpdatedAt"].(time.Time); ok && !v.IsZero() {
			histUpdatedAt = v.UTC()
		} else {
			histUpdatedAt = pb.UpdatedAt
		}

		var histUpdatedBy *string
		if v, ok := data["historyUpdatedBy"].(string); ok && strings.TrimSpace(v) != "" {
			s := strings.TrimSpace(v)
			histUpdatedBy = &s
		} else {
			histUpdatedBy = pb.UpdatedBy
		}

		out = append(out, pbdom.HistoryRecord{
			Blueprint: pb,
			Version:   version,
			UpdatedAt: histUpdatedAt,
			UpdatedBy: histUpdatedBy,
		})
	}
	return out, nil
}

func (r *ProductBlueprintRepositoryFS) GetHistoryByVersion(ctx context.Context, blueprintID string, version int64) (pbdom.HistoryRecord, error) {
	if r.Client == nil {
		return pbdom.HistoryRecord{}, errors.New("firestore client is nil")
	}

	blueprintID = strings.TrimSpace(blueprintID)
	if blueprintID == "" {
		return pbdom.HistoryRecord{}, pbdom.ErrInvalidID
	}

	docID := fmt.Sprintf("%d", version)
	snap, err := r.historyCol(blueprintID).Doc(docID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.HistoryRecord{}, pbdom.ErrNotFound
		}
		return pbdom.HistoryRecord{}, err
	}

	data := snap.Data()
	if data == nil {
		return pbdom.HistoryRecord{}, fmt.Errorf("empty history document: %s", snap.Ref.Path)
	}

	pb, err := docToProductBlueprint(snap)
	if err != nil {
		return pbdom.HistoryRecord{}, err
	}

	var ver int64
	if v, ok := data["version"]; ok {
		switch x := v.(type) {
		case int64:
			ver = x
		case int:
			ver = int64(x)
		case float64:
			ver = int64(x)
		}
	}
	if ver == 0 {
		ver = version
	}

	var histUpdatedAt time.Time
	if v, ok := data["historyUpdatedAt"].(time.Time); ok && !v.IsZero() {
		histUpdatedAt = v.UTC()
	} else {
		histUpdatedAt = pb.UpdatedAt
	}

	var histUpdatedBy *string
	if v, ok := data["historyUpdatedBy"].(string); ok && strings.TrimSpace(v) != "" {
		s := strings.TrimSpace(v)
		histUpdatedBy = &s
	} else {
		histUpdatedBy = pb.UpdatedBy
	}

	return pbdom.HistoryRecord{
		Blueprint: pb,
		Version:   ver,
		UpdatedAt: histUpdatedAt,
		UpdatedBy: histUpdatedBy,
	}, nil
}

// ========================
// Helpers
// ========================

func docToProductBlueprint(doc *firestore.DocumentSnapshot) (pbdom.ProductBlueprint, error) {
	data := doc.Data()
	if data == nil {
		return pbdom.ProductBlueprint{}, fmt.Errorf("empty product_blueprints document: %s", doc.Ref.ID)
	}

	getStr := func(key string) string {
		if v, ok := data[key].(string); ok {
			return strings.TrimSpace(v)
		}
		return ""
	}
	getStrPtr := func(key string) *string {
		if v, ok := data[key].(string); ok {
			s := strings.TrimSpace(v)
			if s != "" {
				return &s
			}
		}
		return nil
	}
	getTimeVal := func(key string) time.Time {
		if v, ok := data[key].(time.Time); ok && !v.IsZero() {
			return v.UTC()
		}
		return time.Time{}
	}
	getStringSlice := func(key string) []string {
		raw, ok := data[key]
		if !ok || raw == nil {
			return nil
		}
		switch vv := raw.(type) {
		case []interface{}:
			out := make([]string, 0, len(vv))
			for _, x := range vv {
				if s, ok := x.(string); ok {
					s = strings.TrimSpace(s)
					if s != "" {
						out = append(out, s)
					}
				}
			}
			return dedupTrimStrings(out)
		case []string:
			return dedupTrimStrings(vv)
		default:
			return nil
		}
	}

	// printed は bool のみ
	printed := false
	if v, ok := data["printed"].(bool); ok {
		printed = v
	}

	// modelRefs
	var modelRefs []pbdom.ModelRef
	if raw, ok := data["modelRefs"]; ok && raw != nil {
		switch xs := raw.(type) {
		case []interface{}:
			tmp := make([]pbdom.ModelRef, 0, len(xs))
			for _, it := range xs {
				m, ok := it.(map[string]interface{})
				if !ok || m == nil {
					continue
				}
				mid, _ := m["modelId"].(string)

				order := 0
				switch v := m["displayOrder"].(type) {
				case int:
					order = v
				case int32:
					order = int(v)
				case int64:
					order = int(v)
				case float64:
					order = int(v)
				}

				mid = strings.TrimSpace(mid)
				if mid == "" || order <= 0 {
					continue
				}
				tmp = append(tmp, pbdom.ModelRef{
					ModelID:      mid,
					DisplayOrder: order,
				})
			}
			if len(tmp) > 0 {
				// 安全のため displayOrder 順に並べ替え
				sort.SliceStable(tmp, func(i, j int) bool {
					return tmp[i].DisplayOrder < tmp[j].DisplayOrder
				})
				modelRefs = tmp
			}
		}
	}

	var deletedAtPtr *time.Time
	if t := getTimeVal("deletedAt"); !t.IsZero() {
		deletedAtPtr = &t
	}

	var expireAtPtr *time.Time
	if t := getTimeVal("expireAt"); !t.IsZero() {
		expireAtPtr = &t
	}

	id := ""
	if v, ok := data["id"].(string); ok && strings.TrimSpace(v) != "" {
		id = strings.TrimSpace(v)
	} else {
		id = doc.Ref.ID
	}

	pb := pbdom.ProductBlueprint{
		ID:          id,
		ProductName: getStr("productName"),
		BrandID:     getStr("brandId"),
		ItemType:    pbdom.ItemType(getStr("itemType")),
		Fit:         getStr("fit"),
		Material:    getStr("material"),
		Weight:      getFloat64(data["weight"]),

		QualityAssurance: dedupTrimStrings(getStringSlice("qualityAssurance")),
		ProductIdTag: pbdom.ProductIDTag{
			Type: pbdom.ProductIDTagType(getStr("productIdTagType")),
		},
		CompanyID:  getStr("companyId"),
		AssigneeID: getStr("assigneeId"),

		ModelRefs: modelRefs,

		Printed: printed,

		CreatedBy: getStrPtr("createdBy"),
		CreatedAt: getTimeVal("createdAt"),
		UpdatedBy: getStrPtr("updatedBy"),
		UpdatedAt: getTimeVal("updatedAt"),
		DeletedBy: getStrPtr("deletedBy"),
		DeletedAt: deletedAtPtr,
		ExpireAt:  expireAtPtr,
	}

	return pb, nil
}

func productBlueprintToDoc(v pbdom.ProductBlueprint, createdAt, updatedAt time.Time) (map[string]any, error) {
	m := map[string]any{
		"productName": strings.TrimSpace(v.ProductName),
		"brandId":     strings.TrimSpace(v.BrandID),
		"itemType":    strings.TrimSpace(string(v.ItemType)),
		"fit":         strings.TrimSpace(v.Fit),
		"material":    strings.TrimSpace(v.Material),
		"weight":      v.Weight,
		"assigneeId":  strings.TrimSpace(v.AssigneeID),
		"companyId":   strings.TrimSpace(v.CompanyID),
		"createdAt":   createdAt.UTC(),
		"updatedAt":   updatedAt.UTC(),
		"printed":     v.Printed,
	}

	if len(v.QualityAssurance) > 0 {
		m["qualityAssurance"] = dedupTrimStrings(v.QualityAssurance)
	}

	if v.ProductIdTag.Type != "" {
		m["productIdTagType"] = strings.TrimSpace(string(v.ProductIdTag.Type))
	}

	// modelRefs（nil の場合は「未指定」として保存しない。空スライスで明示したい場合は empty を渡す）
	if v.ModelRefs != nil {
		arr := make([]map[string]any, 0, len(v.ModelRefs))
		for _, mr := range v.ModelRefs {
			mid := strings.TrimSpace(mr.ModelID)
			if mid == "" || mr.DisplayOrder <= 0 {
				continue
			}
			arr = append(arr, map[string]any{
				"modelId":      mid,
				"displayOrder": mr.DisplayOrder,
			})
		}
		m["modelRefs"] = arr
	}

	if v.CreatedBy != nil {
		if s := strings.TrimSpace(*v.CreatedBy); s != "" {
			m["createdBy"] = s
		}
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
	if v.ExpireAt != nil && !v.ExpireAt.IsZero() {
		m["expireAt"] = v.ExpireAt.UTC()
	}

	return m, nil
}

func sanitizeModelRefs(in []pbdom.ModelRef) ([]pbdom.ModelRef, error) {
	// displayOrder 順で安定化し、modelId の重複は先勝ちで除外、最後に 1..N で再採番
	tmp := make([]pbdom.ModelRef, 0, len(in))
	seen := make(map[string]struct{}, len(in))

	// まずは入力を displayOrder で安定ソート（同順位は入力順）
	withIdx := make([]struct {
		ref pbdom.ModelRef
		idx int
	}, 0, len(in))
	for i, r := range in {
		withIdx = append(withIdx, struct {
			ref pbdom.ModelRef
			idx int
		}{ref: r, idx: i})
	}
	sort.SliceStable(withIdx, func(i, j int) bool {
		ri, rj := withIdx[i].ref, withIdx[j].ref
		if ri.DisplayOrder == rj.DisplayOrder {
			return withIdx[i].idx < withIdx[j].idx
		}
		return ri.DisplayOrder < rj.DisplayOrder
	})

	ids := make([]string, 0, len(in))
	for _, w := range withIdx {
		mid := strings.TrimSpace(w.ref.ModelID)
		if mid == "" {
			continue
		}
		if _, ok := seen[mid]; ok {
			continue
		}
		seen[mid] = struct{}{}
		ids = append(ids, mid)
	}

	// 1..N で再採番
	for i, mid := range ids {
		tmp = append(tmp, pbdom.ModelRef{
			ModelID:      mid,
			DisplayOrder: i + 1,
		})
	}
	return tmp, nil
}

func mergeAndRenumberModelRefs(existing []pbdom.ModelRef, appendRefs []pbdom.ModelRef) []pbdom.ModelRef {
	seen := make(map[string]struct{}, len(existing)+len(appendRefs))
	ids := make([]string, 0, len(existing)+len(appendRefs))

	// existing は displayOrder で安定化してから取り込む
	ex := cloneModelRefs(existing)
	sort.SliceStable(ex, func(i, j int) bool { return ex[i].DisplayOrder < ex[j].DisplayOrder })

	for _, r := range ex {
		mid := strings.TrimSpace(r.ModelID)
		if mid == "" {
			continue
		}
		if _, ok := seen[mid]; ok {
			continue
		}
		seen[mid] = struct{}{}
		ids = append(ids, mid)
	}

	// appendRefs は displayOrder で安定化してから末尾に追加
	ap := cloneModelRefs(appendRefs)
	sort.SliceStable(ap, func(i, j int) bool { return ap[i].DisplayOrder < ap[j].DisplayOrder })

	for _, r := range ap {
		mid := strings.TrimSpace(r.ModelID)
		if mid == "" {
			continue
		}
		if _, ok := seen[mid]; ok {
			continue
		}
		seen[mid] = struct{}{}
		ids = append(ids, mid)
	}

	out := make([]pbdom.ModelRef, 0, len(ids))
	for i, mid := range ids {
		out = append(out, pbdom.ModelRef{
			ModelID:      mid,
			DisplayOrder: i + 1,
		})
	}
	return out
}

func cloneModelRefs(in []pbdom.ModelRef) []pbdom.ModelRef {
	if in == nil {
		return nil
	}
	out := make([]pbdom.ModelRef, len(in))
	copy(out, in)
	return out
}

func getFloat64(v any) float64 {
	switch x := v.(type) {
	case int:
		return float64(x)
	case int32:
		return float64(x)
	case int64:
		return float64(x)
	case float32:
		return float64(x)
	case float64:
		return x
	default:
		return 0
	}
}

func dedupTrimStrings(xs []string) []string {
	seen := make(map[string]struct{}, len(xs))
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		x = strings.TrimSpace(x)
		if x == "" {
			continue
		}
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	return out
}

// ListByCompanyID returns non-deleted ProductBlueprints for the given companyID.
// NOTE:
//   - Firestore で deletedAt==nil を厳密に拾うのはフィールド未設定問題があるため、
//     companyId で取得してから deletedAt を in-memory で除外する。
func (r *ProductBlueprintRepositoryFS) ListByCompanyID(
	ctx context.Context,
	companyID string,
) ([]pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, pbdom.ErrInvalidCompanyID
	}

	iter := r.col().
		Where("companyId", "==", companyID).
		Documents(ctx)
	defer iter.Stop()

	out := make([]pbdom.ProductBlueprint, 0, 64)
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		pb, err := docToProductBlueprint(snap)
		if err != nil {
			return nil, err
		}

		// deleted は除外（live list）
		if pb.DeletedAt != nil && !pb.DeletedAt.IsZero() {
			continue
		}

		out = append(out, pb)
	}

	return out, nil
}

// ListDeletedByCompanyID returns only logically deleted ProductBlueprints for the given companyID.
func (r *ProductBlueprintRepositoryFS) ListDeletedByCompanyID(
	ctx context.Context,
	companyID string,
) ([]pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, pbdom.ErrInvalidCompanyID
	}

	q := r.col().Query.
		Where("companyId", "==", companyID).
		Where("deletedAt", ">", time.Time{})

	snaps, err := q.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	out := make([]pbdom.ProductBlueprint, 0, len(snaps))
	for _, snap := range snaps {
		pb, err := docToProductBlueprint(snap)
		if err != nil {
			return nil, err
		}
		out = append(out, pb)
	}
	return out, nil
}

// Save upserts a ProductBlueprint (Set).
// - printed=true の場合は更新不可（ErrForbidden）
// - updatedAt は now に更新（updatedBy は v.UpdatedBy を保存する）
// - companyId 境界の検証は usecase 側で担保されている前提（必要ならここで追加可能）
func (r *ProductBlueprintRepositoryFS) Save(ctx context.Context, v pbdom.ProductBlueprint) (pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return pbdom.ProductBlueprint{}, errors.New("firestore client is nil")
	}

	id := strings.TrimSpace(v.ID)
	if id == "" {
		return pbdom.ProductBlueprint{}, pbdom.ErrInvalidID
	}

	docRef := r.col().Doc(id)

	// existence + printed guard
	snap, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}
	if vv, ok := snap.Data()["printed"].(bool); ok && vv {
		return pbdom.ProductBlueprint{}, pbdom.ErrForbidden
	}

	// createdAt は維持（入力がゼロなら既存から補完）
	now := time.Now().UTC()

	if v.CreatedAt.IsZero() {
		// 既存 createdAt を優先
		if t, ok := snap.Data()["createdAt"].(time.Time); ok && !t.IsZero() {
			v.CreatedAt = t.UTC()
		} else {
			v.CreatedAt = now
		}
	} else {
		v.CreatedAt = v.CreatedAt.UTC()
	}

	v.UpdatedAt = now

	// doc 化
	data, err := productBlueprintToDoc(v, v.CreatedAt, v.UpdatedAt)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}
	data["id"] = id

	// upsert
	if _, err := docRef.Set(ctx, data); err != nil {
		return pbdom.ProductBlueprint{}, err
	}

	// re-fetch
	snap, err = docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}
	return docToProductBlueprint(snap)
}
