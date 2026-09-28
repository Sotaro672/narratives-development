// backend/internal/application/query/mall/list_query.go
package mall

import (
	"context"
	"errors"
	"math/rand/v2"
	"sort"
	"strings"

	applicationport "narratives/internal/application/port"
	mallshared "narratives/internal/application/query/mall/shared"
	ldom "narratives/internal/domain/list"
)

const mallListCursorBatchSize = 200

type ListQuery struct {
	listRepo  ldom.Repository
	imageRepo applicationport.ListImageLister
}

func NewListQuery(
	listRepo ldom.Repository,
	imageRepo applicationport.ListImageLister,
) *ListQuery {
	return &ListQuery{
		listRepo:  listRepo,
		imageRepo: imageRepo,
	}
}

type ListItemDTO struct {
	ID          string              `json:"id"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Image       string              `json:"image"`
	Prices      []ldom.ListPriceRow `json:"prices"`

	InventoryID        string `json:"inventoryId,omitempty"`
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`
}

type ListIndexResponseDTO struct {
	Items      []ListItemDTO `json:"items"`
	TotalCount int           `json:"totalCount"`
	TotalPages int           `json:"totalPages"`
	Page       int           `json:"page"`
	PerPage    int           `json:"perPage"`
}

func (q *ListQuery) ListIndex(
	ctx context.Context,
	pageNum int,
	perPage int,
) (ListIndexResponseDTO, error) {
	if q == nil || q.listRepo == nil {
		return ListIndexResponseDTO{}, errors.New("mall list query: list repo is nil")
	}

	pageNum, perPage = mallshared.NormalizeIntPage(
		pageNum,
		perPage,
		1,
		20,
		50,
	)

	publicLists, err := q.listAllPublicListings(ctx)
	if err != nil {
		return ListIndexResponseDTO{}, err
	}

	groups := groupMallListsByInventory(publicLists)

	totalCount := len(groups)
	totalPages := 0
	if totalCount > 0 {
		totalPages = (totalCount + perPage - 1) / perPage
	}

	start := (pageNum - 1) * perPage
	if start >= totalCount {
		return ListIndexResponseDTO{
			Items:      []ListItemDTO{},
			TotalCount: totalCount,
			TotalPages: totalPages,
			Page:       pageNum,
			PerPage:    perPage,
		}, nil
	}

	end := start + perPage
	if end > totalCount {
		end = totalCount
	}

	items := make([]ListItemDTO, 0, end-start)
	for _, group := range groups[start:end] {
		selected, ok := pickMallList(group)
		if !ok {
			continue
		}

		item, err := q.toListItemDTO(ctx, selected)
		if err != nil {
			return ListIndexResponseDTO{}, err
		}

		items = append(items, item)
	}

	return ListIndexResponseDTO{
		Items:      items,
		TotalCount: totalCount,
		TotalPages: totalPages,
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (q *ListQuery) listAllPublicListings(
	ctx context.Context,
) ([]ldom.List, error) {
	status := ldom.StatusListing
	filter := ldom.Filter{
		Status: &status,
	}

	items := make([]ldom.List, 0)
	after := ""

	for {
		result, err := q.listRepo.ListByCursor(
			ctx,
			filter,
			ldom.Sort{},
			ldom.CursorPage{
				After: after,
				Limit: mallListCursorBatchSize,
			},
		)
		if err != nil {
			return nil, err
		}

		for _, l := range result.Items {
			if !isMallPublicListing(l.Status) {
				continue
			}

			if strings.TrimSpace(l.InventoryID) == "" {
				continue
			}

			items = append(items, l)
		}

		if result.NextCursor == nil {
			break
		}

		next := strings.TrimSpace(*result.NextCursor)
		if next == "" {
			break
		}

		if next == after {
			return nil, errors.New("mall list query: cursor did not advance")
		}

		after = next
	}

	return items, nil
}

func groupMallListsByInventory(
	items []ldom.List,
) [][]ldom.List {
	sortedItems := append([]ldom.List(nil), items...)

	sort.SliceStable(sortedItems, func(i, j int) bool {
		return mallListComesBefore(sortedItems[i], sortedItems[j])
	})

	groups := make([][]ldom.List, 0)
	groupIndexes := make(map[string]int)

	for _, l := range sortedItems {
		inventoryID := strings.TrimSpace(l.InventoryID)
		if inventoryID == "" {
			continue
		}

		index, exists := groupIndexes[inventoryID]
		if !exists {
			groupIndexes[inventoryID] = len(groups)
			groups = append(groups, []ldom.List{l})
			continue
		}

		groups[index] = append(groups[index], l)
	}

	return groups
}

func mallListComesBefore(
	left ldom.List,
	right ldom.List,
) bool {
	if left.UpdatedAt != nil && right.UpdatedAt != nil {
		if !left.UpdatedAt.Equal(*right.UpdatedAt) {
			return left.UpdatedAt.After(*right.UpdatedAt)
		}
	} else if left.UpdatedAt != nil {
		return true
	} else if right.UpdatedAt != nil {
		return false
	}

	if !left.CreatedAt.Equal(right.CreatedAt) {
		return left.CreatedAt.After(right.CreatedAt)
	}

	return left.ID > right.ID
}

func pickMallList(
	items []ldom.List,
) (ldom.List, bool) {
	switch len(items) {
	case 0:
		return ldom.List{}, false
	case 1:
		return items[0], true
	default:
		return items[rand.IntN(len(items))], true
	}
}

func (q *ListQuery) GetByID(
	ctx context.Context,
	id string,
) (ListItemDTO, error) {
	if q == nil || q.listRepo == nil {
		return ListItemDTO{}, errors.New("mall list query: list repo is nil")
	}

	if id == "" {
		return ListItemDTO{}, ldom.ErrNotFound
	}

	l, err := q.listRepo.GetByID(ctx, id)
	if err != nil {
		return ListItemDTO{}, err
	}

	if !isMallPublicListing(l.Status) {
		return ListItemDTO{}, ldom.ErrNotFound
	}

	return q.toListItemDTO(ctx, l)
}

func (q *ListQuery) toListItemDTO(
	ctx context.Context,
	l ldom.List,
) (ListItemDTO, error) {
	inventoryID, productBlueprintID, tokenBlueprintID := extractMallInventoryAndBlueprintIDs(l)

	imageURL, err := q.resolveFirebaseStorageImageURL(ctx, l)
	if err != nil {
		return ListItemDTO{}, err
	}

	return ListItemDTO{
		ID:                 l.ID,
		Title:              l.Title,
		Description:        l.Description,
		Image:              imageURL,
		Prices:             l.Prices,
		InventoryID:        inventoryID,
		ProductBlueprintID: productBlueprintID,
		TokenBlueprintID:   tokenBlueprintID,
	}, nil
}

func (q *ListQuery) resolveFirebaseStorageImageURL(
	ctx context.Context,
	l ldom.List,
) (string, error) {
	if q == nil || q.imageRepo == nil {
		return "", nil
	}

	if l.ID == "" {
		return "", nil
	}

	images, err := q.imageRepo.ListByListID(ctx, l.ID)
	if err != nil {
		return "", err
	}

	return mallshared.SelectPrimaryListImageURL(l, images), nil
}

func extractMallInventoryAndBlueprintIDs(
	l ldom.List,
) (inventoryID string, productBlueprintID string, tokenBlueprintID string) {
	inventoryID = l.InventoryID

	if inventoryID != "" && strings.Contains(inventoryID, "__") {
		parts := strings.SplitN(inventoryID, "__", 2)
		if len(parts) >= 1 {
			productBlueprintID = parts[0]
		}
		if len(parts) == 2 {
			tokenBlueprintID = parts[1]
		}
	}

	return inventoryID, productBlueprintID, tokenBlueprintID
}

func isMallPublicListing(status ldom.ListStatus) bool {
	return strings.EqualFold(string(status), string(ldom.StatusListing))
}
