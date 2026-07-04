// backend/internal/adapters/in/http/mall/handler/market_handler.go
package mallHandler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	mallquery "narratives/internal/application/query/mall"
	resaledom "narratives/internal/domain/resale"
)

type MarketHandler struct {
	marketQ *mallquery.MarketQuery
}

type NewMarketHandlerParams struct {
	MarketQ *mallquery.MarketQuery
}

func NewMarketHandler(p NewMarketHandlerParams) http.Handler {
	return &MarketHandler{
		marketQ: p.MarketQ,
	}
}

const marketResalesPath = "/mall/market/resales"

func (h *MarketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")

	if path == marketResalesPath {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		if isCursorMarketRequest(r) {
			h.listResalesByCursor(w, r)
			return
		}

		h.listResales(w, r)
		return
	}

	if path == marketResalesPath+"/cursor" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.listResalesByCursor(w, r)
		return
	}

	if !strings.HasPrefix(path, marketResalesPath+"/") {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	rest := strings.TrimPrefix(path, marketResalesPath+"/")
	parts := strings.Split(rest, "/")
	resaleID := strings.TrimSpace(parts[0])

	if resaleID == "" || len(parts) != 1 {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	h.getResale(w, r, resaleID)
}

func (h *MarketHandler) listResales(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.marketQ == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
		return
	}

	filter := buildMarketResaleFilterFromQuery(r)
	filter = attachViewerAvatarIDsToMarketFilter(r, filter)

	sortSpec := buildMarketResaleSortFromQuery(r)
	page := buildMarketResalePageFromQuery(r)

	result, err := h.marketQ.List(ctx, filter, sortSpec, page)
	if err != nil {
		writeResaleErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"items":      result.Items,
		"totalCount": result.TotalCount,
		"totalPages": result.TotalPages,
		"page":       result.Page,
		"perPage":    result.PerPage,
	})
}

func (h *MarketHandler) listResalesByCursor(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.marketQ == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
		return
	}

	filter := buildMarketResaleFilterFromQuery(r)
	filter = attachViewerAvatarIDsToMarketFilter(r, filter)

	sortSpec := buildMarketResaleSortFromQuery(r)
	cpage := buildMarketResaleCursorPageFromQuery(r)

	result, err := h.marketQ.ListByCursor(ctx, filter, sortSpec, cpage)
	if err != nil {
		writeResaleErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"items":      result.Items,
		"nextCursor": result.NextCursor,
		"limit":      result.Limit,
	})
}

func (h *MarketHandler) getResale(w http.ResponseWriter, r *http.Request, resaleID string) {
	ctx := r.Context()

	if h == nil || h.marketQ == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
		return
	}

	item, err := h.marketQ.GetByID(ctx, resaleID)
	if err != nil {
		writeResaleErr(w, err)
		return
	}

	if item.Status != resaledom.StatusListing {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": item,
	})
}

func isCursorMarketRequest(r *http.Request) bool {
	qp := r.URL.Query()

	mode := strings.ToLower(strings.TrimSpace(qp.Get("mode")))
	if mode == "cursor" {
		return true
	}

	if strings.TrimSpace(qp.Get("after")) != "" {
		return true
	}

	if strings.TrimSpace(qp.Get("cursor")) != "" {
		return true
	}

	return false
}

