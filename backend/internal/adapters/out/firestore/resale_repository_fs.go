// backend/internal/adapters/out/firestore/resale_repository_fs.go
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
	resaledom "narratives/internal/domain/resale"
)

// ResaleRepositoryFS implements resale.Repository using Firestore.
//
// Collection:
// - resales/{resaleId}
//
// Primary image policy:
// - resales/{resaleId}.image_id stores primary condition image id.
// - Image URL is resolved from /resales/{resaleId}/conditionImages/{imageId}.
// - image_id is not a URL.
//
// Duplicate prevention policy:
// - 1 product_id can have only 1 resale document.
// - resale_product_locks/{productId} is created in the same transaction.
// - Existing resales with the same product_id are also checked defensively.
//
// Delete policy:
// - Delete physically deletes the resale document.
// - Delete also deletes child condition image records.
// - Delete also removes resale_product_locks/{productId}, so re-listing becomes possible.
type ResaleRepositoryFS struct {
	Client *gfs.Client
}

func NewResaleRepositoryFS(client *gfs.Client) *ResaleRepositoryFS {
	return &ResaleRepositoryFS{Client: client}
}

func (r *ResaleRepositoryFS) col() *gfs.CollectionRef {
	return r.Client.Collection("resales")
}

func (r *ResaleRepositoryFS) productLockRef(productID string) *gfs.DocumentRef {
	return r.Client.Collection(resaleProductLocksCol).Doc(productID)
}

const (
	resaleConditionImagesSub = "conditionImages"
	resaleProductLocksCol    = "resale_product_locks"
)

var _ resaledom.Repository = (*ResaleRepositoryFS)(nil)

// ============================================================
// Queries
// ============================================================

func (r *ResaleRepositoryFS) GetByID(
	ctx context.Context,
	id string,
) (resaledom.Resale, error) {
	if r == nil || r.Client == nil {
		return resaledom.Resale{}, errors.New("firestore client is nil")
	}

	if id == "" {
		return resaledom.Resale{}, resaledom.ErrNotFound
	}

	doc, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return resaledom.Resale{}, resaledom.ErrNotFound
		}

		return resaledom.Resale{}, err
	}

	item, err := decodeResaleDoc(doc)
	if err != nil {
		return resaledom.Resale{}, err
	}

	return item, nil
}

func (r *ResaleRepositoryFS) ListByAvatarID(
	ctx context.Context,
	avatarID string,
) ([]resaledom.Resale, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	if avatarID == "" {
		return []resaledom.Resale{}, nil
	}

	// NOTE:
	// Do not add OrderBy here.
	// Where("avatar_id", "==", avatarID) + OrderBy(...) can require a Firestore composite index.
	// Fetch by avatar_id only and sort in Go.
	it := r.col().
		Where("avatar_id", "==", avatarID).
		Documents(ctx)
	defer it.Stop()

	items := make([]resaledom.Resale, 0, 16)

	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			return nil, err
		}

		item, err := decodeResaleDoc(doc)
		if err != nil {
			return nil, err
		}

		if item.AvatarID != avatarID {
			continue
		}

		items = append(items, item)
	}

	sort.SliceStable(items, func(i, j int) bool {
		aUpdated := timeOrZero(items[i].UpdatedAt)
		bUpdated := timeOrZero(items[j].UpdatedAt)

		if !aUpdated.Equal(bUpdated) {
			return aUpdated.After(bUpdated)
		}

		if !items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].CreatedAt.After(items[j].CreatedAt)
		}

		return items[i].ID > items[j].ID
	})

	return items, nil
}

