// backend/internal/adapters/in/http/admin/handler/avatar_handler.go
package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	avatardom "narratives/internal/domain/avatar"
	productblueprintdom "narratives/internal/domain/productBlueprint"
	resaledom "narratives/internal/domain/resale"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
	userdom "narratives/internal/domain/user"
)

const adminAvatarsPath = "/admin/avatars"

type AvatarListReader interface {
	ListAll(ctx context.Context) ([]avatardom.Avatar, error)
}

type AvatarUserReader interface {
	GetByID(ctx context.Context, id string) (*userdom.User, error)
}

type AvatarReportCountReader interface {
	GetAvatarReportCount(ctx context.Context, avatarID string) (int, error)
}

type AvatarResaleReader interface {
	ListByAvatarID(ctx context.Context, avatarID string) ([]resaledom.Resale, error)
}

type AvatarProductBlueprintReader interface {
	GetByID(ctx context.Context, id string) (productblueprintdom.ProductBlueprint, error)
}

type AvatarTokenBlueprintReader interface {
	GetByID(ctx context.Context, id string) (*tokenblueprintdom.TokenBlueprint, error)
}

type AvatarHandler struct {
	avatarRepo           AvatarListReader
	userRepo             AvatarUserReader
	reportRepo           AvatarReportCountReader
	resaleRepo           AvatarResaleReader
	productBlueprintRepo AvatarProductBlueprintReader
	tokenBlueprintRepo   AvatarTokenBlueprintReader
}

type avatarResponse struct {
	ID           string  `json:"id"`
	AvatarName   string  `json:"avatarName"`
	AvatarIcon   *string `json:"avatarIcon,omitempty"`
	Profile      *string `json:"profile,omitempty"`
	ExternalLink *string `json:"externalLink,omitempty"`
	UserName     string  `json:"userName"`
	ResaleCount  int     `json:"resaleCount"`
	ReportCount  int     `json:"reportCount"`
	CreatedAt    string  `json:"createdAt"`
	UpdatedAt    string  `json:"updatedAt"`
}

type avatarListResponse struct {
	Items []avatarResponse `json:"items"`
}

type avatarResaleResponse struct {
	ID          string  `json:"id"`
	ProductName string  `json:"productName"`
	TokenName   string  `json:"tokenName"`
	Status      string  `json:"status"`
	Price       int     `json:"price"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   *string `json:"updatedAt,omitempty"`
}

type avatarResaleListResponse struct {
	Items []avatarResaleResponse `json:"items"`
}

func NewAvatarHandler(
	avatarRepo AvatarListReader,
	userRepo AvatarUserReader,
	reportRepo AvatarReportCountReader,
	resaleRepo AvatarResaleReader,
	productBlueprintRepo AvatarProductBlueprintReader,
	tokenBlueprintRepo AvatarTokenBlueprintReader,
) http.Handler {
	return http.HandlerFunc((&AvatarHandler{
		avatarRepo:           avatarRepo,
		userRepo:             userRepo,
		reportRepo:           reportRepo,
		resaleRepo:           resaleRepo,
		productBlueprintRepo: productBlueprintRepo,
		tokenBlueprintRepo:   tokenBlueprintRepo,
	}).handle)
}

func (h *AvatarHandler) handle(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")

	if path == adminAvatarsPath {
		h.handleList(w, r)
		return
	}

	avatarID, ok := parseAvatarResalesPath(path)
	if ok {
		h.handleResales(w, r, avatarID)
		return
	}

	writeJSONError(w, http.StatusNotFound, "avatar_resource_not_found")
}

func (h *AvatarHandler) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}

	if h.avatarRepo == nil || h.userRepo == nil || h.reportRepo == nil || h.resaleRepo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "avatar_dependencies_not_initialized")
		return
	}

	avatars, err := h.avatarRepo.ListAll(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "avatar_list_failed")
		return
	}

	items := make([]avatarResponse, 0, len(avatars))
	for _, avatar := range avatars {
		userName, err := h.resolveUserName(r.Context(), avatar.UserID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "avatar_user_resolve_failed")
			return
		}

		resales, err := h.resaleRepo.ListByAvatarID(r.Context(), avatar.ID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "avatar_resale_count_failed")
			return
		}

		reportCount, err := h.reportRepo.GetAvatarReportCount(r.Context(), avatar.ID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "avatar_report_count_failed")
			return
		}

		items = append(items, avatarResponse{
			ID:           avatar.ID,
			AvatarName:   avatar.AvatarName,
			AvatarIcon:   avatar.AvatarIcon,
			Profile:      avatar.Profile,
			ExternalLink: avatar.ExternalLink,
			UserName:     userName,
			ResaleCount:  len(resales),
			ReportCount:  reportCount,
			CreatedAt:    avatar.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:    avatar.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	writeJSON(w, http.StatusOK, avatarListResponse{Items: items})
}

func (h *AvatarHandler) handleResales(
	w http.ResponseWriter,
	r *http.Request,
	avatarID string,
) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}

	if h.resaleRepo == nil || h.productBlueprintRepo == nil || h.tokenBlueprintRepo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "avatar_resale_dependencies_not_initialized")
		return
	}

	resales, err := h.resaleRepo.ListByAvatarID(r.Context(), avatarID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "avatar_resale_list_failed")
		return
	}

	items := make([]avatarResaleResponse, 0, len(resales))
	for _, resale := range resales {
		productBlueprint, err := h.productBlueprintRepo.GetByID(
			r.Context(),
			resale.ProductBlueprintID,
		)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "avatar_resale_product_resolve_failed")
			return
		}

		tokenBlueprint, err := h.tokenBlueprintRepo.GetByID(
			r.Context(),
			resale.TokenBlueprintID,
		)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "avatar_resale_token_resolve_failed")
			return
		}

		var updatedAt *string
		if resale.UpdatedAt != nil {
			value := resale.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
			updatedAt = &value
		}

		tokenName := ""
		if tokenBlueprint != nil {
			tokenName = tokenBlueprint.Name
		}

		items = append(items, avatarResaleResponse{
			ID:          resale.ID,
			ProductName: productBlueprint.ProductName,
			TokenName:   tokenName,
			Status:      string(resale.Status),
			Price:       resale.Price,
			CreatedAt:   resale.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   updatedAt,
		})
	}

	writeJSON(w, http.StatusOK, avatarResaleListResponse{Items: items})
}

func parseAvatarResalesPath(path string) (string, bool) {
	prefix := adminAvatarsPath + "/"
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}

	parts := strings.Split(strings.TrimPrefix(path, prefix), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "resales" {
		return "", false
	}

	return parts[0], true
}

func (h *AvatarHandler) resolveUserName(
	ctx context.Context,
	userID string,
) (string, error) {
	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, userdom.ErrNotFound) {
			return "", nil
		}
		return "", err
	}

	return userdom.FormatName(user), nil
}
