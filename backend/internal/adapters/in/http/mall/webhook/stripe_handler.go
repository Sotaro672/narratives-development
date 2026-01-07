// backend/internal/adapters/in/http/mall/webhook/stripe_handler.go
package webhook

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	uc "narratives/internal/application/usecase"
	paymentdom "narratives/internal/domain/payment"
)

// StripeWebhookHandler is a TEMP stub handler.
// 現状「外部Stripeと接続できない」前提のため、受け取ったイベントは全て「支払い成功（paid）」として扱う。
// ただし ✅ invoice が存在することは確認してから payment を起票する。
// 後で必ず：署名検証 / イベント種別判定 / amount整合 / 冪等キー / providerID保存 を実装すること。
type StripeWebhookHandler struct {
	invoiceUC *uc.InvoiceUsecase
	paymentUC *uc.PaymentUsecase
}

// NewStripeWebhookHandler creates a handler.
// invoiceUC / paymentUC are required.
func NewStripeWebhookHandler(invoiceUC *uc.InvoiceUsecase, paymentUC *uc.PaymentUsecase) http.Handler {
	return &StripeWebhookHandler{
		invoiceUC: invoiceUC,
		paymentUC: paymentUC,
	}
}

// StripeWebhookInput is a simplified payload for dev/testing.
// Stripeの実ペイロードではありません（現時点では接続できないため）。
type StripeWebhookInput struct {
	InvoiceID        string `json:"invoiceId"`
	OrderID          string `json:"orderId"` // fallback: invoiceId が無い場合に使う（docId=orderId 前提）
	BillingAddressID string `json:"billingAddressId"`
	Amount           *int   `json:"amount"` // optional: 無い場合は 0
}

func (h *StripeWebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if h.paymentUC == nil {
		writeJSONError(w, http.StatusInternalServerError, "payment usecase is not configured")
		return
	}
	if h.invoiceUC == nil {
		writeJSONError(w, http.StatusInternalServerError, "invoice usecase is not configured")
		return
	}

	// body は将来の署名検証に使うので一旦全部読む（サイズ制限あり）
	const maxBody = 1 << 20 // 1MB
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBody))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	_ = r.Body.Close()

	// ✅ いまは「接続できない」前提なので、できるだけ柔軟に invoiceId を受け取る
	var in StripeWebhookInput
	if len(bytesTrim(body)) > 0 {
		_ = json.Unmarshal(body, &in) // 失敗しても query fallback する
	}

	invoiceID := strings.TrimSpace(in.InvoiceID)
	if invoiceID == "" {
		invoiceID = strings.TrimSpace(in.OrderID)
	}
	if invoiceID == "" {
		invoiceID = strings.TrimSpace(r.URL.Query().Get("invoiceId"))
	}
	if invoiceID == "" {
		invoiceID = strings.TrimSpace(r.URL.Query().Get("orderId"))
	}

	billingAddrID := strings.TrimSpace(in.BillingAddressID)
	if billingAddrID == "" {
		billingAddrID = strings.TrimSpace(r.URL.Query().Get("billingAddressId"))
	}

	if invoiceID == "" {
		writeJSONError(w, http.StatusBadRequest, "invoiceId is required (json.invoiceId or query invoiceId/orderId)")
		return
	}
	if billingAddrID == "" {
		writeJSONError(w, http.StatusBadRequest, "billingAddressId is required (json.billingAddressId or query billingAddressId)")
		return
	}

	amount := 0
	if in.Amount != nil {
		amount = *in.Amount
	}

	// ✅ 重要：invoice 起票済み確認（存在しないなら payment を作らない）
	exists, exErr := h.invoiceUC.Exists(r.Context(), invoiceID)
	if exErr != nil {
		log.Printf("[mall/webhook/stripe] invoice exists check failed invoiceId=%s err=%v", invoiceID, exErr)
		writeJSONError(w, http.StatusInternalServerError, "failed to check invoice existence")
		return
	}
	if !exists {
		log.Printf("[mall/webhook/stripe] invoice not found invoiceId=%s -> 404", invoiceID)
		writeJSONError(w, http.StatusNotFound, "invoice not found")
		return
	}

	// ✅ 冪等性（最低限）：invoiceId ですでに payment があるならOKで返す
	existing, gErr := h.paymentUC.GetByInvoiceID(r.Context(), invoiceID)
	if gErr == nil && len(existing) > 0 {
		log.Printf("[mall/webhook/stripe] already processed invoiceId=%s payments=%d -> 204", invoiceID, len(existing))
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// ✅ TEMP: 全て「支払い成功」として payment を起票する
	created, cErr := h.paymentUC.Create(r.Context(), paymentdom.CreatePaymentInput{
		InvoiceID:        invoiceID,
		BillingAddressID: billingAddrID,
		Amount:           amount,
		Status:           paymentdom.PaymentStatus("paid"), // AllowedStatusesが空なら任意の非空文字列OK
		ErrorType:        nil,
	})
	if cErr != nil {
		log.Printf("[mall/webhook/stripe] create payment failed invoiceId=%s err=%v", invoiceID, cErr)
		writeJSONError(w, http.StatusInternalServerError, "failed to create payment")
		return
	}

	log.Printf("[mall/webhook/stripe] mock paid OK invoiceId=%s paymentInvoiceId=%s amount=%d",
		invoiceID, safePaymentKey(created), amount,
	)

	// webhook は “受領した” ことが重要なので、内容は返さず 204
	w.WriteHeader(http.StatusNoContent)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": msg,
	})
}

func safePaymentKey(p *paymentdom.Payment) string {
	if p == nil {
		return ""
	}
	// domainでは docId=invoiceId 前提なので、識別子として InvoiceID を返す
	return p.InvoiceID
}

func bytesTrim(b []byte) []byte {
	s := strings.TrimSpace(string(b))
	return []byte(s)
}
