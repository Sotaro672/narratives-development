// backend/internal/adapters/in/http/mall/handler/resale_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"narratives/internal/adapters/in/http/middleware"
	usecase "narratives/internal/application/usecase"
	resaledom "narratives/internal/domain/resale"
)

// ResaleQuery is the read-side port used by mall resale handler.
// resaledom.Repository satisfies this interface.
type ResaleQuery interface {
	List(ctx context.Context, filter resaledom.Filter, sort resaledom.Sort, page resaledom.Page) (resaledom.PageResult[resaledom.Resale], error)
	ListByAvatarID(ctx context.Context, avatarID string) ([]resaledom.Resale, error)
	GetByID(ctx context.Context, id string) (resaledom.Resale, error)
}

type ResaleHandler struct {
	uc    *usecase.ResaleUsecase
	query ResaleQuery
}

type NewResaleHandlerParams struct {
	UC    *usecase.ResaleUsecase
	Query ResaleQuery
}

func NewResaleHandler(p NewResaleHandlerParams) http.Handler {
	return &ResaleHandler{
		uc:    p.UC,
		query: p.Query,
	}
}

const meResalesPath = "/mall/me/resales"

func (h *ResaleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")

	if path == meResalesPath {
		switch r.Method {
		case http.MethodPost:
			h.create(w, r)
			return
		case http.MethodGet:
			h.listIndex(w, r)
			return
		default:
			methodNotAllowed(w)
			return
		}
	}

	if !strings.HasPrefix(path, meResalesPath+"/") {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	rest := strings.TrimPrefix(path, meResalesPath+"/")
	parts := strings.Split(rest, "/")
	resaleID := strings.TrimSpace(parts[0])
	if resaleID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid resaleId"})
		return
	}

	if len(parts) > 1 {
		switch parts[1] {
		case "images", "condition-images":
			imageID := ""
			if len(parts) >= 3 {
				imageID = strings.TrimSpace(parts[2])
			}

			if len(parts) == 2 {
				switch r.Method {
				case http.MethodGet:
					h.listImages(w, r, resaleID)
					return
				case http.MethodPost:
					h.createImageFromFirebaseStorage(w, r, resaleID)
					return
				default:
					methodNotAllowed(w)
					return
				}
			}

			if len(parts) == 3 && imageID != "" {
				if r.Method == http.MethodDelete {
					h.deleteImage(w, r, resaleID, imageID)
					return
				}

				methodNotAllowed(w)
				return
			}

			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return

		case "primary-image":
			if r.Method != http.MethodPut {
				methodNotAllowed(w)
				return
			}

			h.setPrimaryImage(w, r, resaleID)
			return

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		h.get(w, r, resaleID)
		return
	case http.MethodPut:
		h.update(w, r, resaleID)
		return
	case http.MethodDelete:
		h.delete(w, r, resaleID)
		return
	default:
		methodNotAllowed(w)
		return
	}
}

func (h *ResaleHandler) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "resale usecase is nil"})
		return
	}

	avatarID, ok := currentResaleAvatarIDFromRequest(w, r)
	if !ok {
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid body"})
		return
	}

	var item resaledom.Resale
	if err := json.Unmarshal(body, &item); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	item.MintAddress = strings.TrimSpace(item.MintAddress)
	item.TokenBlueprintID = strings.TrimSpace(item.TokenBlueprintID)
	item.ProductID = strings.TrimSpace(item.ProductID)
	item.BrandID = strings.TrimSpace(item.BrandID)
	item.ProductBlueprintID = strings.TrimSpace(item.ProductBlueprintID)
	item.AvatarID = avatarID
	item.Description = strings.TrimSpace(item.Description)

	if item.MintAddress == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "mintAddress is required"})
		return
	}
	if item.TokenBlueprintID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "tokenBlueprintId is required"})
		return
	}
	if item.ProductID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "productId is required"})
		return
	}
	if item.Price <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "price must be greater than 0"})
		return
	}

	if item.Status == "" {
		item.Status = resaledom.StatusListing
	}
	if item.Condition == "" {
		item.Condition = resaledom.ConditionLikeNew
	}
	item.CreatedBy = avatarID

	now := time.Now().UTC()
	item.CreatedAt = now
	item.UpdatedAt = &now
	item.UpdatedBy = nil

	created, err := h.uc.Create(ctx, item)
	if err != nil {
		writeResaleErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": created,
	})
}

