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

const listPricesSub = "prices"

var _ ldom.Repository = (*ListRepositoryFS)(nil)

// ============================================================
// Queries
// ============================================================

func (r *ListRepositoryFS) GetByID(ctx context.Context, id string) (ldom.List, error) {
	if r == nil || r.Client == nil {
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

	if err := l.ValidateForPersist(); err != nil {
		return ldom.List{}, err
	}

	return l, nil
}

func (r *ListRepositoryFS) GetReadableIDByID(ctx context.Context, id string) (string, error) {
	if r == nil || r.Client == nil {
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

	l, err := decodeListDoc(snap)
	if err != nil {
		return "", err
	}

	return l.ReadableID, nil
}

func (r *ListRepositoryFS) ListByInventoryID(ctx context.Context, inventoryID string) ([]ldom.List, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	if inventoryID == "" {
		return []ldom.List{}, nil
	}

	it := r.col().
		Where("inventory_id", "==", inventoryID).
		Documents(ctx)
	defer it.Stop()

	items := make([]ldom.List, 0, 8)

	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		if doc == nil || doc.Ref == nil || doc.Ref.ID == "" {
			return nil, ldom.ErrInvalidID
		}

		l, err := decodeListDoc(doc)
		if err != nil {
			return nil, err
		}

		prices, err := r.loadListPricesForOne(ctx, l.ID)
		if err != nil {
			return nil, err
		}

		l.Prices = prices

		if err := l.ValidateForPersist(); err != nil {
			return nil, err
		}

		items = append(items, l)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})

	return items, nil
}

func (r *ListRepositoryFS) List(
	ctx context.Context,
	filter ldom.Filter,
	sortSpec ldom.Sort,
	page ldom.Page,
) (ldom.PageResult[ldom.List], error) {
	if r == nil || r.Client == nil {
		return ldom.PageResult[ldom.List]{}, errors.New("firestore client is nil")
	}

	pageNum, perPage, _ := fscommon.NormalizePage(
		page.Number,
		page.PerPage,
		50,
		0,
	)

	q := applyListQueryNarrowing(
		r.col().Query,
		filter,
	)

	it := q.Documents(ctx)
	defer it.Stop()

	items := make([]ldom.List, 0)
	priceFilterEnabled := hasListPriceFilter(filter)

	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return ldom.PageResult[ldom.List]{}, err
		}

		item, err := decodeListDoc(doc)
		if err != nil {
			return ldom.PageResult[ldom.List]{}, err
		}

		if !matchesListMetadataFilter(item, filter) {
			continue
		}

		if priceFilterEnabled {
			prices, err := r.loadListPricesForOne(ctx, item.ID)
			if err != nil {
				return ldom.PageResult[ldom.List]{}, err
			}

			item.Prices = prices

			if !matchesListPriceFilter(item.Prices, filter) {
				continue
			}
		}

		items = append(items, item)
	}

	sortLists(
		items,
		sortSpec,
		"updatedAt",
		ldom.SortDesc,
	)

	totalCount := len(items)
	totalPages := fscommon.ComputeTotalPages(
		totalCount,
		perPage,
	)

	offset := (pageNum - 1) * perPage
	if offset < 0 {
		offset = 0
	}

	if offset >= totalCount {
		return ldom.PageResult[ldom.List]{
			Items:      []ldom.List{},
			TotalCount: totalCount,
			TotalPages: totalPages,
			Page:       pageNum,
			PerPage:    perPage,
		}, nil
	}

	end := offset + perPage
	if end > totalCount {
		end = totalCount
	}

	pageItems := append(
		[]ldom.List(nil),
		items[offset:end]...,
	)

	if !priceFilterEnabled {
		if err := r.hydrateListPrices(ctx, pageItems); err != nil {
			return ldom.PageResult[ldom.List]{}, err
		}
	}

	for i := range pageItems {
		if err := pageItems[i].ValidateForPersist(); err != nil {
			return ldom.PageResult[ldom.List]{}, err
		}
	}

	return ldom.PageResult[ldom.List]{
		Items:      pageItems,
		TotalCount: totalCount,
		TotalPages: totalPages,
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *ListRepositoryFS) ListByCursor(
	ctx context.Context,
	filter ldom.Filter,
	sortSpec ldom.Sort,
	cpage ldom.CursorPage,
) (ldom.CursorPageResult[ldom.List], error) {
	if r == nil || r.Client == nil {
		return ldom.CursorPageResult[ldom.List]{}, errors.New("firestore client is nil")
	}

	limit := cpage.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	if canUseListCursorFastPath(filter, sortSpec) {
		return r.listByCursorFastPath(
			ctx,
			filter,
			cpage.After,
			limit,
		)
	}

	return r.listByCursorFiltered(
		ctx,
		filter,
		sortSpec,
		cpage.After,
		limit,
	)
}

