// backend/internal/adapters/in/http/console/handler/productBlueprint_handler.go
package consoleHandler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	brand "narratives/internal/domain/brand"
	memdom "narratives/internal/domain/member"
	pbdom "narratives/internal/domain/productBlueprint"
)

// ProductBlueprintHandler は ProductBlueprint 用の HTTP ハンドラです。
type ProductBlueprintHandler struct {
	uc        *usecase.ProductBlueprintUsecase
	brandSvc  *brand.Service
	memberSvc *memdom.Service
}

func NewProductBlueprintHandler(
	uc *usecase.ProductBlueprintUsecase,
	brandSvc *brand.Service,
	memberSvc *memdom.Service,
) http.Handler {
	return &ProductBlueprintHandler{
		uc:        uc,
		brandSvc:  brandSvc,
		memberSvc: memberSvc,
	}
}

// brandId → brandName 解決用ヘルパ
func (h *ProductBlueprintHandler) getBrandNameByID(ctx context.Context, brandID string) string {
	brandID = strings.TrimSpace(brandID)
	if brandID == "" {
		return ""
	}
	if h.brandSvc == nil {
		return brandID
	}

	name, err := h.brandSvc.GetNameByID(ctx, brandID)
	if err != nil {
		return brandID
	}
	return strings.TrimSpace(name)
}

// assigneeId → assigneeName 解決用ヘルパ
func (h *ProductBlueprintHandler) getAssigneeNameByID(ctx context.Context, memberID string) string {
	memberID = strings.TrimSpace(memberID)
	if memberID == "" {
		return ""
	}
	if h.memberSvc == nil {
		return memberID
	}

	name, err := h.memberSvc.GetNameLastFirstByID(ctx, memberID)
	if err != nil {
		return memberID
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return memberID
	}
	return name
}

