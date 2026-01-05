// backend\internal\adapters\in\http\console\handler\company_handler.go
package consoleHandler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	companydom "narratives/internal/domain/company"
)

type CompanyHandler struct {
	uc *usecase.CompanyUsecase
}

func NewCompanyHandler(uc *usecase.CompanyUsecase) http.Handler {
	return &CompanyHandler{uc: uc}
}

func (h *CompanyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	// 新規追加: POST /companies
	case r.Method == http.MethodPost && strings.Trim(r.URL.Path, "/") == "companies":
		h.create(w, r)

	// 単一取得: GET /companies/{id}
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/companies/"):
		id := strings.TrimPrefix(r.URL.Path, "/companies/")
		h.get(w, r, id)

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// ---- GET /companies/{id} ----
func (h *CompanyHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	company, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeCompanyErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(company)
}

// ---- POST /companies ----
type createCompanyRequest struct {
	// 空文字は無効として 400 を返す（必要に応じて緩和）
	Name     string `json:"name"`
	Admin    string `json:"admin"`              // 登録者/責任者メールやUIDなど
	IsActive *bool  `json:"isActive,omitempty"` // 省略時 true
	// 監査系（必要に応じてミドルウェアで設定）
	CreatedBy *string `json:"createdBy,omitempty"`
}

func (h *CompanyHandler) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req createCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	name := strings.TrimSpace(req.Name)
	admin := strings.TrimSpace(req.Admin)
	if name == "" || admin == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "name and admin are required"})
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	c := companydom.Company{
		// ID は repo 側で採番でもOK。指定したい場合はここでセット
		Name:      name,
		Admin:     admin,
		IsActive:  isActive,
		CreatedAt: time.Now().UTC(),
	}
	if req.CreatedBy != nil {
		c.CreatedBy = strings.TrimSpace(*req.CreatedBy)
	}

	created, err := h.uc.Create(ctx, c)
	if err != nil {
		writeCompanyErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

// ---- 共通エラーハンドリング ----
func writeCompanyErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case companydom.ErrInvalidID:
		code = http.StatusBadRequest
	case companydom.ErrNotFound:
		code = http.StatusNotFound
	case companydom.ErrConflict:
		code = http.StatusConflict
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
