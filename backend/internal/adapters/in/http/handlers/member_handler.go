// backend/internal/adapters/in/http/handlers/member_handler.go
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

// MemberHandler handles /members related endpoints.
type MemberHandler struct {
	uc *memberuc.MemberUsecase
}

func NewMemberHandler(uc *memberuc.MemberUsecase) http.Handler {
	return &MemberHandler{uc: uc}
}

func (h *MemberHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimRight(r.URL.Path, "/")
	if path == "" {
		path = "/"
	}

	switch {
	case r.Method == http.MethodPost && path == "/members":
		h.create(w, r)
	case r.Method == http.MethodGet && path == "/members":
		h.list(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/members/"):
		id := strings.TrimPrefix(path, "/members/")
		h.get(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// -----------------------------------------------------------------------------
// POST /members
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
	// ★ 追加: Firebase UID（必要に応じてフロントから渡す）
	FirebaseUID string `json:"firebaseUid,omitempty"`
}

func (h *MemberHandler) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req memberCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	// ----------- companyId は CurrentMember から強制適用（唯一の情報源） -----------
	me, ok := httpmw.CurrentMember(r)
	if !ok || strings.TrimSpace(me.CompanyID) == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}
	companyID := strings.TrimSpace(me.CompanyID)

	input := memberuc.CreateMemberInput{
		ID:             req.ID,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		FirstNameKana:  req.FirstNameKana,
		LastNameKana:   req.LastNameKana,
		Email:          req.Email,
		Permissions:    req.Permissions,
		AssignedBrands: req.AssignedBrands,
		CompanyID:      companyID,
		Status:         req.Status,
		// FirebaseUID は usecase 側にフィールドを追加したタイミングでここに渡す想定
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
// GET /members
// -----------------------------------------------------------------------------

func (h *MemberHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	qv := r.URL.Query()

	var f memberdom.Filter
	f.SearchQuery = strings.TrimSpace(qv.Get("q"))

	// ----------- companyId は CurrentMember から強制適用（唯一の情報源） -----------
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

	// -------- Sort --------
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

	// -------- Page --------
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
// Error Response
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
