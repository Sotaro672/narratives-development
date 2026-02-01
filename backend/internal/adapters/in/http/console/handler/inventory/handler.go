// backend/internal/adapters/in/http/console/handler/inventory/handler.go
package inventory

import (
	"net/http"
	"strings"

	invquery "narratives/internal/application/query/console"
	usecase "narratives/internal/application/usecase"
)

type InventoryHandler struct {
	UC *usecase.InventoryUsecase

	// Read-model(Query) for management list (view-only)
	// ✅ only: currentMember.companyId -> productBlueprintIds -> inventories(docId)
	Q *invquery.InventoryQuery

	// ✅ NEW: listCreate 画面用 Query
	LQ *invquery.ListCreateQuery
}

func NewInventoryHandler(uc *usecase.InventoryUsecase, q *invquery.InventoryQuery) *InventoryHandler {
	return &InventoryHandler{UC: uc, Q: q, LQ: nil}
}

// ✅ NEW: ListCreateQuery も注入できるコンストラクタ
func NewInventoryHandlerWithListCreateQuery(
	uc *usecase.InventoryUsecase,
	q *invquery.InventoryQuery,
	lq *invquery.ListCreateQuery,
) *InventoryHandler {
	return &InventoryHandler{UC: uc, Q: q, LQ: lq}
}

func (h *InventoryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")

	// ============================================================
	// Query endpoints (read-only DTO)
	// ============================================================

	// ✅ GET /inventory/list-create/{pbId}/{tbId}
	// ✅ also allow: GET /inventory/list-create/{inventoryId}  (inventoryId="{pbId}__{tbId}")
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

	// ✅ GET /inventory
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

	// ✅ GET /inventory/{id}
	// - /inventory/ids は廃止したため、ここで弾くだけ残す（誤ルーティング防止）
	if strings.HasPrefix(path, "/inventory/") {
		switch r.Method {
		case http.MethodGet:
			id := strings.TrimSpace(strings.TrimPrefix(path, "/inventory/"))
			if id == "" || id == "ids" {
				writeError(w, http.StatusBadRequest, "invalid inventory id")
				return
			}

			// ✅ fallback 削除: Query で確定
			h.GetDetailByIDQuery(w, r, id)
			return

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	}

	// ============================================================
	// CRUD endpoints (domain/usecase)
	// ============================================================

	if path == "/inventories" {
		switch r.Method {
		case http.MethodPost:
			h.Create(w, r)
			return
		case http.MethodGet:
			h.List(w, r)
			return
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	}

	if strings.HasPrefix(path, "/inventories/") {
		switch r.Method {
		case http.MethodGet:
			h.GetByID(w, r)
			return
		case http.MethodPatch:
			h.Update(w, r)
			return
		case http.MethodDelete:
			h.Delete(w, r)
			return
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
}
