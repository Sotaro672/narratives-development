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
// ListByProductionID: 同一 productionId を持つ Product 一覧を取得
// ============================================================

func (r *ProductRepositoryFS) ListByProductionID(ctx context.Context, productionID string) ([]productdom.Product, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	productionID = strings.TrimSpace(productionID)
	if productionID == "" {
		// productionID 未指定なら空配列を返す（エラーにはしない）
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
// PrintLogRepositoryFS: print_logs 用 Firestore リポジトリ
//   usecase.PrintLogRepo の Create / ListByProductionID を実装する
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

// Create は新しい print_log を 1 件保存します。
// ID が空なら Firestore の auto-ID を採用します。
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
			// PrintLog 固有の ErrConflict は定義していないので、そのまま返す
			return productdom.PrintLog{}, err
		}
		return productdom.PrintLog{}, err
	}

	// Firestore 側で timestamp などが変わる可能性もあるので、再取得して返す
	snap, err := docRef.Get(ctx)
	if err != nil {
		return productdom.PrintLog{}, err
	}
	return docToPrintLog(snap)
}

// ListByProductionID: 同一 productionId を持つ PrintLog 一覧を取得
func (r *PrintLogRepositoryFS) ListByProductionID(ctx context.Context, productionID string) ([]productdom.PrintLog, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	productionID = strings.TrimSpace(productionID)
	if productionID == "" {
		// productionID 未指定なら空配列を返す（エラーにはしない）
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
// InspectionRepositoryFS: inspections_by_production 用 Firestore リポジトリ
//   usecase.InspectionRepo の Create / GetByProductionID / Save を実装する
// ============================================================

type InspectionRepositoryFS struct {
	Client *firestore.Client
}

func NewInspectionRepositoryFS(client *firestore.Client) *InspectionRepositoryFS {
	return &InspectionRepositoryFS{Client: client}
}

func (r *InspectionRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("inspections_by_production")
}

// Create: inspections_by_production/{productionId} に 1 ドキュメント作成
//
//	productionId をドキュメントIDとして保存します。
func (r *InspectionRepositoryFS) Create(ctx context.Context, v productdom.InspectionBatch) (productdom.InspectionBatch, error) {
	if r.Client == nil {
		return productdom.InspectionBatch{}, errors.New("firestore client is nil")
	}

	pid := strings.TrimSpace(v.ProductionID)
	if pid == "" {
		return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionProductionID
	}

	docRef := r.col().Doc(pid)
	data := inspectionBatchToDoc(v)

	_, err := docRef.Create(ctx, data)
	if err != nil {
		// 既に存在している場合はそのままエラーを返す
		if status.Code(err) == codes.AlreadyExists {
			return productdom.InspectionBatch{}, err
		}
		return productdom.InspectionBatch{}, err
	}

	// Firestore から再取得して整形して返す
	snap, err := docRef.Get(ctx)
	if err != nil {
		return productdom.InspectionBatch{}, err
	}
	return docToInspectionBatch(snap)
}

// GetByProductionID: inspections_by_production/{productionId} を取得
func (r *InspectionRepositoryFS) GetByProductionID(
	ctx context.Context,
	productionID string,
) (productdom.InspectionBatch, error) {
	if r.Client == nil {
		return productdom.InspectionBatch{}, errors.New("firestore client is nil")
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionProductionID
	}

	docRef := r.col().Doc(pid)
	snap, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return productdom.InspectionBatch{}, productdom.ErrNotFound
		}
		return productdom.InspectionBatch{}, err
	}

	return docToInspectionBatch(snap)
}

