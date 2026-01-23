// backend\internal\adapters\in\http\console\handler\production_handler.go
package consoleHandler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
	productionapp "narratives/internal/application/production"
	companyquery "narratives/internal/application/query/console"
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

	// ✅ debug: request basics + companyId(from middleware ctxKey)
	if cid, ok := middleware.CompanyID(r); ok {
		log.Printf("[productions] request method=%s path=%s companyId=%q", r.Method, r.URL.Path, cid)
	} else {
		log.Printf("[productions] request method=%s path=%s companyId=<missing>", r.Method, r.URL.Path)
	}

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
		log.Printf("[productions] list: query service is nil")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "query service is nil"})
		return
	}

	// ✅ debug: log companyId again at handler entry (to catch ctx mismatch early)
	if cid, ok := middleware.CompanyID(r); ok {
		log.Printf("[productions] list: start companyId=%q", cid)
	} else {
		log.Printf("[productions] list: start companyId=<missing>")
	}

	// ★ QueryService（company境界付き）を使用
	rows, err := h.query.ListProductionsWithAssigneeName(ctx)
	if err != nil {
		// ✅ debug: classify error
		log.Printf("[productions] list: query error=%v", err)
		writeProductionErr(w, err)
		return
	}

	log.Printf("[productions] list: success rows=%d", len(rows))
	_ = json.NewEncoder(w).Encode(rows)
}

// ------------------------------------------------------------
// GET /productions/{id}
// ------------------------------------------------------------
func (h *ProductionHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if h.uc == nil {
		log.Printf("[productions] get: usecase is nil")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	id = strings.TrimSpace(id)
	if id == "" {
		log.Printf("[productions] get: invalid id (empty)")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	log.Printf("[productions] get: start id=%q", id)

	p, err := h.uc.GetByID(ctx, id)
	if err != nil {
		log.Printf("[productions] get: error id=%q err=%v", id, err)
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
		log.Printf("[productions] post: usecase is nil")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	var req productiondom.Production
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[productions] post: invalid json err=%v", err)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	log.Printf("[productions] post: start")

	p, err := h.uc.Create(ctx, req)
	if err != nil {
		log.Printf("[productions] post: error err=%v", err)
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
		log.Printf("[productions] update: usecase is nil")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	id = strings.TrimSpace(id)
	if id == "" {
		log.Printf("[productions] update: invalid id (empty)")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	var req productiondom.Production
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[productions] update: invalid json id=%q err=%v", id, err)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	// パスの ID を優先
	req.ID = id

	log.Printf("[productions] update: start id=%q", id)

	p, err := h.uc.Update(ctx, id, req)
	if err != nil {
		log.Printf("[productions] update: error id=%q err=%v", id, err)
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
		log.Printf("[productions] delete: usecase is nil")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	id = strings.TrimSpace(id)
	if id == "" {
		log.Printf("[productions] delete: invalid id (empty)")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	log.Printf("[productions] delete: start id=%q", id)

	if err := h.uc.Delete(ctx, id); err != nil {
		log.Printf("[productions] delete: error id=%q err=%v", id, err)
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

	// ✅ debug: final error response
	log.Printf("[productions] respond error status=%d err=%v", code, err)

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
