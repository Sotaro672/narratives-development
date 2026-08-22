// backend/internal/adapters/out/mail/order_dispatch_mailer.go
package mail

import (
	"context"
	"fmt"
	"strings"

	orderdispatchuc "narratives/internal/application/usecase"
	orderdom "narratives/internal/domain/order"
)

const orderDispatchNotificationSubject = "【AMOL】ご注文の商品を発送しました"

type OrderDispatchNotificationEmailClient interface {
	SendWithResult(
		ctx context.Context,
		from string,
		to string,
		subject string,
		body string,
		idempotencyKey string,
	) (EmailSendResult, error)
}

type OrderDispatchNotificationMailer struct {
	client      OrderDispatchNotificationEmailClient
	fromAddress string
}

var _ orderdispatchuc.OrderDispatchNotificationMailerPort = (*OrderDispatchNotificationMailer)(nil)

func NewOrderDispatchNotificationMailer(
	client OrderDispatchNotificationEmailClient,
	fromAddress string,
) *OrderDispatchNotificationMailer {
	return &OrderDispatchNotificationMailer{
		client:      client,
		fromAddress: strings.TrimSpace(fromAddress),
	}
}

func (m *OrderDispatchNotificationMailer) SendOrderDispatchNotification(
	ctx context.Context,
	message orderdispatchuc.OrderDispatchNotificationMailMessage,
) (orderdispatchuc.OrderDispatchNotificationMailSendResult, error) {
	if m == nil {
		return orderdispatchuc.OrderDispatchNotificationMailSendResult{
			Retryable: false,
		}, fmt.Errorf("order dispatch notification mailer is nil")
	}

	if m.client == nil {
		return orderdispatchuc.OrderDispatchNotificationMailSendResult{
				Retryable: false,
			}, fmt.Errorf(
				"order dispatch notification email client is not configured",
			)
	}

	fromAddress := strings.TrimSpace(m.fromAddress)
	if fromAddress == "" {
		return orderdispatchuc.OrderDispatchNotificationMailSendResult{
			Retryable: false,
		}, fmt.Errorf("from address is empty")
	}

	idempotencyKey := strings.TrimSpace(message.IdempotencyKey)
	if idempotencyKey == "" {
		return orderdispatchuc.OrderDispatchNotificationMailSendResult{
			Retryable: false,
		}, orderdom.ErrDispatchNotificationDeliveryIDRequired
	}

	toEmail := strings.ToLower(
		strings.TrimSpace(message.ToEmail),
	)
	if toEmail == "" {
		return orderdispatchuc.OrderDispatchNotificationMailSendResult{
			Retryable: false,
		}, fmt.Errorf("to email is empty")
	}

	orderID := strings.TrimSpace(message.OrderID)
	if orderID == "" {
		return orderdispatchuc.OrderDispatchNotificationMailSendResult{
			Retryable: false,
		}, orderdom.ErrDispatchNotificationOrderIDRequired
	}

	items, err := normalizeOrderDispatchNotificationMailItems(
		message.Items,
	)
	if err != nil {
		return orderdispatchuc.OrderDispatchNotificationMailSendResult{
			Retryable: false,
		}, err
	}

	body := buildOrderDispatchNotificationMailBody(
		orderID,
		items,
	)

	sendResult, err := m.client.SendWithResult(
		ctx,
		fromAddress,
		toEmail,
		orderDispatchNotificationSubject,
		body,
		idempotencyKey,
	)
	if err != nil {
		return orderdispatchuc.OrderDispatchNotificationMailSendResult{
				ProviderMessageID: sendResult.ProviderMessageID,
				Retryable:         sendResult.Retryable,
			}, fmt.Errorf(
				"send order dispatch notification failed: to=%s: %w",
				toEmail,
				err,
			)
	}

	return orderdispatchuc.OrderDispatchNotificationMailSendResult{
		ProviderMessageID: strings.TrimSpace(
			sendResult.ProviderMessageID,
		),
		Retryable: false,
	}, nil
}

func normalizeOrderDispatchNotificationMailItems(
	items []orderdispatchuc.OrderDispatchNotificationMailItem,
) ([]orderdispatchuc.OrderDispatchNotificationMailItem, error) {
	if len(items) == 0 {
		return nil, orderdom.ErrDispatchNotificationItemsRequired
	}

	normalized := make(
		[]orderdispatchuc.OrderDispatchNotificationMailItem,
		0,
		len(items),
	)

	for _, item := range items {
		productName := strings.TrimSpace(item.ProductName)
		if productName == "" || item.Qty <= 0 {
			return nil, orderdom.ErrDispatchNotificationItemInvalid
		}

		normalized = append(
			normalized,
			orderdispatchuc.OrderDispatchNotificationMailItem{
				ProductName: productName,
				Qty:         item.Qty,
			},
		)
	}

	return normalized, nil
}

func buildOrderDispatchNotificationMailBody(
	orderID string,
	items []orderdispatchuc.OrderDispatchNotificationMailItem,
) string {
	var builder strings.Builder

	builder.WriteString("ご注文の商品を発送しました。\n\n")
	builder.WriteString("注文ID:\n")
	builder.WriteString(orderID)
	builder.WriteString("\n\n")
	builder.WriteString("発送商品:\n")

	for _, item := range items {
		builder.WriteString(
			fmt.Sprintf(
				"・%s x %d\n",
				item.ProductName,
				item.Qty,
			),
		)
	}

	builder.WriteString("\n")
	builder.WriteString("配送状況についてはAMOLからご確認ください。\n\n")
	builder.WriteString("--\n")
	builder.WriteString("AMOL")

	return builder.String()
}
