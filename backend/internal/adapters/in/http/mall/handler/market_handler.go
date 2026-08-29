// backend/internal/adapters/in/http/mall/handler/market_handler.go
package mallHandler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	middleware "narratives/internal/adapters/in/http/middleware"
	mallquery "narratives/internal/application/query/mall"
	usecase "narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	resaledom "narratives/internal/domain/resale"
	resalereview "narratives/internal/domain/resale_review"
)

type MarketHandler struct {
	marketQ        *mallquery.MarketQuery
	resaleReviewUC *usecase.ResaleReviewUsecase
}

type NewMarketHandlerParams struct {
	MarketQ        *mallquery.MarketQuery
	ResaleReviewUC *usecase.ResaleReviewUsecase
}

func NewMarketHandler(p NewMarketHandlerParams) http.Handler {
	return &MarketHandler{
		marketQ:        p.MarketQ,
		resaleReviewUC: p.ResaleReviewUC,
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
		notFound(w)
		return
	}

	rest := strings.TrimPrefix(path, marketResalesPath+"/")
	parts := strings.Split(rest, "/")
	resaleID := parts[0]

	if resaleID == "" {
		notFound(w)
		return
	}

	if len(parts) == 2 && parts[1] == "images" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.listResaleImages(w, r, resaleID)
		return
	}

	if len(parts) == 2 && parts[1] == "interactions" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.getResaleInteractions(w, r, resaleID)
		return
	}

	if len(parts) == 2 && parts[1] == "like" {
		switch r.Method {
		case http.MethodPut:
			h.addResaleLike(w, r, resaleID)
			return
		case http.MethodDelete:
			h.removeResaleLike(w, r, resaleID)
			return
		default:
			methodNotAllowed(w)
			return
		}
	}

	if len(parts) == 2 && parts[1] == "comments" {
		switch r.Method {
		case http.MethodGet:
			h.listResaleComments(w, r, resaleID)
			return
		case http.MethodPost:
			h.createResaleComment(w, r, resaleID)
			return
		default:
			methodNotAllowed(w)
			return
		}
	}

	if len(parts) != 1 {
		notFound(w)
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
		writeJSON(w, http.StatusNotImplemented, map[string]string{
			"error": "not_implemented",
		})
		return
	}

	viewerAvatarID, ok := currentMarketAvatarID(w, r)
	if !ok {
		return
	}

	filter := buildMarketResaleFilterFromQuery(r)
	filter.ExcludeAvatarIDs = []string{
		viewerAvatarID,
	}

	sortSpec := buildMarketResaleSortFromQuery(r)
	page := buildMarketResalePageFromQuery(r)

	result, err := h.marketQ.List(ctx, filter, sortSpec, page)
	if err != nil {
		writeResaleErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
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
		writeJSON(w, http.StatusNotImplemented, map[string]string{
			"error": "not_implemented",
		})
		return
	}

	viewerAvatarID, ok := currentMarketAvatarID(w, r)
	if !ok {
		return
	}

	filter := buildMarketResaleFilterFromQuery(r)
	filter.ExcludeAvatarIDs = []string{
		viewerAvatarID,
	}

	sortSpec := buildMarketResaleSortFromQuery(r)
	cpage := buildMarketResaleCursorPageFromQuery(r)

	result, err := h.marketQ.ListByCursor(ctx, filter, sortSpec, cpage)
	if err != nil {
		writeResaleErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items":      result.Items,
		"nextCursor": result.NextCursor,
		"limit":      result.Limit,
	})
}

func (h *MarketHandler) getResale(w http.ResponseWriter, r *http.Request, resaleID string) {
	ctx := r.Context()

	if h == nil || h.marketQ == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{
			"error": "not_implemented",
		})
		return
	}

	viewerAvatarID, ok := currentMarketAvatarID(w, r)
	if !ok {
		return
	}

	item, err := h.marketQ.GetByID(
		ctx,
		resaleID,
		viewerAvatarID,
	)
	if err != nil {
		writeResaleErr(w, err)
		return
	}

	if item.Status != resaledom.StatusListing {
		notFound(w)
		return
	}

	if item.AvatarID == viewerAvatarID {
		notFound(w)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": item,
	})
}

