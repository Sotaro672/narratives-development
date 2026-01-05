// backend\internal\adapters\in\http\mall\handler\cart_handler.go
package handler

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"sort"
	"strings"

	usecase "narratives/internal/application/usecase"
	cartdom "narratives/internal/domain/cart"
)

// CartQueryService abstracts cart_query read-model.
// Return type is intentionally `any` to avoid tight coupling to query DTO package.
// (UI側は raw を扱える設計なので JSON をそのまま返すのが最も安全)
type CartQueryService interface {
	// GetCartQuery should return a JSON-serializable structure (map / struct / slice).
	GetCartQuery(ctx context.Context, avatarID string) (any, error)
}

// PreviewQueryService abstracts preview_query read-model.
type PreviewQueryService interface {
	// GetPreview should return a JSON-serializable structure (map / struct / slice).
	GetPreview(ctx context.Context, avatarID string, itemKey string) (any, error)
}

// CartHandler serves SNS cart endpoints.
// Intended mount examples (router side):
// - GET    /sns/cart            ✅ unified: read-model (CartDTO) を返す
// - DELETE /sns/cart            (clear)
// - POST   /sns/cart/items
// - PUT    /sns/cart/items
// - DELETE /sns/cart/items
//
// NOTE:
// - /sns/cart/query は廃止（この handler では扱わない）
type CartHandler struct {
	uc *usecase.CartUsecase

	// ✅ read-model queries (optional but recommended)
	cartQuery    CartQueryService
	previewQuery PreviewQueryService
}

func NewCartHandler(uc *usecase.CartUsecase) http.Handler {
	return &CartHandler{uc: uc, cartQuery: nil, previewQuery: nil}
}

// ✅ query を注入できる ctor
//
// NOTE:
// DI 側では *snsquery.SNSCartQuery / *snsquery.SNSPreviewQuery をそのまま渡したいが、
// それらがこの package の interface を直接実装していないケースがある。
// そこで引数は `any` とし、ここで “best-effort adapter” を噛ませる。
func NewCartHandlerWithQueries(
	uc *usecase.CartUsecase,
	cartQuery any,
	previewQuery any,
) http.Handler {
	return &CartHandler{
		uc:           uc,
		cartQuery:    wrapCartQuery(cartQuery),
		previewQuery: wrapPreviewQuery(previewQuery),
	}
}

func (h *CartHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.uc == nil {
		writeErr(w, http.StatusInternalServerError, "cart handler is not configured")
		return
	}

	// IMPORTANT:
	// router 側で StripPrefix("/sns") や StripPrefix("/sns/cart") をしていると、
	// ここに入ってくる Path は "/sns/cart" ではなく "/cart" や "/" になる。
	// その揺れを吸収する。
	path := strings.TrimRight(r.URL.Path, "/")
	if path == "" {
		path = "/"
	}

	isGET := r.Method == http.MethodGet
	isDEL := r.Method == http.MethodDelete
	isPOST := r.Method == http.MethodPost
	isPUT := r.Method == http.MethodPut

	// suffix matcher (複数候補のどれかに一致したら true)
	hasSuffixAny := func(p string, suffixes ...string) bool {
		for _, s := range suffixes {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			if strings.HasSuffix(p, s) {
				return true
			}
		}
		return false
	}

	// exact matcher (StripPrefix("/sns/cart") の場合、/sns/cart は "/" になる)
	isAnyExact := func(p string, exacts ...string) bool {
		for _, e := range exacts {
			if p == e {
				return true
			}
		}
		return false
	}

	switch {
	// ============================================================
	// ✅ Unified: GET /sns/cart returns read-model DTO
	// ============================================================

	// ====== GET /sns/cart (or /cart or "/")
	case isGET && (hasSuffixAny(path, "/sns/cart", "/cart") || isAnyExact(path, "/")):
		h.handleGetUnified(w, r)
		return

	// ====== DELETE /sns/cart (or /cart or "/")
	case isDEL && (hasSuffixAny(path, "/sns/cart", "/cart") || isAnyExact(path, "/")):
		h.handleClear(w, r)
		return

	// ====== POST /sns/cart/items (or /cart/items or /items)
	case isPOST && (hasSuffixAny(path, "/sns/cart/items", "/cart/items") || isAnyExact(path, "/items")):
		h.handleAddItem(w, r)
		return

	// ====== PUT /sns/cart/items (or /cart/items or /items)
	case isPUT && (hasSuffixAny(path, "/sns/cart/items", "/cart/items") || isAnyExact(path, "/items")):
		h.handleSetItemQty(w, r)
		return

	// ====== DELETE /sns/cart/items (or /cart/items or /items)
	case isDEL && (hasSuffixAny(path, "/sns/cart/items", "/cart/items") || isAnyExact(path, "/items")):
		h.handleRemoveItem(w, r)
		return

	// ====== (optional) preview
	case isGET && hasSuffixAny(path, "/sns/preview", "/preview"):
		h.handleGetPreview(w, r)
		return
	}

	writeErr(w, http.StatusNotFound, "not found")
}

