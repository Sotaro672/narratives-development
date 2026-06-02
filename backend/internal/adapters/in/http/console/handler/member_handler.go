// backend/internal/adapters/in/http/console/handler/member_handler.go
package consoleHandler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	httpmw "narratives/internal/adapters/in/http/middleware"
	common "narratives/internal/domain/common"
	memberdom "narratives/internal/domain/member"
)

type memberCompanyIDReader interface {
	GetCompanyIDByFirebaseUID(ctx context.Context, uid string) (string, error)
}

// -----------------------------------------------------------------------------
// MemberHandler
// -----------------------------------------------------------------------------
type MemberHandler struct {
	repo memberdom.Repository
}

// NewMemberHandler — メンバーハンドラ
func NewMemberHandler(
	repo memberdom.Repository,
) http.Handler {
	return &MemberHandler{
		repo: repo,
	}
}

// -----------------------------------------------------------------------------
// Response DTOs
// -----------------------------------------------------------------------------
type memberResponse struct {
	ID             string     `json:"id"`
	UID            string     `json:"uid,omitempty"`
	FirstName      string     `json:"firstName,omitempty"`
	LastName       string     `json:"lastName,omitempty"`
	FirstNameKana  string     `json:"firstNameKana,omitempty"`
	LastNameKana   string     `json:"lastNameKana,omitempty"`
	Email          string     `json:"email,omitempty"`
	Permissions    []string   `json:"permissions"`
	AssignedBrands []string   `json:"assignedBrands,omitempty"`
	CompanyID      string     `json:"companyId,omitempty"`
	Status         string     `json:"status,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy      *string    `json:"updatedBy,omitempty"`
	DeletedAt      *time.Time `json:"deletedAt,omitempty"`
	DeletedBy      *string    `json:"deletedBy,omitempty"`
	DisplayName    string     `json:"displayName,omitempty"`
}

func toMemberResponse(docID string, m memberdom.Member) memberResponse {
	return memberResponse{
		ID:             docID,
		UID:            m.UID,
		FirstName:      m.FirstName,
		LastName:       m.LastName,
		FirstNameKana:  m.FirstNameKana,
		LastNameKana:   m.LastNameKana,
		Email:          m.Email,
		Permissions:    m.Permissions,
		AssignedBrands: m.AssignedBrands,
		CompanyID:      m.CompanyID,
		Status:         m.Status,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
		UpdatedBy:      m.UpdatedBy,
		DeletedAt:      m.DeletedAt,
		DeletedBy:      m.DeletedBy,
		DisplayName:    memberdom.FormatLastFirst(m.LastName, m.FirstName),
	}
}

// -----------------------------------------------------------------------------
// ServeHTTP（ルーティング分岐）
// -----------------------------------------------------------------------------
// 方針:
// - /members の POST は招待前 member 作成として扱う。
// - 通常の console member 作成では request body の uid を信用しない。
// - GET /members/me は Authorization token の Firebase UID から現在ログイン中 member を返す。
// - GET /members/{id} は member docId 専用として扱う。
// - PATCH /members/{id} は member docId 専用として扱う。
// - /members/by-firebase-uid/{uid} は廃止。
// - /members/{id}/bind-firebase-uid は request body の uid ではなく CurrentMember の UID を使う。
// - 招待承諾フローは CurrentMember が未確立の状態でも動く必要があるため、別 handler/API で扱う。
func (h *MemberHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimRight(r.URL.Path, "/")
	if path == "" {
		path = "/"
	}

	// /members
	if path == "/members" {
		switch r.Method {
		case http.MethodPost:
			h.create(w, r)
			return
		case http.MethodGet:
			h.list(w, r)
			return
		default:
			methodNotAllowed(w)
			return
		}
	}

	// /members/me
	// IMPORTANT:
	// /members/{id} より先に判定する。
	// me を docId として扱わないため。
	if path == "/members/me" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.me(w, r)
		return
	}

	// /members/by-company
	if path == "/members/by-company" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.listByCompanyID(w, r)
		return
	}

	// /members/by-firebase-uid/{uid} は廃止。
	// ここでは専用分岐を持たず、/members/{...} 配下の未対応ルートとして not_found に落とす。

	// /members/{...}
	if strings.HasPrefix(path, "/members/") {
		rest := strings.TrimPrefix(path, "/members/")
		parts := strings.Split(rest, "/")

		// /members/{id}
		// GET / PATCH は member docId 専用。
		if len(parts) == 1 {
			id := strings.TrimSpace(parts[0])
			if id == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
				return
			}

			switch r.Method {
			case http.MethodGet:
				h.get(w, r, id)
				return
			case http.MethodPatch:
				h.update(w, r, id)
				return
			default:
				methodNotAllowed(w)
				return
			}
		}

		// /members/{id}/display-name
		if len(parts) == 2 && parts[1] == "display-name" {
			id := strings.TrimSpace(parts[0])
			if id == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
				return
			}

			if r.Method != http.MethodGet {
				methodNotAllowed(w)
				return
			}

			h.getDisplayName(w, r, id)
			return
		}

		// /members/{id}/bind-firebase-uid
		// NOTE:
		// この console handler では request body の uid を信用しない。
		// CurrentMember の UID を使って bind する。
		// 招待承諾時の uid bind は CurrentMember が未確立でも動く専用APIで扱うこと。
		if len(parts) == 2 && parts[1] == "bind-firebase-uid" {
			id := strings.TrimSpace(parts[0])
			if id == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
				return
			}

			if r.Method != http.MethodPost {
				methodNotAllowed(w)
				return
			}

			h.bindFirebaseUID(w, r, id)
			return
		}

		// /members/{id}/invitation はこのハンドラでは扱わない。
		// MemberInvitationHandler 側にルーティングさせるため not_found を返す。
		if len(parts) == 2 && parts[1] == "invitation" {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
			return
		}
	}

	writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
}

// -----------------------------------------------------------------------------
// POST /members — Create
// -----------------------------------------------------------------------------
// NOTE:
// uid は request body から受け取らない。
// 通常の console で作成される member は招待前 member として uid 空で作成する。
// 初回会社登録者の uid は /auth/bootstrap 側で Firebase token 由来の UID を保存する。
type memberCreateRequest struct {
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
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	me, ok := httpmw.CurrentMember(r)
	if !ok || me.CompanyID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	m := memberdom.Member{
		UID:            "",
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		FirstNameKana:  req.FirstNameKana,
		LastNameKana:   req.LastNameKana,
		Email:          strings.TrimSpace(req.Email),
		Permissions:    dedupStrings(req.Permissions),
		AssignedBrands: dedupStrings(req.AssignedBrands),
		CompanyID:      me.CompanyID,
		Status:         req.Status,
		CreatedAt:      time.Now().UTC(),
	}

	rec, err := h.repo.Create(ctx, m)
	if err != nil {
		writeMemberErr(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toMemberResponse(rec.DocID, rec.Member))
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
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	me, ok := httpmw.CurrentMember(r)
	if !ok || me.CompanyID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	current, err := h.repo.GetByID(ctx, id)
	if err != nil {
		writeMemberErr(w, err)
		return
	}

	if current.Member.CompanyID != me.CompanyID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	var req memberUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	patch := memberdom.MemberPatch{
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		FirstNameKana:  req.FirstNameKana,
		LastNameKana:   req.LastNameKana,
		Email:          req.Email,
		Permissions:    req.Permissions,
		AssignedBrands: req.AssignedBrands,
		Status:         req.Status,
	}

	rec, err := h.repo.Update(ctx, id, patch)
	if err != nil {
		writeMemberErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toMemberResponse(rec.DocID, rec.Member))
}

// -----------------------------------------------------------------------------
// POST /members/{id}/bind-firebase-uid
// -----------------------------------------------------------------------------
// NOTE:
// request body の uid は使わない。
// CurrentMember の UID を使うことで、クライアントが任意の Firebase UID を指定できないようにする。
func (h *MemberHandler) bindFirebaseUID(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	me, ok := httpmw.CurrentMember(r)
	if !ok || me.CompanyID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	uid := strings.TrimSpace(me.UID)
	if uid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized: current member uid is empty"})
		return
	}

	current, err := h.repo.GetByID(ctx, id)
	if err != nil {
		writeMemberErr(w, err)
		return
	}

	if current.Member.CompanyID != me.CompanyID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	patch := memberdom.MemberPatch{
		UID: &uid,
	}

	rec, err := h.repo.Update(ctx, id, patch)
	if err != nil {
		writeMemberErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toMemberResponse(rec.DocID, rec.Member))
}

// -----------------------------------------------------------------------------
// GET /members
// -----------------------------------------------------------------------------
func (h *MemberHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	qv := r.URL.Query()

	me, ok := httpmw.CurrentMember(r)
	if !ok || me.CompanyID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	f := memberdom.Filter{
		SearchQuery: strings.TrimSpace(qv.Get("q")),
		UID:         strings.TrimSpace(qv.Get("uid")),
		Status:      strings.TrimSpace(qv.Get("status")),
	}

	if v := qv.Get("brandIds"); v != "" {
		f.BrandIDs = splitCSV(v)
	}

	page := common.Page{
		Number:  clampInt(parseIntDefault(qv.Get("page"), 1), 1, 1_000_000),
		PerPage: clampInt(parseIntDefault(qv.Get("perPage"), 50), 1, 200),
	}

	res, err := h.repo.ListByCompanyID(ctx, me.CompanyID, f, page)
	if err != nil {
		writeMemberErr(w, err)
		return
	}

	items := make([]memberResponse, 0, len(res.Items))
	for _, rec := range res.Items {
		items = append(items, toMemberResponse(rec.DocID, rec.Member))
	}

	writeJSON(w, http.StatusOK, items)
}

// -----------------------------------------------------------------------------
// GET /members/me
// -----------------------------------------------------------------------------
// NOTE:
// /members/me は BootstrapAuthMiddleware 配下でも動く必要がある。
// そのため CurrentMember には依存しない。
// Firebase UID は Authorization token から取得し、
// repository adapter 拡張で companyID を解決してから ListByCompanyID + Filter.UID で member を取得する。
func (h *MemberHandler) me(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	uid, _, ok := httpmw.CurrentUIDAndEmail(r)
	if !ok || strings.TrimSpace(uid) == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	if h.repo == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "member repository is not configured"})
		return
	}

	reader, ok := any(h.repo).(memberCompanyIDReader)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "member company resolver is not configured"})
		return
	}

	companyID, err := reader.GetCompanyIDByFirebaseUID(ctx, strings.TrimSpace(uid))
	if err != nil {
		if errors.Is(err, memberdom.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "member not found"})
			return
		}

		writeMemberErr(w, err)
		return
	}

	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "member companyId not found"})
		return
	}

	res, err := h.repo.ListByCompanyID(ctx, companyID, memberdom.Filter{
		UID: strings.TrimSpace(uid),
	}, common.Page{
		Number:  1,
		PerPage: 1,
	})
	if err != nil {
		writeMemberErr(w, err)
		return
	}

	if len(res.Items) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "member not found"})
		return
	}

	rec := res.Items[0]

	writeJSON(w, http.StatusOK, toMemberResponse(rec.DocID, rec.Member))
}

// -----------------------------------------------------------------------------
// GET /members/by-company
// -----------------------------------------------------------------------------
func (h *MemberHandler) listByCompanyID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	me, ok := httpmw.CurrentMember(r)
	if !ok || me.CompanyID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	companyID := me.CompanyID

	log.Printf("[memberHandler.listByCompanyID] ENTER companyID=%q", companyID)

	res, err := h.repo.ListByCompanyID(ctx, companyID, memberdom.Filter{}, common.Page{
		Number:  1,
		PerPage: 200,
	})
	if err != nil {
		log.Printf("[memberHandler.listByCompanyID] ListByCompanyID error: %v", err)
		writeMemberErr(w, err)
		return
	}

	items := make([]memberResponse, 0, len(res.Items))
	for _, rec := range res.Items {
		items = append(items, toMemberResponse(rec.DocID, rec.Member))
	}

	log.Printf("[memberHandler.listByCompanyID] OK items=%d", len(items))

	writeJSON(w, http.StatusOK, items)
}

// -----------------------------------------------------------------------------
// GET /members/{id}
// -----------------------------------------------------------------------------
// NOTE:
// GET /members/{id} は member docId 専用。
// Firebase UID では検索しない。
// レスポンスの id は member の Firestore docId を返す。
func (h *MemberHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	me, ok := httpmw.CurrentMember(r)
	if !ok || me.CompanyID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	rec, err := h.repo.GetByID(ctx, id)
	if err != nil {
		writeMemberErr(w, err)
		return
	}

	if rec.Member.CompanyID != me.CompanyID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	writeJSON(w, http.StatusOK, toMemberResponse(rec.DocID, rec.Member))
}

// -----------------------------------------------------------------------------
// GET /members/{id}/display-name
// -----------------------------------------------------------------------------
// NOTE:
// display-name は member docId 専用。
func (h *MemberHandler) getDisplayName(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	rec, err := h.repo.GetByID(ctx, id)
	if err != nil {
		writeMemberErr(w, err)
		return
	}

	displayName := memberdom.FormatLastFirst(rec.Member.LastName, rec.Member.FirstName)

	writeJSON(w, http.StatusOK, map[string]string{
		"displayName": displayName,
	})
}

// -----------------------------------------------------------------------------
// Error responses
// -----------------------------------------------------------------------------
func writeMemberErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, memberdom.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, memberdom.ErrInvalidUID),
		errors.Is(err, memberdom.ErrInvalidEmail),
		errors.Is(err, memberdom.ErrInvalidFirstName),
		errors.Is(err, memberdom.ErrInvalidLastName),
		errors.Is(err, memberdom.ErrInvalidFirstKana),
		errors.Is(err, memberdom.ErrInvalidLastKana),
		errors.Is(err, memberdom.ErrInvalidCreatedAt),
		errors.Is(err, memberdom.ErrInvalidUpdatedAt),
		errors.Is(err, memberdom.ErrInvalidUpdatedBy),
		errors.Is(err, memberdom.ErrInvalidDeletedAt),
		errors.Is(err, memberdom.ErrInvalidDeletedBy),
		errors.Is(err, memberdom.ErrInvalidStatus):
		code = http.StatusBadRequest
	case errors.Is(err, memberdom.ErrConflict):
		code = http.StatusConflict
	case errors.Is(err, memberdom.ErrPreconditionFailed):
		code = http.StatusPreconditionFailed
	}

	writeJSON(w, code, map[string]string{"error": err.Error()})
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
