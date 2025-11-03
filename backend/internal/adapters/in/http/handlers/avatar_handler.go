package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	uc "narratives/internal/application/usecase"
	avatardom "narratives/internal/domain/avatar"
)

// AvatarHandler は /avatars 関連のエンドポイントを担当します。
// 新しい usecase.AvatarUsecase を利用します。
type AvatarHandler struct {
	uc *uc.AvatarUsecase
}

// NewAvatarHandler はHTTPハンドラを初期化します。
func NewAvatarHandler(avatarUC *uc.AvatarUsecase) http.Handler {
	return &AvatarHandler{uc: avatarUC}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *AvatarHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/avatars":
		// 現行の AvatarUsecase は一覧取得を提供しないため 501 で返す
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/avatars/"):
		id := strings.TrimPrefix(r.URL.Path, "/avatars/")
		h.get(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /avatars/{id}
// aggregate=1|true を付けると Avatar + State + Icons の集約を返します。
func (h *AvatarHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	q := r.URL.Query()
	agg := strings.EqualFold(q.Get("aggregate"), "1") || strings.EqualFold(q.Get("aggregate"), "true")

	if agg {
		data, err := h.uc.GetAggregate(ctx, id)
		if err != nil {
			writeAvatarErr(w, err)
			return
		}
		_ = json.NewEncoder(w).Encode(data)
		return
	}

	avatar, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeAvatarErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(avatar)
}

// エラーハンドリング
func writeAvatarErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	// Only treat invalid id specially (avoid referencing ErrNotFound that may not exist)
	if err == avatardom.ErrInvalidID {
		code = http.StatusBadRequest
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
