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
	ResolveModelInfoByProductID(
		ctx context.Context,
		productID string,
	) (*dto.PreviewModelInfo, error)
}

type TokenBlueprintPatchReader interface {
	GetPatchByID(
		ctx context.Context,
		id string,
	) (tokenbpdom.Patch, error)
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
	ownerQ *sharedquery.OwnerResolveQuery

	tbRepo TokenBlueprintPatchReader
	nameR  PreviewNameResolver
}

type PreviewHandlerOption func(*PreviewHandler)

func WithOwnerResolveQuery(
	ownerQ *sharedquery.OwnerResolveQuery,
) PreviewHandlerOption {
	return func(h *PreviewHandler) {
		h.ownerQ = ownerQ
	}
}

func WithTokenBlueprintPatchRepo(
	tbRepo TokenBlueprintPatchReader,
) PreviewHandlerOption {
	return func(h *PreviewHandler) {
		h.tbRepo = tbRepo
	}
}

func WithNameResolver(
	nameR PreviewNameResolver,
) PreviewHandlerOption {
	return func(h *PreviewHandler) {
		h.nameR = nameR
	}
}

// 唯一の出入り口
func NewPreviewHandler(
	q PreviewQuery,
	opts ...PreviewHandlerOption,
) http.Handler {
	h := &PreviewHandler{
		q: q,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(h)
		}
	}

	return h
}

// ------------------------------------------------------------
// ServeHTTP
// ------------------------------------------------------------

func (h *PreviewHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
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

	productID := strings.TrimSpace(
		r.URL.Query().Get("productId"),
	)
	if productID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "productId is required",
		})
		return
	}

	info, err := h.q.ResolveModelInfoByProductID(
		r.Context(),
		productID,
	)
	if err != nil {
		switch {
		case isNotFound(err):
			writeJSON(w, http.StatusNotFound, map[string]any{
				"error":     "not found",
				"productId": productID,
			})
			return

		case errors.Is(err, context.Canceled),
			errors.Is(err, context.DeadlineExceeded):
			writeJSON(w, http.StatusRequestTimeout, map[string]any{
				"error":     "request canceled",
				"productId": productID,
			})
			return

		default:
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"error":     "resolve failed",
				"productId": productID,
			})
			return
		}
	}

	if info == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":     "resolve failed (nil result)",
			"productId": productID,
		})
		return
	}

	data := buildPreviewData(
		r.Context(),
		info,
		h.ownerQ,
		h.tbRepo,
		h.nameR,
	)

	writeJSON(w, http.StatusOK, map[string]any{
		"data": data,
	})
}
