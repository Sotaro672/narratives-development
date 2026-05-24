package consoleHandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	companyquery "narratives/internal/application/query/console"
	usecase "narratives/internal/application/usecase"
	productbpdom "narratives/internal/domain/productBlueprint"
	productiondom "narratives/internal/domain/production"
)

type ProductionHandler struct {
	query *companyquery.CompanyProductionQueryService
	uc    *usecase.ProductionUsecase
}

func NewProductionHandler(
	companyProductionQueryService *companyquery.CompanyProductionQueryService,
	uc *usecase.ProductionUsecase,
) http.Handler {
	return &ProductionHandler{
		query: companyProductionQueryService,
		uc:    uc,
	}
}

func (h *ProductionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/productions":
		h.listProduction(w, r)

	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/productions/"):
		id := strings.TrimPrefix(r.URL.Path, "/productions/")
		h.getProduction(w, r, id)

	case r.Method == http.MethodPost && r.URL.Path == "/productions":
		h.postProduction(w, r)

	case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/productions/"):
		id := strings.TrimPrefix(r.URL.Path, "/productions/")
		h.updateProduction(w, r, id)

	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/productions/"):
		id := strings.TrimPrefix(r.URL.Path, "/productions/")
		h.deleteProduction(w, r, id)

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

func (h *ProductionHandler) listProduction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.query == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "query service is nil"})
		return
	}

	rows, err := h.query.ListProductionsWithAssigneeName(ctx)
	if err != nil {
		writeProductionErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(rows)
}

func (h *ProductionHandler) getProduction(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

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

func (h *ProductionHandler) postProduction(w http.ResponseWriter, r *http.Request) {
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

func (h *ProductionHandler) updateProduction(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	defer r.Body.Close()

	if h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

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

	req.ID = id

	p, err := h.uc.Update(ctx, id, req)
	if err != nil {
		writeProductionErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(p)
}

func (h *ProductionHandler) deleteProduction(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

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

func writeProductionErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	if errors.Is(err, productiondom.ErrInvalidID) {
		code = http.StatusBadRequest
	} else if errors.Is(err, productiondom.ErrNotFound) {
		code = http.StatusNotFound
	} else if errors.Is(err, productbpdom.ErrInvalidCompanyID) {
		code = http.StatusBadRequest
	} else if errors.Is(err, productbpdom.ErrInvalidID) {
		code = http.StatusBadRequest
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
