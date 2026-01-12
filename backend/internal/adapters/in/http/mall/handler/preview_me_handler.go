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
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"error": "method not allowed",
		})
		return
	}

	if h.q == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "preview query not configured",
		})
		return
	}

	// ✅ /mall/me/preview は認証前提（通常は middleware で検証される）
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": "authorization header is required",
		})
		return
	}

	productID := strings.TrimSpace(r.URL.Query().Get("productId"))
	if productID == "" {
		// 互換: /mall/me/preview/{productId}
		productID = extractLastPathSegment(r.URL.Path, "/mall/me/preview")
	}

	if productID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "productId is required",
		})
		return
	}

	// ✅ 入口ログ
	authPrefix := ""
	if len(auth) > 12 {
		authPrefix = auth[:12]
	} else {
		authPrefix = auth
	}
	log.Printf(
		`[mall.preview.me] incoming method=%s path=%s rawQuery=%q hasAuth=%t authPrefix=%q`,
		r.Method,
		r.URL.Path,
		r.URL.RawQuery,
		auth != "",
		authPrefix,
	)

	log.Printf(`[mall.preview.me] resolving modelId productId=%q`, productID)

	modelID, err := h.q.ResolveModelIDByProductID(r.Context(), productID)
	if err != nil {
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

	modelNumber, size, colorName, rgb, err := h.q.ResolveModelMetaByModelID(r.Context(), modelID)
	if err != nil {
		if isNotFound(err) {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"error":     "model not found",
				"productId": productID,
				"modelId":   modelID,
			})
			return
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			writeJSON(w, http.StatusRequestTimeout, map[string]any{
				"error":     "request canceled",
				"productId": productID,
				"modelId":   modelID,
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":     "resolve model meta failed",
			"productId": productID,
			"modelId":   modelID,
		})
		return
	}

	log.Printf(
		`[mall.preview.me] resolved productId=%q modelId=%q modelNumber=%q size=%q color=%q rgb=%q`,
		productID, modelID, modelNumber, size, colorName, rgb,
	)

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"productId":   productID,
			"modelId":     modelID,
			"modelNumber": modelNumber,
			"size":        size,
			"color":       colorName,
			"rgb":         rgb, // ✅ int のまま返す
		},
	})
}
