// backend\internal\adapters\in\http\mall\handler\shippingAddress_handler.go
package mallHandler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	shadom "narratives/internal/domain/shippingAddress"
)

// ShippingAddressHandler は /mall/shipping-addresses 関連のエンドポイントを担当します。
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

	// ✅ 末尾スラッシュを吸収（/mall/... 前提で扱う）
	path := strings.TrimSuffix(r.URL.Path, "/")

	switch {
	// GET /mall/shipping-addresses/{id}
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/mall/shipping-addresses/"):
		id := strings.TrimPrefix(path, "/mall/shipping-addresses/")
		h.get(w, r, id)
		return

	// POST /mall/shipping-addresses
	// ✅ docId=UID 統一方針では本来 PUT /mall/shipping-addresses/{id} が望ましいが、
	//    互換のため POST も残す（ただし id 必須）
	case r.Method == http.MethodPost && path == "/mall/shipping-addresses":
		h.post(w, r)
		return

	// PATCH /mall/shipping-addresses/{id}
	case r.Method == http.MethodPatch && strings.HasPrefix(path, "/mall/shipping-addresses/"):
		id := strings.TrimPrefix(path, "/mall/shipping-addresses/")
		h.patch(w, r, id)
		return

	// DELETE /mall/shipping-addresses/{id}
	case r.Method == http.MethodDelete && strings.HasPrefix(path, "/mall/shipping-addresses/"):
		id := strings.TrimPrefix(path, "/mall/shipping-addresses/")
		h.del(w, r, id)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

// GET /mall/shipping-addresses/{id}
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

// POST /mall/shipping-addresses
func (h *ShippingAddressHandler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	raw, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid body"})
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(raw))

	// frontend/mall の入力欄に合わせる（zipCode/state/city/street/street2）
	// ✅ docId=UID 統一: id(=uid) を必須にする
	type createReq struct {
		ID      string  `json:"id"` // ✅ docId = UID
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

	id := strings.TrimSpace(req.ID)
	if id == "" {
		// ✅ docId=UID 統一方針: id が無い POST は許可しない（Firestore自動採番をさせない）
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
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

	// ✅ ID を docId(=UID) で固定
	ent, err := shadom.NewWithNow(
		id, // ✅ id=UID
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

	// ✅ docId=UID: 住所は “upsert” が自然なので Save に寄せる
	saved, err := h.uc.Save(ctx, ent)
	if err != nil {
		writeShippingAddressErr(w, err)
		return
	}

	// 互換のため 201 で返す（厳密な Created/Updated 判定は Usecase 側が必要）
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(saved)
}

// PATCH /mall/shipping-addresses/{id}
func (h *ShippingAddressHandler) patch(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	raw, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid body"})
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(raw))

	// 部分更新（null/未指定は現状維持）
	type patchReq struct {
		ZipCode *string `json:"zipCode,omitempty"`
		State   *string `json:"state,omitempty"`
		City    *string `json:"city,omitempty"`
		Street  *string `json:"street,omitempty"`
		Street2 *string `json:"street2,omitempty"`
		Country *string `json:"country,omitempty"`
		UserID  *string `json:"userId,omitempty"` // 受けても ID 連携上は信頼しない
	}

	var req patchReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	// ✅ docId=UID 統一: PATCH は「存在すれば更新・無ければ作成」でもよい
	current, err := h.uc.GetByID(ctx, id)
	if err != nil {
		if err == shadom.ErrNotFound {
			// upsert-create path
			need := func(p *string) (string, bool) {
				if p == nil {
					return "", false
				}
				s := strings.TrimSpace(*p)
				return s, s != ""
			}

			zip, ok := need(req.ZipCode)
			if !ok {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid zipCode"})
				return
			}
			state, ok := need(req.State)
			if !ok {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid state"})
				return
			}
			city, ok := need(req.City)
			if !ok {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid city"})
				return
			}
			street, ok := need(req.Street)
			if !ok {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid street"})
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

			userID := ""
			if req.UserID != nil {
				userID = strings.TrimSpace(*req.UserID)
			}
			if userID == "" {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid userId"})
				return
			}

			now := time.Now().UTC()
			ent, derr := shadom.NewWithNow(
				id,
				userID,
				zip,
				state,
				city,
				street,
				street2,
				country,
				now,
			)
			if derr != nil {
				writeShippingAddressErr(w, derr)
				return
			}

			saved, serr := h.uc.Save(ctx, ent)
			if serr != nil {
				writeShippingAddressErr(w, serr)
				return
			}

			_ = json.NewEncoder(w).Encode(saved)
			return
		}

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

	saved, err := h.uc.Save(ctx, current)
	if err != nil {
		writeShippingAddressErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(saved)
}

// DELETE /mall/shipping-addresses/{id}
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
