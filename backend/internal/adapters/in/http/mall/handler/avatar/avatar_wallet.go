// backend/internal/adapters/in/http/mall/handler/avatar/avatar_wallet.go
package avatarHandler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

func (h *AvatarHandler) openWallet(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	log.Printf("[mall_avatar_handler] POST /mall/avatars/%s/wallet request\n", id)

	a, err := h.uc.GetByID(ctx, id)
	if err != nil {
		log.Printf("[mall_avatar_handler] POST /mall/avatars/%s/wallet get error=%v\n", id, err)
		writeAvatarErr(w, err)
		return
	}
	if a.WalletAddress != nil && strings.TrimSpace(*a.WalletAddress) != "" {
		log.Printf("[mall_avatar_handler] POST /mall/avatars/%s/wallet conflict walletAddress=%q\n", id, ptrStr(a.WalletAddress))
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "wallet already opened"})
		return
	}

	updated, err := h.uc.OpenWallet(ctx, id)
	if err != nil {
		log.Printf("[mall_avatar_handler] POST /mall/avatars/%s/wallet error=%v\n", id, err)
		writeAvatarErr(w, err)
		return
	}

	log.Printf("[mall_avatar_handler] POST /mall/avatars/%s/wallet ok walletAddress=%q\n", id, ptrStr(updated.WalletAddress))
	_ = json.NewEncoder(w).Encode(toAvatarResponse(updated))
}