func (r *ResaleRepositoryFS) List(
	ctx context.Context,
	filter resaledom.Filter,
	sortSpec resaledom.Sort,
	page resaledom.Page,
) (resaledom.PageResult[resaledom.Resale], error) {
	if r == nil || r.Client == nil {
		return resaledom.PageResult[resaledom.Resale]{}, errors.New("firestore client is nil")
	}

	pageNum, perPage, _ := fscommon.NormalizePage(page.Number, page.PerPage, 50, 0)
	if perPage <= 0 {
		perPage = 50
	}

	if pageNum <= 0 {
		pageNum = 1
	}

	// NOTE:
	// Do not add Firestore OrderBy here.
	// Public market requests filter and sort in Go below.
	// Firestore compound OrderBy such as updated_at + created_at + documentID can require
	// composite indexes and cause 500 errors before application-side filtering runs.
	q := r.col().Query

	it := q.Documents(ctx)
	defer it.Stop()

	all := make([]resaledom.Resale, 0, perPage)

	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			return resaledom.PageResult[resaledom.Resale]{}, err
		}

		item, err := decodeResaleDoc(doc)
		if err != nil {
			return resaledom.PageResult[resaledom.Resale]{}, err
		}

		if !matchesResaleFilter(item, filter) {
			continue
		}

		all = append(all, item)
	}

	sortResales(all, sortSpec)

	totalCount := len(all)
	totalPages := 0
	if perPage > 0 && totalCount > 0 {
		totalPages = (totalCount + perPage - 1) / perPage
	}

	offset := (pageNum - 1) * perPage
	if offset < 0 {
		offset = 0
	}

	if offset >= len(all) {
		return resaledom.PageResult[resaledom.Resale]{
			Items:      []resaledom.Resale{},
			TotalCount: totalCount,
			TotalPages: totalPages,
			Page:       pageNum,
			PerPage:    perPage,
		}, nil
	}

	end := offset + perPage
	if end > len(all) {
		end = len(all)
	}

	return resaledom.PageResult[resaledom.Resale]{
		Items:      all[offset:end],
		TotalCount: totalCount,
		TotalPages: totalPages,
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *ResaleRepositoryFS) ListByCursor(
	ctx context.Context,
	filter resaledom.Filter,
	sortSpec resaledom.Sort,
	cpage resaledom.CursorPage,
) (resaledom.CursorPageResult[resaledom.Resale], error) {
	if r == nil || r.Client == nil {
		return resaledom.CursorPageResult[resaledom.Resale]{}, errors.New("firestore client is nil")
	}

	limit := cpage.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	q := r.col().OrderBy(gfs.DocumentID, gfs.Asc)

	it := q.Documents(ctx)
	defer it.Stop()

	all := make([]resaledom.Resale, 0, limit+1)

	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			return resaledom.CursorPageResult[resaledom.Resale]{}, err
		}

		item, err := decodeResaleDoc(doc)
		if err != nil {
			return resaledom.CursorPageResult[resaledom.Resale]{}, err
		}

		if !matchesResaleFilter(item, filter) {
			continue
		}

		all = append(all, item)
	}

	sortResales(all, sortSpec)

	items := make([]resaledom.Resale, 0, limit+1)
	after := cpage.After
	skipping := after != ""
	last := ""

	for _, item := range all {
		if skipping {
			if item.ID <= after {
				continue
			}

			skipping = false
		}

		items = append(items, item)
		last = item.ID

		if len(items) >= limit+1 {
			break
		}
	}

	var next *string
	if len(items) > limit {
		items = items[:limit]
		next = &last
	}

	return resaledom.CursorPageResult[resaledom.Resale]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

// ============================================================
// Mutations
// ============================================================