// -------------------------
// handlers (Unified GET)
// -------------------------

// handleGetUnified returns CartDTO (read-model) on GET /sns/cart.
// - If cartQuery is configured: prefer read-model.
// - If cart doc is missing: return empty cart (200) for stable UX.
// - If cartQuery is not configured: fallback to legacy response.
func (h *CartHandler) handleGetUnified(w http.ResponseWriter, r *http.Request) {
	avatarID := readAvatarID(r, "")
	if avatarID == "" {
		writeErr(w, http.StatusBadRequest, "avatarId is required")
		return
	}

	// ✅ Prefer read-model when available
	if h.cartQuery != nil {
		v, err := h.cartQuery.GetCartQuery(r.Context(), avatarID)
		if err == nil {
			writeJSON(w, http.StatusOK, v)
			return
		}

		// cart が無いなら “空カート” を 200 で返す（/sns/cart を安定させる）
		if isNotFoundErr(err) {
			writeJSON(w, http.StatusOK, map[string]any{
				"avatarId":  avatarID,
				"items":     map[string]any{},
				"createdAt": nil,
				"updatedAt": nil,
				"expiresAt": nil,
			})
			return
		}

		// invalid argument 系 / その他は通常通り
		h.writeQueryErr(w, err)
		return
	}

	// ✅ Fallback: legacy (DI misconfig safety)
	h.handleGet(w, r)
}

// -------------------------
// handlers (Preview)
// -------------------------

func (h *CartHandler) handleGetPreview(w http.ResponseWriter, r *http.Request) {
	avatarID := readAvatarID(r, "")
	if avatarID == "" {
		writeErr(w, http.StatusBadRequest, "avatarId is required")
		return
	}

	itemKey := readItemKey(r, "")
	if itemKey == "" {
		writeErr(w, http.StatusBadRequest, "itemKey is required")
		return
	}

	if h.previewQuery == nil {
		writeErr(w, http.StatusInternalServerError, "preview_query is not configured")
		return
	}

	v, err := h.previewQuery.GetPreview(r.Context(), avatarID, itemKey)
	if err != nil {
		h.writeQueryErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, v)
}

// writeQueryErr maps typical query errors to HTTP codes.
// - invalid arg => 400
// - otherwise => 500
//
// NOTE: not found は unified GET では 200(empty) にしているため、ここでは 404 に寄せない。
func (h *CartHandler) writeQueryErr(w http.ResponseWriter, err error) {
	if err == nil {
		writeErr(w, http.StatusInternalServerError, "unknown error")
		return
	}

	// invalid argument 系
	if errors.Is(err, usecase.ErrCartInvalidArgument) || errors.Is(err, cartdom.ErrInvalidCart) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	// fallback
	writeErr(w, http.StatusInternalServerError, err.Error())
}

func isNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(s, "not found") ||
		strings.Contains(s, "notfound") ||
		strings.Contains(s, "no such") ||
		strings.Contains(s, "missing")
}

// -------------------------
// handlers (Legacy / mutations)
// -------------------------

func (h *CartHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	avatarID := readAvatarID(r, "")
	if avatarID == "" {
		writeErr(w, http.StatusBadRequest, "avatarId is required")
		return
	}

	c, err := h.uc.GetOrCreate(r.Context(), avatarID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toCartResponse(avatarID, c))
}

func (h *CartHandler) handleAddItem(w http.ResponseWriter, r *http.Request) {
	var req cartItemReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}

	avatarID := readAvatarID(r, req.AvatarID)

	invID := strings.TrimSpace(req.InventoryID)
	listID := strings.TrimSpace(req.ListID)
	modelID := strings.TrimSpace(req.ModelID)

	if avatarID == "" || invID == "" || listID == "" || modelID == "" || req.Qty <= 0 {
		writeErr(w, http.StatusBadRequest, "avatarId, inventoryId, listId, modelId, qty(>=1) are required")
		return
	}

	c, err := h.uc.AddItem(r.Context(), avatarID, invID, listID, modelID, req.Qty)
	if err != nil {
		if errors.Is(err, usecase.ErrCartInvalidArgument) || errors.Is(err, cartdom.ErrInvalidCart) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toCartResponse(avatarID, c))
}