func (h *ProductBlueprintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimRight(r.URL.Path, "/")

	switch {
	case r.Method == http.MethodGet && path == "/product-blueprints":
		h.list(w, r)

	case r.Method == http.MethodGet && path == "/product-blueprints/deleted":
		h.listDeleted(w, r)

	case r.Method == http.MethodGet && path == "/product-blueprints/printed":
		h.listPrinted(w, r)

	case r.Method == http.MethodGet &&
		strings.HasPrefix(path, "/product-blueprints/") &&
		strings.HasSuffix(path, "/history"):
		trimmed := strings.TrimPrefix(path, "/product-blueprints/")
		trimmed = strings.TrimSuffix(trimmed, "/history")
		id := strings.Trim(trimmed, "/")
		h.listHistory(w, r, id)

	case r.Method == http.MethodPost && path == "/product-blueprints":
		h.post(w, r)

	case r.Method == http.MethodPost &&
		strings.HasPrefix(path, "/product-blueprints/") &&
		strings.HasSuffix(path, "/restore"):
		trimmed := strings.TrimPrefix(path, "/product-blueprints/")
		trimmed = strings.TrimSuffix(trimmed, "/restore")
		id := strings.Trim(trimmed, "/")
		h.restore(w, r, id)

	case r.Method == http.MethodPost &&
		strings.HasPrefix(path, "/product-blueprints/") &&
		strings.HasSuffix(path, "/mark-printed"):
		trimmed := strings.TrimPrefix(path, "/product-blueprints/")
		trimmed = strings.TrimSuffix(trimmed, "/mark-printed")
		id := strings.Trim(trimmed, "/")
		h.markPrinted(w, r, id)

	case (r.Method == http.MethodPut || r.Method == http.MethodPatch) &&
		strings.HasPrefix(path, "/product-blueprints/"):
		id := strings.TrimPrefix(path, "/product-blueprints/")
		h.update(w, r, id)

	case r.Method == http.MethodDelete &&
		strings.HasPrefix(path, "/product-blueprints/"):
		id := strings.TrimPrefix(path, "/product-blueprints/")
		h.delete(w, r, id)

	case r.Method == http.MethodGet && strings.HasPrefix(path, "/product-blueprints/"):
		id := strings.TrimPrefix(path, "/product-blueprints/")
		h.get(w, r, id)

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// ---------------------------------------------------
// POST /product-blueprints
// ---------------------------------------------------

type ProductIdTagInput struct {
	Type string `json:"type"`
}

type CreateProductBlueprintInput struct {
	ProductName      string            `json:"productName"`
	BrandId          string            `json:"brandId"`
	ItemType         string            `json:"itemType"`
	Fit              string            `json:"fit"`
	Material         string            `json:"material"`
	Weight           float64           `json:"weight"`
	QualityAssurance []string          `json:"qualityAssurance"`
	ProductIdTag     ProductIdTagInput `json:"productIdTag"`
	AssigneeId       string            `json:"assigneeId"`
	CompanyId        string            `json:"companyId"`
	CreatedBy        string            `json:"createdBy,omitempty"`
}

func (h *ProductBlueprintHandler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var in CreateProductBlueprintInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	var createdBy *string
	if v := strings.TrimSpace(in.CreatedBy); v != "" {
		createdBy = &v
	}

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
		// printed は bool。create 時は常に false（未印刷）
		Printed: false,
		ProductIdTag: pbdom.ProductIDTag{
			Type: pbdom.ProductIDTagType(in.ProductIdTag.Type),
		},
	}

	created, err := h.uc.Create(ctx, pb)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

// ---------------------------------------------------
// PUT/PATCH /product-blueprints/{id}
// ---------------------------------------------------

type UpdateProductBlueprintInput struct {
	ProductName      string            `json:"productName"`
	BrandId          string            `json:"brandId"`
	ItemType         string            `json:"itemType"`
	Fit              string            `json:"fit"`
	Material         string            `json:"material"`
	Weight           float64           `json:"weight"`
	QualityAssurance []string          `json:"qualityAssurance"`
	ProductIdTag     ProductIdTagInput `json:"productIdTag"`
	AssigneeId       string            `json:"assigneeId"`
	CompanyId        string            `json:"companyId"`
	UpdatedBy        string            `json:"updatedBy,omitempty"`
}

func (h *ProductBlueprintHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	var in UpdateProductBlueprintInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	var updatedBy *string
	if v := strings.TrimSpace(in.UpdatedBy); v != "" {
		updatedBy = &v
	}

	pb := pbdom.ProductBlueprint{
		ID:               id,
		ProductName:      in.ProductName,
		BrandID:          in.BrandId,
		ItemType:         pbdom.ItemType(in.ItemType),
		Fit:              in.Fit,
		Material:         in.Material,
		Weight:           in.Weight,
		QualityAssurance: in.QualityAssurance,
		AssigneeID:       in.AssigneeId,
		CompanyID:        in.CompanyId,
		UpdatedBy:        updatedBy,
		ProductIdTag: pbdom.ProductIDTag{
			Type: pbdom.ProductIDTagType(in.ProductIdTag.Type),
		},
	}

	updated, err := h.uc.Update(ctx, pb)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(updated)
}

// ---------------------------------------------------
// DELETE /product-blueprints/{id}
// ---------------------------------------------------

func (h *ProductBlueprintHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	if err := h.uc.SoftDeleteWithModels(ctx, id, nil); err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------
// POST /product-blueprints/{id}/restore
// ---------------------------------------------------

func (h *ProductBlueprintHandler) restore(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	if err := h.uc.RestoreWithModels(ctx, id, nil); err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	pb, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(pb)
}

// ---------------------------------------------------
// GET /product-blueprints
// - 旧式互換は不要: createdAt/updatedAt は "2006-01-02T15:04:05Z07:00"（RFC3339）で返す
// ---------------------------------------------------

type ProductBlueprintListOutput struct {
	ID           string `json:"id"`
	ProductName  string `json:"productName"`
	BrandId      string `json:"brandId"`
	BrandName    string `json:"brandName"`
	AssigneeId   string `json:"assigneeId"`
	AssigneeName string `json:"assigneeName"`
	ProductIdTag string `json:"productIdTag"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

func (h *ProductBlueprintHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rows, err := h.uc.List(ctx)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	out := make([]ProductBlueprintListOutput, 0, len(rows))
	for _, pb := range rows {
		brandId := strings.TrimSpace(pb.BrandID)

		assigneeId := strings.TrimSpace(pb.AssigneeID)
		if assigneeId == "" {
			assigneeId = "-"
		}

		brandName := h.getBrandNameByID(ctx, brandId)
		if brandName == "" {
			brandName = brandId
		}

		assigneeName := "-"
		if assigneeId != "-" {
			assigneeName = h.getAssigneeNameByID(ctx, assigneeId)
		}

		productIdTag := "-"
		if pb.ProductIdTag.Type != "" {
			productIdTag = strings.ToUpper(string(pb.ProductIdTag.Type))
		}

		createdAt := ""
		if !pb.CreatedAt.IsZero() {
			createdAt = pb.CreatedAt.Format(time.RFC3339)
		}

		updatedAt := ""
		if !pb.UpdatedAt.IsZero() {
			updatedAt = pb.UpdatedAt.Format(time.RFC3339)
		}

		out = append(out, ProductBlueprintListOutput{
			ID:           pb.ID,
			ProductName:  pb.ProductName,
			BrandId:      brandId,
			BrandName:    brandName,
			AssigneeId:   assigneeId,
			AssigneeName: assigneeName,
			ProductIdTag: productIdTag,
			CreatedAt:    createdAt,
			UpdatedAt:    updatedAt,
		})
	}

	_ = json.NewEncoder(w).Encode(out)
}

// ---------------------------------------------------
// GET /product-blueprints/deleted
// ---------------------------------------------------

type ProductBlueprintDeletedListOutput struct {
	ID          string `json:"id"`
	ProductName string `json:"productName"`
	BrandId     string `json:"brandId"`
	AssigneeId  string `json:"assigneeId"`
	DeletedAt   string `json:"deletedAt"`
	ExpireAt    string `json:"expireAt"`
}

func (h *ProductBlueprintHandler) listDeleted(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rows, err := h.uc.ListDeleted(ctx)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	out := make([]ProductBlueprintDeletedListOutput, 0, len(rows))
	for _, pb := range rows {
		brandId := strings.TrimSpace(pb.BrandID)
		assigneeId := strings.TrimSpace(pb.AssigneeID)
		if assigneeId == "" {
			assigneeId = "-"
		}

		deletedAtStr := ""
		if pb.DeletedAt != nil && !pb.DeletedAt.IsZero() {
			deletedAtStr = pb.DeletedAt.Format(time.RFC3339)
		}

		expireAtStr := ""
		if pb.ExpireAt != nil && !pb.ExpireAt.IsZero() {
			expireAtStr = pb.ExpireAt.Format(time.RFC3339)
		}

		out = append(out, ProductBlueprintDeletedListOutput{
			ID:          pb.ID,
			ProductName: pb.ProductName,
			BrandId:     brandId,
			AssigneeId:  assigneeId,
			DeletedAt:   deletedAtStr,
			ExpireAt:    expireAtStr,
		})
	}

	_ = json.NewEncoder(w).Encode(out)
}

// ---------------------------------------------------
// GET /product-blueprints/printed
// - 旧式互換は不要: createdAt/updatedAt は RFC3339 で返す
// ---------------------------------------------------

func (h *ProductBlueprintHandler) listPrinted(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rows, err := h.uc.ListPrinted(ctx)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	out := make([]ProductBlueprintListOutput, 0, len(rows))
	for _, pb := range rows {
		brandId := strings.TrimSpace(pb.BrandID)

		assigneeId := strings.TrimSpace(pb.AssigneeID)
		if assigneeId == "" {
			assigneeId = "-"
		}

		brandName := h.getBrandNameByID(ctx, brandId)
		if brandName == "" {
			brandName = brandId
		}

		assigneeName := "-"
		if assigneeId != "-" {
			assigneeName = h.getAssigneeNameByID(ctx, assigneeId)
		}

		productIdTag := "-"
		if pb.ProductIdTag.Type != "" {
			productIdTag = strings.ToUpper(string(pb.ProductIdTag.Type))
		}

		createdAt := ""
		if !pb.CreatedAt.IsZero() {
			createdAt = pb.CreatedAt.Format(time.RFC3339)
		}

		updatedAt := ""
		if !pb.UpdatedAt.IsZero() {
			updatedAt = pb.UpdatedAt.Format(time.RFC3339)
		}

		out = append(out, ProductBlueprintListOutput{
			ID:           pb.ID,
			ProductName:  pb.ProductName,
			BrandId:      brandId,
			BrandName:    brandName,
			AssigneeId:   assigneeId,
			AssigneeName: assigneeName,
			ProductIdTag: productIdTag,
			CreatedAt:    createdAt,
			UpdatedAt:    updatedAt,
		})
	}

	_ = json.NewEncoder(w).Encode(out)
}

// ---------------------------------------------------
// POST /product-blueprints/{id}/mark-printed
// ---------------------------------------------------

func (h *ProductBlueprintHandler) markPrinted(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	pb, err := h.uc.MarkPrinted(ctx, id)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(pb)
}

// ---------------------------------------------------
// GET /product-blueprints/{id}/history
// - history は秒まで欲しいケースがあるため RFC3339 を維持（旧式互換不要）
// ---------------------------------------------------

type ProductBlueprintHistoryOutput struct {
	ID          string  `json:"id"`
	ProductName string  `json:"productName"`
	BrandId     string  `json:"brandId"`
	AssigneeId  string  `json:"assigneeId"`
	UpdatedAt   string  `json:"updatedAt"`
	UpdatedBy   *string `json:"updatedBy,omitempty"`
	DeletedAt   string  `json:"deletedAt,omitempty"`
	ExpireAt    string  `json:"expireAt,omitempty"`
}

func (h *ProductBlueprintHandler) listHistory(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	rows, err := h.uc.ListHistory(ctx, id)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	out := make([]ProductBlueprintHistoryOutput, 0, len(rows))
	for _, pb := range rows {
		brandId := strings.TrimSpace(pb.BrandID)
		assigneeId := strings.TrimSpace(pb.AssigneeID)
		if assigneeId == "" {
			assigneeId = "-"
		}

		updatedAtStr := ""
		if !pb.UpdatedAt.IsZero() {
			updatedAtStr = pb.UpdatedAt.Format(time.RFC3339)
		}

		deletedAtStr := ""
		if pb.DeletedAt != nil && !pb.DeletedAt.IsZero() {
			deletedAtStr = pb.DeletedAt.Format(time.RFC3339)
		}

		expireAtStr := ""
		if pb.ExpireAt != nil && !pb.ExpireAt.IsZero() {
			expireAtStr = pb.ExpireAt.Format(time.RFC3339)
		}

		out = append(out, ProductBlueprintHistoryOutput{
			ID:          pb.ID,
			ProductName: pb.ProductName,
			BrandId:     brandId,
			AssigneeId:  assigneeId,
			UpdatedAt:   updatedAtStr,
			UpdatedBy:   pb.UpdatedBy,
			DeletedAt:   deletedAtStr,
			ExpireAt:    expireAtStr,
		})
	}

	_ = json.NewEncoder(w).Encode(out)
}

// ---------------------------------------------------
// GET /product-blueprints/{id}
// ---------------------------------------------------

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

// ---------------------------------------------------
// 共通エラーハンドラ
// ---------------------------------------------------

func writeProductBlueprintErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case pbdom.IsInvalid(err):
		code = http.StatusBadRequest
	case pbdom.IsNotFound(err):
		code = http.StatusNotFound
	case pbdom.IsConflict(err):
		code = http.StatusConflict
	case pbdom.IsUnauthorized(err):
		code = http.StatusUnauthorized
	case pbdom.IsForbidden(err):
		code = http.StatusForbidden
	default:
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
