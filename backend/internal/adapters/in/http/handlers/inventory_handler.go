// backend/internal/adapters/in/http/handlers/inventory_handler.go
package handlers

import (
	"context"
	"encoding/json"
	"errors"
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

	// ✅ NEW: GET /inventory/list-create/{pbId}/{tbId}
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

	if path == "/inventory/ids" {
		switch r.Method {
		case http.MethodGet:
			h.ResolveInventoryIDsByProductAndTokenQuery(w, r)
			return
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	}

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
	if strings.HasPrefix(path, "/inventory/") {
		switch r.Method {
		case http.MethodGet:
			id := strings.TrimSpace(strings.TrimPrefix(path, "/inventory/"))
			if id == "" || id == "ids" {
				writeError(w, http.StatusBadRequest, "invalid inventory id")
				return
			}
			h.GetDetailByIDQueryOrFallback(w, r, id)
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
// Query Handler (Read-only DTO)
// ============================================================

func (h *InventoryHandler) ListByCurrentCompanyQuery(w http.ResponseWriter, r *http.Request) {
	if h.Q == nil {
		writeError(w, http.StatusNotImplemented, "inventory query is not configured")
		return
	}

	ctx := r.Context()

	rows, err := h.Q.ListByCurrentCompany(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_ = querydto.InventoryManagementRowDTO{}
	writeJSON(w, http.StatusOK, rows)
}

func (h *InventoryHandler) ResolveInventoryIDsByProductAndTokenQuery(w http.ResponseWriter, r *http.Request) {
	if h.Q == nil {
		writeError(w, http.StatusNotImplemented, "inventory query is not configured")
		return
	}

	ctx := r.Context()

	pbID := strings.TrimSpace(r.URL.Query().Get("productBlueprintId"))
	tbID := strings.TrimSpace(r.URL.Query().Get("tokenBlueprintId"))
	if pbID == "" || tbID == "" {
		writeError(w, http.StatusBadRequest, "productBlueprintId and tokenBlueprintId are required")
		return
	}

	ids, err := h.Q.ListInventoryIDsByProductAndToken(ctx, pbID, tbID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := querydto.InventoryIDsByProductAndTokenDTO{
		ProductBlueprintID: pbID,
		TokenBlueprintID:   tbID,
		InventoryIDs:       ids,
	}

	writeJSON(w, http.StatusOK, resp)
}

// ============================================================
// ✅ NEW: ListCreate DTO endpoint
// - GET /inventory/list-create/{pbId}/{tbId}
// - GET /inventory/list-create/{inventoryId}  (inventoryId="{pbId}__{tbId}")
// ============================================================

func (h *InventoryHandler) GetListCreateByPathQuery(w http.ResponseWriter, r *http.Request, path string) {
	if h.LQ == nil {
		writeError(w, http.StatusNotImplemented, "list create query is not configured")
		return
	}

	ctx := r.Context()

	rest := strings.TrimSpace(strings.TrimPrefix(path, "/inventory/list-create/"))
	rest = strings.Trim(rest, "/")
	if rest == "" {
		writeError(w, http.StatusBadRequest, "missing params")
		return
	}

	seg := strings.Split(rest, "/")

	var pbID, tbID string

	switch len(seg) {
	case 1:
		// inventoryId = "{pbId}__{tbId}"
		invID := strings.TrimSpace(seg[0])
		parts := strings.Split(invID, "__")
		if len(parts) < 2 {
			writeError(w, http.StatusBadRequest, "invalid inventoryId format (expected {pbId}__{tbId})")
			return
		}
		pbID = strings.TrimSpace(parts[0])
		tbID = strings.TrimSpace(parts[1])

	case 2:
		pbID = strings.TrimSpace(seg[0])
		tbID = strings.TrimSpace(seg[1])

	default:
		writeError(w, http.StatusBadRequest, "invalid path params")
		return
	}

	if pbID == "" || tbID == "" {
		writeError(w, http.StatusBadRequest, "productBlueprintId and tokenBlueprintId are required")
		return
	}

	dto, err := h.LQ.GetByIDs(ctx, pbID, tbID)
	if err != nil {
		// validation系は 400、それ以外は 500 に寄せる
		if isProbablyBadRequest(err) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dto)
}

func isProbablyBadRequest(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(msg, "required") ||
		strings.Contains(msg, "missing") ||
		strings.Contains(msg, "invalid")
}

// ============================================================
// ✅ Detail endpoint（確定）
// - Query があれば必ず GetDetailByID を呼ぶ
// - Query が無い場合のみ UC fallback
// ============================================================

func (h *InventoryHandler) GetDetailByIDQueryOrFallback(w http.ResponseWriter, r *http.Request, inventoryID string) {
	ctx := r.Context()
	id := strings.TrimSpace(inventoryID)

	if id == "" {
		writeError(w, http.StatusBadRequest, "missing id")
		return
	}

	// 1) Query があるなら確定で呼ぶ
	if h.Q != nil {
		dto, err := h.Q.GetDetailByID(ctx, id)
		if err != nil {
			if errors.Is(err, invdom.ErrNotFound) {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// ✅ 返却 DTO に "tokenBlueprintPatch" を合成して返す
		// - dto 型を変えずに拡張するため、JSON round-trip で map にする
		respMap, mErr := anyToMap(dto)
		if mErr == nil {
			tbID := firstNonEmptyString(
				asString(respMap["tokenBlueprintId"]),
				asString(respMap["tokenBlueprintID"]),
				asString(respMap["tokenBlueprintId"]),
			)

			if tbID != "" {
				if patch, _ := h.FetchTokenBlueprintPatch(ctx, tbID); patch != nil {
					// dto 側に既に tokenBlueprintPatch があっても上書きする（常に最新を優先）
					respMap["tokenBlueprintPatch"] = patch
				}
			}

			writeJSON(w, http.StatusOK, respMap)
			return
		}

		// map 化に失敗したら通常返却
		writeJSON(w, http.StatusOK, dto)
		return
	}

	// 2) fallback: UC.GetByID
	if h.UC == nil {
		writeError(w, http.StatusNotImplemented, "inventory usecase is not configured")
		return
	}

	m, err := h.UC.GetByID(ctx, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	resp := map[string]any{
		"inventoryId":        strings.TrimSpace(m.ID),
		"id":                 strings.TrimSpace(m.ID),
		"inventoryIds":       []string{strings.TrimSpace(m.ID)},
		"tokenBlueprintId":   strings.TrimSpace(m.TokenBlueprintID),
		"productBlueprintId": strings.TrimSpace(m.ProductBlueprintID),
		"modelId":            "",

		"productBlueprintPatch": map[string]any{},
		"tokenBlueprintPatch":   map[string]any{}, // ✅ 追加（fallback は空で返す）
		"rows":                  []any{},
		"totalStock":            totalProducts(m),
	}

	writeJSON(w, http.StatusOK, resp)
}

// ============================================================
// ✅ TokenBlueprint Patch（呼び出し関数）
// - Query 側に実装があれば呼ぶ（存在しない場合は nil を返す）
// ============================================================

func (h *InventoryHandler) FetchTokenBlueprintPatch(ctx context.Context, tokenBlueprintID string) (any, error) {
	if h == nil || h.Q == nil {
		return nil, nil
	}
	id := strings.TrimSpace(tokenBlueprintID)
	if id == "" {
		return nil, nil
	}

	// 期待する Query 側メソッド名（将来の命名揺れを許容）
	candidates := []string{
		"GetTokenBlueprintPatchByID",
		"GetTokenBlueprintPatchById",
		"FetchTokenBlueprintPatchByID",
		"FetchTokenBlueprintPatchById",
		"GetTokenBlueprintPatch",
		"FetchTokenBlueprintPatch",
	}

	qv := reflect.ValueOf(h.Q)
	for _, name := range candidates {
		m := qv.MethodByName(name)
		if !m.IsValid() {
			continue
		}

		// func(ctx context.Context, id string) (T, error)
		mt := m.Type()
		if mt.NumIn() != 2 || mt.In(0) != reflect.TypeOf((*context.Context)(nil)).Elem() || mt.In(1).Kind() != reflect.String {
			continue
		}
		if mt.NumOut() != 2 || !mt.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			continue
		}

		outs := m.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(id)})

		var outErr error
		if !outs[1].IsNil() {
			outErr = outs[1].Interface().(error)
		}
		if outErr != nil {
			return nil, outErr
		}

		patch := outs[0].Interface()
		return patch, nil
	}

	// Query 側に未実装（またはシグネチャ不一致）
	return nil, nil
}

// ============================================================
// CRUD Handlers (Usecase)
// ============================================================

func (h *InventoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	if h.UC == nil {
		writeError(w, http.StatusNotImplemented, "inventory usecase is not configured")
		return
	}

	ctx := r.Context()

	var req createInventoryMintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
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
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, m)
}

func (h *InventoryHandler) List(w http.ResponseWriter, r *http.Request) {
	if h.UC == nil {
		writeError(w, http.StatusNotImplemented, "inventory usecase is not configured")
		return
	}

	ctx := r.Context()

	q := r.URL.Query()
	tbID := strings.TrimSpace(q.Get("tokenBlueprintId"))
	pbID := strings.TrimSpace(q.Get("productBlueprintId"))
	modelID := strings.TrimSpace(q.Get("modelId"))

	if tbID == "" && pbID == "" && modelID == "" {
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
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, list)
}

func (h *InventoryHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	if h.UC == nil {
		writeError(w, http.StatusNotImplemented, "inventory usecase is not configured")
		return
	}

	ctx := r.Context()

	id := strings.TrimSpace(pathParamLast(r.URL.Path))
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing id")
		return
	}

	m, err := h.UC.GetByID(ctx, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, m)
}

func (h *InventoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	if h.UC == nil {
		writeError(w, http.StatusNotImplemented, "inventory usecase is not configured")
		return
	}

	ctx := r.Context()

	id := strings.TrimSpace(pathParamLast(r.URL.Path))
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing id")
		return
	}

	var req updateInventoryMintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	req.ModelID = strings.TrimSpace(req.ModelID)
	if req.ModelID == "" {
		writeError(w, http.StatusBadRequest, "modelId is required")
		return
	}
	if len(req.ProductIDs) == 0 {
		writeError(w, http.StatusBadRequest, "productIds is required")
		return
	}

	current, err := h.UC.GetByID(ctx, id)
	if err != nil {
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
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *InventoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if h.UC == nil {
		writeError(w, http.StatusNotImplemented, "inventory usecase is not configured")
		return
	}

	ctx := r.Context()

	id := strings.TrimSpace(pathParamLast(r.URL.Path))
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing id")
		return
	}

	if err := h.UC.Delete(ctx, id); err != nil {
		writeDomainError(w, err)
		return
	}

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

	if rv.Kind() == reflect.Map {
		return rv.Len()
	}
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		return rv.Len()
	}
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
	writeJSON(w, status, map[string]any{"error": msg})
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, invdom.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())

	case errors.Is(err, invdom.ErrInvalidMintID),
		errors.Is(err, invdom.ErrInvalidTokenBlueprintID),
		errors.Is(err, invdom.ErrInvalidProductBlueprintID),
		errors.Is(err, invdom.ErrInvalidModelID),
		errors.Is(err, invdom.ErrInvalidProducts):
		writeError(w, http.StatusBadRequest, err.Error())

	default:
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

func anyToMap(v any) (map[string]any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func asString(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func firstNonEmptyString(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}
