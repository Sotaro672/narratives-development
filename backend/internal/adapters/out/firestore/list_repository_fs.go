package firestore

import (
	"context"
	"errors"
	"sort"
	"time"

	gfs "cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
	ldom "narratives/internal/domain/list"
)

// ListRepositoryFS implements list.Repository using Firestore.
//
// Primary image policy:
// - lists/{listId}.image_id stores primary imageId, which is a Firestore docID.
// - Image URLs are derived from /lists/{listId}/images subcollection records by query layer.
// - image_id is not a URL.
//
// Delete policy:
// - Delete physically deletes the list document.
// - deleted_at / deleted_by are not used.
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
	listPricesSub = "prices"
)

var _ ldom.Repository = (*ListRepositoryFS)(nil)

// ============================================================
// Queries
// ============================================================

func (r *ListRepositoryFS) GetByID(ctx context.Context, id string) (ldom.List, error) {
	if r.Client == nil {
		return ldom.List{}, errors.New("firestore client is nil")
	}

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

func (r *ListRepositoryFS) Count(ctx context.Context, _ ldom.Filter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("firestore client is nil")
	}

	aq := r.col().Query.NewAggregationQuery().WithCount("all")
	res, err := aq.Get(ctx)
	if err != nil {
		return 0, err
	}

	v, ok := res["all"]
	if !ok {
		return 0, nil
	}

	switch n := v.(type) {
	case int64:
		return int(n), nil
	case int:
		return n, nil
	default:
		return 0, nil
	}
}

func (r *ListRepositoryFS) GetReadableIDByID(ctx context.Context, id string) (string, error) {
	if r.Client == nil {
		return "", errors.New("firestore client is nil")
	}

	if id == "" {
		return "", ldom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", ldom.ErrNotFound
		}
		return "", err
	}

	if data := snap.Data(); data != nil {
		if v, ok := data["readable_id"].(string); ok {
			return v, nil
		}
	}

	l, err := decodeListDoc(snap)
	if err != nil {
		return "", err
	}

	return l.ReadableID, nil
}

func (r *ListRepositoryFS) ListIDsByInventoryID(ctx context.Context, inventoryID string) ([]string, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	if inventoryID == "" {
		return []string{}, nil
	}

	it := r.col().
		Where("inventory_id", "==", inventoryID).
		Documents(ctx)
	defer it.Stop()

	ids := make([]string, 0, 8)

	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		if doc == nil || doc.Ref == nil || doc.Ref.ID == "" {
			continue
		}

		ids = append(ids, doc.Ref.ID)
	}

	sort.Strings(ids)
	return ids, nil
}

