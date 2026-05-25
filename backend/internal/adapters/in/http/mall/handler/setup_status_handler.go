// backend/internal/adapters/in/http/mall/handler/setup_status_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"narratives/internal/adapters/in/http/middleware"
	"narratives/internal/application/usecase"
)

type SetupStatusUsecase interface {
	GetSetupStatus(ctx context.Context, uid string) (usecase.SetupStatusOutput, error)
}

type SetupStatusHandler struct {
	Usecase SetupStatusUsecase
}

func NewSetupStatusHandler(setupUsecase SetupStatusUsecase) http.Handler {
	return &SetupStatusHandler{
		Usecase: setupUsecase,
	}
}

func (h *SetupStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if h == nil || h.Usecase == nil {
		http.Error(w, "setup-status usecase is not configured", http.StatusInternalServerError)
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || uid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	status, err := h.Usecase.GetSetupStatus(r.Context(), uid)
	if err != nil {
		http.Error(w, "failed to get setup status", http.StatusInternalServerError)
		return
	}

	resp := map[string]any{
		"data": map[string]any{
			"hasAvatar":      status.HasAvatar,
			"setupCompleted": status.SetupCompleted,
			"required": map[string]bool{
				"avatar": status.Required.Avatar,
			},
		},
	}

	log.Printf(
		"[mall_setup_status_handler] uid=%q requiredAvatarOnly=true hasAvatar=%t setupCompleted=%t",
		uid,
		status.HasAvatar,
		status.SetupCompleted,
	)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(resp)
}