func (r *ResaleRepositoryFS) Create(
	ctx context.Context,
	item resaledom.Resale,
) (resaledom.Resale, error) {
	if r == nil || r.Client == nil {
		return resaledom.Resale{}, errors.New("firestore client is nil")
	}

	now := time.Now().UTC()

	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	} else {
		item.CreatedAt = item.CreatedAt.UTC()
	}

	if item.UpdatedAt == nil {
		item.UpdatedAt = &now
	} else if !item.UpdatedAt.IsZero() {
		t := item.UpdatedAt.UTC()
		item.UpdatedAt = &t
	}

	if item.UpdatedBy != nil {
		v := *item.UpdatedBy
		if v == "" {
			item.UpdatedBy = nil
		} else {
			item.UpdatedBy = &v
		}
	}

	var ref *gfs.DocumentRef
	if item.ID == "" {
		ref = r.col().NewDoc()
		item.ID = ref.ID
	} else {
		if strings.Contains(item.ID, "/") || strings.Contains(item.ID, "://") {
			return resaledom.Resale{}, resaledom.ErrInvalidID
		}

		ref = r.col().Doc(item.ID)
	}

	if item.Status == "" {
		item.Status = resaledom.StatusListing
	}

	if err := item.ValidateForPersist(); err != nil {
		return resaledom.Resale{}, err
	}

	productLockRef := r.productLockRef(item.ProductID)

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		// Defensive check for already-existing data.
		// This blocks duplicates created before resale_product_locks was introduced.
		it := tx.Documents(
			r.col().
				Where("product_id", "==", item.ProductID).
				Limit(1),
		)
		defer it.Stop()

		existingDoc, err := it.Next()
		if err == nil && existingDoc != nil {
			return resaledom.ErrConflict
		}

		if err != nil && !errors.Is(err, iterator.Done) {
			return err
		}

		_, err = tx.Get(ref)
		if err == nil {
			return resaledom.ErrConflict
		}

		if status.Code(err) != codes.NotFound {
			return err
		}

		if err := tx.Create(productLockRef, map[string]any{
			"product_id": item.ProductID,
			"resale_id":  item.ID,
			"avatar_id":  item.AvatarID,
			"created_at": now,
			"created_by": item.CreatedBy,
		}); err != nil {
			if status.Code(err) == codes.AlreadyExists {
				return resaledom.ErrConflict
			}

			return err
		}

		if err := tx.Create(ref, encodeResaleDoc(item)); err != nil {
			if status.Code(err) == codes.AlreadyExists {
				return resaledom.ErrConflict
			}

			return err
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, resaledom.ErrConflict) {
			return resaledom.Resale{}, resaledom.ErrConflict
		}

		return resaledom.Resale{}, err
	}

	return r.GetByID(ctx, item.ID)
}

func (r *ResaleRepositoryFS) Update(
	ctx context.Context,
	id string,
	item resaledom.Resale,
) (resaledom.Resale, error) {
	if r == nil || r.Client == nil {
		return resaledom.Resale{}, errors.New("firestore client is nil")
	}

	if id == "" {
		return resaledom.Resale{}, resaledom.ErrNotFound
	}

	if strings.Contains(id, "/") || strings.Contains(id, "://") {
		return resaledom.Resale{}, resaledom.ErrInvalidID
	}

	ref := r.col().Doc(id)

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		doc, err := tx.Get(ref)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return resaledom.ErrNotFound
			}

			return err
		}

		cur, err := decodeResaleDoc(doc)
		if err != nil {
			return err
		}

		cur.ID = id

		// Mutable fields.
		if item.Status != "" {
			cur.Status = item.Status
		}

		cur.Price = item.Price

		if item.Condition != "" {
			cur.Condition = item.Condition
		}

		cur.Description = item.Description
		cur.ImageID = item.ImageID

		// Preserve immutable fields by default.
		if cur.CreatedBy == "" && item.CreatedBy != "" {
			cur.CreatedBy = item.CreatedBy
		}

		if cur.CreatedAt.IsZero() && !item.CreatedAt.IsZero() {
			cur.CreatedAt = item.CreatedAt.UTC()
		}

		clearUpdatedBy := false
		clearUpdatedAt := false

		if item.UpdatedBy != nil {
			v := *item.UpdatedBy
			if v == "" {
				cur.UpdatedBy = nil
				clearUpdatedBy = true
			} else {
				cur.UpdatedBy = &v
			}
		}

		if item.UpdatedAt != nil {
			if item.UpdatedAt.IsZero() {
				cur.UpdatedAt = nil
				clearUpdatedAt = true
			} else {
				t := item.UpdatedAt.UTC()
				cur.UpdatedAt = &t
			}
		} else {
			t := time.Now().UTC()
			cur.UpdatedAt = &t
		}

		if err := cur.ValidateForPersist(); err != nil {
			return err
		}

		data := encodeResaleDoc(cur)

		if clearUpdatedBy {
			data["updated_by"] = gfs.Delete
		}

		if clearUpdatedAt {
			data["updated_at"] = gfs.Delete
		}

		if err := tx.Set(ref, data, gfs.MergeAll); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, resaledom.ErrNotFound) {
			return resaledom.Resale{}, resaledom.ErrNotFound
		}

		return resaledom.Resale{}, err
	}

	return r.GetByID(ctx, id)
}

