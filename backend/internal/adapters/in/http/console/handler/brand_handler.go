// backend\internal\adapters\in\http\console\handler\brand_handler.go
package consoleHandler

import (
	"encoding/json"
	"net/http"
	"time"

	shared "narratives/internal/adapters/in/http/shared"
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

	// ✅ 末尾スラッシュの“吸収（TrimSuffix）”はしない
	// - 方針: canonicalize したい場合は redirect（308）
	// - “拒否”にしたい場合は RejectTrailingSlash を使う
	if shared.RedirectTrailingSlash(w, r) {
		return
	}

	path := r.URL.Path

	switch {

	// 一覧（GET /brands）
	case r.Method == http.MethodGet && path == "/brands":
		h.list(w, r)

	// 作成（POST /brands）
	case r.Method == http.MethodPost && path == "/brands":
		h.create(w, r)

	// 単一取得（GET /brands/:id）
	case r.Method == http.MethodGet && len(path) > len("/brands/") && path[:len("/brands/")] == "/brands/":
		id := path[len("/brands/"):]
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

	// ✅ Trimで吸収しない。前後空白がある入力は拒否。
	vid, err := shared.StrictID(id, "id")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	brand, err := h.uc.GetByID(ctx, vid)
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

	// ✅ Trimで吸収しない。前後空白がある入力は拒否。
	companyID, err := shared.StrictRequired(in.CompanyID, "companyId")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "companyId is required"})
		return
	}
	name, err := shared.StrictRequired(in.Name, "name")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "name is required"})
		return
	}

	// optional fields: reject if present but has outer/control whitespace
	description := in.Description
	if description != "" {
		if shared.HasOuterWhitespace(description) || shared.HasControlWhitespace(description) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "description must not have leading/trailing whitespace or tab/newline"})
			return
		}
	}
	websiteURL := in.WebsiteURL
	if websiteURL != "" {
		if shared.HasOuterWhitespace(websiteURL) || shared.HasControlWhitespace(websiteURL) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "websiteUrl must not have leading/trailing whitespace or tab/newline"})
			return
		}
	}

	var managerID *string
	if in.ManagerID != nil {
		v, err := shared.StrictRequired(*in.ManagerID, "managerId")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid managerId"})
			return
		}
		managerID = &v
	}

	var createdBy *string
	if in.CreatedBy != nil {
		v, err := shared.StrictRequired(*in.CreatedBy, "createdBy")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid createdBy"})
			return
		}
		createdBy = &v
	}

	isActive := true
	if in.IsActive != nil {
		isActive = *in.IsActive
	}

	now := time.Now().UTC()
	b, err := branddom.New(
		"",
		companyID,
		name,
		description,
		"",
		websiteURL,
		isActive,
		managerID,
		createdBy,
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

	if raw := q.Get("managerId"); raw != "" {
		v, err := shared.StrictRequired(raw, "managerId")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid managerId"})
			return
		}
		f.ManagerID = &v
	}

	if raw := q.Get("walletAddress"); raw != "" {
		v, err := shared.StrictRequired(raw, "walletAddress")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid walletAddress"})
			return
		}
		f.WalletAddress = &v
	}

	if raw := q.Get("isActive"); raw != "" {
		b, ok, err := shared.StrictBoolParam(raw, "isActive")
		if err != nil || !ok {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid isActive"})
			return
		}
		f.IsActive = &b
	}

	if raw := q.Get("q"); raw != "" {
		v, err := shared.StrictRequired(raw, "q")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid q"})
			return
		}
		f.SearchQuery = v
	}

	pageNum, err := shared.StrictPositiveIntParam(q.Get("page"), "page", 1)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid page"})
		return
	}

	perPage, err := shared.StrictPositiveIntParam(q.Get("perPage"), "perPage", 50)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid perPage"})
		return
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
