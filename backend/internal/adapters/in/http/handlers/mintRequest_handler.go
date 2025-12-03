// backend/internal/adapters/in/http/handlers/mintRequest_handler.go
package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	mintreqdom "narratives/internal/domain/mintRequest"
)

// レスポンス DTO（frontend の MintRequestDTO に合わせる）
type mintRequestDTO struct {
	ID                 string  `json:"id"`
	ProductionID       string  `json:"productionId"`
	ProductBlueprintID string  `json:"productBlueprintId,omitempty"`
	ProductName        *string `json:"productName,omitempty"`
	TokenBlueprintID   *string `json:"tokenBlueprintId,omitempty"`

	MintQuantity       int `json:"mintQuantity"`
	ProductionQuantity int `json:"productionQuantity"`

	Status string `json:"status"`

	RequestedBy       *string `json:"requestedBy,omitempty"`
	RequestedAt       *string `json:"requestedAt,omitempty"`
	MintedAt          *string `json:"mintedAt,omitempty"`
	ScheduledBurnDate *string `json:"scheduledBurnDate,omitempty"`
}

// MintRequestHandler は /mint-requests 関連のエンドポイントを担当します。
type MintRequestHandler struct {
	uc *usecase.MintRequestUsecase
}

// NewMintRequestHandler はHTTPハンドラを初期化します。
func NewMintRequestHandler(
	uc *usecase.MintRequestUsecase,
) http.Handler {
	return &MintRequestHandler{
		uc: uc,
	}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *MintRequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	// GET /mint-requests  → 現在の companyId に紐づく MintRequest 一覧
	case r.Method == http.MethodGet && r.URL.Path == "/mint-requests":
		h.listByCurrentCompany(w, r)

	// GET /mint-requests/{id} → 単一取得
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/mint-requests/"):
		id := strings.TrimPrefix(r.URL.Path, "/mint-requests/")
		h.get(w, r, id)

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /mint-requests/{id}
func (h *MintRequestHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	mr, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeMintRequestErr(w, err)
		return
	}

	// ドメイン → DTO 変換（単体取得版）
	dto := mintRequestDTO{
		ID:                 mr.ID,
		ProductionID:       mr.ProductionID,
		ProductBlueprintID: "",  // 単体取得では現状まだ解決していないため空文字
		ProductName:        nil, // 同上（今後 Production / ProductBlueprint 経由で解決予定）
		TokenBlueprintID:   mr.TokenBlueprintID,
		MintQuantity:       mr.MintQuantity,
		ProductionQuantity: 0, // 単体取得では未解決（必要なら後続で拡張）
		Status:             string(mr.Status),
		RequestedBy:        mr.RequestedBy,
		RequestedAt:        formatTimePtr(mr.RequestedAt),
		MintedAt:           formatTimePtr(mr.MintedAt),
		ScheduledBurnDate:  formatTimePtr(mr.ScheduledBurnDate),
	}

	_ = json.NewEncoder(w).Encode(dto)
}

// GET /mint-requests
// AuthMiddleware により context に注入された companyId を起点に、
// productBlueprint → production → mintRequest をたどって一覧を返す。
func (h *MintRequestHandler) listByCurrentCompany(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	results, err := h.uc.ListByCurrentCompany(ctx)
	if err != nil {
		writeMintRequestErr(w, err)
		return
	}

	dtoList := make([]mintRequestDTO, 0, len(results))
	for _, res := range results {
		var productNamePtr *string
		if strings.TrimSpace(res.ProductName) != "" {
			name := strings.TrimSpace(res.ProductName)
			productNamePtr = &name
		}

		dto := mintRequestDTO{
			ID:                 res.ID,
			ProductionID:       res.ProductionID,
			ProductBlueprintID: res.ProductBlueprintID,
			ProductName:        productNamePtr,
			TokenBlueprintID:   res.TokenBlueprintID,
			MintQuantity:       res.MintQuantity,
			ProductionQuantity: res.ProductionQuantity,
			Status:             res.Status,
			RequestedBy:        res.RequestedBy,
			RequestedAt:        formatTimePtr(res.RequestedAt),
			MintedAt:           formatTimePtr(res.MintedAt),
			ScheduledBurnDate:  formatTimePtr(res.BurnDate),
		}
		dtoList = append(dtoList, dto)
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(dtoList)
}

// time.Time ポインタ → RFC3339 文字列ポインタ
func formatTimePtr(t *time.Time) *string {
	if t == nil || t.IsZero() {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}

// エラーハンドリング
func writeMintRequestErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch err {
	case mintreqdom.ErrInvalidID:
		code = http.StatusBadRequest
	case mintreqdom.ErrNotFound:
		code = http.StatusNotFound
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