func (h *ResaleHandler) listIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.query == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
		return
	}

	avatarID, ok := currentResaleAvatarIDFromRequest(w, r)
	if !ok {
		return
	}

	items, err := h.query.ListByAvatarID(ctx, avatarID)
	if err != nil {
		writeResaleErr(w, err)
		return
	}

	page := buildResalePageFromQuery(r)

	pageNum := page.Number
	if pageNum <= 0 {
		pageNum = 1
	}

	perPage := page.PerPage
	if perPage <= 0 {
		perPage = 50
	}
	if perPage > 100 {
		perPage = 100
	}

	totalCount := len(items)
	totalPages := 0
	if totalCount > 0 {
		totalPages = (totalCount + perPage - 1) / perPage
	}

	offset := (pageNum - 1) * perPage
	if offset < 0 {
		offset = 0
	}

	pagedItems := []resaledom.Resale{}
	if offset < totalCount {
		end := offset + perPage
		if end > totalCount {
			end = totalCount
		}
		pagedItems = items[offset:end]
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"items":      pagedItems,
		"totalCount": totalCount,
		"totalPages": totalPages,
		"page":       pageNum,
		"perPage":    perPage,
	})
}

func (h *ResaleHandler) get(w http.ResponseWriter, r *http.Request, resaleID string) {
	ctx := r.Context()

	item, ok := h.getOwnedResale(w, r, ctx, resaleID)
	if !ok {
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": item,
	})
}

func (h *ResaleHandler) update(w http.ResponseWriter, r *http.Request, resaleID string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "resale usecase is nil"})
		return
	}

	avatarID, ok := currentResaleAvatarIDFromRequest(w, r)
	if !ok {
		return
	}

	existing, ok := h.getOwnedResale(w, r, ctx, resaleID)
	if !ok {
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid body"})
		return
	}

	var item resaledom.Resale
	if err := json.Unmarshal(body, &item); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	now := time.Now().UTC()
	updatedBy := avatarID

	item.ID = resaleID
	item.AvatarID = avatarID
	item.MintAddress = existing.MintAddress
	item.TokenBlueprintID = existing.TokenBlueprintID
	item.ProductID = existing.ProductID
	item.BrandID = existing.BrandID
	item.ProductBlueprintID = existing.ProductBlueprintID
	item.ImageID = existing.ImageID
	item.Description = strings.TrimSpace(item.Description)
	item.CreatedAt = existing.CreatedAt
	item.CreatedBy = existing.CreatedBy
	item.UpdatedAt = &now
	item.UpdatedBy = &updatedBy

	if item.Price <= 0 {
		item.Price = existing.Price
	}
	if item.Status == "" {
		item.Status = existing.Status
	}
	if item.Condition == "" {
		item.Condition = existing.Condition
	}

	updated, err := h.uc.Update(ctx, item)
	if err != nil {
		writeResaleErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": updated,
	})
}

func (h *ResaleHandler) delete(w http.ResponseWriter, r *http.Request, resaleID string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "resale usecase is nil"})
		return
	}

	if _, ok := h.getOwnedResale(w, r, ctx, resaleID); !ok {
		return
	}

	if err := h.uc.Delete(ctx, resaleID); err != nil {
		writeResaleErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":       true,
		"resaleId": resaleID,
	})
}

func (h *ResaleHandler) listImages(w http.ResponseWriter, r *http.Request, resaleID string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "resale usecase is nil"})
		return
	}

	if _, ok := h.getOwnedResale(w, r, ctx, resaleID); !ok {
		return
	}

	images, err := h.uc.ListImages(ctx, resaleID)
	if err != nil {
		writeResaleErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": images,
	})
}

