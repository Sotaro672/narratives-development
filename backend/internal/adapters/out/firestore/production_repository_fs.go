// backend/internal/adapters/out/firestore/production_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
	proddom "narratives/internal/domain/production"
)

// ============================================================
// Firestore-based Production Repository
// ============================================================

type ProductionRepositoryFS struct {
	Client *firestore.Client
}

func NewProductionRepositoryFS(client *firestore.Client) *ProductionRepositoryFS {
	return &ProductionRepositoryFS{Client: client}
}

func (r *ProductionRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("productions")
}

// ============================================================
// RepositoryPort 実装
// ============================================================

// GetByID returns a Production by document ID.
func (r *ProductionRepositoryFS) GetByID(ctx context.Context, id string) (*proddom.Production, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, proddom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, proddom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	p, err := docToProduction(snap)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Exists returns true if a document with that ID exists.
func (r *ProductionRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
	if r.Client == nil {
		return false, errors.New("firestore client is nil")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}

	_, err := r.col().Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Create creates a new Production document from CreateProductionInput.
// - ID は CreateProductionInput には含まれないため、常に Firestore の auto ID を採番
// - Printed が nil の場合は false 扱い
// - CreatedAt/UpdatedAt は省略時 now(UTC)
func (r *ProductionRepositoryFS) Create(
	ctx context.Context,
	in proddom.CreateProductionInput,
) (*proddom.Production, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	now := time.Now().UTC()

	// CreatedAt の決定
	createdAt := now
	if in.CreatedAt != nil && !in.CreatedAt.IsZero() {
		createdAt = in.CreatedAt.UTC()
	}

	// Printed の決定（nil -> false）
	printed := false
	if in.Printed != nil {
		printed = *in.Printed
	}

	// PrintedAt の決定（printed=true の場合は必須扱いなので補完）
	var printedAt *time.Time
	if printed {
		if in.PrintedAt != nil && !in.PrintedAt.IsZero() {
			t := in.PrintedAt.UTC()
			printedAt = &t
		} else {
			t := now
			printedAt = &t
		}
	} else {
		printedAt = nil
	}

	// Entity 組み立て
	p := proddom.Production{
		// ID は後で NewDoc から採番
		ProductBlueprintID: strings.TrimSpace(in.ProductBlueprintID),
		AssigneeID:         strings.TrimSpace(in.AssigneeID),
		Models:             in.Models,

		Printed:   printed,
		PrintedAt: printedAt,
		PrintedBy: nil,

		CreatedBy: fscommon.TrimPtr(in.CreatedBy),
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		UpdatedBy: nil,
	}

	// Firestore doc ref（常に新規）
	ref := r.col().NewDoc()
	p.ID = ref.ID

	data := productionToDoc(p)

	if _, err := ref.Create(ctx, data); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil, proddom.ErrConflict
		}
		return nil, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return nil, err
	}
	out, err := docToProduction(snap)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Save is upsert-ish:
// - If ID is empty -> Create (auto ID)
// - If ID exists -> update via Set(MergeAll)
// - If ID does not exist -> treated as Create with that ID
func (r *ProductionRepositoryFS) Save(ctx context.Context, p proddom.Production) (*proddom.Production, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	now := time.Now().UTC()

	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = now
	}

	// Printed 整合性補正
	if p.Printed {
		if p.PrintedAt == nil || p.PrintedAt.IsZero() {
			t := now
			p.PrintedAt = &t
		} else {
			t := p.PrintedAt.UTC()
			p.PrintedAt = &t
		}
		// PrintedBy は空白を nil に寄せる
		p.PrintedBy = fscommon.TrimPtr(p.PrintedBy)
	} else {
		// 未印刷なら printedAt/printedBy は必ず nil
		p.PrintedAt = nil
		p.PrintedBy = nil
	}

	// UpdatedBy の正規化
	p.UpdatedBy = fscommon.TrimPtr(p.UpdatedBy)

	var ref *firestore.DocumentRef
	id := strings.TrimSpace(p.ID)
	if id == "" {
		ref = r.col().NewDoc()
		p.ID = ref.ID
	} else {
		ref = r.col().Doc(id)
	}

	data := productionToDoc(p)

	if _, err := ref.Set(ctx, data, firestore.MergeAll); err != nil {
		return nil, err
	}

	snap, err := ref.Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, proddom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	out, err := docToProduction(snap)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete performs a hard delete of the document.
func (r *ProductionRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return proddom.ErrNotFound
	}

	ref := r.col().Doc(id)
	_, err := ref.Get(ctx)
	if status.Code(err) == codes.NotFound {
		return proddom.ErrNotFound
	}
	if err != nil {
		return err
	}

	if _, err := ref.Delete(ctx); err != nil {
		return err
	}
	return nil
}

