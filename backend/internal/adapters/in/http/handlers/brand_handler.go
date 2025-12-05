package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
)

type BrandHandler struct {
	uc *usecase.BrandUsecase
}

func NewBrandHandler(uc *usecase.BrandUsecase) http.Handler {
	return &BrandHandler{uc: uc}
}

func (h *BrandHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// ★★★★★ 重要：末尾スラッシュを強制的に除去して正規化 ★★★★★
	path := strings.TrimSuffix(r.URL.Path, "/")

	switch {

	// 一覧（GET /brands）
	case r.Method == http.MethodGet && path == "/brands":
		h.list(w, r)

	// 作成（POST /brands）
	case r.Method == http.MethodPost && path == "/brands":
		h.create(w, r)

	// 単一取得（GET /brands/:id）
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/brands/"):
		id := strings.TrimPrefix(path, "/brands/")
		h.get(w, r, id)

	// OPTIONS (CORS)
	case r.Method == http.MethodOptions:
		w.WriteHeader(http.StatusNoContent)

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /brands/{id}
func (h *BrandHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}
	brand, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeBrandErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(brand)
}

// POST /brands
func (h *BrandHandler) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// ★ フロントと合わせて JSON フィールド名を managerId に統一
	var in struct {
		CompanyID   string  `json:"companyId"`
		Name        string  `json:"name"`
		Description string  `json:"description"`
		WebsiteURL  string  `json:"websiteUrl"`
		IsActive    *bool   `json:"isActive"`
		ManagerID   *string `json:"managerId"`
		CreatedBy   *string `json:"createdBy"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	if strings.TrimSpace(in.CompanyID) == "" || strings.TrimSpace(in.Name) == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "companyId and name are required"})
		return
	}

	isActive := true
	if in.IsActive != nil {
		isActive = *in.IsActive
	}

	now := time.Now().UTC()
	b, err := branddom.New(
		"",
		in.CompanyID,
		in.Name,
		in.Description,
		"pending",
		in.WebsiteURL,
		isActive,
		in.ManagerID,
		in.CreatedBy,
		now,
	)
	if err != nil {
		writeBrandErr(w, err)
		return
	}

	created, err := h.uc.Create(ctx, b)
	if err != nil {
		writeBrandErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

// GET /brands?...（一覧）
func (h *BrandHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	var f branddom.Filter

	// ✅ companyId はクエリからは受け取らず、Usecase 側で
	//    companyIDFromContext(ctx) によって必ず上書きされる前提。
	if v := strings.TrimSpace(q.Get("managerId")); v != "" {
		f.ManagerID = &v
	}
	if v := strings.TrimSpace(q.Get("walletAddress")); v != "" {
		f.WalletAddress = &v
	}
	if v := strings.TrimSpace(q.Get("isActive")); v != "" {
		switch v {
		case "true":
			b := true
			f.IsActive = &b
		case "false":
			b := false
			f.IsActive = &b
		}
	}
	if v := strings.TrimSpace(q.Get("q")); v != "" {
		f.SearchQuery = v
	}

	// ★ sort / order は廃止 → デフォルトは Repository / Usecase 側の実装に任せる

	pageNum := 1
	if v := strings.TrimSpace(q.Get("page")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			pageNum = n
		}
	}
	perPage := 50
	if v := strings.TrimSpace(q.Get("perPage")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			perPage = n
		}
	}
	p := branddom.Page{Number: pageNum, PerPage: perPage}

	// Usecase.List(ctx, filter, page) に変更（Sort 渡しは廃止）
	result, err := h.uc.List(ctx, f, p)
	if err != nil {
		writeBrandErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(result)
}

func writeBrandErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case branddom.ErrInvalidID:
		code = http.StatusBadRequest
	case branddom.ErrNotFound:
		code = http.StatusNotFound
	case branddom.ErrConflict:
		code = http.StatusConflict
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
