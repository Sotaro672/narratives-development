// backend\internal\adapters\in\http\mall\handler\cart_handler.go
package mallHandler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	cartdom "narratives/internal/domain/cart"
)

// CartQueryService abstracts cart_query read-model.
type CartQueryService interface {
	GetCartQuery(ctx context.Context, avatarID string) (any, error)
}

// CartHandler serves Mall cart endpoints.
type CartHandler struct {
	uc *usecase.CartUsecase

	// read-model queries (required for unified GET)
	cartQuery CartQueryService
}

func NewCartHandler(uc *usecase.CartUsecase) http.Handler {
	return &CartHandler{uc: uc, cartQuery: nil}
}

func NewCartHandlerWithQueries(
	uc *usecase.CartUsecase,
	cartQuery any,
) http.Handler {
	return &CartHandler{
		uc:        uc,
		cartQuery: wrapCartQuery(cartQuery),
	}
}

func (h *CartHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// ---- request entry log (always) ----
	start := time.Now()
	rawPath := r.URL.Path
	path := strings.TrimRight(rawPath, "/")
	if path == "" {
		path = "/"
	}

	avatarFromQ := strings.TrimSpace(r.URL.Query().Get("avatarId"))
	avatarFromH := strings.TrimSpace(r.Header.Get("X-Avatar-Id"))
	avatarID := readAvatarID(r, "")

	log.Printf(
		"[mall_cart_handler] enter method=%s rawPath=%q path=%q query=%q avatarId(q)=%q avatarId(h)=%q avatarId(resolved)=%q configured(uc=%t cartQuery=%t)\n",
		r.Method,
		rawPath,
		path,
		r.URL.RawQuery,
		avatarFromQ,
		avatarFromH,
		avatarID,
		h.uc != nil,
		h.cartQuery != nil,
	)

	if h.uc == nil {
		log.Printf("[mall_cart_handler] exit status=500 reason=cart handler uc is nil elapsed=%s\n", time.Since(start))
		writeErr(w, http.StatusInternalServerError, "cart handler is not configured")
		return
	}

	isGET := r.Method == http.MethodGet
	isDEL := r.Method == http.MethodDelete
	isPOST := r.Method == http.MethodPost
	isPUT := r.Method == http.MethodPut

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

	isAnyExact := func(p string, exacts ...string) bool {
		for _, e := range exacts {
			if p == e {
				return true
			}
		}
		return false
	}

	switch {
	// Unified GET
	case isGET && (hasSuffixAny(path, "/mall/me/cart", "/cart") || isAnyExact(path, "/")):
		h.handleGetUnified(w, r, start)
		return

	// Clear
	case isDEL && (hasSuffixAny(path, "/mall/me/cart", "/cart") || isAnyExact(path, "/")):
		h.handleClear(w, r, start)
		return

	// Add item
	case isPOST && (hasSuffixAny(path, "/mall/me/cart/items", "/cart/items") || isAnyExact(path, "/items")):
		h.handleAddItem(w, r, start)
		return

	// Set qty
	case isPUT && (hasSuffixAny(path, "/mall/me/cart/items", "/cart/items") || isAnyExact(path, "/items")):
		h.handleSetItemQty(w, r, start)
		return

	// Remove item
	case isDEL && (hasSuffixAny(path, "/mall/me/cart/items", "/cart/items") || isAnyExact(path, "/items")):
		h.handleRemoveItem(w, r, start)
		return
	}

	log.Printf("[mall_cart_handler] exit status=404 reason=not found method=%s path=%q elapsed=%s\n", r.Method, path, time.Since(start))
	writeErr(w, http.StatusNotFound, "not found")
}

// -------------------------
// handlers (Unified GET)
// -------------------------

