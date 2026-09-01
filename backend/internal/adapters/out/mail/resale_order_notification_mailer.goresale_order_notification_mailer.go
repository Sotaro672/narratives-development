// backend/internal/adapters/out/mail/resale_order_notification_mailer.go
package mail

import (
	"context"
	"fmt"
	"strings"
)

const resaleOrderNotificationSubject = "【AMOL】出品商品に注文が入りました"

type ResaleOrderNotificationEmailClient interface {
	Send(ctx context.Context, from string, to string, subject string, body string) error
}

type ResaleOrderNotificationMailer struct {
	client          ResaleOrderNotificationEmailClient
	fromAddress     string
	frontendBaseURL string
}

func NewResaleOrderNotificationMailer(
	client ResaleOrderNotificationEmailClient,
	fromAddress string,
	frontendBaseURL string,
) *ResaleOrderNotificationMailer {
	return &ResaleOrderNotificationMailer{
		client:          client,
		fromAddress:     fromAddress,
		frontendBaseURL: frontendBaseURL,
	}
}

func (m *ResaleOrderNotificationMailer) SendResaleOrderNotification(
	ctx context.Context,
	toEmail string,
	orderID string,
	itemIndex int,
	resaleID string,
	price int,
) error {
	if m == nil {
		return fmt.Errorf("resale order notification mailer is nil")
	}
	if m.client == nil {
		return fmt.Errorf("resale order notification email client is not configured")
	}
	if m.fromAddress == "" {
		return fmt.Errorf("from address is empty")
	}
	if m.frontendBaseURL == "" {
		return fmt.Errorf("frontend base url is empty")
	}
	if toEmail == "" {
		return fmt.Errorf("to email is empty")
	}
	if orderID == "" {
		return fmt.Errorf("order id is empty")
	}
	if itemIndex < 0 {
		return fmt.Errorf("item index is invalid")
	}
	if resaleID == "" {
		return fmt.Errorf("resale id is empty")
	}
	if price < 0 {
		return fmt.Errorf("price is invalid")
	}

	chatURL := fmt.Sprintf(
		"%s/chats/trades/order-items/%s/%d",
		strings.TrimRight(m.frontendBaseURL, "/"),
		orderID,
		itemIndex,
	)

	body := buildResaleOrderNotificationMailBody(
		orderID,
		resaleID,
		price,
		chatURL,
	)

	if err := m.client.Send(
		ctx,
		m.fromAddress,
		toEmail,
		resaleOrderNotificationSubject,
		body,
	); err != nil {
		return fmt.Errorf(
			"send resale order notification failed: to=%s: %w",
			toEmail,
			err,
		)
	}

	return nil
}

func buildResaleOrderNotificationMailBody(
	orderID string,
	resaleID string,
	price int,
	chatURL string,
) string {
	var builder strings.Builder

	builder.WriteString("出品中の商品に注文が入りました。\n\n")
	builder.WriteString("注文情報\n")
	builder.WriteString(fmt.Sprintf("注文ID: %s\n", orderID))
	builder.WriteString(fmt.Sprintf("出品ID: %s\n", resaleID))
	builder.WriteString(fmt.Sprintf("販売価格: %d円\n", price))
	builder.WriteString("\n")
	builder.WriteString("購入者との取引チャットはこちら\n")
	builder.WriteString(chatURL)
	builder.WriteString("\n\n")
	builder.WriteString("注文内容をご確認のうえ、発送の準備をお願いいたします。\n\n")
	builder.WriteString("本メールは自動送信です。\n\n")
	builder.WriteString("--\n")
	builder.WriteString("AMOL")

	return builder.String()
}
