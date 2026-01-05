// backend\internal\adapters\in\http\console\handler\announcement_handler.go
package consoleHandler

import (
	"encoding/json"
	"net/http"
	"strings"

	uc "narratives/internal/application/usecase"
	ann "narratives/internal/domain/announcement"
	aa "narratives/internal/domain/announcementAttachment"
)

// AnnouncementHandler は /announcements 関連のエンドポイントを担当します。
type AnnouncementHandler struct {
	uc *uc.AnnouncementUsecase
}

// NewAnnouncementHandler はHTTPハンドラを初期化します。
func NewAnnouncementHandler(announcementUC *uc.AnnouncementUsecase) http.Handler {
	return &AnnouncementHandler{uc: announcementUC}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *AnnouncementHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	path := r.URL.Path

	switch {
	// DELETE /announcements/{id} (cascade)
	case r.Method == http.MethodDelete && strings.HasPrefix(path, "/announcements/"):
		id := strings.TrimPrefix(path, "/announcements/")
		h.deleteAnnouncementCascade(w, r, strings.Trim(id, "/"))
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// DELETE /announcements/{id} (cascade)
func (h *AnnouncementHandler) deleteAnnouncementCascade(w http.ResponseWriter, r *http.Request, announcementID string) {
	if h.uc == nil {
		http.Error(w, `{"error":"not_configured"}`, http.StatusInternalServerError)
		return
	}
	announcementID = strings.TrimSpace(announcementID)
	if announcementID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	if err := h.uc.DeleteAnnouncementCascade(r.Context(), announcementID); err != nil {
		writeAnnouncementErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- helpers ---
func writeAnnouncementErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case ann.ErrNotFound, aa.ErrNotFound:
		code = http.StatusNotFound
	case aa.ErrInvalidAnnouncementID:
		code = http.StatusBadRequest
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
