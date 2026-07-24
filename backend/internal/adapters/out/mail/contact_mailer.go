// backend/internal/adapters/out/mail/contact_mailer.go
package mail

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

const (
	envResendContactAdminTo = "RESEND_CONTACT_ADMIN_TO"
)

type ContactMailerWithResend struct {
	client   *ResendClient
	fromAddr string
	adminTo  string
}

func NewContactMailerWithResend() *ContactMailerWithResend {
	apiKey := strings.TrimSpace(
		os.Getenv(envResendAPIKey),
	)
	fromAddr := strings.TrimSpace(
		os.Getenv(envResendFrom),
	)
	adminTo := strings.TrimSpace(
		os.Getenv(envResendContactAdminTo),
	)

	if adminTo == "" {
		adminTo = fromAddr
	}

	if apiKey == "" {
		log.Printf(
			"[mail] WARN: RESEND_API_KEY is empty. ContactMailerWithResend will fail to send mail.",
		)
	}

	if fromAddr == "" {
		log.Printf(
			"[mail] WARN: RESEND_FROM is empty. ContactMailerWithResend will fail to send mail.",
		)
	}

	if adminTo == "" {
		log.Printf(
			"[mail] WARN: RESEND_CONTACT_ADMIN_TO and RESEND_FROM are empty. Admin notification mail will fail to send.",
		)
	}

	log.Printf(
		"[mail] ContactMailerWithResend initialized. from=%s adminTo=%s",
		fromAddr,
		adminTo,
	)

	return &ContactMailerWithResend{
		client:   NewResendClient(apiKey),
		fromAddr: fromAddr,
		adminTo:  adminTo,
	}
}

func (m *ContactMailerWithResend) SendContactReceipt(
	ctx context.Context,
	name string,
	email string,
	company string,
	message string,
	source string,
) error {
	_ = source

	if err := m.validateCommon(); err != nil {
		return err
	}

	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf(
			"contact receipt mail: recipient email is empty",
		)
	}

	subject := "【AMOL】お問い合わせを受け付けました"

	plain := buildContactReceiptPlain(
		name,
		email,
		company,
		message,
	)
	html := buildContactReceiptHTML(
		name,
		email,
		company,
		message,
	)

	return m.sendMail(
		ctx,
		email,
		subject,
		plain,
		html,
		"contact receipt",
	)
}

func (m *ContactMailerWithResend) SendContactAdminNotification(
	ctx context.Context,
	id string,
	name string,
	email string,
	company string,
	message string,
	source string,
	createdAt time.Time,
) error {
	_ = id
	_ = source
	_ = createdAt

	if err := m.validateCommon(); err != nil {
		return err
	}

	adminTo := strings.TrimSpace(m.adminTo)
	if adminTo == "" {
		return fmt.Errorf(
			"contact admin notification mail: admin recipient is empty",
		)
	}

	subject := "【AMOL】新しいお問い合わせを受信しました"

	plain := buildContactAdminPlain(
		name,
		email,
		company,
		message,
	)
	html := buildContactAdminHTML(
		name,
		email,
		company,
		message,
	)

	return m.sendMail(
		ctx,
		adminTo,
		subject,
		plain,
		html,
		"contact admin notification",
	)
}

func (m *ContactMailerWithResend) validateCommon() error {
	if m == nil {
		return fmt.Errorf(
			"contact mailer is nil",
		)
	}

	if m.client == nil {
		return fmt.Errorf(
			"resend client is nil",
		)
	}

	if strings.TrimSpace(m.fromAddr) == "" {
		return fmt.Errorf(
			"RESEND_FROM is empty",
		)
	}

	return nil
}

