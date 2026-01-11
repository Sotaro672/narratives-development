// backend/internal/adapters/in/http/mall/handler/preview_me_handler.go
package mallHandler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
)

// PreviewMeHandler handles authenticated preview endpoint.
//
// 想定エンドポイント:
// - GET /mall/me/preview?productId=...
//
// 互換で以下も吸収:
// - GET /mall/me/preview/{productId}
type PreviewMeHandler struct {
	q PreviewQuery
}

func NewPreviewMeHandler(q PreviewQuery) http.Handler {
	return &PreviewMeHandler{q: q}
}

func (h *PreviewMeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Preflight
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodGet {
		log.Printf("[mall.preview.me] method_not_allowed method=%s path=%s", r.Method, r.URL.Path)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"error": "method not allowed",
		})
		return
	}

	if h.q == nil {
		log.Printf("[mall.preview.me] ERROR: preview query not configured path=%s", r.URL.Path)
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "preview query not configured",
		})
		return
	}

	// ✅ /mall/me/preview は認証前提（通常は middleware で検証される）
	// ただし、万一 middleware 未適用でも分かりやすく落とす
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	// 秘密情報を出さない：存在と prefix のみ
	authPrefix := ""
	if auth != "" {
		parts := strings.SplitN(auth, " ", 2)
		authPrefix = strings.TrimSpace(parts[0])
	}
	log.Printf(
		"[mall.preview.me] incoming method=%s path=%s rawQuery=%q hasAuth=%t authPrefix=%q",
		r.Method,
		r.URL.Path,
		r.URL.RawQuery,
		auth != "",
		authPrefix,
	)

	if auth == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": "authorization header is required",
		})
		return
	}

	productIDQuery := strings.TrimSpace(r.URL.Query().Get("productId"))
	productIDPath := ""
	if productIDQuery == "" {
		// 互換: /mall/me/preview/{productId}
		productIDPath = extractLastPathSegment(r.URL.Path, "/mall/me/preview")
	}
	productID := productIDQuery
	if productID == "" {
		productID = productIDPath
	}

	log.Printf(
		"[mall.preview.me] parsed productId query=%q path=%q resolved=%q",
		productIDQuery,
		productIDPath,
		productID,
	)

	if productID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "productId is required",
		})
		return
	}

	log.Printf("[mall.preview.me] resolving modelId productId=%q", productID)

	modelID, err := h.q.ResolveModelIDByProductID(r.Context(), productID)
	if err != nil {
		log.Printf(
			"[mall.preview.me] resolve failed productId=%q err=%T %v",
			productID,
			err,
			err,
		)

		if isNotFound(err) {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"error":     "not found",
				"productId": productID,
			})
			return
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			writeJSON(w, http.StatusRequestTimeout, map[string]any{
				"error":     "request canceled",
				"productId": productID,
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":     "resolve failed",
			"productId": productID,
		})
		return
	}

	log.Printf(
		"[mall.preview.me] resolve OK productId=%q modelId=%q",
		productID,
		strings.TrimSpace(modelID),
	)

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"productId": productID,
			"modelId":   modelID,
		},
	})
}
