// backend/internal/adapters/in/http/handlers/productBlueprint_handler.go
package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	brand "narratives/internal/domain/brand"
	pbdom "narratives/internal/domain/productBlueprint"
)

// ProductBlueprintHandler は ProductBlueprint 用の HTTP ハンドラです。
type ProductBlueprintHandler struct {
	uc       *usecase.ProductBlueprintUsecase
	brandSvc *brand.Service
}

// DI コンテナ側で ProductBlueprintUsecase と brand.Service を渡してください。
// brandSvc は現状未使用だが、将来 brand 名取得に使うために受け取っておく。
func NewProductBlueprintHandler(
	uc *usecase.ProductBlueprintUsecase,
	brandSvc *brand.Service,
) http.Handler {
	return &ProductBlueprintHandler{
		uc:       uc,
		brandSvc: brandSvc,
	}
}

func (h *ProductBlueprintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 末尾のスラッシュを削ってから判定
	path := strings.TrimRight(r.URL.Path, "/")

	switch {
	// ---------------------------------------------------
	// GET /product-blueprints  ← 一覧 API（未削除のみ）
	// ---------------------------------------------------
	case r.Method == http.MethodGet && path == "/product-blueprints":
		h.list(w, r)

	// ---------------------------------------------------
	// GET /product-blueprints/deleted ← 削除済み一覧 API
	// ---------------------------------------------------
	case r.Method == http.MethodGet && path == "/product-blueprints/deleted":
		h.listDeleted(w, r)

	// ---------------------------------------------------
	// POST /product-blueprints  ← 新規作成 API
	// ---------------------------------------------------
	case r.Method == http.MethodPost && path == "/product-blueprints":
		h.post(w, r)

	// ---------------------------------------------------
	// POST /product-blueprints/{id}/restore  ← 復元 API
	// ---------------------------------------------------
	case r.Method == http.MethodPost &&
		strings.HasPrefix(path, "/product-blueprints/") &&
		strings.HasSuffix(path, "/restore"):

		trimmed := strings.TrimPrefix(path, "/product-blueprints/")
		trimmed = strings.TrimSuffix(trimmed, "/restore")
		id := strings.Trim(trimmed, "/")
		h.restore(w, r, id)

	// ---------------------------------------------------
	// PUT/PATCH /product-blueprints/{id} ← 更新 API
	// ---------------------------------------------------
	case (r.Method == http.MethodPut || r.Method == http.MethodPatch) &&
		strings.HasPrefix(path, "/product-blueprints/"):

		id := strings.TrimPrefix(path, "/product-blueprints/")
		h.update(w, r, id)

	// ---------------------------------------------------
	// DELETE /product-blueprints/{id} ← 論理削除 API
	// ---------------------------------------------------
	case r.Method == http.MethodDelete &&
		strings.HasPrefix(path, "/product-blueprints/"):

		id := strings.TrimPrefix(path, "/product-blueprints/")
		h.delete(w, r, id)

	// ---------------------------------------------------
	// GET /product-blueprints/{id} ← 詳細取得 API
	// ---------------------------------------------------
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

	// createdBy を *string に変換（空文字なら nil）
	var createdBy *string
	if v := strings.TrimSpace(in.CreatedBy); v != "" {
		createdBy = &v
	}

	// Domain 変換
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
		// ProductIdTag を保存対象としてセット
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
// PUT/PATCH /product-blueprints/{id}  ← 更新 API
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

	// updatedBy を *string に変換（空文字なら nil）
	var updatedBy *string
	if v := strings.TrimSpace(in.UpdatedBy); v != "" {
		updatedBy = &v
	}

	// Domain 変換
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
// DELETE /product-blueprints/{id}  ← 論理削除 API
// ---------------------------------------------------

func (h *ProductBlueprintHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	// 削除実行（deletedBy は現状 context からの解決に任せるため nil）
	if err := h.uc.SoftDeleteWithModels(ctx, id, nil); err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	// 論理削除なので 204 No Content を返す
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------
// POST /product-blueprints/{id}/restore  ← 復元 API
// ---------------------------------------------------

func (h *ProductBlueprintHandler) restore(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	// 復元実行（restoredBy も現状は context からの解決に任せるため nil）
	if err := h.uc.RestoreWithModels(ctx, id, nil); err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	// 復元後の最新状態を返す
	pb, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(pb)
}

// ---------------------------------------------------
// GET /product-blueprints   ← 一覧 API（未削除）
// ---------------------------------------------------

// フロントエンドの期待に合わせた一覧用 DTO
// - brandId: ID をそのまま返す（name 変換はフロント側）
// - assigneeId: 担当者の MemberID をそのまま返す
// - productIdTag: "QR" / "NFC" など文字列ラベル
type ProductBlueprintListOutput struct {
	ID           string `json:"id"`
	ProductName  string `json:"productName"`
	BrandId      string `json:"brandId"`
	AssigneeId   string `json:"assigneeId"`
	ProductIdTag string `json:"productIdTag"`
	CreatedAt    string `json:"createdAt"` // YYYY/MM/DD
	UpdatedAt    string `json:"updatedAt"` // YYYY/MM/DD
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
		// brandId / assigneeId は ID をそのまま返す（空なら空のまま or "-"）
		brandId := strings.TrimSpace(pb.BrandID)
		assigneeId := strings.TrimSpace(pb.AssigneeID)
		if assigneeId == "" {
			assigneeId = "-"
		}

		// ProductIdTag.Type → 表示用ラベル（存在しない場合は "-"）
		productIdTag := "-"
		if pb.ProductIdTag.Type != "" {
			productIdTag = strings.ToUpper(string(pb.ProductIdTag.Type))
		}

		// 日付を "2006/01/02" 形式に整形
		createdAt := ""
		if !pb.CreatedAt.IsZero() {
			createdAt = pb.CreatedAt.Format("2006/01/02")
		}
		updatedAt := createdAt
		if !pb.UpdatedAt.IsZero() {
			updatedAt = pb.UpdatedAt.Format("2006/01/02")
		}

		out = append(out, ProductBlueprintListOutput{
			ID:           pb.ID,
			ProductName:  pb.ProductName,
			BrandId:      brandId,
			AssigneeId:   assigneeId,
			ProductIdTag: productIdTag,
			CreatedAt:    createdAt,
			UpdatedAt:    updatedAt,
		})
	}

	_ = json.NewEncoder(w).Encode(out)
}

// ---------------------------------------------------
// GET /product-blueprints/deleted  ← 削除済み一覧 API
// ---------------------------------------------------

// 削除済み一覧用 DTO
type ProductBlueprintDeletedListOutput struct {
	ID          string `json:"id"`
	ProductName string `json:"productName"`
	BrandId     string `json:"brandId"`
	AssigneeId  string `json:"assigneeId"`
	DeletedAt   string `json:"deletedAt"` // YYYY/MM/DD
	ExpireAt    string `json:"expireAt"`  // YYYY/MM/DD
}

func (h *ProductBlueprintHandler) listDeleted(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	log.Printf("[ProductBlueprintHandler] listDeleted: start")

	rows, err := h.uc.ListDeleted(ctx)
	if err != nil {
		log.Printf("[ProductBlueprintHandler] listDeleted: error: %v", err)
		writeProductBlueprintErr(w, err)
		return
	}

	log.Printf("[ProductBlueprintHandler] listDeleted: got %d rows", len(rows))

	out := make([]ProductBlueprintDeletedListOutput, 0, len(rows))
	for i, pb := range rows {
		brandId := strings.TrimSpace(pb.BrandID)
		assigneeId := strings.TrimSpace(pb.AssigneeID)
		if assigneeId == "" {
			assigneeId = "-"
		}

		deletedAtStr := ""
		if pb.DeletedAt != nil && !pb.DeletedAt.IsZero() {
			deletedAtStr = pb.DeletedAt.Format("2006/01/02")
		}

		expireAtStr := ""
		if pb.ExpireAt != nil && !pb.ExpireAt.IsZero() {
			expireAtStr = pb.ExpireAt.Format("2006/01/02")
		}

		log.Printf(
			"[ProductBlueprintHandler] listDeleted: row[%d]: id=%s name=%s deletedAt=%v companyId=%s",
			i, pb.ID, pb.ProductName, pb.DeletedAt, pb.CompanyID,
		)

		out = append(out, ProductBlueprintDeletedListOutput{
			ID:          pb.ID,
			ProductName: pb.ProductName,
			BrandId:     brandId,
			AssigneeId:  assigneeId,
			DeletedAt:   deletedAtStr,
			ExpireAt:    expireAtStr,
		})
	}

	// ★ 必ず JSON 配列を返す
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
		// それ以外は 500 のまま
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