func (h *CartHandler) handleSetItemQty(w http.ResponseWriter, r *http.Request) {
	var req cartItemReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}

	avatarID := readAvatarID(r, req.AvatarID)

	invID := strings.TrimSpace(req.InventoryID)
	listID := strings.TrimSpace(req.ListID)
	modelID := strings.TrimSpace(req.ModelID)

	if avatarID == "" || invID == "" || listID == "" || modelID == "" {
		writeErr(w, http.StatusBadRequest, "avatarId, inventoryId, listId, modelId are required")
		return
	}

	c, err := h.uc.SetItemQty(r.Context(), avatarID, invID, listID, modelID, req.Qty)
	if err != nil {
		if errors.Is(err, usecase.ErrCartInvalidArgument) || errors.Is(err, cartdom.ErrInvalidCart) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toCartResponse(avatarID, c))
}

func (h *CartHandler) handleRemoveItem(w http.ResponseWriter, r *http.Request) {
	var req cartItemReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}

	avatarID := readAvatarID(r, req.AvatarID)

	invID := strings.TrimSpace(req.InventoryID)
	listID := strings.TrimSpace(req.ListID)
	modelID := strings.TrimSpace(req.ModelID)

	if avatarID == "" || invID == "" || listID == "" || modelID == "" {
		writeErr(w, http.StatusBadRequest, "avatarId, inventoryId, listId, modelId are required")
		return
	}

	c, err := h.uc.RemoveItem(r.Context(), avatarID, invID, listID, modelID)
	if err != nil {
		if errors.Is(err, usecase.ErrCartInvalidArgument) || errors.Is(err, cartdom.ErrInvalidCart) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toCartResponse(avatarID, c))
}

