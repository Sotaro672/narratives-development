// backend/internal/adapters/out/firestore/list_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	gfs "cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
	ldom "narratives/internal/domain/list"
)

// ListRepositoryFS implements list.Repository using Firestore.
type ListRepositoryFS struct {
	Client *gfs.Client
}

func NewListRepositoryFS(client *gfs.Client) *ListRepositoryFS {
	return &ListRepositoryFS{Client: client}
}

func (r *ListRepositoryFS) col() *gfs.CollectionRef {
	return r.Client.Collection("lists")
}

const (
	listPricesSub = "prices" // subcollection under each list doc
)

// Compile-time check
var _ ldom.Repository = (*ListRepositoryFS)(nil)

// =======================
// Queries
// =======================

func (r *ListRepositoryFS) GetByID(ctx context.Context, id string) (ldom.List, error) {
	if r.Client == nil {
		return ldom.List{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return ldom.List{}, ldom.ErrNotFound
	}

	doc, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return ldom.List{}, ldom.ErrNotFound
		}
		return ldom.List{}, err
	}

	l, err := decodeListDoc(doc)
	if err != nil {
		return ldom.List{}, err
	}

	prices, err := r.loadListPricesForOne(ctx, l.ID)
	if err != nil {
		return ldom.List{}, err
	}
	l.Prices = prices

	return l, nil
}

func (r *ListRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
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

// Count: best-effort via scanning and applying Filter in-memory.
func (r *ListRepositoryFS) Count(ctx context.Context, filter ldom.Filter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	total := 0
	priceFilterNeeded := needsPriceFilter(filter)

	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return 0, err
		}

		l, err := decodeListDoc(doc)
		if err != nil {
			return 0, err
		}

		if !matchListFilterMeta(l, filter) {
			continue
		}

		if priceFilterNeeded {
			prices, err := r.loadListPricesForOne(ctx, l.ID)
			if err != nil {
				return 0, err
			}
			if !matchListFilterPrice(prices, filter) {
				continue
			}
		}

		total++
	}

	return total, nil
}

func (r *ListRepositoryFS) List(
	ctx context.Context,
	filter ldom.Filter,
	sortOpt ldom.Sort,
	page ldom.Page,
) (ldom.PageResult[ldom.List], error) {
	if r.Client == nil {
		return ldom.PageResult[ldom.List]{}, errors.New("firestore client is nil")
	}

	pageNum, perPage, _ := fscommon.NormalizePage(page.Number, page.PerPage, 50, 0)

	q := r.col().Query
	q = applyListSortToQuery(q, sortOpt)

	it := q.Documents(ctx)
	defer it.Stop()

	var all []ldom.List
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return ldom.PageResult[ldom.List]{}, err
		}

		l, err := decodeListDoc(doc)
		if err != nil {
			return ldom.PageResult[ldom.List]{}, err
		}

		if matchListFilterMeta(l, filter) {
			all = append(all, l)
		}
	}

	priceFilterNeeded := needsPriceFilter(filter)

	// Enrich with prices (needed both for response & price-based filtering)
	if err := r.enrichListsWithPrices(ctx, all); err != nil {
		return ldom.PageResult[ldom.List]{}, err
	}

	// Apply price-based filters if any
	if priceFilterNeeded {
		filtered := make([]ldom.List, 0, len(all))
		for _, l := range all {
			if matchListFilterPrice(l.Prices, filter) {
				filtered = append(filtered, l)
			}
		}
		all = filtered
	}

	total := len(all)
	if total == 0 {
		return ldom.PageResult[ldom.List]{
			Items:      []ldom.List{},
			TotalCount: 0,
			TotalPages: 0,
			Page:       pageNum,
			PerPage:    perPage,
		}, nil
	}

	offset := (pageNum - 1) * perPage
	if offset > total {
		offset = total
	}
	end := offset + perPage
	if end > total {
		end = total
	}
	items := all[offset:end]

	totalPages := fscommon.ComputeTotalPages(total, perPage)

	return ldom.PageResult[ldom.List]{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *ListRepositoryFS) ListByCursor(
	ctx context.Context,
	filter ldom.Filter,
	_ ldom.Sort,
	cpage ldom.CursorPage,
) (ldom.CursorPageResult[ldom.List], error) {
	if r.Client == nil {
		return ldom.CursorPageResult[ldom.List]{}, errors.New("firestore client is nil")
	}

	limit := cpage.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	// Cursor by DocumentID (id) ASC
	q := r.col().OrderBy(gfs.DocumentID, gfs.Asc)

	it := q.Documents(ctx)
	defer it.Stop()

	after := strings.TrimSpace(cpage.After)
	skipping := after != ""

	var (
		items []ldom.List
		last  string
	)

	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return ldom.CursorPageResult[ldom.List]{}, err
		}

		l, err := decodeListDoc(doc)
		if err != nil {
			return ldom.CursorPageResult[ldom.List]{}, err
		}

		if skipping {
			if l.ID <= after {
				continue
			}
			skipping = false
		}

		if !matchListFilterMeta(l, filter) {
			continue
		}

		items = append(items, l)
		last = l.ID

		if len(items) >= limit+1 {
			break
		}
	}

	// Enrich prices
	if err := r.enrichListsWithPrices(ctx, items); err != nil {
		return ldom.CursorPageResult[ldom.List]{}, err
	}

	// Apply price-filter if needed
	if needsPriceFilter(filter) {
		filtered := make([]ldom.List, 0, len(items))
		for _, l := range items {
			if matchListFilterPrice(l.Prices, filter) {
				filtered = append(filtered, l)
			}
		}
		items = filtered
	}

	var next *string
	if len(items) > limit {
		items = items[:limit]
		next = &last
	}

	return ldom.CursorPageResult[ldom.List]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

