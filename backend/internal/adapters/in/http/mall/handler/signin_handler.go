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
//
// NOTE:
// - 旧互換(CreateFromEntity/Save)は廃止したので、ここは uc.Create を使う。
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
		// bootstrap mw が付いていない/失敗している
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	// 既存ユーザーなら返す
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

	// 無ければ作る（name 系は nil でOK。id は uid を使う）
	in := userdom.CreateUserInput{
		FirstName:     nil,
		FirstNameKana: nil,
		LastNameKana:  nil,
		LastName:      nil,
		// createdAt/updatedAt は usecase が server now を差し込む
		// deletedAt は nil(未指定) = not deleted
	}

	created, cerr := h.uc.Create(ctx, uid, in)
	if cerr != nil {
		// 競合（並行で作成された）なら取り直して返す
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

// NOTE:
// BootstrapAuthMiddleware が context に uid を載せるキーに合わせてここを調整してください。
// 例: ctx.Value("uid") / ctx.Value(middleware.ContextKeyUID) など。
func getUIDFromContext(ctx any) string {
	// "ctx any" に対して nil チェックを入れると static analyzer が
	// 「non-nil != nil」扱いにしやすいので、nil 判定はしない。
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
