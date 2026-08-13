// backend/internal/adapters/in/http/console/handler/print_handler.go
package consoleHandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	consolequery "narratives/internal/application/query/console"
	usecase "narratives/internal/application/usecase"
	printdom "narratives/internal/domain/print"
	productdom "narratives/internal/domain/product"
)

type PrintHandler struct {
	uc    *usecase.PrintUsecase
	query *consolequery.PrintQueryService
}

func NewPrintHandler(
	uc *usecase.PrintUsecase,
	query *consolequery.PrintQueryService,
) http.Handler {
	return &PrintHandler{
		uc:    uc,
		query: query,
	}
}

func (h *PrintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/products/print-logs":
		h.createPrintLog(w, r)
		return

	case r.Method == http.MethodGet && r.URL.Path == "/products/print-logs":
		productionID := strings.Trim(r.URL.Query().Get("productionId"), " \t\r\n/")
		if productionID == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "productionId query parameter is required",
			})
			return
		}

		h.listPrintLogsByProductionID(w, r, productionID)
		return

	case r.Method == http.MethodGet && r.URL.Path == "/products":
		productionID := strings.Trim(r.URL.Query().Get("productionId"), " \t\r\n/")
		if productionID == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "productionId query parameter is required",
			})
			return
		}

		h.listByProductionID(w, r, productionID)
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

func (h *PrintHandler) listByProductionID(
	w http.ResponseWriter,
	r *http.Request,
	productionID string,
) {
	if h.query == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "print query service is not configured",
		})
		return
	}

	list, err := h.query.ListProductsByProductionID(r.Context(), productionID)
	if err != nil {
		writeProductErr(w, err)
		return
	}

	if list == nil {
		_ = json.NewEncoder(w).Encode([]any{})
		return
	}

	_ = json.NewEncoder(w).Encode(list)
}

func (h *PrintHandler) listPrintLogsByProductionID(
	w http.ResponseWriter,
	r *http.Request,
	productionID string,
) {
	if h.query == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "print query service is not configured",
		})
		return
	}

	logs, err := h.query.ListPrintLogsByProductionID(
		r.Context(),
		productionID,
	)
	if err != nil {
		if errors.Is(err, printdom.ErrNotFound) {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]any{})
			return
		}

		writeProductErr(w, err)
		return
	}

	if logs == nil {
		_ = json.NewEncoder(w).Encode([]any{})
		return
	}

	_ = json.NewEncoder(w).Encode(logs)
}

func (h *PrintHandler) createPrintLog(w http.ResponseWriter, r *http.Request) {
	if h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "print usecase is not configured",
		})
		return
	}

	if h.query == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "print query service is not configured",
		})
		return
	}

	var req struct {
		ProductionID string `json:"productionId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid json",
		})
		return
	}

	productionID := strings.Trim(req.ProductionID, " \t\r\n/")
	if productionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "productionId is required",
		})
		return
	}

	ctx := r.Context()

	if _, err := h.uc.CreatePrintLogForProduction(ctx, productionID); err != nil {
		writeProductErr(w, err)
		return
	}

	logs, err := h.query.ListPrintLogsByProductionID(ctx, productionID)
	if err != nil {
		writeProductErr(w, err)
		return
	}

	if len(logs) == 0 {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "created print log could not be loaded",
		})
		return
	}

	_ = json.NewEncoder(w).Encode(logs[0])
}

func (h *PrintHandler) create(w http.ResponseWriter, r *http.Request) {
	if h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "print usecase is not configured",
		})
		return
	}

	var req struct {
		ModelID      string    `json:"modelId"`
		ProductionID string    `json:"productionId"`
		PrintedAt    time.Time `json:"printedAt"`
		PrintedBy    *string   `json:"printedBy"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid json",
		})
		return
	}

	req.ModelID = strings.Trim(req.ModelID, " \t\r\n/")
	req.ProductionID = strings.Trim(req.ProductionID, " \t\r\n/")

	if req.ModelID == "" || req.ProductionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "modelId and productionId are required",
		})
		return
	}

	if req.PrintedAt.IsZero() {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "printedAt is required",
		})
		return
	}

	printedAt := req.PrintedAt.UTC()

	p := productdom.Product{
		ModelID:          req.ModelID,
		ProductionID:     req.ProductionID,
		InspectionResult: productdom.InspectionNotYet,
		PrintedAt:        &printedAt,
		InspectedAt:      nil,
		InspectedBy:      nil,
	}

	created, err := h.uc.Create(r.Context(), p)
	if err != nil {
		writeProductErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(created)
}

func writeProductErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, productdom.ErrInvalidID):
		code = http.StatusBadRequest
	case errors.Is(err, productdom.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, productdom.ErrConflict):
		code = http.StatusConflict
	case errors.Is(err, printdom.ErrInvalidPrintLogProductionID):
		code = http.StatusBadRequest
	case errors.Is(err, printdom.ErrNotFound):
		code = http.StatusNotFound
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": err.Error(),
	})
}
