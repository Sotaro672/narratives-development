// backend/internal/adapters/in/http/admin/handler/resale_handler.go
package handler

import (
	"context"
	"net/http"

	productblueprintdom "narratives/internal/domain/productBlueprint"
	resaledom "narratives/internal/domain/resale"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
)

type ResaleListReader interface {
	ListByAvatarID(ctx context.Context, avatarID string) ([]resaledom.Resale, error)
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
	resaleRepo           ResaleListReader
	reportRepo           ResaleReportCountReader
	productBlueprintRepo ResaleProductBlueprintReader
	tokenBlueprintRepo   ResaleTokenBlueprintReader
}

type resaleResponse struct {
	ID          string  `json:"id"`
	ProductName string  `json:"productName"`
	TokenName   string  `json:"tokenName"`
	Status      string  `json:"status"`
	Price       int     `json:"price"`
	ReportCount int     `json:"reportCount"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   *string `json:"updatedAt,omitempty"`
}

type resaleListResponse struct {
	Items []resaleResponse `json:"items"`
}

func NewResaleHandler(
	resaleRepo ResaleListReader,
	reportRepo ResaleReportCountReader,
	productBlueprintRepo ResaleProductBlueprintReader,
	tokenBlueprintRepo ResaleTokenBlueprintReader,
) http.Handler {
	h := &ResaleHandler{
		resaleRepo:           resaleRepo,
		reportRepo:           reportRepo,
		productBlueprintRepo: productBlueprintRepo,
		tokenBlueprintRepo:   tokenBlueprintRepo,
	}
	return http.HandlerFunc(h.handleListByAvatar)
}

func (h *ResaleHandler) handleListByAvatar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}

	if h.resaleRepo == nil || h.reportRepo == nil || h.productBlueprintRepo == nil || h.tokenBlueprintRepo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "resale_dependencies_not_initialized")
		return
	}

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
		productBlueprint, err := h.productBlueprintRepo.GetByID(
			r.Context(),
			resale.ProductBlueprintID,
		)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "resale_product_resolve_failed")
			return
		}

		tokenBlueprint, err := h.tokenBlueprintRepo.GetByID(
			r.Context(),
			resale.TokenBlueprintID,
		)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "resale_token_resolve_failed")
			return
		}

		reportCount, err := h.reportRepo.GetResaleReportCount(
			r.Context(),
			resale.ID,
		)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "resale_report_count_failed")
			return
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

		items = append(items, resaleResponse{
			ID:          resale.ID,
			ProductName: productBlueprint.ProductName,
			TokenName:   tokenName,
			Status:      string(resale.Status),
			Price:       resale.Price,
			ReportCount: reportCount,
			CreatedAt:   resale.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   updatedAt,
		})
	}

	writeJSON(w, http.StatusOK, resaleListResponse{Items: items})
}
