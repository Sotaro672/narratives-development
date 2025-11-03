package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	memberuc "narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	memberdom "narratives/internal/domain/member"
)

// MemberHandler は /members 関連のエンドポイントを担当します。
type MemberHandler struct {
	uc *memberuc.MemberUsecase
}

// NewMemberHandler はHTTPハンドラを初期化します。
func NewMemberHandler(uc *memberuc.MemberUsecase) http.Handler {
	return &MemberHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *MemberHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/members":
		h.list(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/members/"):
		id := strings.TrimPrefix(r.URL.Path, "/members/")
		h.get(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /members
func (h *MemberHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// フィルタ（必要ならクエリから詰める）
	var f memberdom.Filter
	var sort common.Sort
	var page common.Page

	res, err := h.uc.List(ctx, f, sort, page)
	if err != nil {
		writeMemberErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(res.Items)
}

// GET /members/{id}
func (h *MemberHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	member, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeMemberErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(member)
}

// エラーハンドリング
func writeMemberErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch err {
	case memberdom.ErrInvalidID:
		code = http.StatusBadRequest
	case memberdom.ErrNotFound:
		code = http.StatusNotFound
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
