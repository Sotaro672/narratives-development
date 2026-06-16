package mallHandler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
	sharedquery "narratives/internal/application/query/shared"
)

type PreviewMeHandler struct {
	q PreviewQuery

	// optional
	ownerQ *sharedquery.OwnerResolveQuery

	// tokenBlueprint patch (optional)
	tbRepo TokenBlueprintPatchReader

	// name resolver (optional)
	nameR PreviewNameResolver
}

func NewPreviewMeHandler(
	q PreviewQuery,
	ownerQ *sharedquery.OwnerResolveQuery,
	tbRepo TokenBlueprintPatchReader,
	nameR PreviewNameResolver,
) http.Handler {
	return &PreviewMeHandler{
		q:      q,
		ownerQ: ownerQ,
		tbRepo: tbRepo,
		nameR:  nameR,
	}
}

func (h *PreviewMeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"error": "method not allowed",
		})
		return
	}

	if h == nil || h.q == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "preview query not configured",
		})
		return
	}

	auth := r.Header.Get("Authorization")
	if auth == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": "authorization header is required",
		})
		return
	}

	avatarID, _ := middleware.CurrentAvatarID(r)

	productID := strings.TrimSpace(r.URL.Query().Get("productId"))
	if productID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "productId is required",
		})
		return
	}

	info, err := h.q.ResolveModelInfoByProductID(r.Context(), productID)
	if err != nil {
		if isNotFound(err) {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"error":     "not found",
				"productId": productID,
				"avatarId":  avatarID,
			})
			return
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			writeJSON(w, http.StatusRequestTimeout, map[string]any{
				"error":     "request canceled",
				"productId": productID,
				"avatarId":  avatarID,
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":     "resolve failed",
			"productId": productID,
			"avatarId":  avatarID,
		})
		return
	}

	if info == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":     "resolve failed (nil result)",
			"productId": productID,
			"avatarId":  avatarID,
		})
		return
	}

	if info.Owner == nil && h.ownerQ != nil && info.Token != nil {
		addr := strings.TrimSpace(info.Token.ToAddress)
		if addr != "" {
			res, rerr := h.ownerQ.Resolve(r.Context(), addr)
			if rerr == nil {
				info.Owner = res
			}
		}
	}

	tbPatch := resolveTokenBlueprintPatch(r.Context(), info, h.tbRepo)
	tbDTO := buildTokenBlueprintPatchDTO(r.Context(), tbPatch, h.nameR)

	data := map[string]any{
		"productId":   info.ProductID,
		"modelId":     info.ModelID,
		"modelKind":   info.ModelKind,
		"modelNumber": info.ModelNumber,
		"modelLabel":  info.ModelLabel,

		// apparel
		"size":         info.Size,
		"color":        info.Color,
		"rgb":          info.RGB,
		"measurements": info.Measurements,

		// alcohol
		"volumeValue": info.VolumeValue,
		"volumeUnit":  info.VolumeUnit,

		// category / productBlueprint
		"productBlueprintId":           info.ProductBlueprintID,
		"productBlueprintCategoryCode": info.ProductBlueprintCategoryCode,
		"productBlueprintCategoryKind": info.ProductBlueprintCategoryKind,
		"productBlueprintCategoryName": info.ProductBlueprintCategoryName,
		"productBlueprintCategory":     info.ProductBlueprintCategory,
		"productBlueprintPatch":        info.ProductBlueprintPatch,
		"categoryInputSchema":          info.CategoryInputSchema,

		// display
		"brandName":   info.BrandName,
		"companyName": info.CompanyName,

		// token / owner / transfer
		"token":               info.Token,
		"owner":               info.Owner,
		"transfers":           info.Transfers,
		"tokenBlueprintPatch": tbDTO,
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":     data,
		"avatarId": avatarID,
	})
}
