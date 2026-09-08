// backend/internal/adapters/in/http/mall/handler/resale_crud_handler.go
package mallHandler

import (
	"encoding/json"
	"io"
	"net/http"

	usecase "narratives/internal/application/usecase"
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

type updateResaleRequest struct {
	Price       int                       `json:"price"`
	Status      resaledom.ResaleStatus    `json:"status"`
	Condition   resaledom.ResaleCondition `json:"condition"`
	Description string                    `json:"description"`
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

	created, err := h.uc.CreateResale(
		ctx,
		usecase.CreateResaleInput{
			AvatarID:         avatarID,
			AssetID:          req.AssetID,
			TokenBlueprintID: req.TokenBlueprintID,
			ProductID:        req.ProductID,
			Price:            req.Price,
			Condition:        req.Condition,
			Description:      req.Description,
		},
	)
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

	result, err := h.query.ListOwned(
		ctx,
		avatarID,
		buildResalePageFromQuery(r),
	)
	if err != nil {
		writeResaleErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(result)
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
	if h.query == nil {
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

	if _, err := h.uc.GetOwned(ctx, resaleID, avatarID); err != nil {
		writeResaleErr(w, err)
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

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid body",
		})
		return
	}

	var req updateResaleRequest
	if err := json.Unmarshal(body, &req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid json",
		})
		return
	}

	updated, err := h.uc.UpdateOwnedResale(
		ctx,
		usecase.UpdateOwnedResaleInput{
			ResaleID:    resaleID,
			AvatarID:    avatarID,
			Price:       req.Price,
			Status:      req.Status,
			Condition:   req.Condition,
			Description: req.Description,
		},
	)
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

	if err := h.uc.DeleteOwned(ctx, resaleID, avatarID); err != nil {
		writeResaleErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":       true,
		"resaleId": resaleID,
	})
}
