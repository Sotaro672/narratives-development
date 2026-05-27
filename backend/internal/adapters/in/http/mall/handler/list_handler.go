package mallHandler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sort"
	"strings"

	usecase "narratives/internal/application/usecase"
	ldom "narratives/internal/domain/list"
)

type MallListHandler struct {
	uc *usecase.ListUsecase
}

func NewMallListHandler(uc *usecase.ListUsecase) http.Handler {
	return &MallListHandler{uc: uc}
}

type MallListItem struct {
	ID          string              `json:"id"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Image       string              `json:"image"`
	Prices      []ldom.ListPriceRow `json:"prices"`

	InventoryID        string `json:"inventoryId,omitempty"`
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`
}

type MallListIndexResponse struct {
	Items      []MallListItem `json:"items"`
	TotalCount int            `json:"totalCount"`
	TotalPages int            `json:"totalPages"`
	Page       int            `json:"page"`
	PerPage    int            `json:"perPage"`
}

func (h *MallListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimSuffix(r.URL.Path, "/")

	if path == "/mall/lists" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		h.listIndex(w, r)
		return
	}

	if strings.HasPrefix(path, "/mall/lists/") {
		rest := strings.TrimPrefix(path, "/mall/lists/")
		parts := strings.Split(rest, "/")
		id := parts[0]
		if id == "" {
			badRequest(w, "invalid id")
			return
		}
		if len(parts) > 1 {
			notFound(w)
			return
		}
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		h.get(w, r, id)
		return
	}

	notFound(w)
}

func (h *MallListHandler) listIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		internalError(w, "usecase is nil")
		return
	}

	qp := r.URL.Query()
	pageNum := parseIntDefault(qp.Get("page"), 1)
	perPage := parseIntDefault(qp.Get("perPage"), 20)
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 50 {
		perPage = 50
	}

	page := ldom.Page{Number: pageNum, PerPage: perPage}

	var f ldom.Filter
	st := ldom.StatusListing
	f.Status = &st

	sortCond := ldom.Sort{}

	result, err := h.uc.List(ctx, f, sortCond, page)
	if err != nil {
		log.Printf("[mall][lists] uc.List error page=%d perPage=%d err=%T %v", pageNum, perPage, err, err)
		writeListErr(w, err)
		return
	}

	items := make([]MallListItem, 0, len(result.Items))
	for i, l := range result.Items {
		if !isPublicListing(l.Status) {
			log.Printf("[mall][lists] skip non-public item index=%d id=%q status=%q", i, l.ID, string(l.Status))
			continue
		}

		it, err := h.toMallListItem(ctx, l)
		if err != nil {
			log.Printf("[mall][lists] resolve image error listId=%q err=%T %v", l.ID, err, err)
			writeListErr(w, err)
			return
		}

		if it.InventoryID == "" {
			log.Printf("[mall][lists] WARN inventoryId empty listId=%q", it.ID)
		} else if (it.ProductBlueprintID == "" || it.TokenBlueprintID == "") && strings.Contains(it.InventoryID, "__") {
			log.Printf(
				"[mall][lists] WARN inventoryId parse incomplete listId=%q inventoryId=%q pbId=%q tbId=%q",
				it.ID,
				it.InventoryID,
				it.ProductBlueprintID,
				it.TokenBlueprintID,
			)
		}

		if it.Image == "" {
			log.Printf("[mall][lists] WARN firebase storage image url empty listId=%q imageId=%q", l.ID, l.ImageID)
		}

		items = append(items, it)
	}

	resp := MallListIndexResponse{
		Items:      items,
		TotalCount: result.TotalCount,
		TotalPages: result.TotalPages,
		Page:       result.Page,
		PerPage:    perPage,
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *MallListHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		internalError(w, "usecase is nil")
		return
	}

	l, err := h.uc.GetByID(ctx, id)
	if err != nil {
		log.Printf("[mall][lists] uc.GetByID error id=%q err=%T %v", id, err, err)
		if errors.Is(err, ldom.ErrNotFound) {
			notFound(w)
			return
		}
		writeListErr(w, err)
		return
	}

	if !isPublicListing(l.Status) {
		log.Printf("[mall][lists] not public id=%q status=%q", l.ID, string(l.Status))
		notFound(w)
		return
	}

	dto, err := h.toMallListItem(ctx, l)
	if err != nil {
		log.Printf("[mall][lists] resolve image error listId=%q err=%T %v", l.ID, err, err)
		writeListErr(w, err)
		return
	}

	if dto.InventoryID == "" {
		log.Printf("[mall][lists] WARN inventoryId empty listId=%q", dto.ID)
	} else if (dto.ProductBlueprintID == "" || dto.TokenBlueprintID == "") && strings.Contains(dto.InventoryID, "__") {
		log.Printf(
			"[mall][lists] WARN inventoryId parse incomplete listId=%q inventoryId=%q pbId=%q tbId=%q",
			dto.ID,
			dto.InventoryID,
			dto.ProductBlueprintID,
			dto.TokenBlueprintID,
		)
	}

	if dto.Image == "" {
		log.Printf("[mall][lists] WARN firebase storage image url empty listId=%q imageId=%q", l.ID, l.ImageID)
	}

	writeJSON(w, http.StatusOK, dto)
}

func (h *MallListHandler) toMallListItem(ctx context.Context, l ldom.List) (MallListItem, error) {
	invID, pbID, tbID := extractInventoryAndBlueprintIDs(l)

	imageURL, err := h.resolveFirebaseStorageImageURL(ctx, l)
	if err != nil {
		return MallListItem{}, err
	}

	return MallListItem{
		ID:                 l.ID,
		Title:              l.Title,
		Description:        l.Description,
		Image:              imageURL,
		Prices:             l.Prices,
		InventoryID:        invID,
		ProductBlueprintID: pbID,
		TokenBlueprintID:   tbID,
	}, nil
}

func (h *MallListHandler) resolveFirebaseStorageImageURL(ctx context.Context, l ldom.List) (string, error) {
	if h == nil || h.uc == nil {
		return "", errors.New("usecase is nil")
	}

	images, err := h.uc.GetImages(ctx, l.ID)
	if err != nil {
		return "", err
	}

	if len(images) == 0 {
		return "", nil
	}

	primaryImageID := strings.TrimSpace(l.ImageID)
	if primaryImageID != "" {
		for _, img := range images {
			if strings.TrimSpace(img.ID) == primaryImageID {
				return strings.TrimSpace(img.URL), nil
			}
		}

		log.Printf("[mall][lists] WARN primary image not found listId=%q imageId=%q", l.ID, primaryImageID)
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
		if url := strings.TrimSpace(img.URL); url != "" {
			return url, nil
		}
	}

	return "", nil
}

func extractInventoryAndBlueprintIDs(l ldom.List) (inventoryID, productBlueprintID, tokenBlueprintID string) {
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

func isPublicListing(st ldom.ListStatus) bool {
	return strings.EqualFold(string(st), string(ldom.StatusListing))
}

func writeListErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, ldom.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, ldom.ErrConflict):
		code = http.StatusConflict
	default:
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "invalid") ||
			strings.Contains(msg, "required") ||
			strings.Contains(msg, "must") {
			code = http.StatusBadRequest
		}
	}

	log.Printf("[mall][lists] ERROR status=%d err=%T %v", code, err, err)

	writeJSON(w, code, map[string]string{"error": err.Error()})
}
