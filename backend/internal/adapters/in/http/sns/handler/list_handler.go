// backend/internal/adapters/in/http/sns/handler/list_handler.go
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	ldom "narratives/internal/domain/list"
)

// SNSListHandler serves buyer-facing list endpoints.
// Only returns: title, description, image(url), prices (+ optional inventory/product/token ids).
//
// Routes:
// - GET /sns/lists
// - GET /sns/lists/{id}
type SNSListHandler struct {
	uc *usecase.ListUsecase
}

func NewSNSListHandler(uc *usecase.ListUsecase) http.Handler {
	return &SNSListHandler{uc: uc}
}

// ------------------------------
// Response DTOs (SNS)
// ------------------------------

type SnsListItem struct {
	ID          string              `json:"id"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Image       string              `json:"image"` // URL
	Prices      []ldom.ListPriceRow `json:"prices"`

	// ✅ optional (catalog で inventory を引くための補助)
	InventoryID        string `json:"inventoryId,omitempty"`
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`
}

type SnsListIndexResponse struct {
	Items      []SnsListItem `json:"items"`
	TotalCount int           `json:"totalCount"`
	TotalPages int           `json:"totalPages"`
	Page       int           `json:"page"`
	PerPage    int           `json:"perPage"`
}

// ------------------------------
// http.Handler
// ------------------------------

func (h *SNSListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimSuffix(strings.TrimSpace(r.URL.Path), "/")

	// GET /sns/lists
	if path == "/sns/lists" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		h.listIndex(w, r)
		return
	}

	// GET /sns/lists/{id}
	if strings.HasPrefix(path, "/sns/lists/") {
		rest := strings.TrimPrefix(path, "/sns/lists/")
		parts := strings.Split(rest, "/")
		id := strings.TrimSpace(parts[0])
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

// ------------------------------
// GET /sns/lists
// ------------------------------

func (h *SNSListHandler) listIndex(w http.ResponseWriter, r *http.Request) {
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

	// SNS: public-only safety filter
	var f ldom.Filter
	{
		st := ldom.StatusListing
		f.Status = &st
		deleted := false
		f.Deleted = &deleted
	}
	sort := ldom.Sort{} // default

	result, err := h.uc.List(ctx, f, sort, page)
	if err != nil {
		writeListErr(w, err)
		return
	}

	items := make([]SnsListItem, 0, len(result.Items))
	for _, l := range result.Items {
		if !isPublicListing(l.Status) {
			continue
		}
		items = append(items, toSnsListItem(l))
	}

	writeJSON(w, http.StatusOK, SnsListIndexResponse{
		Items:      items,
		TotalCount: result.TotalCount,
		TotalPages: result.TotalPages,
		Page:       result.Page,
		PerPage:    perPage,
	})
}

// ------------------------------
// GET /sns/lists/{id}
// ------------------------------

func (h *SNSListHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		internalError(w, "usecase is nil")
		return
	}

	l, err := h.uc.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ldom.ErrNotFound) {
			notFound(w)
			return
		}
		writeListErr(w, err)
		return
	}

	// public-only safety
	if !isPublicListing(l.Status) {
		notFound(w)
		return
	}

	writeJSON(w, http.StatusOK, toSnsListItem(l))
}

// ------------------------------
// Mapping
// ------------------------------

func toSnsListItem(l ldom.List) SnsListItem {
	invID, pbID, tbID := extractInventoryAndBlueprintIDs(l)

	return SnsListItem{
		ID:          strings.TrimSpace(l.ID),
		Title:       strings.TrimSpace(l.Title),
		Description: strings.TrimSpace(l.Description),
		Image:       strings.TrimSpace(l.ImageID),
		Prices:      l.Prices,

		InventoryID:        invID,
		ProductBlueprintID: pbID,
		TokenBlueprintID:   tbID,
	}
}

func extractInventoryAndBlueprintIDs(l ldom.List) (inventoryID, productBlueprintID, tokenBlueprintID string) {
	var m map[string]any
	{
		b, err := json.Marshal(l)
		if err == nil {
			_ = json.Unmarshal(b, &m)
		}
	}

	if m != nil {
		if s, ok := getString(m, "inventoryId", "inventoryID", "inventory_id"); ok {
			inventoryID = strings.TrimSpace(s)
		}
		if s, ok := getString(m, "productBlueprintId", "productBlueprintID", "product_blueprint_id"); ok {
			productBlueprintID = strings.TrimSpace(s)
		}
		if s, ok := getString(m, "tokenBlueprintId", "tokenBlueprintID", "token_blueprint_id"); ok {
			tokenBlueprintID = strings.TrimSpace(s)
		}
	}

	if (productBlueprintID == "" || tokenBlueprintID == "") && inventoryID != "" && strings.Contains(inventoryID, "__") {
		parts := strings.SplitN(inventoryID, "__", 2)
		if productBlueprintID == "" {
			productBlueprintID = strings.TrimSpace(parts[0])
		}
		if len(parts) == 2 && tokenBlueprintID == "" {
			tokenBlueprintID = strings.TrimSpace(parts[1])
		}
	}

	return inventoryID, productBlueprintID, tokenBlueprintID
}

func isPublicListing(st ldom.ListStatus) bool {
	return strings.EqualFold(strings.TrimSpace(string(st)), string(ldom.StatusListing))
}

func writeListErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, ldom.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, ldom.ErrConflict):
		code = http.StatusConflict
	default:
		msg := strings.ToLower(strings.TrimSpace(err.Error()))
		if strings.Contains(msg, "invalid") ||
			strings.Contains(msg, "required") ||
			strings.Contains(msg, "must") {
			code = http.StatusBadRequest
		}
	}

	writeJSON(w, code, map[string]string{"error": err.Error()})
}
