// backend/internal/adapters/in/http/console/handler/order_handler.go
package consoleHandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	orderq "narratives/internal/application/query/console"
	common "narratives/internal/domain/common"
	orderdom "narratives/internal/domain/order"
)

// OrderHandler handles:
//   - GET /orders/items
//   - GET /orders/{id}
type OrderHandler struct {
	q       *orderq.OrderManagementQuery
	detailQ *orderq.OrderDetailQuery
}

func NewOrderHandler(
	q *orderq.OrderManagementQuery,
	detailQ *orderq.OrderDetailQuery,
) http.Handler {
	return &OrderHandler{
		q:       q,
		detailQ: detailQ,
	}
}

func (h *OrderHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/orders/items":
		h.listItemRows(w, r)
		return

	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/orders/"):
		id := strings.TrimPrefix(r.URL.Path, "/orders/")
		h.get(w, r, id)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

func (h *OrderHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.Trim(id, " \t\r\n/")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	if h == nil || h.detailQ == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "order_detail_query_not_wired"})
		return
	}

	dto, err := h.detailQ.GetByID(ctx, id)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(dto)
}

func (h *OrderHandler) listItemRows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.q == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "order_management_query_not_wired"})
		return
	}

	filter, page, err := parseOrderListParams(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	var sort common.Sort

	pr, err := h.q.ListItemInventoryRows(ctx, filter, sort, page)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(pr)
}

// ============================================================
// Query param parsing
// ============================================================

func parseOrderListParams(r *http.Request) (orderdom.Filter, common.Page, error) {
	q := r.URL.Query()

	pageNum := parseIntDefault(q.Get("page"), 1)
	perPage := parseIntDefault(q.Get("perPage"), 20)

	f := orderdom.Filter{
		ID: strings.TrimSpace(q.Get("id")),
	}

	if v := strings.TrimSpace(q.Get("userId")); v != "" {
		f.UserID = v
	}
	if v := strings.TrimSpace(q.Get("avatarId")); v != "" {
		f.AvatarID = v
	}
	if v := strings.TrimSpace(q.Get("cartId")); v != "" {
		f.CartID = v
	}
	if v := strings.TrimSpace(q.Get("modelId")); v != "" {
		f.ModelID = v
	}
	if v := strings.TrimSpace(q.Get("inventoryId")); v != "" {
		f.InventoryID = v
	}
	if v := strings.TrimSpace(q.Get("listId")); v != "" {
		f.ListID = v
	}

	if v := strings.TrimSpace(q.Get("createdFrom")); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return orderdom.Filter{}, common.Page{}, errors.New("invalid createdFrom (expected RFC3339)")
		}
		f.CreatedFrom = &t
	}
	if v := strings.TrimSpace(q.Get("createdTo")); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return orderdom.Filter{}, common.Page{}, errors.New("invalid createdTo (expected RFC3339)")
		}
		f.CreatedTo = &t
	}

	p := common.Page{
		Number:  pageNum,
		PerPage: perPage,
	}

	return f, p, nil
}

// ============================================================
// Error handling
// ============================================================

func writeOrderErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, orderdom.ErrInvalidID):
		code = http.StatusBadRequest
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
