// backend/internal/adapters/in/http/mall/handler/resale_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	resaledom "narratives/internal/domain/resale"
)

// ResaleQuery is the read-side port used by mall resale handler.
// mall.ResaleQuery satisfies this interface.
//
// NOTE:
// /mall/me/resales は「自分の出品管理」専用。
// /mall/resales/avatar/{avatarId} は公開アバターの出品一覧表示専用。
// 公開マーケット一覧の List / ListByCursor は market_handler.go に移譲する。
type ResaleQuery interface {
	ListByAvatarID(
		ctx context.Context,
		avatarID string,
	) ([]resaledom.Resale, error)

	GetByID(
		ctx context.Context,
		id string,
	) (resaledom.Resale, error)

	ListImages(
		ctx context.Context,
		resaleID string,
	) ([]resaledom.ResaleImage, error)
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

const (
	meResalesPath     = "/mall/me/resales"
	publicResalesPath = "/mall/resales"
)

func (h *ResaleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")
	if path == "" {
		path = r.URL.Path
	}

	if path == publicResalesPath ||
		strings.HasPrefix(path, publicResalesPath+"/") {
		h.servePublic(w, r, path)
		return
	}

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
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "not_found",
		})
		return
	}

	rest := strings.TrimPrefix(path, meResalesPath+"/")
	parts := strings.Split(rest, "/")
	resaleID := parts[0]

	if resaleID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid resaleId",
		})
		return
	}

	if len(parts) > 1 {
		switch parts[1] {
		case "images", "condition-images":
			imageID := ""

			if len(parts) >= 3 {
				imageID = parts[2]
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
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "not_found",
			})
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
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "not_found",
			})
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

func (h *ResaleHandler) servePublic(
	w http.ResponseWriter,
	r *http.Request,
	path string,
) {
	if h == nil || h.query == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "not_implemented",
		})
		return
	}

	if path == publicResalesPath {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "not_found",
		})
		return
	}

	rest := strings.TrimPrefix(path, publicResalesPath+"/")
	parts := strings.Split(rest, "/")

	if len(parts) == 2 && parts[0] == "avatar" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		avatarID := parts[1]
		h.listPublicByAvatarID(w, r, avatarID)
		return
	}

	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		resaleID := parts[0]
		h.getPublic(w, r, resaleID)
		return
	}

	if len(parts) == 2 &&
		(parts[1] == "images" ||
			parts[1] == "condition-images") {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		resaleID := parts[0]
		h.listPublicImages(w, r, resaleID)
		return
	}

	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": "not_found",
	})
}

func (h *ResaleHandler) listPublicByAvatarID(
	w http.ResponseWriter,
	r *http.Request,
	avatarID string,
) {
	ctx := r.Context()

	if avatarID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "avatarId is required",
		})
		return
	}

	items, err := h.query.ListByAvatarID(ctx, avatarID)
	if err != nil {
		writeResaleErr(w, err)
		return
	}

	page := buildResalePageResponse(
		items,
		buildResalePageFromQuery(r),
	)

	_ = json.NewEncoder(w).Encode(page)
}

func (h *ResaleHandler) getPublic(
	w http.ResponseWriter,
	r *http.Request,
	resaleID string,
) {
	ctx := r.Context()

	if resaleID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "resaleId is required",
		})
		return
	}

	item, err := h.query.GetByID(ctx, resaleID)
	if err != nil {
		writeResaleErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": item,
	})
}

func (h *ResaleHandler) listPublicImages(
	w http.ResponseWriter,
	r *http.Request,
	resaleID string,
) {
	ctx := r.Context()

	if resaleID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "resaleId is required",
		})
		return
	}

	images, err := h.query.ListImages(ctx, resaleID)
	if err != nil {
		writeResaleErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": images,
	})
}