func buildMarketResaleFilterFromQuery(r *http.Request) resaledom.Filter {
	qp := r.URL.Query()

	filter := resaledom.Filter{}

	if s := strings.TrimSpace(qp.Get("q")); s != "" {
		filter.SearchQuery = s
	} else if s := strings.TrimSpace(qp.Get("search")); s != "" {
		filter.SearchQuery = s
	} else if s := strings.TrimSpace(qp.Get("searchQuery")); s != "" {
		filter.SearchQuery = s
	}

	if vv := qp["ids"]; len(vv) > 0 {
		for _, v := range vv {
			filter.IDs = append(filter.IDs, splitMarketResaleCSV(v)...)
		}
	}

	if vv := qp["mintAddresses"]; len(vv) > 0 {
		for _, v := range vv {
			filter.MintAddresses = append(filter.MintAddresses, splitMarketResaleCSV(v)...)
		}
	} else if vv := qp["mint_addresses"]; len(vv) > 0 {
		for _, v := range vv {
			filter.MintAddresses = append(filter.MintAddresses, splitMarketResaleCSV(v)...)
		}
	}

	if vv := qp["tokenBlueprintIds"]; len(vv) > 0 {
		for _, v := range vv {
			filter.TokenBlueprintIDs = append(filter.TokenBlueprintIDs, splitMarketResaleCSV(v)...)
		}
	} else if vv := qp["token_blueprint_ids"]; len(vv) > 0 {
		for _, v := range vv {
			filter.TokenBlueprintIDs = append(filter.TokenBlueprintIDs, splitMarketResaleCSV(v)...)
		}
	}

	if vv := qp["productIds"]; len(vv) > 0 {
		for _, v := range vv {
			filter.ProductIDs = append(filter.ProductIDs, splitMarketResaleCSV(v)...)
		}
	} else if vv := qp["product_ids"]; len(vv) > 0 {
		for _, v := range vv {
			filter.ProductIDs = append(filter.ProductIDs, splitMarketResaleCSV(v)...)
		}
	}

	if vv := qp["brandIds"]; len(vv) > 0 {
		for _, v := range vv {
			filter.BrandIDs = append(filter.BrandIDs, splitMarketResaleCSV(v)...)
		}
	} else if vv := qp["brand_ids"]; len(vv) > 0 {
		for _, v := range vv {
			filter.BrandIDs = append(filter.BrandIDs, splitMarketResaleCSV(v)...)
		}
	}

	if vv := qp["productBlueprintIds"]; len(vv) > 0 {
		for _, v := range vv {
			filter.ProductBlueprintIDs = append(filter.ProductBlueprintIDs, splitMarketResaleCSV(v)...)
		}
	} else if vv := qp["product_blueprint_ids"]; len(vv) > 0 {
		for _, v := range vv {
			filter.ProductBlueprintIDs = append(filter.ProductBlueprintIDs, splitMarketResaleCSV(v)...)
		}
	}

	if vv := qp["avatarIds"]; len(vv) > 0 {
		for _, v := range vv {
			filter.AvatarIDs = append(filter.AvatarIDs, splitMarketResaleCSV(v)...)
		}
	} else if vv := qp["avatar_ids"]; len(vv) > 0 {
		for _, v := range vv {
			filter.AvatarIDs = append(filter.AvatarIDs, splitMarketResaleCSV(v)...)
		}
	}

	statusesRaw := strings.TrimSpace(qp.Get("statuses"))
	if statusesRaw == "" {
		statusesRaw = strings.TrimSpace(qp.Get("status"))
	}

	if statusesRaw != "" {
		statuses := splitMarketResaleCSV(statusesRaw)
		if len(statuses) == 1 {
			status := resaledom.ResaleStatus(statuses[0])
			if status != "" {
				filter.Status = &status
			}
		} else if len(statuses) > 1 {
			out := make([]resaledom.ResaleStatus, 0, len(statuses))
			for _, s := range statuses {
				status := resaledom.ResaleStatus(s)
				if status != "" {
					out = append(out, status)
				}
			}
			filter.Statuses = out
		}
	} else {
		status := resaledom.StatusListing
		filter.Status = &status
	}

	conditionsRaw := strings.TrimSpace(qp.Get("conditions"))
	if conditionsRaw == "" {
		conditionsRaw = strings.TrimSpace(qp.Get("condition"))
	}

	if conditionsRaw != "" {
		conditions := splitMarketResaleCSV(conditionsRaw)
		if len(conditions) == 1 {
			condition := resaledom.ResaleCondition(conditions[0])
			if condition != "" {
				filter.Condition = &condition
			}
		} else if len(conditions) > 1 {
			out := make([]resaledom.ResaleCondition, 0, len(conditions))
			for _, s := range conditions {
				condition := resaledom.ResaleCondition(s)
				if condition != "" {
					out = append(out, condition)
				}
			}
			filter.Conditions = out
		}
	}

	if v := strings.TrimSpace(qp.Get("minPrice")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.MinPrice = &n
		}
	}

	if v := strings.TrimSpace(qp.Get("maxPrice")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.MaxPrice = &n
		}
	}

	return filter
}

