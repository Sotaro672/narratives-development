// backend/internal/adapters/out/mail/order_cancellation_mailer.go
package mail

import (
	"context"
	"fmt"
	"strings"

	orderdom "narratives/internal/domain/order"
)

const orderCancellationSubject = "【AMOL】商品のキャンセルを受領しました"

type OrderCancellationEmailClient interface {
	Send(
		ctx context.Context,
		from string,
		to string,
		subject string,
		body string,
	) error
}

type OrderCancellationMailer struct {
	client      OrderCancellationEmailClient
	fromAddress string
}

func NewOrderCancellationMailer(
	client OrderCancellationEmailClient,
	fromAddress string,
) *OrderCancellationMailer {
	return &OrderCancellationMailer{
		client:      client,
		fromAddress: strings.TrimSpace(fromAddress),
	}
}

func (m *OrderCancellationMailer) SendOrderCancellationReceipt(
	ctx context.Context,
	toEmail string,
	orderID string,
	itemIndex int,
) error {
	if m == nil {
		return fmt.Errorf(
			"order cancellation mailer is nil",
		)
	}

	if m.client == nil {
		return fmt.Errorf(
			"order cancellation email client is not configured",
		)
	}

	fromAddress := strings.TrimSpace(
		m.fromAddress,
	)
	if fromAddress == "" {
		return fmt.Errorf(
			"from address is empty",
		)
	}

	toEmail = strings.ToLower(
		strings.TrimSpace(toEmail),
	)
	if toEmail == "" {
		return fmt.Errorf(
			"to email is empty",
		)
	}

	orderID = strings.TrimSpace(
		orderID,
	)
	if orderID == "" {
		return orderdom.ErrInvalidID
	}

	if itemIndex < 0 {
		return orderdom.ErrInvalidItems
	}

	body := buildOrderCancellationMailBody(
		orderID,
		itemIndex,
	)

	if err := m.client.Send(
		ctx,
		fromAddress,
		toEmail,
		orderCancellationSubject,
		body,
	); err != nil {
		return fmt.Errorf(
			"send order cancellation receipt failed: to=%s: %w",
			toEmail,
			err,
		)
	}

	return nil
}

func buildOrderCancellationMailBody(
	orderID string,
	itemIndex int,
) string {
	var builder strings.Builder

	builder.WriteString(
		"商品のキャンセルを受領しました。\n\n",
	)

	builder.WriteString(
		"注文ID:\n",
	)
	builder.WriteString(
		orderID,
	)

	builder.WriteString(
		"\n\n",
	)

	builder.WriteString(
		"キャンセルした商品:\n",
	)
	builder.WriteString(
		fmt.Sprintf(
			"注文内の商品 %d\n",
			itemIndex+1,
		),
	)

	builder.WriteString(
		"\n",
	)

	builder.WriteString(
		"キャンセル内容はAMOLの注文詳細からご確認ください。\n\n",
	)

	builder.WriteString(
		"本メールは自動送信です。\n\n",
	)

	builder.WriteString(
		"--\n",
	)
	builder.WriteString(
		"AMOL",
	)

	return builder.String()
}
