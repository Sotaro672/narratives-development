// backend/internal/adapters/in/http/handlers/inventory_handler.go
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	invdom "narratives/internal/domain/inventory"
)

type InventoryHandler struct {
	UC *usecase.InventoryUsecase
}

func NewInventoryHandler(uc *usecase.InventoryUsecase) *InventoryHandler {
	return &InventoryHandler{UC: uc}
}

func (h *InventoryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")

	if path == "/inventories" {
		switch r.Method {
		case http.MethodPost:
			h.Create(w, r)
			return
		case http.MethodGet:
			h.List(w, r)
			return
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	}

	if strings.HasPrefix(path, "/inventories/") {
		switch r.Method {
		case http.MethodGet:
			h.GetByID(w, r)
			return
		case http.MethodPatch:
			h.Update(w, r)
			return
		case http.MethodDelete:
			h.Delete(w, r)
			return
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
}

// DTOs

type createInventoryMintRequest struct {
	TokenBlueprintID   string   `json:"tokenBlueprintId"`
	ProductBlueprintID string   `json:"productBlueprintId"`
	ProductIDs         []string `json:"productIds"`
	Accumulation       int      `json:"accumulation"`
}

type updateInventoryMintRequest struct {
	Products     []string `json:"products"` // ★ []string only
	Accumulation *int     `json:"accumulation,omitempty"`
}

// Handlers

func (h *InventoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req createInventoryMintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	m, err := h.UC.CreateMint(
		ctx,
		req.TokenBlueprintID,
		req.ProductBlueprintID,
		req.ProductIDs,
		req.Accumulation,
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, m)
}

func (h *InventoryHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	q := r.URL.Query()
	tbID := strings.TrimSpace(q.Get("tokenBlueprintId"))
	pbID := strings.TrimSpace(q.Get("productBlueprintId"))

	if tbID == "" && pbID == "" {
		writeError(w, http.StatusBadRequest, "tokenBlueprintId or productBlueprintId is required")
		return
	}

	var (
		list []invdom.Mint
		err  error
	)

	switch {
	case tbID != "" && pbID != "":
		list, err = h.UC.ListByTokenAndProductBlueprintID(ctx, tbID, pbID)
	case tbID != "":
		list, err = h.UC.ListByTokenBlueprintID(ctx, tbID)
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
	ctx := r.Context()

	id := strings.TrimSpace(pathParamLast(r.URL.Path))
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing id")
		return
	}

	m, err := h.UC.GetMintByID(ctx, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, m)
}

func (h *InventoryHandler) Update(w http.ResponseWriter, r *http.Request) {
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

	current, err := h.UC.GetMintByID(ctx, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	// Products は「指定されたら置き換え」
	if req.Products != nil {
		current.Products = req.Products
	}
	if req.Accumulation != nil {
		current.Accumulation = *req.Accumulation
	}

	updated, err := h.UC.UpdateMint(ctx, current)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *InventoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := strings.TrimSpace(pathParamLast(r.URL.Path))
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing id")
		return
	}

	if err := h.UC.DeleteMint(ctx, id); err != nil {
		writeDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helpers

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{
		"error": msg,
	})
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, invdom.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())

	case errors.Is(err, invdom.ErrInvalidMintID),
		errors.Is(err, invdom.ErrInvalidTokenBlueprintID),
		errors.Is(err, invdom.ErrInvalidProductBlueprintID),
		errors.Is(err, invdom.ErrInvalidProducts),
		errors.Is(err, invdom.ErrInvalidAccumulation),
		errors.Is(err, invdom.ErrInvalidCreatedAt),
		errors.Is(err, invdom.ErrInvalidUpdatedAt):
		writeError(w, http.StatusBadRequest, err.Error())

	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func pathParamLast(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	path = strings.TrimSuffix(path, "/")
	i := strings.LastIndex(path, "/")
	if i < 0 || i == len(path)-1 {
		return ""
	}
	return path[i+1:]
}
