// backend/internal/adapters/in/http/sns/handler/cart_handler.go
package handler

import (
	"errors"
	"net/http"
	"sort"
	"strings"

	usecase "narratives/internal/application/usecase"
	cartdom "narratives/internal/domain/cart"
)

// CartHandler serves SNS cart endpoints.
// Intended mount examples (router side):
// - GET    /sns/cart
// - POST   /sns/cart/items           (add)
// - PUT    /sns/cart/items           (set qty)
// - DELETE /sns/cart/items           (remove)
// - DELETE /sns/cart                (clear)
//
// AvatarID resolution policy:
// - query: ?avatarId=...
// - header: X-Avatar-Id: ...
// - (optional) body.avatarId (for mutations)
type CartHandler struct {
	uc *usecase.CartUsecase
}

func NewCartHandler(uc *usecase.CartUsecase) http.Handler {
	return &CartHandler{uc: uc}
}

func (h *CartHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.uc == nil {
		writeErr(w, http.StatusInternalServerError, "cart handler is not configured")
		return
	}

	path := strings.TrimRight(r.URL.Path, "/")

	switch {
	// ====== GET /sns/cart
	case strings.HasSuffix(path, "/sns/cart") && r.Method == http.MethodGet:
		h.handleGet(w, r)
		return

	// ====== DELETE /sns/cart
	case strings.HasSuffix(path, "/sns/cart") && r.Method == http.MethodDelete:
		h.handleClear(w, r)
		return

	// ====== POST /sns/cart/items (add)
	case strings.HasSuffix(path, "/sns/cart/items") && r.Method == http.MethodPost:
		h.handleAddItem(w, r)
		return

	// ====== PUT /sns/cart/items (set qty)
	case strings.HasSuffix(path, "/sns/cart/items") && r.Method == http.MethodPut:
		h.handleSetItemQty(w, r)
		return

	// ====== DELETE /sns/cart/items (remove)
	case strings.HasSuffix(path, "/sns/cart/items") && r.Method == http.MethodDelete:
		h.handleRemoveItem(w, r)
		return
	}

	writeErr(w, http.StatusNotFound, "not found")
}

// -------------------------
// handlers
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
	modelID := strings.TrimSpace(req.ModelID)
	if avatarID == "" || modelID == "" || req.Qty <= 0 {
		writeErr(w, http.StatusBadRequest, "avatarId, modelId, qty(>=1) are required")
		return
	}

	c, err := h.uc.AddItem(r.Context(), avatarID, modelID, req.Qty)
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
	modelID := strings.TrimSpace(req.ModelID)
	if avatarID == "" || modelID == "" {
		writeErr(w, http.StatusBadRequest, "avatarId and modelId are required")
		return
	}

	// qty can be 0 or negative -> treated as remove
	c, err := h.uc.SetItemQty(r.Context(), avatarID, modelID, req.Qty)
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
	modelID := strings.TrimSpace(req.ModelID)
	if avatarID == "" || modelID == "" {
		writeErr(w, http.StatusBadRequest, "avatarId and modelId are required")
		return
	}

	c, err := h.uc.RemoveItem(r.Context(), avatarID, modelID)
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

	// 204 No Content
	w.WriteHeader(http.StatusNoContent)
}

// -------------------------
// request/response DTO
// -------------------------

type cartItemReq struct {
	AvatarID string `json:"avatarId"`
	ModelID  string `json:"modelId"`
	Qty      int    `json:"qty"`
}

type cartResponse struct {
	// ✅ docId=avatarId のため、Cart ドメインから AvatarID を削除してもレスポンスでは返す
	AvatarID string `json:"avatarId"`

	Items map[string]int `json:"items"` // modelId -> qty

	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	ExpiresAt string `json:"expiresAt"`
}

func toCartResponse(avatarID string, c *cartdom.Cart) cartResponse {
	items := map[string]int{}
	if c != nil && c.Items != nil {
		// stable copy
		keys := make([]string, 0, len(c.Items))
		for k := range c.Items {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			items[k] = c.Items[k]
		}
	}

	if c == nil {
		// ここに来るのは通常ないが、念のため
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
	// query
	if v := strings.TrimSpace(r.URL.Query().Get("avatarId")); v != "" {
		return v
	}
	// header
	if v := strings.TrimSpace(r.Header.Get("X-Avatar-Id")); v != "" {
		return v
	}
	// fallback (body)
	return strings.TrimSpace(fallback)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{
		"error": msg,
	})
}
