// backend/internal/adapters/in/http/mall/handler/cart_handler.go
package mallHandler

import (
	"context"
	"errors"
	"net/http"
	"strings"

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

	// read-model query.
	// GET /mall/me/cart は、cartQuery が設定されていれば cartQuery を正として返す。
	// cartQuery が nil または not found の場合だけ domain cart DTO に fallback する。
	cartQuery CartQueryService
}

func NewCartHandler(
	uc *usecase.CartUsecase,
	cartQuery CartQueryService,
) http.Handler {
	return &CartHandler{
		uc:        uc,
		cartQuery: cartQuery,
	}
}

func (h *CartHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if h.uc == nil {
		writeErr(w, http.StatusInternalServerError, "cart handler is not configured")
		return
	}

	isGET := r.Method == http.MethodGet
	isDEL := r.Method == http.MethodDelete
	isPOST := r.Method == http.MethodPost
	isPUT := r.Method == http.MethodPut

	switch {
	case isGET && path == "/mall/me/cart":
		h.handleGetUnified(w, r)
		return
	case isDEL && path == "/mall/me/cart":
		h.handleClear(w, r)
		return
	case isPOST && path == "/mall/me/cart/items":
		h.handleAddItem(w, r)
		return
	case isPUT && path == "/mall/me/cart/items":
		h.handleSetItemQty(w, r)
		return
	case isDEL && path == "/mall/me/cart/items":
		h.handleRemoveItem(w, r)
		return
	case isPOST && path == "/mall/me/cart/resales":
		h.handleAddResaleItem(w, r)
		return
	case isDEL && path == "/mall/me/cart/resales":
		h.handleRemoveResaleItem(w, r)
		return
	}

	writeErr(w, http.StatusNotFound, "not found")
}

func (h *CartHandler) handleGetUnified(w http.ResponseWriter, r *http.Request) {
	avatarID := readAvatarID(r, "")
	if avatarID == "" {
		writeErr(w, http.StatusBadRequest, "avatarId is required")
		return
	}

	h.respondCartDTO(w, r, avatarID)
}

func (h *CartHandler) handleAddItem(w http.ResponseWriter, r *http.Request) {
	var req cartItemReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}

	avatarID := readAvatarID(r, req.AvatarID)
	if avatarID == "" || req.InventoryID == "" || req.ListID == "" || req.ModelID == "" || req.Qty <= 0 {
		writeErr(w, http.StatusBadRequest, "avatarId, inventoryId, listId, modelId, qty(>=1) are required")
		return
	}

	_, err := h.uc.AddItem(r.Context(), avatarID, req.InventoryID, req.ListID, req.ModelID, req.Qty)
	if err != nil {
		h.writeMutationErr(w, err)
		return
	}

	h.respondCartDTO(w, r, avatarID)
}

func (h *CartHandler) handleAddResaleItem(w http.ResponseWriter, r *http.Request) {
	var req cartItemReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}

	avatarID := readAvatarID(r, req.AvatarID)
	if avatarID == "" || req.ResaleID == "" || req.ProductID == "" {
		writeErr(w, http.StatusBadRequest, "avatarId, resaleId, productId are required")
		return
	}

	_, err := h.uc.AddResaleItem(r.Context(), avatarID, req.ResaleID, req.ProductID)
	if err != nil {
		h.writeMutationErr(w, err)
		return
	}

	h.respondCartDTO(w, r, avatarID)
}

func (h *CartHandler) handleSetItemQty(w http.ResponseWriter, r *http.Request) {
	var req cartItemReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}

	avatarID := readAvatarID(r, req.AvatarID)
	if avatarID == "" || req.InventoryID == "" || req.ListID == "" || req.ModelID == "" {
		writeErr(w, http.StatusBadRequest, "avatarId, inventoryId, listId, modelId are required")
		return
	}

	_, err := h.uc.SetItemQty(r.Context(), avatarID, req.InventoryID, req.ListID, req.ModelID, req.Qty)
	if err != nil {
		h.writeMutationErr(w, err)
		return
	}

	h.respondCartDTO(w, r, avatarID)
}

func (h *CartHandler) handleRemoveItem(w http.ResponseWriter, r *http.Request) {
	var req cartItemReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}

	avatarID := readAvatarID(r, req.AvatarID)
	if avatarID == "" || req.InventoryID == "" || req.ListID == "" || req.ModelID == "" {
		writeErr(w, http.StatusBadRequest, "avatarId, inventoryId, listId, modelId are required")
		return
	}

	_, err := h.uc.RemoveItem(r.Context(), avatarID, req.InventoryID, req.ListID, req.ModelID)
	if err != nil {
		h.writeMutationErr(w, err)
		return
	}

	h.respondCartDTO(w, r, avatarID)
}

func (h *CartHandler) handleRemoveResaleItem(w http.ResponseWriter, r *http.Request) {
	var req cartItemReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}

	avatarID := readAvatarID(r, req.AvatarID)
	if avatarID == "" || req.ResaleID == "" || req.ProductID == "" {
		writeErr(w, http.StatusBadRequest, "avatarId, resaleId, productId are required")
		return
	}

	_, err := h.uc.RemoveResaleItem(r.Context(), avatarID, req.ResaleID, req.ProductID)
	if err != nil {
		h.writeMutationErr(w, err)
		return
	}

	h.respondCartDTO(w, r, avatarID)
}

