// backend/internal/adapters/in/http/admin/handler/company_handler.go
package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	adminquery "narratives/internal/application/query/admin"
	common "narratives/internal/domain/common"
	companydom "narratives/internal/domain/company"
	memberdom "narratives/internal/domain/member"
	productblueprintreviewdom "narratives/internal/domain/productBlueprintReview"
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
	Get(
		ctx context.Context,
		companyID string,
	) (adminquery.ContractDetailResult, error)
}

type ContractAnnouncementDetailReader interface {
	Get(
		ctx context.Context,
		companyID string,
		announcementID string,
	) (adminquery.ContractAnnouncementDetailResult, error)
}

type ContractListReader interface {
	Get(
		ctx context.Context,
		companyID string,
		listID string,
	) (adminquery.ContractListDetailResult, error)
}

type ContractTokenBlueprintReader interface {
	Get(
		ctx context.Context,
		companyID string,
		tokenBlueprintID string,
	) (adminquery.ContractTokenBlueprintDetailResult, error)
}

type ContractProductBlueprintReader interface {
	Get(
		ctx context.Context,
		companyID string,
		productBlueprintID string,
	) (adminquery.ContractProductBlueprintDetailResult, error)
}

type ContractTokenBlueprintReviewReader interface {
	List(
		ctx context.Context,
		companyID string,
		tokenBlueprintID string,
		page common.Page,
	) (adminquery.ContractTokenBlueprintReviewResult, error)
}

type ContractProductBlueprintReviewReader interface {
	List(
		ctx context.Context,
		companyID string,
		productBlueprintID string,
		status productblueprintreviewdom.ReviewStatus,
		page common.Page,
	) (adminquery.ContractProductBlueprintReviewResult, error)
}

type CompanyHandler struct {
	companyRepo                         CompanyListReader
	memberRepo                          MemberReader
	contractDetailQuery                 ContractDetailReader
	contractAnnouncementDetailQuery     ContractAnnouncementDetailReader
	contractListQuery                   ContractListReader
	contractTokenBlueprintQuery         ContractTokenBlueprintReader
	contractProductBlueprintQuery       ContractProductBlueprintReader
	contractTokenBlueprintReviewQuery   ContractTokenBlueprintReviewReader
	contractProductBlueprintReviewQuery ContractProductBlueprintReviewReader
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
	contractAnnouncementDetailQuery ContractAnnouncementDetailReader,
	contractListQuery ContractListReader,
	contractTokenBlueprintQuery ContractTokenBlueprintReader,
	contractProductBlueprintQuery ContractProductBlueprintReader,
	contractTokenBlueprintReviewQuery ContractTokenBlueprintReviewReader,
	contractProductBlueprintReviewQuery ContractProductBlueprintReviewReader,
) http.Handler {
	return http.HandlerFunc((&CompanyHandler{
		companyRepo:                         companyRepo,
		memberRepo:                          memberRepo,
		contractDetailQuery:                 contractDetailQuery,
		contractAnnouncementDetailQuery:     contractAnnouncementDetailQuery,
		contractListQuery:                   contractListQuery,
		contractTokenBlueprintQuery:         contractTokenBlueprintQuery,
		contractProductBlueprintQuery:       contractProductBlueprintQuery,
		contractTokenBlueprintReviewQuery:   contractTokenBlueprintReviewQuery,
		contractProductBlueprintReviewQuery: contractProductBlueprintReviewQuery,
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

	if companyID, tokenBlueprintID, ok := parseContractReviewResourcePath(path, "token-blueprints"); ok {
		h.handleContractTokenBlueprintReviews(w, r, companyID, tokenBlueprintID)
		return
	}

	if companyID, productBlueprintID, ok := parseContractReviewResourcePath(path, "product-blueprints"); ok {
		h.handleContractProductBlueprintReviews(w, r, companyID, productBlueprintID)
		return
	}

	if companyID, announcementID, ok := parseContractResourcePath(path, "announcements"); ok {
		h.handleContractAnnouncementDetail(w, r, companyID, announcementID)
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
		representativeName, err := h.resolveRepresentativeName(
			r.Context(),
			company.Admin,
		)
		if err != nil {
			writeJSONError(
				w,
				http.StatusInternalServerError,
				"company_representative_resolve_failed",
			)
			return
		}

		items = append(
			items,
			toAdminCompanyResponse(company, representativeName),
		)
	}

	writeJSON(
		w,
		http.StatusOK,
		companyListResponse{Items: items},
	)
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
