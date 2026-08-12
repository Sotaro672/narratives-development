// backend/internal/adapters/in/http/console/handler/mint_handler.go
package consoleHandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	companyquery "narratives/internal/application/query/console"
	querydto "narratives/internal/application/query/console/dto"
	mintapp "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	inspectiondom "narratives/internal/domain/inspection"
	mintdom "narratives/internal/domain/mint"
	pbpdom "narratives/internal/domain/productBlueprint"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

type MintRequestQueryService interface {
	ListMintRequestManagementRows(ctx context.Context, input querydto.ListMintRequestManagementRowsInput) ([]querydto.ProductionInspectionMintDTO, error)
	GetMintRequestDetail(ctx context.Context, productionID string) (*querydto.MintRequestDetailDTO, error)
	GetProductBlueprintForMint(ctx context.Context, productBlueprintID string) (*querydto.MintProductBlueprintDTO, error)
	ListBrandsForMint(ctx context.Context) (branddom.PageResult[branddom.Brand], error)
	ListTokenBlueprintsForMint(ctx context.Context, input querydto.ListTokenBlueprintsForMintInput) ([]querydto.TokenBlueprintForMintDTO, error)
}

type MintFundingEstimateQueryService interface {
	GetMintFundingEstimate(ctx context.Context, input companyquery.GetMintFundingEstimateInput) (*companyquery.MintFundingEstimateResult, error)
}

type MintHandler struct {
	mintUC              *mintapp.MintUsecase
	mintRequestQS       MintRequestQueryService
	mintFundingEstimate MintFundingEstimateQueryService
}

func NewMintHandler(mintUC *mintapp.MintUsecase, mintRequestQS MintRequestQueryService, mintFundingEstimate MintFundingEstimateQueryService) http.Handler {
	return &MintHandler{mintUC: mintUC, mintRequestQS: mintRequestQS, mintFundingEstimate: mintFundingEstimate}
}

func (h *MintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/mint/funding-estimate":
		h.getMintFundingEstimate(w, r)
		return
	case r.Method == http.MethodGet && r.URL.Path == "/mint/requests":
		h.listMintRequestsByCurrentCompany(w, r)
		return
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/mint/inspections/") && strings.HasSuffix(r.URL.Path, "/request"):
		h.updateRequestInfo(w, r)
		return
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/mint/inspections/"):
		h.getMintRequestDetailByProductionID(w, r)
		return
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/mint/product_blueprints/"):
		h.getProductBlueprintByID(w, r)
		return
	case r.Method == http.MethodGet && r.URL.Path == "/mint/brands":
		h.listBrandsForCurrentCompany(w, r)
		return
	case r.Method == http.MethodGet && r.URL.Path == "/mint/token_blueprints":
		h.listTokenBlueprintsByBrand(w, r)
		return
	default:
		http.NotFound(w, r)
	}
}

type updateMintRequestInfoRequest struct {
	TokenBlueprintID  string  `json:"tokenBlueprintId"`
	ScheduledBurnDate *string `json:"scheduledBurnDate"`
}

type mintQueuedResponse struct {
	MintRequestID string `json:"mintRequestId"`
	ProductionID  string `json:"productionId"`
	Status        string `json:"status"`
	Message       string `json:"message"`
}

func (h *MintHandler) getMintFundingEstimate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.mintFundingEstimate == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "mint funding estimate query is not configured"})
		return
	}

	productionID := strings.TrimSpace(r.URL.Query().Get("productionId"))
	if productionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "productionId is required"})
		return
	}

	tokenBlueprintID := strings.TrimSpace(r.URL.Query().Get("tokenBlueprintId"))
	if tokenBlueprintID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tokenBlueprintId is required"})
		return
	}

	result, err := h.mintFundingEstimate.GetMintFundingEstimate(ctx, companyquery.GetMintFundingEstimateInput{
		ProductionID:     productionID,
		TokenBlueprintID: tokenBlueprintID,
	})
	if err != nil {
		switch {
		case errors.Is(err, mintapp.ErrCompanyIDMissing):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "companyId is missing"})
			return
		case errors.Is(err, companyquery.ErrMintFundingEstimateInvalidInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		case errors.Is(err, companyquery.ErrMintFundingEstimateForbidden):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
			return
		case errors.Is(err, tbdom.ErrNotFound), errors.Is(err, branddom.ErrNotFound), errors.Is(err, inspectiondom.ErrNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		case errors.Is(err, companyquery.ErrMintFundingEstimateNoPassedProducts), errors.Is(err, companyquery.ErrMintFundingEstimateBrandWalletMissing):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
			return
		case errors.Is(err, companyquery.ErrMintFundingEstimateQueryNotConfigured):
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "mint funding estimate query is not configured"})
			return
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}

	if result == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "mint funding estimate result is empty"})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *MintHandler) updateRequestInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.mintUC == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "mint usecase is not configured"})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/mint/inspections/")
	path = strings.TrimSuffix(path, "/request")
	productionID := strings.Trim(path, "/")

	if productionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "productionId is empty"})
		return
	}
	if strings.Contains(productionID, "/") {
		http.NotFound(w, r)
		return
	}

	var req updateMintRequestInfoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	tokenBlueprintID := strings.TrimSpace(req.TokenBlueprintID)
	if tokenBlueprintID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tokenBlueprintId is required"})
		return
	}

	if err := h.mintUC.UpdateRequestInfo(ctx, productionID, tokenBlueprintID, req.ScheduledBurnDate); err != nil {
		if errors.Is(err, mintapp.ErrCompanyIDMissing) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "companyId is missing"})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusAccepted, mintQueuedResponse{
		MintRequestID: productionID,
		ProductionID:  productionID,
		Status:        "QUEUED",
		Message:       "mint request accepted. product mint tasks will be processed one by one.",
	})
}

