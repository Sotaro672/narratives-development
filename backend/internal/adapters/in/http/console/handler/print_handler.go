package consoleHandler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	resolver "narratives/internal/application/resolver"
	usecase "narratives/internal/application/usecase"
	productdom "narratives/internal/domain/product"
)

type PrintHandler struct {
	uc *usecase.PrintUsecase

	productionUC *usecase.ProductionUsecase

	modelUC *usecase.ModelUsecase

	nameResolver *resolver.NameResolver
}

func NewPrintHandler(
	uc *usecase.PrintUsecase,
	productionUC *usecase.ProductionUsecase,
	modelUC *usecase.ModelUsecase,
	nameResolver *resolver.NameResolver,
) http.Handler {
	return &PrintHandler{
		uc:           uc,
		productionUC: productionUC,
		modelUC:      modelUC,
		nameResolver: nameResolver,
	}
}

func (h *PrintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/inspector/products/"):
		id := strings.TrimPrefix(r.URL.Path, "/inspector/products/")
		h.get(w, r, id)
		return

	case r.Method == http.MethodPost && r.URL.Path == "/products/print-logs":
		h.createPrintLog(w, r)
		return

	case r.Method == http.MethodGet && r.URL.Path == "/products/print-logs":
		productionID := r.URL.Query().Get("productionId")
		if productionID == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "productionId query parameter is required",
			})
			return
		}

		h.listPrintLogsByProductionID(w, r, productionID)
		return

	case r.Method == http.MethodPost && r.URL.Path == "/products/inspections":
		h.createInspectionBatch(w, r)
		return

	case r.Method == http.MethodGet && r.URL.Path == "/products":
		productionID := r.URL.Query().Get("productionId")
		if productionID == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "productionId query parameter is required",
			})
			return
		}

		h.listByProductionID(w, r, productionID)
		return

	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/products/"):
		id := strings.TrimPrefix(r.URL.Path, "/products/")
		h.get(w, r, id)
		return

	case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/products/"):
		id := strings.TrimPrefix(r.URL.Path, "/products/")
		h.update(w, r, id)
		return

	case r.Method == http.MethodPost && r.URL.Path == "/products":
		h.create(w, r)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

func (h *PrintHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.Trim(id, " \t\r\n/")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	p, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeProductErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(p)
}

func (h *PrintHandler) listByProductionID(w http.ResponseWriter, r *http.Request, productionID string) {
	ctx := r.Context()

	productionID = strings.Trim(productionID, " \t\r\n/")
	if productionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid productionId"})
		return
	}

	list, err := h.uc.ListByProductionID(ctx, productionID)
	if err != nil {
		writeProductErr(w, err)
		return
	}

	type productWithModelNumber struct {
		ID           string `json:"id"`
		ModelID      string `json:"modelId"`
		ProductionID string `json:"productionId"`
		ModelNumber  string `json:"modelNumber"`
	}

	out := make([]productWithModelNumber, 0, len(list))

	for _, p := range list {
		modelID := strings.Trim(p.ModelID, " \t\r\n/")
		modelNumber := ""

		if modelID != "" && h.nameResolver != nil {
			mn := h.nameResolver.ResolveModelNumber(ctx, modelID)
			modelNumber = strings.Trim(mn, " \t\r\n/")
		}

		out = append(out, productWithModelNumber{
			ID:           p.ID,
			ModelID:      p.ModelID,
			ProductionID: p.ProductionID,
			ModelNumber:  modelNumber,
		})
	}

	_ = json.NewEncoder(w).Encode(out)
}

func (h *PrintHandler) listPrintLogsByProductionID(w http.ResponseWriter, r *http.Request, productionID string) {
	ctx := r.Context()

	productionID = strings.Trim(productionID, " \t\r\n/")
	if productionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid productionId"})
		return
	}

	logs, err := h.uc.ListPrintLogsByProductionID(ctx, productionID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(logs)
}

func (h *PrintHandler) createPrintLog(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		ProductionID string `json:"productionId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	productionID := strings.Trim(req.ProductionID, " \t\r\n/")
	if productionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "productionId is required"})
		return
	}

	pl, err := h.uc.CreatePrintLogForProduction(ctx, productionID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(pl)
}

func (h *PrintHandler) createInspectionBatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		ProductionID string `json:"productionId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	productionID := strings.Trim(req.ProductionID, " \t\r\n/")
	if productionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "productionId is required"})
		return
	}

	batch, err := h.uc.CreateInspectionBatchForProduction(ctx, productionID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(batch)
}

func (h *PrintHandler) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		ModelID      string    `json:"modelId"`
		ProductionID string    `json:"productionId"`
		PrintedAt    time.Time `json:"printedAt"`
		PrintedBy    *string   `json:"printedBy"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	req.ModelID = strings.Trim(req.ModelID, " \t\r\n/")
	req.ProductionID = strings.Trim(req.ProductionID, " \t\r\n/")

	if req.ModelID == "" || req.ProductionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "modelId and productionId are required"})
		return
	}

	if req.PrintedAt.IsZero() {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "printedAt is required"})
		return
	}

	p := productdom.Product{
		ModelID:          req.ModelID,
		ProductionID:     req.ProductionID,
		InspectionResult: productdom.InspectionNotYet,
		PrintedAt:        &req.PrintedAt,
		InspectedAt:      nil,
		InspectedBy:      nil,
	}

	created, err := h.uc.Create(ctx, p)
	if err != nil {
		writeProductErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(created)
}

func (h *PrintHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.Trim(id, " \t\r\n/")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	var req struct {
		InspectionResult productdom.InspectionResult `json:"inspectionResult"`
		InspectedAt      *time.Time                  `json:"inspectedAt"`
		InspectedBy      *string                     `json:"inspectedBy"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	var p productdom.Product
	p.InspectionResult = req.InspectionResult
	p.InspectedAt = req.InspectedAt
	p.InspectedBy = req.InspectedBy

	updated, err := h.uc.Update(ctx, id, p)
	if err != nil {
		writeProductErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(updated)
}

func writeProductErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch err {
	case productdom.ErrInvalidID:
		code = http.StatusBadRequest
	case productdom.ErrNotFound:
		code = http.StatusNotFound
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
