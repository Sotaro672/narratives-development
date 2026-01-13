// backend/internal/adapters/in/http/console/handler/transfer_handler.go
package consoleHandler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	common "narratives/internal/domain/common"
	transferdom "narratives/internal/domain/transfer"
)

type TransferHandler struct {
	repo transferdom.RepositoryPort
}

func NewTransferHandler(repo transferdom.RepositoryPort) http.Handler {
	return &TransferHandler{repo: repo}
}

func (h *TransferHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Preflight
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if h == nil || h.repo == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "transfer handler not configured",
		})
		return
	}

	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"error": "method not allowed",
		})
		return
	}

	// routing (very small, router-less)
	// - GET /console/transfers?...
	// - GET /console/transfers/{productId}/latest
	// - GET /console/transfers/{productId}/attempts/{attempt}
	path := strings.TrimSpace(r.URL.Path)

	// normalize trailing slash
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}

	if strings.Contains(path, "/attempts/") {
		h.handleGetAttempt(w, r, path)
		return
	}
	if strings.HasSuffix(path, "/latest") {
		h.handleGetLatest(w, r, path)
		return
	}

	h.handleList(w, r)
}

func (h *TransferHandler) handleGetLatest(w http.ResponseWriter, r *http.Request, path string) {
	// /console/transfers/{productId}/latest
	productID := strings.TrimSpace(r.URL.Query().Get("productId"))
	if productID == "" {
		// fallback from path
		productID = extractBetween(path, "/console/transfers/", "/latest")
	}
	if productID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "productId is required",
		})
		return
	}

	log.Printf("[console.transfer] get latest productId=%q", productID)

	t, err := h.repo.GetLatestByProductID(r.Context(), productID)
	if err != nil {
		if isNotFound(err) {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"error":     "not found",
				"productId": productID,
			})
			return
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			writeJSON(w, http.StatusRequestTimeout, map[string]any{
				"error":     "request canceled",
				"productId": productID,
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":     "get latest failed",
			"productId": productID,
		})
		return
	}
	if t == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error":     "not found",
			"productId": productID,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": t,
	})
}

func (h *TransferHandler) handleGetAttempt(w http.ResponseWriter, r *http.Request, path string) {
	// /console/transfers/{productId}/attempts/{attempt}
	productID := strings.TrimSpace(r.URL.Query().Get("productId"))
	if productID == "" {
		// between "/console/transfers/" and "/attempts/"
		productID = extractBetween(path, "/console/transfers/", "/attempts/")
	}
	attemptStr := strings.TrimSpace(r.URL.Query().Get("attempt"))
	if attemptStr == "" {
		// last segment after "/attempts/"
		attemptStr = extractLastPathSegment(path, "/attempts")
	}

	if productID == "" || attemptStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":     "productId and attempt are required",
			"productId": productID,
			"attempt":   attemptStr,
		})
		return
	}

	attempt, err := strconv.Atoi(attemptStr)
	if err != nil || attempt <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":     "attempt must be a positive integer",
			"productId": productID,
			"attempt":   attemptStr,
		})
		return
	}

	log.Printf("[console.transfer] get attempt productId=%q attempt=%d", productID, attempt)

	t, gerr := h.repo.GetByProductIDAndAttempt(r.Context(), productID, attempt)
	if gerr != nil {
		if isNotFound(gerr) {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"error":     "not found",
				"productId": productID,
				"attempt":   attempt,
			})
			return
		}
		if errors.Is(gerr, context.Canceled) || errors.Is(gerr, context.DeadlineExceeded) {
			writeJSON(w, http.StatusRequestTimeout, map[string]any{
				"error":     "request canceled",
				"productId": productID,
				"attempt":   attempt,
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":     "get attempt failed",
			"productId": productID,
			"attempt":   attempt,
		})
		return
	}
	if t == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error":     "not found",
			"productId": productID,
			"attempt":   attempt,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": t,
	})
}

func (h *TransferHandler) handleList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// filters
	var f transferdom.Filter

	if s := strings.TrimSpace(q.Get("id")); s != "" {
		f.ID = &s
	}
	if s := strings.TrimSpace(q.Get("productId")); s != "" {
		f.ProductID = &s
	}
	if s := strings.TrimSpace(q.Get("orderId")); s != "" {
		f.OrderID = &s
	}
	if s := strings.TrimSpace(q.Get("avatarId")); s != "" {
		f.AvatarID = &s
	}

	if s := strings.TrimSpace(q.Get("status")); s != "" {
		st := transferdom.Status(s)
		f.Status = &st
	}
	if s := strings.TrimSpace(q.Get("errorType")); s != "" {
		et := transferdom.ErrorType(s)
		f.ErrorType = &et
	}

	// sort
	sortField := strings.TrimSpace(q.Get("sortField"))
	if sortField == "" {
		sortField = "createdAt"
	}
	desc := parseBoolLoose(q.Get("desc"))
	s := transferdom.Sort{
		Field: sortField,
		Desc:  desc,
	}

	// page
	pageNum := parseIntLoose(q.Get("page"), 1)
	perPage := parseIntLoose(q.Get("perPage"), 50)
	p := common.Page{Number: pageNum, PerPage: perPage}

	log.Printf("[console.transfer] list productId=%q orderId=%q avatarId=%q status=%q errorType=%q page=%d perPage=%d sort=%s desc=%t",
		derefStr(f.ProductID), derefStr(f.OrderID), derefStr(f.AvatarID),
		func() string {
			if f.Status == nil {
				return ""
			}
			return string(*f.Status)
		}(),
		func() string {
			if f.ErrorType == nil {
				return ""
			}
			return string(*f.ErrorType)
		}(),
		p.Number, p.PerPage, s.Field, s.Desc,
	)

	res, err := h.repo.List(r.Context(), f, s, p)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			writeJSON(w, http.StatusRequestTimeout, map[string]any{
				"error": "request canceled",
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "list failed",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": res,
	})
}

// ============================================================
// helpers (local, safe)
// ============================================================

func parseIntLoose(s string, def int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

func parseBoolLoose(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "t", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	// repo 実装が独自 not found error を返す可能性があるため、文字列でも吸収
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(msg, "not found") || strings.Contains(msg, "no such") || strings.Contains(msg, "missing")
}

// extractBetween returns substring between left and right markers.
// ex: "/console/transfers/p_123/latest", left="/console/transfers/", right="/latest" => "p_123"
func extractBetween(s, left, right string) string {
	s = strings.TrimSpace(s)
	i := strings.Index(s, left)
	if i < 0 {
		return ""
	}
	s = s[i+len(left):]
	j := strings.Index(s, right)
	if j < 0 {
		return ""
	}
	return strings.Trim(strings.TrimSpace(s[:j]), "/")
}

// extractLastPathSegment returns the last segment after the given prefix.
// Example:
// path="/console/transfers/p_123/attempts/2", prefix="/attempts" => "2"
func extractLastPathSegment(path string, prefix string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	// find prefix position
	i := strings.LastIndex(path, prefix)
	if i < 0 {
		return ""
	}

	// take substring after prefix
	s := path[i+len(prefix):]
	s = strings.Trim(s, "/")
	if s == "" {
		return ""
	}

	// if still contains '/', take last part
	parts := strings.Split(s, "/")
	return strings.TrimSpace(parts[len(parts)-1])
}