// =======================
// Mutations
// =======================

func (r *ListRepositoryFS) Create(ctx context.Context, l ldom.List) (ldom.List, error) {
	if r.Client == nil {
		return ldom.List{}, errors.New("firestore client is nil")
	}

	// ✅ ABテスト前提: 同一 inventoryId に複数 List を許容するため、
	// docId を inventoryId に固定しない。
	// - l.ID が空なら Firestore の自動採番で docId を発行する
	id := strings.TrimSpace(l.ID)

	now := time.Now().UTC()
	if l.CreatedAt.IsZero() {
		l.CreatedAt = now
	}
	if l.UpdatedAt == nil {
		l.UpdatedAt = &now
	}

	var ref *gfs.DocumentRef
	if id == "" {
		ref = r.col().NewDoc()
		l.ID = ref.ID
		id = ref.ID
	} else {
		ref = r.col().Doc(id)
		l.ID = id
	}

	// conflict check + create main doc + prices in a transaction
	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		// ID 指定の場合のみ conflict check
		if strings.TrimSpace(ref.ID) != "" && strings.TrimSpace(l.ID) != "" && strings.TrimSpace(l.ID) == strings.TrimSpace(ref.ID) {
			_, err := tx.Get(ref)
			if err == nil {
				return ldom.ErrConflict
			}
			if status.Code(err) != codes.NotFound {
				return err
			}
		}

		if err := tx.Create(ref, encodeListDoc(l)); err != nil {
			if status.Code(err) == codes.AlreadyExists {
				return ldom.ErrConflict
			}
			return err
		}

		return r.txReplaceListPrices(ctx, tx, ref, l.Prices)
	})
	if err != nil {
		if errors.Is(err, ldom.ErrConflict) {
			return ldom.List{}, ldom.ErrConflict
		}
		return ldom.List{}, err
	}

	return r.GetByID(ctx, id)
}

