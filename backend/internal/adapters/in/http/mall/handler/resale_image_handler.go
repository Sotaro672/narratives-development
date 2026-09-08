// backend/internal/adapters/in/http/mall/handler/resale_image_handler.go
package mallHandler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	resaledom "narratives/internal/domain/resale"
)

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
