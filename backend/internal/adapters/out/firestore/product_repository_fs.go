// backend/internal/adapters/out/firestore/product_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	inspectiondom "narratives/internal/domain/inspection"
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
// Product Repository methods
// ============================================================

// GetByID returns a single Product by ID.
func (r *ProductRepositoryFS) GetByID(ctx context.Context, id string) (productdom.Product, error) {
	if r.Client == nil {
		return productdom.Product{}, errors.New("firestore client is nil")
	}

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

// Create inserts a new product. Firestore auto-ID is allowed.
func (r *ProductRepositoryFS) Create(ctx context.Context, v productdom.Product) (productdom.Product, error) {
	if r.Client == nil {
		return productdom.Product{}, errors.New("firestore client is nil")
	}

	id := v.ID
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

// ListByProductionID returns products that belong to the given productionID.
func (r *ProductRepositoryFS) ListByProductionID(ctx context.Context, productionID string) ([]productdom.Product, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	if productionID == "" {
		return []productdom.Product{}, nil
	}

	q := r.col().Where("productionId", "==", productionID)
	it := q.Documents(ctx)
	defer it.Stop()

	var items []productdom.Product
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		p, err := docToProduct(doc)
		if err != nil {
			return nil, err
		}

		items = append(items, p)
	}

	return items, nil
}

// ============================================================
// ProductModelResolver 用メソッド
// ============================================================

// GetModelIDByProductID returns modelId for a product.
// This method implements usecase.ProductModelResolver without importing the usecase package.
func (r *ProductRepositoryFS) GetModelIDByProductID(
	ctx context.Context,
	productID string,
) (string, error) {
	if r == nil || r.Client == nil {
		return "", errors.New("product repository/firestore client is nil")
	}

	if productID == "" {
		return "", productdom.ErrNotFound
	}

	product, err := r.GetByID(ctx, productID)
	if err != nil {
		return "", err
	}

	if product.ModelID == "" {
		return "", errors.New("product modelId is empty")
	}

	return product.ModelID, nil
}

// ============================================================
// ProductInspectionRepo 用メソッド
// ============================================================

// UpdateInspectionResult implements application/inspection.ProductInspectionRepo.
//
// inspections テーブルの更新にあわせて、products/{productId} の
// inspectionResult を部分更新します。
func (r *ProductRepositoryFS) UpdateInspectionResult(
	ctx context.Context,
	productID string,
	result inspectiondom.InspectionResult,
) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id := productID
	if id == "" {
		return productdom.ErrNotFound
	}

	docRef := r.col().Doc(id)

	if _, err := docRef.Get(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return productdom.ErrNotFound
		}
		return err
	}

	updates := map[string]any{
		"inspectionResult": string(result),
	}

	_, err := docRef.Set(ctx, updates, firestore.MergeAll)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return productdom.ErrNotFound
		}
		return err
	}

	return nil
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
				return v
			}
		}
		return ""
	}

	getStrPtr := func(keys ...string) *string {
		for _, k := range keys {
			if v, ok := data[k].(string); ok {
				if v != "" {
					s := v
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
		PrintedAt:        getTimePtr("printedAt"),
		InspectedAt:      getTimePtr("inspectedAt"),
		InspectedBy:      getStrPtr("inspectedBy"),
	}, nil
}

func productToDoc(v productdom.Product) map[string]any {
	m := map[string]any{
		"modelId":      v.ModelID,
		"productionId": v.ProductionID,
	}

	if ir := string(v.InspectionResult); ir != "" {
		m["inspectionResult"] = ir
	}

	if v.PrintedAt != nil && !v.PrintedAt.IsZero() {
		m["printedAt"] = v.PrintedAt.UTC()
	}

	if v.InspectedAt != nil && !v.InspectedAt.IsZero() {
		m["inspectedAt"] = v.InspectedAt.UTC()
	}

	if v.InspectedBy != nil && *v.InspectedBy != "" {
		m["inspectedBy"] = *v.InspectedBy
	}

	return m
}
