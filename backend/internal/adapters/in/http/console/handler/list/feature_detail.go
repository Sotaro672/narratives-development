// backend/internal/adapters/in/http/console/handler/list/feature_detail.go
//
// Responsibility:
// - /lists/{id} の詳細取得と、/lists/{id}/aggregate を担当する。
// - detail は Query（read-model）を優先して返す。
package list

import (
	"encoding/json"
	"net/http"

	listdom "narratives/internal/domain/list"
)

func (h *ListHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if h == nil || h.qDetail == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
		return
	}

	dto, err := h.qDetail.BuildListDetailDTO(ctx, id)
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}
		writeListErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(dto)
}

func (h *ListHandler) getAggregate(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	agg, err := h.uc.GetAggregate(ctx, id)
	if err != nil {
		writeListErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(agg)
}

// compile guard: keep listdom imported if needed elsewhere; remove if unused after your edits
var _ = listdom.StatusListing
