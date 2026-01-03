// backend\internal\adapters\in\http\sns\handler\shippingAddress_handler.go
package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
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
	// ✅ user_handler と同様: 入口ログ
	trace := r.Header.Get("X-Cloud-Trace-Context")
	log.Printf("[sns_shipping_address_handler] enter method=%s path=%s trace=%q contentType=%q contentLen=%d",
		r.Method, r.URL.Path, trace, r.Header.Get("Content-Type"), r.ContentLength)

	w.Header().Set("Content-Type", "application/json")

	// ✅ 末尾スラッシュを吸収
	path := strings.TrimSuffix(r.URL.Path, "/")

	// ✅ /sns プレフィックスを吸収（/sns/shipping-addresses -> /shipping-addresses）
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
	// ✅ docId=UID 統一方針では本来 PUT /shipping-addresses/{id} が望ましいが、
	//    互換のため POST も残す（ただし id 必須）
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
		log.Printf("[sns_shipping_address_handler] not_found method=%s path=%s (raw=%s)", r.Method, path, r.URL.Path)
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
		log.Printf("[sns_shipping_address_handler] get bad_request empty_id")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	log.Printf("[sns_shipping_address_handler] get start id=%q", id)
	addr, err := h.uc.GetByID(ctx, id)
	if err != nil {
		log.Printf("[sns_shipping_address_handler] get failed id=%q err=%v", id, err)
		writeShippingAddressErr(w, err)
		return
	}

	log.Printf("[sns_shipping_address_handler] get ok id=%q userId=%q", id, strings.TrimSpace(addr.UserID))
	_ = json.NewEncoder(w).Encode(addr)
}

// POST /shipping-addresses
func (h *ShippingAddressHandler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// ✅ user_handler と同様: raw body を先に読む（400原因を残す）
	raw, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		log.Printf("[sns_shipping_address_handler] post read body failed err=%v", readErr)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid body"})
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(raw))

	log.Printf("[sns_shipping_address_handler] post raw body len=%d head=%q", len(raw), headString(raw, 300))

	// frontend/sns の入力欄に合わせる（zipCode/state/city/street/street2）
	// ✅ docId=UID 統一: id(=uid) を必須にする
	type createReq struct {
		ID      string  `json:"id"` // ✅ 追加: docId = UID
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
		log.Printf("[sns_shipping_address_handler] post decode failed err=%v bodyHead=%q", err, headString(raw, 300))
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	id := strings.TrimSpace(req.ID)
	if id == "" {
		// ✅ docId=UID 統一方針: id が無い POST は許可しない（Firestore自動採番をさせない）
		log.Printf("[sns_shipping_address_handler] post bad_request missing id (docId=uid required)")
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

	log.Printf("[sns_shipping_address_handler] post parsed id=%q userId=%q zip=%q state=%q city=%q street=%q street2=%q country=%q",
		id,
		strings.TrimSpace(req.UserID),
		strings.TrimSpace(req.ZipCode),
		strings.TrimSpace(req.State),
		strings.TrimSpace(req.City),
		strings.TrimSpace(req.Street),
		strings.TrimSpace(street2),
		strings.TrimSpace(country),
	)

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
		log.Printf("[sns_shipping_address_handler] post domain.NewWithNow failed id=%q err=%v", id, err)
		writeShippingAddressErr(w, err)
		return
	}

	// ✅ docId=UID: Create は「同じIDが既にあると Conflict」になる。
	//    住所は “upsert” が自然なので Save に寄せる（既存なら更新、無ければ作成）。
	log.Printf("[sns_shipping_address_handler] post usecase.Save(upsert) start id=%q", id)
	saved, err := h.uc.Save(ctx, ent)
	if err != nil {
		log.Printf("[sns_shipping_address_handler] post usecase.Save(upsert) failed id=%q err=%v", id, err)
		writeShippingAddressErr(w, err)
		return
	}

	// 「作れた/更新できた」の判定を厳密にするには Usecase から返す必要があるので、
	// ここでは Created を返さず 200 に統一してもいいが、互換のため 201 で返す。
	log.Printf("[sns_shipping_address_handler] post ok id=%q userId=%q", strings.TrimSpace(saved.ID), strings.TrimSpace(saved.UserID))
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(saved)
}

