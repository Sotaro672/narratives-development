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
	memberdom "narratives/internal/domain/member"
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

type CompanyHandler struct {
	companyRepo         CompanyListReader
	memberRepo          MemberReader
	contractDetailQuery ContractDetailReader
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
) http.Handler {
	return http.HandlerFunc((&CompanyHandler{
		companyRepo:         companyRepo,
		memberRepo:          memberRepo,
		contractDetailQuery: contractDetailQuery,
	}).handle)
}

func (h *CompanyHandler) handle(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")

	if path == adminCompaniesPath {
		h.handleList(w, r)
		return
	}

	companyID, ok := parseContractDetailCompanyID(path)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "company_not_found")
		return
	}

	h.handleContractDetail(w, r, companyID)
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

	writeJSON(w, http.StatusOK, companyListResponse{
		Items: items,
	})
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
		case errors.Is(err, companydom.ErrNotFound),
			errors.Is(err, companydom.ErrInvalidID):
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
