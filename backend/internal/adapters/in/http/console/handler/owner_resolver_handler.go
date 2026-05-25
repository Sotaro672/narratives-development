// backend/internal/adapters/in/http/console/handler/owner_resolver_handler.go
package consoleHandler

import (
	"encoding/json"
	"net/http"
	"strings"

	sharedquery "narratives/internal/application/query/shared"
)

type OwnerResolveHandler struct {
	q *sharedquery.OwnerResolveQuery
}

func NewOwnerResolveHandler(q *sharedquery.OwnerResolveQuery) http.Handler {
	return &OwnerResolveHandler{q: q}
}

func (h *OwnerResolveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if h == nil || h.q == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": sharedquery.ErrOwnerResolveNotConfigured.Error(),
		})
		return
	}

	addr := resolveOwnerAddressFromQuery(r)
	if addr == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": "walletAddress (or toAddress/address) is required",
		})
		return
	}

	res, err := h.q.Resolve(r.Context(), addr)
	if err != nil {
		writeOwnerResolveErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": res,
	})
}

func resolveOwnerAddressFromQuery(r *http.Request) string {
	q := r.URL.Query()

	addr := strings.Trim(q.Get("walletAddress"), " \t\r\n")
	if addr != "" {
		return addr
	}

	addr = strings.Trim(q.Get("toAddress"), " \t\r\n")
	if addr != "" {
		return addr
	}

	return strings.Trim(q.Get("address"), " \t\r\n")
}

func writeOwnerResolveErr(w http.ResponseWriter, err error) {
	switch err {
	case sharedquery.ErrInvalidWalletAddress:
		w.WriteHeader(http.StatusBadRequest)
	case sharedquery.ErrOwnerNotFound:
		w.WriteHeader(http.StatusNotFound)
	case sharedquery.ErrOwnerResolveNotConfigured:
		w.WriteHeader(http.StatusServiceUnavailable)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": err.Error(),
	})
}
