// backend/internal/adapters/in/http/console/handler/production/production_handler.go
package productionHandler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
	productionapp "narratives/internal/application/production"
	companyquery "narratives/internal/application/query/console"
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
