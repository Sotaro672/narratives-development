// backend/internal/adapters/in/http/mall/handler/signin_handler.go
package mallHandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	userdom "narratives/internal/domain/user"
)

// SignInHandler is buyer-facing onboarding entry.
// - POST /mall/sign-in
// - Requires BootstrapAuthMiddleware (Firebase token verified)
// - Ensures user exists (docId = uid), returns user.
type SignInHandler struct {
	uc *usecase.UserUsecase
}

func NewSignInHandler(uc *usecase.UserUsecase) http.Handler {
	return &SignInHandler{uc: uc}
}

func (h *SignInHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")
	if path != "/mall/sign-in" {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "method_not_allowed"})
		return
	}

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "user_usecase_not_initialized"})
		return
	}

	ctx := r.Context()

	uid := getUIDFromContext(ctx)
	if uid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	u, err := h.uc.GetByID(ctx, uid)
	if err == nil {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"created": false,
			"user":    u,
		})
		return
	}

	in := userdom.CreateUserInput{
		FirstName:     nil,
		FirstNameKana: nil,
		LastNameKana:  nil,
		LastName:      nil,
	}

	created, cerr := h.uc.Create(ctx, uid, in)
	if cerr != nil {
		if errors.Is(cerr, userdom.ErrConflict) {
			got, gerr := h.uc.GetByID(ctx, uid)
			if gerr != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": gerr.Error()})
				return
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":  "ok",
				"created": false,
				"user":    got,
			})
			return
		}

		if errors.Is(cerr, userdom.ErrInvalidID) ||
			errors.Is(cerr, userdom.ErrInvalidFirstName) ||
			errors.Is(cerr, userdom.ErrInvalidFirstNameKana) ||
			errors.Is(cerr, userdom.ErrInvalidLastNameKana) ||
			errors.Is(cerr, userdom.ErrInvalidLastName) ||
			errors.Is(cerr, userdom.ErrInvalidCreatedAt) ||
			errors.Is(cerr, userdom.ErrInvalidUpdatedAt) ||
			errors.Is(cerr, userdom.ErrInvalidDeletedAt) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": cerr.Error()})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": cerr.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"created": true,
		"user":    created,
	})
}

func getUIDFromContext(ctx any) string {
	type vctx interface{ Value(any) any }
	c, ok := ctx.(vctx)
	if !ok {
		return ""
	}
	if v := c.Value("uid"); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