func (h *MarketHandler) listResaleImages(w http.ResponseWriter, r *http.Request, resaleID string) {
	ctx := r.Context()

	if h == nil || h.marketQ == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{
			"error": "not_implemented",
		})
		return
	}

	viewerAvatarID, ok := currentMarketAvatarID(w, r)
	if !ok {
		return
	}

	images, err := h.marketQ.ListImagesByResaleID(
		ctx,
		resaleID,
		viewerAvatarID,
	)
	if err != nil {
		writeResaleErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": images,
	})
}

func (h *MarketHandler) getResaleInteractions(w http.ResponseWriter, r *http.Request, resaleID string) {
	ctx := r.Context()

	if h == nil || h.resaleReviewUC == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{
			"error": "not_implemented",
		})
		return
	}

	avatarID, ok := currentMarketAvatarID(w, r)
	if !ok {
		return
	}

	summary, err := h.resaleReviewUC.GetSummary(ctx, resaleID, avatarID)
	if err != nil {
		writeResaleReviewErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": summary,
	})
}

func (h *MarketHandler) addResaleLike(w http.ResponseWriter, r *http.Request, resaleID string) {
	ctx := r.Context()

	if h == nil || h.resaleReviewUC == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{
			"error": "not_implemented",
		})
		return
	}

	avatarID, ok := currentMarketAvatarID(w, r)
	if !ok {
		return
	}

	summary, err := h.resaleReviewUC.AddLike(ctx, resaleID, avatarID)
	if err != nil {
		writeResaleReviewErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": summary,
	})
}

func (h *MarketHandler) removeResaleLike(w http.ResponseWriter, r *http.Request, resaleID string) {
	ctx := r.Context()

	if h == nil || h.resaleReviewUC == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{
			"error": "not_implemented",
		})
		return
	}

	avatarID, ok := currentMarketAvatarID(w, r)
	if !ok {
		return
	}

	summary, err := h.resaleReviewUC.RemoveLike(ctx, resaleID, avatarID)
	if err != nil {
		writeResaleReviewErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": summary,
	})
}

func (h *MarketHandler) listResaleComments(w http.ResponseWriter, r *http.Request, resaleID string) {
	ctx := r.Context()

	if h == nil || h.resaleReviewUC == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{
			"error": "not_implemented",
		})
		return
	}

	if _, ok := currentMarketAvatarID(w, r); !ok {
		return
	}

	result, err := h.resaleReviewUC.ListComments(
		ctx,
		resaleID,
		buildMarketResaleReviewPageFromQuery(r),
	)
	if err != nil {
		writeResaleReviewErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items":      result.Items,
		"totalCount": result.TotalCount,
		"totalPages": result.TotalPages,
		"page":       result.Page,
		"perPage":    result.PerPage,
	})
}

type marketResaleCommentRequest struct {
	Body string `json:"body"`
}

func (h *MarketHandler) createResaleComment(w http.ResponseWriter, r *http.Request, resaleID string) {
	ctx := r.Context()

	if h == nil || h.resaleReviewUC == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{
			"error": "not_implemented",
		})
		return
	}

	avatarID, ok := currentMarketAvatarID(w, r)
	if !ok {
		return
	}

	var req marketResaleCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid_json",
		})
		return
	}

	comment, summary, err := h.resaleReviewUC.CreateComment(
		ctx,
		usecase.CreateResaleReviewCommentInput{
			ResaleID: resaleID,
			AvatarID: avatarID,
			Body:     req.Body,
		},
	)
	if err != nil {
		writeResaleReviewErr(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"data":        comment,
		"interaction": summary,
	})
}

func currentMarketAvatarID(
	w http.ResponseWriter,
	r *http.Request,
) (string, bool) {
	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized: missing avatarId",
		})
		return "", false
	}

	return avatarID, true
}