func (r *ListRepositoryFS) listByCursorFastPath(
	ctx context.Context,
	filter ldom.Filter,
	after string,
	limit int,
) (ldom.CursorPageResult[ldom.List], error) {
	q := r.col().Query

	if filter.Status != nil {
		q = q.Where(
			"status",
			"==",
			string(*filter.Status),
		)
	} else if len(filter.Statuses) == 1 {
		q = q.Where(
			"status",
			"==",
			string(filter.Statuses[0]),
		)
	}

	q = q.OrderBy(gfs.DocumentID, gfs.Asc)

	if after != "" {
		q = q.StartAfter(after)
	}

	it := q.Limit(limit + 1).Documents(ctx)
	defer it.Stop()

	items := make([]ldom.List, 0, limit+1)

	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return ldom.CursorPageResult[ldom.List]{}, err
		}

		item, err := decodeListDoc(doc)
		if err != nil {
			return ldom.CursorPageResult[ldom.List]{}, err
		}

		if !matchesListMetadataFilter(item, filter) {
			continue
		}

		items = append(items, item)
	}

	var next *string

	if len(items) > limit {
		items = items[:limit]

		cursor := items[len(items)-1].ID
		next = &cursor
	}

	if err := r.hydrateListPrices(ctx, items); err != nil {
		return ldom.CursorPageResult[ldom.List]{}, err
	}

	for i := range items {
		if err := items[i].ValidateForPersist(); err != nil {
			return ldom.CursorPageResult[ldom.List]{}, err
		}
	}

	return ldom.CursorPageResult[ldom.List]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

func (r *ListRepositoryFS) listByCursorFiltered(
	ctx context.Context,
	filter ldom.Filter,
	sortSpec ldom.Sort,
	after string,
	limit int,
) (ldom.CursorPageResult[ldom.List], error) {
	q := applyListQueryNarrowing(
		r.col().Query,
		filter,
	)

	it := q.Documents(ctx)
	defer it.Stop()

	items := make([]ldom.List, 0)
	priceFilterEnabled := hasListPriceFilter(filter)

	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return ldom.CursorPageResult[ldom.List]{}, err
		}

		item, err := decodeListDoc(doc)
		if err != nil {
			return ldom.CursorPageResult[ldom.List]{}, err
		}

		if !matchesListMetadataFilter(item, filter) {
			continue
		}

		if priceFilterEnabled {
			prices, err := r.loadListPricesForOne(ctx, item.ID)
			if err != nil {
				return ldom.CursorPageResult[ldom.List]{}, err
			}

			item.Prices = prices

			if !matchesListPriceFilter(item.Prices, filter) {
				continue
			}
		}

		items = append(items, item)
	}

	sortLists(
		items,
		sortSpec,
		"id",
		ldom.SortAsc,
	)

	start := findListCursorStart(
		items,
		after,
		sortSpec,
	)

	if start >= len(items) {
		return ldom.CursorPageResult[ldom.List]{
			Items:      []ldom.List{},
			NextCursor: nil,
			Limit:      limit,
		}, nil
	}

	end := start + limit + 1
	if end > len(items) {
		end = len(items)
	}

	pageItems := append(
		[]ldom.List(nil),
		items[start:end]...,
	)

	var next *string

	if len(pageItems) > limit {
		pageItems = pageItems[:limit]

		cursor := pageItems[len(pageItems)-1].ID
		next = &cursor
	}

	if !priceFilterEnabled {
		if err := r.hydrateListPrices(ctx, pageItems); err != nil {
			return ldom.CursorPageResult[ldom.List]{}, err
		}
	}

	for i := range pageItems {
		if err := pageItems[i].ValidateForPersist(); err != nil {
			return ldom.CursorPageResult[ldom.List]{}, err
		}
	}

	return ldom.CursorPageResult[ldom.List]{
		Items:      pageItems,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

// ============================================================
// Mutations
// ============================================================

func (r *ListRepositoryFS) Create(ctx context.Context, l ldom.List) (ldom.List, error) {
	if r == nil || r.Client == nil {
		return ldom.List{}, errors.New("firestore client is nil")
	}

	id := l.ID
	now := time.Now().UTC()

	if l.CreatedAt.IsZero() {
		l.CreatedAt = now
	}

	if l.UpdatedAt == nil {
		t := now
		l.UpdatedAt = &t
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

	if err := l.ValidateForPersist(); err != nil {
		return ldom.List{}, err
	}

	if err := validateUniqueListPriceModelIDs(l.Prices); err != nil {
		return ldom.List{}, err
	}

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		_, err := tx.Get(ref)
		if err == nil {
			return ldom.ErrConflict
		}

		if status.Code(err) != codes.NotFound {
			return err
		}

		if err := tx.Create(ref, encodeListDoc(l)); err != nil {
			if status.Code(err) == codes.AlreadyExists {
				return ldom.ErrConflict
			}
			return err
		}

		return r.txReplaceListPrices(
			ctx,
			tx,
			ref,
			l.Prices,
		)
	})
	if err != nil {
		if errors.Is(err, ldom.ErrConflict) {
			return ldom.List{}, ldom.ErrConflict
		}
		return ldom.List{}, err
	}

	return r.GetByID(ctx, id)
}

