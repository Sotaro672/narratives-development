// backend/internal/adapters/in/http/mall/handler/resale_crud_handler.go
package mallHandler

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	resaledom "narratives/internal/domain/resale"
)

type createResaleRequest struct {
	AssetID          string                    `json:"assetId"`
	TokenBlueprintID string                    `json:"tokenBlueprintId"`
	ProductID        string                    `json:"productId"`
	Price            int                       `json:"price"`
	Condition        resaledom.ResaleCondition `json:"condition"`
	Description      string                    `json:"description"`
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

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid body",
		})
		return
	}

	var req createResaleRequest
	if err := json.Unmarshal(body, &req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid json",
		})
		return
	}

	if req.AssetID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "assetId is required",
		})
		return
	}

	if req.TokenBlueprintID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "tokenBlueprintId is required",
		})
		return
	}

	if req.ProductID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "productId is required",
		})
		return
	}

	if req.Price <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "price must be greater than 0",
		})
		return
	}

	condition := req.Condition
	if condition == "" {
		condition = resaledom.ConditionLikeNew
	}

	now := time.Now().UTC()
	item := resaledom.Resale{
		Status:           resaledom.StatusListing,
		AssetID:          req.AssetID,
		TokenBlueprintID: req.TokenBlueprintID,
		ProductID:        req.ProductID,
		AvatarID:         avatarID,
		Price:            req.Price,
		Condition:        condition,
		Description:      req.Description,
		CreatedBy:        avatarID,
		CreatedAt:        now,
		UpdatedAt:        &now,
	}

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

	item, err := h.uc.GetOwned(ctx, resaleID, avatarID)
	if err != nil {
		writeResaleErr(w, err)
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

	existing, err := h.uc.GetOwned(ctx, resaleID, avatarID)
	if err != nil {
		writeResaleErr(w, err)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
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
	item.AssetID = existing.AssetID
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

	avatarID, ok := requireAvatarID(w, r)
	if !ok {
		return
	}

	if _, err := h.uc.GetOwned(ctx, resaleID, avatarID); err != nil {
		writeResaleErr(w, err)
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
