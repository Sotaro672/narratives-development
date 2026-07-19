// backend/internal/adapters/in/http/mall/handler/cart_handler.go
package mallHandler

import (
	"context"
	"errors"
	"net/http"

	middleware "narratives/internal/adapters/in/http/middleware"
	usecase "narratives/internal/application/usecase"
	cartdom "narratives/internal/domain/cart"
)

type CartQueryService interface {
	GetCartQuery(
		ctx context.Context,
		avatarID string,
	) (any, error)
}

type CartHandler struct {
	uc        *usecase.CartUsecase
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

func (h *CartHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil || h.uc == nil {
		writeErr(
			w,
			http.StatusInternalServerError,
			"cart handler is not configured",
		)
		return
	}

	path := r.URL.Path

	switch {
	case r.Method == http.MethodGet &&
		path == "/mall/me/cart":
		h.handleGet(w, r)

	case r.Method == http.MethodDelete &&
		path == "/mall/me/cart":
		h.handleClear(w, r)

	case r.Method == http.MethodPost &&
		path == "/mall/me/cart/items":
		h.handleAddItem(w, r)

	case r.Method == http.MethodPut &&
		path == "/mall/me/cart/items":
		h.handleSetItemQty(w, r)

	case r.Method == http.MethodDelete &&
		path == "/mall/me/cart/items":
		h.handleRemoveItem(w, r)

	case r.Method == http.MethodPost &&
		path == "/mall/me/cart/resales":
		h.handleAddResaleItem(w, r)

	case r.Method == http.MethodDelete &&
		path == "/mall/me/cart/resales":
		h.handleRemoveResaleItem(w, r)

	default:
		writeErr(w, http.StatusNotFound, "not found")
	}
}

func (h *CartHandler) handleGet(
	w http.ResponseWriter,
	r *http.Request,
) {
	avatarID, ok := currentCartAvatarID(w, r)
	if !ok {
		return
	}

	h.respondCartDTO(w, r, avatarID)
}

func (h *CartHandler) handleAddItem(
	w http.ResponseWriter,
	r *http.Request,
) {
	avatarID, ok := currentCartAvatarID(w, r)
	if !ok {
		return
	}

	var request cartItemReq
	if err := readJSON(r, &request); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}

	if request.InventoryID == "" ||
		request.ListID == "" ||
		request.ModelID == "" ||
		request.Qty <= 0 {
		writeErr(
			w,
			http.StatusBadRequest,
			"inventoryId, listId, modelId, qty(>=1) are required",
		)
		return
	}

	_, err := h.uc.AddItem(
		r.Context(),
		avatarID,
		request.InventoryID,
		request.ListID,
		request.ModelID,
		request.Qty,
	)
	if err != nil {
		h.writeMutationErr(w, err)
		return
	}

	h.respondCartDTO(w, r, avatarID)
}

func (h *CartHandler) handleAddResaleItem(
	w http.ResponseWriter,
	r *http.Request,
) {
	avatarID, ok := currentCartAvatarID(w, r)
	if !ok {
		return
	}

	var request cartItemReq
	if err := readJSON(r, &request); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}

	if request.ResaleID == "" ||
		request.ProductID == "" {
		writeErr(
			w,
			http.StatusBadRequest,
			"resaleId and productId are required",
		)
		return
	}

	_, err := h.uc.AddResaleItem(
		r.Context(),
		avatarID,
		request.ResaleID,
		request.ProductID,
	)
	if err != nil {
		h.writeMutationErr(w, err)
		return
	}

	h.respondCartDTO(w, r, avatarID)
}

func (h *CartHandler) handleSetItemQty(
	w http.ResponseWriter,
	r *http.Request,
) {
	avatarID, ok := currentCartAvatarID(w, r)
	if !ok {
		return
	}

	var request cartItemReq
	if err := readJSON(r, &request); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}

	if request.InventoryID == "" ||
		request.ListID == "" ||
		request.ModelID == "" {
		writeErr(
			w,
			http.StatusBadRequest,
			"inventoryId, listId and modelId are required",
		)
		return
	}

	_, err := h.uc.SetItemQty(
		r.Context(),
		avatarID,
		request.InventoryID,
		request.ListID,
		request.ModelID,
		request.Qty,
	)
	if err != nil {
		h.writeMutationErr(w, err)
		return
	}

	h.respondCartDTO(w, r, avatarID)
}

