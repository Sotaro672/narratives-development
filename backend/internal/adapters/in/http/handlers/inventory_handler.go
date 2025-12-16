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
	ModelID            string   `json:"modelId"` // ★ NEW
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
		req.ModelID, // ★ NEW
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
	modelID := strings.TrimSpace(q.Get("modelId"))

	// 旧: tokenBlueprintId or productBlueprintId 必須
	// 新: modelId も条件に追加
	if tbID == "" && pbID == "" && modelID == "" {
		writeError(w, http.StatusBadRequest, "tokenBlueprintId or productBlueprintId or modelId is required")
		return
	}

	var (
		list []invdom.Mint
		err  error
	)

	// ✅ 新仕様の優先順位：
	// 1) tokenBlueprintId + modelId -> 1レコードに近い検索（docIdは modelId__tokenBlueprintId）
	// 2) tokenBlueprintId のみ
	// 3) modelId のみ
	// 4) productBlueprintId のみ（参照用途として残す）
	switch {
	case tbID != "" && modelID != "":
		list, err = h.UC.ListByTokenAndModelID(ctx, tbID, modelID)

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

		// ✅ products を置き換えたのに accumulation が来ない場合は、
		//    Usecase/Repo 側で len(products) に揃えられるよう 0 に寄せる
		if req.Accumulation == nil {
			current.Accumulation = 0
		}
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
		errors.Is(err, invdom.ErrInvalidModelID), // ★ NEW
		errors.Is(err, invdom.ErrInvalidProducts),
		errors.Is(err, invdom.ErrInvalidAccumulation):
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
