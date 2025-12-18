// backend/internal/adapters/in/http/handlers/inventory_handler.go
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"reflect"
	"strings"

	invquery "narratives/internal/application/query"
	querydto "narratives/internal/application/query/dto"
	usecase "narratives/internal/application/usecase"
	invdom "narratives/internal/domain/inventory"
)

type InventoryHandler struct {
	UC *usecase.InventoryUsecase

	// Read-model(Query) for management list (view-only)
	// ✅ only: currentMember.companyId -> productBlueprintIds -> inventories(docId)
	Q *invquery.InventoryQuery
}

func NewInventoryHandler(uc *usecase.InventoryUsecase, q *invquery.InventoryQuery) *InventoryHandler {
	return &InventoryHandler{UC: uc, Q: q}
}

func (h *InventoryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")

	// ✅ 入口ログ：画面から何が来たか
	log.Printf("[inventory_handler] IN %s %s rawPath=%s query=%q",
		r.Method, path, r.URL.Path, r.URL.RawQuery,
	)

	// ============================================================
	// Query endpoints (read-only DTO)
	// ============================================================

	// ✅ GET /inventory/ids  (pbId+tbId -> inventoryIds)
	if path == "/inventory/ids" {
		switch r.Method {
		case http.MethodGet:
			log.Printf("[inventory_handler] route=Query.ResolveInventoryIDsByProductAndToken")
			h.ResolveInventoryIDsByProductAndTokenQuery(w, r)
			return
		default:
			log.Printf("[inventory_handler] METHOD_NOT_ALLOWED %s %s", r.Method, path)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	}

	// ✅ GET /inventory  (currentMember.companyId -> productBlueprintIds -> inventories)
	if path == "/inventory" {
		switch r.Method {
		case http.MethodGet:
			log.Printf("[inventory_handler] route=Query.ListByCurrentCompany")
			h.ListByCurrentCompanyQuery(w, r)
			return
		default:
			log.Printf("[inventory_handler] METHOD_NOT_ALLOWED %s %s", r.Method, path)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	}

	// ✅ NEW: GET /inventory/{id}  (detail)
	// - /inventory/ids は上で処理済みなので、ここでは /inventory/{anything} を detail として扱う
	if strings.HasPrefix(path, "/inventory/") {
		switch r.Method {
		case http.MethodGet:
			id := strings.TrimSpace(strings.TrimPrefix(path, "/inventory/"))
			if id == "" || id == "ids" {
				log.Printf("[inventory_handler] BAD_REQUEST invalid inventory id=%q", id)
				writeError(w, http.StatusBadRequest, "invalid inventory id")
				return
			}
			log.Printf("[inventory_handler] route=QueryOrFallback.Detail id=%q", id)
			h.GetDetailByIDQueryOrFallback(w, r, id)
			return
		default:
			log.Printf("[inventory_handler] METHOD_NOT_ALLOWED %s %s", r.Method, path)
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
			log.Printf("[inventory_handler] route=CRUD.Upsert (POST /inventories)")
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
			log.Printf("[inventory_handler] route=CRUD.UpsertModelStock (PATCH /inventories/{id})")
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
}

type updateInventoryMintRequest struct {
	ModelID    string   `json:"modelId"`
	ProductIDs []string `json:"productIds"`
}

// ============================================================
// Query Handler (Read-only DTO)
// ============================================================

func (h *InventoryHandler) ListByCurrentCompanyQuery(w http.ResponseWriter, r *http.Request) {
	if h.Q == nil {
		log.Printf("[inventory_handler][ListByCurrentCompany] QueryNotConfigured")
		writeError(w, http.StatusNotImplemented, "inventory query is not configured")
		return
	}

	ctx := r.Context()
	log.Printf("[inventory_handler][ListByCurrentCompany] start rawQuery=%q", r.URL.RawQuery)

	rows, err := h.Q.ListByCurrentCompany(ctx)
	if err != nil {
		log.Printf("[inventory_handler][ListByCurrentCompany] failed err=%v", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sample := ""
	if len(rows) > 0 {
		sample = rows[0].ProductBlueprintID
	}
	log.Printf("[inventory_handler][ListByCurrentCompany] ok count=%d samplePbId=%q", len(rows), sample)

	_ = querydto.InventoryManagementRowDTO{}
	writeJSON(w, http.StatusOK, rows)
}

// ✅ NEW: pbId + tbId -> inventoryIds
func (h *InventoryHandler) ResolveInventoryIDsByProductAndTokenQuery(w http.ResponseWriter, r *http.Request) {
	if h.Q == nil {
		log.Printf("[inventory_handler][ResolveInventoryIDsByProductAndToken] QueryNotConfigured")
		writeError(w, http.StatusNotImplemented, "inventory query is not configured")
		return
	}

	ctx := r.Context()

	pbID := strings.TrimSpace(r.URL.Query().Get("productBlueprintId"))
	tbID := strings.TrimSpace(r.URL.Query().Get("tokenBlueprintId"))

	log.Printf("[inventory_handler][ResolveInventoryIDsByProductAndToken] start pbId=%q tbId=%q rawQuery=%q",
		pbID, tbID, r.URL.RawQuery,
	)

	if pbID == "" || tbID == "" {
		writeError(w, http.StatusBadRequest, "productBlueprintId and tokenBlueprintId are required")
		return
	}

	ids, err := h.Q.ListInventoryIDsByProductAndToken(ctx, pbID, tbID)
	if err != nil {
		log.Printf("[inventory_handler][ResolveInventoryIDsByProductAndToken] failed err=%v", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := querydto.InventoryIDsByProductAndTokenDTO{
		ProductBlueprintID: pbID,
		TokenBlueprintID:   tbID,
		InventoryIDs:       ids,
	}

	log.Printf("[inventory_handler][ResolveInventoryIDsByProductAndToken] ok count=%d sample=%v",
		len(ids), func() []string {
			if len(ids) > 5 {
				return ids[:5]
			}
			return ids
		}(),
	)

	writeJSON(w, http.StatusOK, resp)
}

// ============================================================
// ✅ NEW: Detail endpoint (Query preferred, fallback to UC)
// GET /inventory/{id}
// ============================================================

func (h *InventoryHandler) GetDetailByIDQueryOrFallback(w http.ResponseWriter, r *http.Request, inventoryID string) {
	ctx := r.Context()
	id := strings.TrimSpace(inventoryID)

	log.Printf("[inventory_handler][Detail] start id=%q", id)

	if id == "" {
		writeError(w, http.StatusBadRequest, "missing id")
		return
	}

	// 1) Query があれば detail DTO を試す（メソッド名の揺れを reflect で吸収）
	if h.Q != nil {
		if dto, ok, err := tryCallInventoryQueryDetail(ctx, h.Q, id); ok {
			if err != nil {
				log.Printf("[inventory_handler][Detail] query failed id=%q err=%v", id, err)
				// not found を 404 に寄せたい場合はここで判定してもOK
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			log.Printf("[inventory_handler][Detail] query ok id=%q", id)
			writeJSON(w, http.StatusOK, dto)
			return
		}
		log.Printf("[inventory_handler][Detail] query detail method not found (fallback to usecase) id=%q", id)
	}

	// 2) fallback: UC.GetByID（※DTOの完全一致は Query 側推奨）
	if h.UC == nil {
		log.Printf("[inventory_handler][Detail] UsecaseNotConfigured")
		writeError(w, http.StatusNotImplemented, "inventory usecase is not configured")
		return
	}

	m, err := h.UC.GetByID(ctx, id)
	if err != nil {
		log.Printf("[inventory_handler][Detail] usecase failed id=%q err=%v", id, err)
		writeDomainError(w, err)
		return
	}

	// fallback DTO（最低限）
	// フロントの mapper は rows が無くても落ちないので、まず 200 を返すことを優先。
	resp := map[string]any{
		"inventoryId":        strings.TrimSpace(m.ID),
		"id":                 strings.TrimSpace(m.ID),
		"inventoryIds":       []string{strings.TrimSpace(m.ID)},
		"tokenBlueprintId":   strings.TrimSpace(m.TokenBlueprintID),
		"productBlueprintId": strings.TrimSpace(m.ProductBlueprintID),
		"modelId":            "",

		"productBlueprintPatch": map[string]any{},
		"rows":                  []any{},
		"totalStock":            totalProducts(m),
	}

	log.Printf("[inventory_handler][Detail] fallback ok id=%q pbId=%q tbId=%q totalStock=%v",
		strings.TrimSpace(m.ID),
		strings.TrimSpace(m.ProductBlueprintID),
		strings.TrimSpace(m.TokenBlueprintID),
		resp["totalStock"],
	)

	writeJSON(w, http.StatusOK, resp)
}

// tryCallInventoryQueryDetail tries to call a detail method on InventoryQuery without compile-time dependency.
// It returns (dto, ok(method found), err).
func tryCallInventoryQueryDetail(ctx context.Context, q any, id string) (any, bool, error) {
	// 候補名（プロジェクトの変遷を吸収）
	candidates := []string{
		"GetDetail",
		"GetDetailByID",
		"GetDetailByInventoryID",
		"GetByIDDetail",
		"Detail",
		"GetInventoryDetail",
	}

	rv := reflect.ValueOf(q)

	for _, name := range candidates {
		m := rv.MethodByName(name)
		if !m.IsValid() {
			continue
		}

		// シグネチャが違っても panic しうるので保護
		var (
			out any
			err error
		)
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					log.Printf("[inventory_handler][Detail] reflect call panic name=%s rec=%v", name, rec)
					out = nil
					err = errors.New("reflect call panic")
				}
			}()

			mt := m.Type()
			// 期待: func(context.Context, string) (any, error)
			if mt.NumIn() != 2 || mt.NumOut() != 2 {
				// 合わないなら次へ
				out = nil
				err = nil
				return
			}

			outs := m.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(id)})
			if len(outs) != 2 {
				out = nil
				err = nil
				return
			}

			// 1st return: dto
			out = outs[0].Interface()

			// 2nd return: error
			if !outs[1].IsValid() || outs[1].IsNil() {
				err = nil
				return
			}
			if e, ok := outs[1].Interface().(error); ok {
				err = e
				return
			}
			err = errors.New("second return is not error")
		}()

		// ここまで来たら「候補名のメソッドは存在した」
		// err が reflect panic 由来なら次候補へ進めても良いが、ここでは ok 扱いで返す
		return out, true, err
	}

	return nil, false, nil
}

// ============================================================
// CRUD Handlers (Usecase)
// ============================================================

func (h *InventoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	if h.UC == nil {
		log.Printf("[inventory_handler][Create] UsecaseNotConfigured")
		writeError(w, http.StatusNotImplemented, "inventory usecase is not configured")
		return
	}

	ctx := r.Context()

	var req createInventoryMintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[inventory_handler][Create] BAD_REQUEST invalid json err=%v", err)
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	log.Printf("[inventory_handler][Create] start tbId=%q pbId=%q modelId=%q productIds=%d",
		strings.TrimSpace(req.TokenBlueprintID),
		strings.TrimSpace(req.ProductBlueprintID),
		strings.TrimSpace(req.ModelID),
		len(req.ProductIDs),
	)

	m, err := h.UC.UpsertFromMintByModel(
		ctx,
		req.TokenBlueprintID,
		req.ProductBlueprintID,
		req.ModelID,
		req.ProductIDs,
	)
	if err != nil {
		log.Printf("[inventory_handler][Create] failed err=%v", err)
		writeDomainError(w, err)
		return
	}

	log.Printf("[inventory_handler][Create] ok id=%q pbId=%q tbId=%q models=%d totalProducts=%d",
		strings.TrimSpace(m.ID),
		strings.TrimSpace(m.ProductBlueprintID),
		strings.TrimSpace(m.TokenBlueprintID),
		len(m.Stock),
		totalProducts(m),
	)

	writeJSON(w, http.StatusCreated, m)
}

func (h *InventoryHandler) List(w http.ResponseWriter, r *http.Request) {
	if h.UC == nil {
		log.Printf("[inventory_handler][List] UsecaseNotConfigured")
		writeError(w, http.StatusNotImplemented, "inventory usecase is not configured")
		return
	}

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
	case tbID != "" && pbID != "":
		all, e := h.UC.ListByProductBlueprintID(ctx, pbID)
		if e != nil {
			err = e
			break
		}
		tmp := make([]invdom.Mint, 0, len(all))
		for _, m := range all {
			if strings.TrimSpace(m.TokenBlueprintID) == tbID {
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
	if h.UC == nil {
		log.Printf("[inventory_handler][GetByID] UsecaseNotConfigured")
		writeError(w, http.StatusNotImplemented, "inventory usecase is not configured")
		return
	}

	ctx := r.Context()

	id := strings.TrimSpace(pathParamLast(r.URL.Path))
	log.Printf("[inventory_handler][GetByID] start id=%q", id)

	if id == "" {
		log.Printf("[inventory_handler][GetByID] BAD_REQUEST missing id")
		writeError(w, http.StatusBadRequest, "missing id")
		return
	}

	m, err := h.UC.GetByID(ctx, id)
	if err != nil {
		log.Printf("[inventory_handler][GetByID] failed id=%q err=%v", id, err)
		writeDomainError(w, err)
		return
	}

	log.Printf("[inventory_handler][GetByID] ok id=%q pbId=%q tbId=%q models=%d totalProducts=%d",
		strings.TrimSpace(m.ID),
		strings.TrimSpace(m.ProductBlueprintID),
		strings.TrimSpace(m.TokenBlueprintID),
		len(m.Stock),
		totalProducts(m),
	)

	writeJSON(w, http.StatusOK, m)
}

func (h *InventoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	if h.UC == nil {
		log.Printf("[inventory_handler][Update] UsecaseNotConfigured")
		writeError(w, http.StatusNotImplemented, "inventory usecase is not configured")
		return
	}

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

	req.ModelID = strings.TrimSpace(req.ModelID)

	log.Printf("[inventory_handler][Update] payload id=%q modelId=%q productIds=%d",
		id, req.ModelID, len(req.ProductIDs),
	)

	if req.ModelID == "" {
		log.Printf("[inventory_handler][Update] BAD_REQUEST missing modelId")
		writeError(w, http.StatusBadRequest, "modelId is required")
		return
	}
	if len(req.ProductIDs) == 0 {
		log.Printf("[inventory_handler][Update] BAD_REQUEST missing productIds")
		writeError(w, http.StatusBadRequest, "productIds is required")
		return
	}

	current, err := h.UC.GetByID(ctx, id)
	if err != nil {
		log.Printf("[inventory_handler][Update] failed to load current id=%q err=%v", id, err)
		writeDomainError(w, err)
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
		log.Printf("[inventory_handler][Update] failed id=%q err=%v", id, err)
		writeDomainError(w, err)
		return
	}

	log.Printf("[inventory_handler][Update] ok id=%q modelId=%q models=%d totalProducts=%d",
		strings.TrimSpace(updated.ID),
		req.ModelID,
		len(updated.Stock),
		totalProducts(updated),
	)

	writeJSON(w, http.StatusOK, updated)
}

func (h *InventoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if h.UC == nil {
		log.Printf("[inventory_handler][Delete] UsecaseNotConfigured")
		writeError(w, http.StatusNotImplemented, "inventory usecase is not configured")
		return
	}

	ctx := r.Context()

	id := strings.TrimSpace(pathParamLast(r.URL.Path))
	log.Printf("[inventory_handler][Delete] start id=%q", id)

	if id == "" {
		log.Printf("[inventory_handler][Delete] BAD_REQUEST missing id")
		writeError(w, http.StatusBadRequest, "missing id")
		return
	}

	if err := h.UC.Delete(ctx, id); err != nil {
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

func totalProducts(m invdom.Mint) int {
	if m.Stock == nil {
		return 0
	}
	total := 0
	for _, ms := range m.Stock {
		total += modelStockLen(ms)
	}
	return total
}

func modelStockLen(ms invdom.ModelStock) int {
	rv := reflect.ValueOf(ms)
	if !rv.IsValid() {
		return 0
	}

	// ✅ map[string]bool / map[string]T
	if rv.Kind() == reflect.Map {
		return rv.Len()
	}

	// ✅ []string / []T / [N]T
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		return rv.Len()
	}

	// ✅ struct 内に map[string]bool を持つケース
	if rv.Kind() == reflect.Struct {
		for i := 0; i < rv.NumField(); i++ {
			f := rv.Field(i)

			if f.Kind() == reflect.Map {
				if f.Type().Key().Kind() == reflect.String {
					return f.Len()
				}
			}
			if f.Kind() == reflect.Slice || f.Kind() == reflect.Array {
				return f.Len()
			}
		}
	}

	return 0
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	log.Printf("[inventory_handler] RESP_ERROR status=%d msg=%q", status, msg)
	writeJSON(w, status, map[string]any{"error": msg})
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
		errors.Is(err, invdom.ErrInvalidProducts):
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
