// backend/internal/adapters/out/firestore/productBlueprint_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	// ✅ ProductBlueprint usecase に紐づく port 定義の移動先
	pbuc "narratives/internal/application/productBlueprint/usecase"

	// ✅ ここは CompanyIDFromContext を使っているため残す
	usecase "narratives/internal/application/usecase"

	pbdom "narratives/internal/domain/productBlueprint"
)

// ProductBlueprintRepositoryFS implements ProductBlueprintRepo using Firestore.
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

// Compile-time check: ensure this satisfies pbuc.ProductBlueprintRepo
// および pbuc.ProductBlueprintPrintedRepo.
var (
	_ pbuc.ProductBlueprintRepo = (*ProductBlueprintRepositoryFS)(nil)
)

// ========================
// Core methods (ProductBlueprintRepo)
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

// GetIDByModelID gets productBlueprintId by modelID (best-effort).
func (r *ProductBlueprintRepositoryFS) GetIDByModelID(ctx context.Context, modelID string) (string, error) {
	if r.Client == nil {
		return "", errors.New("firestore client is nil")
	}

	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return "", pbdom.ErrNotFound
	}

	// product_blueprints 側に modelId を持っているケース
	iter := r.col().Where("modelId", "==", modelID).Limit(1).Documents(ctx)
	snap, err := iter.Next()
	iter.Stop()

	if err == nil && snap != nil {
		return strings.TrimSpace(snap.Ref.ID), nil
	}
	if err != nil && err != iterator.Done {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return "", err
		}
	}

	// models / model_variations 側に productBlueprintId を持っているケース（docID=modelID 想定）
	collections := []string{"models", "model_variations", "modelVariations"}

	for _, col := range collections {
		doc, err := r.Client.Collection(col).Doc(modelID).Get(ctx)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				continue
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return "", err
			}
			continue
		}
		data := doc.Data()
		if data == nil {
			continue
		}
		if v, ok := data["productBlueprintId"]; ok && v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s), nil
			}
		}
	}

	return "", pbdom.ErrNotFound
}

// GetProductNameByID returns productName only.
func (r *ProductBlueprintRepositoryFS) GetProductNameByID(
	ctx context.Context,
	id string,
) (string, error) {
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

// GetPatchByID returns patch for mint/read-model usecases.
func (r *ProductBlueprintRepositoryFS) GetPatchByID(
	ctx context.Context,
	id string,
) (pbdom.Patch, error) {
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
	}, nil
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

// ListIDsByCompany returns blueprint IDs for given companyID.
func (r *ProductBlueprintRepositoryFS) ListIDsByCompany(
	ctx context.Context,
	companyID string,
) ([]string, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return []string{}, nil
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

// ListPrinted returns printed==true blueprints from ids.
func (r *ProductBlueprintRepositoryFS) ListPrinted(
	ctx context.Context,
	ids []string,
) ([]pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	uniq := make(map[string]struct{}, len(ids))
	cleaned := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := uniq[id]; ok {
			continue
		}
		uniq[id] = struct{}{}
		cleaned = append(cleaned, id)
	}
	if len(cleaned) == 0 {
		return []pbdom.ProductBlueprint{}, nil
	}

	out := make([]pbdom.ProductBlueprint, 0, len(cleaned))
	for _, id := range cleaned {
		snap, err := r.col().Doc(id).Get(ctx)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				continue
			}
			return nil, err
		}
		pb, err := docToProductBlueprint(snap)
		if err != nil {
			return nil, err
		}
		if pb.Printed {
			out = append(out, pb)
		}
	}
	return out, nil
}