func (r *ListRepositoryFS) Update(ctx context.Context, id string, l ldom.List) (ldom.List, error) {
	if r == nil || r.Client == nil {
		return ldom.List{}, errors.New("firestore client is nil")
	}
	if id == "" {
		return ldom.List{}, ldom.ErrNotFound
	}
	if l.ID != "" && l.ID != id {
		return ldom.List{}, ldom.ErrInvalidID
	}
	if err := validateUniqueListPriceModelIDs(l.Prices); err != nil {
		return ldom.List{}, err
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

		cur.Status = l.Status
		cur.AssigneeID = l.AssigneeID
		cur.Title = l.Title
		cur.ImageID = l.ImageID
		cur.InventoryID = l.InventoryID
		cur.ReadableID = l.ReadableID
		cur.Description = l.Description
		cur.Prices = l.Prices

		clearUpdatedBy := false
		clearUpdatedAt := false

		if l.UpdatedBy != nil {
			v := *l.UpdatedBy

			if v == "" {
				cur.UpdatedBy = nil
				clearUpdatedBy = true
			} else {
				cur.UpdatedBy = &v
			}
		}

		if l.UpdatedAt != nil {
			if l.UpdatedAt.IsZero() {
				cur.UpdatedAt = nil
				clearUpdatedAt = true
			} else {
				t := l.UpdatedAt.UTC()
				cur.UpdatedAt = &t
			}
		} else {
			t := time.Now().UTC()
			cur.UpdatedAt = &t
		}

		if err := cur.ValidateForPersist(); err != nil {
			return err
		}

		data := encodeListDoc(cur)

		// Transportation configuration has been fully migrated to inventory.
		data["transportation_option"] = gfs.Delete
		data["transportation_id"] = gfs.Delete

		if clearUpdatedBy {
			data["updated_by"] = gfs.Delete
		}

		if clearUpdatedAt {
			data["updated_at"] = gfs.Delete
		}

		if err := tx.Set(ref, data, gfs.MergeAll); err != nil {
			return err
		}

		return r.txReplaceListPrices(
			ctx,
			tx,
			ref,
			l.Prices,
		)
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
	if r == nil || r.Client == nil {
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
			if doc == nil || doc.Ref == nil {
				return ldom.ErrInvalidPrices
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
			if doc == nil || doc.Ref == nil {
				return ldom.ErrInvalidID
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
// Helpers - query
// ============================================================

func applyListQueryNarrowing(
	q gfs.Query,
	filter ldom.Filter,
) gfs.Query {
	if len(filter.IDs) == 1 {
		return q.Where(
			gfs.DocumentID,
			"==",
			filter.IDs[0],
		)
	}

	if len(filter.InventoryIDs) == 1 {
		return q.Where(
			"inventory_id",
			"==",
			filter.InventoryIDs[0],
		)
	}

	if len(filter.ReadableIDs) == 1 {
		return q.Where(
			"readable_id",
			"==",
			filter.ReadableIDs[0],
		)
	}

	if filter.AssigneeID != nil && *filter.AssigneeID != "" {
		return q.Where(
			"assignee_id",
			"==",
			*filter.AssigneeID,
		)
	}

	if filter.Status != nil {
		return q.Where(
			"status",
			"==",
			string(*filter.Status),
		)
	}

	if len(filter.Statuses) == 1 {
		return q.Where(
			"status",
			"==",
			string(filter.Statuses[0]),
		)
	}

	return q
}

func matchesListMetadataFilter(
	item ldom.List,
	filter ldom.Filter,
) bool {
	if len(filter.IDs) > 0 &&
		!listStringIn(item.ID, filter.IDs) {
		return false
	}

	if len(filter.ReadableIDs) > 0 &&
		!listStringIn(item.ReadableID, filter.ReadableIDs) {
		return false
	}

	if filter.AssigneeID != nil &&
		item.AssigneeID != *filter.AssigneeID {
		return false
	}

	if filter.Status != nil &&
		item.Status != *filter.Status {
		return false
	}

	if len(filter.Statuses) > 0 &&
		!listStatusIn(item.Status, filter.Statuses) {
		return false
	}

	if len(filter.InventoryIDs) > 0 &&
		!listStringIn(item.InventoryID, filter.InventoryIDs) {
		return false
	}

	searchQuery := strings.ToLower(
		strings.TrimSpace(filter.SearchQuery),
	)

	if searchQuery != "" {
		haystack := strings.ToLower(
			strings.Join(
				[]string{
					item.ID,
					item.ReadableID,
					item.Title,
					item.Description,
					item.AssigneeID,
					item.InventoryID,
					string(item.Status),
				},
				" ",
			),
		)

		if !strings.Contains(
			haystack,
			searchQuery,
		) {
			return false
		}
	}

	return true
}

func matchesListPriceFilter(
	prices []ldom.ListPriceRow,
	filter ldom.Filter,
) bool {
	if !hasListPriceFilter(filter) {
		return true
	}

	for _, row := range prices {
		if len(filter.ModelIDs) > 0 &&
			!listStringIn(row.ModelID, filter.ModelIDs) {
			continue
		}

		if filter.MinPrice != nil &&
			row.Price < *filter.MinPrice {
			continue
		}

		if filter.MaxPrice != nil &&
			row.Price > *filter.MaxPrice {
			continue
		}

		return true
	}

	return false
}

func hasListPriceFilter(filter ldom.Filter) bool {
	return len(filter.ModelIDs) > 0 ||
		filter.MinPrice != nil ||
		filter.MaxPrice != nil
}

func canUseListCursorFastPath(
	filter ldom.Filter,
	sortSpec ldom.Sort,
) bool {
	if filter.SearchQuery != "" ||
		len(filter.IDs) > 0 ||
		len(filter.ReadableIDs) > 0 ||
		filter.AssigneeID != nil ||
		len(filter.ModelIDs) > 0 ||
		filter.MinPrice != nil ||
		filter.MaxPrice != nil ||
		len(filter.InventoryIDs) > 0 {
		return false
	}

	if len(filter.Statuses) > 1 {
		return false
	}

	column := strings.TrimSpace(sortSpec.Column)
	if column != "" &&
		column != "id" {
		return false
	}

	if sortSpec.Order != "" &&
		sortSpec.Order != ldom.SortAsc {
		return false
	}

	return true
}

func findListCursorStart(
	items []ldom.List,
	after string,
	sortSpec ldom.Sort,
) int {
	if after == "" {
		return 0
	}

	for i := range items {
		if items[i].ID == after {
			return i + 1
		}
	}

	column := strings.TrimSpace(sortSpec.Column)
	order := sortSpec.Order

	if column == "" {
		column = "id"
	}

	if order == "" {
		order = ldom.SortAsc
	}

	if column != "id" {
		return 0
	}

	if order == ldom.SortDesc {
		for i := range items {
			if items[i].ID < after {
				return i
			}
		}

		return len(items)
	}

	for i := range items {
		if items[i].ID > after {
			return i
		}
	}

	return len(items)
}

func sortLists(
	items []ldom.List,
	sortSpec ldom.Sort,
	defaultColumn string,
	defaultOrder ldom.SortOrder,
) {
	column := strings.TrimSpace(sortSpec.Column)
	order := sortSpec.Order

	if column == "" {
		column = defaultColumn
	}

	if order != ldom.SortAsc &&
		order != ldom.SortDesc {
		order = defaultOrder
	}

	sort.SliceStable(items, func(i, j int) bool {
		comparison := compareListsForSort(
			items[i],
			items[j],
			column,
		)

		if order == ldom.SortDesc {
			return comparison > 0
		}

		return comparison < 0
	})
}

func compareListsForSort(
	left ldom.List,
	right ldom.List,
	column string,
) int {
	switch column {
	case "id":
		return compareListStrings(
			left.ID,
			right.ID,
		)

	case "readableId", "readable_id":
		if comparison := compareListStrings(
			left.ReadableID,
			right.ReadableID,
		); comparison != 0 {
			return comparison
		}

	case "title":
		if comparison := compareListStrings(
			strings.ToLower(left.Title),
			strings.ToLower(right.Title),
		); comparison != 0 {
			return comparison
		}

	case "status":
		if comparison := compareListStrings(
			string(left.Status),
			string(right.Status),
		); comparison != 0 {
			return comparison
		}

	case "assigneeId", "assignee_id":
		if comparison := compareListStrings(
			left.AssigneeID,
			right.AssigneeID,
		); comparison != 0 {
			return comparison
		}

	case "inventoryId", "inventory_id":
		if comparison := compareListStrings(
			left.InventoryID,
			right.InventoryID,
		); comparison != 0 {
			return comparison
		}

	case "createdAt", "created_at":
		if comparison := compareListTimes(
			left.CreatedAt,
			right.CreatedAt,
		); comparison != 0 {
			return comparison
		}

	case "updatedAt", "updated_at":
		if comparison := compareListTimes(
			listTimeOrZero(left.UpdatedAt),
			listTimeOrZero(right.UpdatedAt),
		); comparison != 0 {
			return comparison
		}

		if comparison := compareListTimes(
			left.CreatedAt,
			right.CreatedAt,
		); comparison != 0 {
			return comparison
		}

	default:
		if comparison := compareListTimes(
			listTimeOrZero(left.UpdatedAt),
			listTimeOrZero(right.UpdatedAt),
		); comparison != 0 {
			return comparison
		}

		if comparison := compareListTimes(
			left.CreatedAt,
			right.CreatedAt,
		); comparison != 0 {
			return comparison
		}
	}

	return compareListStrings(
		left.ID,
		right.ID,
	)
}

func compareListStrings(
	left string,
	right string,
) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func compareListTimes(
	left time.Time,
	right time.Time,
) int {
	switch {
	case left.Before(right):
		return -1
	case left.After(right):
		return 1
	default:
		return 0
	}
}

func listStringIn(
	value string,
	values []string,
) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}

	return false
}

func listStatusIn(
	value ldom.ListStatus,
	values []ldom.ListStatus,
) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}

	return false
}

func listTimeOrZero(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}

	return value.UTC()
}

