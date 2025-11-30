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
// - inspection_result, connected_token, printed_at/by, inspected_at/by can be nil.
func (r *ProductRepositoryFS) Create(ctx context.Context, v productdom.Product) (productdom.Product, error) {
	if r.Client == nil {
		return productdom.Product{}, errors.New("firestore client is nil")
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

	data := productToDoc(v)

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
//
// NOTE: updatedAt / updatedBy は Product から削除済みのため扱いません。
func (r *ProductRepositoryFS) Save(ctx context.Context, v productdom.Product) (productdom.Product, error) {
	if r.Client == nil {
		return productdom.Product{}, errors.New("firestore client is nil")
	}

	id := strings.TrimSpace(v.ID)
	if id == "" {
		return r.Create(ctx, v)
	}

	v.ID = id
	docRef := r.col().Doc(id)
	data := productToDoc(v)

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
// Extra helper/query methods
// ============================================================

// List: 簡易版。filter / sort / orderBy をすべて無視して全件返す。
func (r *ProductRepositoryFS) List(
	ctx context.Context,
	filter productdom.Filter,
	sort productdom.Sort,
	page productdom.Page,
) (productdom.PageResult, error) {
	if r.Client == nil {
		return productdom.PageResult{}, errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	var items []productdom.Product
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
		items = append(items, p)
	}

	total := len(items)
	if total == 0 {
		return productdom.PageResult{
			Items:      []productdom.Product{},
			TotalCount: 0,
			TotalPages: 0,
			Page:       1,
			PerPage:    0,
		}, nil
	}

	return productdom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: 1,
		Page:       1,
		PerPage:    total,
	}, nil
}

// Count: 簡易版。filter を無視して全件数を返す。
func (r *ProductRepositoryFS) Count(ctx context.Context, filter productdom.Filter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	total := 0
	for {
		_, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		total++
	}
	return total, nil
}

// Update: partial update patch API
//
// updatedAt / updatedBy の更新は削除。
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

	if len(updates) == 0 {
		// 変更なしならそのまま返す
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
//
// updatedAt / updatedBy は更新しない。
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
//
// updatedAt は更新せず、connectedToken のみ更新。
func (r *ProductRepositoryFS) ConnectToken(ctx context.Context, id string, in productdom.ConnectTokenInput) (productdom.Product, error) {
	if r.Client == nil {
		return productdom.Product{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return productdom.Product{}, productdom.ErrNotFound
	}

	docRef := r.col().Doc(id)

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

	if len(updates) == 0 {
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
	}

	return p, nil
}

func productToDoc(v productdom.Product) map[string]any {
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

	return m
}
