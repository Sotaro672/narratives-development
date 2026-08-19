// backend/internal/adapters/out/firestore/produtcBlueprintCategory_fs.go
package firestore

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

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

var _ categorydom.ReadOnlyRepositoryPort = (*ProductBlueprintCategoryRepositoryFS)(nil)

// ------------------------------------------------------------
// Firestore DTO
// ------------------------------------------------------------

type productBlueprintCategoryDoc struct {
	ProductBlueprintCategoryPath []string `firestore:"productBlueprintCategoryPath"`
}

// ------------------------------------------------------------
// Read methods
// ------------------------------------------------------------

func (r *ProductBlueprintCategoryRepositoryFS) GetByPath(
	ctx context.Context,
	path []string,
) (categorydom.ProductBlueprintCategory, error) {
	if r == nil || r.client == nil {
		return categorydom.ProductBlueprintCategory{}, categorydom.ErrRepositoryInvalidInput
	}

	if !isValidProductBlueprintCategoryPath(path) {
		return categorydom.ProductBlueprintCategory{}, categorydom.ErrInvalidPath
	}

	iter := r.client.Collection(productBlueprintCategoryCollection).
		Where(
			"productBlueprintCategoryPath",
			"==",
			append(
				[]string(nil),
				path...,
			),
		).
		Limit(2).
		Documents(ctx)
	defer iter.Stop()

	snap, err := iter.Next()
	if err == iterator.Done {
		return categorydom.ProductBlueprintCategory{}, categorydom.ErrNotFound
	}
	if err != nil {
		return categorydom.ProductBlueprintCategory{}, err
	}

	category, err := productBlueprintCategoryFromSnapshot(snap)
	if err != nil {
		return categorydom.ProductBlueprintCategory{}, err
	}

	duplicate, err := iter.Next()
	if err == nil && duplicate != nil {
		return categorydom.ProductBlueprintCategory{}, categorydom.ErrConflict
	}
	if err != nil && err != iterator.Done {
		return categorydom.ProductBlueprintCategory{}, err
	}

	return category, nil
}

func (r *ProductBlueprintCategoryRepositoryFS) List(
	ctx context.Context,
	filter categorydom.Filter,
	sortSpec common.Sort,
	page common.Page,
) (common.PageResult[categorydom.ProductBlueprintCategory], error) {
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

	sortColumn := sortSpec.Column
	if sortColumn == "" {
		sortColumn = categorydom.SortColumnPath
	}
	if !categorydom.IsAllowedSortColumn(sortColumn) {
		return common.PageResult[categorydom.ProductBlueprintCategory]{}, categorydom.ErrRepositoryInvalidInput
	}

	sortOrder := sortSpec.Order
	if sortOrder == "" {
		sortOrder = common.SortAsc
	}

	switch sortOrder {
	case common.SortAsc:
	case common.SortDesc:
	default:
		return common.PageResult[categorydom.ProductBlueprintCategory]{}, categorydom.ErrRepositoryInvalidInput
	}

	all, err := r.listAll(ctx)
	if err != nil {
		return common.PageResult[categorydom.ProductBlueprintCategory]{}, err
	}

	filtered := make([]categorydom.ProductBlueprintCategory, 0, len(all))

	for _, category := range all {
		if !matchesProductBlueprintCategoryFilter(
			category,
			filter,
		) {
			continue
		}

		filtered = append(
			filtered,
			category,
		)
	}

	sortProductBlueprintCategories(
		filtered,
		sortOrder,
	)

	totalCount := len(filtered)
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
		Items:      filtered[start:end],
		TotalCount: totalCount,
		TotalPages: totalPages,
		Page:       pageNumber,
		PerPage:    perPage,
	}, nil
}

func (r *ProductBlueprintCategoryRepositoryFS) ListTree(
	ctx context.Context,
) ([]categorydom.ProductBlueprintCategory, error) {
	if r == nil || r.client == nil {
		return nil, categorydom.ErrRepositoryInvalidInput
	}

	items, err := r.listAll(ctx)
	if err != nil {
		return nil, err
	}

	sortProductBlueprintCategories(
		items,
		common.SortAsc,
	)

	return items, nil
}

