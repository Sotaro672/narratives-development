package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	invquery "narratives/internal/application/query"
	usecase "narratives/internal/application/usecase"
	invdom "narratives/internal/domain/inventory"
)

type InventoryHandler struct {
	UC *usecase.InventoryUsecase

	// Read-model(Query) for detail DTO (view-only)
	Q *invquery.InventoryQuery
}

func NewInventoryHandler(uc *usecase.InventoryUsecase, q *invquery.InventoryQuery) *InventoryHandler {
	return &InventoryHandler{UC: uc, Q: q}
}

// 互換: 既存 DI が NewInventoryHandler(uc) を呼ぶ場合
func NewInventoryHandlerUCOnly(uc *usecase.InventoryUsecase) *InventoryHandler {
	return &InventoryHandler{UC: uc, Q: nil}
}

func (h *InventoryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")

	// ✅ 入口ログ：画面から何が来たか
	log.Printf("[inventory_handler] IN %s %s rawPath=%s query=%q",
		r.Method, path, r.URL.Path, r.URL.RawQuery,
	)

	// ============================================================
	// Query endpoints (read-only DTO)
	//
	// ✅ NEW (期待値):
	//   GET /inventory?productBlueprintId={pbId}
	//
	// 既存:
	//   GET /inventory/{inventoryId}  (docId 指定)
	// ============================================================

	// ✅ GET /inventory?productBlueprintId=...
	if path == "/inventory" {
		switch r.Method {
		case http.MethodGet:
			log.Printf("[inventory_handler] route=Query.GetDetailByProductBlueprintIDQuery")
			h.GetDetailByProductBlueprintIDQuery(w, r)
			return
		default:
			log.Printf("[inventory_handler] METHOD_NOT_ALLOWED %s %s", r.Method, path)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	}

	// GET /inventory/{inventoryId}
	if strings.HasPrefix(path, "/inventory/") {
		switch r.Method {
		case http.MethodGet:
			log.Printf("[inventory_handler] route=Query.GetDetail (by inventoryId/docId)")
			h.GetDetail(w, r)
			return
		default:
			log.Printf("[inventory_handler] METHOD_NOT_ALLOWED %s %s", r.Method, path)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	}

	// ============================================================
	// CRUD endpoints (domain/usecase)
	// - /inventories
	// - /inventories/{id}
	// ============================================================

	if path == "/inventories" {
		switch r.Method {
		case http.MethodPost:
			log.Printf("[inventory_handler] route=CRUD.Create")
			h.Create(w, r)
			return
		case http.MethodGet:
			log.Printf("[inventory_handler] route=CRUD.List")
			h.List(w, r)
			return
		default:
			log.Printf("[inventory_handler] METHOD_NOT_ALLOWED %s %s", r.Method, path)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	}

	if strings.HasPrefix(path, "/inventories/") {
		switch r.Method {
		case http.MethodGet:
			log.Printf("[inventory_handler] route=CRUD.GetByID")
			h.GetByID(w, r)
			return
		case http.MethodPatch:
			log.Printf("[inventory_handler] route=CRUD.Update")
			h.Update(w, r)
			return
		case http.MethodDelete:
			log.Printf("[inventory_handler] route=CRUD.Delete")
			h.Delete(w, r)
			return
		default:
			log.Printf("[inventory_handler] METHOD_NOT_ALLOWED %s %s", r.Method, path)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	}

	log.Printf("[inventory_handler] NOT_FOUND %s %s", r.Method, path)
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
	Accumulation       int      `json:"accumulation"`
}

type updateInventoryMintRequest struct {
	Products     []string `json:"products"`
	Accumulation *int     `json:"accumulation,omitempty"`
}

// ============================================================
// Query Handlers (Read-only DTO)
// ============================================================

// GET /inventory/{inventoryId}
func (h *InventoryHandler) GetDetail(w http.ResponseWriter, r *http.Request) {
	if h.Q == nil {
		log.Printf("[inventory_handler][GetDetail] QueryNotConfigured")
		writeError(w, http.StatusNotImplemented, "inventory query is not configured")
		return
	}

	ctx := r.Context()

	id := strings.TrimSpace(pathParamLast(r.URL.Path))
	log.Printf("[inventory_handler][GetDetail] start inventoryId=%q", id)

	if id == "" {
		log.Printf("[inventory_handler][GetDetail] BAD_REQUEST missing inventoryId")
		writeError(w, http.StatusBadRequest, "missing inventoryId")
		return
	}

	dto, err := h.Q.GetDetail(ctx, id)
	if err != nil {
		log.Printf("[inventory_handler][GetDetail] failed inventoryId=%q err=%v", id, err)
		writeQueryError(w, err)
		return
	}

	log.Printf("[inventory_handler][GetDetail] ok inventoryId=%q pbId=%q rows=%d totalStock=%d",
		dto.InventoryID, dto.ProductBlueprintID, len(dto.Rows), dto.TotalStock,
	)
	writeJSON(w, http.StatusOK, dto)
}

// ✅ NEW: GET /inventory?productBlueprintId={pbId}
func (h *InventoryHandler) GetDetailByProductBlueprintIDQuery(w http.ResponseWriter, r *http.Request) {
	if h.Q == nil {
		log.Printf("[inventory_handler][GetDetailByPBID] QueryNotConfigured")
		writeError(w, http.StatusNotImplemented, "inventory query is not configured")
		return
	}

	ctx := r.Context()

	pbID := strings.TrimSpace(r.URL.Query().Get("productBlueprintId"))
	log.Printf("[inventory_handler][GetDetailByPBID] start productBlueprintId=%q rawQuery=%q", pbID, r.URL.RawQuery)

	if pbID == "" {
		log.Printf("[inventory_handler][GetDetailByPBID] BAD_REQUEST productBlueprintId is required")
		writeError(w, http.StatusBadRequest, "productBlueprintId is required")
		return
	}

	dto, err := h.Q.GetDetailByProductBlueprintID(ctx, pbID)
	if err != nil {
		log.Printf("[inventory_handler][GetDetailByPBID] failed productBlueprintId=%q err=%v", pbID, err)
		writeQueryError(w, err)
		return
	}

	log.Printf("[inventory_handler][GetDetailByPBID] ok resolved inventoryId=%q pbId=%q rows=%d totalStock=%d",
		dto.InventoryID, dto.ProductBlueprintID, len(dto.Rows), dto.TotalStock,
	)
	writeJSON(w, http.StatusOK, dto)
}

// query error mapping
func writeQueryError(w http.ResponseWriter, err error) {
	msg := err.Error()
	lower := strings.ToLower(msg)

	switch {
	case errors.Is(err, invdom.ErrNotFound):
		log.Printf("[inventory_handler][QueryError] 404 ErrNotFound err=%v", err)
		writeError(w, http.StatusNotFound, msg)

	case errors.Is(err, invdom.ErrInvalidMintID),
		errors.Is(err, invdom.ErrInvalidTokenBlueprintID),
		errors.Is(err, invdom.ErrInvalidProductBlueprintID),
		errors.Is(err, invdom.ErrInvalidModelID),
		errors.Is(err, invdom.ErrInvalidProducts),
		errors.Is(err, invdom.ErrInvalidAccumulation):
		log.Printf("[inventory_handler][QueryError] 400 DomainValidation err=%v", err)
		writeError(w, http.StatusBadRequest, msg)

	// errors.New("... not found ...") の救済
	case strings.Contains(lower, "not found"):
		log.Printf("[inventory_handler][QueryError] 404 ContainsNotFound err=%v", err)
		writeError(w, http.StatusNotFound, msg)

	default:
		log.Printf("[inventory_handler][QueryError] 500 err=%v", err)
		writeError(w, http.StatusInternalServerError, msg)
	}
}

// ============================================================
// CRUD Handlers (Usecase)
// ============================================================

func (h *InventoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req createInventoryMintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[inventory_handler][Create] BAD_REQUEST invalid json err=%v", err)
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	log.Printf("[inventory_handler][Create] start tbId=%q pbId=%q modelId=%q products=%d accumulation=%d",
		strings.TrimSpace(req.TokenBlueprintID),
		strings.TrimSpace(req.ProductBlueprintID),
		strings.TrimSpace(req.ModelID),
		len(req.ProductIDs),
		req.Accumulation,
	)

	m, err := h.UC.CreateMint(
		ctx,
		req.TokenBlueprintID,
		req.ProductBlueprintID,
		req.ModelID,
		req.ProductIDs,
		req.Accumulation,
	)
	if err != nil {
		log.Printf("[inventory_handler][Create] failed err=%v", err)
		writeDomainError(w, err)
		return
	}

	log.Printf("[inventory_handler][Create] ok id=%q pbId=%q modelId=%q products=%d accumulation=%d",
		m.ID, m.ProductBlueprintID, m.ModelID, len(m.Products), m.Accumulation,
	)

	writeJSON(w, http.StatusCreated, m)
}

func (h *InventoryHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	q := r.URL.Query()
	tbID := strings.TrimSpace(q.Get("tokenBlueprintId"))
	pbID := strings.TrimSpace(q.Get("productBlueprintId"))
	modelID := strings.TrimSpace(q.Get("modelId"))

	log.Printf("[inventory_handler][List] start tbId=%q pbId=%q modelId=%q rawQuery=%q",
		tbID, pbID, modelID, r.URL.RawQuery,
	)

	if tbID == "" && pbID == "" && modelID == "" {
		log.Printf("[inventory_handler][List] BAD_REQUEST missing filters")
		writeError(w, http.StatusBadRequest, "tokenBlueprintId or productBlueprintId or modelId is required")
		return
	}

	var (
		list []invdom.Mint
		err  error
	)

	switch {
	case tbID != "" && modelID != "":
		list, err = h.UC.ListByTokenAndModelID(ctx, tbID, modelID)
	case tbID != "":
		list, err = h.UC.ListByTokenBlueprintID(ctx, tbID)
	case modelID != "":
		list, err = h.UC.ListByModelID(ctx, modelID)
	default:
		list, err = h.UC.ListByProductBlueprintID(ctx, pbID)
	}

	if err != nil {
		log.Printf("[inventory_handler][List] failed err=%v", err)
		writeDomainError(w, err)
		return
	}

	sample := ""
	if len(list) > 0 {
		sample = list[0].ID
	}
	log.Printf("[inventory_handler][List] ok count=%d sampleId=%q", len(list), sample)

	writeJSON(w, http.StatusOK, list)
}

func (h *InventoryHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := strings.TrimSpace(pathParamLast(r.URL.Path))
	log.Printf("[inventory_handler][GetByID] start id=%q", id)

	if id == "" {
		log.Printf("[inventory_handler][GetByID] BAD_REQUEST missing id")
		writeError(w, http.StatusBadRequest, "missing id")
		return
	}

	m, err := h.UC.GetMintByID(ctx, id)
	if err != nil {
		log.Printf("[inventory_handler][GetByID] failed id=%q err=%v", id, err)
		writeDomainError(w, err)
		return
	}

	log.Printf("[inventory_handler][GetByID] ok id=%q pbId=%q modelId=%q products=%d accumulation=%d",
		m.ID, m.ProductBlueprintID, m.ModelID, len(m.Products), m.Accumulation,
	)

	writeJSON(w, http.StatusOK, m)
}

func (h *InventoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := strings.TrimSpace(pathParamLast(r.URL.Path))
	log.Printf("[inventory_handler][Update] start id=%q", id)

	if id == "" {
		log.Printf("[inventory_handler][Update] BAD_REQUEST missing id")
		writeError(w, http.StatusBadRequest, "missing id")
		return
	}

	var req updateInventoryMintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[inventory_handler][Update] BAD_REQUEST invalid json err=%v", err)
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	log.Printf("[inventory_handler][Update] payload id=%q productsProvided=%t products=%d accumulationProvided=%t",
		id,
		req.Products != nil,
		len(req.Products),
		req.Accumulation != nil,
	)

	current, err := h.UC.GetMintByID(ctx, id)
	if err != nil {
		log.Printf("[inventory_handler][Update] failed to load current id=%q err=%v", id, err)
		writeDomainError(w, err)
		return
	}

	if req.Products != nil {
		current.Products = req.Products
		if req.Accumulation == nil {
			current.Accumulation = 0
		}
	}
	if req.Accumulation != nil {
		current.Accumulation = *req.Accumulation
	}

	updated, err := h.UC.UpdateMint(ctx, current)
	if err != nil {
		log.Printf("[inventory_handler][Update] failed id=%q err=%v", id, err)
		writeDomainError(w, err)
		return
	}

	log.Printf("[inventory_handler][Update] ok id=%q products=%d accumulation=%d",
		updated.ID, len(updated.Products), updated.Accumulation,
	)

	writeJSON(w, http.StatusOK, updated)
}

func (h *InventoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := strings.TrimSpace(pathParamLast(r.URL.Path))
	log.Printf("[inventory_handler][Delete] start id=%q", id)

	if id == "" {
		log.Printf("[inventory_handler][Delete] BAD_REQUEST missing id")
		writeError(w, http.StatusBadRequest, "missing id")
		return
	}

	if err := h.UC.DeleteMint(ctx, id); err != nil {
		log.Printf("[inventory_handler][Delete] failed id=%q err=%v", id, err)
		writeDomainError(w, err)
		return
	}

	log.Printf("[inventory_handler][Delete] ok id=%q", id)
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================
// Helpers
// ============================================================

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	// ✅ どのエラーを返したかをログ
	log.Printf("[inventory_handler] RESP_ERROR status=%d msg=%q", status, msg)

	writeJSON(w, status, map[string]any{
		"error": msg,
	})
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, invdom.ErrNotFound):
		log.Printf("[inventory_handler][DomainError] 404 err=%v", err)
		writeError(w, http.StatusNotFound, err.Error())

	case errors.Is(err, invdom.ErrInvalidMintID),
		errors.Is(err, invdom.ErrInvalidTokenBlueprintID),
		errors.Is(err, invdom.ErrInvalidProductBlueprintID),
		errors.Is(err, invdom.ErrInvalidModelID),
		errors.Is(err, invdom.ErrInvalidProducts),
		errors.Is(err, invdom.ErrInvalidAccumulation):
		log.Printf("[inventory_handler][DomainError] 400 err=%v", err)
		writeError(w, http.StatusBadRequest, err.Error())

	default:
		log.Printf("[inventory_handler][DomainError] 500 err=%v", err)
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func pathParamLast(path string) string {
	path = strings.TrimSpace(path)
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
