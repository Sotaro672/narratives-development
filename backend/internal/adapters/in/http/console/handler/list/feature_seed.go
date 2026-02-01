// backend/internal/adapters/in/http/console/handler/list/feature_seed.go
//
// Responsibility:
// - /lists/create-seed を担当する。
// - listCreate.tsx など、作成画面の初期値（seed）を組み立てる。
package list

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (h *ListHandler) createSeed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "handler is nil"})
		return
	}

	if h.qMgmt == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
		return
	}

	qp := r.URL.Query()

	invID := strings.TrimSpace(qp.Get("inventoryId"))
	if invID == "" {
		invID = strings.TrimSpace(qp.Get("inventory_id"))
	}
	if invID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "inventoryId is required"})
		return
	}

	modelIDs := []string{}
	if vv := qp["modelIds"]; len(vv) > 0 {
		for _, x := range vv {
			x = strings.TrimSpace(x)
			if x == "" {
				continue
			}
			for _, s := range splitCSV(x) {
				s = strings.TrimSpace(s)
				if s != "" {
					modelIDs = append(modelIDs, s)
				}
			}
		}
	} else if vv := qp["model_ids"]; len(vv) > 0 {
		for _, x := range vv {
			x = strings.TrimSpace(x)
			if x == "" {
				continue
			}
			for _, s := range splitCSV(x) {
				s = strings.TrimSpace(s)
				if s != "" {
					modelIDs = append(modelIDs, s)
				}
			}
		}
	} else {
		raw := strings.TrimSpace(qp.Get("modelIds"))
		if raw == "" {
			raw = strings.TrimSpace(qp.Get("model_ids"))
		}
		if raw != "" {
			for _, s := range splitCSV(raw) {
				s = strings.TrimSpace(s)
				if s != "" {
					modelIDs = append(modelIDs, s)
				}
			}
		}
	}

	out, err := h.qMgmt.BuildCreateSeed(ctx, invID, modelIDs)
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}

		msg := strings.TrimSpace(err.Error())
		if strings.Contains(strings.ToLower(msg), "invalid") || strings.Contains(strings.ToLower(msg), "inventory") {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
			return
		}

		writeListErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(out)
}
