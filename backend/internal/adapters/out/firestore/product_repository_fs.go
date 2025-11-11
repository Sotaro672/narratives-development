// backend/internal/adapters/out/firestore/product_repository_fs.go
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

	fscommon "narratives/internal/adapters/out/firestore/common"
	productdom "narratives/internal/domain/product"
)

// ProductRepositoryFS is a Firestore-based implementation of the product repository.
type ProductRepositoryFS struct {
	Client *firestore.Client
}

func NewProductRepositoryFS(client *firestore.Client) *ProductRepositoryFS {
	return &ProductRepositoryFS{Client: client}
}

func (r *ProductRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("products")
}

// ============================================================
// ProductRepo interface methods
// ============================================================

// GetByID returns a single Product by ID (value return, not pointer)
func (r *ProductRepositoryFS) GetByID(ctx context.Context, id string) (productdom.Product, error) {
	if r.Client == nil {
		return productdom.Product{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return productdom.Product{}, productdom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return productdom.Product{}, productdom.ErrNotFound
		}
		return productdom.Product{}, err
	}

	p, err := docToProduct(snap)
	if err != nil {
		return productdom.Product{}, err
	}
	return p, nil
}

// Exists checks if a product with the given ID exists
func (r *ProductRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
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

// Create inserts a new product and returns the created entity.
//
// Semantics aligned with PG version:
// - If v.ID is empty, Firestore auto-ID is used.
// - updatedAt is set to now (if zero).
// - updatedBy is taken from v.UpdatedBy.
// - inspection_result, connected_token, printed_at/by, inspected_at/by can be nil.
func (r *ProductRepositoryFS) Create(ctx context.Context, v productdom.Product) (productdom.Product, error) {
	if r.Client == nil {
		return productdom.Product{}, errors.New("firestore client is nil")
	}

	now := time.Now().UTC()
	if v.UpdatedAt.IsZero() {
		v.UpdatedAt = now
	}

	id := strings.TrimSpace(v.ID)
	var docRef *firestore.DocumentRef
	if id == "" {
		docRef = r.col().NewDoc()
		v.ID = docRef.ID
	} else {
		docRef = r.col().Doc(id)
		v.ID = id
	}

	data := productToDoc(v, now)

	_, err := docRef.Create(ctx, data)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return productdom.Product{}, productdom.ErrConflict
		}
		return productdom.Product{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return productdom.Product{}, err
	}
	out, err := docToProduct(snap)
	if err != nil {
		return productdom.Product{}, err
	}
	return out, nil
}

// Save performs upsert-style write of Product by ID.
//
// Semantics aligned with PG version:
// - If ID is empty -> behaves like Create (auto-ID).
// - If ID exists -> fields overwritten, updatedAt set to now.
func (r *ProductRepositoryFS) Save(ctx context.Context, v productdom.Product) (productdom.Product, error) {
	if r.Client == nil {
		return productdom.Product{}, errors.New("firestore client is nil")
	}

	id := strings.TrimSpace(v.ID)
	if id == "" {
		return r.Create(ctx, v)
	}

	now := time.Now().UTC()
	if v.UpdatedAt.IsZero() {
		v.UpdatedAt = now
	}
	v.ID = id

	docRef := r.col().Doc(id)
	data := productToDoc(v, now)

	_, err := docRef.Set(ctx, data, firestore.MergeAll)
	if err != nil {
		return productdom.Product{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return productdom.Product{}, err
	}
	out, err := docToProduct(snap)
	if err != nil {
		return productdom.Product{}, err
	}
	return out, nil
}

// Delete removes a product by ID
func (r *ProductRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return productdom.ErrNotFound
	}

	_, err := r.col().Doc(id).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return productdom.ErrNotFound
		}
		return err
	}
	return nil
}

// ============================================================
// Extra helper/query methods (not required by ProductRepo interface)
// ============================================================