// createImageFromFirebaseStorage stores a resale condition image record.
//
// Current policy:
// - frontend uploads images directly to Firebase Storage.
// - backend receives and stores only the Firebase Storage download URL.
// - backend does not validate or persist objectPath, fileName, contentType, or size.
func (h *ResaleHandler) createImageFromFirebaseStorage(w http.ResponseWriter, r *http.Request, resaleID string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "resale usecase is nil"})
		return
	}

	avatarID, ok := currentResaleAvatarIDFromRequest(w, r)
	if !ok {
		return
	}

	if _, ok := h.getOwnedResale(w, r, ctx, resaleID); !ok {
		return
	}

	var req struct {
		ID           string `json:"id"`
		URL          string `json:"url"`
		DisplayOrder int    `json:"displayOrder"`
		CreatedBy    string `json:"createdBy,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	req.ID = strings.TrimSpace(req.ID)
	req.URL = strings.TrimSpace(req.URL)
	req.CreatedBy = avatarID

	if req.ID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "id is required"})
		return
	}

	if strings.Contains(req.ID, "/") || strings.Contains(req.ID, "://") {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid image id"})
		return
	}

	if req.URL == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "url is required"})
		return
	}

	if req.DisplayOrder < 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "displayOrder must be >= 0"})
		return
	}

	now := time.Now().UTC()

	img, err := h.uc.CreateImage(
		ctx,
		resaledom.ResaleConditionImage{
			ID:           req.ID,
			ResaleID:     resaleID,
			URL:          req.URL,
			DisplayOrder: req.DisplayOrder,
			CreatedAt:    now,
			CreatedBy:    req.CreatedBy,
		},
	)
	if err != nil {
		writeResaleErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": img,
	})
}

func (h *ResaleHandler) deleteImage(w http.ResponseWriter, r *http.Request, resaleID string, imageID string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "resale usecase is nil"})
		return
	}

	if _, ok := h.getOwnedResale(w, r, ctx, resaleID); !ok {
		return
	}

	if imageID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "imageId is required"})
		return
	}

	if err := h.uc.DeleteImage(ctx, resaleID, imageID); err != nil {
		writeResaleErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":       true,
		"resaleId": resaleID,
		"imageId":  imageID,
	})
}

func (h *ResaleHandler) setPrimaryImage(w http.ResponseWriter, r *http.Request, resaleID string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "resale usecase is nil"})
		return
	}

	avatarID, ok := currentResaleAvatarIDFromRequest(w, r)
	if !ok {
		return
	}

	if _, ok := h.getOwnedResale(w, r, ctx, resaleID); !ok {
		return
	}

	var req struct {
		ImageID   string  `json:"imageId"`
		UpdatedBy *string `json:"updatedBy"`
		Now       *string `json:"now"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	imageID := strings.TrimSpace(req.ImageID)
	if imageID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "imageId is required"})
		return
	}

	req.UpdatedBy = &avatarID

	now := time.Now().UTC()
	if req.Now != nil && strings.TrimSpace(*req.Now) != "" {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.Now)); err == nil {
			now = t.UTC()
		}
	}

	item, err := h.uc.SetPrimaryImage(ctx, resaleID, imageID, now, req.UpdatedBy)
	if err != nil {
		writeResaleErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": item,
	})
}

func (h *ResaleHandler) getOwnedResale(
	w http.ResponseWriter,
	r *http.Request,
	ctx context.Context,
	resaleID string,
) (resaledom.Resale, bool) {
	if h == nil || h.query == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
		return resaledom.Resale{}, false
	}

	avatarID, ok := currentResaleAvatarIDFromRequest(w, r)
	if !ok {
		return resaledom.Resale{}, false
	}

	item, err := h.query.GetByID(ctx, resaleID)
	if err != nil {
		writeResaleErr(w, err)
		return resaledom.Resale{}, false
	}

	if item.AvatarID != avatarID {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "resale_access_denied"})
		return resaledom.Resale{}, false
	}

	return item, true
}

func currentResaleAvatarIDFromRequest(w http.ResponseWriter, r *http.Request) (string, bool) {
	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || strings.TrimSpace(avatarID) == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar context is required"})
		return "", false
	}

	return strings.TrimSpace(avatarID), true
}

