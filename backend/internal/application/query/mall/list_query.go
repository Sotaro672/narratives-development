// backend/internal/application/query/mall/list_query.go
package mall

import (
	"context"
	"errors"
	"sort"
	"strings"

	ldom "narratives/internal/domain/list"
)

type ListQuery struct {
	listRepo  ldom.Repository
	imageRepo ldom.ImageRepository
}

func NewListQuery(
	listRepo ldom.Repository,
	imageRepo ldom.ImageRepository,
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

	if pageNum <= 0 {
		pageNum = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 50 {
		perPage = 50
	}

	var filter ldom.Filter
	status := ldom.StatusListing
	filter.Status = &status

	result, err := q.listRepo.List(
		ctx,
		filter,
		ldom.Sort{},
		ldom.Page{
			Number:  pageNum,
			PerPage: perPage,
		},
	)
	if err != nil {
		return ListIndexResponseDTO{}, err
	}

	items := make([]ListItemDTO, 0, len(result.Items))
	for _, l := range result.Items {
		if !isMallPublicListing(l.Status) {
			continue
		}

		item, err := q.toListItemDTO(ctx, l)
		if err != nil {
			return ListIndexResponseDTO{}, err
		}

		items = append(items, item)
	}

	return ListIndexResponseDTO{
		Items:      items,
		TotalCount: result.TotalCount,
		TotalPages: result.TotalPages,
		Page:       result.Page,
		PerPage:    perPage,
	}, nil
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

	if len(images) == 0 {
		return "", nil
	}

	primaryImageID := l.ImageID
	if primaryImageID != "" {
		for _, img := range images {
			if img.ID == primaryImageID {
				return img.URL, nil
			}
		}
	}

	sort.SliceStable(images, func(i, j int) bool {
		if images[i].DisplayOrder != images[j].DisplayOrder {
			return images[i].DisplayOrder < images[j].DisplayOrder
		}

		if !images[i].CreatedAt.Equal(images[j].CreatedAt) {
			return images[i].CreatedAt.Before(images[j].CreatedAt)
		}

		return images[i].ID < images[j].ID
	})

	for _, img := range images {
		if img.URL != "" {
			return img.URL, nil
		}
	}

	return "", nil
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
