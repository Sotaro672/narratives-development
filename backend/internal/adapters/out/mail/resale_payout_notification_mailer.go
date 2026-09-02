// backend/internal/adapters/out/mail/resale_payout_notification_mailer.go
package mail

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	applicationport "narratives/internal/application/port"
)

const resalePayoutNotificationSubject = "【AMOL】再販代金の振込が完了しました"

var resalePayoutNotificationJST = time.FixedZone("JST", 9*60*60)

type ResalePayoutNotificationEmailClient interface {
	SendWithResult(
		ctx context.Context,
		from string,
		to string,
		subject string,
		body string,
		idempotencyKey string,
	) (EmailSendResult, error)
}

type ResalePayoutNotificationMailer struct {
	client      ResalePayoutNotificationEmailClient
	fromAddress string
}

var _ applicationport.ResalePayoutNotificationMailerPort = (*ResalePayoutNotificationMailer)(nil)

func NewResalePayoutNotificationMailer(
	client ResalePayoutNotificationEmailClient,
	fromAddress string,
) *ResalePayoutNotificationMailer {
	return &ResalePayoutNotificationMailer{
		client:      client,
		fromAddress: strings.TrimSpace(fromAddress),
	}
}

func (m *ResalePayoutNotificationMailer) SendResalePayoutNotification(
	ctx context.Context,
	message applicationport.ResalePayoutNotificationMailMessage,
) (applicationport.ResalePayoutNotificationMailSendResult, error) {
	if m == nil {
		return applicationport.ResalePayoutNotificationMailSendResult{Retryable: false},
			fmt.Errorf("resale payout notification mailer is nil")
	}
	if m.client == nil {
		return applicationport.ResalePayoutNotificationMailSendResult{Retryable: false},
			fmt.Errorf("resale payout notification email client is not configured")
	}

	fromAddress := strings.TrimSpace(m.fromAddress)
	if fromAddress == "" {
		return applicationport.ResalePayoutNotificationMailSendResult{Retryable: false},
			fmt.Errorf("from address is empty")
	}

	idempotencyKey := strings.TrimSpace(message.IdempotencyKey)
	if idempotencyKey == "" {
		return applicationport.ResalePayoutNotificationMailSendResult{Retryable: false},
			fmt.Errorf("resale payout notification idempotency key is empty")
	}

	toEmail := strings.ToLower(strings.TrimSpace(message.ToEmail))
	if toEmail == "" {
		return applicationport.ResalePayoutNotificationMailSendResult{Retryable: false},
			fmt.Errorf("to email is empty")
	}

	bankPayoutID := strings.TrimSpace(message.BankPayoutID)
	if bankPayoutID == "" {
		return applicationport.ResalePayoutNotificationMailSendResult{Retryable: false},
			fmt.Errorf("bank payout id is empty")
	}

	salesReceivableID := strings.TrimSpace(message.SalesReceivableID)
	if salesReceivableID == "" {
		return applicationport.ResalePayoutNotificationMailSendResult{Retryable: false},
			fmt.Errorf("sales receivable id is empty")
	}

	orderID := strings.TrimSpace(message.OrderID)
	if orderID == "" {
		return applicationport.ResalePayoutNotificationMailSendResult{Retryable: false},
			fmt.Errorf("order id is empty")
	}

	resaleID := strings.TrimSpace(message.ResaleID)
	if resaleID == "" {
		return applicationport.ResalePayoutNotificationMailSendResult{Retryable: false},
			fmt.Errorf("resale id is empty")
	}

	if message.Amount <= 0 {
		return applicationport.ResalePayoutNotificationMailSendResult{Retryable: false},
			fmt.Errorf("resale payout amount is invalid")
	}
	if strings.TrimSpace(message.Currency) != "JPY" {
		return applicationport.ResalePayoutNotificationMailSendResult{Retryable: false},
			fmt.Errorf("resale payout currency is invalid")
	}

	bankName := strings.TrimSpace(message.BankName)
	if bankName == "" {
		return applicationport.ResalePayoutNotificationMailSendResult{Retryable: false},
			fmt.Errorf("bank name is empty")
	}

	branchName := strings.TrimSpace(message.BranchName)
	if branchName == "" {
		return applicationport.ResalePayoutNotificationMailSendResult{Retryable: false},
			fmt.Errorf("branch name is empty")
	}

	bankLast4 := strings.TrimSpace(message.BankLast4)
	if !isResalePayoutBankLast4(bankLast4) {
		return applicationport.ResalePayoutNotificationMailSendResult{Retryable: false},
			fmt.Errorf("bank last4 is invalid")
	}

	if message.PaidAt.IsZero() {
		return applicationport.ResalePayoutNotificationMailSendResult{Retryable: false},
			fmt.Errorf("paid at is invalid")
	}

	body := buildResalePayoutNotificationMailBody(
		orderID,
		resaleID,
		bankPayoutID,
		message.Amount,
		bankName,
		branchName,
		bankLast4,
		message.PaidAt,
	)

	sendResult, err := m.client.SendWithResult(
		ctx,
		fromAddress,
		toEmail,
		resalePayoutNotificationSubject,
		body,
		idempotencyKey,
	)
	if err != nil {
		return applicationport.ResalePayoutNotificationMailSendResult{
				ProviderMessageID: sendResult.ProviderMessageID,
				Retryable:         sendResult.Retryable,
			}, fmt.Errorf(
				"send resale payout notification failed: to=%s bankPayoutId=%s: %w",
				toEmail,
				bankPayoutID,
				err,
			)
	}

	return applicationport.ResalePayoutNotificationMailSendResult{
		ProviderMessageID: strings.TrimSpace(sendResult.ProviderMessageID),
		Retryable:         false,
	}, nil
}

func buildResalePayoutNotificationMailBody(
	orderID string,
	resaleID string,
	bankPayoutID string,
	amount int,
	bankName string,
	branchName string,
	bankLast4 string,
	paidAt time.Time,
) string {
	var builder strings.Builder

	builder.WriteString("再販商品の売上代金の振込が完了しました。\n\n")
	builder.WriteString("注文ID:\n")
	builder.WriteString(orderID)
	builder.WriteString("\n\n")
	builder.WriteString("再販ID:\n")
	builder.WriteString(resaleID)
	builder.WriteString("\n\n")
	builder.WriteString("振込額:\n")
	builder.WriteString(formatResalePayoutJPY(amount))
	builder.WriteString("\n\n")
	builder.WriteString("振込先:\n")
	builder.WriteString(bankName)
	builder.WriteString(" ")
	builder.WriteString(branchName)
	builder.WriteString(" 口座番号末尾 ")
	builder.WriteString(bankLast4)
	builder.WriteString("\n\n")
	builder.WriteString("振込完了日時:\n")
	builder.WriteString(paidAt.In(resalePayoutNotificationJST).Format("2006年01月02日 15:04"))
	builder.WriteString(" JST\n\n")
	builder.WriteString("振込ID:\n")
	builder.WriteString(bankPayoutID)
	builder.WriteString("\n\n")
	builder.WriteString("--\n")
	builder.WriteString("AMOL")

	return builder.String()
}

func formatResalePayoutJPY(amount int) string {
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

func isResalePayoutBankLast4(value string) bool {
	if len(value) != 4 {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
