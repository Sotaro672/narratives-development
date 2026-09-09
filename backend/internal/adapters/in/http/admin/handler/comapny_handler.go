// backend/internal/adapters/in/http/admin/handler/company_handler.go

package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	adminquery "narratives/internal/application/query/admin"
	companydom "narratives/internal/domain/company"
	inventorydom "narratives/internal/domain/inventory"
	listdom "narratives/internal/domain/list"
	memberdom "narratives/internal/domain/member"
	productblueprintdom "narratives/internal/domain/productBlueprint"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
)

const (
	adminCompaniesPath            = "/admin/companies"
	adminContractDetailPathSuffix = "/contract-detail"
)

type CompanyListReader interface {
	ListAll(ctx context.Context) ([]companydom.Company, error)
}

type MemberReader interface {
	GetByID(ctx context.Context, id string) (memberdom.Record, error)
}

type ContractDetailReader interface {
	Get(ctx context.Context, companyID string) (adminquery.ContractDetailResult, error)
}

type ContractListReader interface {
	Get(ctx context.Context, companyID string, listID string) (adminquery.ContractListDetailResult, error)
}

type ContractTokenBlueprintReader interface {
	Get(ctx context.Context, companyID string, tokenBlueprintID string) (adminquery.ContractTokenBlueprintDetailResult, error)
}

type ContractProductBlueprintReader interface {
	Get(ctx context.Context, companyID string, productBlueprintID string) (adminquery.ContractProductBlueprintDetailResult, error)
}

type CompanyHandler struct {
	companyRepo                   CompanyListReader
	memberRepo                    MemberReader
	contractDetailQuery           ContractDetailReader
	contractListQuery             ContractListReader
	contractTokenBlueprintQuery   ContractTokenBlueprintReader
	contractProductBlueprintQuery ContractProductBlueprintReader
}

