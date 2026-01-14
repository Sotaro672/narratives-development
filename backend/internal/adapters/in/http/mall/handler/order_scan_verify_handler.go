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

	// uid / avatarId を取る
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
type OrderScanVerifyHandler struct {
	q        ScanVerifyQuery
	resolver middleware.AvatarIDResolver // optional: uid -> avatarId (anti-spoof)
}

// NewOrderScanVerifyHandler creates handler without uid->avatarId validation.
// (Use WithResolver version if you want to prevent spoofing strictly.)
func NewOrderScanVerifyHandler(q ScanVerifyQuery) http.Handler {
	return &OrderScanVerifyHandler{q: q, resolver: nil}
}

// NewOrderScanVerifyHandlerWithResolver creates handler with uid->avatarId validation.
// Pass cont.OrderQ if it implements middleware.AvatarIDResolver.
func NewOrderScanVerifyHandlerWithResolver(q ScanVerifyQuery, r middleware.AvatarIDResolver) http.Handler {
	return &OrderScanVerifyHandler{q: q, resolver: r}
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

	var body struct {
		AvatarID  string `json:"avatarId"`
		ProductID string `json:"productId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "invalid json",
		})
		return
	}

	bodyAvatarID := strings.TrimSpace(body.AvatarID)
	productID := strings.TrimSpace(body.ProductID)
	if productID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "productId is required",
		})
		return
	}

	// Prefer avatarId from context if AvatarContextMiddleware is enabled.
	ctxAvatarID, _ := middleware.CurrentAvatarID(r)

	avatarID := ""
	switch {
	case strings.TrimSpace(ctxAvatarID) != "" && bodyAvatarID != "":
		// both present -> must match
		if strings.TrimSpace(ctxAvatarID) != bodyAvatarID {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"error":         "avatarId mismatch (context vs body)",
				"contextAvatar": strings.TrimSpace(ctxAvatarID),
				"bodyAvatar":    bodyAvatarID,
			})
			return
		}
		avatarID = strings.TrimSpace(ctxAvatarID)

	case strings.TrimSpace(ctxAvatarID) != "":
		avatarID = strings.TrimSpace(ctxAvatarID)

	default:
		avatarID = bodyAvatarID
	}

	if avatarID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "avatarId is required",
		})
		return
	}

	// Optional anti-spoof: validate uid -> avatarId
	if h.resolver != nil {
		resolved, err := h.resolver.ResolveAvatarIDByUID(r.Context(), uid)
		if err != nil {
			// uid に avatar が無い/解決不能
			writeJSON(w, http.StatusNotFound, map[string]any{
				"error": "avatar_not_found_for_uid",
			})
			return
		}
		resolved = strings.TrimSpace(resolved)
		if resolved == "" || resolved != avatarID {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"error":    "avatarId does not belong to current uid",
				"avatarId": avatarID,
			})
			return
		}
	}

	log.Printf(
		`[mall.order.scan.verify] incoming uid=%q avatarId=%q productId=%q`,
		maskUID(uid),
		avatarID,
		productID,
	)

	// ✅ A案: value 戻り値
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