func (m *ContactMailerWithResend) sendMail(
	ctx context.Context,
	to string,
	subject string,
	plain string,
	html string,
	logLabel string,
) error {
	fromAddr := strings.TrimSpace(m.fromAddr)
	to = strings.TrimSpace(to)

	if err := m.client.Send(
		ctx,
		fromAddr,
		to,
		subject,
		plain,
	); err != nil {
		log.Printf(
			"[mail] %s resend send error: to=%s err=%v",
			logLabel,
			to,
			err,
		)

		return err
	}

	log.Printf(
		"[mail] %s resend send success: to=%s subject=%s",
		logLabel,
		to,
		subject,
	)

	_ = html

	return nil
}

func buildContactReceiptPlain(
	name string,
	email string,
	company string,
	message string,
) string {
	return fmt.Sprintf(
		`%s 様

お問い合わせありがとうございます。
以下の内容で受け付けました。

【お名前】
%s

【メールアドレス】
%s

【会社名】
%s

【お問い合わせ内容】
%s

内容を確認のうえ、担当よりご連絡いたします。
このメールは送信専用です。返信いただいても確認できない場合があります。

AMOL`,
		emptyFallback(name, "お客様"),
		emptyFallback(name, "-"),
		emptyFallback(email, "-"),
		emptyFallback(company, "-"),
		emptyFallback(message, "-"),
	)
}

func buildContactReceiptHTML(
	name string,
	email string,
	company string,
	message string,
) string {
	return fmt.Sprintf(
		`
<html>
  <body>
    <p>%s 様</p>
    <p>お問い合わせありがとうございます。<br>以下の内容で受け付けました。</p>

    <p><strong>お名前</strong><br>%s</p>
    <p><strong>メールアドレス</strong><br>%s</p>
    <p><strong>会社名</strong><br>%s</p>
    <p><strong>お問い合わせ内容</strong><br>%s</p>

    <p>内容を確認のうえ、担当よりご連絡いたします。<br>
    このメールは送信専用です。返信いただいても確認できない場合があります。</p>

    <p>AMOL</p>
  </body>
</html>`,
		escapeHTML(
			emptyFallback(name, "お客様"),
		),
		escapeHTML(
			emptyFallback(name, "-"),
		),
		escapeHTML(
			emptyFallback(email, "-"),
		),
		escapeHTML(
			emptyFallback(company, "-"),
		),
		nl2br(
			escapeHTML(
				emptyFallback(message, "-"),
			),
		),
	)
}

func buildContactAdminPlain(
	name string,
	email string,
	company string,
	message string,
) string {
	return fmt.Sprintf(
		`新しいお問い合わせを受信しました。

【名前】
%s

【メールアドレス】
%s

【会社名】
%s

【お問い合わせ内容】
%s`,
		emptyFallback(name, "-"),
		emptyFallback(email, "-"),
		emptyFallback(company, "-"),
		emptyFallback(message, "-"),
	)
}

func buildContactAdminHTML(
	name string,
	email string,
	company string,
	message string,
) string {
	return fmt.Sprintf(
		`
<html>
  <body>
    <p>新しいお問い合わせを受信しました。</p>

    <p><strong>名前</strong><br>%s</p>
    <p><strong>メールアドレス</strong><br>%s</p>
    <p><strong>会社名</strong><br>%s</p>
    <p><strong>お問い合わせ内容</strong><br>%s</p>
  </body>
</html>`,
		escapeHTML(
			emptyFallback(name, "-"),
		),
		escapeHTML(
			emptyFallback(email, "-"),
		),
		escapeHTML(
			emptyFallback(company, "-"),
		),
		nl2br(
			escapeHTML(
				emptyFallback(message, "-"),
			),
		),
	)
}

func emptyFallback(
	value string,
	fallback string,
) string {
	value = strings.TrimSpace(value)

	if value == "" {
		return fallback
	}

	return value
}

func escapeHTML(
	value string,
) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&#39;",
	)

	return replacer.Replace(value)
}

func nl2br(
	value string,
) string {
	return strings.ReplaceAll(
		value,
		"\n",
		"<br>",
	)
}
