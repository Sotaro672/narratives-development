// backend\internal\adapters\in\http\mall\handler\billingAddress_handler.go
package mallHandler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	badom "narratives/internal/domain/billingAddress"
)

const billingHandlerTag = "[sns_billing_address_handler]"

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

	// ✅ /sns プレフィックス吸収（/sns/billing-addresses -> /billing-addresses）
	if strings.HasPrefix(path, "/sns/") {
		path = strings.TrimPrefix(path, "/sns")
	}

	// ✅ user_handler と同等の入口ログ
	log.Printf(
		"%s enter method=%s path=%s trace=%q contentType=%q contentLen=%d",
		billingHandlerTag,
		strings.ToUpper(r.Method),
		r.URL.Path, // 元のパス（/sns/...）を残す
		r.Header.Get("X-Cloud-Trace-Context"),
		r.Header.Get("Content-Type"),
		r.ContentLength,
	)

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
	// ✅ shipping_address と同様に「PATCH で Upsert」へ寄せる
	case r.Method == http.MethodPatch && strings.HasPrefix(path, "/billing-addresses/"):
		id := strings.TrimPrefix(path, "/billing-addresses/")
		h.patch(w, r, id)
		return

	// PUT /billing-addresses/{id}
	case r.Method == http.MethodPut && strings.HasPrefix(path, "/billing-addresses/"):
		id := strings.TrimPrefix(path, "/billing-addresses/")
		h.put(w, r, id)
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

	// ✅ raw body をログしてから decode（user/shipping と同粒度）
	raw, head, err := readBodyWithHead(r, 220)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid body"})
		return
	}
	log.Printf("%s post raw body len=%d head=%q", billingHandlerTag, len(raw), head)

	var in badom.CreateBillingAddressInput
	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	log.Printf(
		"%s post parsed userId=%q cardNumber=%q cardholderName=%q cvc=%q",
		billingHandlerTag,
		strings.TrimSpace(in.UserID),
		maskCard(strings.TrimSpace(in.CardNumber)),
		strings.TrimSpace(in.CardholderName),
		maskCVC(strings.TrimSpace(in.CVC)),
	)

	created, err := h.uc.Create(ctx, in)
	if err != nil {
		writeBillingAddressErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

// PATCH /billing-addresses/{id}
// ✅ shipping_address と同じ：PATCH=Upsert（not_found のとき create）
func (h *BillingAddressHandler) patch(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	raw, head, err := readBodyWithHead(r, 220)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid body"})
		return
	}
	log.Printf("%s patch raw body id=%q len=%d head=%q", billingHandlerTag, id, len(raw), head)

	// PATCH は UpdateInput（部分更新）を受ける
	var in badom.UpdateBillingAddressInput
	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	// まず更新を試みる
	log.Printf("%s patch usecase.Update start id=%q", billingHandlerTag, id)
	updated, err := h.uc.Update(ctx, id, in)
	if err == nil {
		log.Printf("%s patch ok (updated) id=%q userId=%q", billingHandlerTag, updated.ID, updated.UserID)
		_ = json.NewEncoder(w).Encode(updated)
		return
	}

	// not_found の場合だけ upsert-create
	if errors.Is(err, badom.ErrNotFound) {
		log.Printf("%s patch get not_found -> upsert create id=%q", billingHandlerTag, id)

		// ✅ Create 入力に変換（id は指定できない設計なので、userId は body か id を採用）
		userID := pickUserIDForUpsert(id, in)

		createIn := badom.CreateBillingAddressInput{
			UserID:         userID,
			CardNumber:     pickPtr(in.CardNumber),
			CardholderName: pickPtr(in.CardholderName),
			CVC:            pickPtr(in.CVC),
		}

		// createdAt/updatedAt はポインタなので、必要ならここで付与（なければ repo が now を使う）
		now := time.Now().UTC()
		createIn.CreatedAt = &now
		createIn.UpdatedAt = &now

		log.Printf(
			"%s patch usecase.Create(upsert-create) start id=%q userId=%q cardNumber=%q cardholderName=%q cvc=%q",
			billingHandlerTag,
			id,
			strings.TrimSpace(createIn.UserID),
			maskCard(strings.TrimSpace(createIn.CardNumber)),
			strings.TrimSpace(createIn.CardholderName),
			maskCVC(strings.TrimSpace(createIn.CVC)),
		)

		created, cerr := h.uc.Create(ctx, createIn)
		if cerr != nil {
			writeBillingAddressErr(w, cerr)
			return
		}

		w.WriteHeader(http.StatusCreated)
		log.Printf("%s patch ok (created) id=%q userId=%q", billingHandlerTag, created.ID, created.UserID)
		_ = json.NewEncoder(w).Encode(created)
		return
	}

	// その他エラー
	writeBillingAddressErr(w, err)
}

// PUT /billing-addresses/{id}
// ✅ PUT は従来通り update として扱う（Upsert はしない）
func (h *BillingAddressHandler) put(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	raw, head, err := readBodyWithHead(r, 220)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid body"})
		return
	}
	log.Printf("%s put raw body id=%q len=%d head=%q", billingHandlerTag, id, len(raw), head)

	var in badom.UpdateBillingAddressInput
	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	log.Printf("%s put usecase.Update start id=%q", billingHandlerTag, id)
	updated, err := h.uc.Update(ctx, id, in)
	if err != nil {
		writeBillingAddressErr(w, err)
		return
	}

	log.Printf("%s put ok (updated) id=%q userId=%q", billingHandlerTag, updated.ID, updated.UserID)
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

	log.Printf("%s delete start id=%q", billingHandlerTag, id)
	if err := h.uc.Delete(ctx, id); err != nil {
		writeBillingAddressErr(w, err)
		return
	}
	log.Printf("%s delete ok id=%q", billingHandlerTag, id)

	w.WriteHeader(http.StatusNoContent)
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

// ============================================================
// helpers (local)
// ============================================================

func readBodyWithHead(r *http.Request, headN int) (raw []byte, head string, err error) {
	if r.Body == nil {
		return []byte{}, "", nil
	}
	defer r.Body.Close()

	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, "", err
	}

	h := string(b)
	h = strings.TrimSpace(h)
	if headN <= 0 {
		headN = 200
	}
	if len(h) > headN {
		h = h[:headN]
	}
	return b, h, nil
}

// UpdateInput の ptr を string に落とす（nil のとき ""）
func pickPtr(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

// upsert-create 時の userId 決定：body に userId が無い型なので、まず id を採用。
// ※ 将来 UpdateBillingAddressInput に UserID を追加したらここで優先できる
func pickUserIDForUpsert(id string, _ badom.UpdateBillingAddressInput) string {
	return strings.TrimSpace(id)
}

// cardNumber/cvc はログで伏せる
func maskCard(v string) string {
	s := strings.TrimSpace(v)
	if s == "" {
		return ""
	}
	d := strings.ReplaceAll(s, " ", "")
	d = strings.ReplaceAll(d, "-", "")
	if len(d) <= 4 {
		return "****"
	}
	return "**** **** **** " + d[len(d)-4:]
}

func maskCVC(v string) string {
	s := strings.TrimSpace(v)
	if s == "" {
		return ""
	}
	if len(s) <= 1 {
		return "*"
	}
	return "***"
}