// ListAll returns all productions (sorted by createdAt DESC, then document ID DESC).
// ※ RepositoryPort に無いが、既存利用が残っている可能性があるため維持。
func (r *ProductionRepositoryFS) ListAll(ctx context.Context) ([]proddom.Production, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	q := r.col().Query.
		OrderBy("createdAt", firestore.Desc).
		OrderBy(firestore.DocumentID, firestore.Desc)

	it := q.Documents(ctx)
	defer it.Stop()

	var all []proddom.Production
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		p, err := docToProduction(doc)
		if err != nil {
			return nil, err
		}
		all = append(all, p)
	}

	return all, nil
}

// List implements filtering/paging over ListAll.
// ※ RepositoryPort に無いが、既存利用が残っている可能性があるため維持。
func (r *ProductionRepositoryFS) List(
	ctx context.Context,
	filter proddom.Filter,
	page proddom.Page,
) (proddom.PageResult, error) {
	all, err := r.ListAll(ctx)
	if err != nil {
		return proddom.PageResult{}, err
	}

	var filtered []proddom.Production
	for _, p := range all {
		// ID
		if strings.TrimSpace(filter.ID) != "" && p.ID != strings.TrimSpace(filter.ID) {
			continue
		}
		// ProductBlueprintID
		if strings.TrimSpace(filter.ProductBlueprintID) != "" &&
			p.ProductBlueprintID != strings.TrimSpace(filter.ProductBlueprintID) {
			continue
		}
		// AssigneeID
		if strings.TrimSpace(filter.AssigneeID) != "" &&
			p.AssigneeID != strings.TrimSpace(filter.AssigneeID) {
			continue
		}
		// ModelID（ModelQuantity に含まれるかどうか）
		if strings.TrimSpace(filter.ModelID) != "" {
			target := strings.TrimSpace(filter.ModelID)
			found := false
			for _, mq := range p.Models {
				if strings.TrimSpace(mq.ModelID) == target {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Printed（nil の場合はフィルタしない）
		if filter.Printed != nil {
			if p.Printed != *filter.Printed {
				continue
			}
		}

		// PrintedFrom / PrintedTo
		if filter.PrintedFrom != nil || filter.PrintedTo != nil {
			if p.PrintedAt == nil {
				// PrintedAt が必要条件になるので nil の場合は除外
				continue
			}
			if filter.PrintedFrom != nil && p.PrintedAt.Before(filter.PrintedFrom.UTC()) {
				continue
			}
			if filter.PrintedTo != nil && p.PrintedAt.After(filter.PrintedTo.UTC()) {
				continue
			}
		}

		// CreatedFrom / CreatedTo
		if filter.CreatedFrom != nil && p.CreatedAt.Before(filter.CreatedFrom.UTC()) {
			continue
		}
		if filter.CreatedTo != nil && p.CreatedAt.After(filter.CreatedTo.UTC()) {
			continue
		}

		filtered = append(filtered, p)
	}

	// Paging
	perPage := page.PerPage
	if perPage <= 0 {
		perPage = len(filtered)
	}
	pageNum := page.Number
	if pageNum <= 0 {
		pageNum = 1
	}

	totalCount := len(filtered)
	totalPages := 0
	if perPage > 0 {
		totalPages = (totalCount + perPage - 1) / perPage
	}

	start := (pageNum - 1) * perPage
	if start > totalCount {
		start = totalCount
	}
	end := start + perPage
	if end > totalCount {
		end = totalCount
	}

	items := filtered[start:end]

	return proddom.PageResult{
		Items:      items,
		TotalCount: totalCount,
		TotalPages: totalPages,
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

// ListByProductBlueprintID は、指定された productBlueprintId のいずれかを持つ
// Production をすべて取得します。
// Firestore の "in" オペレータ制限（最大10要素）に対応するため、IDs をチャンクに分けて問い合わせます。
func (r *ProductionRepositoryFS) ListByProductBlueprintID(
	ctx context.Context,
	productBlueprintIDs []string,
) ([]proddom.Production, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	// 空なら即終了
	if len(productBlueprintIDs) == 0 {
		return []proddom.Production{}, nil
	}

	// 空文字を取り除きつつ trim & 重複排除
	uniq := make(map[string]struct{}, len(productBlueprintIDs))
	var ids []string
	for _, id := range productBlueprintIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := uniq[id]; ok {
			continue
		}
		uniq[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return []proddom.Production{}, nil
	}

	const maxIn = 10
	var results []proddom.Production

	for start := 0; start < len(ids); start += maxIn {
		end := start + maxIn
		if end > len(ids) {
			end = len(ids)
		}
		chunk := ids[start:end]

		q := r.col().
			Where("productBlueprintId", "in", chunk)

		it := q.Documents(ctx)
		defer it.Stop()

		for {
			doc, err := it.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				return nil, err
			}
			p, err := docToProduction(doc)
			if err != nil {
				return nil, err
			}
			results = append(results, p)
		}
	}

	return results, nil
}

// GetTotalQuantityByModelID は、productBlueprintIDs 配下の Production.Models を集計し、modelId ごとの totalQuantity を返す。
// 仕様（正式）:
// - productBlueprintIDs は trim + 空除外 + 重複排除（case-insensitive）
// - modelId は case-insensitive に合算し、最初に出現した表記（orig）を返す
// - 戻り値は modelId（case-insensitive）で安定ソートする
func (r *ProductionRepositoryFS) GetTotalQuantityByModelID(
	ctx context.Context,
	productBlueprintIDs []string,
) ([]proddom.ModelTotalQuantity, error) {
	// sanitize + dedup productBlueprintIDs (case-insensitive)
	ids := make([]string, 0, len(productBlueprintIDs))
	seenIDs := make(map[string]struct{}, len(productBlueprintIDs))
	for _, id := range productBlueprintIDs {
		t := strings.TrimSpace(id)
		if t == "" {
			continue
		}
		k := strings.ToLower(t)
		if _, ok := seenIDs[k]; ok {
			continue
		}
		seenIDs[k] = struct{}{}
		ids = append(ids, t)
	}
	if len(ids) == 0 {
		return []proddom.ModelTotalQuantity{}, nil
	}

	prods, err := r.ListByProductBlueprintID(ctx, ids)
	if err != nil {
		return nil, err
	}

	// case-insensitive sum by modelId
	totalByKey := make(map[string]int, 64)
	origByKey := make(map[string]string, 64)

	for _, p := range prods {
		for _, mq := range p.Models {
			mid := strings.TrimSpace(mq.ModelID)
			if mid == "" || mq.Quantity <= 0 {
				continue
			}
			key := strings.ToLower(mid)
			if _, ok := origByKey[key]; !ok {
				origByKey[key] = mid
			}
			totalByKey[key] += mq.Quantity
		}
	}

	out := make([]proddom.ModelTotalQuantity, 0, len(totalByKey))
	for k, total := range totalByKey {
		out = append(out, proddom.ModelTotalQuantity{
			ModelID:       origByKey[k],
			TotalQuantity: total,
		})
	}

	// stable order (case-insensitive by modelId)
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].ModelID) < strings.ToLower(out[j].ModelID)
	})

	return out, nil
}

