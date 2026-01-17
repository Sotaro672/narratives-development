// backend/internal/adapters/in/http/mall/handler/transfer_handler.go
package mallHandler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
)

// ScanTransferUsecase is the dependency for:
// POST /mall/me/orders/scan/transfer
//
// 実装は usecase.TransferUsecase 等に接続してください。
// ここでは handler 側が必要とする最小の形だけを定義します。
type ScanTransferUsecase interface {
	TransferByScanPurchasedByAvatarID(ctx context.Context, avatarID, productID string) (*ScanTransferResult, error)
}

// ScanTransferResult is the response "data" shape expected by frontend.
type ScanTransferResult struct {
	AvatarID  string `json:"avatarId"`
	ProductID string `json:"productId"`

	Matched bool `json:"matched"`

	// Transfer result
	TxSignature string `json:"txSignature,omitempty"`

	// Optional debug/info
	FromWallet string `json:"fromWallet,omitempty"`
	ToWallet   string `json:"toWallet,omitempty"`

	// tokens/{productId}.toAddress updated?
	UpdatedToAddress bool `json:"updatedToAddress,omitempty"`

	// ✅ NEW: moved mintAddress
	MintAddress string `json:"mintAddress,omitempty"`
}

// TransferHandler handles:
// POST /mall/me/orders/scan/transfer
type TransferHandler struct {
	uc       ScanTransferUsecase
	resolver middleware.AvatarIDResolver // optional anti-spoof: uid -> avatarId
}

// NewTransferHandler creates handler without uid->avatarId validation.
func NewTransferHandler(uc ScanTransferUsecase) http.Handler {
	return &TransferHandler{uc: uc, resolver: nil}
}

// NewTransferHandlerWithResolver creates handler with uid->avatarId validation.
func NewTransferHandlerWithResolver(uc ScanTransferUsecase, r middleware.AvatarIDResolver) http.Handler {
	return &TransferHandler{uc: uc, resolver: r}
}

func (h *TransferHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Preflight
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	if h == nil || h.uc == nil {
		internalError(w, "transfer usecase not configured")
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
	if err := readJSON(r, &body); err != nil {
		badRequest(w, "invalid json")
		return
	}

	bodyAvatarID := strings.TrimSpace(body.AvatarID)
	productID := strings.TrimSpace(body.ProductID)
	if productID == "" {
		badRequest(w, "productId is required")
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
		badRequest(w, "avatarId is required")
		return
	}

	// Optional anti-spoof: validate uid -> avatarId
	if h.resolver != nil {
		resolved, err := h.resolver.ResolveAvatarIDByUID(r.Context(), uid)
		if err != nil {
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
		`[mall.order.scan.transfer] incoming uid=%q avatarId=%q productId=%q`,
		maskUID(uid),
		maskUID(avatarID),
		maskUID(productID),
	)

	out, err := h.uc.TransferByScanPurchasedByAvatarID(r.Context(), avatarID, productID)
	if err != nil {
		// ✅ Added: log concrete error type/value to pinpoint failing step
		log.Printf(
			"[mall.order.scan.transfer] ERROR uid=%q avatarId=%q productId=%q err=%T %v",
			maskUID(uid),
			maskUID(avatarID),
			maskUID(productID),
			err, err,
		)

		if isNotFoundLike(err) {
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
			"error":     "transfer failed",
			"avatarId":  avatarID,
			"productId": productID,
		})
		return
	}

	if out == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":     "transfer failed (nil result)",
			"avatarId":  avatarID,
			"productId": productID,
		})
		return
	}

	// Ensure ids are set
	out.AvatarID = strings.TrimSpace(out.AvatarID)
	if out.AvatarID == "" {
		out.AvatarID = avatarID
	}
	out.ProductID = strings.TrimSpace(out.ProductID)
	if out.ProductID == "" {
		out.ProductID = productID
	}

	// normalize optional fields
	out.TxSignature = strings.TrimSpace(out.TxSignature)
	out.FromWallet = strings.TrimSpace(out.FromWallet)
	out.ToWallet = strings.TrimSpace(out.ToWallet)
	out.MintAddress = strings.TrimSpace(out.MintAddress)

	log.Printf(
		`[mall.order.scan.transfer] ok uid=%q avatarId=%q productId=%q matched=%t tx=%q updatedToAddress=%t mint=%q`,
		maskUID(uid),
		maskUID(out.AvatarID),
		maskUID(out.ProductID),
		out.Matched,
		maskUID(out.TxSignature),
		out.UpdatedToAddress,
		maskUID(out.MintAddress),
	)

	writeJSON(w, http.StatusOK, map[string]any{
		"data": out,
	})
}
