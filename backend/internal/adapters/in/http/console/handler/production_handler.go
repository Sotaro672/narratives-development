// backend/internal/adapters/in/http/console/handler/production_handler.go
package consoleHandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

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

// ========================================
// HTTP request DTOs
// ========================================

type productionModelRequest struct {
	ModelID  string `json:"modelId"`
	Quantity int    `json:"quantity"`
}

type createProductionRequest struct {
	ProductBlueprintID string                   `json:"productBlueprintId"`
	AssigneeID         string                   `json:"assigneeId"`
	Models             []productionModelRequest `json:"models"`

	Printed   *bool      `json:"printed,omitempty"`
	PrintedAt *time.Time `json:"printedAt,omitempty"`

	CreatedBy *string `json:"createdBy,omitempty"`
}

type updateProductionRequest struct {
	AssigneeID string                   `json:"assigneeId"`
	Models     []productionModelRequest `json:"models"`

	Printed   *bool      `json:"printed,omitempty"`
	PrintedAt *time.Time `json:"printedAt,omitempty"`
	PrintedBy *string    `json:"printedBy,omitempty"`

	UpdatedBy *string `json:"updatedBy,omitempty"`
}

func (m productionModelRequest) toCommand() usecase.ModelQuantityCommand {
	return usecase.ModelQuantityCommand{
		ModelID:  m.ModelID,
		Quantity: m.Quantity,
	}
}

func productionModelRequestsToCommands(
	models []productionModelRequest,
) []usecase.ModelQuantityCommand {
	out := make([]usecase.ModelQuantityCommand, 0, len(models))
	for _, m := range models {
		out = append(out, m.toCommand())
	}
	return out
}

func (req createProductionRequest) toCommand() usecase.CreateProductionCommand {
	return usecase.CreateProductionCommand{
		ProductBlueprintID: req.ProductBlueprintID,
		AssigneeID:         req.AssigneeID,
		Models:             productionModelRequestsToCommands(req.Models),
		Printed:            req.Printed,
		PrintedAt:          req.PrintedAt,
		CreatedBy:          req.CreatedBy,
	}
}

func (req updateProductionRequest) toCommand(id string) usecase.UpdateProductionCommand {
	return usecase.UpdateProductionCommand{
		ID:         id,
		AssigneeID: req.AssigneeID,
		Models:     productionModelRequestsToCommands(req.Models),
		Printed:    req.Printed,
		PrintedAt:  req.PrintedAt,
		PrintedBy:  req.PrintedBy,
		UpdatedBy:  req.UpdatedBy,
	}
}

// ========================================
// Router
// ========================================

func (h *ProductionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/productions":
		h.listProduction(w, r)

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

func (h *ProductionHandler) postProduction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	defer r.Body.Close()

	if h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	var req createProductionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	p, err := h.uc.Create(ctx, req.toCommand())
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

	var req updateProductionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	p, err := h.uc.Update(ctx, req.toCommand(id))
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
	} else if errors.Is(err, productiondom.ErrInvalidProductBlueprintID) {
		code = http.StatusBadRequest
	} else if errors.Is(err, productiondom.ErrInvalidAssigneeID) {
		code = http.StatusBadRequest
	} else if errors.Is(err, productiondom.ErrInvalidModels) {
		code = http.StatusBadRequest
	} else if errors.Is(err, productiondom.ErrInvalidModelID) {
		code = http.StatusBadRequest
	} else if errors.Is(err, productiondom.ErrInvalidQuantity) {
		code = http.StatusBadRequest
	} else if errors.Is(err, productiondom.ErrInvalidPrintedAt) {
		code = http.StatusBadRequest
	} else if errors.Is(err, productiondom.ErrInvalidPrintedBy) {
		code = http.StatusBadRequest
	} else if errors.Is(err, productiondom.ErrInvalidCreatedAt) {
		code = http.StatusBadRequest
	} else if errors.Is(err, productiondom.ErrInvalidUpdatedAt) {
		code = http.StatusBadRequest
	} else if errors.Is(err, productiondom.ErrInvalidUpdatedBy) {
		code = http.StatusBadRequest
	} else if errors.Is(err, productiondom.ErrTransition) {
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
