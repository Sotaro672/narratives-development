// backend/internal/adapters/in/http/handlers/member_handler.go
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	memberuc "narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	memberdom "narratives/internal/domain/member"
)

//
// ──────────────────────────────────────────────────────────────────────────────
// Auth コンテキスト注入（Inbound Adapter / Middleware）
//  - リクエストの認証情報（uid）から、そのユーザーの companyId を解決して context に注入
//  - 本番では ID トークン検証 → uid 取得を別ミドルウェアで行い、ここでは uid→companyId 解決のみを担当
//  - 開発用に "X-Auth-UID" ヘッダー（なければクエリ uid）から uid を拾います
// ──────────────────────────────────────────────────────────────────────────────
//

type authCtxKey int

const (
	authKey authCtxKey = iota
)

// AuthInfo は下流で参照する最小限の認証情報
type AuthInfo struct {
	UID       string
	CompanyID string
}

// authFromContext は AuthInfo を取り出します（無ければゼロ値）
func authFromContext(ctx context.Context) AuthInfo {
	v := ctx.Value(authKey)
	if ai, ok := v.(AuthInfo); ok {
		return ai
	}
	return AuthInfo{}
}

// withAuth は AuthInfo を context に詰めます
func withAuth(ctx context.Context, ai AuthInfo) context.Context {
	return context.WithValue(ctx, authKey, ai)
}

// MiddlewareAuthCompany は uid → member → companyId を解決して context に注入する HTTP ミドルウェア。
// uid の取得は開発用に "X-Auth-UID" ヘッダ、無ければクエリ ?uid= を使用します。
// 本番運用では別途トークン検証ミドルウェアで uid を埋め、その値をここで参照する形を推奨します。
func MiddlewareAuthCompany(uc *memberuc.MemberUsecase) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uid := strings.TrimSpace(r.Header.Get("X-Auth-UID"))
			if uid == "" {
				uid = strings.TrimSpace(r.URL.Query().Get("uid"))
			}

			ai := AuthInfo{UID: uid}

			// uid があれば、member を引いて companyId を解決
			if uid != "" && uc != nil {
				if m, err := uc.GetByID(r.Context(), uid); err == nil {
					// Member エンティティに CompanyID フィールドがある前提（無い場合は "" になる）
					ai.CompanyID = strings.TrimSpace(m.CompanyID)
				}
				// 取得に失敗しても致命的ではないため、そのまま続行（下流で 0 値扱い）
			}

			ctx := withAuth(r.Context(), ai)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

//
// ──────────────────────────────────────────────────────────────────────────────
// /members Handler 本体
// ──────────────────────────────────────────────────────────────────────────────
//

// MemberHandler は /members 関連のエンドポイントを担当します。
type MemberHandler struct {
	uc *memberuc.MemberUsecase
}

// NewMemberHandler はHTTPハンドラを初期化します。
func NewMemberHandler(uc *memberuc.MemberUsecase) http.Handler {
	// このハンドラ自体は ServeHTTP を実装する multiplexer です。
	// 実際のルータで使う際は MiddlewareAuthCompany(uc) でラップしてください。
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

	// 任意: 所属会社/ステータス（会社は下流でサーバ強制上書き）
	CompanyID string `json:"companyId,omitempty"`
	Status    string `json:"status,omitempty"` // 例: "active" | "inactive"
}

func (h *MemberHandler) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req memberCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	// 認証コンテキストから companyId を強制適用
	ai := authFromContext(ctx)

	input := memberuc.CreateMemberInput{
		ID:             req.ID,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		FirstNameKana:  req.FirstNameKana,
		LastNameKana:   req.LastNameKana,
		Email:          req.Email,
		Permissions:    req.Permissions,
		AssignedBrands: req.AssignedBrands,

		// ★ クライアント指定は無視してサーバで上書き
		CompanyID: ai.CompanyID,
		Status:    req.Status,

		CreatedAt: nil, // サーバ側で now
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
	qv := r.URL.Query()

	// --- Filter ---
	var f memberdom.Filter
	f.SearchQuery = strings.TrimSpace(qv.Get("q"))

	// 認証コンテキストから companyId を強制適用
	ai := authFromContext(ctx)
	if ai.CompanyID != "" {
		f.CompanyID = ai.CompanyID
	} else {
		// 開発時の便宜: 明示指定があれば拾うが、本番では無視される想定
		f.CompanyID = strings.TrimSpace(qv.Get("companyId"))
	}

	f.Status = strings.TrimSpace(qv.Get("status"))
	// 既存互換: brandIds=, brands= のどちらでもカンマ区切り対応
	if v := strings.TrimSpace(qv.Get("brandIds")); v != "" {
		f.BrandIDs = splitCSV(v)
	}
	if v := strings.TrimSpace(qv.Get("brands")); v != "" {
		f.Brands = splitCSV(v)
	}

	// --- Sort ---
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
		// 明示が無ければ updatedAt desc を想定
		sort.Column = "updatedAt"
	}
	switch strings.ToLower(strings.TrimSpace(qv.Get("order"))) {
	case "asc":
		sort.Order = "asc"
	default:
		sort.Order = "desc"
	}

	// --- Page ---
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

// ========================
// Helpers
// ========================

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

// 値 v を [min, max] に丸める
func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
