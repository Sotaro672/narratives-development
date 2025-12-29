// backend\internal\adapters\in\http\sns\handler\shippingAddress_handler.go
package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	shadom "narratives/internal/domain/shippingAddress"
)

// ShippingAddressHandler は /shipping-addresses 関連のエンドポイントを担当します。
type ShippingAddressHandler struct {
	uc *usecase.ShippingAddressUsecase
}

// NewShippingAddressHandler はHTTPハンドラを初期化します。
func NewShippingAddressHandler(uc *usecase.ShippingAddressUsecase) http.Handler {
	return &ShippingAddressHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *ShippingAddressHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimSuffix(r.URL.Path, "/")

	// ✅ /sns プレフィックスを吸収（/sns/shipping-addresses -> /shipping-addresses）
	// - mux 側が /sns/** にハンドラを登録していても、このハンドラの内部ルーティングが一致するようにする
	if strings.HasPrefix(path, "/sns/") {
		path = strings.TrimPrefix(path, "/sns")
	}

	switch {
	// GET /shipping-addresses/{id}
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/shipping-addresses/"):
		id := strings.TrimPrefix(path, "/shipping-addresses/")
		h.get(w, r, id)
		return

	// POST /shipping-addresses
	case r.Method == http.MethodPost && path == "/shipping-addresses":
		h.post(w, r)
		return

	// PATCH /shipping-addresses/{id}
	case r.Method == http.MethodPatch && strings.HasPrefix(path, "/shipping-addresses/"):
		id := strings.TrimPrefix(path, "/shipping-addresses/")
		h.patch(w, r, id)
		return

	// DELETE /shipping-addresses/{id}
	case r.Method == http.MethodDelete && strings.HasPrefix(path, "/shipping-addresses/"):
		id := strings.TrimPrefix(path, "/shipping-addresses/")
		h.del(w, r, id)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

// GET /shipping-addresses/{id}
func (h *ShippingAddressHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	addr, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeShippingAddressErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(addr)
}

// POST /shipping-addresses
func (h *ShippingAddressHandler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// frontend/sns の入力欄に合わせる（zipCode/state/city/street/street2）
	type createReq struct {
		UserID  string  `json:"userId"`
		ZipCode string  `json:"zipCode"`
		State   string  `json:"state"`
		City    string  `json:"city"`
		Street  string  `json:"street"`
		Street2 *string `json:"street2,omitempty"`
		Country *string `json:"country,omitempty"` // UIに無い想定なので任意
	}

	var req createReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	street2 := ""
	if req.Street2 != nil {
		street2 = strings.TrimSpace(*req.Street2)
	}

	country := "JP"
	if req.Country != nil {
		if c := strings.TrimSpace(*req.Country); c != "" {
			country = c
		}
	}

	now := time.Now().UTC()

	// ID は空でOK（Firestore側で自動採番する方針）
	ent, err := shadom.NewWithNow(
		"", // id
		req.UserID,
		req.ZipCode,
		req.State,
		req.City,
		req.Street,
		street2,
		country,
		now,
	)
	if err != nil {
		writeShippingAddressErr(w, err)
		return
	}

	created, err := h.uc.Create(ctx, ent)
	if err != nil {
		writeShippingAddressErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

// PATCH /shipping-addresses/{id}
func (h *ShippingAddressHandler) patch(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	// 部分更新（null/未指定は現状維持）
	type patchReq struct {
		ZipCode *string `json:"zipCode,omitempty"`
		State   *string `json:"state,omitempty"`
		City    *string `json:"city,omitempty"`
		Street  *string `json:"street,omitempty"`
		Street2 *string `json:"street2,omitempty"`
		Country *string `json:"country,omitempty"`
		UserID  *string `json:"userId,omitempty"` // 基本は更新しない想定だが受けても無視する
	}

	var req patchReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	current, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeShippingAddressErr(w, err)
		return
	}

	// マージ
	zipCode := current.ZipCode
	state := current.State
	city := current.City
	street := current.Street
	street2 := current.Street2
	country := current.Country

	if req.ZipCode != nil {
		zipCode = *req.ZipCode
	}
	if req.State != nil {
		state = *req.State
	}
	if req.City != nil {
		city = *req.City
	}
	if req.Street != nil {
		street = *req.Street
	}
	if req.Street2 != nil {
		street2 = *req.Street2
	}
	if req.Country != nil {
		country = *req.Country
	}

	if strings.TrimSpace(country) == "" {
		country = "JP"
	}

	// ドメインの更新メソッドで検証＋更新
	if err := current.UpdateFromForm(
		zipCode,
		state,
		city,
		street,
		street2,
		country,
		time.Now().UTC(),
	); err != nil {
		writeShippingAddressErr(w, err)
		return
	}

	// Usecase に Update が無いので Save で反映する
	saved, err := h.uc.Save(ctx, current)
	if err != nil {
		writeShippingAddressErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(saved)
}

// DELETE /shipping-addresses/{id}
func (h *ShippingAddressHandler) del(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	if err := h.uc.Delete(ctx, id); err != nil {
		writeShippingAddressErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// エラーハンドリング
func writeShippingAddressErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch err {
	case shadom.ErrInvalidID,
		shadom.ErrInvalidUserID,
		shadom.ErrInvalidZipCode,
		shadom.ErrInvalidState,
		shadom.ErrInvalidCity,
		shadom.ErrInvalidStreet,
		shadom.ErrInvalidStreet2,
		shadom.ErrInvalidCountry,
		shadom.ErrInvalidCreatedAt,
		shadom.ErrInvalidUpdatedAt:
		code = http.StatusBadRequest
	case shadom.ErrNotFound:
		code = http.StatusNotFound
	case shadom.ErrConflict:
		code = http.StatusConflict
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