func (h *CartHandler) handleClear(w http.ResponseWriter, r *http.Request) {
	avatarID := readAvatarID(r, "")
	if avatarID == "" {
		writeErr(w, http.StatusBadRequest, "avatarId is required")
		return
	}

	if err := h.uc.Clear(r.Context(), avatarID); err != nil {
		if errors.Is(err, usecase.ErrCartInvalidArgument) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// -------------------------
// request/response DTO (legacy)
// -------------------------

type cartItemReq struct {
	AvatarID     string `json:"avatarId"`
	InventoryID  string `json:"inventoryId"`
	ListID       string `json:"listId"`
	ModelID      string `json:"modelId"`
	Qty          int    `json:"qty"`
	ItemKey      string `json:"itemKey"` // optional (future use)
	LegacyModel  string `json:"-"`       // unused; keep for forward compatibility if needed
	LegacyListID string `json:"-"`       // unused
}

type cartResponse struct {
	AvatarID string                 `json:"avatarId"`
	Items    map[string]cartItemDTO `json:"items"`

	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	ExpiresAt string `json:"expiresAt"`
}

type cartItemDTO struct {
	InventoryID string `json:"inventoryId"`
	ListID      string `json:"listId"`
	ModelID     string `json:"modelId"`
	Qty         int    `json:"qty"`
}

func toCartResponse(avatarID string, c *cartdom.Cart) cartResponse {
	items := map[string]cartItemDTO{}
	if c != nil && c.Items != nil {
		keys := make([]string, 0, len(c.Items))
		for k := range c.Items {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			it := c.Items[k]
			items[k] = cartItemDTO{
				InventoryID: strings.TrimSpace(it.InventoryID),
				ListID:      strings.TrimSpace(it.ListID),
				ModelID:     strings.TrimSpace(it.ModelID),
				Qty:         it.Qty,
			}
		}
	}

	if c == nil {
		return cartResponse{
			AvatarID:  strings.TrimSpace(avatarID),
			Items:     items,
			CreatedAt: "",
			UpdatedAt: "",
			ExpiresAt: "",
		}
	}

	return cartResponse{
		AvatarID:  strings.TrimSpace(avatarID),
		Items:     items,
		CreatedAt: toRFC3339(c.CreatedAt),
		UpdatedAt: toRFC3339(c.UpdatedAt),
		ExpiresAt: toRFC3339(c.ExpiresAt),
	}
}

// -------------------------
// helpers
// -------------------------

func readAvatarID(r *http.Request, fallback string) string {
	if v := strings.TrimSpace(r.URL.Query().Get("avatarId")); v != "" {
		return v
	}
	if v := strings.TrimSpace(r.Header.Get("X-Avatar-Id")); v != "" {
		return v
	}
	return strings.TrimSpace(fallback)
}

func readItemKey(r *http.Request, fallback string) string {
	if v := strings.TrimSpace(r.URL.Query().Get("itemKey")); v != "" {
		return v
	}
	if v := strings.TrimSpace(r.Header.Get("X-Item-Key")); v != "" {
		return v
	}
	return strings.TrimSpace(fallback)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{
		"error": msg,
	})
}

// -------------------------
// best-effort adapters (DI 互換)
// -------------------------

type dynamicCartQuery struct{ impl any }
type dynamicPreviewQuery struct{ impl any }

func wrapCartQuery(v any) CartQueryService {
	if v == nil {
		return nil
	}
	if s, ok := v.(CartQueryService); ok {
		return s
	}
	return &dynamicCartQuery{impl: v}
}

func wrapPreviewQuery(v any) PreviewQueryService {
	if v == nil {
		return nil
	}
	if s, ok := v.(PreviewQueryService); ok {
		return s
	}
	return &dynamicPreviewQuery{impl: v}
}

func (d *dynamicCartQuery) GetCartQuery(ctx context.Context, avatarID string) (any, error) {
	return callQuery2(d.impl, ctx, avatarID,
		"GetCartQuery",
		"GetByAvatarID",
		"GetCart",
		"Get",
		"Query",
		"Fetch",
	)
}

func (d *dynamicPreviewQuery) GetPreview(ctx context.Context, avatarID string, itemKey string) (any, error) {
	if strings.TrimSpace(itemKey) != "" {
		if v, err := callQuery3(d.impl, ctx, avatarID, itemKey,
			"GetPreview",
			"GetByAvatarIDAndItemKey",
			"GetByAvatarAndItemKey",
			"Preview",
			"Get",
			"Query",
			"Fetch",
		); err == nil {
			return v, nil
		}
	}
	return callQuery2(d.impl, ctx, avatarID,
		"GetPreview",
		"Preview",
		"Get",
		"Query",
		"Fetch",
	)
}

func callQuery2(impl any, ctx context.Context, avatarID string, methodNames ...string) (any, error) {
	if impl == nil {
		return nil, errors.New("query service is nil")
	}

	rv := reflect.ValueOf(impl)
	if !rv.IsValid() {
		return nil, errors.New("query service is invalid")
	}

	for _, name := range methodNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		mv := rv.MethodByName(name)
		if !mv.IsValid() {
			continue
		}

		mt := mv.Type()
		if mt.NumIn() != 2 || mt.NumOut() != 2 {
			continue
		}

		ctxType := reflect.TypeOf((*context.Context)(nil)).Elem()
		if !mt.In(0).Implements(ctxType) && !ctxType.AssignableTo(mt.In(0)) && !reflect.TypeOf(ctx).AssignableTo(mt.In(0)) {
			continue
		}
		if mt.In(1).Kind() != reflect.String {
			continue
		}

		errType := reflect.TypeOf((*error)(nil)).Elem()
		if !mt.Out(1).Implements(errType) {
			continue
		}

		out := mv.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(avatarID)})
		if len(out) != 2 {
			continue
		}

		var e error
		if !out[1].IsNil() {
			if ee, ok := out[1].Interface().(error); ok {
				e = ee
			} else {
				e = errors.New("unknown error")
			}
		}

		return out[0].Interface(), e
	}

	return nil, errors.New("query service method not found")
}

func callQuery3(impl any, ctx context.Context, avatarID string, itemKey string, methodNames ...string) (any, error) {
	if impl == nil {
		return nil, errors.New("query service is nil")
	}

	rv := reflect.ValueOf(impl)
	if !rv.IsValid() {
		return nil, errors.New("query service is invalid")
	}

	for _, name := range methodNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		mv := rv.MethodByName(name)
		if !mv.IsValid() {
			continue
		}

		mt := mv.Type()
		if mt.NumIn() != 3 || mt.NumOut() != 2 {
			continue
		}

		ctxType := reflect.TypeOf((*context.Context)(nil)).Elem()
		if !mt.In(0).Implements(ctxType) && !ctxType.AssignableTo(mt.In(0)) && !reflect.TypeOf(ctx).AssignableTo(mt.In(0)) {
			continue
		}
		if mt.In(1).Kind() != reflect.String || mt.In(2).Kind() != reflect.String {
			continue
		}

		errType := reflect.TypeOf((*error)(nil)).Elem()
		if !mt.Out(1).Implements(errType) {
			continue
		}

		out := mv.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(avatarID), reflect.ValueOf(itemKey)})
		if len(out) != 2 {
			continue
		}

		var e error
		if !out[1].IsNil() {
			if ee, ok := out[1].Interface().(error); ok {
				e = ee
			} else {
				e = errors.New("unknown error")
			}
		}

		return out[0].Interface(), e
	}

	return nil, errors.New("query service method not found")
}
