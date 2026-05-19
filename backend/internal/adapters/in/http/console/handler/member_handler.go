// backend/internal/adapters/in/http/console/handler/member_handler.go
package consoleHandler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

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
// ※ 第二引数 repo は docId を返す用途でも利用する
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
// - GET /members/{uid} は Firebase UID 専用として扱う。
// - PATCH /members/{docId} は member docId 専用として扱う。
// - /members/by-firebase-uid/{uid} は廃止。
// - /members/{docId}/bind-firebase-uid は request body の uid ではなく CurrentMember の UID を使う。
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
			w.WriteHeader(http.StatusMethodNotAllowed)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "method_not_allowed"})
			return
		}
	}

	// /members/by-company
	if path == "/members/by-company" {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "method_not_allowed"})
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

		// /members/{uid}
		// GET は Firebase UID 専用。
		// PATCH は member docId 専用。
		if len(parts) == 1 {
			id := parts[0]
			if id == "" {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
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
				w.WriteHeader(http.StatusMethodNotAllowed)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "method_not_allowed"})
				return
			}
		}

		// /members/{docId}/display-name
		if len(parts) == 2 && parts[1] == "display-name" {
			id := parts[0]
			if id == "" {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
				return
			}

			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "method_not_allowed"})
				return
			}

			h.getDisplayName(w, r, id)
			return
		}

		// /members/{docId}/bind-firebase-uid
		// NOTE:
		// この console handler では request body の uid を信用しない。
		// CurrentMember の UID を使って bind する。
		// 招待承諾時の uid bind は CurrentMember が未確立でも動く専用APIで扱うこと。
		if len(parts) == 2 && parts[1] == "bind-firebase-uid" {
			id := parts[0]
			if id == "" {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
				return
			}

			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "method_not_allowed"})
				return
			}

			h.bindFirebaseUID(w, r, id)
			return
		}

		// /members/{docId}/invitation はこのハンドラでは扱わない
		// MemberInvitationHandler 側にルーティングさせるため not_found を返す
		if len(parts) == 2 && parts[1] == "invitation" {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
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
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	me, ok := httpmw.CurrentMember(r)
	if !ok || me.CompanyID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	m := memberdom.Member{
		UID:            "",
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		FirstNameKana:  req.FirstNameKana,
		LastNameKana:   req.LastNameKana,
		Email:          req.Email,
		Permissions:    dedupStrings(req.Permissions),
		AssignedBrands: dedupStrings(req.AssignedBrands),
		CompanyID:      me.CompanyID,
		Status:         req.Status,
		CreatedAt:      time.Now().UTC(),
	}

	rec, err := h.repo.CreateWithDocID(ctx, m)
	if err != nil {
		writeMemberErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(toMemberResponse(rec.DocID, rec.Member))
}

// -----------------------------------------------------------------------------
// PATCH /members/{docId}
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

	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	me, ok := httpmw.CurrentMember(r)
	if !ok || me.CompanyID == "" {
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

	updated, err := h.uc.Update(ctx, input)
	if err != nil {
		writeMemberErr(w, err)
		return
	}

	rec, err := h.repo.GetByDocID(ctx, id)
	if err != nil {
		writeMemberErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(toMemberResponse(rec.DocID, updated))
}

// -----------------------------------------------------------------------------
// POST /members/{docId}/bind-firebase-uid
// -----------------------------------------------------------------------------
// NOTE:
// request body の uid は使わない。
// CurrentMember の UID を使うことで、クライアントが任意の Firebase UID を指定できないようにする。
func (h *MemberHandler) bindFirebaseUID(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	me, ok := httpmw.CurrentMember(r)
	if !ok || me.CompanyID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	uid := me.UID
	if uid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: current member uid is empty"})
		return
	}

	rec, err := h.uc.BindFirebaseUID(ctx, memberuc.BindFirebaseUIDInput{
		DocID: id,
		UID:   uid,
	})
	if err != nil {
		writeMemberErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(toMemberResponse(rec.DocID, rec.Member))
}

// -----------------------------------------------------------------------------
// GET /members
// -----------------------------------------------------------------------------
func (h *MemberHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	qv := r.URL.Query()

	var f memberdom.Filter
	f.SearchQuery = qv.Get("q")
	f.UID = qv.Get("uid")

	me, ok := httpmw.CurrentMember(r)
	if !ok || me.CompanyID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	f.CompanyID = me.CompanyID
	f.Status = qv.Get("status")

	if v := qv.Get("brandIds"); v != "" {
		f.BrandIDs = splitCSV(v)
	}

	var sort common.Sort
	switch strings.ToLower(qv.Get("sort")) {
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

	switch strings.ToLower(qv.Get("order")) {
	case "asc":
		sort.Order = "asc"
	default:
		sort.Order = "desc"
	}

	var page common.Page
	page.Number = clampInt(parseIntDefault(qv.Get("page"), 1), 1, 1_000_000)
	page.PerPage = clampInt(parseIntDefault(qv.Get("perPage"), 50), 1, 200)

	res, err := h.repo.ListWithDocID(ctx, f, sort, page)
	if err != nil {
		writeMemberErr(w, err)
		return
	}

	items := make([]memberResponse, 0, len(res.Items))
	for _, rec := range res.Items {
		items = append(items, toMemberResponse(rec.DocID, rec.Member))
	}

	_ = json.NewEncoder(w).Encode(items)
}

// -----------------------------------------------------------------------------
// GET /members/by-company
// -----------------------------------------------------------------------------
func (h *MemberHandler) listByCompanyID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	me, ok := httpmw.CurrentMember(r)
	if !ok || me.CompanyID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	companyID := me.CompanyID

	res, err := h.repo.ListWithDocID(ctx, memberdom.Filter{
		CompanyID: companyID,
	}, common.Sort{}, common.Page{
		Number:  1,
		PerPage: 200,
	})
	if err != nil {
		writeMemberErr(w, err)
		return
	}

	items := make([]memberResponse, 0, len(res.Items))
	for _, rec := range res.Items {
		items = append(items, toMemberResponse(rec.DocID, rec.Member))
	}

	_ = json.NewEncoder(w).Encode(items)
}

// -----------------------------------------------------------------------------
// GET /members/{uid}
// -----------------------------------------------------------------------------
// NOTE:
// GET /members/{uid} は Firebase UID 専用。
// member docId では検索しない。
// レスポンスの id は member の Firestore docId を返す。
func (h *MemberHandler) get(w http.ResponseWriter, r *http.Request, uid string) {
	ctx := r.Context()

	if uid == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid uid"})
		return
	}

	me, ok := httpmw.CurrentMember(r)
	if !ok || me.CompanyID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	rec, err := h.repo.GetRecordByFirebaseUID(ctx, uid)
	if err != nil {
		writeMemberErr(w, err)
		return
	}

	if rec.Member.CompanyID != me.CompanyID {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	_ = json.NewEncoder(w).Encode(toMemberResponse(rec.DocID, rec.Member))
}

// -----------------------------------------------------------------------------
// GET /members/{docId}/display-name
// -----------------------------------------------------------------------------
// NOTE:
// display-name は docId 専用。
func (h *MemberHandler) getDisplayName(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	rec, err := h.repo.GetByDocID(ctx, id)
	if err != nil {
		writeMemberErr(w, err)
		return
	}

	displayName := memberdom.FormatLastFirst(rec.Member.LastName, rec.Member.FirstName)
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
	case memberdom.ErrNotFound:
		code = http.StatusNotFound
	case memberdom.ErrInvalidUID,
		memberdom.ErrInvalidEmail,
		memberdom.ErrInvalidCreatedAt:
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

func dedupStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
