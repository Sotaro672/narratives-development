// backend/internal/adapters/out/mail/refund_completion_mailer.go
package mail

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	refundnotificationuc "narratives/internal/application/usecase"
	refunddom "narratives/internal/domain/refund"
)

const refundCompletionNotificationSubject = "【AMOL】返品の返金が完了しました"

type RefundCompletionNotificationEmailClient interface {
	SendWithResult(
		ctx context.Context,
		from string,
		to string,
		subject string,
		body string,
		idempotencyKey string,
	) (EmailSendResult, error)
}

type RefundCompletionNotificationMailer struct {
	client      RefundCompletionNotificationEmailClient
	fromAddress string
}

var _ refundnotificationuc.RefundCompletionNotificationMailerPort = (*RefundCompletionNotificationMailer)(nil)

func NewRefundCompletionNotificationMailer(
	client RefundCompletionNotificationEmailClient,
	fromAddress string,
) *RefundCompletionNotificationMailer {
	return &RefundCompletionNotificationMailer{
		client:      client,
		fromAddress: strings.TrimSpace(fromAddress),
	}
}

func (m *RefundCompletionNotificationMailer) SendRefundCompletionNotification(
	ctx context.Context,
	message refundnotificationuc.RefundCompletionNotificationMailMessage,
) (refundnotificationuc.RefundCompletionNotificationMailSendResult, error) {
	if m == nil {
		return refundnotificationuc.RefundCompletionNotificationMailSendResult{
			Retryable: false,
		}, fmt.Errorf("refund completion notification mailer is nil")
	}

	if m.client == nil {
		return refundnotificationuc.RefundCompletionNotificationMailSendResult{
			Retryable: false,
		}, fmt.Errorf("refund completion notification email client is not configured")
	}

	fromAddress := strings.TrimSpace(m.fromAddress)
	if fromAddress == "" {
		return refundnotificationuc.RefundCompletionNotificationMailSendResult{
			Retryable: false,
		}, fmt.Errorf("from address is empty")
	}

	idempotencyKey := strings.TrimSpace(message.IdempotencyKey)
	if idempotencyKey == "" {
		return refundnotificationuc.RefundCompletionNotificationMailSendResult{
			Retryable: false,
		}, refunddom.ErrCompletionNotificationDeliveryIDRequired
	}

	toEmail := strings.ToLower(strings.TrimSpace(message.ToEmail))
	if toEmail == "" {
		return refundnotificationuc.RefundCompletionNotificationMailSendResult{
			Retryable: false,
		}, fmt.Errorf("to email is empty")
	}

	paymentID := strings.TrimSpace(message.PaymentID)
	if paymentID == "" {
		return refundnotificationuc.RefundCompletionNotificationMailSendResult{
			Retryable: false,
		}, refunddom.ErrCompletionNotificationPaymentIDRequired
	}

	orderID := strings.TrimSpace(message.OrderID)
	if orderID == "" {
		return refundnotificationuc.RefundCompletionNotificationMailSendResult{
			Retryable: false,
		}, refunddom.ErrCompletionNotificationOrderIDRequired
	}

	stripeRefundID := strings.TrimSpace(message.StripeRefundID)
	if stripeRefundID == "" {
		return refundnotificationuc.RefundCompletionNotificationMailSendResult{
			Retryable: false,
		}, refunddom.ErrCompletionNotificationStripeRefundIDRequired
	}

	if !strings.HasPrefix(stripeRefundID, "re_") {
		return refundnotificationuc.RefundCompletionNotificationMailSendResult{
			Retryable: false,
		}, refunddom.ErrCompletionNotificationStripeRefundIDInvalid
	}

	if message.RefundedAmount <= 0 {
		return refundnotificationuc.RefundCompletionNotificationMailSendResult{
			Retryable: false,
		}, refunddom.ErrCompletionNotificationRefundedAmountInvalid
	}

	body := buildRefundCompletionNotificationMailBody(
		orderID,
		stripeRefundID,
		message.RefundedAmount,
	)

	sendResult, err := m.client.SendWithResult(
		ctx,
		fromAddress,
		toEmail,
		refundCompletionNotificationSubject,
		body,
		idempotencyKey,
	)
	if err != nil {
		return refundnotificationuc.RefundCompletionNotificationMailSendResult{
				ProviderMessageID: sendResult.ProviderMessageID,
				Retryable:         sendResult.Retryable,
			}, fmt.Errorf(
				"send refund completion notification failed: to=%s paymentId=%s: %w",
				toEmail,
				paymentID,
				err,
			)
	}

	return refundnotificationuc.RefundCompletionNotificationMailSendResult{
		ProviderMessageID: strings.TrimSpace(sendResult.ProviderMessageID),
		Retryable:         false,
	}, nil
}

func buildRefundCompletionNotificationMailBody(
	orderID string,
	stripeRefundID string,
	refundedAmount int,
) string {
	var builder strings.Builder

	builder.WriteString("返品の返金が完了しました。\n\n")
	builder.WriteString("注文ID:\n")
	builder.WriteString(orderID)
	builder.WriteString("\n\n")
	builder.WriteString("返金額:\n")
	builder.WriteString(formatRefundCompletionJPY(refundedAmount))
	builder.WriteString("\n\n")
	builder.WriteString("返金処理ID:\n")
	builder.WriteString(stripeRefundID)
	builder.WriteString("\n\n")
	builder.WriteString("ご利用の決済方法への返金額の反映時期は、カード会社等によって異なる場合があります。\n\n")
	builder.WriteString("--\n")
	builder.WriteString("AMOL")

	return builder.String()
}

func formatRefundCompletionJPY(amount int) string {
	if amount == 0 {
		return "¥0"
	}

	negative := amount < 0
	if negative {
		amount = -amount
	}

	raw := strconv.Itoa(amount)
	var builder strings.Builder

	if negative {
		builder.WriteString("-")
	}

	builder.WriteString("¥")

	firstGroupLength := len(raw) % 3
	if firstGroupLength == 0 {
		firstGroupLength = 3
	}

	builder.WriteString(raw[:firstGroupLength])

	for index := firstGroupLength; index < len(raw); index += 3 {
		builder.WriteString(",")
		builder.WriteString(raw[index : index+3])
	}

	return builder.String()
}
