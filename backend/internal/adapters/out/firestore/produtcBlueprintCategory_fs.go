// backend/internal/adapters/out/firestore/produtcBlueprintCategory_fs.go
package firestore

import (
	"context"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"narratives/internal/domain/common"
	categorydom "narratives/internal/domain/productBlueprintCategory"
)

const productBlueprintCategoryCollection = "productBlueprintCategories"

type ProductBlueprintCategoryRepositoryFS struct {
	client *firestore.Client
}

func NewProductBlueprintCategoryRepositoryFS(client *firestore.Client) *ProductBlueprintCategoryRepositoryFS {
	return &ProductBlueprintCategoryRepositoryFS{client: client}
}

// ------------------------------------------------------------
// Firestore DTO
// ------------------------------------------------------------

type productBlueprintCategoryDoc struct {
	Code         string                                `firestore:"code"`
	NameJa       string                                `firestore:"nameJa"`
	NameEn       string                                `firestore:"nameEn"`
	ParentID     *string                               `firestore:"parentId"`
	Path         []string                              `firestore:"path"`
	Kind         string                                `firestore:"kind"`
	DisplayOrder int                                   `firestore:"displayOrder"`
	Attributes   productBlueprintCategoryAttributesDoc `firestore:"attributes"`
	CreatedAt    time.Time                             `firestore:"createdAt"`
	UpdatedAt    time.Time                             `firestore:"updatedAt"`
}

type productBlueprintCategoryAttributesDoc struct {
	RequiresExpirationDate bool `firestore:"requiresExpirationDate"`
	RequiresLotNumber      bool `firestore:"requiresLotNumber"`
	RequiresIngredients    bool `firestore:"requiresIngredients"`
	RequiresAlcoholNotice  bool `firestore:"requiresAlcoholNotice"`
	RequiresCosmeticNotice bool `firestore:"requiresCosmeticNotice"`
	RequiresStorageMethod  bool `firestore:"requiresStorageMethod"`
}

// ------------------------------------------------------------
// Read methods
// ------------------------------------------------------------

func (r *ProductBlueprintCategoryRepositoryFS) GetByID(ctx context.Context, id string) (categorydom.ProductBlueprintCategory, error) {
	if r == nil || r.client == nil {
		return categorydom.ProductBlueprintCategory{}, categorydom.ErrRepositoryInvalidInput
	}
	if id == "" {
		return categorydom.ProductBlueprintCategory{}, categorydom.ErrInvalidID
	}

	snap, err := r.client.Collection(productBlueprintCategoryCollection).Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return categorydom.ProductBlueprintCategory{}, categorydom.ErrNotFound
		}
		return categorydom.ProductBlueprintCategory{}, err
	}

	return productBlueprintCategoryFromSnapshot(snap)
}

func (r *ProductBlueprintCategoryRepositoryFS) List(ctx context.Context, filter categorydom.Filter, sort common.Sort, page common.Page) (common.PageResult[categorydom.ProductBlueprintCategory], error) {
	if r == nil || r.client == nil {
		return common.PageResult[categorydom.ProductBlueprintCategory]{}, categorydom.ErrRepositoryInvalidInput
	}

	pageNumber := page.Number
	if pageNumber <= 0 {
		pageNumber = 1
	}

	perPage := page.PerPage
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 500 {
		perPage = 500
	}

	sortColumn := sort.Column
	if sortColumn == "" {
		sortColumn = categorydom.SortColumnDisplayOrder
	}
	if !categorydom.IsAllowedSortColumn(sortColumn) {
		return common.PageResult[categorydom.ProductBlueprintCategory]{}, categorydom.ErrRepositoryInvalidInput
	}

	sortField, err := productBlueprintCategoryFirestoreSortField(sortColumn)
	if err != nil {
		return common.PageResult[categorydom.ProductBlueprintCategory]{}, err
	}

	sortOrder := sort.Order
	if sortOrder == "" {
		sortOrder = common.SortAsc
	}

	var direction firestore.Direction
	switch sortOrder {
	case common.SortAsc:
		direction = firestore.Asc
	case common.SortDesc:
		direction = firestore.Desc
	default:
		return common.PageResult[categorydom.ProductBlueprintCategory]{}, categorydom.ErrRepositoryInvalidInput
	}

	iter := r.client.Collection(productBlueprintCategoryCollection).OrderBy(sortField, direction).Documents(ctx)
	defer iter.Stop()

	all := make([]categorydom.ProductBlueprintCategory, 0)
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return common.PageResult[categorydom.ProductBlueprintCategory]{}, err
		}

		category, err := productBlueprintCategoryFromSnapshot(snap)
		if err != nil {
			return common.PageResult[categorydom.ProductBlueprintCategory]{}, err
		}
		if !matchesProductBlueprintCategoryFilter(category, filter) {
			continue
		}

		all = append(all, category)
	}

	totalCount := len(all)
	totalPages := 0
	if totalCount > 0 {
		totalPages = (totalCount + perPage - 1) / perPage
	}

	start := (pageNumber - 1) * perPage
	if start > totalCount {
		start = totalCount
	}

	end := start + perPage
	if end > totalCount {
		end = totalCount
	}

	return common.PageResult[categorydom.ProductBlueprintCategory]{
		Items:      all[start:end],
		TotalCount: totalCount,
		TotalPages: totalPages,
		Page:       pageNumber,
		PerPage:    perPage,
	}, nil
}

