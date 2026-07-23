// backend/internal/adapters/out/mail/resend_client.go
package mail

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log"
	"net"
	"strings"

	"github.com/resend/resend-go/v3"
)

type ResendClient struct {
	client *resend.Client
}

func NewResendClient(apiKey string) *ResendClient {
	return &ResendClient{
		client: resend.NewClient(strings.TrimSpace(apiKey)),
	}
}

// Sendは既存メール機能との互換性を維持します。
func (c *ResendClient) Send(
	ctx context.Context,
	from string,
	to string,
	subject string,
	body string,
) error {
	_, err := c.SendWithResult(
		ctx,
		from,
		to,
		subject,
		body,
		"",
	)

	return err
}

// SendWithResultは招待delivery向けです。
// provider message IDと、失敗が再試行可能かを返します。
func (c *ResendClient) SendWithResult(
	ctx context.Context,
	from string,
	to string,
	subject string,
	body string,
	idempotencyKey string,
) (EmailSendResult, error) {
	if c == nil {
		return EmailSendResult{
			Retryable: false,
		}, fmt.Errorf("resend client wrapper is nil")
	}

	if c.client == nil {
		return EmailSendResult{
			Retryable: false,
		}, fmt.Errorf("resend client is nil")
	}

	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	subject = strings.TrimSpace(subject)
	idempotencyKey = strings.TrimSpace(idempotencyKey)

	if from == "" {
		return EmailSendResult{
			Retryable: false,
		}, fmt.Errorf("from address is empty")
	}

	if to == "" {
		return EmailSendResult{
			Retryable: false,
		}, fmt.Errorf("to address is empty")
	}

	if subject == "" {
		return EmailSendResult{
			Retryable: false,
		}, fmt.Errorf("subject is empty")
	}

	params := &resend.SendEmailRequest{
		From:    from,
		To:      []string{to},
		Subject: subject,
		Text:    body,
		Html: fmt.Sprintf(
			"<pre>%s</pre>",
			html.EscapeString(body),
		),
	}

	resp, err := c.client.Emails.SendWithContext(
		ctx,
		params,
	)
	if err != nil {
		return EmailSendResult{
			Retryable: isRetryableResendError(err),
		}, fmt.Errorf("resend send error: %w", err)
	}

	if resp == nil {
		return EmailSendResult{
			Retryable: true,
		}, fmt.Errorf("resend response is nil")
	}

	providerMessageID := strings.TrimSpace(resp.Id)

	log.Printf(
		"[resend] mail sent: id=%s to=%s subject=%s idempotencyKey=%s",
		providerMessageID,
		to,
		subject,
		idempotencyKey,
	)

	return EmailSendResult{
		ProviderMessageID: providerMessageID,
		Retryable:         false,
	}, nil
}

func isRetryableResendError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	if errors.Is(err, context.Canceled) {
		return false
	}

	var networkError net.Error
	if errors.As(err, &networkError) {
		return networkError.Timeout() ||
			networkError.Temporary()
	}

	message := strings.ToLower(err.Error())

	retryableFragments := []string{
		"timeout",
		"temporarily unavailable",
		"temporary failure",
		"connection reset",
		"connection refused",
		"too many requests",
		"rate limit",
		"status 429",
		"status code 429",
		"status 500",
		"status code 500",
		"status 502",
		"status code 502",
		"status 503",
		"status code 503",
		"status 504",
		"status code 504",
	}

	for _, fragment := range retryableFragments {
		if strings.Contains(message, fragment) {
			return true
		}
	}

	return false
}
