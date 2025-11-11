// backend/internal/adapters/out/firestore/discount_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"strings"
	"time"

	gfs "cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ddom "narratives/internal/domain/discount"
)

// DiscountRepositoryFS is the Firestore implementation of the discount repository.
type DiscountRepositoryFS struct {
	Client *gfs.Client
}

func NewDiscountRepositoryFS(client *gfs.Client) *DiscountRepositoryFS {
	return &DiscountRepositoryFS{Client: client}
}

const (
	discountsCol      = "discounts"
	discountItemsSub  = "items" // subcollection under each discount doc
	defaultPerPage    = 50
	maxCursorPageSize = 200
)

// =======================
// Queries
// =======================

func (r *DiscountRepositoryFS) List(
	ctx context.Context,
	filter ddom.Filter,
	sort ddom.Sort,
	page ddom.Page,
) (ddom.PageResult[ddom.Discount], error) {
	if r.Client == nil {
		return ddom.PageResult[ddom.Discount]{}, errors.New("firestore client is nil")
	}

	if page.PerPage <= 0 {
		page.PerPage = defaultPerPage
	}
	if page.Number <= 0 {
		page.Number = 1
	}
	offset := (page.Number - 1) * page.PerPage

	q := r.Client.Collection(discountsCol).Query
	q = applyDiscountFilterToQuery(q, filter)
	q = applyDiscountSortToQuery(q, sort)
	q = q.Offset(offset).Limit(page.PerPage)

	iter := q.Documents(ctx)
	defer iter.Stop()

	var items []ddom.Discount
	for {
		doc, err := iter.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return ddom.PageResult[ddom.Discount]{}, err
		}
		d, err := decodeDiscountDoc(doc)
		if err != nil {
			return ddom.PageResult[ddom.Discount]{}, err
		}
		items = append(items, d)
	}

	// best-effort total count via separate Count()
	total, err := r.Count(ctx, filter)
	if err != nil {
		total = 0
	}

	// load items subcollection for each discount
	if err := r.enrichDiscountsWithItems(ctx, items); err != nil {
		return ddom.PageResult[ddom.Discount]{}, err
	}

	totalPages := 0
	if page.PerPage > 0 && total > 0 {
		totalPages = (total + page.PerPage - 1) / page.PerPage
	}

	return ddom.PageResult[ddom.Discount]{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       page.Number,
		PerPage:    page.PerPage,
	}, nil
}

func (r *DiscountRepositoryFS) ListByCursor(
	ctx context.Context,
	filter ddom.Filter,
	_ ddom.Sort,
	cpage ddom.CursorPage,
) (ddom.CursorPageResult[ddom.Discount], error) {
	if r.Client == nil {
		return ddom.CursorPageResult[ddom.Discount]{}, errors.New("firestore client is nil")
	}

	limit := cpage.Limit
	if limit <= 0 || limit > maxCursorPageSize {
		limit = defaultPerPage
	}

	q := r.Client.Collection(discountsCol).Query
	q = applyDiscountFilterToQuery(q, filter)
	// cursor pagination: order by DocumentID asc
	q = q.OrderBy(gfs.DocumentID, gfs.Asc)

	if after := strings.TrimSpace(cpage.After); after != "" {
		q = q.StartAfter(after)
	}

	q = q.Limit(limit + 1) // fetch one extra to detect next cursor

	iter := q.Documents(ctx)
	defer iter.Stop()

	var items []ddom.Discount
	var lastID string
	for {
		doc, err := iter.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return ddom.CursorPageResult[ddom.Discount]{}, err
		}
		d, err := decodeDiscountDoc(doc)
		if err != nil {
			return ddom.CursorPageResult[ddom.Discount]{}, err
		}
		items = append(items, d)
		lastID = d.ID
	}

	var next *string
	if len(items) > limit {
		items = items[:limit]
		next = &lastID
	}

	if err := r.enrichDiscountsWithItems(ctx, items); err != nil {
		return ddom.CursorPageResult[ddom.Discount]{}, err
	}

	return ddom.CursorPageResult[ddom.Discount]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

func (r *DiscountRepositoryFS) GetByID(ctx context.Context, id string) (ddom.Discount, error) {
	if r.Client == nil {
		return ddom.Discount{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return ddom.Discount{}, ddom.ErrNotFound
	}

	doc, err := r.Client.Collection(discountsCol).Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return ddom.Discount{}, ddom.ErrNotFound
		}
		return ddom.Discount{}, err
	}

	d, err := decodeDiscountDoc(doc)
	if err != nil {
		return ddom.Discount{}, err
	}

	items, err := r.loadDiscountItemsForOne(ctx, id)
	if err != nil {
		return ddom.Discount{}, err
	}
	d.Discounts = items

	return d, nil
}