func (r *ListRepositoryFS) Update(
	ctx context.Context,
	id string,
	patch ldom.ListPatch,
) (ldom.List, error) {
	if r.Client == nil {
		return ldom.List{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return ldom.List{}, ldom.ErrNotFound
	}

	ref := r.col().Doc(id)

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		doc, err := tx.Get(ref)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return ldom.ErrNotFound
			}
			return err
		}

		cur, err := decodeListDoc(doc)
		if err != nil {
			return err
		}

		changed := false

		// Status
		if patch.Status != nil {
			cur.Status = *patch.Status
			changed = true
		}

		// AssigneeID
		if patch.AssigneeID != nil {
			v := strings.TrimSpace(*patch.AssigneeID)
			cur.AssigneeID = v
			changed = true
		}

		// ImageID
		if patch.ImageID != nil {
			v := strings.TrimSpace(*patch.ImageID)
			cur.ImageID = v
			changed = true
		}

		// Title
		if patch.Title != nil {
			cur.Title = strings.TrimSpace(*patch.Title)
			changed = true
		}

		// Description
		if patch.Description != nil {
			cur.Description = strings.TrimSpace(*patch.Description)
			changed = true
		}

		// UpdatedBy
		if patch.UpdatedBy != nil {
			v := strings.TrimSpace(*patch.UpdatedBy)
			if v == "" {
				cur.UpdatedBy = nil
			} else {
				cur.UpdatedBy = &v
			}
			changed = true
		}

		// DeletedAt
		if patch.DeletedAt != nil {
			if patch.DeletedAt.IsZero() {
				cur.DeletedAt = nil
			} else {
				t := patch.DeletedAt.UTC()
				cur.DeletedAt = &t
			}
			changed = true
		}

		// DeletedBy
		if patch.DeletedBy != nil {
			v := strings.TrimSpace(*patch.DeletedBy)
			if v == "" {
				cur.DeletedBy = nil
			} else {
				cur.DeletedBy = &v
			}
			changed = true
		}

		pricesWillChange := patch.Prices != nil

		// UpdatedAt: explicit or auto when changed
		if patch.UpdatedAt != nil {
			if patch.UpdatedAt.IsZero() {
				cur.UpdatedAt = nil
			} else {
				t := patch.UpdatedAt.UTC()
				cur.UpdatedAt = &t
			}
		} else if changed || pricesWillChange {
			t := time.Now().UTC()
			cur.UpdatedAt = &t
		}

		// persist main doc
		if changed || pricesWillChange {
			if err := tx.Set(ref, encodeListDoc(cur)); err != nil {
				return err
			}
		}

		// replace prices if provided
		if patch.Prices != nil {
			if err := r.txReplaceListPrices(ctx, tx, ref, *patch.Prices); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, ldom.ErrNotFound) {
			return ldom.List{}, ldom.ErrNotFound
		}
		return ldom.List{}, err
	}

	return r.GetByID(ctx, id)
}

func (r *ListRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return ldom.ErrNotFound
	}

	ref := r.col().Doc(id)

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		// ensure exists
		_, err := tx.Get(ref)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return ldom.ErrNotFound
			}
			return err
		}

		// delete subcollection prices
		it := ref.Collection(listPricesSub).Documents(ctx)
		for {
			doc, err := it.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				return err
			}
			if err := tx.Delete(doc.Ref); err != nil {
				return err
			}
		}

		// delete main doc
		return tx.Delete(ref)
	})
	if err != nil {
		if errors.Is(err, ldom.ErrNotFound) {
			return ldom.ErrNotFound
		}
		return err
	}
	return nil
}

func (r *ListRepositoryFS) Save(ctx context.Context, l ldom.List, _ *ldom.SaveOptions) (ldom.List, error) {
	if r.Client == nil {
		return ldom.List{}, errors.New("firestore client is nil")
	}

	id := strings.TrimSpace(l.ID)

	now := time.Now().UTC()
	if l.CreatedAt.IsZero() {
		l.CreatedAt = now
	}
	if l.UpdatedAt == nil {
		l.UpdatedAt = &now
	}

	var ref *gfs.DocumentRef
	if id == "" {
		ref = r.col().NewDoc()
		l.ID = ref.ID
		id = ref.ID
	} else {
		ref = r.col().Doc(id)
		l.ID = id
	}

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		if err := tx.Set(ref, encodeListDoc(l), gfs.MergeAll); err != nil {
			return err
		}
		return r.txReplaceListPrices(ctx, tx, ref, l.Prices)
	})
	if err != nil {
		return ldom.List{}, err
	}

	return r.GetByID(ctx, id)
}

// =======================
// Helpers - encode/decode
// =======================

