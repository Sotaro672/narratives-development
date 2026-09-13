// backend/internal/adapters/out/mail/inquiry_mailer.go

package mail

import (
	"context"
	"fmt"
	"strings"

	avatardom "narratives/internal/domain/avatar"
	inquirydom "narratives/internal/domain/inquiry"
)

type InquiryAvatarReader interface {
	GetByID(
		ctx context.Context,
		id string,
	) (avatardom.Avatar, error)
}

type InquiryMailer struct {
	client       *ResendClient
	avatarReader InquiryAvatarReader
}

func NewInquiryMailer(
	client *ResendClient,
	avatarReader InquiryAvatarReader,
) *InquiryMailer {
	return &InquiryMailer{
		client:       client,
		avatarReader: avatarReader,
	}
}

// SendInquiryCreatedNotification sends an inquiry notification mail.
//
// from: sender email address
// to: recipient email address
// inq: created inquiry aggregate
func (m *InquiryMailer) SendInquiryCreatedNotification(
	ctx context.Context,
	from string,
	to string,
	inq inquirydom.Inquiry,
) error {
	if m == nil || m.client == nil {
		return fmt.Errorf("inquiry mailer is nil")
	}

	if strings.TrimSpace(from) == "" {
		return fmt.Errorf("from address is empty")
	}

	if strings.TrimSpace(to) == "" {
		return fmt.Errorf("to address is empty")
	}

	if strings.TrimSpace(inq.ID) == "" {
		return fmt.Errorf("inquiry id is empty")
	}

	if strings.TrimSpace(inq.AvatarID) == "" {
		return fmt.Errorf("avatar id is empty")
	}

	if err := validateInquiryCreatedMailIdentity(inq); err != nil {
		return err
	}

	avatarName := m.resolveAvatarName(
		ctx,
		inq.AvatarID,
	)

	subject := buildInquiryCreatedMailSubject(inq)
	body := buildInquiryCreatedMailBody(
		inq,
		avatarName,
	)

	return m.client.Send(
		ctx,
		from,
		to,
		subject,
		body,
	)
}

func (m *InquiryMailer) resolveAvatarName(
	ctx context.Context,
	avatarID string,
) string {
	if m == nil ||
		m.avatarReader == nil ||
		strings.TrimSpace(avatarID) == "" {
		return "-"
	}

	avatar, err := m.avatarReader.GetByID(
		ctx,
		avatarID,
	)
	if err != nil {
		return "-"
	}

	avatarName := strings.TrimSpace(
		avatar.AvatarName,
	)
	if avatarName == "" {
		return "-"
	}

	return avatarName
}

func validateInquiryCreatedMailIdentity(
	inq inquirydom.Inquiry,
) error {
	switch inq.InquiryType {
	case inquirydom.InquiryTypeProduct:
		if strings.TrimSpace(inq.ProductID) == "" {
			return fmt.Errorf("product id is empty")
		}

	case inquirydom.InquiryTypeReturnUnopened,
		inquirydom.InquiryTypeReturnOpened:
		if strings.TrimSpace(inq.OrderID) == "" {
			return fmt.Errorf("order id is empty")
		}

		if inq.OrderItemIndex == nil ||
			*inq.OrderItemIndex < 0 {
			return fmt.Errorf("order item index is invalid")
		}

	default:
		return fmt.Errorf(
			"unsupported inquiry type: %s",
			inq.InquiryType,
		)
	}

	return nil
}

func buildInquiryCreatedMailSubject(
	_ inquirydom.Inquiry,
) string {
	return "【AMOL】新しいお問い合わせが届きました"
}

func buildInquiryCreatedMailBody(
	inq inquirydom.Inquiry,
	avatarName string,
) string {
	var b strings.Builder

	b.WriteString(
		"新しいお問い合わせが届きました。\n\n",
	)

	b.WriteString(
		"お問い合わせ内容\n",
	)

	writeInquiryIdentityForMail(
		&b,
		inq,
	)

	b.WriteString(
		fmt.Sprintf(
			"アバター名: %s\n",
			emptyInquiryMailValue(avatarName),
		),
	)

	if strings.TrimSpace(inq.Subject) != "" {
		b.WriteString(
			fmt.Sprintf(
				"件名: %s\n",
				inq.Subject,
			),
		)
	}

	if strings.TrimSpace(inq.Content) != "" {
		b.WriteString("本文:\n")
		b.WriteString(inq.Content)
		b.WriteString("\n")
	}

	if string(inq.InquiryType) != "" {
		b.WriteString(
			fmt.Sprintf(
				"問い合わせ種別: %s\n",
				string(inq.InquiryType),
			),
		)
	}

	if !inq.CreatedAt.IsZero() {
		b.WriteString(
			fmt.Sprintf(
				"作成日時: %s\n",
				inq.CreatedAt.Format(
					"2006-01-02 15:04:05 MST",
				),
			),
		)
	}

	b.WriteString("\n")
	b.WriteString("本メールは自動送信です。\n")

	return b.String()
}

func writeInquiryIdentityForMail(
	b *strings.Builder,
	inq inquirydom.Inquiry,
) {
	if b == nil {
		return
	}

	switch inq.InquiryType {
	case inquirydom.InquiryTypeProduct:
		b.WriteString(
			fmt.Sprintf(
				"商品ID: %s\n",
				inq.ProductID,
			),
		)

	case inquirydom.InquiryTypeReturnUnopened:
		b.WriteString(
			fmt.Sprintf(
				"注文ID: %s\n",
				inq.OrderID,
			),
		)

		if inq.OrderItemIndex != nil {
			b.WriteString(
				fmt.Sprintf(
					"注文商品番号: %d\n",
					*inq.OrderItemIndex+1,
				),
			)
		}

	case inquirydom.InquiryTypeReturnOpened:
		b.WriteString(
			fmt.Sprintf(
				"注文ID: %s\n",
				inq.OrderID,
			),
		)

		if inq.OrderItemIndex != nil {
			b.WriteString(
				fmt.Sprintf(
					"注文商品番号: %d\n",
					*inq.OrderItemIndex+1,
				),
			)
		}

		if strings.TrimSpace(inq.ProductID) != "" {
			b.WriteString(
				fmt.Sprintf(
					"商品ID: %s\n",
					inq.ProductID,
				),
			)
		}
	}
}

func emptyInquiryMailValue(
	value string,
) string {
	value = strings.TrimSpace(value)

	if value == "" {
		return "-"
	}

	return value
}
