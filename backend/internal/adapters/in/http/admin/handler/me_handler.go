// backend/internal/adapters/in/http/admin/handler/me_handler.go
package handler

import (
	"encoding/json"
	"net/http"

	"narratives/internal/adapters/in/http/middleware"
)

type MeResponse struct {
	UID   string `json:"uid"`
	Email string `json:"email"`
}

func NewMeHandler() http.Handler {
	return http.HandlerFunc(handleMe)
}

func handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}

	uid, email, ok := middleware.CurrentAdminUIDAndEmail(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "admin_identity_not_found")
		return
	}

	writeJSON(w, http.StatusOK, MeResponse{
		UID:   uid,
		Email: email,
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		return
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}