// MarkPrinted sets printed=true and returns updated blueprint.
func (r *ProductBlueprintRepositoryFS) MarkPrinted(
	ctx context.Context,
	id string,
) (pbdom.ProductBlueprint, error) {
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

	data := snap.Data()
	if data != nil {
		if v, ok := data["printed"].(bool); ok && v {
			return pbdom.ProductBlueprint{}, pbdom.ErrForbidden
		}
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

// List returns all ProductBlueprints, optionally filtered by companyId in context.
func (r *ProductBlueprintRepositoryFS) List(ctx context.Context) ([]pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	q := r.col().Query

	if cid := strings.TrimSpace(usecase.CompanyIDFromContext(ctx)); cid != "" {
		q = q.Where("companyId", "==", cid)
	}

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

// ListDeleted returns only logically deleted blueprints (deletedAt != null), optionally filtered by companyId in context.
func (r *ProductBlueprintRepositoryFS) ListDeleted(ctx context.Context) ([]pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	q := r.col().Query

	if cid := strings.TrimSpace(usecase.CompanyIDFromContext(ctx)); cid != "" {
		q = q.Where("companyId", "==", cid)
	}

	q = q.Where("deletedAt", ">", time.Time{})

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

// Create inserts a new ProductBlueprint (no upsert).
func (r *ProductBlueprintRepositoryFS) Create(
	ctx context.Context,
	pb pbdom.ProductBlueprint,
) (pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return pbdom.ProductBlueprint{}, errors.New("firestore client is nil")
	}

	now := time.Now().UTC()

	id := strings.TrimSpace(pb.ID)
	var docRef *firestore.DocumentRef
	if id == "" {
		docRef = r.col().NewDoc()
		pb.ID = docRef.ID
	} else {
		docRef = r.col().Doc(id)
	}

	if pb.CreatedAt.IsZero() {
		pb.CreatedAt = now
	} else {
		pb.CreatedAt = pb.CreatedAt.UTC()
	}
	if pb.UpdatedAt.IsZero() {
		pb.UpdatedAt = now
	} else {
		pb.UpdatedAt = pb.UpdatedAt.UTC()
	}

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

// Save upserts a ProductBlueprint.
func (r *ProductBlueprintRepositoryFS) Save(
	ctx context.Context,
	pb pbdom.ProductBlueprint,
) (pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return pbdom.ProductBlueprint{}, errors.New("firestore client is nil")
	}

	now := time.Now().UTC()

	id := strings.TrimSpace(pb.ID)
	var docRef *firestore.DocumentRef
	if id == "" {
		docRef = r.col().NewDoc()
		pb.ID = docRef.ID
	} else {
		docRef = r.col().Doc(id)
	}

	if pb.CreatedAt.IsZero() {
		pb.CreatedAt = now
	} else {
		pb.CreatedAt = pb.CreatedAt.UTC()
	}
	pb.UpdatedAt = now

	data, err := productBlueprintToDoc(pb, pb.CreatedAt, pb.UpdatedAt)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}
	data["id"] = pb.ID

	if _, err := docRef.Set(ctx, data); err != nil {
		return pbdom.ProductBlueprint{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
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

// SoftDeleteWithModels sets deletedAt only.
func (r *ProductBlueprintRepositoryFS) SoftDeleteWithModels(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return pbdom.ErrInvalidID
	}

	now := time.Now().UTC()

	pbRef := r.col().Doc(id)
	if _, err := pbRef.Get(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ErrNotFound
		}
		return err
	}

	if _, err := pbRef.Update(ctx, []firestore.Update{
		{Path: "deletedAt", Value: now},
	}); err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ErrNotFound
		}
		return err
	}

	return nil
}

// RestoreWithModels clears deletedAt/deletedBy.
func (r *ProductBlueprintRepositoryFS) RestoreWithModels(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return pbdom.ErrInvalidID
	}

	pbRef := r.col().Doc(id)
	if _, err := pbRef.Get(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ErrNotFound
		}
		return err
	}

	if _, err := pbRef.Update(ctx, []firestore.Update{
		{Path: "deletedAt", Value: firestore.Delete},
		{Path: "deletedBy", Value: firestore.Delete},
	}); err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ErrNotFound
		}
		return err
	}
	return nil
}

// ========================
// History (snapshot, versioned)
// ========================

func (r *ProductBlueprintRepositoryFS) SaveHistorySnapshot(
	ctx context.Context,
	blueprintID string,
	h pbdom.HistoryRecord,
) error {
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

func (r *ProductBlueprintRepositoryFS) ListHistory(
	ctx context.Context,
	blueprintID string,
) ([]pbdom.HistoryRecord, error) {
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

func (r *ProductBlueprintRepositoryFS) GetHistoryByVersion(
	ctx context.Context,
	blueprintID string,
	version int64,
) (pbdom.HistoryRecord, error) {
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
