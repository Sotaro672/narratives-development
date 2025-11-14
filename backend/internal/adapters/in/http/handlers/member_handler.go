// backend/internal/adapters/in/http/handlers/member_handler.go
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
	case r.Method == http.MethodPost && r.URL.Path == "/members":
		h.create(w, r)
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

// ========================
// POST /members
// ========================

type memberCreateRequest struct {
	ID             string   `json:"id"`
	FirstName      string   `json:"firstName"`
	LastName       string   `json:"lastName"`
	FirstNameKana  string   `json:"firstNameKana"`
	LastNameKana   string   `json:"lastNameKana"`
	Email          string   `json:"email"`
	Permissions    []string `json:"permissions"`
	AssignedBrands []string `json:"assignedBrands"`
}

func (h *MemberHandler) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req memberCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	input := memberuc.CreateMemberInput{
		ID:             req.ID,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		FirstNameKana:  req.FirstNameKana,
		LastNameKana:   req.LastNameKana,
		Email:          req.Email,
		Permissions:    req.Permissions,
		AssignedBrands: req.AssignedBrands,
		CreatedAt:      nil,
	}

	m, err := h.uc.Create(ctx, input)
	if err != nil {
		writeMemberErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(m)
}

// ========================
// GET /members
// ========================

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

// ========================
// GET /members/{id}
// ========================

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

// ========================
// エラーハンドリング
// ========================

func writeMemberErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch err {
	case memberdom.ErrInvalidID:
		code = http.StatusBadRequest
	case memberdom.ErrNotFound:
		code = http.StatusNotFound
	case memberdom.ErrInvalidEmail, memberdom.ErrInvalidCreatedAt:
		code = http.StatusBadRequest
	case memberdom.ErrConflict:
		code = http.StatusConflict
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