func (r *ProductBlueprintCategoryRepositoryFS) ListCursor(
	ctx context.Context,
	filter categorydom.Filter,
	page common.CursorPage,
) (common.CursorPageResult[categorydom.ProductBlueprintCategory], error) {
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

	all, err := r.listAll(ctx)
	if err != nil {
		return common.CursorPageResult[categorydom.ProductBlueprintCategory]{}, err
	}

	filtered := make([]categorydom.ProductBlueprintCategory, 0, len(all))

	for _, category := range all {
		if !matchesProductBlueprintCategoryFilter(
			category,
			filter,
		) {
			continue
		}

		filtered = append(
			filtered,
			category,
		)
	}

	sortProductBlueprintCategories(
		filtered,
		common.SortAsc,
	)

	start := 0

	if page.After != "" {
		afterPath, err := decodeProductBlueprintCategoryCursor(
			page.After,
		)
		if err != nil {
			return common.CursorPageResult[categorydom.ProductBlueprintCategory]{}, categorydom.ErrRepositoryInvalidInput
		}

		found := false

		for i, category := range filtered {
			if !equalProductBlueprintCategoryPath(
				category.Path,
				afterPath,
			) {
				continue
			}

			start = i + 1
			found = true
			break
		}

		if !found {
			return common.CursorPageResult[categorydom.ProductBlueprintCategory]{}, categorydom.ErrRepositoryInvalidInput
		}
	}

	if start > len(filtered) {
		start = len(filtered)
	}

	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	items := append(
		[]categorydom.ProductBlueprintCategory(nil),
		filtered[start:end]...,
	)

	var next *string

	if end < len(filtered) && len(items) > 0 {
		cursor, err := encodeProductBlueprintCategoryCursor(
			items[len(items)-1].Path,
		)
		if err != nil {
			return common.CursorPageResult[categorydom.ProductBlueprintCategory]{}, err
		}

		next = &cursor
	}

	return common.CursorPageResult[categorydom.ProductBlueprintCategory]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

func (r *ProductBlueprintCategoryRepositoryFS) ExistsByPath(
	ctx context.Context,
	path []string,
) (bool, error) {
	if r == nil || r.client == nil {
		return false, categorydom.ErrRepositoryInvalidInput
	}

	if !isValidProductBlueprintCategoryPath(path) {
		return false, categorydom.ErrInvalidPath
	}

	iter := r.client.Collection(productBlueprintCategoryCollection).
		Where(
			"productBlueprintCategoryPath",
			"==",
			append(
				[]string(nil),
				path...,
			),
		).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	_, err := iter.Next()
	if err == iterator.Done {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// ------------------------------------------------------------
// Mapping
// ------------------------------------------------------------

func productBlueprintCategoryFromSnapshot(
	snap *firestore.DocumentSnapshot,
) (categorydom.ProductBlueprintCategory, error) {
	if snap == nil {
		return categorydom.ProductBlueprintCategory{}, categorydom.ErrNotFound
	}

	var doc productBlueprintCategoryDoc
	if err := snap.DataTo(&doc); err != nil {
		return categorydom.ProductBlueprintCategory{}, err
	}

	if !isValidProductBlueprintCategoryPath(
		doc.ProductBlueprintCategoryPath,
	) {
		return categorydom.ProductBlueprintCategory{}, categorydom.ErrInvalidPath
	}

	return categorydom.Reconstruct(
		append(
			[]string(nil),
			doc.ProductBlueprintCategoryPath...,
		),
	)
}

// ------------------------------------------------------------
// Filtering
// ------------------------------------------------------------

func matchesProductBlueprintCategoryFilter(
	category categorydom.ProductBlueprintCategory,
	filter categorydom.Filter,
) bool {
	if len(filter.Paths) > 0 &&
		!containsProductBlueprintCategoryPath(
			filter.Paths,
			category.Path,
		) {
		return false
	}

	searchQuery := strings.ToLower(
		filter.SearchQuery,
	)
	if searchQuery != "" {
		haystack := strings.ToLower(
			strings.Join(
				category.Path,
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

func containsProductBlueprintCategoryPath(
	paths [][]string,
	path []string,
) bool {
	for _, candidate := range paths {
		if equalProductBlueprintCategoryPath(
			candidate,
			path,
		) {
			return true
		}
	}

	return false
}

func equalProductBlueprintCategoryPath(
	a []string,
	b []string,
) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// ------------------------------------------------------------
// Sorting
// ------------------------------------------------------------

func sortProductBlueprintCategories(
	items []categorydom.ProductBlueprintCategory,
	order common.SortOrder,
) {
	sort.SliceStable(
		items,
		func(i int, j int) bool {
			comparison := compareProductBlueprintCategoryPath(
				items[i].Path,
				items[j].Path,
			)

			if order == common.SortDesc {
				return comparison > 0
			}

			return comparison < 0
		},
	)
}

func compareProductBlueprintCategoryPath(
	a []string,
	b []string,
) int {
	length := len(a)
	if len(b) < length {
		length = len(b)
	}

	for i := 0; i < length; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}

	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}

	return 0
}

// ------------------------------------------------------------
// Cursor
// ------------------------------------------------------------

func encodeProductBlueprintCategoryCursor(
	path []string,
) (string, error) {
	if !isValidProductBlueprintCategoryPath(path) {
		return "", categorydom.ErrInvalidPath
	}

	value, err := json.Marshal(
		path,
	)
	if err != nil {
		return "", err
	}

	return string(value), nil
}

func decodeProductBlueprintCategoryCursor(
	cursor string,
) ([]string, error) {
	if cursor == "" {
		return nil, categorydom.ErrRepositoryInvalidInput
	}

	var path []string
	if err := json.Unmarshal(
		[]byte(cursor),
		&path,
	); err != nil {
		return nil, categorydom.ErrRepositoryInvalidInput
	}

	if !isValidProductBlueprintCategoryPath(path) {
		return nil, categorydom.ErrRepositoryInvalidInput
	}

	return append(
		[]string(nil),
		path...,
	), nil
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

func (r *ProductBlueprintCategoryRepositoryFS) listAll(
	ctx context.Context,
) ([]categorydom.ProductBlueprintCategory, error) {
	iter := r.client.Collection(productBlueprintCategoryCollection).
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

		category, err := productBlueprintCategoryFromSnapshot(
			snap,
		)
		if err != nil {
			return nil, err
		}

		items = append(
			items,
			category,
		)
	}

	return items, nil
}

func isValidProductBlueprintCategoryPath(
	path []string,
) bool {
	if len(path) == 0 {
		return false
	}

	for _, segment := range path {
		if segment == "" {
			return false
		}
	}

	return true
}
