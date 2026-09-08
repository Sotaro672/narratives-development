// backend/internal/adapters/in/http/mall/handler/resale_access_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"net/http"

	resaledom "narratives/internal/domain/resale"
)

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
