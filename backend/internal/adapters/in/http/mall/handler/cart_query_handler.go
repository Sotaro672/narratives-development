// backend\internal\adapters\in\http\sns\handler\cart_query_handler.go
package mallHandler

import (
	"errors"
	"log"
	"net/http"
	"strings"

	snsquery "narratives/internal/application/query/mall"
	snsdto "narratives/internal/application/query/mall/dto"
)

// SNSCartQueryHandler exposes SNSCartQuery via HTTP.
// - GET /sns/cart/query?avatarId=...
// - (router 側で振り分ければ) GET /sns/cart?avatarId=... も同じレスポンスで返せる
type SNSCartQueryHandler struct {
	Q *snsquery.SNSCartQuery
}

func NewSNSCartQueryHandler(q *snsquery.SNSCartQuery) http.Handler {
	return &SNSCartQueryHandler{Q: q}
}

func (h *SNSCartQueryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Q == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "sns cart query handler: query is nil",
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
		if errors.Is(err, snsquery.ErrNotFound) {
			empty := snsdto.CartDTO{
				AvatarID: aid,
				Items:    map[string]snsdto.CartItemDTO{},
			}
			writeJSON(w, http.StatusOK, empty)
			return
		}

		log.Printf("[sns_cart_query_handler] error path=%s avatarId=%q err=%v", r.URL.Path, maskUIDLite(aid), err)
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