// GetByModelID は、指定 modelId を Models に含む Production 一覧を返します。
// ※ RepositoryPort に無いが、既存利用が残っている可能性があるため維持。
func (r *ProductionRepositoryFS) GetByModelID(ctx context.Context, modelID string) ([]proddom.Production, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return []proddom.Production{}, nil
	}

	all, err := r.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	var out []proddom.Production
	for _, p := range all {
		for _, mq := range p.Models {
			if strings.TrimSpace(mq.ModelID) == modelID {
				out = append(out, p)
				break
			}
		}
	}
	return out, nil
}

// GetProductBlueprintIDByProductionID は productionId → productBlueprintId を返します。
func (r *ProductionRepositoryFS) GetProductBlueprintIDByProductionID(
	ctx context.Context,
	productionID string,
) (string, error) {
	p, err := r.GetByID(ctx, productionID)
	if err != nil {
		return "", err
	}
	if p == nil {
		return "", proddom.ErrNotFound
	}
	return strings.TrimSpace(p.ProductBlueprintID), nil
}

// ============================================================
// Mapping Helpers
// ============================================================

func docToProduction(doc *firestore.DocumentSnapshot) (proddom.Production, error) {
	var raw struct {
		ProductBlueprintID string                  `firestore:"productBlueprintId"`
		AssigneeID         string                  `firestore:"assigneeId"`
		Models             []proddom.ModelQuantity `firestore:"models"`

		Printed   *bool      `firestore:"printed"`
		PrintedAt *time.Time `firestore:"printedAt"`
		PrintedBy *string    `firestore:"printedBy"`

		CreatedBy *string    `firestore:"createdBy"`
		CreatedAt time.Time  `firestore:"createdAt"`
		UpdatedBy *string    `firestore:"updatedBy"`
		UpdatedAt *time.Time `firestore:"updatedAt"`
	}

	if err := doc.DataTo(&raw); err != nil {
		return proddom.Production{}, err
	}

	createdAt := raw.CreatedAt.UTC()

	printed := false
	if raw.Printed != nil {
		printed = *raw.Printed
	}

	printedAt := normalizeTimePtr(raw.PrintedAt)
	printedBy := fscommon.TrimPtr(raw.PrintedBy)

	// 整合性補正（ドメインルールに合わせる）
	if printed {
		if printedAt == nil {
			t := createdAt
			printedAt = &t
		}
	} else {
		printedAt = nil
		printedBy = nil
	}

	out := proddom.Production{
		ID:                 doc.Ref.ID,
		ProductBlueprintID: strings.TrimSpace(raw.ProductBlueprintID),
		AssigneeID:         strings.TrimSpace(raw.AssigneeID),
		Models:             raw.Models,

		Printed:   printed,
		PrintedAt: printedAt,
		PrintedBy: printedBy,

		CreatedBy: fscommon.TrimPtr(raw.CreatedBy),
		CreatedAt: createdAt,
		UpdatedBy: fscommon.TrimPtr(raw.UpdatedBy),
	}

	if raw.UpdatedAt != nil && !raw.UpdatedAt.IsZero() {
		out.UpdatedAt = raw.UpdatedAt.UTC()
	} else {
		out.UpdatedAt = out.CreatedAt
	}

	return out, nil
}

