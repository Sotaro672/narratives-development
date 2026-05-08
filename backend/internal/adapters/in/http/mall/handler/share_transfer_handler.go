// backend/internal/adapters/in/http/mall/handler/share_transfer_handler.go
package mallHandler

import (
	"context"
	"errors"
	"log"
	"net/http"

	"narratives/internal/adapters/in/http/middleware"
	"narratives/internal/application/usecase"
)

// ShareTransferUsecase is the dependency for:
// POST /mall/me/contents/share
type ShareTransferUsecase interface {
	ShareToAvatar(ctx context.Context, in usecase.ShareTransferInput) (usecase.ShareTransferResult, error)
}

// ShareTransferResult is the response "data" shape expected by frontend.
type ShareTransferResult struct {
	AvatarID       string `json:"avatarId"`
	TargetAvatarID string `json:"targetAvatarId"`
	ProductID      string `json:"productId"`

	TxSignature string `json:"txSignature,omitempty"`

	FromWallet string `json:"fromWallet,omitempty"`
	ToWallet   string `json:"toWallet,omitempty"`

	UpdatedToAddress bool   `json:"updatedToAddress,omitempty"`
	MintAddress      string `json:"mintAddress,omitempty"`
	TokenBlueprintID string `json:"tokenBlueprintId,omitempty"`
}

// ShareTransferHandler handles:
// POST /mall/me/contents/share
//
// Assumption:
// - AvatarContextMiddleware is enabled for this route
// - sender avatarId is always resolved from request context
// - target avatarId is accepted from request body
type ShareTransferHandler struct {
	uc ShareTransferUsecase
}

func NewShareTransferHandler(uc ShareTransferUsecase) http.Handler {
	return &ShareTransferHandler{uc: uc}
}

func (h *ShareTransferHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	if h == nil || h.uc == nil {
		internalError(w, "share transfer usecase not configured")
		return
	}

	auth := r.Header.Get("Authorization")
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
		ProductID      string `json:"productId"`
		TargetAvatarID string `json:"targetAvatarId"`
	}
	if err := readJSON(r, &body); err != nil {
		badRequest(w, "invalid json")
		return
	}

	productID := body.ProductID
	targetAvatarID := body.TargetAvatarID

	if productID == "" {
		badRequest(w, "productId is required")
		return
	}
	if targetAvatarID == "" {
		badRequest(w, "targetAvatarId is required")
		return
	}

	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": "avatar_context_missing",
		})
		return
	}

	log.Printf(
		`[mall.contents.share] incoming uid=%q avatarId=%q targetAvatarId=%q productId=%q`,
		uid,
		avatarID,
		targetAvatarID,
		productID,
	)

	ucOut, err := h.uc.ShareToAvatar(
		r.Context(),
		usecase.ShareTransferInput{
			FromAvatarID: avatarID,
			ToAvatarID:   targetAvatarID,
			ProductID:    productID,
		},
	)
	if err != nil {
		log.Printf(
			"[mall.contents.share] ERROR uid=%q avatarId=%q targetAvatarId=%q productId=%q err=%T %v",
			uid,
			avatarID,
			targetAvatarID,
			productID,
			err, err,
		)

		switch {
		case errors.Is(err, usecase.ErrShareTransferProductIDEmpty):
			badRequest(w, "productId is required")
			return
		case errors.Is(err, usecase.ErrShareTransferToAvatarEmpty):
			badRequest(w, "targetAvatarId is required")
			return
		case errors.Is(err, usecase.ErrShareTransferFromAvatarEmpty):
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"error": "avatar_context_missing",
			})
			return
		case errors.Is(err, usecase.ErrShareTransferSameAvatar):
			badRequest(w, "targetAvatarId must be different from avatarId")
			return
		case isNotFoundLike(err):
			writeJSON(w, http.StatusNotFound, map[string]any{
				"error":          "not found",
				"avatarId":       avatarID,
				"targetAvatarId": targetAvatarID,
				"productId":      productID,
			})
			return
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			writeJSON(w, http.StatusRequestTimeout, map[string]any{
				"error":          "request canceled",
				"avatarId":       avatarID,
				"targetAvatarId": targetAvatarID,
				"productId":      productID,
			})
			return
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"error":          "share transfer failed",
				"avatarId":       avatarID,
				"targetAvatarId": targetAvatarID,
				"productId":      productID,
			})
			return
		}
	}

	out := &ShareTransferResult{
		AvatarID:         ucOut.FromAvatarID,
		TargetAvatarID:   ucOut.ToAvatarID,
		ProductID:        ucOut.ProductID,
		TxSignature:      ucOut.TxSignature,
		FromWallet:       ucOut.FromWallet,
		ToWallet:         ucOut.ToWallet,
		UpdatedToAddress: true,
		MintAddress:      ucOut.MintAddress,
		TokenBlueprintID: ucOut.TokenBlueprintID,
	}

	log.Printf(
		`[mall.contents.share] ok uid=%q avatarId=%q targetAvatarId=%q productId=%q tx=%q updatedToAddress=%t mint=%q tokenBlueprintId=%q`,
		uid,
		out.AvatarID,
		out.TargetAvatarID,
		out.ProductID,
		out.TxSignature,
		out.UpdatedToAddress,
		out.MintAddress,
		out.TokenBlueprintID,
	)

	writeJSON(w, http.StatusOK, map[string]any{
		"data": out,
	})
}
