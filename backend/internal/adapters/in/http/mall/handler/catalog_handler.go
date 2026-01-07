// backend/internal/adapters/in/http/sns/handler/catalog_handler.go
package mallHandler

import (
	"errors"
	"net/http"
	"strings"

	snsquery "narratives/internal/application/query/mall"
	ldom "narratives/internal/domain/list"
)

// SNSCatalogHandler serves buyer-facing catalog endpoint.
//
// Routes:
// - GET /sns/catalog/{listId}
type SNSCatalogHandler struct {
	Q *snsquery.CatalogQuery
}

func NewSNSCatalogHandler(q *snsquery.CatalogQuery) http.Handler {
	return &SNSCatalogHandler{Q: q}
}

func (h *SNSCatalogHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Q == nil {
		internalError(w, "catalog handler is not ready")
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")

	// detail: /mall/catalog/{listId}
	if strings.HasPrefix(path, "/mall/catalog/") {
		listID := strings.TrimSpace(strings.TrimPrefix(path, "/mall/catalog/"))
		if listID == "" {
			notFound(w)
			return
		}

		dto, err := h.Q.GetByListID(r.Context(), listID)
		if err != nil {
			// buyer-facing: not found should be 404 (not 500)
			if errors.Is(err, ldom.ErrNotFound) {
				notFound(w)
				return
			}
			internalError(w, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, dto)
		return
	}

	// (future) /sns/catalog index not implemented
	notFound(w)
}
