// backend/internal/adapters/in/http/mall/handler/shippingAddress_handler.go
package mallHandler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
	usecase "narratives/internal/application/usecase"
	shadom "narratives/internal/domain/shippingAddress"
)

const maxShippingAddressBodyBytes int64 = 1 << 20 // 1 MiB

// ShippingAddressHandlerは、
// /mall/me/shipping-addresses関連のHTTP endpointを処理します。
type ShippingAddressHandler struct {
	uc *usecase.ShippingAddressUsecase
}

// NewShippingAddressHandlerはHTTP Handlerを生成します。
//
// ucがnilの場合は各endpointで503を返します。
func NewShippingAddressHandler(
	uc *usecase.ShippingAddressUsecase,
) http.Handler {
	return &ShippingAddressHandler{
		uc: uc,
	}
}

// shippingAddressCreateRequestは配送先住所の作成requestです。
//
// ID、UserID、CreatedAtおよびUpdatedAtは受け取りません。
// IDと時刻はUsecaseが生成し、UserIDは認証contextから取得します。
type shippingAddressCreateRequest struct {
	ZipCode string  `json:"zipCode"`
	State   string  `json:"state"`
	City    string  `json:"city"`
	Street  string  `json:"street"`
	Street2 *string `json:"street2,omitempty"`
	Country *string `json:"country,omitempty"`
}

// shippingAddressUpdateRequestは配送先住所の部分更新requestです。
//
// nilは変更なしを表します。
// Street2へ空文字を指定すると明示的に消去します。
type shippingAddressUpdateRequest struct {
	ZipCode *string `json:"zipCode,omitempty"`
	State   *string `json:"state,omitempty"`
	City    *string `json:"city,omitempty"`
	Street  *string `json:"street,omitempty"`
	Street2 *string `json:"street2,omitempty"`
	Country *string `json:"country,omitempty"`
}

// ServeHTTPはshippingAddress endpointのroutingを行います。
func (h *ShippingAddressHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimSuffix(r.URL.Path, "/")

	switch {
	case r.Method == http.MethodGet &&
		path == "/mall/me/shipping-addresses":
		h.listMe(w, r)
		return

	case r.Method == http.MethodGet &&
		strings.HasPrefix(
			path,
			"/mall/me/shipping-addresses/",
		):
		id := strings.TrimPrefix(
			path,
			"/mall/me/shipping-addresses/",
		)
		h.get(w, r, id)
		return

	case r.Method == http.MethodPost &&
		path == "/mall/me/shipping-addresses":
		h.post(w, r)
		return

	case r.Method == http.MethodPatch &&
		strings.HasPrefix(
			path,
			"/mall/me/shipping-addresses/",
		):
		id := strings.TrimPrefix(
			path,
			"/mall/me/shipping-addresses/",
		)
		h.patch(w, r, id)
		return

	case r.Method == http.MethodDelete &&
		strings.HasPrefix(
			path,
			"/mall/me/shipping-addresses/",
		):
		id := strings.TrimPrefix(
			path,
			"/mall/me/shipping-addresses/",
		)
		h.del(w, r, id)
		return

	default:
		writeShippingAddressJSON(
			w,
			http.StatusNotFound,
			map[string]string{
				"error": "not_found",
			},
		)
	}
}

// --------------------
// Helpers
// --------------------

func (h *ShippingAddressHandler) requireUsecase(
	w http.ResponseWriter,
) bool {
	if h != nil && h.uc != nil {
		return true
	}

	writeShippingAddressJSON(
		w,
		http.StatusServiceUnavailable,
		map[string]string{
			"error": "shipping_address_usecase_not_initialized",
		},
	)

	return false
}

// requireUIDは認証middlewareがcontextへ設定したUIDを取得します。
//
// header、queryおよびrequest bodyからUIDを受け取りません。
func (h *ShippingAddressHandler) requireUID(
	w http.ResponseWriter,
	r *http.Request,
) (string, bool) {
	uid, ok := middleware.CurrentUserUID(r)
	if ok && strings.TrimSpace(uid) != "" {
		return uid, true
	}

	writeShippingAddressJSON(
		w,
		http.StatusUnauthorized,
		map[string]string{
			"error": "unauthorized",
		},
	)

	return "", false
}

// decodeShippingAddressJSONはrequest bodyを厳格にdecodeします。
//
//   - 最大サイズは1 MiB
//   - 未定義fieldを拒否
//   - JSON値が複数存在するbodyを拒否
//   - 空bodyを拒否
func decodeShippingAddressJSON(
	w http.ResponseWriter,
	r *http.Request,
	dst any,
) error {
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxShippingAddressBodyBytes,
	)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	var trailing any
	err := decoder.Decode(&trailing)
	if !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New(
				"request body must contain exactly one JSON value",
			)
		}

		return err
	}

	return nil
}

func writeShippingAddressJSON(
	w http.ResponseWriter,
	status int,
	body any,
) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeInvalidShippingAddressJSON(
	w http.ResponseWriter,
) {
	writeShippingAddressJSON(
		w,
		http.StatusBadRequest,
		map[string]string{
			"error": "invalid_json",
		},
	)
}

// --------------------
// GET /mall/me/shipping-addresses
// --------------------

