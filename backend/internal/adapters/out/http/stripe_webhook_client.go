// backend/internal/adapters/out/http/stripe_webhook_client.go
package httpout

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type StripeWebhookClient struct {
	baseURL string
	client  *http.Client
}

type StripeWebhookPayload struct {
	InvoiceID        string `json:"invoiceId"`
	BillingAddressID string `json:"billingAddressId"`
	Amount           int    `json:"amount"`
}

// baseURL example:
// - Cloud Run: https://xxxxx.asia-northeast1.run.app
// - local: http://localhost:8080
func NewStripeWebhookClient(baseURL string) *StripeWebhookClient {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	return &StripeWebhookClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

// TriggerPaid implements CheckoutUsecase's outbound port.
func (c *StripeWebhookClient) TriggerPaid(ctx context.Context, invoiceID, billingAddressID string, amount int) error {
	if c == nil {
		return fmt.Errorf("stripe webhook client is nil")
	}
	if c.baseURL == "" {
		return fmt.Errorf("stripe webhook client baseURL is empty")
	}

	url := c.baseURL + "/mall/webhooks/stripe"

	payload := StripeWebhookPayload{
		InvoiceID:        strings.TrimSpace(invoiceID),
		BillingAddressID: strings.TrimSpace(billingAddressID),
		Amount:           amount,
	}

	b, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	// 内部呼び出し識別（将来、外部からの直叩きを制限したい場合に使える）
	req.Header.Set("X-Internal-Webhook", "1")

	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNoContent {
		return nil
	}

	// StripeWebhookHandler は 204 を返す設計なので、基本は 204 以外を失敗扱い
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	return fmt.Errorf("webhook call failed status=%d body=%s", res.StatusCode, strings.TrimSpace(string(body)))
}