func (h *CartHandler) handleClear(w http.ResponseWriter, r *http.Request) {
	avatarID := readAvatarID(r, "")
	if avatarID == "" {
		writeErr(w, http.StatusBadRequest, "avatarId is required")
		return
	}

	if err := h.uc.Clear(r.Context(), avatarID); err != nil {
		h.writeMutationErr(w, err)
		return
	}

	h.respondCartDTO(w, r, avatarID)
}

func (h *CartHandler) respondCartDTO(w http.ResponseWriter, r *http.Request, avatarID string) {
	if h.cartQuery != nil {
		v, err := h.cartQuery.GetCartQuery(r.Context(), avatarID)
		if err == nil {
			writeJSON(w, http.StatusOK, v)
			return
		}

		if !isNotFoundErr(err) {
			h.writeQueryErr(w, err)
			return
		}
	}

	domainDTO, domainErr := h.getDomainCartDTO(r.Context(), avatarID)
	if domainErr != nil {
		h.writeQueryErr(w, domainErr)
		return
	}

	writeJSON(w, http.StatusOK, domainDTO)
}

func (h *CartHandler) getDomainCartDTO(ctx context.Context, avatarID string) (map[string]any, error) {
	c, err := h.uc.Get(ctx, avatarID)
	if err != nil {
		if errors.Is(err, usecase.ErrCartNotFound) {
			return emptyCartDTO(avatarID), nil
		}
		return nil, err
	}

	return cartToDTO(avatarID, c), nil
}

func cartToDTO(avatarID string, c *cartdom.Cart) map[string]any {
	if c == nil {
		return emptyCartDTO(avatarID)
	}

	items := map[string]any{}
	for k, it := range c.Items {
		item, ok := cartItemToDTO(it)
		if k == "" || !ok {
			continue
		}
		items[k] = item
	}

	return map[string]any{
		"avatarId":  avatarID,
		"items":     items,
		"createdAt": c.CreatedAt,
		"updatedAt": c.UpdatedAt,
		"expiresAt": c.ExpiresAt,
	}
}

func cartItemToDTO(it cartdom.CartItem) (map[string]any, bool) {
	switch inferCartItemType(it) {
	case cartdom.CartItemTypeList:
		if it.InventoryID == "" || it.ListID == "" || it.ModelID == "" || it.Qty <= 0 {
			return nil, false
		}
		return map[string]any{
			"type":        string(cartdom.CartItemTypeList),
			"inventoryId": it.InventoryID,
			"listId":      it.ListID,
			"modelId":     it.ModelID,
			"qty":         it.Qty,
		}, true

	case cartdom.CartItemTypeResale:
		if it.ResaleID == "" || it.ProductID == "" {
			return nil, false
		}
		return map[string]any{
			"type":      string(cartdom.CartItemTypeResale),
			"resaleId":  it.ResaleID,
			"productId": it.ProductID,
			"qty":       1,
		}, true

	default:
		return nil, false
	}
}

func inferCartItemType(it cartdom.CartItem) cartdom.CartItemType {
	switch it.Type {
	case cartdom.CartItemTypeList, cartdom.CartItemTypeResale:
		return it.Type
	}

	if it.ResaleID != "" || it.ProductID != "" {
		return cartdom.CartItemTypeResale
	}

	if it.InventoryID != "" || it.ListID != "" || it.ModelID != "" {
		return cartdom.CartItemTypeList
	}

	return ""
}

func emptyCartDTO(avatarID string) map[string]any {
	return map[string]any{
		"avatarId":  avatarID,
		"items":     map[string]any{},
		"createdAt": nil,
		"updatedAt": nil,
		"expiresAt": nil,
	}
}

func (h *CartHandler) writeMutationErr(w http.ResponseWriter, err error) {
	if errors.Is(err, usecase.ErrCartInvalidArgument) || errors.Is(err, cartdom.ErrInvalidCart) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	writeErr(w, http.StatusInternalServerError, err.Error())
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

	s := strings.ToLower(err.Error())
	return strings.Contains(s, "not found") ||
		strings.Contains(s, "notfound") ||
		strings.Contains(s, "no such") ||
		strings.Contains(s, "missing")
}

type cartItemReq struct {
	AvatarID    string `json:"avatarId"`
	InventoryID string `json:"inventoryId"`
	ListID      string `json:"listId"`
	ModelID     string `json:"modelId"`
	ResaleID    string `json:"resaleId"`
	ProductID   string `json:"productId"`
	Qty         int    `json:"qty"`
	ItemKey     string `json:"itemKey"`
}

func readAvatarID(r *http.Request, fallback string) string {
	if v := r.URL.Query().Get("avatarId"); v != "" {
		return v
	}

	if v := r.Header.Get("X-Avatar-Id"); v != "" {
		return v
	}

	return fallback
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{
		"error": msg,
	})
}
