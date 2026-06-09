// backend/internal/adapters/in/http/mall/handler/preview_handler.go
package mallHandler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	dto "narratives/internal/application/query/mall/dto"
	sharedquery "narratives/internal/application/query/shared"

	tokenbpdom "narratives/internal/domain/tokenBlueprint"
)

// ------------------------------------------------------------
// Interfaces (ports)
// ------------------------------------------------------------

type PreviewQuery interface {
	ResolveModelInfoByProductID(ctx context.Context, productID string) (*dto.PreviewModelInfo, error)
}

type TokenBlueprintPatchReader interface {
	GetPatchByID(ctx context.Context, id string) (tokenbpdom.Patch, error)
}

type PreviewNameResolver interface {
	ResolveBrandName(ctx context.Context, brandID string) string
	ResolveCompanyName(ctx context.Context, companyID string) string
	ResolveBrandCompanyID(ctx context.Context, brandID string) string
}

// ------------------------------------------------------------
// Handler + Options
// ------------------------------------------------------------

type PreviewHandler struct {
	q      PreviewQuery
	ownerQ *sharedquery.OwnerResolveQuery // optional

	tbRepo TokenBlueprintPatchReader // optional
	nameR  PreviewNameResolver       // optional
}

type PreviewHandlerOption func(*PreviewHandler)

func WithOwnerResolveQuery(ownerQ *sharedquery.OwnerResolveQuery) PreviewHandlerOption {
	return func(h *PreviewHandler) { h.ownerQ = ownerQ }
}

func WithTokenBlueprintPatchRepo(tbRepo TokenBlueprintPatchReader) PreviewHandlerOption {
	return func(h *PreviewHandler) { h.tbRepo = tbRepo }
}

func WithNameResolver(nameR PreviewNameResolver) PreviewHandlerOption {
	return func(h *PreviewHandler) { h.nameR = nameR }
}

// 唯一の出入り口
func NewPreviewHandler(q PreviewQuery, opts ...PreviewHandlerOption) http.Handler {
	h := &PreviewHandler{
		q:      q,
		ownerQ: nil,
		tbRepo: nil,
		nameR:  nil,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(h)
		}
	}

	return h
}

// ------------------------------------------------------------
// DTO
// ------------------------------------------------------------

type tokenBlueprintPatchDTO struct {
	ID          string `json:"id"`
	TokenName   string `json:"tokenName"`
	Symbol      string `json:"symbol"`
	BrandName   string `json:"brandName,omitempty"`
	CompanyName string `json:"companyName,omitempty"`
	Description string `json:"description,omitempty"`
	TokenIcon   string `json:"tokenIcon,omitempty"`
}

// ------------------------------------------------------------
// ServeHTTP
// ------------------------------------------------------------

func (h *PreviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	if h == nil || h.q == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "preview query not configured"})
		return
	}

	productID := strings.TrimSpace(r.URL.Query().Get("productId"))
	if productID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "productId is required"})
		return
	}

	info, err := h.q.ResolveModelInfoByProductID(r.Context(), productID)
	if err != nil {
		if isNotFound(err) {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found", "productId": productID})
			return
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			writeJSON(w, http.StatusRequestTimeout, map[string]any{"error": "request canceled", "productId": productID})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "resolve failed", "productId": productID})
		return
	}

	if info == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "resolve failed (nil result)", "productId": productID})
		return
	}

	// owner resolve (best-effort)
	if info.Owner == nil && h.ownerQ != nil && info.Token != nil {
		addr := strings.TrimSpace(info.Token.ToAddress)
		if addr != "" {
			res, rerr := h.ownerQ.Resolve(r.Context(), addr)
			if rerr == nil {
				info.Owner = res
			}
		}
	}

	// tokenBlueprint patch (best-effort)
	var tbDTO *tokenBlueprintPatchDTO
	if h.tbRepo != nil && info.Token != nil {
		tbID := strings.TrimSpace(info.Token.TokenBlueprintID)
		if tbID != "" {
			p, perr := h.tbRepo.GetPatchByID(r.Context(), tbID)
			if perr == nil {
				brandName := ""
				companyName := ""

				if h.nameR != nil {
					brandID := strings.TrimSpace(p.BrandID)
					companyID := strings.TrimSpace(p.CompanyID)

					if brandID != "" {
						brandName = h.nameR.ResolveBrandName(r.Context(), brandID)
					}

					if companyID != "" {
						companyName = h.nameR.ResolveCompanyName(r.Context(), companyID)
					}

					if companyName == "" && brandID != "" {
						brandCompanyID := h.nameR.ResolveBrandCompanyID(r.Context(), brandID)
						if brandCompanyID != "" {
							companyName = h.nameR.ResolveCompanyName(r.Context(), brandCompanyID)
						}
					}
				}

				// Firebase Storage 移行後:
				// - Patch.IconURL には Firebase Storage の downloadURL が入る
				// - GCS objectPath から URL を解決しない
				// - gcs.NewTokenIconURLResolver / TokenIconObjectPath は使わない
				resolvedIconURL := strings.TrimSpace(p.IconURL)

				tbDTO = &tokenBlueprintPatchDTO{
					ID:          p.ID,
					TokenName:   p.TokenName,
					Symbol:      p.Symbol,
					BrandName:   brandName,
					CompanyName: companyName,
					Description: p.Description,
					TokenIcon:   resolvedIconURL,
				}
			}
		}
	}

	// 期待値：productBlueprint は返さず patch のみ
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

	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}
