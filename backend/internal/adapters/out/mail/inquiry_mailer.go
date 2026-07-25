// backend/internal/adapters/out/mail/inquiry_mailer.go
package mail

import (
	"context"
	"fmt"
	"strings"

	inquirydom "narratives/internal/domain/inquiry"
)

type InquiryMailer struct {
	client *ResendClient
}

func NewInquiryMailer(client *ResendClient) *InquiryMailer {
	return &InquiryMailer{
		client: client,
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

	if from == "" {
		return fmt.Errorf("from address is empty")
	}
	if to == "" {
		return fmt.Errorf("to address is empty")
	}
	if inq.ID == "" {
		return fmt.Errorf("inquiry id is empty")
	}
	if inq.ProductID == "" {
		return fmt.Errorf("product id is empty")
	}
	if inq.AvatarID == "" {
		return fmt.Errorf("avatar id is empty")
	}

	subject := buildInquiryCreatedMailSubject(inq)
	body := buildInquiryCreatedMailBody(inq)

	return m.client.Send(ctx, from, to, subject, body)
}

func buildInquiryCreatedMailSubject(_ inquirydom.Inquiry) string {
	return "【AMOL】新しいお問い合わせが届きました"
}

func buildInquiryCreatedMailBody(inq inquirydom.Inquiry) string {
	var b strings.Builder

	b.WriteString("新しいお問い合わせが届きました。\n\n")

	b.WriteString("お問い合わせ内容\n")
	b.WriteString(fmt.Sprintf("問い合わせID: %s\n", inq.ID))
	b.WriteString(fmt.Sprintf("商品ID: %s\n", inq.ProductID))
	b.WriteString(fmt.Sprintf("アバターID: %s\n", inq.AvatarID))

	if inq.Subject != "" {
		b.WriteString(fmt.Sprintf("件名: %s\n", inq.Subject))
	}
	if inq.Content != "" {
		b.WriteString("本文:\n")
		b.WriteString(inq.Content)
		b.WriteString("\n")
	}

	if string(inq.Status) != "" {
		b.WriteString(fmt.Sprintf("ステータス: %s\n", string(inq.Status)))
	}
	if string(inq.InquiryType) != "" {
		b.WriteString(fmt.Sprintf("問い合わせ種別: %s\n", string(inq.InquiryType)))
	}

	b.WriteString(fmt.Sprintf("既読: %s\n", formatInquiryReadStatus(inq.IsRead)))

	if !inq.CreatedAt.IsZero() {
		b.WriteString(fmt.Sprintf("作成日時: %s\n", inq.CreatedAt.Format("2006-01-02 15:04:05 MST")))
	}
	if !inq.UpdatedAt.IsZero() {
		b.WriteString(fmt.Sprintf("更新日時: %s\n", inq.UpdatedAt.Format("2006-01-02 15:04:05 MST")))
	}
	if inq.UpdatedBy != nil && *inq.UpdatedBy != "" {
		b.WriteString(fmt.Sprintf("更新者: %s\n", *inq.UpdatedBy))
	}
	if inq.DeletedAt != nil && !inq.DeletedAt.IsZero() {
		b.WriteString(fmt.Sprintf("削除日時: %s\n", inq.DeletedAt.Format("2006-01-02 15:04:05 MST")))
	}
	if inq.DeletedBy != nil && *inq.DeletedBy != "" {
		b.WriteString(fmt.Sprintf("削除者: %s\n", *inq.DeletedBy))
	}

	b.WriteString("\n")

	writeInquiryImagesForMail(&b, inq.Images)

	b.WriteString("本メールは自動送信です。\n")

	return b.String()
}

func writeInquiryImagesForMail(
	b *strings.Builder,
	images []inquirydom.ImageFile,
) {
	if b == nil {
		return
	}

	if len(images) == 0 {
		return
	}

	b.WriteString("添付画像\n")

	for i, img := range images {
		b.WriteString(fmt.Sprintf("%d. 画像\n", i+1))

		if img.FileName != "" {
			b.WriteString(fmt.Sprintf("ファイル名: %s\n", img.FileName))
		}
		if img.FileURL != "" {
			b.WriteString(fmt.Sprintf("URL: %s\n", img.FileURL))
		}
		if img.ObjectPath != nil && *img.ObjectPath != "" {
			b.WriteString(fmt.Sprintf("ObjectPath: %s\n", *img.ObjectPath))
		}
		if img.FileSize > 0 {
			b.WriteString(fmt.Sprintf("ファイルサイズ: %d bytes\n", img.FileSize))
		}
		if img.MimeType != "" {
			b.WriteString(fmt.Sprintf("MIME Type: %s\n", img.MimeType))
		}
		if !img.CreatedAt.IsZero() {
			b.WriteString(fmt.Sprintf("作成日時: %s\n", img.CreatedAt.Format("2006-01-02 15:04:05 MST")))
		}
		if img.CreatedBy != "" {
			b.WriteString(fmt.Sprintf("作成者: %s\n", img.CreatedBy))
		}
		if img.UpdatedAt != nil && !img.UpdatedAt.IsZero() {
			b.WriteString(fmt.Sprintf("更新日時: %s\n", img.UpdatedAt.Format("2006-01-02 15:04:05 MST")))
		}
		if img.UpdatedBy != nil && *img.UpdatedBy != "" {
			b.WriteString(fmt.Sprintf("更新者: %s\n", *img.UpdatedBy))
		}
		if img.DeletedAt != nil && !img.DeletedAt.IsZero() {
			b.WriteString(fmt.Sprintf("削除日時: %s\n", img.DeletedAt.Format("2006-01-02 15:04:05 MST")))
		}
		if img.DeletedBy != nil && *img.DeletedBy != "" {
			b.WriteString(fmt.Sprintf("削除者: %s\n", *img.DeletedBy))
		}

		b.WriteString("\n")
	}
}

func formatInquiryReadStatus(isRead bool) string {
	if isRead {
		return "既読"
	}
	return "未読"
}