func (r *ListRepositoryFS) List(
	ctx context.Context,
	_ ldom.Filter,
	_ ldom.Sort,
	page ldom.Page,
) (ldom.PageResult[ldom.List], error) {
	if r.Client == nil {
		return ldom.PageResult[ldom.List]{}, errors.New("firestore client is nil")
	}

	pageNum, perPage, _ := fscommon.NormalizePage(page.Number, page.PerPage, 50, 0)
	if perPage <= 0 {
		perPage = 50
	}
	if pageNum <= 0 {
		pageNum = 1
	}

	offset := (pageNum - 1) * perPage
	if offset < 0 {
		offset = 0
	}

	q := r.col().Query.
		OrderBy("updated_at", gfs.Desc).
		OrderBy("created_at", gfs.Desc).
		OrderBy(gfs.DocumentID, gfs.Desc).
		Offset(offset).
		Limit(perPage)

	it := q.Documents(ctx)
	defer it.Stop()

	items := make([]ldom.List, 0, perPage)

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

		items = append(items, l)
	}

	if err := r.enrichListsWithPrices(ctx, items); err != nil {
		return ldom.PageResult[ldom.List]{}, err
	}

	return ldom.PageResult[ldom.List]{
		Items:      items,
		TotalCount: 0,
		TotalPages: 0,
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *ListRepositoryFS) ListByCursor(
	ctx context.Context,
	_ ldom.Filter,
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

	q := r.col().OrderBy(gfs.DocumentID, gfs.Asc)

	it := q.Documents(ctx)
	defer it.Stop()

	after := cpage.After
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

		items = append(items, l)
		last = l.ID

		if len(items) >= limit+1 {
			break
		}
	}

	if err := r.enrichListsWithPrices(ctx, items); err != nil {
		return ldom.CursorPageResult[ldom.List]{}, err
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

// ============================================================
// Mutations
// ============================================================

func (r *ListRepositoryFS) Create(ctx context.Context, l ldom.List) (ldom.List, error) {
	if r.Client == nil {
		return ldom.List{}, errors.New("firestore client is nil")
	}

	id := l.ID

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
		if ref.ID != "" && l.ID != "" && l.ID == ref.ID {
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
		clearUpdatedBy := false
		clearUpdatedAt := false

		if patch.Status != nil {
			cur.Status = *patch.Status
			changed = true
		}

		if patch.AssigneeID != nil {
			cur.AssigneeID = *patch.AssigneeID
			changed = true
		}

		if patch.ImageID != nil {
			cur.ImageID = *patch.ImageID
			changed = true
		}

		if patch.ReadableID != nil {
			cur.ReadableID = *patch.ReadableID
			changed = true
		}

		if patch.Title != nil {
			cur.Title = *patch.Title
			changed = true
		}

		if patch.Description != nil {
			cur.Description = *patch.Description
			changed = true
		}

		if patch.UpdatedBy != nil {
			v := *patch.UpdatedBy
			if v == "" {
				cur.UpdatedBy = nil
				clearUpdatedBy = true
			} else {
				cur.UpdatedBy = &v
			}
			changed = true
		}

		pricesWillChange := patch.Prices != nil

		if patch.UpdatedAt != nil {
			if patch.UpdatedAt.IsZero() {
				cur.UpdatedAt = nil
				clearUpdatedAt = true
			} else {
				t := patch.UpdatedAt.UTC()
				cur.UpdatedAt = &t
			}
		} else if changed || pricesWillChange {
			t := time.Now().UTC()
			cur.UpdatedAt = &t
		}

		if changed || pricesWillChange {
			data := encodeListDoc(cur)

			if clearUpdatedBy {
				data["updated_by"] = gfs.Delete
			}
			if clearUpdatedAt {
				data["updated_at"] = gfs.Delete
			}

			if err := tx.Set(ref, data, gfs.MergeAll); err != nil {
				return err
			}
		}

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

	if id == "" {
		return ldom.ErrNotFound
	}

	ref := r.col().Doc(id)

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		_, err := tx.Get(ref)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return ldom.ErrNotFound
			}
			return err
		}

		it := ref.Collection(listPricesSub).Documents(ctx)
		defer it.Stop()

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

		itImages := ref.Collection("images").Documents(ctx)
		defer itImages.Stop()

		for {
			doc, err := itImages.Next()
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

// ============================================================
// Helpers - encode/decode
// ============================================================

func decodeListDoc(doc *gfs.DocumentSnapshot) (ldom.List, error) {
	var raw struct {
		Status      string     `firestore:"status"`
		AssigneeID  string     `firestore:"assignee_id"`
		Title       string     `firestore:"title"`
		ImageID     string     `firestore:"image_id"`
		ReadableID  string     `firestore:"readable_id"`
		Description *string    `firestore:"description"`
		CreatedBy   string     `firestore:"created_by"`
		CreatedAt   time.Time  `firestore:"created_at"`
		UpdatedBy   *string    `firestore:"updated_by"`
		UpdatedAt   *time.Time `firestore:"updated_at"`

		InventoryID string `firestore:"inventory_id"`
	}

	if err := doc.DataTo(&raw); err != nil {
		return ldom.List{}, err
	}

	id := doc.Ref.ID

	desc := ""
	if raw.Description != nil {
		desc = *raw.Description
	}

	updatedBy := raw.UpdatedBy

	var updatedAt *time.Time
	if raw.UpdatedAt != nil && !raw.UpdatedAt.IsZero() {
		t := raw.UpdatedAt.UTC()
		updatedAt = &t
	}

	return ldom.List{
		ID:          id,
		Status:      ldom.ListStatus(raw.Status),
		AssigneeID:  raw.AssigneeID,
		Title:       raw.Title,
		ImageID:     raw.ImageID,
		InventoryID: raw.InventoryID,
		ReadableID:  raw.ReadableID,

		Description: desc,
		Prices:      nil,

		CreatedBy: raw.CreatedBy,
		CreatedAt: raw.CreatedAt.UTC(),
		UpdatedBy: updatedBy,
		UpdatedAt: updatedAt,
	}, nil
}

func encodeListDoc(l ldom.List) map[string]any {
	m := map[string]any{
		"status":      string(l.Status),
		"assignee_id": l.AssigneeID,
		"title":       l.Title,
		"image_id":    l.ImageID,
		"description": l.Description,
		"created_by":  l.CreatedBy,
		"created_at":  l.CreatedAt.UTC(),
	}

	if v := l.InventoryID; v != "" {
		m["inventory_id"] = v
	}
	if v := l.ReadableID; v != "" {
		m["readable_id"] = v
	}

	if l.UpdatedBy != nil {
		if v := *l.UpdatedBy; v != "" {
			m["updated_by"] = v
		}
	}
	if l.UpdatedAt != nil && !l.UpdatedAt.IsZero() {
		m["updated_at"] = l.UpdatedAt.UTC()
	}

	return m
}

// ============================================================
// Helpers - prices
// ============================================================

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

func (r *ListRepositoryFS) loadListPricesForOne(ctx context.Context, listID string) ([]ldom.ListPriceRow, error) {
	if listID == "" {
		return nil, nil
	}

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

		var raw struct {
			ModelID string `firestore:"model_id"`
			Price   int    `firestore:"price"`
		}

		if err := doc.DataTo(&raw); err != nil {
			return nil, err
		}

		modelID := raw.ModelID
		if modelID == "" {
			modelID = doc.Ref.ID
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

	sort.Slice(out, func(i, j int) bool {
		return out[i].ModelID < out[j].ModelID
	})

	return out, nil
}

func (r *ListRepositoryFS) txReplaceListPrices(
	ctx context.Context,
	tx *gfs.Transaction,
	listRef *gfs.DocumentRef,
	prices []ldom.ListPriceRow,
) error {
	it := listRef.Collection(listPricesSub).Documents(ctx)
	defer it.Stop()

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
		mid := row.ModelID
		if mid == "" {
			continue
		}

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