func (h *CartHandler) handleRemoveItem(
	w http.ResponseWriter,
	r *http.Request,
) {
	avatarID, ok := currentCartAvatarID(w, r)
	if !ok {
		return
	}

	var request cartItemReq
	if err := readJSON(r, &request); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}

	if request.InventoryID == "" ||
		request.ListID == "" ||
		request.ModelID == "" {
		writeErr(
			w,
			http.StatusBadRequest,
			"inventoryId, listId and modelId are required",
		)
		return
	}

	_, err := h.uc.RemoveItem(
		r.Context(),
		avatarID,
		request.InventoryID,
		request.ListID,
		request.ModelID,
	)
	if err != nil {
		h.writeMutationErr(w, err)
		return
	}

	h.respondCartDTO(w, r, avatarID)
}

func (h *CartHandler) handleRemoveResaleItem(
	w http.ResponseWriter,
	r *http.Request,
) {
	avatarID, ok := currentCartAvatarID(w, r)
	if !ok {
		return
	}

	var request cartItemReq
	if err := readJSON(r, &request); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}

	if request.ResaleID == "" ||
		request.ProductID == "" {
		writeErr(
			w,
			http.StatusBadRequest,
			"resaleId and productId are required",
		)
		return
	}

	_, err := h.uc.RemoveResaleItem(
		r.Context(),
		avatarID,
		request.ResaleID,
		request.ProductID,
	)
	if err != nil {
		h.writeMutationErr(w, err)
		return
	}

	h.respondCartDTO(w, r, avatarID)
}

func (h *CartHandler) handleClear(
	w http.ResponseWriter,
	r *http.Request,
) {
	avatarID, ok := currentCartAvatarID(w, r)
	if !ok {
		return
	}

	if err := h.uc.Clear(
		r.Context(),
		avatarID,
	); err != nil {
		h.writeMutationErr(w, err)
		return
	}

	h.respondCartDTO(w, r, avatarID)
}

func currentCartAvatarID(
	w http.ResponseWriter,
	r *http.Request,
) (string, bool) {
	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		writeErr(
			w,
			http.StatusUnauthorized,
			"unauthorized: missing avatarId",
		)
		return "", false
	}

	return avatarID, true
}

func (h *CartHandler) respondCartDTO(
	w http.ResponseWriter,
	r *http.Request,
	avatarID string,
) {
	if h.cartQuery == nil {
		writeErr(
			w,
			http.StatusInternalServerError,
			"cart query is not configured",
		)
		return
	}

	result, err := h.cartQuery.GetCartQuery(
		r.Context(),
		avatarID,
	)
	if err != nil {
		h.writeQueryErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *CartHandler) writeMutationErr(
	w http.ResponseWriter,
	err error,
) {
	if errors.Is(err, usecase.ErrCartInvalidArgument) ||
		errors.Is(err, cartdom.ErrInvalidCart) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	writeErr(
		w,
		http.StatusInternalServerError,
		err.Error(),
	)
}

func (h *CartHandler) writeQueryErr(
	w http.ResponseWriter,
	err error,
) {
	if err == nil {
		writeErr(
			w,
			http.StatusInternalServerError,
			"unknown error",
		)
		return
	}

	if errors.Is(err, usecase.ErrCartInvalidArgument) ||
		errors.Is(err, cartdom.ErrInvalidCart) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	writeErr(
		w,
		http.StatusInternalServerError,
		err.Error(),
	)
}

type cartItemReq struct {
	InventoryID string `json:"inventoryId"`
	ListID      string `json:"listId"`
	ModelID     string `json:"modelId"`
	ResaleID    string `json:"resaleId"`
	ProductID   string `json:"productId"`
	Qty         int    `json:"qty"`
	ItemKey     string `json:"itemKey"`
}

func writeErr(
	w http.ResponseWriter,
	status int,
	message string,
) {
	writeJSON(w, status, map[string]any{
		"error": message,
	})
}
