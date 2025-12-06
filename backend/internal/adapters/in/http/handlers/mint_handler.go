// backend/internal/adapters/in/http/handlers/mint_handler.go
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	pbpdom "narratives/internal/domain/productBlueprint"
)

// MintHandler は、MintUsecase を HTTP 経由で公開するハンドラです。
// GET /mint/inspections などを提供します。
type MintHandler struct {
	mintUC *usecase.MintUsecase
}

// NewMintHandler は MintHandler を生成します。
func NewMintHandler(mintUC *usecase.MintUsecase) http.Handler {
	return &MintHandler{
		mintUC: mintUC,
	}
}

func (h *MintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	// ------------------------------------------------------------
	// GET /mint/inspections
	// ------------------------------------------------------------
	case r.Method == http.MethodGet &&
		(r.URL.Path == "/mint/inspections" || strings.HasPrefix(r.URL.Path, "/mint/inspections/")):
		h.listInspectionsForCurrentCompany(w, r)
		return

	// ------------------------------------------------------------
	// GET /mint/product_blueprints/{id}/patch
	// ------------------------------------------------------------
	case r.Method == http.MethodGet &&
		strings.HasPrefix(r.URL.Path, "/mint/product_blueprints/") &&
		strings.HasSuffix(r.URL.Path, "/patch"):
		h.getProductBlueprintPatchByID(w, r)
		return

	// ------------------------------------------------------------
	// GET /mint/brands
	// ------------------------------------------------------------
	case r.Method == http.MethodGet && r.URL.Path == "/mint/brands":
		h.listBrandsForCurrentCompany(w, r)
		return

	default:
		http.NotFound(w, r)
	}
}

// ------------------------------------------------------------
// GET /mint/inspections
// ------------------------------------------------------------
func (h *MintHandler) listInspectionsForCurrentCompany(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.mintUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "mint usecase is not configured",
		})
		return
	}

	batches, err := h.mintUC.ListInspectionsForCurrentCompany(ctx)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, usecase.ErrCompanyIDMissing) {
			status = http.StatusBadRequest
		}

		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(batches)
}

// productBlueprint Patch + brandName を返すレスポンス用 DTO。
// Patch を埋め込みつつ brandName フィールドを追加する。
type productBlueprintPatchResponse struct {
	pbpdom.Patch
	BrandName string `json:"brandName"`
}

// ------------------------------------------------------------
// GET /mint/product_blueprints/{id}/patch
// ------------------------------------------------------------
func (h *MintHandler) getProductBlueprintPatchByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.mintUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "mint usecase is not configured",
		})
		return
	}

	// パスから {id} を抽出: /mint/product_blueprints/{id}/patch
	path := strings.TrimPrefix(r.URL.Path, "/mint/product_blueprints/")
	path = strings.TrimSuffix(path, "/patch")
	id := strings.Trim(path, "/")

	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "productBlueprintID is empty",
		})
		return
	}

	patch, err := h.mintUC.GetProductBlueprintPatchByID(ctx, id)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, pbpdom.ErrNotFound) {
			status = http.StatusNotFound
		}

		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Patch に含まれる brandId(*string) から brandName を解決する
	brandName := ""
	if patch.BrandID != nil {
		brandID := strings.TrimSpace(*patch.BrandID)
		if brandID != "" {
			name, err := h.mintUC.ResolveBrandNameByID(ctx, brandID)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": err.Error(),
				})
				return
			}
			brandName = name
		}
	}

	resp := productBlueprintPatchResponse{
		Patch:     patch,
		BrandName: brandName,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// ------------------------------------------------------------
// GET /mint/brands
// ------------------------------------------------------------
func (h *MintHandler) listBrandsForCurrentCompany(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.mintUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "mint usecase is not configured",
		})
		return
	}

	// ページング指定はとりあえずデフォルト値を利用
	var page branddom.Page

	result, err := h.mintUC.ListBrandsForCurrentCompany(ctx, page)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, usecase.ErrCompanyIDMissing) {
			status = http.StatusBadRequest
		}

		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	// PageResult[Brand] をそのまま返す
	_ = json.NewEncoder(w).Encode(result)
}