func (r *DiscountRepositoryFS) GetByListID(
	ctx context.Context,
	listID string,
	sort ddom.Sort,
	page ddom.Page,
) (ddom.PageResult[ddom.Discount], error) {
	if r.Client == nil {
		return ddom.PageResult[ddom.Discount]{}, errors.New("firestore client is nil")
	}

	trimmed := strings.TrimSpace(listID)
	if trimmed == "" {
		return ddom.PageResult[ddom.Discount]{}, nil
	}
	f := ddom.Filter{ListID: &trimmed}
	return r.List(ctx, f, sort, page)
}

func (r *DiscountRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
	if r.Client == nil {
		return false, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}
	_, err := r.Client.Collection(discountsCol).Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Count: best-effort via scanning matching docs.
func (r *DiscountRepositoryFS) Count(ctx context.Context, filter ddom.Filter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("firestore client is nil")
	}

	q := r.Client.Collection(discountsCol).Query
	q = applyDiscountFilterToQuery(q, filter)

	iter := q.Documents(ctx)
	defer iter.Stop()

	total := 0
	for {
		_, err := iter.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return 0, err
		}
		total++
	}
	return total, nil
}

// =======================
// Mutations
// =======================

func (r *DiscountRepositoryFS) Create(ctx context.Context, in ddom.Discount) (ddom.Discount, error) {
	if r.Client == nil {
		return ddom.Discount{}, errors.New("firestore client is nil")
	}

	if strings.TrimSpace(in.ID) == "" {
		return ddom.Discount{}, errors.New("missing id")
	}
	now := time.Now().UTC()
	if in.DiscountedAt.IsZero() {
		in.DiscountedAt = now
	}
	if in.UpdatedAt.IsZero() {
		in.UpdatedAt = now
	}

	ref := r.Client.Collection(discountsCol).Doc(in.ID)

	// conflict check
	_, err := ref.Get(ctx)
	if err == nil {
		return ddom.Discount{}, ddom.ErrConflict
	}
	if err != nil && status.Code(err) != codes.NotFound {
		return ddom.Discount{}, err
	}

	data := encodeDiscountDoc(in)

	// transaction: create main doc + items
	err = r.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		if err := tx.Set(ref, data); err != nil {
			return err
		}
		return r.txReplaceDiscountItems(ctx, tx, ref, in.Discounts)
	})
	if err != nil {
		return ddom.Discount{}, err
	}

	return r.GetByID(ctx, in.ID)
}

func (r *DiscountRepositoryFS) Update(ctx context.Context, id string, patch ddom.DiscountPatch) (ddom.Discount, error) {
	if r.Client == nil {
		return ddom.Discount{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return ddom.Discount{}, ddom.ErrNotFound
	}

	ref := r.Client.Collection(discountsCol).Doc(id)

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		doc, err := tx.Get(ref)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return ddom.ErrNotFound
			}
			return err
		}

		cur, err := decodeDiscountDoc(doc)
		if err != nil {
			return err
		}

		// apply patch
		if patch.ListID != nil {
			cur.ListID = strings.TrimSpace(*patch.ListID)
		}
		if patch.Description != nil {
			if patch.Description == nil {
				cur.Description = nil
			} else {
				desc := strings.TrimSpace(*patch.Description)
				if desc == "" {
					cur.Description = nil
				} else {
					cur.Description = &desc
				}
			}
		}
		if patch.DiscountedBy != nil {
			cur.DiscountedBy = strings.TrimSpace(*patch.DiscountedBy)
		}
		if patch.DiscountedAt != nil {
			cur.DiscountedAt = patch.DiscountedAt.UTC()
		}
		if patch.UpdatedBy != nil {
			cur.UpdatedBy = strings.TrimSpace(*patch.UpdatedBy)
		}
		if patch.UpdatedAt != nil {
			cur.UpdatedAt = patch.UpdatedAt.UTC()
		} else {
			cur.UpdatedAt = time.Now().UTC()
		}

		if err := tx.Set(ref, encodeDiscountDoc(cur)); err != nil {
			return err
		}

		// replace items if provided
		if patch.Discounts != nil {
			if err := r.txReplaceDiscountItems(ctx, tx, ref, *patch.Discounts); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, ddom.ErrNotFound) {
			return ddom.Discount{}, ddom.ErrNotFound
		}
		return ddom.Discount{}, err
	}

	return r.GetByID(ctx, id)
}

