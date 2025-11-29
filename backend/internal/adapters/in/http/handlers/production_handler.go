// backend/internal/adapters/in/http/handlers/production_handler.go
package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	productiondom "narratives/internal/domain/production"
)

// ProductionHandler は /productions 関連のエンドポイントを担当します。
type ProductionHandler struct {
	uc *usecase.ProductionUsecase
}

func NewProductionHandler(uc *usecase.ProductionUsecase) http.Handler {
	return &ProductionHandler{uc: uc}
}

func (h *ProductionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	// GET /productions （一覧）
	case r.Method == http.MethodGet && r.URL.Path == "/productions":
		h.list(w, r)

	// GET /productions/{id}
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/productions/"):
		id := strings.TrimPrefix(r.URL.Path, "/productions/")
		h.get(w, r, id)

	// POST /productions
	case r.Method == http.MethodPost && r.URL.Path == "/productions":
		h.post(w, r)

	// PUT /productions/{id}
	case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/productions/"):
		id := strings.TrimPrefix(r.URL.Path, "/productions/")
		h.put(w, r, id)

	// DELETE /productions/{id}
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/productions/"):
		id := strings.TrimPrefix(r.URL.Path, "/productions/")
		h.delete(w, r, id)

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /productions （一覧）
func (h *ProductionHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// ★ Usecase 側に List を生やしている想定
	productions, err := h.uc.List(ctx)
	if err != nil {
		writeProductionErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(productions)
}

// GET /productions/{id}
func (h *ProductionHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	p, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeProductionErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(p)
}

// POST /productions
func (h *ProductionHandler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	defer r.Body.Close()

	var req productiondom.Production
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	// Usecase は値（productiondom.Production）を受け取る
	p, err := h.uc.Create(ctx, req)
	if err != nil {
		writeProductionErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(p)
}

// PUT /productions/{id}
func (h *ProductionHandler) put(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	defer r.Body.Close()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	var req productiondom.Production
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	// パスの id を優先してセット（Body 側の ID は信用しない）
	req.ID = id

	// Update 用のユースケースは Save を利用（upsert 的な役割）
	p, err := h.uc.Save(ctx, req)
	if err != nil {
		writeProductionErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(p)
}

// DELETE /productions/{id}
func (h *ProductionHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	if err := h.uc.Delete(ctx, id); err != nil {
		writeProductionErr(w, err)
		return
	}

	// 204 No Content
	w.WriteHeader(http.StatusNoContent)
}

func writeProductionErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case productiondom.ErrInvalidID:
		code = http.StatusBadRequest
	case productiondom.ErrNotFound:
		code = http.StatusNotFound
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
