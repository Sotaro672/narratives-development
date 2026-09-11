// backend/internal/adapters/in/http/admin/handler/resale_handler.go
package handler

import (
	"context"
	"errors"
	"net/http"

	productblueprintdom "narratives/internal/domain/productBlueprint"
	reportdom "narratives/internal/domain/report"
	resaledom "narratives/internal/domain/resale"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
)

type ResaleReader interface {
	GetByID(ctx context.Context, id string) (resaledom.Resale, error)
	ListByAvatarID(ctx context.Context, avatarID string) ([]resaledom.Resale, error)
}

type ResaleImageReader interface {
	ListByResaleID(ctx context.Context, resaleID string) ([]resaledom.ResaleImage, error)
}

type ResaleReportCountReader interface {
	GetResaleReportCount(ctx context.Context, resaleID string) (int, error)
}

type ResaleProductBlueprintReader interface {
	GetByID(ctx context.Context, id string) (productblueprintdom.ProductBlueprint, error)
}

type ResaleTokenBlueprintReader interface {
	GetByID(ctx context.Context, id string) (*tokenblueprintdom.TokenBlueprint, error)
}

type ResaleHandler struct {
	resaleRepo           ResaleReader
	resaleImageRepo      ResaleImageReader
	reportRepo           ResaleReportCountReader
	productBlueprintRepo ResaleProductBlueprintReader
	tokenBlueprintRepo   ResaleTokenBlueprintReader
}

type resaleImageResponse struct {
	ID           string `json:"id"`
	URL          string `json:"url"`
	DisplayOrder int    `json:"displayOrder"`
}

type resaleResponse struct {
	ID                 string                `json:"id"`
	CompanyID          string                `json:"companyId"`
	ProductBlueprintID string                `json:"productBlueprintId"`
	ProductName        string                `json:"productName"`
	TokenBlueprintID   string                `json:"tokenBlueprintId"`
	TokenName          string                `json:"tokenName"`
	ReportCaseID       string                `json:"reportCaseId"`
	Status             string                `json:"status"`
	Price              int                   `json:"price"`
	Condition          string                `json:"condition"`
	Description        string                `json:"description"`
	Images             []resaleImageResponse `json:"images"`
	ReportCount        int                   `json:"reportCount"`
	CreatedAt          string                `json:"createdAt"`
	UpdatedAt          *string               `json:"updatedAt,omitempty"`
}

type resaleListResponse struct {
	Items []resaleResponse `json:"items"`
}

func NewResaleHandler(
	resaleRepo ResaleReader,
	resaleImageRepo ResaleImageReader,
	reportRepo ResaleReportCountReader,
	productBlueprintRepo ResaleProductBlueprintReader,
	tokenBlueprintRepo ResaleTokenBlueprintReader,
) http.Handler {
	h := &ResaleHandler{
		resaleRepo:           resaleRepo,
		resaleImageRepo:      resaleImageRepo,
		reportRepo:           reportRepo,
		productBlueprintRepo: productBlueprintRepo,
		tokenBlueprintRepo:   tokenBlueprintRepo,
	}
	return http.HandlerFunc(h.handle)
}

func (h *ResaleHandler) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}

	if h.resaleRepo == nil || h.resaleImageRepo == nil || h.reportRepo == nil || h.productBlueprintRepo == nil || h.tokenBlueprintRepo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "resale_dependencies_not_initialized")
		return
	}

	if r.PathValue("resaleID") != "" {
		h.handleDetailByAvatar(w, r)
		return
	}

	h.handleListByAvatar(w, r)
}

func (h *ResaleHandler) handleListByAvatar(w http.ResponseWriter, r *http.Request) {
	avatarID := r.PathValue("avatarID")
	if avatarID == "" {
		writeJSONError(w, http.StatusBadRequest, "avatar_id_required")
		return
	}

	resales, err := h.resaleRepo.ListByAvatarID(r.Context(), avatarID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "resale_list_failed")
		return
	}

	items := make([]resaleResponse, 0, len(resales))
	for _, resale := range resales {
		response, err := h.buildResaleResponse(r.Context(), resale)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "resale_response_build_failed")
			return
		}
		items = append(items, response)
	}

	writeJSON(w, http.StatusOK, resaleListResponse{Items: items})
}

func (h *ResaleHandler) handleDetailByAvatar(w http.ResponseWriter, r *http.Request) {
	avatarID := r.PathValue("avatarID")
	if avatarID == "" {
		writeJSONError(w, http.StatusBadRequest, "avatar_id_required")
		return
	}

	resaleID := r.PathValue("resaleID")
	if resaleID == "" {
		writeJSONError(w, http.StatusBadRequest, "resale_id_required")
		return
	}

	resale, err := h.resaleRepo.GetByID(r.Context(), resaleID)
	if err != nil {
		if errors.Is(err, resaledom.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "resale_not_found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "resale_get_failed")
		return
	}

	if resale.AvatarID != avatarID {
		writeJSONError(w, http.StatusNotFound, "resale_not_found")
		return
	}

	response, err := h.buildResaleResponse(r.Context(), resale)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "resale_response_build_failed")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *ResaleHandler) buildResaleResponse(
	ctx context.Context,
	resale resaledom.Resale,
) (resaleResponse, error) {
	productBlueprint, err := h.productBlueprintRepo.GetByID(ctx, resale.ProductBlueprintID)
	if err != nil {
		return resaleResponse{}, err
	}

	tokenBlueprint, err := h.tokenBlueprintRepo.GetByID(ctx, resale.TokenBlueprintID)
	if err != nil {
		return resaleResponse{}, err
	}

	resaleImages, err := h.resaleImageRepo.ListByResaleID(ctx, resale.ID)
	if err != nil {
		return resaleResponse{}, err
	}

	images := make([]resaleImageResponse, 0, len(resaleImages))
	for _, image := range resaleImages {
		images = append(images, resaleImageResponse{
			ID:           image.ID,
			URL:          image.URL,
			DisplayOrder: image.DisplayOrder,
		})
	}

	reportCount, err := h.reportRepo.GetResaleReportCount(ctx, resale.ID)
	if err != nil {
		return resaleResponse{}, err
	}

	reportCaseID := ""
	if reportCount > 0 {
		caseID, err := reportdom.BuildCaseID(reportdom.TargetTypeResale, resale.ID)
		if err != nil {
			return resaleResponse{}, err
		}
		reportCaseID = string(caseID)
	}

	tokenName := ""
	if tokenBlueprint != nil {
		tokenName = tokenBlueprint.Name
	}

	var updatedAt *string
	if resale.UpdatedAt != nil {
		value := resale.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
		updatedAt = &value
	}

	return resaleResponse{
		ID:                 resale.ID,
		CompanyID:          productBlueprint.CompanyID,
		ProductBlueprintID: resale.ProductBlueprintID,
		ProductName:        productBlueprint.ProductName,
		TokenBlueprintID:   resale.TokenBlueprintID,
		TokenName:          tokenName,
		ReportCaseID:       reportCaseID,
		Status:             string(resale.Status),
		Price:              resale.Price,
		Condition:          string(resale.Condition),
		Description:        resale.Description,
		Images:             images,
		ReportCount:        reportCount,
		CreatedAt:          resale.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:          updatedAt,
	}, nil
}
