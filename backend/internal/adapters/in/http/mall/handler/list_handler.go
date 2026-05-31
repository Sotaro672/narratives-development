// backend/internal/adapters/in/http/mall/handler/list_handler.go
package mallHandler

import (
	"errors"
	"net/http"
	"strings"

	mallquery "narratives/internal/application/query/mall"
	ldom "narratives/internal/domain/list"
)

type MallListHandler struct {
	q *mallquery.ListQuery
}

func NewMallListHandler(q *mallquery.ListQuery) http.Handler {
	return &MallListHandler{q: q}
}

func (h *MallListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimSuffix(r.URL.Path, "/")

	if path == "/mall/lists" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.listIndex(w, r)
		return
	}

	if strings.HasPrefix(path, "/mall/lists/") {
		rest := strings.TrimPrefix(path, "/mall/lists/")
		parts := strings.Split(rest, "/")
		id := parts[0]
		if id == "" {
			badRequest(w, "invalid id")
			return
		}
		if len(parts) > 1 {
			notFound(w)
			return
		}
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.get(w, r, id)
		return
	}

	notFound(w)
}

func (h *MallListHandler) listIndex(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.q == nil {
		internalError(w, "list query is nil")
		return
	}

	qp := r.URL.Query()
	pageNum := parseIntDefault(qp.Get("page"), 1)
	perPage := parseIntDefault(qp.Get("perPage"), 20)

	resp, err := h.q.ListIndex(r.Context(), pageNum, perPage)
	if err != nil {
		writeMallListErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *MallListHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	if h == nil || h.q == nil {
		internalError(w, "list query is nil")
		return
	}

	dto, err := h.q.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ldom.ErrNotFound) {
			notFound(w)
			return
		}

		writeMallListErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto)
}

func writeMallListErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, ldom.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, ldom.ErrConflict):
		code = http.StatusConflict
	default:
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "invalid") ||
			strings.Contains(msg, "required") ||
			strings.Contains(msg, "must") {
			code = http.StatusBadRequest
		}
	}

	writeJSON(w, code, map[string]string{"error": err.Error()})
}
