// backend/internal/adapters/in/http/mall/handler/resale_public_handler.go
package mallHandler

import (
	"encoding/json"
	"net/http"
	"strings"
)

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
