// backend/internal/adapters/in/http/handlers/productBlueprint_handler.go
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
	// ----------------------------
	// POST /product-blueprints
	// ----------------------------
	case r.Method == http.MethodPost && r.URL.Path == "/product-blueprints":
		h.post(w, r)

	// ----------------------------
	// GET /product-blueprints/{id}
	// ----------------------------
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
	CreatedBy        string   `json:"createdBy,omitempty"`
}

func (h *ProductBlueprintHandler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var in CreateProductBlueprintInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	// createdBy を *string に変換（空文字なら nil）
	var createdBy *string
	if v := strings.TrimSpace(in.CreatedBy); v != "" {
		createdBy = &v
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
		CreatedBy:        createdBy,
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

// ----------------------------
// 共通エラーハンドラ
// ----------------------------

func writeProductBlueprintErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	if err == pbdom.ErrInvalidID {
		code = http.StatusBadRequest
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
