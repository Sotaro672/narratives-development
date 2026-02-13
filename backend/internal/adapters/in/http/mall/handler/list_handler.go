// backend/internal/adapters/in/http/mall/handler/list_handler.go
package mallHandler

import (
	"errors"
	"log"
	"net/http"
	"strings"

	listuc "narratives/internal/application/usecase/list"
	ldom "narratives/internal/domain/list"
)

// MallListHandler serves buyer-facing list endpoints.
//
// Routes:
// - GET /mall/lists
// - GET /mall/lists/{id}
type MallListHandler struct {
	uc *listuc.ListUsecase
}

func NewMallListHandler(uc *listuc.ListUsecase) http.Handler {
	return &MallListHandler{uc: uc}
}

// ------------------------------
// Response DTOs (Mall buyer-facing)
// ------------------------------

type MallListItem struct {
	ID          string              `json:"id"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Image       string              `json:"image"` // URL
	Prices      []ldom.ListPriceRow `json:"prices"`

	// optional (catalog で inventory を引くための補助)
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

// ------------------------------
// http.Handler
// ------------------------------

func (h *MallListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// TrimSpace を使わない（そのまま扱う）
	path := strings.TrimSuffix(r.URL.Path, "/")

	// GET /mall/lists
	if path == "/mall/lists" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		h.listIndex(w, r)
		return
	}

	// GET /mall/lists/{id}
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

// ------------------------------
// GET /mall/lists
// ------------------------------

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

	// buyer-facing safety filter: listing & not deleted
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
		log.Printf("[mall][lists] uc.List error page=%d perPage=%d err=%T %v", pageNum, perPage, err, err)
		writeListErr(w, err)
		return
	}

	items := make([]MallListItem, 0, len(result.Items))
	for i, l := range result.Items {
		if !isPublicListing(l.Status) {
			// 500 原因切り分け用：filter漏れや status 不整合の検知
			log.Printf("[mall][lists] skip non-public item index=%d id=%q status=%q", i, l.ID, string(l.Status))
			continue
		}

		it := toMallListItem(l)

		// 500 原因切り分け用：inventoryId / 分解結果が空のケースを検知
		if it.InventoryID == "" {
			log.Printf("[mall][lists] WARN inventoryId empty listId=%q", it.ID)
		} else if (it.ProductBlueprintID == "" || it.TokenBlueprintID == "") && strings.Contains(it.InventoryID, "__") {
			log.Printf("[mall][lists] WARN inventoryId parse incomplete listId=%q inventoryId=%q pbId=%q tbId=%q",
				it.ID, it.InventoryID, it.ProductBlueprintID, it.TokenBlueprintID)
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

// ------------------------------
// GET /mall/lists/{id}
// ------------------------------

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

	// buyer-facing safety
	if !isPublicListing(l.Status) {
		log.Printf("[mall][lists] not public id=%q status=%q", l.ID, string(l.Status))
		notFound(w)
		return
	}

	dto := toMallListItem(l)

	// 500 原因切り分け用：ID解決結果を出す
	if dto.InventoryID == "" {
		log.Printf("[mall][lists] WARN inventoryId empty listId=%q", dto.ID)
	} else if (dto.ProductBlueprintID == "" || dto.TokenBlueprintID == "") && strings.Contains(dto.InventoryID, "__") {
		log.Printf("[mall][lists] WARN inventoryId parse incomplete listId=%q inventoryId=%q pbId=%q tbId=%q",
			dto.ID, dto.InventoryID, dto.ProductBlueprintID, dto.TokenBlueprintID)
	}

	writeJSON(w, http.StatusOK, dto)
}

// ------------------------------
// Mapping
// ------------------------------

func toMallListItem(l ldom.List) MallListItem {
	invID, pbID, tbID := extractInventoryAndBlueprintIDs(l)

	return MallListItem{
		ID:          l.ID,
		Title:       l.Title,
		Description: l.Description,
		Image:       l.ImageID,
		Prices:      l.Prices,

		InventoryID:        invID,
		ProductBlueprintID: pbID,
		TokenBlueprintID:   tbID,
	}
}

func extractInventoryAndBlueprintIDs(l ldom.List) (inventoryID, productBlueprintID, tokenBlueprintID string) {
	// domain/list/entity.go を正とする：InventoryID はフィールドで持っている
	inventoryID = l.InventoryID

	// productBlueprintId / tokenBlueprintId は list ドメインには無い前提なので、
	// inventoryId が "pb__tb" 形式ならそこから解決する（名揺れ吸収はしない）
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

	// 500 原因切り分け用：エラー型とメッセージを必ず出す
	log.Printf("[mall][lists] ERROR status=%d err=%T %v", code, err, err)

	writeJSON(w, code, map[string]string{"error": err.Error()})
}
