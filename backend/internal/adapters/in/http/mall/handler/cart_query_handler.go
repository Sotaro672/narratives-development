// backend\internal\adapters\in\http\mall\handler\cart_query_handler.go
package mallHandler

import (
	"errors"
	"log"
	"net/http"
	"strings"

	mallquery "narratives/internal/application/query/mall"
	malldto "narratives/internal/application/query/mall/dto"
)

// MallCartQueryHandler exposes CartQuery via HTTP.
// - GET /mall/cart/query?avatarId=...
// - (router 側で振り分ければ) GET /mall/cart?avatarId=... も同じレスポンスで返せる
type CartQueryHandler struct {
	Q *mallquery.CartQuery
}

func NewCartQueryHandler(q *mallquery.CartQuery) http.Handler {
	return &CartQueryHandler{Q: q}
}

func (h *CartQueryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Q == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "mall cart query handler: query is nil",
		})
		return
	}

	// Only GET
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"error": "method not allowed",
		})
		return
	}

	aid := strings.TrimSpace(r.URL.Query().Get("avatarId"))
	if aid == "" {
		// 互換: aid でも拾う（あっても害はない）
		aid = strings.TrimSpace(r.URL.Query().Get("aid"))
	}
	if aid == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "avatarId is required",
		})
		return
	}

	dto, err := h.Q.GetByAvatarID(r.Context(), aid)
	if err != nil {
		// ✅ carts/{avatarId} が無い場合でも “空カートで 200” にする（UI を白画面にしない）
		if errors.Is(err, mallquery.ErrNotFound) {
			empty := malldto.CartDTO{
				AvatarID: aid,
				Items:    map[string]malldto.CartItemDTO{},
			}
			writeJSON(w, http.StatusOK, empty)
			return
		}

		log.Printf("[mall_cart_query_handler] error path=%s avatarId=%q err=%v", r.URL.Path, maskUIDLite(aid), err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, dto)
}

// ここはログ用の軽いマスク（query 側の maskUID と違ってもOK）
func maskUIDLite(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 6 {
		return "***"
	}
	return s[:3] + "***" + s[len(s)-3:]
}