func (h *ResaleHandler) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "resale usecase is nil",
		})
		return
	}

	avatarID, ok := requireAvatarID(w, r)
	if !ok {
		return
	}

	body, err := io.ReadAll(
		io.LimitReader(r.Body, 1<<20),
	)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid body",
		})
		return
	}

	var item resaledom.Resale
	if err := json.Unmarshal(body, &item); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid json",
		})
		return
	}

	item.AvatarID = avatarID

	if item.MintAddress == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "mintAddress is required",
		})
		return
	}

	if item.TokenBlueprintID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "tokenBlueprintId is required",
		})
		return
	}

	if item.ProductID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "productId is required",
		})
		return
	}

	if item.Price <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "price must be greater than 0",
		})
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
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "not_implemented",
		})
		return
	}

	avatarID, ok := requireAvatarID(w, r)
	if !ok {
		return
	}

	items, err := h.query.ListByAvatarID(ctx, avatarID)
	if err != nil {
		writeResaleErr(w, err)
		return
	}

	page := buildResalePageResponse(
		items,
		buildResalePageFromQuery(r),
	)

	_ = json.NewEncoder(w).Encode(page)
}

func (h *ResaleHandler) get(
	w http.ResponseWriter,
	r *http.Request,
	resaleID string,
) {
	ctx := r.Context()

	item, ok := h.getOwnedResale(
		w,
		r,
		ctx,
		resaleID,
	)
	if !ok {
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": item,
	})
}

func (h *ResaleHandler) update(
	w http.ResponseWriter,
	r *http.Request,
	resaleID string,
) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "resale usecase is nil",
		})
		return
	}

	avatarID, ok := requireAvatarID(w, r)
	if !ok {
		return
	}

	existing, ok := h.getOwnedResale(
		w,
		r,
		ctx,
		resaleID,
	)
	if !ok {
		return
	}

	body, err := io.ReadAll(
		io.LimitReader(r.Body, 1<<20),
	)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid body",
		})
		return
	}

	var item resaledom.Resale
	if err := json.Unmarshal(body, &item); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid json",
		})
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

func (h *ResaleHandler) delete(
	w http.ResponseWriter,
	r *http.Request,
	resaleID string,
) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "resale usecase is nil",
		})
		return
	}

	if _, ok := h.getOwnedResale(
		w,
		r,
		ctx,
		resaleID,
	); !ok {
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

func (h *ResaleHandler) listImages(
	w http.ResponseWriter,
	r *http.Request,
	resaleID string,
) {
	ctx := r.Context()

	if h == nil || h.query == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "not_implemented",
		})
		return
	}

	if _, ok := h.getOwnedResale(
		w,
		r,
		ctx,
		resaleID,
	); !ok {
		return
	}

	images, err := h.query.ListImages(ctx, resaleID)
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
func (h *ResaleHandler) createImageFromFirebaseStorage(
	w http.ResponseWriter,
	r *http.Request,
	resaleID string,
) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "resale usecase is nil",
		})
		return
	}

	avatarID, ok := requireAvatarID(w, r)
	if !ok {
		return
	}

	if _, ok := h.getOwnedResale(
		w,
		r,
		ctx,
		resaleID,
	); !ok {
		return
	}

	var req struct {
		ID           string `json:"id"`
		URL          string `json:"url"`
		DisplayOrder int    `json:"displayOrder"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid json",
		})
		return
	}

	if req.ID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "id is required",
		})
		return
	}

	if strings.Contains(req.ID, "/") ||
		strings.Contains(req.ID, "://") {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid image id",
		})
		return
	}

	if req.URL == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "url is required",
		})
		return
	}

	if req.DisplayOrder < 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "displayOrder must be >= 0",
		})
		return
	}

	now := time.Now().UTC()

	img, err := h.uc.CreateImage(
		ctx,
		resaledom.ResaleImage{
			ID:           req.ID,
			ResaleID:     resaleID,
			URL:          req.URL,
			DisplayOrder: req.DisplayOrder,
			CreatedAt:    now,
			CreatedBy:    avatarID,
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

func (h *ResaleHandler) deleteImage(
	w http.ResponseWriter,
	r *http.Request,
	resaleID string,
	imageID string,
) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "resale usecase is nil",
		})
		return
	}

	if _, ok := h.getOwnedResale(
		w,
		r,
		ctx,
		resaleID,
	); !ok {
		return
	}

	if imageID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "imageId is required",
		})
		return
	}

	if err := h.uc.DeleteImage(
		ctx,
		resaleID,
		imageID,
	); err != nil {
		writeResaleErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":       true,
		"resaleId": resaleID,
		"imageId":  imageID,
	})
}

func (h *ResaleHandler) setPrimaryImage(
	w http.ResponseWriter,
	r *http.Request,
	resaleID string,
) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "resale usecase is nil",
		})
		return
	}

	avatarID, ok := requireAvatarID(w, r)
	if !ok {
		return
	}

	if _, ok := h.getOwnedResale(
		w,
		r,
		ctx,
		resaleID,
	); !ok {
		return
	}

	var req struct {
		ImageID string  `json:"imageId"`
		Now     *string `json:"now"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid json",
		})
		return
	}

	imageID := req.ImageID
	if imageID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "imageId is required",
		})
		return
	}

	now := time.Now().UTC()

	if req.Now != nil &&
		*req.Now != "" {
		if parsed, err := time.Parse(
			time.RFC3339,
			*req.Now,
		); err == nil {
			now = parsed.UTC()
		}
	}

	item, err := h.uc.SetPrimaryImage(
		ctx,
		resaleID,
		imageID,
		now,
		&avatarID,
	)
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
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "not_implemented",
		})
		return resaledom.Resale{}, false
	}

	avatarID, ok := requireAvatarID(w, r)
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
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "resale_access_denied",
		})
		return resaledom.Resale{}, false
	}

	return item, true
}