func writeResaleReviewErr(w http.ResponseWriter, err error) {
	switch {
	case err == nil:
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "internal_error",
		})

	case resalereview.IsInvalid(err):
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})

	case resalereview.IsForbidden(err):
		writeJSON(w, http.StatusForbidden, map[string]string{
			"error": "forbidden",
		})

	case resalereview.IsNotFound(err):
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "not_found",
		})

	case resalereview.IsConflict(err):
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "conflict",
		})

	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "internal_error",
		})
	}
}

func isCursorMarketRequest(r *http.Request) bool {
	qp := r.URL.Query()

	mode := strings.ToLower(qp.Get("mode"))
	if mode == "cursor" {
		return true
	}

	if qp.Get("after") != "" {
		return true
	}

	if qp.Get("cursor") != "" {
		return true
	}

	return false
}

func buildMarketResaleFilterFromQuery(r *http.Request) resaledom.Filter {
	qp := r.URL.Query()

	filter := resaledom.Filter{}

	if s := qp.Get("q"); s != "" {
		filter.SearchQuery = s
	} else if s := qp.Get("search"); s != "" {
		filter.SearchQuery = s
	} else if s := qp.Get("searchQuery"); s != "" {
		filter.SearchQuery = s
	}

	if vv := qp["ids"]; len(vv) > 0 {
		for _, v := range vv {
			filter.IDs = append(filter.IDs, splitMarketResaleCSV(v)...)
		}
	}

	if vv := qp["assetIds"]; len(vv) > 0 {
		for _, v := range vv {
			filter.AssetIDs = append(filter.AssetIDs, splitMarketResaleCSV(v)...)
		}
	} else if vv := qp["asset_ids"]; len(vv) > 0 {
		for _, v := range vv {
			filter.AssetIDs = append(filter.AssetIDs, splitMarketResaleCSV(v)...)
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

	statusesRaw := qp.Get("statuses")
	if statusesRaw == "" {
		statusesRaw = qp.Get("status")
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

	conditionsRaw := qp.Get("conditions")
	if conditionsRaw == "" {
		conditionsRaw = qp.Get("condition")
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

	if v := qp.Get("minPrice"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.MinPrice = &n
		}
	}

	if v := qp.Get("maxPrice"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.MaxPrice = &n
		}
	}

	return filter
}

func buildMarketResaleSortFromQuery(r *http.Request) resaledom.Sort {
	qp := r.URL.Query()

	column := qp.Get("sort")
	if column == "" {
		column = qp.Get("sortBy")
	}
	if column == "" {
		column = qp.Get("orderBy")
	}
	if column == "" {
		column = "createdAt"
	}

	orderRaw := strings.ToLower(qp.Get("order"))
	if orderRaw == "" {
		orderRaw = strings.ToLower(qp.Get("sortOrder"))
	}
	if orderRaw == "" {
		orderRaw = strings.ToLower(qp.Get("direction"))
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

	pageNum := parsePositiveIntDefault(qp.Get("page"), 1)
	perPage := parsePositiveIntDefault(qp.Get("perPage"), 50)
	if perPage > 100 {
		perPage = 100
	}

	return resaledom.Page{
		Number:  pageNum,
		PerPage: perPage,
	}
}

func buildMarketResaleReviewPageFromQuery(r *http.Request) common.Page {
	qp := r.URL.Query()

	pageNum := parsePositiveIntDefault(qp.Get("page"), 1)
	perPage := parsePositiveIntDefault(qp.Get("perPage"), 20)
	if perPage > 100 {
		perPage = 100
	}

	return common.Page{
		Number:  pageNum,
		PerPage: perPage,
	}
}

func buildMarketResaleCursorPageFromQuery(r *http.Request) resaledom.CursorPage {
	qp := r.URL.Query()

	after := qp.Get("after")
	if after == "" {
		after = qp.Get("cursor")
	}

	limit := parsePositiveIntDefault(qp.Get("limit"), 50)
	if limit > 100 {
		limit = 100
	}

	return resaledom.CursorPage{
		After: after,
		Limit: limit,
	}
}

func splitMarketResaleCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))

	for _, part := range parts {
		if part == "" {
			continue
		}

		out = append(out, part)
	}

	return out
}