// List with filter/sort/pagination (implemented via Firestore query + in-memory filter)
func (r *ProductRepositoryFS) List(
	ctx context.Context,
	filter productdom.Filter,
	sort productdom.Sort,
	page productdom.Page,
) (productdom.PageResult, error) {
	if r.Client == nil {
		return productdom.PageResult{}, errors.New("firestore client is nil")
	}

	pageNum, perPage, offset := fscommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	q := r.col().Query
	q = applyProductSort(q, sort)

	it := q.Documents(ctx)
	defer it.Stop()

	var all []productdom.Product
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return productdom.PageResult{}, err
		}
		p, err := docToProduct(doc)
		if err != nil {
			return productdom.PageResult{}, err
		}
		if matchProductFilter(p, filter) {
			all = append(all, p)
		}
	}

	total := len(all)
	if total == 0 {
		return productdom.PageResult{
			Items:      []productdom.Product{},
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

	return productdom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: fscommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *ProductRepositoryFS) Count(ctx context.Context, filter productdom.Filter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("firestore client is nil")
	}

	q := r.col().Query
	it := q.Documents(ctx)
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
		p, err := docToProduct(doc)
		if err != nil {
			return 0, err
		}
		if matchProductFilter(p, filter) {
			total++
		}
	}
	return total, nil
}

// Update: partial update patch API (not required by interface)
func (r *ProductRepositoryFS) Update(ctx context.Context, id string, in productdom.UpdateProductInput) (productdom.Product, error) {
	if r.Client == nil {
		return productdom.Product{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return productdom.Product{}, productdom.ErrNotFound
	}

	docRef := r.col().Doc(id)
	var updates []firestore.Update

	setStr := func(path string, p *string) {
		if p != nil {
			updates = append(updates, firestore.Update{
				Path:  path,
				Value: strings.TrimSpace(*p),
			})
		}
	}
	setTime := func(path string, t *time.Time) {
		if t != nil {
			updates = append(updates, firestore.Update{
				Path:  path,
				Value: t.UTC(),
			})
		}
	}

	setStr("modelId", in.ModelID)
	setStr("productionId", in.ProductionID)

	if in.InspectionResult != nil {
		updates = append(updates, firestore.Update{
			Path:  "inspectionResult",
			Value: strings.TrimSpace(string(*in.InspectionResult)),
		})
	}

	if in.ConnectedToken != nil {
		v := strings.TrimSpace(*in.ConnectedToken)
		if v == "" {
			updates = append(updates, firestore.Update{
				Path:  "connectedToken",
				Value: firestore.Delete,
			})
		} else {
			updates = append(updates, firestore.Update{
				Path:  "connectedToken",
				Value: v,
			})
		}
	}

	setTime("printedAt", in.PrintedAt)
	setStr("printedBy", in.PrintedBy)
	setTime("inspectedAt", in.InspectedAt)
	setStr("inspectedBy", in.InspectedBy)

	if in.UpdatedBy != nil {
		setStr("updatedBy", in.UpdatedBy)
	}

	// always bump updatedAt
	updates = append(updates, firestore.Update{
		Path:  "updatedAt",
		Value: time.Now().UTC(),
	})

	if len(updates) == 1 { // only updatedAt
		return r.GetByID(ctx, id)
	}

	_, err := docRef.Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return productdom.Product{}, productdom.ErrNotFound
		}
		return productdom.Product{}, err
	}

	return r.GetByID(ctx, id)
}

