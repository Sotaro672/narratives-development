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

// GetByID returns a single Product by ID
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

	return docToProduct(snap)
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

// Create inserts a new product (Firestore auto-ID allowed)
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
	return docToProduct(snap)
}

// Save = full upsert
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
	return docToProduct(snap)
}

// Update(ctx, id, product) = usecase.ProductRepo と互換の full update
// usecase 側で更新可能フィールドだけを上書き済みの Product が渡される想定。
func (r *ProductRepositoryFS) Update(ctx context.Context, id string, v productdom.Product) (productdom.Product, error) {
	if r.Client == nil {
		return productdom.Product{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return productdom.Product{}, productdom.ErrNotFound
	}

	// ID は常にパスの id を優先
	v.ID = id

	return r.Save(ctx, v)
}

// ============================================================
// List （filter / sort を無視した簡易版）
// ============================================================

func (r *ProductRepositoryFS) List(
	ctx context.Context,
	filter productdom.Filter,
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

	return productdom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: 1,
		Page:       1,
		PerPage:    total,
	}, nil
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

	return productdom.Product{
		ID:               doc.Ref.ID,
		ModelID:          getStr("modelId"),
		ProductionID:     getStr("productionId"),
		InspectionResult: productdom.InspectionResult(getStr("inspectionResult")),
		ConnectedToken:   getStrPtr("connectedToken"),
		PrintedAt:        getTimePtr("printedAt"),
		PrintedBy:        getStrPtr("printedBy"),
		InspectedAt:      getTimePtr("inspectedAt"),
		InspectedBy:      getStrPtr("inspectedBy"),
	}, nil
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