func (h *CartHandler) handleGetUnified(w http.ResponseWriter, r *http.Request, start time.Time) {
	avatarID := readAvatarID(r, "")
	if avatarID == "" {
		log.Printf("[mall_cart_handler] GET unified exit status=400 reason=avatarId missing rawQuery=%q\n", r.URL.RawQuery)
		writeErr(w, http.StatusBadRequest, "avatarId is required")
		return
	}

	if h.cartQuery == nil {
		log.Printf("[mall_cart_handler] GET unified exit status=500 reason=cartQuery nil avatarId=%q\n", avatarID)
		writeErr(w, http.StatusInternalServerError, "cart_query is not configured")
		return
	}

	log.Printf("[mall_cart_handler] GET unified call cartQuery avatarId=%q queryImpl=%T\n", avatarID, h.cartQuery)

	v, err := h.cartQuery.GetCartQuery(r.Context(), avatarID)
	if err == nil {
		log.Printf("[mall_cart_handler] GET unified ok status=200 avatarId=%q elapsed=%s\n", avatarID, time.Since(start))
		writeJSON(w, http.StatusOK, v)
		return
	}

	nf := isNotFoundErr(err)
	log.Printf("[mall_cart_handler] GET unified cartQuery error avatarId=%q notFound=%t err=%v\n", avatarID, nf, err)

	if nf {
		// empty cart (stable UX)
		log.Printf("[mall_cart_handler] GET unified return empty-cart status=200 avatarId=%q elapsed=%s\n", avatarID, time.Since(start))
		writeJSON(w, http.StatusOK, map[string]any{
			"avatarId":  avatarID,
			"items":     map[string]any{},
			"createdAt": nil,
			"updatedAt": nil,
			"expiresAt": nil,
		})
		return
	}

	log.Printf("[mall_cart_handler] GET unified exit status=500 avatarId=%q elapsed=%s\n", avatarID, time.Since(start))
	h.writeQueryErr(w, err)
}

