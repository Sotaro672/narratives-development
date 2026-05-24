package consoleHandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	invquery "narratives/internal/application/query/console"
	querydto "narratives/internal/application/query/console/dto"
	usecase "narratives/internal/application/usecase"
	invdom "narratives/internal/domain/inventory"
)

type InventoryHandler struct {
	UC *usecase.InventoryUsecase

	// Read-model(Query) for management list (view-only)
	// only: currentMember.companyId -> productBlueprintIds -> inventories(docId)
	Q *invquery.InventoryQuery

	// listCreate 画面用 Query
	LQ *invquery.ListCreateQuery
}

func NewInventoryHandler(uc *usecase.InventoryUsecase, q *invquery.InventoryQuery) *InventoryHandler {
	return &InventoryHandler{UC: uc, Q: q, LQ: nil}
}

// ListCreateQuery も注入できるコンストラクタ
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

// ============================================================
// DTOs
// ============================================================

type createInventoryMintRequest struct {
	TokenBlueprintID   string   `json:"tokenBlueprintId"`
	ProductBlueprintID string   `json:"productBlueprintId"`
	ModelID            string   `json:"modelId"`
	ProductIDs         []string `json:"productIds"`
}

type updateInventoryMintRequest struct {
	ModelID    string   `json:"modelId"`
	ProductIDs []string `json:"productIds"`
}

// ============================================================
// Query endpoints
// ============================================================