// ============================================================
// Helpers - encode/decode
// ============================================================

func decodeListDoc(doc *gfs.DocumentSnapshot) (ldom.List, error) {
	if doc == nil || doc.Ref == nil || doc.Ref.ID == "" {
		return ldom.List{}, ldom.ErrInvalidID
	}

	var raw struct {
		Status      string `firestore:"status"`
		AssigneeID  string `firestore:"assignee_id"`
		Title       string `firestore:"title"`
		ImageID     string `firestore:"image_id"`
		ReadableID  string `firestore:"readable_id"`
		Description string `firestore:"description"`

		CreatedBy   string     `firestore:"created_by"`
		CreatedAt   time.Time  `firestore:"created_at"`
		UpdatedBy   *string    `firestore:"updated_by"`
		UpdatedAt   *time.Time `firestore:"updated_at"`
		InventoryID string     `firestore:"inventory_id"`
	}

	if err := doc.DataTo(&raw); err != nil {
		return ldom.List{}, err
	}

	l := ldom.List{
		ID:          doc.Ref.ID,
		Status:      ldom.ListStatus(raw.Status),
		AssigneeID:  raw.AssigneeID,
		Title:       raw.Title,
		ImageID:     raw.ImageID,
		InventoryID: raw.InventoryID,
		ReadableID:  raw.ReadableID,
		Description: raw.Description,
		Prices:      nil,
		CreatedBy:   raw.CreatedBy,
		CreatedAt:   raw.CreatedAt,
		UpdatedBy:   raw.UpdatedBy,
		UpdatedAt:   raw.UpdatedAt,
	}

	if err := l.ValidateForPersist(); err != nil {
		return ldom.List{}, err
	}

	return l, nil
}

