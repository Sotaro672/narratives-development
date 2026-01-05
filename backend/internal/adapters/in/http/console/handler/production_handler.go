// backend\internal\adapters\in\http\console\handler\production_handler.go
package consoleHandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	productionapp "narratives/internal/application/production"
	companyquery "narratives/internal/application/query"
	productbpdom "narratives/internal/domain/productBlueprint"
	productiondom "narratives/internal/domain/production"
)

// ProductionHandler は /productions 関連のエンドポイントを担当します。
// 方針:
// - GET /productions (一覧) は CompanyProductionQueryService（company境界付き）
// - CRUD（GET /{id}, POST, PUT, DELETE）は ProductionUsecase（従来通り）
type ProductionHandler struct {
	query *companyquery.CompanyProductionQueryService
	uc    *productionapp.ProductionUsecase
}

func NewProductionHandler(
	companyProductionQueryService *companyquery.CompanyProductionQueryService,
	uc *productionapp.ProductionUsecase,
) http.Handler {
	return &ProductionHandler{
		query: companyProductionQueryService,
		uc:    uc,
	}
}

func (h *ProductionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {

	// GET /productions （一覧）
	case r.Method == http.MethodGet && r.URL.Path == "/productions":
		h.list(w, r)

	// GET /productions/{id}
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/productions/"):
		id := strings.TrimPrefix(r.URL.Path, "/productions/")
		h.get(w, r, id)

	// POST /productions
	case r.Method == http.MethodPost && r.URL.Path == "/productions":
		h.post(w, r)

	// PUT /productions/{id}（UPDATE）
	case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/productions/"):
		id := strings.TrimPrefix(r.URL.Path, "/productions/")
		h.update(w, r, id)

	// DELETE /productions/{id}
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/productions/"):
		id := strings.TrimPrefix(r.URL.Path, "/productions/")
		h.delete(w, r, id)

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// ------------------------------------------------------------
// GET /productions （一覧）
// ------------------------------------------------------------
func (h *ProductionHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.query == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "query service is nil"})
		return
	}

	// ★ QueryService（company境界付き）を使用
	rows, err := h.query.ListProductionsWithAssigneeName(ctx)
	if err != nil {
		writeProductionErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(rows)
}

// ------------------------------------------------------------
// GET /productions/{id}
// ------------------------------------------------------------
func (h *ProductionHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	p, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeProductionErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(p)
}

// ------------------------------------------------------------
// POST /productions（CREATE）
// ------------------------------------------------------------
func (h *ProductionHandler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	defer r.Body.Close()

	if h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	var req productiondom.Production
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	p, err := h.uc.Create(ctx, req)
	if err != nil {
		writeProductionErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(p)
}

// ------------------------------------------------------------
// PUT /productions/{id}（UPDATE）
// ------------------------------------------------------------
func (h *ProductionHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	defer r.Body.Close()

	if h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	var req productiondom.Production
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	// パスの ID を優先
	req.ID = id

	p, err := h.uc.Update(ctx, id, req)
	if err != nil {
		writeProductionErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(p)
}

// ------------------------------------------------------------
// DELETE /productions/{id}
// ------------------------------------------------------------
func (h *ProductionHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	if err := h.uc.Delete(ctx, id); err != nil {
		writeProductionErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ------------------------------------------------------------
// エラーを JSON で返す共通処理
// ------------------------------------------------------------
func writeProductionErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	// Production domain errors
	if errors.Is(err, productiondom.ErrInvalidID) {
		code = http.StatusBadRequest
	} else if errors.Is(err, productiondom.ErrNotFound) {
		code = http.StatusNotFound

		// ProductBlueprint domain errors（companyId 無しなど、QueryService 側で起きる）
	} else if errors.Is(err, productbpdom.ErrInvalidCompanyID) {
		code = http.StatusBadRequest
	} else if errors.Is(err, productbpdom.ErrInvalidID) {
		code = http.StatusBadRequest
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