func (h *InventoryHandler) ListByCurrentCompanyQuery(w http.ResponseWriter, r *http.Request) {
	if h.Q == nil {
		writeInventoryError(w, http.StatusNotImplemented, "inventory query is not configured")
		return
	}

	ctx := r.Context()

	rows, err := h.Q.ListByCurrentCompany(ctx)
	if err != nil {
		writeInventoryError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 参照だけして import を維持（返却型が interface の場合などに備える）
	_ = querydto.InventoryManagementRowDTO{}

	writeInventoryJSON(w, http.StatusOK, rows)
}

// ============================================================
// ListCreate DTO endpoint
// - GET /inventory/list-create/{inventoryId}
// ============================================================

func (h *InventoryHandler) GetListCreateByPathQuery(w http.ResponseWriter, r *http.Request, path string) {
	if h.LQ == nil {
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
	if h == nil || h.Q == nil {
		writeInventoryError(w, http.StatusNotImplemented, "inventory query is not configured")
		return
	}

	ctx := r.Context()
	if inventoryID == "" {
		writeInventoryError(w, http.StatusBadRequest, "missing id")
		return
	}

	dto, err := h.Q.GetDetailByID(ctx, inventoryID)
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
// CRUD endpoints
// ============================================================

func (h *InventoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	if h.UC == nil {
		writeInventoryError(w, http.StatusNotImplemented, "inventory usecase is not configured")
		return
	}

	ctx := r.Context()

	var req createInventoryMintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeInventoryError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	m, err := h.UC.UpsertFromMintByModel(
		ctx,
		req.TokenBlueprintID,
		req.ProductBlueprintID,
		req.ModelID,
		req.ProductIDs,
	)
	if err != nil {
		writeInventoryDomainError(w, err)
		return
	}

	writeInventoryJSON(w, http.StatusCreated, m)
}

func (h *InventoryHandler) List(w http.ResponseWriter, r *http.Request) {
	if h.UC == nil {
		writeInventoryError(w, http.StatusNotImplemented, "inventory usecase is not configured")
		return
	}

	ctx := r.Context()

	q := r.URL.Query()
	tbID := q.Get("tokenBlueprintId")
	pbID := q.Get("productBlueprintId")
	modelID := q.Get("modelId")

	if tbID == "" && pbID == "" && modelID == "" {
		writeInventoryError(w, http.StatusBadRequest, "tokenBlueprintId or productBlueprintId or modelId is required")
		return
	}

	var (
		list []invdom.Mint
		err  error
	)

	switch {
	case tbID != "" && modelID != "":
		list, err = h.UC.ListByTokenAndModelID(ctx, tbID, modelID)

	case tbID != "" && pbID != "":
		// RepositoryPort に「TB+PB」直クエリが無いので PB で絞ってからフィルタ
		all, e := h.UC.ListByProductBlueprintID(ctx, pbID)
		if e != nil {
			err = e
			break
		}

		tmp := make([]invdom.Mint, 0, len(all))
		for _, m := range all {
			if m.TokenBlueprintID == tbID {
				tmp = append(tmp, m)
			}
		}
		list = tmp

	case tbID != "":
		list, err = h.UC.ListByTokenBlueprintID(ctx, tbID)

	case modelID != "":
		list, err = h.UC.ListByModelID(ctx, modelID)

	default:
		list, err = h.UC.ListByProductBlueprintID(ctx, pbID)
	}

	if err != nil {
		writeInventoryDomainError(w, err)
		return
	}

	writeInventoryJSON(w, http.StatusOK, list)
}

func (h *InventoryHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	if h.UC == nil {
		writeInventoryError(w, http.StatusNotImplemented, "inventory usecase is not configured")
		return
	}

	ctx := r.Context()

	id := inventoryPathParamLast(r.URL.Path)
	if id == "" {
		writeInventoryError(w, http.StatusBadRequest, "missing id")
		return
	}

	m, err := h.UC.GetByID(ctx, id)
	if err != nil {
		writeInventoryDomainError(w, err)
		return
	}

	writeInventoryJSON(w, http.StatusOK, m)
}

func (h *InventoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	if h.UC == nil {
		writeInventoryError(w, http.StatusNotImplemented, "inventory usecase is not configured")
		return
	}

	ctx := r.Context()

	id := inventoryPathParamLast(r.URL.Path)
	if id == "" {
		writeInventoryError(w, http.StatusBadRequest, "missing id")
		return
	}

	var req updateInventoryMintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeInventoryError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	if req.ModelID == "" {
		writeInventoryError(w, http.StatusBadRequest, "modelId is required")
		return
	}
	if len(req.ProductIDs) == 0 {
		writeInventoryError(w, http.StatusBadRequest, "productIds is required")
		return
	}

	current, err := h.UC.GetByID(ctx, id)
	if err != nil {
		writeInventoryDomainError(w, err)
		return
	}

	updated, err := h.UC.UpsertFromMintByModel(
		ctx,
		current.TokenBlueprintID,
		current.ProductBlueprintID,
		req.ModelID,
		req.ProductIDs,
	)
	if err != nil {
		writeInventoryDomainError(w, err)
		return
	}

	writeInventoryJSON(w, http.StatusOK, updated)
}

func (h *InventoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if h.UC == nil {
		writeInventoryError(w, http.StatusNotImplemented, "inventory usecase is not configured")
		return
	}

	ctx := r.Context()

	id := inventoryPathParamLast(r.URL.Path)
	if id == "" {
		writeInventoryError(w, http.StatusBadRequest, "missing id")
		return
	}

	if err := h.UC.Delete(ctx, id); err != nil {
		writeInventoryDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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

func writeInventoryDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, invdom.ErrNotFound):
		writeInventoryError(w, http.StatusNotFound, err.Error())

	case errors.Is(err, invdom.ErrInvalidMintID),
		errors.Is(err, invdom.ErrInvalidTokenBlueprintID),
		errors.Is(err, invdom.ErrInvalidProductBlueprintID),
		errors.Is(err, invdom.ErrInvalidModelID),
		errors.Is(err, invdom.ErrInvalidProducts):
		writeInventoryError(w, http.StatusBadRequest, err.Error())

	default:
		writeInventoryError(w, http.StatusInternalServerError, err.Error())
	}
}

func inventoryPathParamLast(path string) string {
	if path == "" {
		return ""
	}

	path = strings.TrimSuffix(path, "/")
	i := strings.LastIndex(path, "/")
	if i < 0 || i == len(path)-1 {
		return ""
	}

	return path[i+1:]
}

// compile guard
var _ = usecase.InventoryUsecase{}
