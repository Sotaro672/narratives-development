// backend/internal/adapters/in/http/mall/handler/brand_handler.go
package mallHandler

import (
	"net/http"
	"strings"

	mallquery "narratives/internal/application/query/mall"
	branddom "narratives/internal/domain/brand"
)

// MallBrandHandler serves buyer-facing brand endpoint.
//
// Route:
//   - GET /mall/brands/{id}
type MallBrandHandler struct {
	q *mallquery.BrandQuery
}

func NewMallBrandHandler(
	q *mallquery.BrandQuery,
) http.Handler {
	return &MallBrandHandler{
		q: q,
	}
}

func (h *MallBrandHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil || h.q == nil {
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "brand handler is not ready",
			},
		)
		return
	}

	if r.Method != http.MethodGet {
		writeJSON(
			w,
			http.StatusMethodNotAllowed,
			map[string]string{
				"error": "method_not_allowed",
			},
		)
		return
	}

	path := strings.TrimSuffix(
		r.URL.Path,
		"/",
	)

	// GET /mall/brands/{id}
	if strings.HasPrefix(
		path,
		"/mall/brands/",
	) {
		id := strings.TrimPrefix(
			path,
			"/mall/brands/",
		)

		h.get(
			w,
			r,
			id,
		)
		return
	}

	writeJSON(
		w,
		http.StatusNotFound,
		map[string]string{
			"error": "not_found",
		},
	)
}

// get handles GET /mall/brands/{id}.
func (h *MallBrandHandler) get(
	w http.ResponseWriter,
	r *http.Request,
	id string,
) {
	if id == "" {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "invalid id",
			},
		)
		return
	}

	brand, err := h.q.GetBrandDetailByID(
		r.Context(),
		id,
	)
	if err != nil {
		writeMallBrandErr(
			w,
			err,
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		brand,
	)
}

// writeMallBrandErr converts a brand domain error into an HTTP response.
func writeMallBrandErr(
	w http.ResponseWriter,
	err error,
) {
	code := http.StatusInternalServerError

	switch err {
	case branddom.ErrInvalidID:
		code = http.StatusBadRequest

	case branddom.ErrNotFound:
		code = http.StatusNotFound

	case branddom.ErrConflict:
		code = http.StatusConflict
	}

	writeJSON(
		w,
		code,
		map[string]string{
			"error": err.Error(),
		},
	)
}