func (r *ResaleRepositoryFS) Delete(
	ctx context.Context,
	id string,
) error {
	if r == nil || r.Client == nil {
		return errors.New("firestore client is nil")
	}

	if id == "" {
		return resaledom.ErrNotFound
	}

	if strings.Contains(id, "/") || strings.Contains(id, "://") {
		return resaledom.ErrInvalidID
	}

	ref := r.col().Doc(id)

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		doc, err := tx.Get(ref)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return resaledom.ErrNotFound
			}

			return err
		}

		item, err := decodeResaleDoc(doc)
		if err != nil {
			return err
		}

		if err := item.ValidateDelete(); err != nil {
			return err
		}

		itImages := ref.Collection(resaleConditionImagesSub).Documents(ctx)
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

		if item.ProductID != "" {
			if err := tx.Delete(r.productLockRef(item.ProductID)); err != nil {
				return err
			}
		}

		return tx.Delete(ref)
	})
	if err != nil {
		if errors.Is(err, resaledom.ErrNotFound) {
			return resaledom.ErrNotFound
		}

		return err
	}

	return nil
}

// ============================================================
// Firestore encode/decode - resale
// ============================================================

func encodeResaleDoc(item resaledom.Resale) map[string]any {
	m := map[string]any{
		"status":               string(item.Status),
		"mint_address":         item.MintAddress,
		"token_blueprint_id":   item.TokenBlueprintID,
		"product_id":           item.ProductID,
		"brand_id":             item.BrandID,
		"product_blueprint_id": item.ProductBlueprintID,
		"avatar_id":            item.AvatarID,
		"price":                item.Price,
		"condition":            string(item.Condition),
		"description":          item.Description,
		"image_id":             item.ImageID,
		"created_by":           item.CreatedBy,
		"created_at":           item.CreatedAt.UTC(),
	}

	if item.UpdatedBy != nil {
		if v := *item.UpdatedBy; v != "" {
			m["updated_by"] = v
		}
	}

	if item.UpdatedAt != nil && !item.UpdatedAt.IsZero() {
		m["updated_at"] = item.UpdatedAt.UTC()
	}

	return m
}

func decodeResaleDoc(doc *gfs.DocumentSnapshot) (resaledom.Resale, error) {
	if doc == nil || doc.Ref == nil || doc.Ref.ID == "" {
		return resaledom.Resale{}, resaledom.ErrNotFound
	}

	var raw struct {
		Status             string     `firestore:"status"`
		MintAddress        string     `firestore:"mint_address"`
		TokenBlueprintID   string     `firestore:"token_blueprint_id"`
		ProductID          string     `firestore:"product_id"`
		BrandID            string     `firestore:"brand_id"`
		ProductBlueprintID string     `firestore:"product_blueprint_id"`
		AvatarID           string     `firestore:"avatar_id"`
		Price              int        `firestore:"price"`
		Condition          string     `firestore:"condition"`
		Description        string     `firestore:"description"`
		ImageID            string     `firestore:"image_id"`
		CreatedBy          string     `firestore:"created_by"`
		CreatedAt          time.Time  `firestore:"created_at"`
		UpdatedBy          *string    `firestore:"updated_by"`
		UpdatedAt          *time.Time `firestore:"updated_at"`
	}

	if err := doc.DataTo(&raw); err != nil {
		return resaledom.Resale{}, err
	}

	createdAt := raw.CreatedAt.UTC()

	var updatedAt *time.Time
	if raw.UpdatedAt != nil {
		t := raw.UpdatedAt.UTC()
		updatedAt = &t
	}

	item := resaledom.Resale{
		ID:                 doc.Ref.ID,
		Status:             resaledom.ResaleStatus(raw.Status),
		MintAddress:        raw.MintAddress,
		TokenBlueprintID:   raw.TokenBlueprintID,
		ProductID:          raw.ProductID,
		BrandID:            raw.BrandID,
		ProductBlueprintID: raw.ProductBlueprintID,
		AvatarID:           raw.AvatarID,
		Price:              raw.Price,
		Condition:          resaledom.ResaleCondition(raw.Condition),
		Description:        raw.Description,
		ImageID:            raw.ImageID,
		CreatedBy:          raw.CreatedBy,
		CreatedAt:          createdAt,
		UpdatedBy:          raw.UpdatedBy,
		UpdatedAt:          updatedAt,
	}

	if err := item.ValidateForPersist(); err != nil {
		return resaledom.Resale{}, err
	}

	return item, nil
}

// ============================================================
// Filter / sort helpers
// ============================================================

