// backend/internal/adapters/out/firestore/productBlueprint_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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

// Compile-time check: ensure this satisfies usecase.ProductBlueprintRepo.
var _ usecase.ProductBlueprintRepo = (*ProductBlueprintRepositoryFS)(nil)

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

// List returns all ProductBlueprints (optionally filtered by companyId in context).
func (r *ProductBlueprintRepositoryFS) List(ctx context.Context) ([]pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	q := r.col().Query

	// usecase.CompanyIDFromContext で context から companyId を取得し、
	// 指定があればテナント単位で絞り込む
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

// Create inserts a new ProductBlueprint (no upsert).
// If ID is empty, it is auto-generated.
// If CreatedAt/UpdatedAt are zero, they are set to now (UTC).
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
// If ID is empty, a new one is generated.
// If CreatedAt is zero, it is set to now (UTC).
// UpdatedAt is always set to now (UTC) when saving.
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

	if _, err := docRef.Set(ctx, data, firestore.MergeAll); err != nil {
		return pbdom.ProductBlueprint{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}
	return docToProductBlueprint(snap)
}

// Delete removes a ProductBlueprint by ID.
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

// ========================
// Helpers
// ========================

func docToProductBlueprint(doc *firestore.DocumentSnapshot) (pbdom.ProductBlueprint, error) {
	data := doc.Data()
	if data == nil {
		return pbdom.ProductBlueprint{}, fmt.Errorf("empty product_blueprints document: %s", doc.Ref.ID)
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
	getTimeVal := func(keys ...string) time.Time {
		for _, k := range keys {
			if v, ok := data[k].(time.Time); ok && !v.IsZero() {
				return v.UTC()
			}
		}
		return time.Time{}
	}

	getStringSlice := func(keys ...string) []string {
		for _, key := range keys {
			raw, ok := data[key]
			if !ok || raw == nil {
				continue
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
			}
		}
		return nil
	}

	qas := getStringSlice("qualityAssurance", "quality_assurance")
	tagTypeStr := getStr("productIdTagType", "product_id_tag_type")
	itemTypeStr := getStr("itemType", "item_type")

	pb := pbdom.ProductBlueprint{
		ID:               doc.Ref.ID,
		ProductName:      getStr("productName", "product_name"),
		BrandID:          getStr("brandId", "brand_id"),
		ItemType:         pbdom.ItemType(itemTypeStr),
		Fit:              getStr("fit"),
		Material:         getStr("material"),
		Weight:           getFloat64(data["weight"]),
		QualityAssurance: dedupTrimStrings(qas),
		ProductIdTag: pbdom.ProductIDTag{
			Type: pbdom.ProductIDTagType(tagTypeStr),
		},
		CompanyID:  getStr("companyId", "company_id"),
		AssigneeID: getStr("assigneeId", "assignee_id"),
		CreatedBy:  getStrPtr("createdBy", "created_by"),
		CreatedAt:  getTimeVal("createdAt", "created_at"),
		UpdatedBy:  getStrPtr("updatedBy", "updated_by"),
		UpdatedAt:  getTimeVal("updatedAt", "updated_at"),
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