type companyResponse struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	RepresentativeName string `json:"representativeName"`
	IsActive           bool   `json:"isActive"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
}

type companyListResponse struct {
	Items []companyResponse `json:"items"`
}

func NewCompanyHandler(
	companyRepo CompanyListReader,
	memberRepo MemberReader,
	contractDetailQuery ContractDetailReader,
	contractListQuery ContractListReader,
	contractTokenBlueprintQuery ContractTokenBlueprintReader,
	contractProductBlueprintQuery ContractProductBlueprintReader,
) http.Handler {
	return http.HandlerFunc((&CompanyHandler{
		companyRepo:                   companyRepo,
		memberRepo:                    memberRepo,
		contractDetailQuery:           contractDetailQuery,
		contractListQuery:             contractListQuery,
		contractTokenBlueprintQuery:   contractTokenBlueprintQuery,
		contractProductBlueprintQuery: contractProductBlueprintQuery,
	}).handle)
}

func (h *CompanyHandler) handle(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")

	if path == adminCompaniesPath {
		h.handleList(w, r)
		return
	}

	if companyID, ok := parseContractDetailCompanyID(path); ok {
		h.handleContractDetail(w, r, companyID)
		return
	}

	if companyID, listID, ok := parseContractResourcePath(path, "lists"); ok {
		h.handleContractListDetail(w, r, companyID, listID)
		return
	}

	if companyID, tokenBlueprintID, ok := parseContractResourcePath(path, "token-blueprints"); ok {
		h.handleContractTokenBlueprintDetail(w, r, companyID, tokenBlueprintID)
		return
	}

	if companyID, productBlueprintID, ok := parseContractResourcePath(path, "product-blueprints"); ok {
		h.handleContractProductBlueprintDetail(w, r, companyID, productBlueprintID)
		return
	}

	writeJSONError(w, http.StatusNotFound, "company_resource_not_found")
}

func (h *CompanyHandler) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	if h.companyRepo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "company_repository_not_initialized")
		return
	}
	if h.memberRepo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "member_repository_not_initialized")
		return
	}

	companies, err := h.companyRepo.ListAll(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "company_list_failed")
		return
	}

	items := make([]companyResponse, 0, len(companies))
	for _, company := range companies {
		representativeName, err := h.resolveRepresentativeName(r.Context(), company.Admin)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "company_representative_resolve_failed")
			return
		}
		items = append(items, toAdminCompanyResponse(company, representativeName))
	}

	writeJSON(w, http.StatusOK, companyListResponse{Items: items})
}

func (h *CompanyHandler) handleContractDetail(
	w http.ResponseWriter,
	r *http.Request,
	companyID string,
) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	if h.contractDetailQuery == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "contract_detail_query_not_initialized")
		return
	}

	result, err := h.contractDetailQuery.Get(r.Context(), companyID)
	if err != nil {
		switch {
		case errors.Is(err, companydom.ErrNotFound), errors.Is(err, companydom.ErrInvalidID):
			writeJSONError(w, http.StatusNotFound, "company_not_found")
		case errors.Is(err, adminquery.ErrContractDetailQueryNotConfigured):
			writeJSONError(w, http.StatusServiceUnavailable, "contract_detail_query_not_initialized")
		default:
			writeJSONError(w, http.StatusInternalServerError, "contract_detail_get_failed")
		}
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *CompanyHandler) handleContractListDetail(
	w http.ResponseWriter,
	r *http.Request,
	companyID string,
	listID string,
) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	if h.contractListQuery == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "contract_list_query_not_initialized")
		return
	}

	result, err := h.contractListQuery.Get(r.Context(), companyID, listID)
	if err != nil {
		switch {
		case errors.Is(err, companydom.ErrNotFound), errors.Is(err, companydom.ErrInvalidID):
			writeJSONError(w, http.StatusNotFound, "company_not_found")
		case errors.Is(err, listdom.ErrNotFound),
			errors.Is(err, listdom.ErrInvalidID),
			errors.Is(err, inventorydom.ErrNotFound),
			errors.Is(err, inventorydom.ErrInvalidMintID),
			errors.Is(err, productblueprintdom.ErrNotFound),
			errors.Is(err, tokenblueprintdom.ErrNotFound):
			writeJSONError(w, http.StatusNotFound, "list_not_found")
		case errors.Is(err, adminquery.ErrContractListQueryNotConfigured):
			writeJSONError(w, http.StatusServiceUnavailable, "contract_list_query_not_initialized")
		default:
			writeJSONError(w, http.StatusInternalServerError, "contract_list_get_failed")
		}
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *CompanyHandler) handleContractTokenBlueprintDetail(
	w http.ResponseWriter,
	r *http.Request,
	companyID string,
	tokenBlueprintID string,
) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	if h.contractTokenBlueprintQuery == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "contract_token_blueprint_query_not_initialized")
		return
	}

	result, err := h.contractTokenBlueprintQuery.Get(r.Context(), companyID, tokenBlueprintID)
	if err != nil {
		switch {
		case errors.Is(err, companydom.ErrNotFound), errors.Is(err, companydom.ErrInvalidID):
			writeJSONError(w, http.StatusNotFound, "company_not_found")
		case errors.Is(err, tokenblueprintdom.ErrNotFound), errors.Is(err, tokenblueprintdom.ErrInvalidID):
			writeJSONError(w, http.StatusNotFound, "token_blueprint_not_found")
		case errors.Is(err, adminquery.ErrContractTokenBlueprintQueryNotConfigured):
			writeJSONError(w, http.StatusServiceUnavailable, "contract_token_blueprint_query_not_initialized")
		default:
			writeJSONError(w, http.StatusInternalServerError, "contract_token_blueprint_get_failed")
		}
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *CompanyHandler) handleContractProductBlueprintDetail(
	w http.ResponseWriter,
	r *http.Request,
	companyID string,
	productBlueprintID string,
) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	if h.contractProductBlueprintQuery == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "contract_product_blueprint_query_not_initialized")
		return
	}

	result, err := h.contractProductBlueprintQuery.Get(r.Context(), companyID, productBlueprintID)
	if err != nil {
		switch {
		case errors.Is(err, companydom.ErrNotFound), errors.Is(err, companydom.ErrInvalidID):
			writeJSONError(w, http.StatusNotFound, "company_not_found")
		case errors.Is(err, productblueprintdom.ErrNotFound), errors.Is(err, productblueprintdom.ErrInvalidID):
			writeJSONError(w, http.StatusNotFound, "product_blueprint_not_found")
		case errors.Is(err, adminquery.ErrContractProductBlueprintQueryNotConfigured):
			writeJSONError(w, http.StatusServiceUnavailable, "contract_product_blueprint_query_not_initialized")
		default:
			writeJSONError(w, http.StatusInternalServerError, "contract_product_blueprint_get_failed")
		}
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func parseContractDetailCompanyID(path string) (string, bool) {
	prefix := adminCompaniesPath + "/"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, adminContractDetailPathSuffix) {
		return "", false
	}

	companyID := strings.TrimSuffix(strings.TrimPrefix(path, prefix), adminContractDetailPathSuffix)
	companyID = strings.TrimSuffix(companyID, "/")
	companyID = strings.TrimSpace(companyID)
	if companyID == "" || strings.Contains(companyID, "/") {
		return "", false
	}
	return companyID, true
}

func parseContractResourcePath(
	path string,
	resource string,
) (companyID string, resourceID string, ok bool) {
	prefix := adminCompaniesPath + "/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}

	relativePath := strings.TrimPrefix(path, prefix)
	parts := strings.Split(relativePath, "/")
	if len(parts) != 3 || parts[1] != resource {
		return "", "", false
	}

	companyID = strings.TrimSpace(parts[0])
	resourceID = strings.TrimSpace(parts[2])
	if companyID == "" || resourceID == "" {
		return "", "", false
	}
	return companyID, resourceID, true
}

func (h *CompanyHandler) resolveRepresentativeName(
	ctx context.Context,
	memberID string,
) (string, error) {
	if memberID == "" {
		return "-", nil
	}

	record, err := h.memberRepo.GetByID(ctx, memberID)
	if err != nil {
		if errors.Is(err, memberdom.ErrNotFound) {
			return "-", nil
		}
		return "", err
	}

	name := memberdom.FormatLastFirst(record.Member.LastName, record.Member.FirstName)
	if name == "" {
		return "-", nil
	}
	return name, nil
}

func toAdminCompanyResponse(
	company companydom.Company,
	representativeName string,
) companyResponse {
	createdAt := ""
	if !company.CreatedAt.IsZero() {
		createdAt = company.CreatedAt.UTC().Format(time.RFC3339Nano)
	}

	updatedAt := ""
	if !company.UpdatedAt.IsZero() {
		updatedAt = company.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}

	return companyResponse{
		ID:                 company.ID,
		Name:               company.Name,
		RepresentativeName: representativeName,
		IsActive:           company.IsActive,
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
	}
}
