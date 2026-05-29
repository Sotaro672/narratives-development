// backend/internal/adapters/in/http/console/handler/inventory_handler.go
package consoleHandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	invquery "narratives/internal/application/query/console"
	invdom "narratives/internal/domain/inventory"
)

type InventoryHandler struct {
	// Read-model(Query) for management list (view-only)
	// only: currentMember.companyId -> productBlueprintIds -> inventories(docId)
	MQ *invquery.InventoryManagementQuery

	// Read-model(Query) for inventory detail
	DQ *invquery.InventoryDetailQuery

	// listCreate 画面用 Query
	LQ *invquery.ListCreateQuery
}

func NewInventoryHandler(
	mq *invquery.InventoryManagementQuery,
	dq *invquery.InventoryDetailQuery,
) *InventoryHandler {
	return &InventoryHandler{
		MQ: mq,
		DQ: dq,
		LQ: nil,
	}
}

// ListCreateQuery も注入できるコンストラクタ
func NewInventoryHandlerWithListCreateQuery(
	mq *invquery.InventoryManagementQuery,
	dq *invquery.InventoryDetailQuery,
	lq *invquery.ListCreateQuery,
) *InventoryHandler {
	return &InventoryHandler{
		MQ: mq,
		DQ: dq,
		LQ: lq,
	}
}

func (h *InventoryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")

	// ============================================================
	// Query endpoints (read-only DTO)
	// ============================================================

	// GET /inventory/list-create/{inventoryId}
	if strings.HasPrefix(path, "/inventory/list-create/") {
		switch r.Method {
		case http.MethodGet:
			h.GetListCreateByPathQuery(w, r, path)
			return
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	}

	// GET /inventory
	if path == "/inventory" {
		switch r.Method {
		case http.MethodGet:
			h.ListByCurrentCompanyQuery(w, r)
			return
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	}

	// GET /inventory/{id}
	// /inventory/ids は廃止したため、ここで弾くだけ残す（誤ルーティング防止）
	if strings.HasPrefix(path, "/inventory/") {
		switch r.Method {
		case http.MethodGet:
			id := strings.TrimPrefix(path, "/inventory/")
			if id == "" || id == "ids" {
				writeInventoryError(w, http.StatusBadRequest, "invalid inventory id")
				return
			}

			// fallback 削除: Query で確定
			h.GetDetailByIDQuery(w, r, id)
			return

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
}

// ============================================================
// Query endpoints
// ============================================================

func (h *InventoryHandler) ListByCurrentCompanyQuery(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.MQ == nil {
		writeInventoryError(w, http.StatusNotImplemented, "inventory management query is not configured")
		return
	}

	ctx := r.Context()

	rows, err := h.MQ.ListByCurrentCompany(ctx)
	if err != nil {
		writeInventoryError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeInventoryJSON(w, http.StatusOK, rows)
}

// ============================================================
// ListCreate DTO endpoint
// - GET /inventory/list-create/{inventoryId}
// ============================================================

func (h *InventoryHandler) GetListCreateByPathQuery(w http.ResponseWriter, r *http.Request, path string) {
	if h == nil || h.LQ == nil {
		writeInventoryError(w, http.StatusNotImplemented, "list create query is not configured")
		return
	}

	ctx := r.Context()

	rest := strings.TrimPrefix(path, "/inventory/list-create/")
	rest = strings.Trim(rest, "/")
	if rest == "" {
		writeInventoryError(w, http.StatusBadRequest, "missing params")
		return
	}

	// inventoryId は docId をそのまま受け取る（pb/tb を path で受けない）
	inventoryID := rest
	if inventoryID == "" {
		writeInventoryError(w, http.StatusBadRequest, "inventoryId is required")
		return
	}

	dto, err := h.LQ.GetByInventoryID(ctx, inventoryID)
	if err != nil {
		// validation系は 400、それ以外は 500 に寄せる
		if isInventoryProbablyBadRequest(err) {
			writeInventoryError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeInventoryError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeInventoryJSON(w, http.StatusOK, dto)
}

func isInventoryProbablyBadRequest(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "required") ||
		strings.Contains(msg, "missing") ||
		strings.Contains(msg, "invalid")
}

// ============================================================
// Detail endpoint（確定）
// - Query が必須（fallback は削除）
// ============================================================

func (h *InventoryHandler) GetDetailByIDQuery(w http.ResponseWriter, r *http.Request, inventoryID string) {
	if h == nil || h.DQ == nil {
		writeInventoryError(w, http.StatusNotImplemented, "inventory detail query is not configured")
		return
	}

	ctx := r.Context()
	if inventoryID == "" {
		writeInventoryError(w, http.StatusBadRequest, "missing id")
		return
	}

	dto, err := h.DQ.GetDetailByID(ctx, inventoryID)
	if err != nil {
		if errors.Is(err, invdom.ErrNotFound) {
			writeInventoryError(w, http.StatusNotFound, err.Error())
			return
		}
		writeInventoryError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeInventoryJSON(w, http.StatusOK, dto)
}

// ============================================================
// HTTP helpers
// ============================================================

func writeInventoryJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeInventoryError(w http.ResponseWriter, status int, msg string) {
	writeInventoryJSON(w, status, map[string]any{"error": msg})
}
