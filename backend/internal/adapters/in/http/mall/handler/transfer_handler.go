// backend/internal/adapters/in/http/mall/handler/transfer_handler.go
package mallHandler

import (
	"context"
	"errors"
	"net/http"

	"narratives/internal/adapters/in/http/middleware"
	"narratives/internal/application/usecase"
)

// ScanTransferUsecase is the dependency for:
// POST /mall/me/orders/scan/transfer
//
// 実装は usecase.TransferUsecase 等に接続してください。
// ここでは handler 側が必要とする最小の形だけを定義します。
type ScanTransferUsecase interface {
	TransferToAvatarByVerifiedScan(ctx context.Context, in usecase.TransferByVerifiedScanInput) (usecase.TransferByVerifiedScanResult, error)
}

// ScanTransferResult is the response "data" shape expected by frontend.
type ScanTransferResult struct {
	AvatarID  string `json:"avatarId"`
	ProductID string `json:"productId"`

	Matched bool `json:"matched"`

	// Transfer result
	TxSignature string `json:"txSignature,omitempty"`

	// Display names resolved by backend.
	// Frontend modal should use these instead of wallet address fallback.
	FromDisplayName string `json:"fromDisplayName,omitempty"`
	ToDisplayName   string `json:"toDisplayName,omitempty"`

	// tokens/{productId}.toAddress updated?
	UpdatedToAddress bool `json:"updatedToAddress,omitempty"`

	// moved mintAddress
	MintAddress string `json:"mintAddress,omitempty"`
}

// TransferHandler handles:
// POST /mall/me/orders/scan/transfer
//
// Option A: anti-spoof を AvatarContextMiddleware に一本化する。
// - handler は uid->avatarId resolver を持たない
// - avatarId は request context からのみ取得する（body の avatarId は使わない）
type TransferHandler struct {
	uc ScanTransferUsecase
}

// NewTransferHandler creates handler.
// NOTE: This handler assumes AvatarContextMiddleware is enabled for this route.
func NewTransferHandler(uc ScanTransferUsecase) http.Handler {
	return &TransferHandler{uc: uc}
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
	auth := r.Header.Get("Authorization")
	if auth == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": "authorization header is required",
		})
		return
	}

	_, ok := middleware.CurrentUserUID(r)
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
	if err := readJSON(r, &body); err != nil {
		badRequest(w, "invalid json")
		return
	}

	productID := body.ProductID
	if productID == "" {
		badRequest(w, "productId is required")
		return
	}

	// avatarId is resolved and stored by AvatarContextMiddleware (required)
	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		// This should not happen if requireAvatarContext is wired.
		// Treat as service misconfiguration to make the bug obvious.
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": "avatar_context_missing",
		})
		return
	}

	ucOut, err := h.uc.TransferToAvatarByVerifiedScan(
		r.Context(),
		usecase.TransferByVerifiedScanInput{
			AvatarID:  avatarID,
			ProductID: productID,
		},
	)
	if err != nil {
		// NotMatched は 200 + matched=false を返す（従来のアダプタ挙動を維持）
		if errors.Is(err, usecase.ErrTransferNotMatched) {
			out := &ScanTransferResult{
				AvatarID:  avatarID,
				ProductID: productID,
				Matched:   false,
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"data": out,
			})
			return
		}

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

	out := &ScanTransferResult{
		AvatarID:         avatarID,
		ProductID:        productID,
		Matched:          true,
		TxSignature:      ucOut.TxSignature,
		FromDisplayName:  ucOut.FromDisplayName,
		ToDisplayName:    ucOut.ToDisplayName,
		UpdatedToAddress: true, // TransferUsecase 内で UpdateToAddressByProductID を実行済み（fail-fast）
		MintAddress:      ucOut.MintAddress,
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": out,
	})
}
