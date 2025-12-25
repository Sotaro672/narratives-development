// backend\internal\adapters\in\http\sns\handler\productBlueprint_handler.go
package handler

import (
	"errors"
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
}

func NewSNSProductBlueprintHandler(uc productBlueprintGetter) http.Handler {
	return &SNSProductBlueprintHandler{uc: uc}
}

// ------------------------------
// Response DTOs (SNS)
// ------------------------------

// ✅ buyer 向け: assignee / created / updated / deleted / expire は返さない
type SnsProductBlueprintResponse struct {
	ID string `json:"id"`

	ProductName string         `json:"productName"`
	CompanyID   string         `json:"companyId"`
	BrandID     string         `json:"brandId"`
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

	writeJSON(w, http.StatusOK, toSnsProductBlueprintResponse(p))
}

// ------------------------------
// Mapping
// ------------------------------

func toSnsProductBlueprintResponse(p pbdom.ProductBlueprint) SnsProductBlueprintResponse {
	return SnsProductBlueprintResponse{
		ID:               strings.TrimSpace(p.ID),
		ProductName:      strings.TrimSpace(p.ProductName),
		CompanyID:        strings.TrimSpace(p.CompanyID),
		BrandID:          strings.TrimSpace(p.BrandID),
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
