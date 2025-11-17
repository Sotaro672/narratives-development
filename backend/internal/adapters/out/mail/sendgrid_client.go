package mail

import (
	"context"
	"fmt"
	"log"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// SendGridClient implements EmailClient interface
type SendGridClient struct {
	apiKey string
}

func NewSendGridClient(apiKey string) *SendGridClient {
	return &SendGridClient{apiKey: apiKey}
}

// Send sends an email using SendGrid
func (c *SendGridClient) Send(ctx context.Context, from, to, subject, body string) error {
	if c.apiKey == "" {
		return fmt.Errorf("sendgrid api key is empty")
	}
	if from == "" {
		return fmt.Errorf("from address is empty")
	}
	if to == "" {
		return fmt.Errorf("to address is empty")
	}

	// Email objects
	fromEmail := mail.NewEmail("Narratives", from)
	toEmail := mail.NewEmail("", to)

	// Text & HTML — HTML は最低限整形
	plainTextContent := body
	htmlContent := fmt.Sprintf("<pre>%s</pre>", body)

	// Build the email
	message := mail.NewSingleEmail(
		fromEmail,
		subject,
		toEmail,
		plainTextContent,
		htmlContent,
	)

	// Create client
	client := sendgrid.NewSendClient(c.apiKey)

	// Send email
	response, err := client.Send(message)
	if err != nil {
		return fmt.Errorf("sendgrid send error: %w", err)
	}

	if response.StatusCode >= 400 {
		log.Printf("[sendgrid] error status=%d, body=%s", response.StatusCode, response.Body)
		return fmt.Errorf(
			"sendgrid send failed: status=%d, body=%s",
			response.StatusCode,
			response.Body,
		)
	}

	log.Printf("[sendgrid] mail sent: status=%d to=%s subject=%s",
		response.StatusCode, to, subject)

	return nil
}