func decodeListDoc(doc *gfs.DocumentSnapshot) (ldom.List, error) {
	var raw struct {
		Status      string     `firestore:"status"`
		AssigneeID  string     `firestore:"assignee_id"`
		Title       string     `firestore:"title"`
		ImageID     string     `firestore:"image_id"`
		Description *string    `firestore:"description"`
		CreatedBy   string     `firestore:"created_by"`
		CreatedAt   time.Time  `firestore:"created_at"`
		UpdatedBy   *string    `firestore:"updated_by"`
		UpdatedAt   *time.Time `firestore:"updated_at"`
		DeletedAt   *time.Time `firestore:"deleted_at"`
		DeletedBy   *string    `firestore:"deleted_by"`

		InventoryID string `firestore:"inventory_id"`
	}

	if err := doc.DataTo(&raw); err != nil {
		return ldom.List{}, err
	}

	id := strings.TrimSpace(doc.Ref.ID)

	desc := ""
	if raw.Description != nil {
		desc = strings.TrimSpace(*raw.Description)
	}

	var updatedBy *string
	if raw.UpdatedBy != nil {
		updatedBy = fscommon.TrimPtr(raw.UpdatedBy)
	}

	var updatedAt *time.Time
	if raw.UpdatedAt != nil && !raw.UpdatedAt.IsZero() {
		t := raw.UpdatedAt.UTC()
		updatedAt = &t
	}

	var deletedAt *time.Time
	if raw.DeletedAt != nil && !raw.DeletedAt.IsZero() {
		t := raw.DeletedAt.UTC()
		deletedAt = &t
	}

	var deletedBy *string
	if raw.DeletedBy != nil {
		deletedBy = fscommon.TrimPtr(raw.DeletedBy)
	}

	// backward compatible: camelCase で保存されている既存データも拾う
	invID := strings.TrimSpace(raw.InventoryID)
	if invID == "" {
		if m := doc.Data(); m != nil {
			for _, key := range []string{"inventoryId", "inventoryID", "inventory_id"} {
				if v, ok := m[key]; ok {
					if s, ok := v.(string); ok {
						invID = strings.TrimSpace(s)
						if invID != "" {
							break
						}
					}
				}
			}
		}
	}

	return ldom.List{
		ID:          id,
		Status:      ldom.ListStatus(strings.TrimSpace(raw.Status)),
		AssigneeID:  strings.TrimSpace(raw.AssigneeID),
		Title:       strings.TrimSpace(raw.Title),
		ImageID:     strings.TrimSpace(raw.ImageID),
		InventoryID: invID,

		Description: desc,
		Prices:      nil, // filled later

		CreatedBy: strings.TrimSpace(raw.CreatedBy),
		CreatedAt: raw.CreatedAt.UTC(),
		UpdatedBy: updatedBy,
		UpdatedAt: updatedAt,
		DeletedAt: deletedAt,
		DeletedBy: deletedBy,
	}, nil
}

func encodeListDoc(l ldom.List) map[string]any {
	m := map[string]any{
		"status":      strings.TrimSpace(string(l.Status)),
		"assignee_id": strings.TrimSpace(l.AssigneeID),
		"title":       strings.TrimSpace(l.Title),
		"image_id":    strings.TrimSpace(l.ImageID),
		"description": strings.TrimSpace(l.Description),
		"created_by":  strings.TrimSpace(l.CreatedBy),
		"created_at":  l.CreatedAt.UTC(),
	}

	// inventory_id を保存（空なら保存しない）
	if v := strings.TrimSpace(l.InventoryID); v != "" {
		m["inventory_id"] = v
	}

	if l.UpdatedBy != nil {
		if v := strings.TrimSpace(*l.UpdatedBy); v != "" {
			m["updated_by"] = v
		}
	}
	if l.UpdatedAt != nil && !l.UpdatedAt.IsZero() {
		m["updated_at"] = l.UpdatedAt.UTC()
	}
	if l.DeletedAt != nil && !l.DeletedAt.IsZero() {
		m["deleted_at"] = l.DeletedAt.UTC()
	}
	if l.DeletedBy != nil {
		if v := strings.TrimSpace(*l.DeletedBy); v != "" {
			m["deleted_by"] = v
		}
	}

	return m
}

// =======================
// Helpers - prices (✅ array only)
// =======================

func (r *ListRepositoryFS) enrichListsWithPrices(ctx context.Context, lists []ldom.List) error {
	for i := range lists {
		prices, err := r.loadListPricesForOne(ctx, lists[i].ID)
		if err != nil {
			return err
		}
		lists[i].Prices = prices
	}
	return nil
}