// UpdateInspection: convenience helper
func (r *ProductRepositoryFS) UpdateInspection(ctx context.Context, id string, in productdom.UpdateInspectionInput) (productdom.Product, error) {
	if r.Client == nil {
		return productdom.Product{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return productdom.Product{}, productdom.ErrNotFound
	}

	docRef := r.col().Doc(id)

	now := time.Now().UTC()
	inspectedAt := now
	if in.InspectedAt != nil && !in.InspectedAt.IsZero() {
		inspectedAt = in.InspectedAt.UTC()
	}

	updates := []firestore.Update{
		{
			Path:  "inspectionResult",
			Value: strings.TrimSpace(string(in.InspectionResult)),
		},
		{
			Path:  "inspectedBy",
			Value: strings.TrimSpace(in.InspectedBy),
		},
		{
			Path:  "inspectedAt",
			Value: inspectedAt,
		},
		{
			Path:  "updatedAt",
			Value: now,
		},
		{
			Path:  "updatedBy",
			Value: strings.TrimSpace(in.InspectedBy),
		},
	}

	_, err := docRef.Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return productdom.Product{}, productdom.ErrNotFound
		}
		return productdom.Product{}, err
	}

	return r.GetByID(ctx, id)
}

// ConnectToken: convenience helper
func (r *ProductRepositoryFS) ConnectToken(ctx context.Context, id string, in productdom.ConnectTokenInput) (productdom.Product, error) {
	if r.Client == nil {
		return productdom.Product{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return productdom.Product{}, productdom.ErrNotFound
	}

	docRef := r.col().Doc(id)

	now := time.Now().UTC()
	var updates []firestore.Update

	if in.TokenID == nil || strings.TrimSpace(*in.TokenID) == "" {
		updates = append(updates, firestore.Update{
			Path:  "connectedToken",
			Value: firestore.Delete,
		})
	} else {
		updates = append(updates, firestore.Update{
			Path:  "connectedToken",
			Value: strings.TrimSpace(*in.TokenID),
		})
	}
	updates = append(updates, firestore.Update{
		Path:  "updatedAt",
		Value: now,
	})

	_, err := docRef.Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return productdom.Product{}, productdom.ErrNotFound
		}
		return productdom.Product{}, err
	}

	return r.GetByID(ctx, id)
}

// ============================================================
// Helpers
// ============================================================

