// backend/internal/adapters/in/http/mall/handler/setup_status_handler.go
package mallHandler

import (
	"context"
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
		methodNotAllowed(w)
		return
	}

	if h == nil || h.Usecase == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "setup-status usecase is not configured",
		})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || uid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	status, err := h.Usecase.GetSetupStatus(r.Context(), uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to get setup status",
		})
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

	writeJSON(w, http.StatusOK, resp)
}