func (h *MintHandler) getMintRequestDetailByProductionID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.mintRequestQS == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "mintRequest query service is not configured"})
		return
	}

	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/mint/inspections/"), "/")
	if path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "productionId is empty"})
		return
	}
	if strings.Contains(path, "/") {
		http.NotFound(w, r)
		return
	}

	detail, err := h.mintRequestQS.GetMintRequestDetail(ctx, path)
	if err != nil {
		if errors.Is(err, mintapp.ErrCompanyIDMissing) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "companyId is missing"})
			return
		}

		if errors.Is(err, inspectiondom.ErrNotFound) ||
			errors.Is(err, mintdom.ErrNotFound) ||
			strings.Contains(strings.ToLower(err.Error()), "not found") {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "mint request detail not found"})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, detail)
}

func (h *MintHandler) listMintRequestsByCurrentCompany(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.mintRequestQS == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "mintRequest query service is not configured"})
		return
	}

	productionIDs := parseCommaSeparatedIDs(r.URL.Query().Get("productionIds"))
	rows, err := h.mintRequestQS.ListMintRequestManagementRows(ctx, querydto.ListMintRequestManagementRowsInput{
		ProductionIDs: productionIDs,
	})
	if err != nil {
		if errors.Is(err, mintapp.ErrCompanyIDMissing) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "companyId is missing"})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, rows)
}

func (h *MintHandler) getProductBlueprintByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.mintRequestQS == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "mintRequest query service is not configured"})
		return
	}

	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/mint/product_blueprints/"), "/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "productBlueprintID is empty"})
		return
	}
	if strings.Contains(id, "/") {
		http.NotFound(w, r)
		return
	}

	resp, err := h.mintRequestQS.GetProductBlueprintForMint(ctx, id)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, pbpdom.ErrNotFound) {
			statusCode = http.StatusNotFound
		}

		writeJSON(w, statusCode, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *MintHandler) listBrandsForCurrentCompany(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.mintRequestQS == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "mintRequest query service is not configured"})
		return
	}

	result, err := h.mintRequestQS.ListBrandsForMint(ctx)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, mintapp.ErrCompanyIDMissing) {
			statusCode = http.StatusBadRequest
		}

		writeJSON(w, statusCode, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *MintHandler) listTokenBlueprintsByBrand(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.mintRequestQS == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "mintRequest query service is not configured"})
		return
	}

	brandID := r.URL.Query().Get("brandId")
	if brandID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "brandId is required"})
		return
	}

	pageNumber, perPage := 1, 100
	if pageParam := r.URL.Query().Get("page"); pageParam != "" {
		if n, err := strconv.Atoi(pageParam); err == nil && n > 0 {
			pageNumber = n
		}
	}
	if perPageParam := r.URL.Query().Get("perPage"); perPageParam != "" {
		if n, err := strconv.Atoi(perPageParam); err == nil && n > 0 {
			perPage = n
		}
	}

	items, err := h.mintRequestQS.ListTokenBlueprintsForMint(ctx, querydto.ListTokenBlueprintsForMintInput{
		BrandID: brandID,
		Page:    pageNumber,
		PerPage: perPage,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func parseCommaSeparatedIDs(raw string) []string {
	if raw == "" {
		return []string{}
	}

	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))

	for _, p := range parts {
		id := strings.TrimSpace(p)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}

		seen[id] = struct{}{}
		out = append(out, id)
	}

	return out
}
