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

// Update(ctx, id, product) = full update
func (r *ProductRepositoryFS) Update(ctx context.Context, id string, v productdom.Product) (productdom.Product, error) {
	if r.Client == nil {
		return productdom.Product{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return productdom.Product{}, productdom.ErrNotFound
	}

	v.ID = id
	return r.Save(ctx, v)
}

// ============================================================
// ListByProductionID
// ============================================================

func (r *ProductRepositoryFS) ListByProductionID(ctx context.Context, productionID string) ([]productdom.Product, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	productionID = strings.TrimSpace(productionID)
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
// List (simple)
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
// ProductInspectionRepo 用メソッド
// ============================================================

// UpdateInspectionResult implements usecase.ProductInspectionRepo.
//
// inspections テーブルの更新にあわせて、products/{productId} の
// inspectionResult を部分更新します。
func (r *ProductRepositoryFS) UpdateInspectionResult(
	ctx context.Context,
	productID string,
	result productdom.InspectionResult,
) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id := strings.TrimSpace(productID)
	if id == "" {
		return productdom.ErrNotFound
	}

	updates := map[string]any{
		"inspectionResult": strings.TrimSpace(string(result)),
	}

	_, err := r.col().Doc(id).Set(ctx, updates, firestore.MergeAll)
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

	id := strings.TrimSpace(v.ID)
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
		if status.Code(err) == codes.AlreadyExists {
			return productdom.PrintLog{}, err
		}
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

	productionID = strings.TrimSpace(productionID)
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
		PrintedAt:        getTimePtr("printedAt"),
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

	if v.PrintedAt != nil && !v.PrintedAt.IsZero() {
		m["printedAt"] = v.PrintedAt.UTC()
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

func docToPrintLog(doc *firestore.DocumentSnapshot) (productdom.PrintLog, error) {
	data := doc.Data()
	if data == nil {
		return productdom.PrintLog{}, fmt.Errorf("empty print_log document: %s", doc.Ref.ID)
	}

	var productIDs []string
	if raw, ok := data["productIds"]; ok {
		switch vv := raw.(type) {
		case []interface{}:
			for _, x := range vv {
				if s, ok := x.(string); ok && strings.TrimSpace(s) != "" {
					productIDs = append(productIDs, strings.TrimSpace(s))
				}
			}
		case []string:
			for _, s := range vv {
				s = strings.TrimSpace(s)
				if s != "" {
					productIDs = append(productIDs, s)
				}
			}
		}
	}

	productionID := strings.TrimSpace(fscommon.AsString(data["productionId"]))

	// ★ PrintLog は printedAt / printedBy を持たないので、
	//   Firestore に残っていても読み込まず、ID / productionId / productIds のみで構築する。
	return productdom.NewPrintLog(
		doc.Ref.ID,
		productionID,
		productIDs,
	)
}

func printLogToDoc(v productdom.PrintLog) map[string]any {
	// ★ PrintLog 側から printedAt / printedBy を削除したので、
	//   Firestore には productionId / productIds だけを保存する。
	return map[string]any{
		"productionId": strings.TrimSpace(v.ProductionID),
		"productIds":   v.ProductIDs,
	}
}