func encodeListDoc(l ldom.List) map[string]any {
	m := map[string]any{
		"status":       string(l.Status),
		"assignee_id":  l.AssigneeID,
		"title":        l.Title,
		"image_id":     l.ImageID,
		"inventory_id": l.InventoryID,
		"readable_id":  l.ReadableID,
		"description":  l.Description,
		"created_by":   l.CreatedBy,
		"created_at":   l.CreatedAt.UTC(),
	}

	if l.UpdatedBy != nil {
		m["updated_by"] = *l.UpdatedBy
	}

	if l.UpdatedAt != nil {
		m["updated_at"] = l.UpdatedAt.UTC()
	}

	return m
}

// ============================================================
// Helpers - prices
// ============================================================

func (r *ListRepositoryFS) hydrateListPrices(
	ctx context.Context,
	items []ldom.List,
) error {
	for i := range items {
		prices, err := r.loadListPricesForOne(
			ctx,
			items[i].ID,
		)
		if err != nil {
			return err
		}

		items[i].Prices = prices
	}

	return nil
}

func (r *ListRepositoryFS) loadListPricesForOne(
	ctx context.Context,
	listID string,
) ([]ldom.ListPriceRow, error) {
	if listID == "" {
		return nil, ldom.ErrInvalidID
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
		if doc == nil ||
			doc.Ref == nil ||
			doc.Ref.ID == "" {
			return nil, ldom.ErrInvalidPriceModelID
		}

		var raw struct {
			Price int `firestore:"price"`
		}

		if err := doc.DataTo(&raw); err != nil {
			return nil, err
		}

		row := ldom.ListPriceRow{
			ModelID: doc.Ref.ID,
			Price:   raw.Price,
		}

		if err := validateListPriceRow(row); err != nil {
			return nil, err
		}

		out = append(out, row)
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
	if listRef == nil || listRef.ID == "" {
		return ldom.ErrInvalidID
	}

	if err := validateUniqueListPriceModelIDs(prices); err != nil {
		return err
	}

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
		if doc == nil || doc.Ref == nil {
			return ldom.ErrInvalidPrices
		}
		if err := tx.Delete(doc.Ref); err != nil {
			return err
		}
	}

	for _, row := range prices {
		itemRef := listRef.
			Collection(listPricesSub).
			Doc(row.ModelID)

		if err := tx.Set(
			itemRef,
			map[string]any{
				"price": row.Price,
			},
		); err != nil {
			return err
		}
	}

	return nil
}

func validateUniqueListPriceModelIDs(prices []ldom.ListPriceRow) error {
	seen := make(
		map[string]struct{},
		len(prices),
	)

	for _, row := range prices {
		if err := validateListPriceRow(row); err != nil {
			return err
		}

		if _, exists := seen[row.ModelID]; exists {
			return ldom.ErrInvalidPrices
		}

		seen[row.ModelID] = struct{}{}
	}

	return nil
}

func validateListPriceRow(row ldom.ListPriceRow) error {
	if row.ModelID == "" {
		return ldom.ErrInvalidPriceModelID
	}

	if row.Price < ldom.MinPrice ||
		row.Price > ldom.MaxPrice {
		return ldom.ErrInvalidPrice
	}

	return nil
}
