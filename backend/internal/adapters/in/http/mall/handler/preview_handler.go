// backend/internal/adapters/in/http/mall/handler/preview_handler.go
package mallHandler

import (
	"context"
	"net/http"

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
	if !validatePreviewGETRequest(w, r) {
		return
	}

	var (
		q      PreviewQuery
		ownerQ *sharedquery.OwnerResolveQuery
		tbRepo TokenBlueprintPatchReader
		nameR  PreviewNameResolver
	)

	if h != nil {
		q = h.q
		ownerQ = h.ownerQ
		tbRepo = h.tbRepo
		nameR = h.nameR
	}

	info, ok := resolvePreviewModelInfoFromRequest(
		w,
		r,
		q,
		nil,
	)
	if !ok {
		return
	}

	data := buildPreviewData(
		r.Context(),
		info,
		ownerQ,
		tbRepo,
		nameR,
	)

	writeJSON(w, http.StatusOK, map[string]any{
		"data": data,
	})
}
