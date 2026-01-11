// backend/internal/adapters/in/http/mall/handler/preview_handler.go
package mallHandler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
)

// ✅ 組み立て用（DI）前提:
// backend/internal/application/query/mall/preview_query.go が提供する Query を注入して使う想定。
// この handler は「productId → modelId」を返す最小実装。
//
// 想定エンドポイント:
// - GET /mall/preview?productId=...
//
// 互換で以下も吸収:
// - GET /mall/preview/{productId}
type PreviewQuery interface {
	ResolveModelIDByProductID(ctx context.Context, productID string) (string, error)
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
		log.Printf("[mall.preview] method_not_allowed method=%s path=%s", r.Method, r.URL.Path)
		// ✅ helper_handler.go 側の writeJSON を使う前提（同一package内）
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"error": "method not allowed",
		})
		return
	}

	if h.q == nil {
		log.Printf("[mall.preview] ERROR: preview query not configured path=%s", r.URL.Path)
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "preview query not configured",
		})
		return
	}

	// public 側でも認証ヘッダが付いてくる可能性があるため、存在だけログ（中身は出さない）
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	authPrefix := ""
	if auth != "" {
		parts := strings.SplitN(auth, " ", 2)
		authPrefix = strings.TrimSpace(parts[0])
	}

	log.Printf(
		"[mall.preview] incoming method=%s path=%s rawQuery=%q hasAuth=%t authPrefix=%q",
		r.Method,
		r.URL.Path,
		r.URL.RawQuery,
		auth != "",
		authPrefix,
	)

	productIDQuery := strings.TrimSpace(r.URL.Query().Get("productId"))
	productIDPath := ""
	if productIDQuery == "" {
		// 互換: /mall/preview/{productId}
		productIDPath = extractLastPathSegment(r.URL.Path, "/mall/preview")
	}
	productID := productIDQuery
	if productID == "" {
		productID = productIDPath
	}

	log.Printf(
		"[mall.preview] parsed productId query=%q path=%q resolved=%q",
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

	log.Printf("[mall.preview] resolving modelId productId=%q", productID)

	modelID, err := h.q.ResolveModelIDByProductID(r.Context(), productID)
	if err != nil {
		log.Printf(
			"[mall.preview] resolve failed productId=%q err=%T %v",
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
		"[mall.preview] resolve OK productId=%q modelId=%q",
		productID,
		strings.TrimSpace(modelID),
	)

	// 最小レスポンス（将来 product / blueprint / brand など増やしても崩れにくい形）
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"productId": productID,
			"modelId":   modelID,
		},
	})
}

func extractLastPathSegment(path string, prefix string) string {
	p := strings.TrimSuffix(path, "/")
	prefix = strings.TrimSuffix(prefix, "/")

	// /mall/preview または /mall/preview/{id}
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
	// 念のためさらに分割（/a/b のような入力対策）
	if i := strings.Index(rest, "/"); i >= 0 {
		rest = rest[:i]
	}
	return strings.TrimSpace(rest)
}

// isNotFound: アプリ側の not found を best-effort で吸収
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	// よくある文字列/ラップの吸収（repo 実装に依存しない）
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "not found") || strings.Contains(msg, "no such") {
		return true
	}

	// context
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	return false
}
