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

	fscommon "narratives/internal/adapters/out/firestore/common"
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

// Update updates an existing product by ID.
func (r *ProductRepositoryFS) Update(ctx context.Context, id string, v productdom.Product) (productdom.Product, error) {
	if r.Client == nil {
		return productdom.Product{}, errors.New("firestore client is nil")
	}

	if id == "" {
		return productdom.Product{}, productdom.ErrNotFound
	}

	docRef := r.col().Doc(id)

	// 存在確認。Set(MergeAll) だけだと存在しない document を作れてしまうため、
	// repository port の Update としては not found を返す。
	if _, err := docRef.Get(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return productdom.Product{}, productdom.ErrNotFound
		}
		return productdom.Product{}, err
	}

	v.ID = id
	data := productToDoc(v)

	_, err := docRef.Set(ctx, data, firestore.MergeAll)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return productdom.Product{}, productdom.ErrNotFound
		}
		return productdom.Product{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return productdom.Product{}, productdom.ErrNotFound
		}
		return productdom.Product{}, err
	}

	return docToProduct(snap)
}

// Delete deletes a product by ID.
func (r *ProductRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

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
// PrintLogRepositoryFS
// ============================================================

type PrintLogRepositoryFS struct {
	Client *firestore.Client
}

func NewPrintLogRepositoryFS(client *firestore.Client) *PrintLogRepositoryFS {
	return &PrintLogRepositoryFS{Client: client}
}

func (r *PrintLogRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("print_logs")
}

func (r *PrintLogRepositoryFS) Create(ctx context.Context, v productdom.PrintLog) (productdom.PrintLog, error) {
	if r.Client == nil {
		return productdom.PrintLog{}, errors.New("firestore client is nil")
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

	data := printLogToDoc(v)

	_, err := docRef.Create(ctx, data)
	if err != nil {
		return productdom.PrintLog{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return productdom.PrintLog{}, err
	}

	return docToPrintLog(snap)
}

func (r *PrintLogRepositoryFS) ListByProductionID(ctx context.Context, productionID string) ([]productdom.PrintLog, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	if productionID == "" {
		return []productdom.PrintLog{}, nil
	}

	q := r.col().Where("productionId", "==", productionID)
	it := q.Documents(ctx)
	defer it.Stop()

	var logs []productdom.PrintLog
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		l, err := docToPrintLog(doc)
		if err != nil {
			return nil, err
		}

		logs = append(logs, l)
	}

	return logs, nil
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

func docToPrintLog(doc *firestore.DocumentSnapshot) (productdom.PrintLog, error) {
	data := doc.Data()
	if data == nil {
		return productdom.PrintLog{}, fmt.Errorf("empty print_log document: %s", doc.Ref.ID)
	}

	var items []productdom.PrintedItem
	if raw, ok := data["items"]; ok {
		switch vv := raw.(type) {
		case []interface{}:
			for _, x := range vv {
				m, ok := x.(map[string]interface{})
				if !ok {
					continue
				}

				pidAny := m["productId"]
				orderAny := m["displayOrder"]

				pid, _ := pidAny.(string)

				var order int
				switch t := orderAny.(type) {
				case int:
					order = t
				case int64:
					order = int(t)
				case float64:
					order = int(t)
				default:
					order = 0
				}

				if pid == "" || order <= 0 {
					continue
				}

				items = append(items, productdom.PrintedItem{
					ProductID:    pid,
					DisplayOrder: order,
				})
			}
		}
	}

	productionID := fscommon.AsString(data["productionId"])

	return productdom.NewPrintLog(
		doc.Ref.ID,
		productionID,
		items,
	)
}

func printLogToDoc(v productdom.PrintLog) map[string]any {
	items := make([]map[string]any, 0, len(v.Items))
	for _, it := range v.Items {
		items = append(items, map[string]any{
			"productId":    it.ProductID,
			"displayOrder": it.DisplayOrder,
		})
	}

	return map[string]any{
		"productionId": v.ProductionID,
		"items":        items,
	}
}
