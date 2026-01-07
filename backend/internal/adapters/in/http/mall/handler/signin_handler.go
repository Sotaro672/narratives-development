// backend\internal\adapters\in\http\mall\handler\signin_handler.go
package mallHandler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	userdom "narratives/internal/domain/user"
)

// SignInHandler is buyer-facing onboarding entry.
// - POST /mall/sign-in
// - Requires BootstrapAuthMiddleware (Firebase token verified)
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

	uid := getUIDFromContext(ctx)
	if strings.TrimSpace(uid) == "" {
		// bootstrap mw が付いていない/失敗している
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

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

// NOTE:
// BootstrapAuthMiddleware が context に uid を載せるキーに合わせてここを調整してください。
// 例: ctx.Value("uid") / ctx.Value(middleware.ContextKeyUID) など。
func getUIDFromContext(ctx any) string {
	// 一旦「文字列キー "uid"」で読む実装（最小依存）。
	// あなたの middleware が別キーならここだけ直せばOK。
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