func matchesResaleFilter(item resaledom.Resale, filter resaledom.Filter) bool {
	if len(filter.IDs) > 0 && !stringIn(item.ID, filter.IDs) {
		return false
	}

	if len(filter.MintAddresses) > 0 && !stringIn(item.MintAddress, filter.MintAddresses) {
		return false
	}

	if len(filter.TokenBlueprintIDs) > 0 && !stringIn(item.TokenBlueprintID, filter.TokenBlueprintIDs) {
		return false
	}

	if len(filter.ProductIDs) > 0 && !stringIn(item.ProductID, filter.ProductIDs) {
		return false
	}

	if len(filter.BrandIDs) > 0 && !stringIn(item.BrandID, filter.BrandIDs) {
		return false
	}

	if len(filter.ProductBlueprintIDs) > 0 && !stringIn(item.ProductBlueprintID, filter.ProductBlueprintIDs) {
		return false
	}

	if len(filter.AvatarIDs) > 0 && !stringIn(item.AvatarID, filter.AvatarIDs) {
		return false
	}

	if len(filter.ExcludeAvatarIDs) > 0 && stringIn(item.AvatarID, filter.ExcludeAvatarIDs) {
		return false
	}

	if filter.Status != nil && item.Status != *filter.Status {
		return false
	}

	if len(filter.Statuses) > 0 && !statusIn(item.Status, filter.Statuses) {
		return false
	}

	if filter.Condition != nil && item.Condition != *filter.Condition {
		return false
	}

	if len(filter.Conditions) > 0 && !conditionIn(item.Condition, filter.Conditions) {
		return false
	}

	if filter.MinPrice != nil && item.Price < *filter.MinPrice {
		return false
	}

	if filter.MaxPrice != nil && item.Price > *filter.MaxPrice {
		return false
	}

	q := strings.ToLower(filter.SearchQuery)
	if q != "" {
		haystack := strings.ToLower(strings.Join([]string{
			item.ID,
			item.MintAddress,
			item.TokenBlueprintID,
			item.ProductID,
			item.BrandID,
			item.ProductBlueprintID,
			item.AvatarID,
			item.Description,
			string(item.Status),
			string(item.Condition),
		}, " "))

		if !strings.Contains(haystack, q) {
			return false
		}
	}

	return true
}

func sortResales(items []resaledom.Resale, sortSpec resaledom.Sort) {
	column := sortSpec.Column
	order := sortSpec.Order

	if column == "" {
		column = "updatedAt"
	}

	if order == "" {
		order = resaledom.SortDesc
	}

	less := func(i, j int) bool {
		a := items[i]
		b := items[j]

		switch column {
		case "id":
			return a.ID < b.ID

		case "price":
			if a.Price == b.Price {
				return a.ID < b.ID
			}

			return a.Price < b.Price

		case "createdAt", "created_at":
			if a.CreatedAt.Equal(b.CreatedAt) {
				return a.ID < b.ID
			}

			return a.CreatedAt.Before(b.CreatedAt)

		case "updatedAt", "updated_at":
			at := timeOrZero(a.UpdatedAt)
			bt := timeOrZero(b.UpdatedAt)
			if at.Equal(bt) {
				if a.CreatedAt.Equal(b.CreatedAt) {
					return a.ID < b.ID
				}

				return a.CreatedAt.Before(b.CreatedAt)
			}

			return at.Before(bt)

		default:
			at := timeOrZero(a.UpdatedAt)
			bt := timeOrZero(b.UpdatedAt)
			if at.Equal(bt) {
				if a.CreatedAt.Equal(b.CreatedAt) {
					return a.ID < b.ID
				}

				return a.CreatedAt.Before(b.CreatedAt)
			}

			return at.Before(bt)
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		if order == resaledom.SortDesc {
			return !less(i, j)
		}

		return less(i, j)
	})
}

func stringIn(value string, values []string) bool {
	for _, v := range values {
		if value == v {
			return true
		}
	}

	return false
}

func statusIn(value resaledom.ResaleStatus, values []resaledom.ResaleStatus) bool {
	for _, v := range values {
		if value == v {
			return true
		}
	}

	return false
}

func conditionIn(value resaledom.ResaleCondition, values []resaledom.ResaleCondition) bool {
	for _, v := range values {
		if value == v {
			return true
		}
	}

	return false
}

func timeOrZero(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}

	return t.UTC()
}
