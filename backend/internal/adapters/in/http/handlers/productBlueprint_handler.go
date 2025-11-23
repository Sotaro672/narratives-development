package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	pbdom "narratives/internal/domain/productBlueprint"
)

type ProductBlueprintHandler struct {
	uc *usecase.ProductBlueprintUsecase
}

func NewProductBlueprintHandler(uc *usecase.ProductBlueprintUsecase) http.Handler {
	return &ProductBlueprintHandler{uc: uc}
}

func (h *ProductBlueprintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/product-blueprints":
		h.post(w, r)

	case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/product-blueprints/") &&
		strings.HasSuffix(r.URL.Path, "/variations"):
		h.attachVariations(w, r)

	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/product-blueprints/"):
		id := strings.TrimPrefix(r.URL.Path, "/product-blueprints/")
		h.get(w, r, id)

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// ----------------------------
// POST /product-blueprints
// ----------------------------
type CreateProductBlueprintInput struct {
	ProductName      string   `json:"productName"`
	BrandId          string   `json:"brandId"`
	ItemType         string   `json:"itemType"`
	Fit              string   `json:"fit"`
	Material         string   `json:"material"`
	Weight           float64  `json:"weight"`
	QualityAssurance []string `json:"qualityAssurance"`
	ProductIdTagType string   `json:"productIdTagType"`
	Colors           []string `json:"colors"`
	AssigneeId       string   `json:"assigneeId"`
	CompanyId        string   `json:"companyId"`
}

func (h *ProductBlueprintHandler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var in CreateProductBlueprintInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	// ---------- Domain 変換 ----------
	pb := pbdom.ProductBlueprint{
		ProductName:      in.ProductName,
		BrandID:          in.BrandId,
		ItemType:         pbdom.ItemType(in.ItemType),
		Fit:              in.Fit,
		Material:         in.Material,
		Weight:           in.Weight,
		QualityAssurance: in.QualityAssurance,
		AssigneeID:       in.AssigneeId,
		CompanyID:        in.CompanyId,

		// variations は POST では扱わない（後から PATCH で紐付け）
		VariationIDs: []string{},
	}

	created, err := h.uc.Create(ctx, pb)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

// ----------------------------
// PATCH /product-blueprints/{id}/variations
// ----------------------------
type AttachVariationsInput struct {
	VariationIDs []string `json:"variationIds"`
}

func (h *ProductBlueprintHandler) attachVariations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := strings.TrimPrefix(r.URL.Path, "/product-blueprints/")
	id = strings.TrimSuffix(id, "/variations")
	id = strings.TrimSpace(id)

	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	var in AttachVariationsInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	err := h.uc.AttachVariations(ctx, id, in.VariationIDs)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ----------------------------
// GET /product-blueprints/{id}
// ----------------------------
func (h *ProductBlueprintHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	pb, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(pb)
}

func writeProductBlueprintErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	if err == pbdom.ErrInvalidID {
		code = http.StatusBadRequest
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