func (r *ProductBlueprintCategoryRepositoryFS) ListTree(ctx context.Context) ([]categorydom.ProductBlueprintCategory, error) {
	if r == nil || r.client == nil {
		return nil, categorydom.ErrRepositoryInvalidInput
	}

	iter := r.client.Collection(productBlueprintCategoryCollection).
		OrderBy("displayOrder", firestore.Asc).
		OrderBy(firestore.DocumentID, firestore.Asc).
		Documents(ctx)
	defer iter.Stop()

	items := make([]categorydom.ProductBlueprintCategory, 0)
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		category, err := productBlueprintCategoryFromSnapshot(snap)
		if err != nil {
			return nil, err
		}

		items = append(items, category)
	}

	return items, nil
}

func (r *ProductBlueprintCategoryRepositoryFS) ListCursor(ctx context.Context, filter categorydom.Filter, page common.CursorPage) (common.CursorPageResult[categorydom.ProductBlueprintCategory], error) {
	if r == nil || r.client == nil {
		return common.CursorPageResult[categorydom.ProductBlueprintCategory]{}, categorydom.ErrRepositoryInvalidInput
	}

	limit := page.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	collection := r.client.Collection(productBlueprintCategoryCollection)
	query := collection.OrderBy("displayOrder", firestore.Asc).OrderBy(firestore.DocumentID, firestore.Asc)

	if page.After != "" {
		cursorSnap, err := collection.Doc(page.After).Get(ctx)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return common.CursorPageResult[categorydom.ProductBlueprintCategory]{}, categorydom.ErrRepositoryInvalidInput
			}
			return common.CursorPageResult[categorydom.ProductBlueprintCategory]{}, err
		}

		cursorCategory, err := productBlueprintCategoryFromSnapshot(cursorSnap)
		if err != nil {
			return common.CursorPageResult[categorydom.ProductBlueprintCategory]{}, err
		}

		query = query.StartAfter(cursorCategory.DisplayOrder, cursorSnap.Ref.ID)
	}

	iter := query.Documents(ctx)
	defer iter.Stop()

	items := make([]categorydom.ProductBlueprintCategory, 0, limit+1)
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return common.CursorPageResult[categorydom.ProductBlueprintCategory]{}, err
		}

		category, err := productBlueprintCategoryFromSnapshot(snap)
		if err != nil {
			return common.CursorPageResult[categorydom.ProductBlueprintCategory]{}, err
		}
		if !matchesProductBlueprintCategoryFilter(category, filter) {
			continue
		}

		items = append(items, category)
		if len(items) >= limit+1 {
			break
		}
	}

	var next *string
	if len(items) > limit {
		cursor := string(items[limit-1].ID)
		items = items[:limit]
		next = &cursor
	}

	return common.CursorPageResult[categorydom.ProductBlueprintCategory]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

