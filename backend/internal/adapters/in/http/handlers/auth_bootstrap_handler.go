package handlers

import (
	"encoding/json"
	"net/http"

	httpmw "narratives/internal/adapters/in/http/middleware"
	authuc "narratives/internal/application/usecase/auth"
)

type AuthBootstrapHandler struct {
	uc *authuc.BootstrapService
}

func NewAuthBootstrapHandler(uc *authuc.BootstrapService) http.Handler {
	return &AuthBootstrapHandler{uc: uc}
}

func (h *AuthBootstrapHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// only POST allowed
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// must be authenticated, but CurrentMemberは不要
	uid, email, ok := httpmw.CurrentUIDAndEmail(r)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	var profile authuc.SignUpProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	if err := h.uc.Bootstrap(r.Context(), uid, email, &profile); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"uid":    uid,
	})
}
