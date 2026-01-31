// backend/internal/adapters/in/http/console/handler/inventory/crud.go
package inventory

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	invdom "narratives/internal/domain/inventory"
)

func (h *InventoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	if h.UC == nil {
		writeError(w, http.StatusNotImplemented, "inventory usecase is not configured")
		return
	}

	ctx := r.Context()

	var req createInventoryMintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	m, err := h.UC.UpsertFromMintByModel(
		ctx,
		req.TokenBlueprintID,
		req.ProductBlueprintID,
		req.ModelID,
		req.ProductIDs,
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, m)
}

func (h *InventoryHandler) List(w http.ResponseWriter, r *http.Request) {
	if h.UC == nil {
		writeError(w, http.StatusNotImplemented, "inventory usecase is not configured")
		return
	}

	ctx := r.Context()

	q := r.URL.Query()
	tbID := strings.TrimSpace(q.Get("tokenBlueprintId"))
	pbID := strings.TrimSpace(q.Get("productBlueprintId"))
	modelID := strings.TrimSpace(q.Get("modelId"))

	if tbID == "" && pbID == "" && modelID == "" {
		writeError(w, http.StatusBadRequest, "tokenBlueprintId or productBlueprintId or modelId is required")
		return
	}

	var (
		list []invdom.Mint
		err  error
	)

	switch {
	case tbID != "" && modelID != "":
		list, err = h.UC.ListByTokenAndModelID(ctx, tbID, modelID)

	case tbID != "" && pbID != "":
		// RepositoryPort に「TB+PB」直クエリが無いので PB で絞ってからフィルタ
		all, e := h.UC.ListByProductBlueprintID(ctx, pbID)
		if e != nil {
			err = e
			break
		}
		tmp := make([]invdom.Mint, 0, len(all))
		for _, m := range all {
			if strings.TrimSpace(m.TokenBlueprintID) == tbID {
				tmp = append(tmp, m)
			}
		}
		list = tmp

	case tbID != "":
		list, err = h.UC.ListByTokenBlueprintID(ctx, tbID)

	case modelID != "":
		list, err = h.UC.ListByModelID(ctx, modelID)

	default:
		list, err = h.UC.ListByProductBlueprintID(ctx, pbID)
	}

	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, list)
}

func (h *InventoryHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	if h.UC == nil {
		writeError(w, http.StatusNotImplemented, "inventory usecase is not configured")
		return
	}

	ctx := r.Context()

	id := strings.TrimSpace(pathParamLast(r.URL.Path))
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing id")
		return
	}

	m, err := h.UC.GetByID(ctx, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, m)
}

func (h *InventoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	if h.UC == nil {
		writeError(w, http.StatusNotImplemented, "inventory usecase is not configured")
		return
	}

	ctx := r.Context()

	id := strings.TrimSpace(pathParamLast(r.URL.Path))
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing id")
		return
	}

	var req updateInventoryMintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	req.ModelID = strings.TrimSpace(req.ModelID)
	if req.ModelID == "" {
		writeError(w, http.StatusBadRequest, "modelId is required")
		return
	}
	if len(req.ProductIDs) == 0 {
		writeError(w, http.StatusBadRequest, "productIds is required")
		return
	}

	current, err := h.UC.GetByID(ctx, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	updated, err := h.UC.UpsertFromMintByModel(
		ctx,
		current.TokenBlueprintID,
		current.ProductBlueprintID,
		req.ModelID,
		req.ProductIDs,
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *InventoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if h.UC == nil {
		writeError(w, http.StatusNotImplemented, "inventory usecase is not configured")
		return
	}

	ctx := r.Context()

	id := strings.TrimSpace(pathParamLast(r.URL.Path))
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing id")
		return
	}

	if err := h.UC.Delete(ctx, id); err != nil {
		writeDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// compile guard (when you want to ensure the usecase import stays in this file)
var _ = usecase.InventoryUsecase{}