// loadListPricesForOne loads prices as []{modelId, price}.
// Backward compatible:
// - old docs might have "inventory_id" instead of "model_id" (we read both)
func (r *ListRepositoryFS) loadListPricesForOne(ctx context.Context, listID string) ([]ldom.ListPriceRow, error) {
	if strings.TrimSpace(listID) == "" {
		return nil, nil
	}

	// ✅ order by DocumentID to avoid schema-dependent OrderBy errors
	it := r.col().
		Doc(listID).
		Collection(listPricesSub).
		OrderBy(gfs.DocumentID, gfs.Asc).
		Documents(ctx)
	defer it.Stop()

	out := make([]ldom.ListPriceRow, 0, 8)

	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}

		// ✅ new: model_id
		// ✅ old: inventory_id (fallback)
		var raw struct {
			ModelID     string `firestore:"model_id"`
			InventoryID string `firestore:"inventory_id"`
			Price       int    `firestore:"price"`
		}
		if err := doc.DataTo(&raw); err != nil {
			return nil, err
		}

		modelID := strings.TrimSpace(raw.ModelID)
		if modelID == "" {
			modelID = strings.TrimSpace(raw.InventoryID) // backward compat
		}
		if modelID == "" {
			modelID = strings.TrimSpace(doc.Ref.ID) // final fallback
		}
		if modelID == "" {
			continue
		}

		out = append(out, ldom.ListPriceRow{
			ModelID: modelID,
			Price:   raw.Price,
		})
	}

	if len(out) == 0 {
		return nil, nil
	}

	// stable order
	sort.Slice(out, func(i, j int) bool {
		return strings.TrimSpace(out[i].ModelID) < strings.TrimSpace(out[j].ModelID)
	})

	return out, nil
}

func (r *ListRepositoryFS) txReplaceListPrices(
	ctx context.Context,
	tx *gfs.Transaction,
	listRef *gfs.DocumentRef,
	prices []ldom.ListPriceRow,
) error {
	// delete existing prices
	it := listRef.Collection(listPricesSub).Documents(ctx)
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		if err := tx.Delete(doc.Ref); err != nil {
			return err
		}
	}

	if len(prices) == 0 {
		return nil
	}

	np := normalizeListPrices(prices)
	if len(np) == 0 {
		return nil
	}

	// stable order by modelId
	keys := make([]string, 0, len(np))
	for k := range np {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, modelID := range keys {
		p := np[modelID]
		itemRef := listRef.Collection(listPricesSub).Doc(modelID)
		if err := tx.Set(itemRef, map[string]any{
			"model_id": modelID,
			"price":    p.Price,
		}); err != nil {
			return err
		}
	}

	return nil
}

