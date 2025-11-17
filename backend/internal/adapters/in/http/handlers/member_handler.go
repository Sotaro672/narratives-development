package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	httpmw "narratives/internal/adapters/in/http/middleware"
	memberuc "narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	memberdom "narratives/internal/domain/member"
)

// -----------------------------------------------------------------------------
// MemberHandler
// -----------------------------------------------------------------------------
type MemberHandler struct {
	uc            *memberuc.MemberUsecase
	invitationCmd memberuc.InvitationCommandPort // ★ 招待メール用
}

// NewMemberHandler — メンバーハンドラ
func NewMemberHandler(
	uc *memberuc.MemberUsecase,
	invCmd memberuc.InvitationCommandPort, // ★ InvitationCommand を追加
) http.Handler {
	return &MemberHandler{
		uc:            uc,
		invitationCmd: invCmd,
	}
}

// -----------------------------------------------------------------------------
// ServeHTTP（ルーティング分岐）
// -----------------------------------------------------------------------------
func (h *MemberHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimRight(r.URL.Path, "/")
	if path == "" {
		path = "/"
	}

	// -------------------------------------------------------------------------
	// ★ Invitation: POST /members/{id}/invitation
	// -------------------------------------------------------------------------
	if r.Method == http.MethodPost &&
		strings.HasPrefix(path, "/members/") &&
		strings.HasSuffix(path, "/invitation") {

		h.sendInvitation(w, r)
		return
	}

	// -------------------------------------------------------------------------
	// 通常のメンバー CRUD ルーティング
	// -------------------------------------------------------------------------
	switch {

	case r.Method == http.MethodPost && path == "/members":
		h.create(w, r)

	case r.Method == http.MethodGet && path == "/members":
		h.list(w, r)

	case r.Method == http.MethodPatch && strings.HasPrefix(path, "/members/"):
		id := strings.TrimPrefix(path, "/members/")
		h.update(w, r, id)

	case r.Method == http.MethodGet && strings.HasPrefix(path, "/members/"):
		id := strings.TrimPrefix(path, "/members/")
		h.get(w, r, id)

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// -----------------------------------------------------------------------------
// POST /members/{id}/invitation
// -----------------------------------------------------------------------------
func (h *MemberHandler) sendInvitation(w http.ResponseWriter, r *http.Request) {
	if h.invitationCmd == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invitation command not configured"})
		return
	}

	// 例: /members/abc123/invitation
	path := strings.TrimPrefix(r.URL.Path, "/members/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 || parts[1] != "invitation" {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	memberID := strings.TrimSpace(parts[0])
	if memberID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_member_id"})
		return
	}

	token, err := h.invitationCmd.CreateInvitationAndSend(r.Context(), memberID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "cannot_send_invitation"})
		return
	}

	resp := map[string]string{
		"memberId": memberID,
		"token":    token,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// -----------------------------------------------------------------------------
// POST /members — Create
// -----------------------------------------------------------------------------
type memberCreateRequest struct {
	ID             string   `json:"id"`
	FirstName      string   `json:"firstName"`
	LastName       string   `json:"lastName"`
	FirstNameKana  string   `json:"firstNameKana"`
	LastNameKana   string   `json:"lastNameKana"`
	Email          string   `json:"email"`
	Permissions    []string `json:"permissions"`
	AssignedBrands []string `json:"assignedBrands"`
	Status         string   `json:"status,omitempty"`
}

func (h *MemberHandler) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req memberCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	// AuthMiddleware から companyId 取得
	me, ok := httpmw.CurrentMember(r)
	if !ok || strings.TrimSpace(me.CompanyID) == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
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
		CompanyID:      strings.TrimSpace(me.CompanyID),
		Status:         req.Status,
	}

	m, err := h.uc.Create(ctx, input)
	if err != nil {
		writeMemberErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(m)
}

// -----------------------------------------------------------------------------
// PATCH /members/{id}
// -----------------------------------------------------------------------------
type memberUpdateRequest struct {
	FirstName      *string   `json:"firstName,omitempty"`
	LastName       *string   `json:"lastName,omitempty"`
	FirstNameKana  *string   `json:"firstNameKana,omitempty"`
	LastNameKana   *string   `json:"lastNameKana,omitempty"`
	Email          *string   `json:"email,omitempty"`
	Permissions    *[]string `json:"permissions,omitempty"`
	AssignedBrands *[]string `json:"assignedBrands,omitempty"`
	Status         *string   `json:"status,omitempty"`
}

func (h *MemberHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	me, ok := httpmw.CurrentMember(r)
	if !ok || strings.TrimSpace(me.CompanyID) == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	var req memberUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	input := memberuc.UpdateMemberInput{
		ID:             id,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		FirstNameKana:  req.FirstNameKana,
		LastNameKana:   req.LastNameKana,
		Email:          req.Email,
		Permissions:    req.Permissions,
		AssignedBrands: req.AssignedBrands,
		Status:         req.Status,
	}

	m, err := h.uc.Update(ctx, input)
	if err != nil {
		writeMemberErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(m)
}

// -----------------------------------------------------------------------------
// GET /members
// -----------------------------------------------------------------------------
func (h *MemberHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	qv := r.URL.Query()

	var f memberdom.Filter
	f.SearchQuery = strings.TrimSpace(qv.Get("q"))

	me, ok := httpmw.CurrentMember(r)
	if !ok || strings.TrimSpace(me.CompanyID) == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}
	f.CompanyID = strings.TrimSpace(me.CompanyID)

	f.Status = strings.TrimSpace(qv.Get("status"))
	if v := strings.TrimSpace(qv.Get("brandIds")); v != "" {
		f.BrandIDs = splitCSV(v)
	}
	if v := strings.TrimSpace(qv.Get("brands")); v != "" {
		f.Brands = splitCSV(v)
	}

	var sort common.Sort
	switch strings.ToLower(strings.TrimSpace(qv.Get("sort"))) {
	case "name":
		sort.Column = "name"
	case "email":
		sort.Column = "email"
	case "joinedat":
		sort.Column = "joinedAt"
	case "updatedat":
		sort.Column = "updatedAt"
	default:
		sort.Column = "updatedAt"
	}
	switch strings.ToLower(strings.TrimSpace(qv.Get("order"))) {
	case "asc":
		sort.Order = "asc"
	default:
		sort.Order = "desc"
	}

	var page common.Page
	page.Number = clampInt(parseIntDefault(qv.Get("page"), 1), 1, 1_000_000)
	page.PerPage = clampInt(parseIntDefault(qv.Get("perPage"), 50), 1, 200)

	res, err := h.uc.List(ctx, f, sort, page)
	if err != nil {
		writeMemberErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(res.Items)
}

// -----------------------------------------------------------------------------
// GET /members/{id}
// -----------------------------------------------------------------------------
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

// -----------------------------------------------------------------------------
// Error responses
// -----------------------------------------------------------------------------
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

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
