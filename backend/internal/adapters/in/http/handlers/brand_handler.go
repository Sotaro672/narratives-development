// backend/internal/adapters/in/http/handlers/brand_handler.go
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
	path := r.URL.Path

	switch {
	// 一覧
	case r.Method == http.MethodGet && (path == "/brands" || path == "/brands/"):
		h.list(w, r)

	// 作成: POST /brands
	case r.Method == http.MethodPost && (path == "/brands" || path == "/brands/"):
		h.create(w, r)

	// 単一取得
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/brands/"):
		id := strings.TrimPrefix(path, "/brands/")
		h.get(w, r, id)

	// CORS preflight（必要なら）
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

	// フロントから来る最小DTO
	var in struct {
		CompanyID   string  `json:"companyId"`
		Name        string  `json:"name"`
		Description string  `json:"description"`
		WebsiteURL  string  `json:"websiteUrl"`
		IsActive    *bool   `json:"isActive"`
		ManagerID   *string `json:"manager"`
		CreatedBy   *string `json:"createdBy"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	// 必須の簡易バリデーション
	if strings.TrimSpace(in.CompanyID) == "" || strings.TrimSpace(in.Name) == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "companyId and name are required"})
		return
	}

	isActive := true
	if in.IsActive != nil {
		isActive = *in.IsActive
	}

	// walletAddress は当初空を許容しないドメインなので暫定で "pending" を設定
	now := time.Now().UTC()
	b, err := branddom.New(
		"", // ID は repo 側で採番（FS）
		in.CompanyID,
		in.Name,
		in.Description,
		"pending",     // walletAddress
		in.WebsiteURL, // websiteUrl
		isActive,      // isActive
		in.ManagerID,  // manager
		in.CreatedBy,  // createdBy
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
	if v := strings.TrimSpace(q.Get("companyId")); v != "" {
		f.CompanyID = &v
	}
	if v := strings.TrimSpace(q.Get("managerId")); v != "" {
		f.ManagerID = &v
	}
	if v := strings.TrimSpace(q.Get("walletAddress")); v != "" {
		f.WalletAddress = &v
	}
	if v := strings.TrimSpace(q.Get("isActive")); v != "" {
		if v == "true" {
			b := true
			f.IsActive = &b
		} else if v == "false" {
			b := false
			f.IsActive = &b
		}
	}
	if v := strings.TrimSpace(q.Get("q")); v != "" {
		f.SearchQuery = v
	}

	column := strings.TrimSpace(q.Get("column"))
	if column == "" {
		column = "created_at"
	}

	orderStr := strings.ToLower(strings.TrimSpace(q.Get("order")))
	var order branddom.SortOrder
	switch orderStr {
	case "asc":
		order = branddom.SortAsc
	case "desc":
		order = branddom.SortDesc
	default:
		order = branddom.SortDesc
	}
	sort := branddom.Sort{Column: column, Order: order}

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

	result, err := h.uc.List(ctx, f, sort, p)
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
