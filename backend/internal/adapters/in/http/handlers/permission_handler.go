package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	permissiondom "narratives/internal/domain/permission"
)

// PermissionHandler は /permissions 関連のエンドポイントを担当します（単一取得のみ）。
type PermissionHandler struct {
	uc *usecase.PermissionUsecase
}

// NewPermissionHandler はHTTPハンドラを初期化します。
func NewPermissionHandler(uc *usecase.PermissionUsecase) http.Handler {
	return &PermissionHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *PermissionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/permissions/"):
		id := strings.TrimPrefix(r.URL.Path, "/permissions/")
		h.get(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /permissions/{id}
func (h *PermissionHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	perm, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writePermissionErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(perm)
}

// エラーハンドリング
func writePermissionErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case permissiondom.ErrInvalidID:
		code = http.StatusBadRequest
	case permissiondom.ErrNotFound:
		code = http.StatusNotFound
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
