package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	badom "narratives/internal/domain/billingAddress"
)

// BillingAddressHandler は /billing-addresses 関連のエンドポイントを担当します。
type BillingAddressHandler struct {
	uc *usecase.BillingAddressUsecase
}

// NewBillingAddressHandler はHTTPハンドラを初期化します。
func NewBillingAddressHandler(uc *usecase.BillingAddressUsecase) http.Handler {
	return &BillingAddressHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *BillingAddressHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimSuffix(r.URL.Path, "/")

	switch {
	// GET /billing-addresses/{id}
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/billing-addresses/"):
		id := strings.TrimPrefix(path, "/billing-addresses/")
		h.get(w, r, id)
		return

	// POST /billing-addresses
	case r.Method == http.MethodPost && path == "/billing-addresses":
		h.post(w, r)
		return

	// PATCH /billing-addresses/{id}
	case (r.Method == http.MethodPatch || r.Method == http.MethodPut) && strings.HasPrefix(path, "/billing-addresses/"):
		id := strings.TrimPrefix(path, "/billing-addresses/")
		h.update(w, r, id)
		return

	// DELETE /billing-addresses/{id}
	case r.Method == http.MethodDelete && strings.HasPrefix(path, "/billing-addresses/"):
		id := strings.TrimPrefix(path, "/billing-addresses/")
		h.delete(w, r, id)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

// GET /billing-addresses/{id}
func (h *BillingAddressHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	addr, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeBillingAddressErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(addr)
}

// POST /billing-addresses
func (h *BillingAddressHandler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var in badom.CreateBillingAddressInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	created, err := h.uc.Create(ctx, in)
	if err != nil {
		writeBillingAddressErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

// PATCH/PUT /billing-addresses/{id}
func (h *BillingAddressHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	var in badom.UpdateBillingAddressInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	updated, err := h.uc.Update(ctx, id, in)
	if err != nil {
		writeBillingAddressErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(updated)
}

// DELETE /billing-addresses/{id}
func (h *BillingAddressHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	if err := h.uc.Delete(ctx, id); err != nil {
		writeBillingAddressErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

// エラーハンドリング
func writeBillingAddressErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, badom.ErrInvalidID):
		code = http.StatusBadRequest
	case errors.Is(err, badom.ErrInvalidUserID):
		code = http.StatusBadRequest
	case errors.Is(err, badom.ErrInvalidCardNumber):
		code = http.StatusBadRequest
	case errors.Is(err, badom.ErrInvalidCardholderName):
		code = http.StatusBadRequest
	case errors.Is(err, badom.ErrInvalidCVC):
		code = http.StatusBadRequest
	case errors.Is(err, badom.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, badom.ErrConflict):
		code = http.StatusConflict
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