func buildResaleFilterFromQuery(r *http.Request) resaledom.Filter {
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
			filter.IDs = append(filter.IDs, splitResaleCSV(v)...)
		}
	}

	if vv := qp["mintAddresses"]; len(vv) > 0 {
		for _, v := range vv {
			filter.MintAddresses = append(filter.MintAddresses, splitResaleCSV(v)...)
		}
	} else if vv := qp["mint_addresses"]; len(vv) > 0 {
		for _, v := range vv {
			filter.MintAddresses = append(filter.MintAddresses, splitResaleCSV(v)...)
		}
	}

	if vv := qp["tokenBlueprintIds"]; len(vv) > 0 {
		for _, v := range vv {
			filter.TokenBlueprintIDs = append(filter.TokenBlueprintIDs, splitResaleCSV(v)...)
		}
	} else if vv := qp["token_blueprint_ids"]; len(vv) > 0 {
		for _, v := range vv {
			filter.TokenBlueprintIDs = append(filter.TokenBlueprintIDs, splitResaleCSV(v)...)
		}
	}

	if vv := qp["productIds"]; len(vv) > 0 {
		for _, v := range vv {
			filter.ProductIDs = append(filter.ProductIDs, splitResaleCSV(v)...)
		}
	} else if vv := qp["product_ids"]; len(vv) > 0 {
		for _, v := range vv {
			filter.ProductIDs = append(filter.ProductIDs, splitResaleCSV(v)...)
		}
	}

	if vv := qp["brandIds"]; len(vv) > 0 {
		for _, v := range vv {
			filter.BrandIDs = append(filter.BrandIDs, splitResaleCSV(v)...)
		}
	} else if vv := qp["brand_ids"]; len(vv) > 0 {
		for _, v := range vv {
			filter.BrandIDs = append(filter.BrandIDs, splitResaleCSV(v)...)
		}
	}

	statusesRaw := strings.TrimSpace(qp.Get("statuses"))
	if statusesRaw == "" {
		statusesRaw = strings.TrimSpace(qp.Get("status"))
	}
	if statusesRaw != "" {
		statuses := splitResaleCSV(statusesRaw)
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
	}

	conditionsRaw := strings.TrimSpace(qp.Get("conditions"))
	if conditionsRaw == "" {
		conditionsRaw = strings.TrimSpace(qp.Get("condition"))
	}
	if conditionsRaw != "" {
		conditions := splitResaleCSV(conditionsRaw)
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

func buildResalePageFromQuery(r *http.Request) resaledom.Page {
	qp := r.URL.Query()

	pageNum := parseResalePositiveInt(qp.Get("page"), 1)
	perPage := parseResalePositiveInt(qp.Get("perPage"), 50)
	if perPage > 100 {
		perPage = 100
	}

	return resaledom.Page{
		Number:  pageNum,
		PerPage: perPage,
	}
}

func parseResalePositiveInt(raw string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n <= 0 {
		return fallback
	}

	return n
}

func splitResaleCSV(raw string) []string {
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

func writeResaleErr(w http.ResponseWriter, err error) {
	if err == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	msg := err.Error()

	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		w.WriteHeader(http.StatusRequestTimeout)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "request_timeout"})
		return

	case errors.Is(err, resaledom.ErrNotFound),
		errors.Is(err, resaledom.ErrConditionImageNotFound):
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return

	case errors.Is(err, resaledom.ErrConflict),
		errors.Is(err, resaledom.ErrConditionImageConflict):
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return

	case errors.Is(err, resaledom.ErrInvalidID),
		errors.Is(err, resaledom.ErrInvalidStatus),
		errors.Is(err, resaledom.ErrInvalidMintAddress),
		errors.Is(err, resaledom.ErrInvalidTokenBlueprintID),
		errors.Is(err, resaledom.ErrInvalidProductID),
		errors.Is(err, resaledom.ErrInvalidBrandID),
		errors.Is(err, resaledom.ErrInvalidProductBlueprintID),
		errors.Is(err, resaledom.ErrInvalidAvatarID),
		errors.Is(err, resaledom.ErrInvalidPrice),
		errors.Is(err, resaledom.ErrInvalidCondition),
		errors.Is(err, resaledom.ErrInvalidDescription),
		errors.Is(err, resaledom.ErrInvalidCreatedBy),
		errors.Is(err, resaledom.ErrInvalidCreatedAt),
		errors.Is(err, resaledom.ErrInvalidUpdatedAt),
		errors.Is(err, resaledom.ErrInvalidUpdatedBy),
		errors.Is(err, resaledom.ErrEmptyImageID),
		errors.Is(err, resaledom.ErrInvalidImageID),
		errors.Is(err, resaledom.ErrInvalidConditionImageID),
		errors.Is(err, resaledom.ErrInvalidConditionImageResaleID),
		errors.Is(err, resaledom.ErrInvalidConditionImageURL),
		errors.Is(err, resaledom.ErrInvalidConditionImageDisplayOrder),
		errors.Is(err, resaledom.ErrInvalidConditionImageCreatedAt),
		errors.Is(err, resaledom.ErrInvalidConditionImageCreatedBy),
		errors.Is(err, resaledom.ErrInvalidConditionImageUpdatedAt),
		errors.Is(err, resaledom.ErrInvalidConditionImageUpdatedBy),
		msg == "invalid_image_id":
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
		return

	case strings.Contains(msg, "not supported"):
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
		return

	default:
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}
}
