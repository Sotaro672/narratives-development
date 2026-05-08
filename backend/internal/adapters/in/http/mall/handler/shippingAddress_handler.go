// backend/internal/adapters/in/http/mall/handler/shippingAddress_handler.go
package mallHandler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"narratives/internal/adapters/in/http/middleware"
	usecase "narratives/internal/application/usecase"
	shadom "narratives/internal/domain/shippingAddress"
)

// ShippingAddressHandler は /mall/me/shipping-addresses 関連のエンドポイントを担当します。
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
	// GET /mall/me/shipping-addresses (一覧)
	case r.Method == http.MethodGet && path == "/mall/me/shipping-addresses":
		h.listMe(w, r)
		return

	// GET /mall/me/shipping-addresses/{id}
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/mall/me/shipping-addresses/"):
		id := strings.TrimPrefix(path, "/mall/me/shipping-addresses/")
		h.get(w, r, id)
		return

	// POST /mall/me/shipping-addresses
	// ✅ 起票は Create のみ（docId は usecase 側でランダム採番）
	case r.Method == http.MethodPost && path == "/mall/me/shipping-addresses":
		h.post(w, r)
		return

	// PATCH /mall/me/shipping-addresses/{id}
	// ✅ 更新は Update のみ（存在しない場合は 404）
	case r.Method == http.MethodPatch && strings.HasPrefix(path, "/mall/me/shipping-addresses/"):
		id := strings.TrimPrefix(path, "/mall/me/shipping-addresses/")
		h.patch(w, r, id)
		return

	// DELETE /mall/me/shipping-addresses/{id}
	case r.Method == http.MethodDelete && strings.HasPrefix(path, "/mall/me/shipping-addresses/"):
		id := strings.TrimPrefix(path, "/mall/me/shipping-addresses/")
		h.del(w, r, id)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

// --------------------
// Helpers
// --------------------

// ✅ /me 系の uid は UserAuthMiddleware により context へ格納されている想定。
// このハンドラではヘッダ/body から uid を受け取らない（旧式互換も削除）。
func (h *ShippingAddressHandler) requireUID(w http.ResponseWriter, r *http.Request) (string, bool) {
	uid, ok := middleware.CurrentUserUID(r)
	if ok && uid != "" {
		return uid, true
	}
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
	return "", false
}

// --------------------
// GET /mall/me/shipping-addresses (一覧)
// --------------------

func (h *ShippingAddressHandler) listMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	uid, ok := h.requireUID(w, r)
	if !ok {
		return
	}

	addrs, err := h.uc.ListByUserID(ctx, uid)
	if err != nil {
		writeShippingAddressErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(addrs)
}

// --------------------
// GET /mall/me/shipping-addresses/{id}
// --------------------

func (h *ShippingAddressHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	_, ok := h.requireUID(w, r)
	if !ok {
		return
	}

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

// --------------------
// POST /mall/me/shipping-addresses
// --------------------

func (h *ShippingAddressHandler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	uid, ok := h.requireUID(w, r)
	if !ok {
		return
	}

	raw, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid body"})
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(raw))

	// frontend/mall の入力欄に合わせる（zipCode/state/city/street/street2）
	// ✅ docId は usecase がランダム採番（body では受け取らない）
	// ✅ userId は /me の文脈では受け取らない（旧式互換削除）
	type createReq struct {
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
		street2 = *req.Street2
	}

	country := "JP"
	if req.Country != nil {
		if c := *req.Country; c != "" {
			country = c
		}
	}

	now := time.Now().UTC()

	// ✅ Create 時は「id なし」を許容する Create 用コンストラクタを使う
	//    （id は usecase が採番するのでここでは不要）
	ent, err := shadom.NewForCreateWithNow(
		uid, // userId は auth から確定
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

	created, err := h.uc.Create(ctx, uid, ent)
	if err != nil {
		writeShippingAddressErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

// --------------------
// PATCH /mall/me/shipping-addresses/{id}
// --------------------

func (h *ShippingAddressHandler) patch(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	uid, ok := h.requireUID(w, r)
	if !ok {
		return
	}

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
	}

	var req patchReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	// ✅ まず既存取得（存在しなければ 404）
	current, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeShippingAddressErr(w, err)
		return
	}

	// マージ（current は *ShippingAddress）
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
	if country == "" {
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

	updated, err := h.uc.Update(ctx, id, uid, *current)
	if err != nil {
		writeShippingAddressErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(updated)
}

// --------------------
// DELETE /mall/me/shipping-addresses/{id}
// --------------------

func (h *ShippingAddressHandler) del(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	uid, ok := h.requireUID(w, r)
	if !ok {
		return
	}

	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	// ✅ 本人チェック付き delete（推奨）
	if err := h.uc.DeleteByUser(ctx, id, uid); err != nil {
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
