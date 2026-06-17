// backend/internal/adapters/out/mail/resend_client.go
package mail

import (
	"context"
	"fmt"
	"html"
	"log"
	"strings"

	"github.com/resend/resend-go/v3"
)

type ResendClient struct {
	client *resend.Client
}

func NewResendClient(apiKey string) *ResendClient {
	return &ResendClient{
		client: resend.NewClient(apiKey),
	}
}

func (c *ResendClient) Send(ctx context.Context, from, to, subject, body string) error {
	if c == nil {
		return fmt.Errorf("resend client wrapper is nil")
	}
	if c.client == nil {
		return fmt.Errorf("resend client is nil")
	}

	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	subject = strings.TrimSpace(subject)

	if from == "" {
		return fmt.Errorf("from address is empty")
	}
	if to == "" {
		return fmt.Errorf("to address is empty")
	}
	if subject == "" {
		return fmt.Errorf("subject is empty")
	}

	params := &resend.SendEmailRequest{
		From:    from,
		To:      []string{to},
		Subject: subject,
		Text:    body,
		Html:    fmt.Sprintf("<pre>%s</pre>", html.EscapeString(body)),
	}

	resp, err := c.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		return fmt.Errorf("resend send error: %w", err)
	}

	log.Printf("[resend] mail sent: id=%s to=%s subject=%s", resp.Id, to, subject)
	return nil
}