func productionToDoc(p proddom.Production) map[string]any {
	m := map[string]any{
		"productBlueprintId": strings.TrimSpace(p.ProductBlueprintID),
		"assigneeId":         strings.TrimSpace(p.AssigneeID),
		"models":             p.Models,

		"printed":   p.Printed,
		"createdAt": p.CreatedAt.UTC(),
		"updatedAt": p.UpdatedAt.UTC(),
	}

	if p.CreatedBy != nil {
		if s := strings.TrimSpace(*p.CreatedBy); s != "" {
			m["createdBy"] = s
		}
	}

	// printedAt / printedBy は printed=true のときだけ格納（false の場合は null を書いて消す）
	if p.Printed {
		if p.PrintedAt != nil && !p.PrintedAt.IsZero() {
			m["printedAt"] = p.PrintedAt.UTC()
		}
		if p.PrintedBy != nil {
			if s := strings.TrimSpace(*p.PrintedBy); s != "" {
				m["printedBy"] = s
			}
		}
	} else {
		m["printedAt"] = nil
		m["printedBy"] = nil
	}

	if p.UpdatedBy != nil {
		if s := strings.TrimSpace(*p.UpdatedBy); s != "" {
			m["updatedBy"] = s
		}
	}

	return m
}

func normalizeTimePtr(t *time.Time) *time.Time {
	if t == nil || t.IsZero() {
		return nil
	}
	tt := t.UTC()
	return &tt
}
