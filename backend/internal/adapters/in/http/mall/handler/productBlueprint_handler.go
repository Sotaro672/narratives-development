// backend/internal/adapters/in/http/mall/handler/productBlueprint_handler.go
package mallHandler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
	pbdom "narratives/internal/domain/productBlueprint"
)

// MallProductBlueprintHandler serves buyer-facing productBlueprint endpoints (read-only).
//
// Routes (read-only):
// - GET /mall/product-blueprints/{id}
//
// NOTE:
//   - buyer に不要/秘匿したいフィールド（assigneeId, created/updated/deleted/expire 系）は返さない。
//   - 論理削除（DeletedAt != nil）の場合は 404 扱いにする。
//   - deletedAt / deletedBy は “拾わない” (= 返さない)。ただし DeletedAt による公開遮断はする。
//   - fit / material / weight / qualityAssurance などのカテゴリ依存項目は
//     ProductBlueprint 直下ではなく categoryFields から返す。
type MallProductBlueprintHandler struct {
	uc productBlueprintGetter

	// Name resolvers (type-safe)
	brandSvc   *branddom.Service
	companySvc *companydom.Service
}

// NewMallProductBlueprintHandler constructs handler without name resolution.
func NewMallProductBlueprintHandler(uc productBlueprintGetter) http.Handler {
	return &MallProductBlueprintHandler{uc: uc}
}

// NewMallProductBlueprintHandlerWithServices injects Brand/Company services for best-effort name resolution.
func NewMallProductBlueprintHandlerWithServices(
	uc productBlueprintGetter,
	brandSvc *branddom.Service,
	companySvc *companydom.Service,
) http.Handler {
	return &MallProductBlueprintHandler{
		uc:         uc,
		brandSvc:   brandSvc,
		companySvc: companySvc,
	}
}

// ------------------------------
// Response DTOs (Mall)
// ------------------------------

// buyer 向け: assignee / created / updated / deleted / expire は返さない
type MallProductBlueprintResponse struct {
	ID string `json:"id"`

	ProductName string `json:"productName"`
	Description string `json:"description"`

	CompanyID   string `json:"companyId"`
	CompanyName string `json:"companyName"`
	BrandID     string `json:"brandId"`
	BrandName   string `json:"brandName"`

	ProductBlueprintCategory MallProductBlueprintCategoryResponse `json:"productBlueprintCategory"`

	// CategoryFields はカテゴリ別の productBlueprint 入力値。
	//
	// 例:
	// - alcohol.sake:
	//   vintage, region, material, alcoholContent, volume
	// - apparel.tops:
	//   weight, fit, material
	// - cosmetics.skincare:
	//   material, volume
	CategoryFields map[string]any `json:"categoryFields,omitempty"`

	ProductIdTag MallProductIDTag `json:"productIdTag"`

	Printed bool `json:"printed"`
}

type MallProductBlueprintCategoryResponse struct {
	ID     string   `json:"id"`
	Code   string   `json:"code"`
	NameJa string   `json:"nameJa"`
	NameEn string   `json:"nameEn"`
	Kind   string   `json:"kind"`
	Path   []string `json:"path"`
}

type MallProductIDTag struct {
	Type string `json:"type"`
}

// ------------------------------
// http.Handler
// ------------------------------

func (h *MallProductBlueprintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h == nil || h.uc == nil {
		internalError(w, "usecase is nil")
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")

	// read-only
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	// mall only: GET /mall/product-blueprints/{id}
	if strings.HasPrefix(path, "/mall/product-blueprints/") {
		rest := strings.TrimPrefix(path, "/mall/product-blueprints/")
		parts := strings.Split(rest, "/")
		id := parts[0]
		if id == "" {
			badRequest(w, "invalid id")
			return
		}
		if len(parts) > 1 {
			notFound(w)
			return
		}
		h.getByID(w, r, id)
		return
	}

	notFound(w)
}

func (h *MallProductBlueprintHandler) getByID(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	log.Printf("[mall_productBlueprint] getById start id=%q brandSvc=%t companySvc=%t",
		id,
		h.brandSvc != nil,
		h.companySvc != nil,
	)

	p, err := h.uc.GetByID(ctx, id)
	if err != nil {
		if isPBNotFound(err) {
			notFound(w)
			return
		}
		writePBErr(w, err)
		return
	}

	resp := h.toMallProductBlueprintResponse(ctx, p)

	log.Printf("[mall_productBlueprint] ok id=%q productName=%q brandId=%q brandName=%q companyId=%q companyName=%q categoryId=%q categoryCode=%q",
		resp.ID,
		resp.ProductName,
		resp.BrandID,
		resp.BrandName,
		resp.CompanyID,
		resp.CompanyName,
		resp.ProductBlueprintCategory.ID,
		resp.ProductBlueprintCategory.Code,
	)

	writeJSON(w, http.StatusOK, resp)
}

// ------------------------------
// Mapping
// ------------------------------

func (h *MallProductBlueprintHandler) toMallProductBlueprintResponse(ctx context.Context, p pbdom.ProductBlueprint) MallProductBlueprintResponse {
	pbID := p.ID
	productName := p.ProductName
	description := p.Description
	companyID := p.CompanyID
	brandID := p.BrandID
	category := p.ProductBlueprintCategory

	// name resolve (best-effort)
	brandName := ""
	companyName := ""

	if brandID != "" && h != nil && h.brandSvc != nil {
		if s, err := h.brandSvc.GetNameByID(ctx, brandID); err == nil {
			brandName = s
		}
	}
	if companyID != "" && h != nil && h.companySvc != nil {
		// best-effort: not-found は握りつぶして空にする
		if s, ok, err := h.companySvc.TryGetCompanyName(ctx, companyID); err == nil && ok {
			companyName = s
		}
	}

	return MallProductBlueprintResponse{
		ID:          pbID,
		ProductName: productName,
		Description: description,

		CompanyID:   companyID,
		CompanyName: companyName,
		BrandID:     brandID,
		BrandName:   brandName,

		ProductBlueprintCategory: MallProductBlueprintCategoryResponse{
			ID:     category.ID,
			Code:   category.Code,
			NameJa: category.NameJa,
			NameEn: category.NameEn,
			Kind:   string(category.Kind),
			Path:   append([]string(nil), category.Path...),
		},

		CategoryFields: cloneMallCategoryFields(p.CategoryFields),

		ProductIdTag: MallProductIDTag{
			Type: string(p.ProductIdTag.Type),
		},
		Printed: p.Printed,
	}
}

func cloneMallCategoryFields(in pbdom.CategoryFields) map[string]any {
	if len(in) == 0 {
		return nil
	}

	out := make(map[string]any, len(in))
	for key, value := range in {
		if key == "" {
			continue
		}
		out[key] = value
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

// ------------------------------
// Error mapping
// ------------------------------

func isPBNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, pbdom.ErrNotFound) || pbdom.IsNotFound(err) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found")
}

func writePBErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case isPBNotFound(err):
		code = http.StatusNotFound
	case errors.Is(err, pbdom.ErrForbidden) || pbdom.IsForbidden(err):
		code = http.StatusForbidden
	case errors.Is(err, pbdom.ErrUnauthorized) || pbdom.IsUnauthorized(err):
		code = http.StatusUnauthorized
	default:
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "invalid") ||
			strings.Contains(msg, "required") ||
			strings.Contains(msg, "must") {
			code = http.StatusBadRequest
		}
	}

	writeJSON(w, code, map[string]string{"error": err.Error()})
}