// PATCH /shipping-addresses/{id}
func (h *ShippingAddressHandler) patch(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		log.Printf("[sns_shipping_address_handler] patch bad_request empty_id")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	raw, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		log.Printf("[sns_shipping_address_handler] patch read body failed id=%q err=%v", id, readErr)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid body"})
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(raw))
	log.Printf("[sns_shipping_address_handler] patch raw body id=%q len=%d head=%q", id, len(raw), headString(raw, 300))

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
		log.Printf("[sns_shipping_address_handler] patch decode failed id=%q err=%v bodyHead=%q", id, err, headString(raw, 300))
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	// ✅ docId=UID 統一: PATCH は「存在すれば更新・無ければ作成」でもよい（フロントの POST→PATCH 移行対策）
	// まず Get を試し、NotFound なら “新規として組み立てて Save”
	current, err := h.uc.GetByID(ctx, id)
	if err != nil {
		if err == shadom.ErrNotFound {
			// upsert-create path
			log.Printf("[sns_shipping_address_handler] patch get not_found -> upsert create id=%q", id)

			// PATCH だけで作る場合は、必須フィールドが揃ってないと domain が弾く。
			// ここでは「全部必須」扱いにして、足りないなら 400。
			need := func(p *string, name string) (string, bool) {
				if p == nil {
					log.Printf("[sns_shipping_address_handler] patch bad_request missing %s for create id=%q", name, id)
					return "", false
				}
				s := strings.TrimSpace(*p)
				if s == "" {
					log.Printf("[sns_shipping_address_handler] patch bad_request empty %s for create id=%q", name, id)
					return "", false
				}
				return s, true
			}

			zip, ok := need(req.ZipCode, "zipCode")
			if !ok {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid zipCode"})
				return
			}
			state, ok := need(req.State, "state")
			if !ok {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid state"})
				return
			}
			city, ok := need(req.City, "city")
			if !ok {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid city"})
				return
			}
			street, ok := need(req.Street, "street")
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

			// userId は domain 的に必須な可能性が高いので、無ければ 400
			userID := ""
			if req.UserID != nil {
				userID = strings.TrimSpace(*req.UserID)
			}
			if userID == "" {
				log.Printf("[sns_shipping_address_handler] patch bad_request missing userId for create id=%q", id)
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
				log.Printf("[sns_shipping_address_handler] patch domain.NewWithNow failed id=%q err=%v", id, derr)
				writeShippingAddressErr(w, derr)
				return
			}

			log.Printf("[sns_shipping_address_handler] patch usecase.Save(upsert-create) start id=%q", id)
			saved, serr := h.uc.Save(ctx, ent)
			if serr != nil {
				log.Printf("[sns_shipping_address_handler] patch usecase.Save(upsert-create) failed id=%q err=%v", id, serr)
				writeShippingAddressErr(w, serr)
				return
			}

			log.Printf("[sns_shipping_address_handler] patch ok (created) id=%q userId=%q", strings.TrimSpace(saved.ID), strings.TrimSpace(saved.UserID))
			_ = json.NewEncoder(w).Encode(saved)
			return
		}

		log.Printf("[sns_shipping_address_handler] patch get failed id=%q err=%v", id, err)
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
		log.Printf("[sns_shipping_address_handler] patch domain.UpdateFromForm failed id=%q err=%v", id, err)
		writeShippingAddressErr(w, err)
		return
	}

	log.Printf("[sns_shipping_address_handler] patch usecase.Save start id=%q", id)
	saved, err := h.uc.Save(ctx, current)
	if err != nil {
		log.Printf("[sns_shipping_address_handler] patch usecase.Save failed id=%q err=%v", id, err)
		writeShippingAddressErr(w, err)
		return
	}

	log.Printf("[sns_shipping_address_handler] patch ok id=%q userId=%q", strings.TrimSpace(saved.ID), strings.TrimSpace(saved.UserID))
	_ = json.NewEncoder(w).Encode(saved)
}

// DELETE /shipping-addresses/{id}
func (h *ShippingAddressHandler) del(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		log.Printf("[sns_shipping_address_handler] delete bad_request empty_id")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	log.Printf("[sns_shipping_address_handler] delete start id=%q", id)
	if err := h.uc.Delete(ctx, id); err != nil {
		log.Printf("[sns_shipping_address_handler] delete failed id=%q err=%v", id, err)
		writeShippingAddressErr(w, err)
		return
	}

	log.Printf("[sns_shipping_address_handler] delete ok id=%q", id)
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