// Save: inspections_by_production/{productionId} を Upsert 的に保存
func (r *InspectionRepositoryFS) Save(
	ctx context.Context,
	v productdom.InspectionBatch,
) (productdom.InspectionBatch, error) {
	if r.Client == nil {
		return productdom.InspectionBatch{}, errors.New("firestore client is nil")
	}

	pid := strings.TrimSpace(v.ProductionID)
	if pid == "" {
		return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionProductionID
	}

	docRef := r.col().Doc(pid)
	data := inspectionBatchToDoc(v)

	_, err := docRef.Set(ctx, data, firestore.MergeAll)
	if err != nil {
		return productdom.InspectionBatch{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return productdom.InspectionBatch{}, err
	}

	return docToInspectionBatch(snap)
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

// print_logs 用の変換
func docToPrintLog(doc *firestore.DocumentSnapshot) (productdom.PrintLog, error) {
	data := doc.Data()
	if data == nil {
		return productdom.PrintLog{}, fmt.Errorf("empty print_log document: %s", doc.Ref.ID)
	}

	// productIds は []interface{} として返ってくることが多いので安全に変換
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

	var printedAt time.Time
	if v, ok := data["printedAt"].(time.Time); ok && !v.IsZero() {
		printedAt = v.UTC()
	}

	log := productdom.PrintLog{
		ID:           doc.Ref.ID,
		ProductionID: strings.TrimSpace(asString(data["productionId"])),
		ProductIDs:   productIDs,
		PrintedAt:    printedAt,
	}

	return log, nil
}

func printLogToDoc(v productdom.PrintLog) map[string]any {
	m := map[string]any{
		"productionId": strings.TrimSpace(v.ProductionID),
		"productIds":   v.ProductIDs,
		"printedAt":    v.PrintedAt.UTC(),
	}
	return m
}

// inspections_by_production 用の変換 (domain -> Firestore)
func inspectionBatchToDoc(v productdom.InspectionBatch) map[string]any {
	items := make([]map[string]any, 0, len(v.Inspections))
	for _, ins := range v.Inspections {
		m := map[string]any{
			"productId": strings.TrimSpace(ins.ProductID),
		}

		if ins.InspectionResult != nil {
			m["inspectionResult"] = string(*ins.InspectionResult)
		} else {
			m["inspectionResult"] = nil
		}

		if ins.InspectedBy != nil {
			s := strings.TrimSpace(*ins.InspectedBy)
			if s != "" {
				m["inspectedBy"] = s
			} else {
				m["inspectedBy"] = nil
			}
		} else {
			m["inspectedBy"] = nil
		}

		if ins.InspectedAt != nil && !ins.InspectedAt.IsZero() {
			m["inspectedAt"] = ins.InspectedAt.UTC()
		} else {
			m["inspectedAt"] = nil
		}

		items = append(items, m)
	}

	return map[string]any{
		"productionId": strings.TrimSpace(v.ProductionID),
		"status":       string(v.Status),
		"inspections":  items,
	}
}

// Firestore -> domain.InspectionBatch
func docToInspectionBatch(doc *firestore.DocumentSnapshot) (productdom.InspectionBatch, error) {
	data := doc.Data()
	if data == nil {
		return productdom.InspectionBatch{}, fmt.Errorf("empty inspection document: %s", doc.Ref.ID)
	}

	productionID := strings.TrimSpace(asString(data["productionId"]))
	statusStr := strings.TrimSpace(asString(data["status"]))

	batch := productdom.InspectionBatch{
		ProductionID: productionID,
		Status:       productdom.InspectionStatus(statusStr),
	}

	// inspections 配列のパース
	if raw, ok := data["inspections"]; ok && raw != nil {
		switch vv := raw.(type) {
		case []interface{}:
			for _, e := range vv {
				m, ok := e.(map[string]interface{})
				if !ok {
					continue
				}
				item := productdom.InspectionItem{}

				if v, ok := m["productId"].(string); ok {
					item.ProductID = strings.TrimSpace(v)
				}

				if v, ok := m["inspectionResult"].(string); ok && strings.TrimSpace(v) != "" {
					r := productdom.InspectionResult(strings.TrimSpace(v))
					item.InspectionResult = &r
				}

				if v, ok := m["inspectedBy"].(string); ok && strings.TrimSpace(v) != "" {
					s := strings.TrimSpace(v)
					item.InspectedBy = &s
				}

				if v, ok := m["inspectedAt"].(time.Time); ok && !v.IsZero() {
					t := v.UTC()
					item.InspectedAt = &t
				}

				batch.Inspections = append(batch.Inspections, item)
			}
		}
	}

	// ざっくりしたバリデーション（最低限）
	if strings.TrimSpace(batch.ProductionID) == "" {
		return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionProductionID
	}
	if len(batch.Inspections) == 0 {
		return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionProductIDs
	}

	return batch, nil
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
