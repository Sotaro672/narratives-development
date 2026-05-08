// backend\internal\adapters\in\http\mall\handler\setup_status_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"narratives/internal/adapters/in/http/middleware"
)

type SetupStatusRepo interface {
	HasAvatar(ctx context.Context, uid string) (bool, error)
}

type SetupStatusHandler struct {
	Repo SetupStatusRepo
}

func NewSetupStatusHandler(repo SetupStatusRepo) http.Handler {
	return &SetupStatusHandler{Repo: repo}
}

func (h *SetupStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if h == nil || h.Repo == nil {
		http.Error(w, "setup-status repo is not configured", http.StatusInternalServerError)
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || uid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	hasAvatar, err := h.Repo.HasAvatar(r.Context(), uid)
	if err != nil {
		http.Error(w, "failed to check avatar", http.StatusInternalServerError)
		return
	}

	resp := map[string]any{
		"data": map[string]any{
			"hasAvatar":      hasAvatar,
			"setupCompleted": hasAvatar,
			"required": map[string]bool{
				"avatar": true,
			},
		},
	}

	log.Printf(
		"[mall_setup_status_handler] uid=%q requiredAvatarOnly=true hasAvatar=%t setupCompleted=%t",
		uid,
		hasAvatar,
		hasAvatar,
	)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(resp)
}