func docToProduct(doc *firestore.DocumentSnapshot) (productdom.Product, error) {
	data := doc.Data()
	if data == nil {
		return productdom.Product{}, fmt.Errorf("empty product document: %s", doc.Ref.ID)
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
	getTimeVal := func(keys ...string) time.Time {
		if tp := getTimePtr(keys...); tp != nil {
			return *tp
		}
		return time.Time{}
	}

	p := productdom.Product{
		ID:               doc.Ref.ID,
		ModelID:          getStr("modelId", "model_id"),
		ProductionID:     getStr("productionId", "production_id"),
		InspectionResult: productdom.InspectionResult(getStr("inspectionResult", "inspection_result")),
		ConnectedToken:   getStrPtr("connectedToken", "connected_token"),
		PrintedAt:        getTimePtr("printedAt", "printed_at"),
		PrintedBy:        getStrPtr("printedBy", "printed_by"),
		InspectedAt:      getTimePtr("inspectedAt", "inspected_at"),
		InspectedBy:      getStrPtr("inspectedBy", "inspected_by"),
		UpdatedAt:        getTimeVal("updatedAt", "updated_at"),
		UpdatedBy:        getStr("updatedBy", "updated_by"),
	}

	return p, nil
}

func productToDoc(v productdom.Product, now time.Time) map[string]any {
	m := map[string]any{
		"modelId":      strings.TrimSpace(v.ModelID),
		"productionId": strings.TrimSpace(v.ProductionID),
	}

	if ir := strings.TrimSpace(string(v.InspectionResult)); ir != "" {
		m["inspectionResult"] = ir
	}

	if v.ConnectedToken != nil {
		if s := strings.TrimSpace(*v.ConnectedToken); s != "" {
			m["connectedToken"] = s
		}
	}

	if v.PrintedAt != nil && !v.PrintedAt.IsZero() {
		m["printedAt"] = v.PrintedAt.UTC()
	}
	if v.PrintedBy != nil {
		if s := strings.TrimSpace(*v.PrintedBy); s != "" {
			m["printedBy"] = s
		}
	}
	if v.InspectedAt != nil && !v.InspectedAt.IsZero() {
		m["inspectedAt"] = v.InspectedAt.UTC()
	}
	if v.InspectedBy != nil {
		if s := strings.TrimSpace(*v.InspectedBy); s != "" {
			m["inspectedBy"] = s
		}
	}

	// updatedAt / updatedBy
	if !v.UpdatedAt.IsZero() {
		m["updatedAt"] = v.UpdatedAt.UTC()
	} else {
		m["updatedAt"] = now
	}
	if s := strings.TrimSpace(v.UpdatedBy); s != "" {
		m["updatedBy"] = s
	}

	return m
}

// matchProductFilter applies productdom.Filter in-memory (Firestore analogue of SQL WHERE).
func matchProductFilter(p productdom.Product, f productdom.Filter) bool {
	if v := strings.TrimSpace(f.ID); v != "" && p.ID != v {
		return false
	}
	if v := strings.TrimSpace(f.ModelID); v != "" && strings.TrimSpace(p.ModelID) != v {
		return false
	}
	if v := strings.TrimSpace(f.ProductionID); v != "" && strings.TrimSpace(p.ProductionID) != v {
		return false
	}

	if len(f.InspectionResults) > 0 {
		ok := false
		for _, ir := range f.InspectionResults {
			if strings.TrimSpace(string(ir)) == strings.TrimSpace(string(p.InspectionResult)) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	if f.HasToken != nil {
		has := p.ConnectedToken != nil && strings.TrimSpace(*p.ConnectedToken) != ""
		if *f.HasToken != has {
			return false
		}
	}

	if v := strings.TrimSpace(f.TokenID); v != "" {
		if p.ConnectedToken == nil || strings.TrimSpace(*p.ConnectedToken) != v {
			return false
		}
	}

	// Time ranges
	if f.PrintedFrom != nil {
		if p.PrintedAt == nil || p.PrintedAt.Before(f.PrintedFrom.UTC()) {
			return false
		}
	}
	if f.PrintedTo != nil {
		if p.PrintedAt == nil || !p.PrintedAt.Before(f.PrintedTo.UTC()) {
			return false
		}
	}
	if f.InspectedFrom != nil {
		if p.InspectedAt == nil || p.InspectedAt.Before(f.InspectedFrom.UTC()) {
			return false
		}
	}
	if f.InspectedTo != nil {
		if p.InspectedAt == nil || !p.InspectedAt.Before(f.InspectedTo.UTC()) {
			return false
		}
	}
	if f.UpdatedFrom != nil {
		if p.UpdatedAt.IsZero() || p.UpdatedAt.Before(f.UpdatedFrom.UTC()) {
			return false
		}
	}
	if f.UpdatedTo != nil {
		if p.UpdatedAt.IsZero() || !p.UpdatedAt.Before(f.UpdatedTo.UTC()) {
			return false
		}
	}

	return true
}

// applyProductSort maps productdom.Sort to Firestore orderBy.
func applyProductSort(q firestore.Query, sort productdom.Sort) firestore.Query {
	col := strings.ToLower(strings.TrimSpace(string(sort.Column)))
	var field string

	switch col {
	case "updatedat", "updated_at":
		field = "updatedAt"
	case "printedat", "printed_at":
		field = "printedAt"
	case "inspectedat", "inspected_at":
		field = "inspectedAt"
	case "modelid", "model_id":
		field = "modelId"
	case "productionid", "production_id":
		field = "productionId"
	default:
		// default: updatedAt DESC, then id
		return q.OrderBy("updatedAt", firestore.Desc).
			OrderBy(firestore.DocumentID, firestore.Desc)
	}

	dir := firestore.Desc
	if strings.EqualFold(string(sort.Order), "asc") {
		dir = firestore.Asc
	}

	return q.OrderBy(field, dir).
		OrderBy(firestore.DocumentID, dir)
}
