// backend/internal/adapters/in/http/handlers/mintRequest_handler.go
package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	mintreqdom "narratives/internal/domain/mintRequest"
)

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
	_ = json.NewEncoder(w).Encode(mr)
}

// GET /mint-requests
// AuthMiddleware により context に注入された companyId を起点に、
// productBlueprint → production → mintRequest をたどって一覧を返す。
func (h *MintRequestHandler) listByCurrentCompany(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "mintRequest usecase not initialized"})
		return
	}

	list, err := h.uc.ListByCurrentCompany(ctx)
	if err != nil {
		writeMintRequestErr(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(list)
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
