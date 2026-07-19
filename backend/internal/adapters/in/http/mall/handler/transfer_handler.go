// backend/internal/adapters/in/http/mall/handler/transfer_handler.go
package mallHandler

import (
	"context"
	"errors"
	"log"
	"net/http"

	"narratives/internal/adapters/in/http/middleware"
	"narratives/internal/application/usecase"
)

// ScanTransferUsecase is the dependency for:
// POST /mall/me/orders/scan/transfer
//
// 螳溯｣・・ usecase.TransferUsecase 遲峨↓謗･邯壹＠縺ｦ縺上□縺輔＞縲・
// 縺薙％縺ｧ縺ｯ handler 蛛ｴ縺悟ｿ・ｦ√→縺吶ｋ譛蟆上・蠖｢縺縺代ｒ螳夂ｾｩ縺励∪縺吶・
type ScanTransferUsecase interface {
	TransferToAvatarByVerifiedScan(
		ctx context.Context,
		in usecase.TransferByVerifiedScanInput,
	) (usecase.TransferByVerifiedScanResult, error)
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
// Option A: anti-spoof 繧・AvatarContextMiddleware 縺ｫ荳譛ｬ蛹悶☆繧九・
// - handler 縺ｯ uid->avatarId resolver 繧呈戟縺溘↑縺・
// - avatarId 縺ｯ request context 縺九ｉ縺ｮ縺ｿ蜿門ｾ励☆繧具ｼ・ody 縺ｮ avatarId 縺ｯ菴ｿ繧上↑縺・ｼ・
type TransferHandler struct {
	uc ScanTransferUsecase
}

// NewTransferHandler creates handler.
// NOTE: This handler assumes AvatarContextMiddleware is enabled for this route.
func NewTransferHandler(uc ScanTransferUsecase) http.Handler {
	return &TransferHandler{uc: uc}
}

func (h *TransferHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
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
		// NotMatched 縺ｯ 200 + matched=false 繧定ｿ斐☆・亥ｾ捺擂縺ｮ繧｢繝繝励ち謖吝虚繧堤ｶｭ謖・ｼ・
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

		if errors.Is(err, context.Canceled) ||
			errors.Is(err, context.DeadlineExceeded) {
			writeJSON(w, http.StatusRequestTimeout, map[string]any{
				"error":     "request canceled",
				"avatarId":  avatarID,
				"productId": productID,
			})
			return
		}

		log.Printf(
			"[mall/order-scan-transfer] failed avatarId=%s productId=%s err=%v",
			avatarID,
			productID,
			err,
		)

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
		UpdatedToAddress: true, // TransferUsecase 蜀・〒 UpdateToAddressByProductID 繧貞ｮ溯｡梧ｸ医∩・・ail-fast・・
		MintAddress:      ucOut.MintAddress,
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": out,
	})
}
