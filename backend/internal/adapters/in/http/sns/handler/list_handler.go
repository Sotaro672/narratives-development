// backend/internal/adapters/in/http/sns/handler/list_handler.go
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	snsquery "narratives/internal/application/query/sns"
	usecase "narratives/internal/application/usecase"
	ldom "narratives/internal/domain/list"
)

// SNSListHandler serves buyer-facing list endpoints.
// Only returns: title, description, image(url), prices.
//
// Routes:
// - GET /sns/lists
// - GET /sns/lists/{id}
type SNSListHandler struct {
	// optional (legacy/fallback; SNSでは基本使わない)
	uc *usecase.ListUsecase

	// ✅ SNS専用Query（companyId境界なし・status=listingのみ）
	q *snsquery.SNSListQuery
}

// NewSNSListHandler keeps backward compatibility.
// NOTE: SNSでは companyId が無いので、この ctor だけだと期待通り動かない可能性があります。
// 可能なら NewSNSListHandlerWithQueries を使って q を注入してください。
func NewSNSListHandler(uc *usecase.ListUsecase) http.Handler {
	return &SNSListHandler{uc: uc, q: nil}
}

// NewSNSListHandlerWithQueries injects SNS query (preferred).
// - uc は nil でもOK
func NewSNSListHandlerWithQueries(
	uc *usecase.ListUsecase,
	q *snsquery.SNSListQuery,
) http.Handler {
	return &SNSListHandler{uc: uc, q: q}
}

// ------------------------------
// Response DTOs (SNS)
// ------------------------------

type SnsListItem struct {
	ID          string              `json:"id,omitempty"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Image       string              `json:"image"` // List.ImageID (URL)
	Prices      []ldom.ListPriceRow `json:"prices"`
}

type SnsListIndexResponse struct {
	Items      []SnsListItem `json:"items"`
	TotalCount int           `json:"totalCount"`
	TotalPages int           `json:"totalPages"`
	Page       int           `json:"page"`
	PerPage    int           `json:"perPage"`
}

// ------------------------------
// http.Handler
// ------------------------------

func (h *SNSListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimSuffix(strings.TrimSpace(r.URL.Path), "/")

	// GET /sns/lists
	if path == "/sns/lists" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		h.listIndex(w, r)
		return
	}

	// GET /sns/lists/{id}
	if strings.HasPrefix(path, "/sns/lists/") {
		rest := strings.TrimPrefix(path, "/sns/lists/")
		parts := strings.Split(rest, "/")
		id := strings.TrimSpace(parts[0])
		if id == "" {
			badRequest(w, "invalid id")
			return
		}
		// SNS: no sub routes for now
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

// ------------------------------
// GET /sns/lists
// ------------------------------

func (h *SNSListHandler) listIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// SNSは company boundary を使わない Query が必須
	if h == nil || h.q == nil {
		internalError(w, "sns query is nil (inject snsquery.SNSListQuery)")
		return
	}

	qp := r.URL.Query()
	pageNum := parseIntDefault(qp.Get("page"), 1)
	perPage := parseIntDefault(qp.Get("perPage"), 20)
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 200 {
		perPage = 200
	}

	dto, err := h.q.ListListing(ctx, pageNum, perPage)
	if err != nil {
		writeListErr(w, err)
		return
	}

	items := make([]SnsListItem, 0, len(dto.Items))
	for _, it := range dto.Items {
		items = append(items, SnsListItem{
			ID:          strings.TrimSpace(it.ID),
			Title:       strings.TrimSpace(it.Title),
			Description: strings.TrimSpace(it.Description),
			Image:       strings.TrimSpace(it.Image),
			Prices:      it.Prices,
		})
	}

	writeJSON(w, http.StatusOK, SnsListIndexResponse{
		Items:      items,
		TotalCount: dto.TotalCount,
		TotalPages: dto.TotalPages,
		Page:       dto.Page,
		PerPage:    dto.PerPage,
	})
}

// ------------------------------
// GET /sns/lists/{id}
// ------------------------------

func (h *SNSListHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if h == nil || h.q == nil {
		internalError(w, "sns query is nil (inject snsquery.SNSListQuery)")
		return
	}

	dto, err := h.q.GetListingDetail(ctx, id)
	if err != nil {
		if errors.Is(err, ldom.ErrNotFound) {
			notFound(w)
			return
		}
		writeListErr(w, err)
		return
	}

	// detailは最小情報のみ（IDは不要だが、クライアント都合で付けても問題ないので付ける）
	writeJSON(w, http.StatusOK, SnsListItem{
		ID:          strings.TrimSpace(id),
		Title:       strings.TrimSpace(dto.Title),
		Description: strings.TrimSpace(dto.Description),
		Image:       strings.TrimSpace(dto.Image),
		Prices:      dto.Prices,
	})
}

// ------------------------------
// Helpers
// ------------------------------

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func methodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
}

func notFound(w http.ResponseWriter) {
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
}

func badRequest(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": strings.TrimSpace(msg)})
}

func internalError(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": strings.TrimSpace(msg)})
}

func parseIntDefault(s string, def int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

// SNS用の最低限のエラー変換
func writeListErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, ldom.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, ldom.ErrConflict):
		code = http.StatusConflict
	default:
		msg := strings.ToLower(strings.TrimSpace(err.Error()))
		if strings.Contains(msg, "invalid") ||
			strings.Contains(msg, "required") ||
			strings.Contains(msg, "must") {
			code = http.StatusBadRequest
		}
	}

	writeJSON(w, code, map[string]string{"error": err.Error()})
}
