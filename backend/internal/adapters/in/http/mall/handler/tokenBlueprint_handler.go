// backend/internal/adapters/in/http/mall/handler/tokenBlueprint_handler.go
package mallHandler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	tbdom "narratives/internal/domain/tokenBlueprint"
)

type tokenBlueprintPatchGetter interface {
	GetPatchByID(ctx context.Context, id string) (tbdom.Patch, error)
}

type tokenBlueprintNameResolver interface {
	ResolveBrandName(ctx context.Context, brandID string) string
	ResolveBrandCompanyID(ctx context.Context, brandID string) string
}

type MallTokenBlueprintHandler struct {
	Repo tokenBlueprintPatchGetter

	NameResolver tokenBlueprintNameResolver
}

func NewMallTokenBlueprintHandler(repo tokenBlueprintPatchGetter) http.Handler {
	return &MallTokenBlueprintHandler{
		Repo: repo,
	}
}

func NewMallTokenBlueprintHandlerWithNameResolver(
	repo tokenBlueprintPatchGetter,
	nameResolver tokenBlueprintNameResolver,
) http.Handler {
	return &MallTokenBlueprintHandler{
		Repo:         repo,
		NameResolver: nameResolver,
	}
}

func (h *MallTokenBlueprintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil {
		internalError(w, "mall tokenBlueprint handler is nil")
		return
	}

	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	id, ok := parseTokenBlueprintPatchPath(r.URL.Path)
	if !ok {
		notFound(w)
		return
	}

	patch, err := h.getPatchByID(r.Context(), id)
	if err != nil {
		msg := err.Error()
		if msg == "" {
			msg = "failed to get tokenBlueprint patch"
		}

		if isNotFoundError(err) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": msg})
			return
		}

		internalError(w, msg)
		return
	}

	writeJSON(w, http.StatusOK, patch)
}

func (h *MallTokenBlueprintHandler) getPatchByID(ctx context.Context, id string) (tbdom.Patch, error) {
	if id == "" {
		return tbdom.Patch{}, errors.New("tokenBlueprint id is empty")
	}
	if h.Repo == nil {
		return tbdom.Patch{}, errors.New("tokenBlueprint repo is nil")
	}

	patch, err := h.Repo.GetPatchByID(ctx, id)
	if err != nil {
		return tbdom.Patch{}, err
	}

	if patch.ID == "" {
		patch.ID = id
	}

	if h.NameResolver != nil {
		if patch.CompanyID == "" && patch.BrandID != "" {
			if companyID := h.NameResolver.ResolveBrandCompanyID(ctx, patch.BrandID); companyID != "" {
				patch.CompanyID = companyID
			}
		}

		if patch.BrandName == "" && patch.BrandID != "" {
			if brandName := h.NameResolver.ResolveBrandName(ctx, patch.BrandID); brandName != "" {
				patch.BrandName = brandName
			}
		}
	}

	return patch, nil
}

func parseTokenBlueprintPatchPath(path string) (id string, ok bool) {
	p := strings.TrimSuffix(path, "/")
	const prefix = "/mall/token-blueprints/"
	const suffix = "/patch"
	if !strings.HasPrefix(p, prefix) || !strings.HasSuffix(p, suffix) {
		return "", false
	}

	inner := strings.TrimSuffix(strings.TrimPrefix(p, prefix), suffix)
	inner = strings.Trim(inner, "/")
	if inner == "" || strings.Contains(inner, "/") {
		return "", false
	}

	return inner, true
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	if msg == "" {
		return false
	}

	if strings.Contains(msg, "not found") || strings.Contains(msg, "errnotfound") {
		return true
	}
	if strings.Contains(msg, "statuscode=404") || strings.Contains(msg, "code=404") {
		return true
	}

	return false
}