func (r *DiscountRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return ddom.ErrNotFound
	}

	ref := r.Client.Collection(discountsCol).Doc(id)

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		// ensure exists
		_, err := tx.Get(ref)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return ddom.ErrNotFound
			}
			return err
		}

		// delete subcollection items
		iter := ref.Collection(discountItemsSub).Documents(ctx)
		for {
			doc, err := iter.Next()
			if err != nil {
				if errors.Is(err, iterator.Done) {
					break
				}
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
		if errors.Is(err, ddom.ErrNotFound) {
			return ddom.ErrNotFound
		}
		return err
	}
	return nil
}

// Save upserts a Discount and its items.
func (r *DiscountRepositoryFS) Save(ctx context.Context, d ddom.Discount) (ddom.Discount, error) {
	if r.Client == nil {
		return ddom.Discount{}, errors.New("firestore client is nil")
	}

	if strings.TrimSpace(d.ID) == "" {
		return ddom.Discount{}, errors.New("missing id")
	}
	now := time.Now().UTC()
	if d.DiscountedAt.IsZero() {
		d.DiscountedAt = now
	}
	if d.UpdatedAt.IsZero() {
		d.UpdatedAt = now
	}

	ref := r.Client.Collection(discountsCol).Doc(d.ID)

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		if err := tx.Set(ref, encodeDiscountDoc(d), gfs.MergeAll); err != nil {
			return err
		}
		return r.txReplaceDiscountItems(ctx, tx, ref, d.Discounts)
	})
	if err != nil {
		return ddom.Discount{}, err
	}

	return r.GetByID(ctx, d.ID)
}

// =======================
// Helpers
// =======================

func decodeDiscountDoc(doc *gfs.DocumentSnapshot) (ddom.Discount, error) {
	var raw struct {
		ListID       string    `firestore:"list_id"`
		Description  *string   `firestore:"description"`
		DiscountedBy string    `firestore:"discounted_by"`
		DiscountedAt time.Time `firestore:"discounted_at"`
		UpdatedBy    string    `firestore:"updated_by"`
		UpdatedAt    time.Time `firestore:"updated_at"`
	}
	if err := doc.DataTo(&raw); err != nil {
		return ddom.Discount{}, err
	}

	id := strings.TrimSpace(doc.Ref.ID)
	listID := strings.TrimSpace(raw.ListID)
	discBy := strings.TrimSpace(raw.DiscountedBy)
	updBy := strings.TrimSpace(raw.UpdatedBy)

	var descPtr *string
	if raw.Description != nil {
		v := strings.TrimSpace(*raw.Description)
		if v != "" {
			descPtr = &v
		}
	}

	return ddom.Discount{
		ID:           id,
		ListID:       listID,
		Description:  descPtr,
		DiscountedBy: discBy,
		DiscountedAt: raw.DiscountedAt.UTC(),
		UpdatedBy:    updBy,
		UpdatedAt:    raw.UpdatedAt.UTC(),
		Discounts:    nil, // filled later
	}, nil
}

func encodeDiscountDoc(d ddom.Discount) map[string]any {
	m := map[string]any{
		"list_id":       strings.TrimSpace(d.ListID),
		"discounted_by": strings.TrimSpace(d.DiscountedBy),
		"discounted_at": d.DiscountedAt.UTC(),
		"updated_by":    strings.TrimSpace(d.UpdatedBy),
		"updated_at":    d.UpdatedAt.UTC(),
	}
	if d.Description != nil {
		desc := strings.TrimSpace(*d.Description)
		if desc != "" {
			m["description"] = desc
		} else {
			m["description"] = gfs.Delete
		}
	}
	return m
}

func (r *DiscountRepositoryFS) enrichDiscountsWithItems(ctx context.Context, discounts []ddom.Discount) error {
	for i := range discounts {
		items, err := r.loadDiscountItemsForOne(ctx, discounts[i].ID)
		if err != nil {
			return err
		}
		discounts[i].Discounts = items
	}
	return nil
}

