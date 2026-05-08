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
	commondom "narratives/internal/domain/common"
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
// ProductRepo interface methods
// ============================================================

// GetByID returns a single Product by ID
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

// Exists checks if a product with the given ID exists
func (r *ProductRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
	if r.Client == nil {
		return false, errors.New("firestore client is nil")
	}

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

// Save = full upsert
func (r *ProductRepositoryFS) Save(ctx context.Context, v productdom.Product) (productdom.Product, error) {
	if r.Client == nil {
		return productdom.Product{}, errors.New("firestore client is nil")
	}

	id := v.ID
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

	if id == "" {
		return productdom.Product{}, productdom.ErrNotFound
	}

	v.ID = id
	return r.Save(ctx, v)
}

// Delete deletes a product by ID
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

// ============================================================
// ListByProductionID
// ============================================================

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
// List
// ============================================================

func (r *ProductRepositoryFS) List(
	ctx context.Context,
	filter productdom.Filter,
	sortOpt commondom.Sort,
	page commondom.Page,
) (commondom.PageResult[productdom.Product], error) {
	if r.Client == nil {
		return commondom.PageResult[productdom.Product]{}, errors.New("firestore client is nil")
	}

	q := r.col().Query

	if filter.ID != "" {
		q = q.Where(firestore.DocumentID, "==", filter.ID)
	}
	if filter.ModelID != "" {
		q = q.Where("modelId", "==", filter.ModelID)
	}
	if filter.ProductionID != "" {
		q = q.Where("productionId", "==", filter.ProductionID)
	}
	if filter.TokenID != "" {
		q = q.Where("connectedToken", "==", filter.TokenID)
	}
	if filter.HasToken != nil {
		if *filter.HasToken {
			q = q.Where("connectedToken", "!=", "")
		}
	}

	if len(filter.InspectionResults) == 1 {
		q = q.Where("inspectionResult", "==", string(filter.InspectionResults[0]))
	}

	if filter.Printed.From != nil {
		q = q.Where("printedAt", ">=", filter.Printed.From.UTC())
	}
	if filter.Printed.To != nil {
		q = q.Where("printedAt", "<=", filter.Printed.To.UTC())
	}
	if filter.Inspected.From != nil {
		q = q.Where("inspectedAt", ">=", filter.Inspected.From.UTC())
	}
	if filter.Inspected.To != nil {
		q = q.Where("inspectedAt", "<=", filter.Inspected.To.UTC())
	}
	if filter.Created.From != nil {
		q = q.Where("createdAt", ">=", filter.Created.From.UTC())
	}
	if filter.Created.To != nil {
		q = q.Where("createdAt", "<=", filter.Created.To.UTC())
	}
	if filter.Updated.From != nil {
		q = q.Where("updatedAt", ">=", filter.Updated.From.UTC())
	}
	if filter.Updated.To != nil {
		q = q.Where("updatedAt", "<=", filter.Updated.To.UTC())
	}

	if sortOpt.Column != "" {
		dir := firestore.Asc
		if sortOpt.Order == commondom.SortDesc {
			dir = firestore.Desc
		}
		q = q.OrderBy(sortOpt.Column, dir)
	}

	it := q.Documents(ctx)
	defer it.Stop()

	var allItems []productdom.Product
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return commondom.PageResult[productdom.Product]{}, err
		}
		p, err := docToProduct(doc)
		if err != nil {
			return commondom.PageResult[productdom.Product]{}, err
		}
		allItems = append(allItems, p)
	}

	total := len(allItems)

	perPage := page.PerPage
	if perPage <= 0 {
		perPage = total
		if perPage == 0 {
			perPage = 20
		}
	}

	pageNum := page.Number
	if pageNum <= 0 {
		pageNum = 1
	}

	start := (pageNum - 1) * perPage
	if start > total {
		start = total
	}
	end := start + perPage
	if end > total {
		end = total
	}

	pagedItems := allItems[start:end]

	totalPages := 0
	if total > 0 {
		totalPages = (total + perPage - 1) / perPage
	}

	return commondom.PageResult[productdom.Product]{
		Items:      pagedItems,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
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

	updates := map[string]any{
		"inspectionResult": string(result),
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
	if v.InspectedBy != nil {
		if *v.InspectedBy != "" {
			m["inspectedBy"] = *v.InspectedBy
		}
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
