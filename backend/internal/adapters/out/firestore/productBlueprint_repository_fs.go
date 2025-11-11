// backend/internal/adapters/out/firestore/productBlueprint_repository_fs.go
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
	usecase "narratives/internal/application/usecase"
	modeldom "narratives/internal/domain/model"
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
// Additional query helpers (still available, not required by usecase interface)
// ========================

// List applies filter/sort/paging using Firestore query + in-memory filtering.
func (r *ProductBlueprintRepositoryFS) List(
	ctx context.Context,
	filter pbdom.Filter,
	sort pbdom.Sort,
	page pbdom.Page,
) (pbdom.PageResult, error) {
	if r.Client == nil {
		return pbdom.PageResult{}, errors.New("firestore client is nil")
	}

	pageNum, perPage, offset := fscommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	q := r.col().Query
	q = applyPBOrderBy(q, sort)

	it := q.Documents(ctx)
	defer it.Stop()

	var all []pbdom.ProductBlueprint
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return pbdom.PageResult{}, err
		}
		pb, err := docToProductBlueprint(doc)
		if err != nil {
			return pbdom.PageResult{}, err
		}
		if matchPBFilter(pb, filter) {
			all = append(all, pb)
		}
	}

	total := len(all)
	if total == 0 {
		return pbdom.PageResult{
			Items:      []pbdom.ProductBlueprint{},
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

	return pbdom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: fscommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

// Count returns number of ProductBlueprints matching the filter.
func (r *ProductBlueprintRepositoryFS) Count(ctx context.Context, filter pbdom.Filter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
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
		pb, err := docToProductBlueprint(doc)
		if err != nil {
			return 0, err
		}
		if matchPBFilter(pb, filter) {
			total++
		}
	}
	return total, nil
}

// Reset is mainly for tests.
func (r *ProductBlueprintRepositoryFS) Reset(ctx context.Context) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	var refs []*firestore.DocumentRef
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		refs = append(refs, doc.Ref)
	}

	if len(refs) == 0 {
		return nil
	}

	const chunkSize = 400
	for start := 0; start < len(refs); start += chunkSize {
		end := start + chunkSize
		if end > len(refs) {
			end = len(refs)
		}
		chunk := refs[start:end]

		if err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			for _, ref := range chunk {
				if err := tx.Delete(ref); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
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
	getVariations := func() []modeldom.ModelVariation {
		raw, ok := data["modelVariations"]
		if !ok || raw == nil {
			raw, ok = data["model_variations"]
			if !ok || raw == nil {
				return nil
			}
		}
		switch vv := raw.(type) {
		case []interface{}, map[string]interface{}:
			b, err := json.Marshal(vv)
			if err != nil {
				return nil
			}
			var out []modeldom.ModelVariation
			if err := json.Unmarshal(b, &out); err != nil {
				return nil
			}
			return out
		case string:
			if strings.TrimSpace(vv) == "" {
				return nil
			}
			var out []modeldom.ModelVariation
			if err := json.Unmarshal([]byte(vv), &out); err != nil {
				return nil
			}
			return out
		default:
			return nil
		}
	}

	qas := getStringSlice("qualityAssurance", "quality_assurance")
	tagTypeStr := getStr("productIdTagType", "product_id_tag_type")
	itemTypeStr := getStr("itemType", "item_type")

	pb := pbdom.ProductBlueprint{
		ID:               doc.Ref.ID,
		ProductName:      getStr("productName", "product_name"),
		BrandID:          getStr("brandId", "brand_id"),
		ItemType:         pbdom.ItemType(itemTypeStr),
		Variations:       getVariations(),
		Fit:              getStr("fit"),
		Material:         getStr("material"),
		Weight:           getFloat64(data["weight"]),
		QualityAssurance: dedupTrimStrings(qas),
		ProductIdTag: pbdom.ProductIDTag{
			Type: pbdom.ProductIDTagType(tagTypeStr),
		},
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
		"createdAt":   createdAt.UTC(),
		"updatedAt":   updatedAt.UTC(),
	}

	// Variations: store as modelVariations if present; keep as structured JSON.
	if v.Variations != nil {
		b, err := json.Marshal(v.Variations)
		if err != nil {
			return nil, err
		}
		var generic any
		if err := json.Unmarshal(b, &generic); err != nil {
			return nil, err
		}
		m["modelVariations"] = generic
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

// matchPBFilter applies pbdom.Filter in-memory.
func matchPBFilter(pb pbdom.ProductBlueprint, f pbdom.Filter) bool {
	// SearchTerm on product name (and brand)
	if s := strings.TrimSpace(f.SearchTerm); s != "" {
		ls := strings.ToLower(s)
		hay := strings.ToLower(pb.ProductName + " " + pb.BrandID)
		if !strings.Contains(hay, ls) {
			return false
		}
	}

	// BrandIDs
	if len(f.BrandIDs) > 0 && !containsStr(f.BrandIDs, pb.BrandID) {
		return false
	}

	// AssigneeIDs
	if len(f.AssigneeIDs) > 0 && !containsStr(f.AssigneeIDs, pb.AssigneeID) {
		return false
	}

	// ItemTypes
	if len(f.ItemTypes) > 0 {
		ok := false
		for _, it := range f.ItemTypes {
			if strings.TrimSpace(string(it)) == strings.TrimSpace(string(pb.ItemType)) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	// TagTypes
	if len(f.TagTypes) > 0 {
		ok := false
		for _, tt := range f.TagTypes {
			if strings.TrimSpace(string(tt)) == strings.TrimSpace(string(pb.ProductIdTag.Type)) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	// VariationIDs
	if len(f.VariationIDs) > 0 {
		want := make(map[string]struct{}, len(f.VariationIDs))
		for _, vid := range f.VariationIDs {
			if s := strings.TrimSpace(vid); s != "" {
				want[s] = struct{}{}
			}
		}
		if len(want) > 0 {
			found := false
			for _, v := range pb.Variations {
				if _, ok := want[strings.TrimSpace(v.ID)]; ok {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	// CreatedAt / UpdatedAt ranges
	if f.CreatedFrom != nil {
		if pb.CreatedAt.IsZero() || pb.CreatedAt.Before(f.CreatedFrom.UTC()) {
			return false
		}
	}
	if f.CreatedTo != nil {
		if pb.CreatedAt.IsZero() || !pb.CreatedAt.Before(f.CreatedTo.UTC()) {
			return false
		}
	}
	if f.UpdatedFrom != nil {
		if pb.UpdatedAt.IsZero() || pb.UpdatedAt.Before(f.UpdatedFrom.UTC()) {
			return false
		}
	}
	if f.UpdatedTo != nil {
		if pb.UpdatedAt.IsZero() || !pb.UpdatedAt.Before(f.UpdatedTo.UTC()) {
			return false
		}
	}

	return true
}

func containsStr(xs []string, v string) bool {
	v = strings.TrimSpace(v)
	for _, x := range xs {
		if strings.TrimSpace(x) == v {
			return true
		}
	}
	return false
}

// applyPBOrderBy maps pbdom.Sort to Firestore orderBy.
func applyPBOrderBy(q firestore.Query, s pbdom.Sort) firestore.Query {
	col := strings.ToLower(strings.TrimSpace(string(s.Column)))
	var field string

	switch col {
	case "createdat", "created_at":
		field = "createdAt"
	case "updatedat", "updated_at":
		field = "updatedAt"
	case "productname", "product_name":
		field = "productName"
	case "brandid", "brand_id":
		field = "brandId"
	default:
		// default: updatedAt DESC, then id
		return q.OrderBy("updatedAt", firestore.Desc).
			OrderBy(firestore.DocumentID, firestore.Desc)
	}

	dir := firestore.Desc
	if strings.EqualFold(string(s.Order), "asc") {
		dir = firestore.Asc
	}

	return q.OrderBy(field, dir).
		OrderBy(firestore.DocumentID, dir)
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
