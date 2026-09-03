// backend/internal/adapters/in/http/console/handler/member_handler.go
package consoleHandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	httpmw "narratives/internal/adapters/in/http/middleware"
	consolequery "narratives/internal/application/query/console"
	memberusecase "narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	memberdom "narratives/internal/domain/member"
)

// -----------------------------------------------------------------------------
// MemberHandler
// -----------------------------------------------------------------------------

type MemberHandler struct {
	memberUC        *memberusecase.MemberUsecase
	detailQuery     *consolequery.MemberDetailQuery
	managementQuery *consolequery.MemberManagementQuery
}

// NewMemberHandler — メンバーハンドラ
func NewMemberHandler(repo memberdom.Repository) http.Handler {
	return &MemberHandler{
		memberUC:        memberusecase.NewMemberUsecase(repo),
		detailQuery:     consolequery.NewMemberDetailQuery(repo),
		managementQuery: consolequery.NewMemberManagementQuery(repo),
	}
}

// -----------------------------------------------------------------------------
// Response DTOs
// -----------------------------------------------------------------------------

// memberResponse は Console frontend が利用する Member の BFF レスポンス。
// この JSON shape を frontend 側の正とし、frontend では欠損値補完や
// Member 用の normalize / mapper を行わない。
//
// Contract:
// - string 項目は値が空でも必ず JSON に含める。
// - permissions は未設定でも [] を返す。
// - assignedBrands は未割当の場合 null を返す。
// - updatedAt / updatedBy は未設定の場合 null を返す。
// - displayName は backend で生成した値を返す。
type memberResponse struct {
	ID             string     `json:"id"`
	UID            string     `json:"uid"`
	FirstName      string     `json:"firstName"`
	LastName       string     `json:"lastName"`
	FirstNameKana  string     `json:"firstNameKana"`
	LastNameKana   string     `json:"lastNameKana"`
	Email          string     `json:"email"`
	Permissions    []string   `json:"permissions"`
	AssignedBrands []string   `json:"assignedBrands"`
	CompanyID      string     `json:"companyId"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      *time.Time `json:"updatedAt"`
	UpdatedBy      *string    `json:"updatedBy"`
	DisplayName    string     `json:"displayName"`
}

func toMemberResponse(docID string, m memberdom.Member) memberResponse {
	permissions := m.Permissions
	if permissions == nil {
		permissions = []string{}
	}

	assignedBrands := m.AssignedBrands
	if len(assignedBrands) == 0 {
		assignedBrands = nil
	}

	return memberResponse{
		ID:             docID,
		UID:            m.UID,
		FirstName:      m.FirstName,
		LastName:       m.LastName,
		FirstNameKana:  m.FirstNameKana,
		LastNameKana:   m.LastNameKana,
		Email:          m.Email,
		Permissions:    permissions,
		AssignedBrands: assignedBrands,
		CompanyID:      m.CompanyID,
		Status:         m.Status,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
		UpdatedBy:      m.UpdatedBy,
		DisplayName:    memberdom.FormatLastFirst(m.LastName, m.FirstName),
	}
}

// -----------------------------------------------------------------------------
// ServeHTTP（ルーティング分岐）
// -----------------------------------------------------------------------------
//
// 方針:
// - /members の POST は招待前 member 作成として扱う。
// - 通常の console member 作成では request body の uid を信用しない。
// - GET /members/me は Authorization token の Firebase UID から現在ログイン中 member を返す。
// - GET /members/by-id/{memberId} は Firestore member docId 専用として扱う。
// - GET /members/{uid} は Firebase UID 専用として扱う。
// - PATCH /members/{id} は member docId 専用として扱う。
// - DELETE /members/{id} は member docId 専用として扱う。
func (h *MemberHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimRight(r.URL.Path, "/")
	if path == "" {
		path = "/"
	}

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

	// /members/{uid} より先に判定し、me を uid として扱わない。
	if path == "/members/me" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.me(w, r)
		return
	}

	// /members/{uid} より先に判定し、by-id を uid として扱わない。
	if path == "/members/by-id" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid member id"})
		return
	}

	if strings.HasPrefix(path, "/members/by-id/") {
		memberID := strings.TrimPrefix(path, "/members/by-id/")
		if memberID == "" || strings.Contains(memberID, "/") {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid member id"})
			return
		}

		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.getByID(w, r, memberID)
		return
	}

	if strings.HasPrefix(path, "/members/") {
		rest := strings.TrimPrefix(path, "/members/")
		parts := strings.Split(rest, "/")

		// GET は Firebase UID、PATCH / DELETE は member docId。
		if len(parts) == 1 {
			idOrUID := parts[0]
			if idOrUID == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
				return
			}

			switch r.Method {
			case http.MethodGet:
				h.getByUID(w, r, idOrUID)
				return
			case http.MethodPatch:
				h.update(w, r, idOrUID)
				return
			case http.MethodDelete:
				h.delete(w, r, idOrUID)
				return
			default:
				methodNotAllowed(w)
				return
			}
		}
	}

	writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
}

// -----------------------------------------------------------------------------
// POST /members — Create
// -----------------------------------------------------------------------------
//
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

	rec, err := h.memberUC.Create(
		r.Context(),
		memberusecase.CreateMemberInput{
			// 通常 console member 作成では request body の uid を信用しない。
			// 招待前 member として uid 空で作成する。
			UID:            "",
			FirstName:      req.FirstName,
			LastName:       req.LastName,
			FirstNameKana:  req.FirstNameKana,
			LastNameKana:   req.LastNameKana,
			Email:          req.Email,
			Permissions:    req.Permissions,
			AssignedBrands: req.AssignedBrands,
			CompanyID:      me.CompanyID,
			Status:         req.Status,
		},
	)
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
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	me, ok := httpmw.CurrentMember(r)
	if !ok || me.CompanyID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req memberUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	rec, err := h.memberUC.Update(
		r.Context(),
		memberusecase.UpdateMemberInput{
			MemberID:       id,
			CompanyID:      me.CompanyID,
			FirstName:      req.FirstName,
			LastName:       req.LastName,
			FirstNameKana:  req.FirstNameKana,
			LastNameKana:   req.LastNameKana,
			Email:          req.Email,
			Permissions:    req.Permissions,
			AssignedBrands: req.AssignedBrands,
			Status:         req.Status,
		},
	)
	if err != nil {
		writeMemberErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toMemberResponse(rec.DocID, rec.Member))
}

// -----------------------------------------------------------------------------
// DELETE /members/{id}
// -----------------------------------------------------------------------------

func (h *MemberHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	me, ok := httpmw.CurrentMember(r)
	if !ok || me.CompanyID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	if err := h.memberUC.Delete(r.Context(), id); err != nil {
		writeMemberErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"id": id})
}

// -----------------------------------------------------------------------------
// GET /members
// -----------------------------------------------------------------------------
//
// NOTE:
// 取得処理、filter/page の組み立て、page/perPage の補正は
// MemberManagementQuery に移譲する。
// handler は HTTP query parameter の読み取り、認証 company scope の取得、
// response DTO への変換だけを担当する。

func (h *MemberHandler) list(w http.ResponseWriter, r *http.Request) {
	qv := r.URL.Query()

	me, ok := httpmw.CurrentMember(r)
	if !ok || me.CompanyID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	res, err := h.managementQuery.ListByCompanyID(
		r.Context(),
		consolequery.MemberListInput{
			CompanyID:   me.CompanyID,
			SearchQuery: qv.Get("q"),
			UID:         qv.Get("uid"),
			Status:      qv.Get("status"),
			BrandIDs:    splitCSV(qv.Get("brandIds")),
			Page:        parseIntDefault(qv.Get("page"), 1),
			PerPage:     parseIntDefault(qv.Get("perPage"), 50),
		},
	)
	if err != nil {
		writeMemberErr(w, err)
		return
	}

	items := make([]memberResponse, 0, len(res.Items))
	for _, rec := range res.Items {
		items = append(items, toMemberResponse(rec.DocID, rec.Member))
	}

	writeJSON(
		w,
		http.StatusOK,
		common.PageResult[memberResponse]{
			Items:      items,
			TotalCount: res.TotalCount,
			TotalPages: res.TotalPages,
			Page:       res.Page,
			PerPage:    res.PerPage,
		},
	)
}

// -----------------------------------------------------------------------------
// GET /members/me
// -----------------------------------------------------------------------------
//
// NOTE:
// /members/me は BootstrapAuthMiddleware 配下でも動く必要がある。
// そのため CurrentMember には依存しない。
// Firebase UID は Authorization token から取得し、GetByUID で member を取得する。

func (h *MemberHandler) me(w http.ResponseWriter, r *http.Request) {
	uid, _, ok := httpmw.CurrentUIDAndEmail(r)
	if !ok || uid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	rec, err := h.memberUC.GetCurrentMember(
		r.Context(),
		memberusecase.GetCurrentMemberInput{FirebaseUID: uid},
	)
	if err != nil {
		if errors.Is(err, memberdom.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "member not found"})
			return
		}

		writeMemberErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toMemberResponse(rec.DocID, rec.Member))
}

// -----------------------------------------------------------------------------
// GET /members/by-id/{memberId}
// -----------------------------------------------------------------------------
//
// NOTE:
// GET /members/by-id/{memberId} は Firestore member docId 専用。
// Firebase UID として解釈しない。
// 取得処理と company scope 判定は MemberDetailQuery に移譲する。

func (h *MemberHandler) getByID(w http.ResponseWriter, r *http.Request, memberID string) {
	if memberID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid member id"})
		return
	}

	me, ok := httpmw.CurrentMember(r)
	if !ok || me.CompanyID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	rec, err := h.detailQuery.GetByID(r.Context(), memberID, me.CompanyID)
	if err != nil {
		if errors.Is(err, consolequery.ErrMemberForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}

		writeMemberErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toMemberResponse(rec.DocID, rec.Member))
}

// -----------------------------------------------------------------------------
// GET /members/{uid}
// -----------------------------------------------------------------------------
//
// NOTE:
// GET /members/{uid} は Firebase UID 専用。
// docId では検索しない。
// レスポンスの id は member の Firestore docId を返す。
//
// 取得処理と company scope 判定は MemberDetailQuery に移譲する。
// handler は uid validation、認証 company scope の取得、
// response DTO への変換だけを担当する。

func (h *MemberHandler) getByUID(w http.ResponseWriter, r *http.Request, uid string) {
	if uid == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid uid"})
		return
	}

	me, ok := httpmw.CurrentMember(r)
	if !ok || me.CompanyID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	rec, err := h.detailQuery.GetByUID(r.Context(), uid, me.CompanyID)
	if err != nil {
		if errors.Is(err, consolequery.ErrMemberForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}

		writeMemberErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toMemberResponse(rec.DocID, rec.Member))
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
		errors.Is(err, memberdom.ErrInvalidStatus):
		code = http.StatusBadRequest
	case errors.Is(err, memberdom.ErrConflict):
		code = http.StatusConflict
	case errors.Is(err, memberdom.ErrPreconditionFailed):
		code = http.StatusPreconditionFailed
	}

	writeJSON(w, code, map[string]string{"error": err.Error()})
}
