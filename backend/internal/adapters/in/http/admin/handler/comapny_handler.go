// backend/internal/adapters/in/http/admin/handler/company_handler.go
package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	companydom "narratives/internal/domain/company"
	memberdom "narratives/internal/domain/member"
)

const adminCompaniesPath = "/admin/companies"

type CompanyListReader interface {
	ListAll(ctx context.Context) ([]companydom.Company, error)
}

type MemberReader interface {
	GetByID(ctx context.Context, id string) (memberdom.Record, error)
}

type CompanyHandler struct {
	companyRepo CompanyListReader
	memberRepo  MemberReader
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
) http.Handler {
	return http.HandlerFunc((&CompanyHandler{
		companyRepo: companyRepo,
		memberRepo:  memberRepo,
	}).handle)
}

func (h *CompanyHandler) handle(w http.ResponseWriter, r *http.Request) {
	if h.companyRepo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "company_repository_not_initialized")
		return
	}
	if h.memberRepo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "member_repository_not_initialized")
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")
	if path != adminCompaniesPath {
		writeJSONError(w, http.StatusNotFound, "company_not_found")
		return
	}

	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}

	companies, err := h.companyRepo.ListAll(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "company_list_failed")
		return
	}

	items := make([]companyResponse, 0, len(companies))

	for _, company := range companies {
		representativeName, err := h.resolveRepresentativeName(
			r.Context(),
			company.Admin,
		)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "company_representative_resolve_failed")
			return
		}

		items = append(items, toAdminCompanyResponse(
			company,
			representativeName,
		))
	}

	writeJSON(w, http.StatusOK, companyListResponse{
		Items: items,
	})
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

	name := memberdom.FormatLastFirst(
		record.Member.LastName,
		record.Member.FirstName,
	)
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
