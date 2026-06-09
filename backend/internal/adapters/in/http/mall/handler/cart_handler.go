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

	// read-model queries (required for unified GET)
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
	// Unified GET
	case isGET && path == "/mall/me/cart":
		h.handleGetUnified(w, r)
		return

	// Clear
	case isDEL && path == "/mall/me/cart":
		h.handleClear(w, r)
		return

	// Add item
	case isPOST && path == "/mall/me/cart/items":
		h.handleAddItem(w, r)
		return

	// Set qty
	case isPUT && path == "/mall/me/cart/items":
		h.handleSetItemQty(w, r)
		return

	// Remove item
	case isDEL && path == "/mall/me/cart/items":
		h.handleRemoveItem(w, r)
		return
	}

	writeErr(w, http.StatusNotFound, "not found")
}

// -------------------------
// handlers (Unified GET)
// -------------------------

func (h *CartHandler) handleGetUnified(w http.ResponseWriter, r *http.Request) {
	avatarID := readAvatarID(r, "")
	if avatarID == "" {
		writeErr(w, http.StatusBadRequest, "avatarId is required")
		return
	}

	if h.cartQuery == nil {
		writeErr(w, http.StatusInternalServerError, "cart_query is not configured")
		return
	}

	v, err := h.cartQuery.GetCartQuery(r.Context(), avatarID)
	if err == nil {
		writeJSON(w, http.StatusOK, v)
		return
	}

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

	s := strings.ToLower(err.Error())
	return strings.Contains(s, "not found") ||
		strings.Contains(s, "notfound") ||
		strings.Contains(s, "no such") ||
		strings.Contains(s, "missing")
}

// -------------------------
// handlers (mutations)
// -------------------------

func (h *CartHandler) handleAddItem(w http.ResponseWriter, r *http.Request) {
	var req cartItemReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}

	avatarID := readAvatarID(r, req.AvatarID)

	invID := req.InventoryID
	listID := req.ListID
	modelID := req.ModelID

	if avatarID == "" || invID == "" || listID == "" || modelID == "" || req.Qty <= 0 {
		writeErr(w, http.StatusBadRequest, "avatarId, inventoryId, listId, modelId, qty(>=1) are required")
		return
	}

	_, err := h.uc.AddItem(r.Context(), avatarID, invID, listID, modelID, req.Qty)
	if err != nil {
		if errors.Is(err, usecase.ErrCartInvalidArgument) || errors.Is(err, cartdom.ErrInvalidCart) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}

		writeErr(w, http.StatusInternalServerError, err.Error())
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

	invID := req.InventoryID
	listID := req.ListID
	modelID := req.ModelID

	if avatarID == "" || invID == "" || listID == "" || modelID == "" {
		writeErr(w, http.StatusBadRequest, "avatarId, inventoryId, listId, modelId are required")
		return
	}

	_, err := h.uc.SetItemQty(r.Context(), avatarID, invID, listID, modelID, req.Qty)
	if err != nil {
		if errors.Is(err, usecase.ErrCartInvalidArgument) || errors.Is(err, cartdom.ErrInvalidCart) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}

		writeErr(w, http.StatusInternalServerError, err.Error())
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

	invID := req.InventoryID
	listID := req.ListID
	modelID := req.ModelID

	if avatarID == "" || invID == "" || listID == "" || modelID == "" {
		writeErr(w, http.StatusBadRequest, "avatarId, inventoryId, listId, modelId are required")
		return
	}

	_, err := h.uc.RemoveItem(r.Context(), avatarID, invID, listID, modelID)
	if err != nil {
		if errors.Is(err, usecase.ErrCartInvalidArgument) || errors.Is(err, cartdom.ErrInvalidCart) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}

		writeErr(w, http.StatusInternalServerError, err.Error())
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
		if errors.Is(err, usecase.ErrCartInvalidArgument) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}

		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondCartDTO(w, r, avatarID)
}

func (h *CartHandler) respondCartDTO(w http.ResponseWriter, r *http.Request, avatarID string) {
	if h.cartQuery == nil {
		writeErr(w, http.StatusInternalServerError, "cart_query is not configured")
		return
	}

	v, err := h.cartQuery.GetCartQuery(r.Context(), avatarID)
	if err == nil {
		writeJSON(w, http.StatusOK, v)
		return
	}

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

	h.writeQueryErr(w, err)
}

// -------------------------
// request DTO
// -------------------------

type cartItemReq struct {
	AvatarID    string `json:"avatarId"`
	InventoryID string `json:"inventoryId"`
	ListID      string `json:"listId"`
	ModelID     string `json:"modelId"`
	Qty         int    `json:"qty"`
	ItemKey     string `json:"itemKey"`
}

// -------------------------
// helpers
// -------------------------

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
