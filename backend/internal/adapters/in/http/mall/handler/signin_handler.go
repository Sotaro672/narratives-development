// backend/internal/adapters/in/http/mall/handler/signin_handler.go
package mallHandler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	userdom "narratives/internal/domain/user"

	"narratives/internal/adapters/in/http/middleware"
)

// SignInHandler is buyer-facing onboarding entry.
// - POST /mall/sign-in
// - Requires auth middleware (BootstrapAuthMiddleware or UserAuthMiddleware) to verify Firebase token
// - Ensures user exists (id = uid), returns user.
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

	// ✅ mall のみ受付（/mall/sign-in のみ。末尾スラッシュは吸収）
	path := strings.TrimSuffix(strings.TrimSpace(r.URL.Path), "/")
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

	ctx := r.Context()

	// ✅ middleware が積んだ uid を取得（BootstrapAuthMiddleware / UserAuthMiddleware 両対応）
	uid, ok := middleware.CurrentUserUID(r)
	if !ok || strings.TrimSpace(uid) == "" {
		// auth mw が付いていない/失敗している
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}
	uid = strings.TrimSpace(uid)

	// 既存ユーザーなら返す
	u, err := h.uc.GetByID(ctx, uid)
	if err == nil {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"created": false,
			"user":    u,
		})
		return
	}

	// 無ければ作る（name 系は nil でOK。id は uid を使う）
	now := time.Now().UTC()
	v, derr := userdom.New(
		uid,
		nil, nil, nil, nil,
		now, now, now,
	)
	if derr != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": derr.Error()})
		return
	}

	created, cerr := h.uc.Create(ctx, v)
	if cerr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": cerr.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"created": true,
		"user":    created,
	})
}