func (r *ProductBlueprintCategoryRepositoryFS) ExistsByID(ctx context.Context, id string) (bool, error) {
	if r == nil || r.client == nil {
		return false, categorydom.ErrRepositoryInvalidInput
	}
	if id == "" {
		return false, categorydom.ErrInvalidID
	}

	_, err := r.client.Collection(productBlueprintCategoryCollection).Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// ------------------------------------------------------------
// Mapping
// ------------------------------------------------------------

func productBlueprintCategoryFromSnapshot(snap *firestore.DocumentSnapshot) (categorydom.ProductBlueprintCategory, error) {
	if snap == nil || snap.Ref == nil {
		return categorydom.ProductBlueprintCategory{}, categorydom.ErrNotFound
	}
	if snap.Ref.ID == "" {
		return categorydom.ProductBlueprintCategory{}, categorydom.ErrInvalidID
	}

	var doc productBlueprintCategoryDoc
	if err := snap.DataTo(&doc); err != nil {
		return categorydom.ProductBlueprintCategory{}, err
	}

	var parentID *categorydom.CategoryID
	if doc.ParentID != nil {
		v := categorydom.CategoryID(*doc.ParentID)
		parentID = &v
	}

	return categorydom.Reconstruct(
		categorydom.CategoryID(snap.Ref.ID),
		categorydom.CategoryCode(doc.Code),
		doc.NameJa,
		doc.NameEn,
		parentID,
		append([]string(nil), doc.Path...),
		categorydom.CategoryKind(doc.Kind),
		doc.DisplayOrder,
		categorydom.CategoryAttributes{
			RequiresExpirationDate: doc.Attributes.RequiresExpirationDate,
			RequiresLotNumber:      doc.Attributes.RequiresLotNumber,
			RequiresIngredients:    doc.Attributes.RequiresIngredients,
			RequiresAlcoholNotice:  doc.Attributes.RequiresAlcoholNotice,
			RequiresCosmeticNotice: doc.Attributes.RequiresCosmeticNotice,
			RequiresStorageMethod:  doc.Attributes.RequiresStorageMethod,
		},
		doc.CreatedAt,
		doc.UpdatedAt,
	)
}

// ------------------------------------------------------------
// Filtering
// ------------------------------------------------------------

func matchesProductBlueprintCategoryFilter(category categorydom.ProductBlueprintCategory, filter categorydom.Filter) bool {
	if len(filter.IDs) > 0 && !containsCategoryID(filter.IDs, category.ID) {
		return false
	}
	if filter.Code != nil && category.Code != *filter.Code {
		return false
	}
	if filter.Kind != nil && category.Kind != *filter.Kind {
		return false
	}
	if filter.ParentID != nil && (category.ParentID == nil || *category.ParentID != *filter.ParentID) {
		return false
	}
	if filter.RootOnly && category.ParentID != nil {
		return false
	}

	searchQuery := strings.ToLower(filter.SearchQuery)
	if searchQuery != "" {
		haystack := strings.ToLower(strings.Join([]string{
			string(category.ID),
			string(category.Code),
			category.NameJa,
			category.NameEn,
			string(category.Kind),
			strings.Join(category.Path, " "),
		}, " "))
		if !strings.Contains(haystack, searchQuery) {
			return false
		}
	}

	if filter.Created.From != nil && category.CreatedAt.Before(filter.Created.From.UTC()) {
		return false
	}
	if filter.Created.To != nil && category.CreatedAt.After(filter.Created.To.UTC()) {
		return false
	}
	if filter.Updated.From != nil && category.UpdatedAt.Before(filter.Updated.From.UTC()) {
		return false
	}
	if filter.Updated.To != nil && category.UpdatedAt.After(filter.Updated.To.UTC()) {
		return false
	}

	return true
}

func containsCategoryID(ids []categorydom.CategoryID, id categorydom.CategoryID) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

func productBlueprintCategoryFirestoreSortField(column string) (string, error) {
	switch column {
	case "id":
		return firestore.DocumentID, nil
	case "code":
		return "code", nil
	case "nameJa":
		return "nameJa", nil
	case "nameEn":
		return "nameEn", nil
	case "kind":
		return "kind", nil
	case "displayOrder":
		return "displayOrder", nil
	case "createdAt":
		return "createdAt", nil
	case "updatedAt":
		return "updatedAt", nil
	default:
		return "", categorydom.ErrRepositoryInvalidInput
	}
}