type resalePageResponse struct {
	Items []resaledom.Resale `json:"items"`

	TotalCount int `json:"totalCount"`
	TotalPages int `json:"totalPages"`

	Page    int `json:"page"`
	PerPage int `json:"perPage"`
}

func buildResalePageResponse(
	items []resaledom.Resale,
	page resaledom.Page,
) resalePageResponse {
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

	return resalePageResponse{
		Items: pagedItems,

		TotalCount: totalCount,
		TotalPages: totalPages,

		Page:    pageNum,
		PerPage: perPage,
	}
}

func buildResalePageFromQuery(r *http.Request) resaledom.Page {
	query := r.URL.Query()

	pageNum := parsePositiveIntDefault(query.Get("page"), 1)
	perPage := parsePositiveIntDefault(query.Get("perPage"), 50)

	if perPage > 100 {
		perPage = 100
	}

	return resaledom.Page{
		Number:  pageNum,
		PerPage: perPage,
	}
}

func writeResaleErr(w http.ResponseWriter, err error) {
	writeJSON(w, resaleHTTPStatus(err), map[string]string{
		"error": resaleErrorMessage(err),
	})
}

func resaleHTTPStatus(err error) int {
	if err == nil {
		return http.StatusInternalServerError
	}

	message := err.Error()

	switch {
	case errors.Is(err, context.Canceled),
		errors.Is(err, context.DeadlineExceeded):
		return http.StatusRequestTimeout

	case errors.Is(err, resaledom.ErrNotFound),
		errors.Is(err, resaledom.ErrConditionImageNotFound):
		return http.StatusNotFound

	case errors.Is(err, resaledom.ErrConflict),
		errors.Is(err, resaledom.ErrConditionImageConflict),
		errors.Is(err, resaledom.ErrSoldResaleCannotBeDeleted):
		return http.StatusConflict

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
		message == "invalid_image_id":
		return http.StatusBadRequest

	case strings.Contains(message, "not supported"):
		return http.StatusNotImplemented

	default:
		return http.StatusInternalServerError
	}
}

func resaleErrorMessage(err error) string {
	if err == nil {
		return "internal_error"
	}

	message := err.Error()

	switch {
	case errors.Is(err, context.Canceled),
		errors.Is(err, context.DeadlineExceeded):
		return "request_timeout"

	case strings.Contains(message, "not supported"):
		return "not_implemented"

	case resaleHTTPStatus(err) == http.StatusInternalServerError:
		return "internal_error"

	default:
		return message
	}
}
