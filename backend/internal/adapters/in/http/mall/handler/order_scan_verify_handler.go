// backend/internal/adapters/in/http/mall/handler/order_scan_verify_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	mallquery "narratives/internal/application/query/mall"

	// uid / avatarId を取る（avatarId は AvatarContextMiddleware が詰める前提）
	"narratives/internal/adapters/in/http/middleware"
)

// ✅ A案: handler 独自型をやめ、アプリ層の型をそのまま使う
// (Dart 側の MallScanVerifyResponse と shape が一致する前提)
type ScanVerifyResult = mallquery.VerifyResult
type ModelTokenPair = mallquery.ModelTokenPair

// ScanVerifyQuery is the dependency for /mall/me/orders/scan/verify.
// ✅ A案: 戻り値を app/query の value 型に合わせる（ポインタをやめる）
type ScanVerifyQuery interface {
	VerifyScanPurchasedByAvatarID(ctx context.Context, avatarId, productId string) (mallquery.VerifyResult, error)
}

// OrderScanVerifyHandler handles:
// POST /mall/me/orders/scan/verify
//
// ✅ Option A: anti-spoof を AvatarContextMiddleware に一本化する。
// - handler は uid->avatarId resolver を持たない
// - avatarId は request context からのみ取得する（body の avatarId は使わない）
type OrderScanVerifyHandler struct {
	q ScanVerifyQuery
}

// NewOrderScanVerifyHandler creates handler.
// NOTE: This handler assumes AvatarContextMiddleware is enabled for this route.
func NewOrderScanVerifyHandler(q ScanVerifyQuery) http.Handler {
	return &OrderScanVerifyHandler{q: q}
}

func (h *OrderScanVerifyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Preflight
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

	// /mall/me/... is auth-required (normally enforced by middleware)
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": "authorization header is required",
		})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": "unauthorized: missing uid",
		})
		return
	}

	// Body: productId only (avatarId is taken from context)
	var body struct {
		ProductID string `json:"productId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "invalid json",
		})
		return
	}

	productID := strings.TrimSpace(body.ProductID)
	if productID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "productId is required",
		})
		return
	}

	// ✅ avatarId is resolved and stored by AvatarContextMiddleware (required)
	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || strings.TrimSpace(avatarID) == "" {
		// This should not happen if requireAvatarContext is wired.
		// Treat as service misconfiguration to make the bug obvious.
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": "avatar_context_missing",
		})
		return
	}
	avatarID = strings.TrimSpace(avatarID)

	log.Printf(
		`[mall.order.scan.verify] incoming uid=%q avatarId=%q productId=%q`,
		maskUID(uid),
		avatarID,
		productID,
	)

	out, err := h.q.VerifyScanPurchasedByAvatarID(r.Context(), avatarID, productID)
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

	// Ensure ids are set (best-effort)
	out.AvatarID = strings.TrimSpace(out.AvatarID)
	if out.AvatarID == "" {
		out.AvatarID = avatarID
	}
	out.ProductID = strings.TrimSpace(out.ProductID)
	if out.ProductID == "" {
		out.ProductID = productID
	}

	log.Printf(
		`[mall.order.scan.verify] ok uid=%q avatarId=%q productId=%q matched=%t purchasedPairs=%d`,
		maskUID(uid),
		out.AvatarID,
		out.ProductID,
		out.Matched,
		len(out.PurchasedPairs),
	)

	writeJSON(w, http.StatusOK, map[string]any{
		"data": out,
	})
}
