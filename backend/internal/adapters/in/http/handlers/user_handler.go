package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	userdom "narratives/internal/domain/user"
)

// UserHandler は /users 関連のエンドポイントを担当します（単一取得のみ）。
type UserHandler struct {
	uc *usecase.UserUsecase
}

// NewUserHandler はHTTPハンドラを初期化します。
func NewUserHandler(uc *usecase.UserUsecase) http.Handler {
	return &UserHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *UserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/users/"):
		id := strings.TrimPrefix(r.URL.Path, "/users/")
		h.get(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /users/{id}
func (h *UserHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	u, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeUserErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(u)
}

// エラーハンドリング
func writeUserErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case userdom.ErrInvalidID:
		code = http.StatusBadRequest
	case userdom.ErrNotFound:
		code = http.StatusNotFound
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
