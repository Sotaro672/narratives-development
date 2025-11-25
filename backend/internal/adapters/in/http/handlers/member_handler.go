// backend/internal/adapters/in/http/handlers/member_handler.go
package handlers

import (
	"encoding/json"
	"log"
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
	uc *memberuc.MemberUsecase
}

// NewMemberHandler — メンバーハンドラ
// ※ 招待メール送信は /members/{id}/invitation 用の MemberInvitationHandler に委譲するため、
//
//	ここでは InvitationCommandPort は扱わない。
func NewMemberHandler(
	uc *memberuc.MemberUsecase,
) http.Handler {
	return &MemberHandler{
		uc: uc,
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

	// 通常のメンバー CRUD ルーティング
	switch {

	case r.Method == http.MethodPost && path == "/members":
		h.create(w, r)

	case r.Method == http.MethodGet && path == "/members":
		h.list(w, r)

	// ★ 追加: GET /members/{id}/display-name
	case r.Method == http.MethodGet &&
		strings.HasPrefix(path, "/members/") &&
		strings.HasSuffix(path, "/display-name"):
		// /members/{id}/display-name から {id} 部分を抽出
		id := strings.TrimPrefix(path, "/members/")
		id = strings.TrimSuffix(id, "/display-name")
		h.getDisplayName(w, r, id)

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

	log.Printf("[memberHandler.list] ENTER path=%s", r.URL.Path)

	var f memberdom.Filter
	f.SearchQuery = strings.TrimSpace(qv.Get("q"))

	me, ok := httpmw.CurrentMember(r)
	if !ok {
		log.Printf("[memberHandler.list] CurrentMember not found in context")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}
	log.Printf(
		"[memberHandler.list] CurrentMember.ID=%s companyId=%q",
		me.ID, me.CompanyID,
	)

	if strings.TrimSpace(me.CompanyID) == "" {
		log.Printf("[memberHandler.list] companyId empty → 401")
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

	log.Printf(
		"[memberHandler.list] calling Usecase.List companyId=%q search=%q status=%q page=%d perPage=%d",
		f.CompanyID, f.SearchQuery, f.Status, page.Number, page.PerPage,
	)

	res, err := h.uc.List(ctx, f, sort, page)
	if err != nil {
		log.Printf("[memberHandler.list] Usecase.List error: %v", err)
		writeMemberErr(w, err)
		return
	}

	log.Printf("[memberHandler.list] OK items=%d", len(res.Items))
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
// GET /members/{id}/display-name
// -----------------------------------------------------------------------------
// assigneeId → assigneeName 変換用のシンプルなエンドポイント。
// Member の lastName / firstName から「姓 名」を組み立てて返す。
func (h *MemberHandler) getDisplayName(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	m, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeMemberErr(w, err)
		return
	}

	displayName := memberdom.FormatLastFirst(m.LastName, m.FirstName)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"displayName": displayName,
	})
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