func (h *CartHandler) writeQueryErr(w http.ResponseWriter, err error) {
	if err == nil {
		writeErr(w, http.StatusInternalServerError, "unknown error")
		return
	}

	if errors.Is(err, usecase.ErrCartInvalidArgument) || errors.Is(err, cartdom.ErrInvalidCart) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

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
// handlers (mutations)
// -------------------------

func (h *CartHandler) handleAddItem(w http.ResponseWriter, r *http.Request, start time.Time) {
	var req cartItemReq
	if err := readJSON(r, &req); err != nil {
		log.Printf("[mall_cart_handler] POST add-item exit status=400 reason=invalid json err=%v\n", err)
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}

	avatarID := readAvatarID(r, req.AvatarID)

	invID := strings.TrimSpace(req.InventoryID)
	listID := strings.TrimSpace(req.ListID)
	modelID := strings.TrimSpace(req.ModelID)

	log.Printf("[mall_cart_handler] POST add-item request avatarId=%q invID=%q listID=%q modelID=%q qty=%d\n", avatarID, invID, listID, modelID, req.Qty)

	if avatarID == "" || invID == "" || listID == "" || modelID == "" || req.Qty <= 0 {
		log.Printf("[mall_cart_handler] POST add-item exit status=400 reason=missing fields avatarId=%q invID=%q listID=%q modelID=%q qty=%d\n", avatarID, invID, listID, modelID, req.Qty)
		writeErr(w, http.StatusBadRequest, "avatarId, inventoryId, listId, modelId, qty(>=1) are required")
		return
	}

	_, err := h.uc.AddItem(r.Context(), avatarID, invID, listID, modelID, req.Qty)
	if err != nil {
		log.Printf("[mall_cart_handler] POST add-item uc error avatarId=%q err=%v\n", avatarID, err)
		if errors.Is(err, usecase.ErrCartInvalidArgument) || errors.Is(err, cartdom.ErrInvalidCart) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Printf("[mall_cart_handler] POST add-item uc ok avatarId=%q -> respond cartDTO\n", avatarID)
	h.respondCartDTO(w, r, avatarID, start)
}

func (h *CartHandler) handleSetItemQty(w http.ResponseWriter, r *http.Request, start time.Time) {
	var req cartItemReq
	if err := readJSON(r, &req); err != nil {
		log.Printf("[mall_cart_handler] PUT set-qty exit status=400 reason=invalid json err=%v\n", err)
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}

	avatarID := readAvatarID(r, req.AvatarID)

	invID := strings.TrimSpace(req.InventoryID)
	listID := strings.TrimSpace(req.ListID)
	modelID := strings.TrimSpace(req.ModelID)

	log.Printf("[mall_cart_handler] PUT set-qty request avatarId=%q invID=%q listID=%q modelID=%q qty=%d\n", avatarID, invID, listID, modelID, req.Qty)

	if avatarID == "" || invID == "" || listID == "" || modelID == "" {
		log.Printf("[mall_cart_handler] PUT set-qty exit status=400 reason=missing fields avatarId=%q invID=%q listID=%q modelID=%q\n", avatarID, invID, listID, modelID)
		writeErr(w, http.StatusBadRequest, "avatarId, inventoryId, listId, modelId are required")
		return
	}

	_, err := h.uc.SetItemQty(r.Context(), avatarID, invID, listID, modelID, req.Qty)
	if err != nil {
		log.Printf("[mall_cart_handler] PUT set-qty uc error avatarId=%q err=%v\n", avatarID, err)
		if errors.Is(err, usecase.ErrCartInvalidArgument) || errors.Is(err, cartdom.ErrInvalidCart) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Printf("[mall_cart_handler] PUT set-qty uc ok avatarId=%q -> respond cartDTO\n", avatarID)
	h.respondCartDTO(w, r, avatarID, start)
}

func (h *CartHandler) handleRemoveItem(w http.ResponseWriter, r *http.Request, start time.Time) {
	var req cartItemReq
	if err := readJSON(r, &req); err != nil {
		log.Printf("[mall_cart_handler] DELETE remove-item exit status=400 reason=invalid json err=%v\n", err)
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}

	avatarID := readAvatarID(r, req.AvatarID)

	invID := strings.TrimSpace(req.InventoryID)
	listID := strings.TrimSpace(req.ListID)
	modelID := strings.TrimSpace(req.ModelID)

	log.Printf("[mall_cart_handler] DELETE remove-item request avatarId=%q invID=%q listID=%q modelID=%q\n", avatarID, invID, listID, modelID)

	if avatarID == "" || invID == "" || listID == "" || modelID == "" {
		log.Printf("[mall_cart_handler] DELETE remove-item exit status=400 reason=missing fields avatarId=%q invID=%q listID=%q modelID=%q\n", avatarID, invID, listID, modelID)
		writeErr(w, http.StatusBadRequest, "avatarId, inventoryId, listId, modelId are required")
		return
	}

	_, err := h.uc.RemoveItem(r.Context(), avatarID, invID, listID, modelID)
	if err != nil {
		log.Printf("[mall_cart_handler] DELETE remove-item uc error avatarId=%q err=%v\n", avatarID, err)
		if errors.Is(err, usecase.ErrCartInvalidArgument) || errors.Is(err, cartdom.ErrInvalidCart) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Printf("[mall_cart_handler] DELETE remove-item uc ok avatarId=%q -> respond cartDTO\n", avatarID)
	h.respondCartDTO(w, r, avatarID, start)
}

func (h *CartHandler) handleClear(w http.ResponseWriter, r *http.Request, start time.Time) {
	avatarID := readAvatarID(r, "")
	if avatarID == "" {
		log.Printf("[mall_cart_handler] DELETE clear exit status=400 reason=avatarId missing rawQuery=%q\n", r.URL.RawQuery)
		writeErr(w, http.StatusBadRequest, "avatarId is required")
		return
	}

	log.Printf("[mall_cart_handler] DELETE clear request avatarId=%q\n", avatarID)

	if err := h.uc.Clear(r.Context(), avatarID); err != nil {
		log.Printf("[mall_cart_handler] DELETE clear uc error avatarId=%q err=%v\n", avatarID, err)
		if errors.Is(err, usecase.ErrCartInvalidArgument) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Printf("[mall_cart_handler] DELETE clear uc ok avatarId=%q -> respond cartDTO\n", avatarID)
	h.respondCartDTO(w, r, avatarID, start)
}

func (h *CartHandler) respondCartDTO(w http.ResponseWriter, r *http.Request, avatarID string, start time.Time) {
	if h.cartQuery == nil {
		log.Printf("[mall_cart_handler] respondCartDTO exit status=500 reason=cartQuery nil avatarId=%q\n", avatarID)
		writeErr(w, http.StatusInternalServerError, "cart_query is not configured")
		return
	}

	log.Printf("[mall_cart_handler] respondCartDTO call cartQuery avatarId=%q queryImpl=%T\n", avatarID, h.cartQuery)

	v, err := h.cartQuery.GetCartQuery(r.Context(), avatarID)
	if err == nil {
		log.Printf("[mall_cart_handler] respondCartDTO ok status=200 avatarId=%q elapsed=%s\n", avatarID, time.Since(start))
		writeJSON(w, http.StatusOK, v)
		return
	}

	nf := isNotFoundErr(err)
	log.Printf("[mall_cart_handler] respondCartDTO cartQuery error avatarId=%q notFound=%t err=%v\n", avatarID, nf, err)

	if nf {
		log.Printf("[mall_cart_handler] respondCartDTO return empty-cart status=200 avatarId=%q elapsed=%s\n", avatarID, time.Since(start))
		writeJSON(w, http.StatusOK, map[string]any{
			"avatarId":  avatarID,
			"items":     map[string]any{},
			"createdAt": nil,
			"updatedAt": nil,
			"expiresAt": nil,
		})
		return
	}

	log.Printf("[mall_cart_handler] respondCartDTO exit status=500 avatarId=%q elapsed=%s\n", avatarID, time.Since(start))
	h.writeQueryErr(w, err)
}

// -------------------------
// request DTO
// -------------------------

type cartItemReq struct {
	AvatarID     string `json:"avatarId"`
	InventoryID  string `json:"inventoryId"`
	ListID       string `json:"listId"`
	ModelID      string `json:"modelId"`
	Qty          int    `json:"qty"`
	ItemKey      string `json:"itemKey"`
	LegacyModel  string `json:"-"`
	LegacyListID string `json:"-"`
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

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{
		"error": msg,
	})
}

// -------------------------
// best-effort adapters (DI 互換)
// -------------------------

type dynamicCartQuery struct {
	impl          any
	lastMethodHit string
}

func wrapCartQuery(v any) CartQueryService {
	if v == nil {
		return nil
	}
	if s, ok := v.(CartQueryService); ok {
		return s
	}
	return &dynamicCartQuery{impl: v}
}

func (d *dynamicCartQuery) GetCartQuery(ctx context.Context, avatarID string) (any, error) {
	v, hit, err := callQuery2WithHit(d.impl, ctx, avatarID,
		"GetCartQuery",
		"GetByAvatarID",
		"GetCart",
		"Get",
		"Query",
		"Fetch",
	)
	if hit != "" {
		d.lastMethodHit = hit
	}
	log.Printf("[mall_cart_handler] dynamicCartQuery call avatarId=%q impl=%T hit=%q err=%v\n", avatarID, d.impl, d.lastMethodHit, err)
	return v, err
}

func callQuery2WithHit(impl any, ctx context.Context, avatarID string, methodNames ...string) (any, string, error) {
	if impl == nil {
		return nil, "", errors.New("query service is nil")
	}

	rv := reflect.ValueOf(impl)
	if !rv.IsValid() {
		return nil, "", errors.New("query service is invalid")
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

		return out[0].Interface(), name, e
	}

	return nil, "", errors.New("query service method not found")
}
