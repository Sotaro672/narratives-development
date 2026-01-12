// backend\internal\adapters\in\http\mall\handler\preview_handler.go
package mallHandler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	mallQuery "narratives/internal/application/query/mall"
)

// ✅ 組み立て用（DI）前提:
// backend/internal/application/query/mall/preview_query.go が提供する Query を注入して使う想定。
// この handler は「productId → model info（型番/サイズ/色/RGB/measurements + productBlueprintPatch）」を返す。
//
// 想定エンドポイント:
// - GET /mall/preview?productId=...
//
// 互換で以下も吸収:
// - GET /mall/preview/{productId}
type PreviewQuery interface {
	// ✅ 推奨: productId から表示用情報を一括で返す（measurements を含む）
	ResolveModelInfoByProductID(ctx context.Context, productID string) (*mallQuery.PreviewModelInfo, error)
}

type PreviewHandler struct {
	q PreviewQuery
}

func NewPreviewHandler(q PreviewQuery) http.Handler {
	return &PreviewHandler{q: q}
}

func (h *PreviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	productID := strings.TrimSpace(r.URL.Query().Get("productId"))
	if productID == "" {
		// 互換: /mall/preview/{productId}
		productID = extractLastPathSegment(r.URL.Path, "/mall/preview")
	}

	if productID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "productId is required",
		})
		return
	}

	// ✅ 入口ログ（Cloud Run ログで「叩かれてるか」を確実に追う）
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	authPrefix := ""
	if auth != "" {
		if len(auth) > 12 {
			authPrefix = auth[:12]
		} else {
			authPrefix = auth
		}
	}
	log.Printf(
		`[mall.preview] incoming method=%s path=%s rawQuery=%q hasAuth=%t authPrefix=%q`,
		r.Method,
		r.URL.Path,
		r.URL.RawQuery,
		auth != "",
		authPrefix,
	)

	log.Printf(`[mall.preview] resolving model info productId=%q`, productID)

	info, err := h.q.ResolveModelInfoByProductID(r.Context(), productID)
	if err != nil {
		log.Printf("[mall.preview.me] ResolveModelInfoByProductID failed: %v", err)

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
	if info == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":     "resolve failed (nil result)",
			"productId": productID,
		})
		return
	}

	log.Printf(
		`[mall.preview] resolved productId=%q modelId=%q modelNumber=%q size=%q color=%q rgb=%d measurements=%v productBlueprintId=%q productBlueprintPatch=%v`,
		info.ProductID,
		info.ModelID,
		info.ModelNumber,
		info.Size,
		info.Color,
		info.RGB,
		info.Measurements,
		info.ProductBlueprintID,
		info.ProductBlueprintPatch,
	)

	// ✅ info をそのまま返す（productBlueprintPatch も data に含まれる）
	writeJSON(w, http.StatusOK, map[string]any{
		"data": info,
	})
}

func extractLastPathSegment(path string, prefix string) string {
	p := strings.TrimSuffix(path, "/")
	prefix = strings.TrimSuffix(prefix, "/")

	if p == prefix {
		return ""
	}
	if !strings.HasPrefix(p, prefix+"/") {
		return ""
	}
	rest := strings.TrimPrefix(p, prefix+"/")
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return ""
	}
	if i := strings.Index(rest, "/"); i >= 0 {
		rest = rest[:i]
	}
	return strings.TrimSpace(rest)
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "not found") || strings.Contains(msg, "no such") {
		return true
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return false
}
