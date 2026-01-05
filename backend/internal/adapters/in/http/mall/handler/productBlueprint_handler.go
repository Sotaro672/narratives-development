// backend/internal/adapters/in/http/sns/handler/productBlueprint_handler.go
package handler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	pbdom "narratives/internal/domain/productBlueprint"
)

// SNSProductBlueprintHandler serves buyer-facing productBlueprint endpoints (read-only).
//
// Routes (read-only):
// - GET /sns/product-blueprints/{id}
//
// NOTE:
// - buyer に不要/秘匿したいフィールド（assigneeId, created/updated/deleted/expire 系）は返さない。
// - 論理削除（DeletedAt != nil）の場合は 404 扱いにする。
// - deletedAt / deletedBy は “拾わない” (= 返さない)。ただし DeletedAt による公開遮断はする。
type SNSProductBlueprintHandler struct {
	uc productBlueprintGetter

	// ✅ NEW: name resolver injection (best-effort)
	BrandNameResolver   any
	CompanyNameResolver any
}

func NewSNSProductBlueprintHandler(uc productBlueprintGetter) http.Handler {
	return &SNSProductBlueprintHandler{uc: uc}
}

// ✅ optional ctor: NameResolver を明示注入したい場合
func NewSNSProductBlueprintHandlerWithNameResolver(uc productBlueprintGetter, nameResolver any) http.Handler {
	return &SNSProductBlueprintHandler{
		uc:                  uc,
		BrandNameResolver:   nameResolver,
		CompanyNameResolver: nameResolver,
	}
}

// ------------------------------
// Response DTOs (SNS)
// ------------------------------

// ✅ buyer 向け: assignee / created / updated / deleted / expire は返さない
type SnsProductBlueprintResponse struct {
	ID string `json:"id"`

	ProductName string         `json:"productName"`
	CompanyID   string         `json:"companyId"`
	CompanyName string         `json:"companyName"` // ✅ NEW
	BrandID     string         `json:"brandId"`
	BrandName   string         `json:"brandName"` // ✅ NEW
	ItemType    pbdom.ItemType `json:"itemType"`
	Fit         string         `json:"fit"`
	Material    string         `json:"material"`
	Weight      float64        `json:"weight"`

	QualityAssurance []string        `json:"qualityAssurance"`
	ProductIdTag     SnsProductIDTag `json:"productIdTag"`

	Printed bool `json:"printed"`
}

// ------------------------------
// http.Handler
// ------------------------------

func (h *SNSProductBlueprintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h == nil || h.uc == nil {
		internalError(w, "usecase is nil")
		return
	}

	path := strings.TrimSuffix(strings.TrimSpace(r.URL.Path), "/")

	// read-only
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	// GET /sns/product-blueprints/{id}
	if strings.HasPrefix(path, "/sns/product-blueprints/") {
		rest := strings.TrimPrefix(path, "/sns/product-blueprints/")
		parts := strings.Split(rest, "/")
		id := strings.TrimSpace(parts[0])
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

func (h *SNSProductBlueprintHandler) getByID(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	log.Printf("[sns_productBlueprint] getById start id=%q resolverBrand=%t resolverCompany=%t",
		strings.TrimSpace(id),
		h.BrandNameResolver != nil,
		h.CompanyNameResolver != nil,
	)

	p, err := h.uc.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		if isPBNotFound(err) {
			notFound(w)
			return
		}
		writePBErr(w, err)
		return
	}

	// SNS public safety: 論理削除は見せない
	if p.DeletedAt != nil {
		notFound(w)
		return
	}

	resp := toSnsProductBlueprintResponse(ctx, p, h.BrandNameResolver, h.CompanyNameResolver)

	log.Printf("[sns_productBlueprint] ok id=%q productName=%q brandId=%q brandName=%q companyId=%q companyName=%q",
		resp.ID,
		resp.ProductName,
		resp.BrandID,
		resp.BrandName,
		resp.CompanyID,
		resp.CompanyName,
	)

	writeJSON(w, http.StatusOK, resp)
}

// ------------------------------
// Mapping
// ------------------------------

func toSnsProductBlueprintResponse(
	ctx context.Context,
	p pbdom.ProductBlueprint,
	brandNameResolver any,
	companyNameResolver any,
) SnsProductBlueprintResponse {
	pbID := strings.TrimSpace(p.ID)
	productName := strings.TrimSpace(p.ProductName)
	companyID := strings.TrimSpace(p.CompanyID)
	brandID := strings.TrimSpace(p.BrandID)

	// ✅ name resolve (best-effort)
	brandName := ""
	companyName := ""

	if brandID != "" && brandNameResolver != nil {
		if s, ok := resolveBrandNameBestEffort(ctx, brandNameResolver, brandID); ok {
			brandName = strings.TrimSpace(s)
		}
	}
	if companyID != "" && companyNameResolver != nil {
		if s, ok := resolveCompanyNameBestEffort(ctx, companyNameResolver, companyID); ok {
			companyName = strings.TrimSpace(s)
		}
	}

	return SnsProductBlueprintResponse{
		ID:               pbID,
		ProductName:      productName,
		CompanyID:        companyID,
		CompanyName:      companyName, // ✅ NEW
		BrandID:          brandID,
		BrandName:        brandName, // ✅ NEW
		ItemType:         p.ItemType,
		Fit:              strings.TrimSpace(p.Fit),
		Material:         strings.TrimSpace(p.Material),
		Weight:           p.Weight,
		QualityAssurance: append([]string{}, p.QualityAssurance...),
		ProductIdTag: SnsProductIDTag{
			Type: strings.TrimSpace(p.ProductIdTag.Type),
		},
		Printed: p.Printed,
	}
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
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
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
		msg := strings.ToLower(strings.TrimSpace(err.Error()))
		if strings.Contains(msg, "invalid") ||
			strings.Contains(msg, "required") ||
			strings.Contains(msg, "must") {
			code = http.StatusBadRequest
		}
	}

	writeJSON(w, code, map[string]string{"error": err.Error()})
}