// attachViewerAvatarIDsToMarketFilter attaches viewer avatar ids for exclusion.
//
// UserAuthMiddleware provides Firebase UID, not avatarId.
// Since resale.AvatarID is avatar id, this handler cannot derive it from uid alone.
// Therefore the frontend should pass viewerAvatarId/avatarId/avatarIds as query params.
func attachViewerAvatarIDsToMarketFilter(
	r *http.Request,
	filter resaledom.Filter,
) resaledom.Filter {
	qp := r.URL.Query()

	viewerIDs := make([]string, 0)

	if vv := qp["viewerAvatarIds"]; len(vv) > 0 {
		for _, v := range vv {
			viewerIDs = append(viewerIDs, splitMarketResaleCSV(v)...)
		}
	} else if vv := qp["viewer_avatar_ids"]; len(vv) > 0 {
		for _, v := range vv {
			viewerIDs = append(viewerIDs, splitMarketResaleCSV(v)...)
		}
	}

	if v := strings.TrimSpace(qp.Get("viewerAvatarId")); v != "" {
		viewerIDs = append(viewerIDs, v)
	} else if v := strings.TrimSpace(qp.Get("viewer_avatar_id")); v != "" {
		viewerIDs = append(viewerIDs, v)
	}

	if v := strings.TrimSpace(qp.Get("avatarId")); v != "" {
		viewerIDs = append(viewerIDs, v)
	} else if v := strings.TrimSpace(qp.Get("avatar_id")); v != "" {
		viewerIDs = append(viewerIDs, v)
	}

	if len(viewerIDs) == 0 {
		return filter
	}

	filter.AvatarIDs = appendUniqueMarketResaleStrings(filter.AvatarIDs, viewerIDs...)

	return filter
}

func appendUniqueMarketResaleStrings(base []string, values ...string) []string {
	seen := make(map[string]struct{}, len(base)+len(values))
	out := make([]string, 0, len(base)+len(values))

	for _, v := range base {
		normalized := strings.TrimSpace(v)
		if normalized == "" {
			continue
		}

		if _, ok := seen[normalized]; ok {
			continue
		}

		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}

	for _, v := range values {
		normalized := strings.TrimSpace(v)
		if normalized == "" {
			continue
		}

		if _, ok := seen[normalized]; ok {
			continue
		}

		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}

	return out
}

func buildMarketResaleSortFromQuery(r *http.Request) resaledom.Sort {
	qp := r.URL.Query()

	column := strings.TrimSpace(qp.Get("sort"))
	if column == "" {
		column = strings.TrimSpace(qp.Get("sortBy"))
	}
	if column == "" {
		column = strings.TrimSpace(qp.Get("orderBy"))
	}
	if column == "" {
		column = "createdAt"
	}

	orderRaw := strings.ToLower(strings.TrimSpace(qp.Get("order")))
	if orderRaw == "" {
		orderRaw = strings.ToLower(strings.TrimSpace(qp.Get("sortOrder")))
	}
	if orderRaw == "" {
		orderRaw = strings.ToLower(strings.TrimSpace(qp.Get("direction")))
	}

	order := resaledom.SortDesc
	if orderRaw == "asc" || orderRaw == string(resaledom.SortAsc) {
		order = resaledom.SortAsc
	}

	return resaledom.Sort{
		Column: column,
		Order:  order,
	}
}

func buildMarketResalePageFromQuery(r *http.Request) resaledom.Page {
	qp := r.URL.Query()

	pageNum := parseMarketResalePositiveInt(qp.Get("page"), 1)
	perPage := parseMarketResalePositiveInt(qp.Get("perPage"), 50)
	if perPage > 100 {
		perPage = 100
	}

	return resaledom.Page{
		Number:  pageNum,
		PerPage: perPage,
	}
}

func buildMarketResaleCursorPageFromQuery(r *http.Request) resaledom.CursorPage {
	qp := r.URL.Query()

	after := strings.TrimSpace(qp.Get("after"))
	if after == "" {
		after = strings.TrimSpace(qp.Get("cursor"))
	}

	limit := parseMarketResalePositiveInt(qp.Get("limit"), 50)
	if limit > 100 {
		limit = 100
	}

	return resaledom.CursorPage{
		After: after,
		Limit: limit,
	}
}

func parseMarketResalePositiveInt(raw string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n <= 0 {
		return fallback
	}

	return n
}

func splitMarketResaleCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))

	for _, part := range parts {
		v := strings.TrimSpace(part)
		if v == "" {
			continue
		}

		out = append(out, v)
	}

	return out
}
