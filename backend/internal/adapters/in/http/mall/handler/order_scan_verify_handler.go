// backend/internal/adapters/in/http/mall/handler/order_scan_verify_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"narratives/internal/adapters/in/http/middleware"
	appusecase "narratives/internal/application/usecase"
)

type ScanVerifyResult = appusecase.VerifyResult
type ModelTokenPair = appusecase.ModelTokenPair

type ScanVerifyQuery interface {
	VerifyMatch(ctx context.Context, in appusecase.VerifyInput) (appusecase.VerifyResult, error)
}

type OrderScanVerifyHandler struct {
	q ScanVerifyQuery
}

func NewOrderScanVerifyHandler(q ScanVerifyQuery) http.Handler {
	return &OrderScanVerifyHandler{q: q}
}

func (h *OrderScanVerifyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"error": "method not allowed",
		})
		return
	}

	if h == nil || h.q == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "scan verify query not configured",
		})
		return
	}

	auth := r.Header.Get("Authorization")
	if auth == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": "authorization header is required",
		})
		return
	}

	if _, ok := middleware.CurrentUserUID(r); !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": "unauthorized: missing uid",
		})
		return
	}

	var body struct {
		ProductID string `json:"productId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "invalid json",
		})
		return
	}

	productID := body.ProductID
	if productID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "productId is required",
		})
		return
	}

	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": "avatar_context_missing",
		})
		return
	}

	out, err := h.q.VerifyMatch(r.Context(), appusecase.VerifyInput{
		AvatarID:  avatarID,
		ProductID: productID,
	})
	if err != nil {
		if isNotFound(err) {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"error":     "not found",
				"avatarId":  avatarID,
				"productId": productID,
			})
			return
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			writeJSON(w, http.StatusRequestTimeout, map[string]any{
				"error":     "request canceled",
				"avatarId":  avatarID,
				"productId": productID,
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":     "verify failed",
			"avatarId":  avatarID,
			"productId": productID,
		})
		return
	}

	if out.AvatarID == "" {
		out.AvatarID = avatarID
	}
	if out.ProductID == "" {
		out.ProductID = productID
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": out,
	})
}