func (r *DiscountRepositoryFS) loadDiscountItemsForOne(ctx context.Context, discountID string) ([]ddom.DiscountItem, error) {
	if strings.TrimSpace(discountID) == "" {
		return nil, nil
	}

	iter := r.Client.Collection(discountsCol).
		Doc(discountID).
		Collection(discountItemsSub).
		OrderBy("model_number", gfs.Asc).
		Documents(ctx)
	defer iter.Stop()

	var out []ddom.DiscountItem
	for {
		doc, err := iter.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return nil, err
		}
		var raw struct {
			ModelNumber string `firestore:"model_number"`
			Percent     int    `firestore:"percent"`
		}
		if err := doc.DataTo(&raw); err != nil {
			return nil, err
		}
		out = append(out, ddom.DiscountItem{
			ModelNumber: strings.TrimSpace(raw.ModelNumber),
			Discount:    raw.Percent,
		})
	}
	return out, nil
}

func (r *DiscountRepositoryFS) txReplaceDiscountItems(
	ctx context.Context,
	tx *gfs.Transaction,
	discountRef *gfs.DocumentRef,
	items []ddom.DiscountItem,
) error {
	// delete existing items
	iter := discountRef.Collection(discountItemsSub).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return err
		}
		if err := tx.Delete(doc.Ref); err != nil {
			return err
		}
	}

	// insert new items
	for _, it := range items {
		mn := strings.TrimSpace(it.ModelNumber)
		if mn == "" {
			continue
		}
		itemRef := discountRef.Collection(discountItemsSub).Doc(mn)
		if err := tx.Set(itemRef, map[string]any{
			"model_number": mn,
			"percent":      it.Discount,
		}); err != nil {
			return err
		}
	}
	return nil
}

// =======================
// Filter & Sort helpers
// =======================

func applyDiscountFilterToQuery(q gfs.Query, f ddom.Filter) gfs.Query {
	// IDs via IN (<=10)
	if len(f.IDs) > 0 && len(f.IDs) <= 10 {
		ids := make([]string, 0, len(f.IDs))
		for _, id := range f.IDs {
			id = strings.TrimSpace(id)
			if id != "" {
				ids = append(ids, id)
			}
		}
		if len(ids) > 0 {
			q = q.Where(gfs.DocumentID, "in", ids)
		}
	}

	if f.ListID != nil && strings.TrimSpace(*f.ListID) != "" {
		q = q.Where("list_id", "==", strings.TrimSpace(*f.ListID))
	}

	if len(f.ListIDs) > 0 && len(f.ListIDs) <= 10 {
		listIDs := make([]string, 0, len(f.ListIDs))
		for _, v := range f.ListIDs {
			v = strings.TrimSpace(v)
			if v != "" {
				listIDs = append(listIDs, v)
			}
		}
		if len(listIDs) > 0 {
			q = q.Where("list_id", "in", listIDs)
		}
	}

	if f.DiscountedBy != nil && strings.TrimSpace(*f.DiscountedBy) != "" {
		q = q.Where("discounted_by", "==", strings.TrimSpace(*f.DiscountedBy))
	}
	if f.UpdatedBy != nil && strings.TrimSpace(*f.UpdatedBy) != "" {
		q = q.Where("updated_by", "==", strings.TrimSpace(*f.UpdatedBy))
	}

	if f.DiscountedFrom != nil {
		q = q.Where("discounted_at", ">=", f.DiscountedFrom.UTC())
	}
	if f.DiscountedTo != nil {
		q = q.Where("discounted_at", "<", f.DiscountedTo.UTC())
	}
	if f.UpdatedFrom != nil {
		q = q.Where("updated_at", ">=", f.UpdatedFrom.UTC())
	}
	if f.UpdatedTo != nil {
		q = q.Where("updated_at", "<", f.UpdatedTo.UTC())
	}

	// Complex filters (SearchQuery, ModelNumbers, PercentMin/Max) are not pushed down here.

	return q
}

func applyDiscountSortToQuery(q gfs.Query, sort ddom.Sort) gfs.Query {
	col := strings.ToLower(string(sort.Column))
	var field string
	switch col {
	case "id":
		field = gfs.DocumentID
	case "listid", "list_id":
		field = "list_id"
	case "discountedby", "discounted_by":
		field = "discounted_by"
	case "discountedat", "discounted_at":
		field = "discounted_at"
	case "updatedby", "updated_by":
		field = "updated_by"
	case "updatedat", "updated_at":
		field = "updated_at"
	case "description":
		field = "description"
	default:
		field = "updated_at"
	}

	dir := strings.ToUpper(string(sort.Order))
	if dir == "DESC" {
		return q.OrderBy(field, gfs.Desc)
	}
	return q.OrderBy(field, gfs.Asc)
}
