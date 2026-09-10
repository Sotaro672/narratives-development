// backend/internal/adapters/in/http/admin/handler/mint_handler.go
package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	adminquery "narratives/internal/application/query/admin"
	mintdom "narratives/internal/domain/mint"
)

const adminMintsPath = "/admin/mints"

type MintListQueryService interface {
	ListMints(ctx context.Context) (*adminquery.MintListResult, error)
}

type MintDetailQueryService interface {
	Get(ctx context.Context, mintID string) (adminquery.MintDetailResult, error)
}

type MintHandler struct {
	listQuery   MintListQueryService
	detailQuery MintDetailQueryService
}

func NewMintHandler(
	listQuery MintListQueryService,
	detailQuery MintDetailQueryService,
) http.Handler {
	return http.HandlerFunc((&MintHandler{
		listQuery:   listQuery,
		detailQuery: detailQuery,
	}).handle)
}

func (h *MintHandler) handle(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")

	if path == adminMintsPath {
		h.handleList(w, r)
		return
	}

	if mintID, ok := parseMintDetailID(path); ok {
		h.handleDetail(w, r, mintID)
		return
	}

	writeJSONError(w, http.StatusNotFound, "mint_resource_not_found")
}

func (h *MintHandler) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}

	if h.listQuery == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "mint_list_query_not_initialized")
		return
	}

	result, err := h.listQuery.ListMints(r.Context())
	if err != nil {
		if errors.Is(err, adminquery.ErrMintListQueryNotConfigured) {
			writeJSONError(w, http.StatusServiceUnavailable, "mint_list_query_not_initialized")
			return
		}

		writeJSONError(w, http.StatusInternalServerError, "mint_list_unavailable")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *MintHandler) handleDetail(
	w http.ResponseWriter,
	r *http.Request,
	mintID string,
) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}

	if h.detailQuery == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "mint_detail_query_not_initialized")
		return
	}

	result, err := h.detailQuery.Get(r.Context(), mintID)
	if err != nil {
		switch {
		case errors.Is(err, mintdom.ErrNotFound),
			errors.Is(err, mintdom.ErrInvalidMintID):
			writeJSONError(w, http.StatusNotFound, "mint_not_found")
		case errors.Is(err, adminquery.ErrMintDetailIDRequired):
			writeJSONError(w, http.StatusBadRequest, "mint_id_required")
		case errors.Is(err, adminquery.ErrMintDetailQueryNotConfigured):
			writeJSONError(w, http.StatusServiceUnavailable, "mint_detail_query_not_initialized")
		default:
			writeJSONError(w, http.StatusInternalServerError, "mint_detail_unavailable")
		}
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func parseMintDetailID(path string) (string, bool) {
	prefix := adminMintsPath + "/"
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}

	mintID := strings.TrimSpace(strings.TrimPrefix(path, prefix))
	if mintID == "" || strings.Contains(mintID, "/") {
		return "", false
	}

	return mintID, true
}