func normalizeListPrices(in []ldom.ListPriceRow) map[string]ldom.ListPriceRow {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]ldom.ListPriceRow, len(in))
	for _, row := range in {
		mid := strings.TrimSpace(row.ModelID)
		if mid == "" {
			continue
		}
		// ドメイン側で価格制約がある前提だが、ここでも最低限の整形だけしておく
		out[mid] = ldom.ListPriceRow{
			ModelID: mid,
			Price:   row.Price,
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// =======================
// Helpers - filtering & sort
// =======================

func needsPriceFilter(f ldom.Filter) bool {
	// Filter の Price 条件は
	// - ModelNumbers (≒ modelId の集合)
	// - MinPrice / MaxPrice
	return len(f.ModelNumbers) > 0 || f.MinPrice != nil || f.MaxPrice != nil
}

// Filters that depend only on list document fields (no Prices).
func matchListFilterMeta(l ldom.List, f ldom.Filter) bool {
	// Free text search
	if sq := strings.TrimSpace(f.SearchQuery); sq != "" {
		lq := strings.ToLower(sq)
		haystack := strings.ToLower(
			l.ID + " " +
				l.Title + " " +
				l.Description + " " +
				l.ImageID + " " +
				l.AssigneeID + " " +
				l.CreatedBy + " " +
				ptrToString(l.UpdatedBy) + " " +
				ptrToString(l.DeletedBy),
		)
		if !strings.Contains(haystack, lq) {
			return false
		}
	}

	// IDs
	if len(f.IDs) > 0 {
		found := false
		for _, v := range f.IDs {
			if strings.TrimSpace(v) == l.ID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Assignee
	if f.AssigneeID != nil && strings.TrimSpace(*f.AssigneeID) != "" {
		if l.AssigneeID != strings.TrimSpace(*f.AssigneeID) {
			return false
		}
	}

	// Status
	if f.Status != nil && strings.TrimSpace(string(*f.Status)) != "" {
		if l.Status != *f.Status {
			return false
		}
	}
	if len(f.Statuses) > 0 {
		ok := false
		for _, st := range f.Statuses {
			if l.Status == st {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	// Date ranges
	if f.CreatedFrom != nil && l.CreatedAt.Before(f.CreatedFrom.UTC()) {
		return false
	}
	if f.CreatedTo != nil && !l.CreatedAt.Before(f.CreatedTo.UTC()) {
		return false
	}
	if f.UpdatedFrom != nil {
		if l.UpdatedAt == nil || l.UpdatedAt.Before(f.UpdatedFrom.UTC()) {
			return false
		}
	}
	if f.UpdatedTo != nil {
		if l.UpdatedAt == nil || !l.UpdatedAt.Before(f.UpdatedTo.UTC()) {
			return false
		}
	}
	if f.DeletedFrom != nil {
		if l.DeletedAt == nil || l.DeletedAt.Before(f.DeletedFrom.UTC()) {
			return false
		}
	}
	if f.DeletedTo != nil {
		if l.DeletedAt == nil || !l.DeletedAt.Before(f.DeletedTo.UTC()) {
			return false
		}
	}

	// Deleted tri-state
	if f.Deleted != nil {
		if *f.Deleted {
			if l.DeletedAt == nil {
				return false
			}
		} else {
			if l.DeletedAt != nil {
				return false
			}
		}
	}

	return true
}

// Price-based filters (EXISTS semantics).
// - prices: []{modelId, price}
// - f.ModelNumbers は「modelId の集合」として解釈する
func matchListFilterPrice(prices []ldom.ListPriceRow, f ldom.Filter) bool {
	if len(f.ModelNumbers) == 0 && f.MinPrice == nil && f.MaxPrice == nil {
		return true
	}
	if len(prices) == 0 {
		return false
	}

	allowed := map[string]struct{}{}
	if len(f.ModelNumbers) > 0 {
		for _, id := range f.ModelNumbers {
			id = strings.TrimSpace(id)
			if id != "" {
				allowed[id] = struct{}{}
			}
		}
	}

	for _, row := range prices {
		modelID := strings.TrimSpace(row.ModelID)
		if modelID == "" {
			continue
		}

		if len(allowed) > 0 {
			if _, ok := allowed[modelID]; !ok {
				continue
			}
		}

		if f.MinPrice != nil && row.Price < *f.MinPrice {
			continue
		}
		if f.MaxPrice != nil && row.Price > *f.MaxPrice {
			continue
		}

		// found one that matches all conditions
		return true
	}

	return false
}

func ptrToString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func applyListSortToQuery(q gfs.Query, sortOpt ldom.Sort) gfs.Query {
	field, dir := mapListSort(sortOpt)
	if field == "" {
		// Firestore can't do COALESCE; approximate:
		// primary sort by updated_at DESC (if present), then created_at DESC, then ID DESC.
		return q.
			OrderBy("updated_at", gfs.Desc).
			OrderBy("created_at", gfs.Desc).
			OrderBy(gfs.DocumentID, gfs.Desc)
	}
	return q.
		OrderBy(field, dir).
		OrderBy(gfs.DocumentID, gfs.Asc)
}

func mapListSort(sortOpt ldom.Sort) (string, gfs.Direction) {
	col := strings.ToLower(string(sortOpt.Column))
	var field string

	switch col {
	case "id":
		field = gfs.DocumentID
	case "status":
		field = "status"
	case "assigneeid", "assignee_id":
		field = "assignee_id"
	case "title":
		field = "title"
	case "imageid", "image_id", "imageurl", "image_url":
		field = "image_id"
	case "createdat", "created_at":
		field = "created_at"
	case "updatedat", "updated_at":
		field = "updated_at"
	case "deletedat", "deleted_at":
		field = "deleted_at"
	default:
		return "", gfs.Desc
	}

	dir := gfs.Asc
	if strings.EqualFold(string(sortOpt.Order), "desc") {
		dir = gfs.Desc
	}
	return field, dir
}
