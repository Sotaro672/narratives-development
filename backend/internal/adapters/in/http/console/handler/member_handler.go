// backend\internal\adapters\in\http\console\handler\member_handler.go
package consoleHandler

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
	uc   *memberuc.MemberUsecase
	repo memberdom.Repository
}

// NewMemberHandler — メンバーハンドラ
// ※ 第二引数 repo は ListMembersByCompanyID 用に追加
func NewMemberHandler(
	uc *memberuc.MemberUsecase,
	repo memberdom.Repository,
) http.Handler {
	return &MemberHandler{
		uc:   uc,
		repo: repo,
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

	// ★ 追加: GET /members/by-company
	// currentMember の companyId を使って ListMembersByCompanyID を叩くデバッグ用エンドポイント
	case r.Method == http.MethodGet && path == "/members/by-company":
		h.listByCompanyID(w, r)

	// ★ 追加: GET /members/{id}/display-name
	case r.Method == http.MethodGet &&
		strings.HasPrefix(path, "/members/") &&
		strings.HasSuffix(path, "/display-name"):

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
		f.BrandIDs = splitCSV(v) // ✅ helpers.go に集約
	}
	if v := strings.TrimSpace(qv.Get("brands")); v != "" {
		f.Brands = splitCSV(v) // ✅ helpers.go に集約
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
// GET /members/by-company
// -----------------------------------------------------------------------------
func (h *MemberHandler) listByCompanyID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	me, ok := httpmw.CurrentMember(r)
	if !ok || strings.TrimSpace(me.CompanyID) == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	companyID := strings.TrimSpace(me.CompanyID)

	// CursorPage のゼロ値（フィールド構成に依存しない）
	var cpage memberdom.CursorPage

	log.Printf("[memberHandler.listByCompanyID] ENTER companyID=%q", companyID)

	res, err := memberdom.ListMembersByCompanyID(ctx, h.repo, companyID, cpage)
	if err != nil {
		log.Printf("[memberHandler.listByCompanyID] ListMembersByCompanyID error: %v", err)
		writeMemberErr(w, err)
		return
	}

	type memberWithDisplayName struct {
		memberdom.Member
		DisplayName string `json:"displayName"`
	}

	items := make([]memberWithDisplayName, 0, len(res.Items))
	for _, m := range res.Items {
		items = append(items, memberWithDisplayName{
			Member:      m,
			DisplayName: memberdom.FormatLastFirst(m.LastName, m.FirstName),
		})
	}

	log.Printf("[memberHandler.listByCompanyID] OK items=%d", len(items))

	_ = json.NewEncoder(w).Encode(items)
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
func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