func (h *ShippingAddressHandler) listMe(
	w http.ResponseWriter,
	r *http.Request,
) {
	uid, ok := h.requireUID(w, r)
	if !ok {
		return
	}

	if !h.requireUsecase(w) {
		return
	}

	addresses, err := h.uc.ListByUserID(
		r.Context(),
		uid,
	)
	if err != nil {
		writeShippingAddressErr(w, err)
		return
	}

	writeShippingAddressJSON(
		w,
		http.StatusOK,
		addresses,
	)
}

// --------------------
// GET /mall/me/shipping-addresses/{id}
// --------------------

func (h *ShippingAddressHandler) get(
	w http.ResponseWriter,
	r *http.Request,
	id string,
) {
	uid, ok := h.requireUID(w, r)
	if !ok {
		return
	}

	if !h.requireUsecase(w) {
		return
	}

	// document IDと認証UIDの両方を条件として取得します。
	// 対象が他ユーザーの所有物である場合も404を返します。
	address, err := h.uc.GetByUser(
		r.Context(),
		id,
		uid,
	)
	if err != nil {
		writeShippingAddressErr(w, err)
		return
	}

	writeShippingAddressJSON(
		w,
		http.StatusOK,
		address,
	)
}

// --------------------
// POST /mall/me/shipping-addresses
// --------------------

func (h *ShippingAddressHandler) post(
	w http.ResponseWriter,
	r *http.Request,
) {
	uid, ok := h.requireUID(w, r)
	if !ok {
		return
	}

	if !h.requireUsecase(w) {
		return
	}

	var request shippingAddressCreateRequest
	if err := decodeShippingAddressJSON(
		w,
		r,
		&request,
	); err != nil {
		writeInvalidShippingAddressJSON(w)
		return
	}

	street2 := ""
	if request.Street2 != nil {
		street2 = *request.Street2
	}

	country := ""
	if request.Country != nil {
		country = *request.Country
	}

	// HandlerはDTO変換だけを行います。
	// Domain生成、Countryの既定値、UUID採番および時刻設定は
	// UsecaseとDomainへ委譲します。
	input := usecase.CreateShippingAddressInput{
		ZipCode: request.ZipCode,
		State:   request.State,
		City:    request.City,
		Street:  request.Street,
		Street2: street2,
		Country: country,
	}

	created, err := h.uc.Create(
		r.Context(),
		uid,
		input,
	)
	if err != nil {
		writeShippingAddressErr(w, err)
		return
	}

	writeShippingAddressJSON(
		w,
		http.StatusCreated,
		created,
	)
}

// --------------------
// PATCH /mall/me/shipping-addresses/{id}
// --------------------

func (h *ShippingAddressHandler) patch(
	w http.ResponseWriter,
	r *http.Request,
	id string,
) {
	uid, ok := h.requireUID(w, r)
	if !ok {
		return
	}

	if !h.requireUsecase(w) {
		return
	}

	var request shippingAddressUpdateRequest
	if err := decodeShippingAddressJSON(
		w,
		r,
		&request,
	); err != nil {
		writeInvalidShippingAddressJSON(w)
		return
	}

	// Handlerでは既存Entityの取得、mergeおよびDomain更新を行いません。
	// 所有者確認付き取得、merge、Domain検証および永続化は
	// Usecaseが一度だけ実行します。
	input := usecase.UpdateShippingAddressInput{
		ZipCode: request.ZipCode,
		State:   request.State,
		City:    request.City,
		Street:  request.Street,
		Street2: request.Street2,
		Country: request.Country,
	}

	updated, err := h.uc.Update(
		r.Context(),
		id,
		uid,
		input,
	)
	if err != nil {
		writeShippingAddressErr(w, err)
		return
	}

	writeShippingAddressJSON(
		w,
		http.StatusOK,
		updated,
	)
}

// --------------------
// DELETE /mall/me/shipping-addresses/{id}
// --------------------

func (h *ShippingAddressHandler) del(
	w http.ResponseWriter,
	r *http.Request,
	id string,
) {
	uid, ok := h.requireUID(w, r)
	if !ok {
		return
	}

	if !h.requireUsecase(w) {
		return
	}

	if err := h.uc.DeleteByUser(
		r.Context(),
		id,
		uid,
	); err != nil {
		writeShippingAddressErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --------------------
// Error mapping
// --------------------

func isInvalidShippingAddressError(err error) bool {
	return errors.Is(err, shadom.ErrInvalidID) ||
		errors.Is(err, shadom.ErrInvalidUserID) ||
		errors.Is(err, shadom.ErrInvalidZipCode) ||
		errors.Is(err, shadom.ErrInvalidState) ||
		errors.Is(err, shadom.ErrInvalidCity) ||
		errors.Is(err, shadom.ErrInvalidStreet) ||
		errors.Is(err, shadom.ErrInvalidCountry) ||
		errors.Is(err, shadom.ErrInvalidCreatedAt) ||
		errors.Is(err, shadom.ErrInvalidUpdatedAt)
}

func writeShippingAddressErr(
	w http.ResponseWriter,
	err error,
) {
	statusCode := http.StatusInternalServerError

	switch {
	case isInvalidShippingAddressError(err):
		statusCode = http.StatusBadRequest

	case errors.Is(err, shadom.ErrNotFound):
		statusCode = http.StatusNotFound

	case errors.Is(err, shadom.ErrConflict):
		statusCode = http.StatusConflict
	}

	writeShippingAddressJSON(
		w,
		statusCode,
		map[string]string{
			"error": err.Error(),
		},
	)
}
