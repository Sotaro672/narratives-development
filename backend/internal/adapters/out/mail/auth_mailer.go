// backend/internal/adapters/out/mail/auth_mailer.go
package mail

import (
	"context"
	"fmt"
)

type AuthMailerPort interface {
	SendVerificationEmail(
		ctx context.Context,
		toEmail string,
		verifyURL string,
	) error

	SendPasswordResetEmail(
		ctx context.Context,
		toEmail string,
		resetURL string,
	) error
}

// AuthEmailClientは、認証関連メール専用の送信契約です。
type AuthEmailClient interface {
	Send(
		ctx context.Context,
		from string,
		to string,
		subject string,
		body string,
	) error
}

type AuthMailer struct {
	client      AuthEmailClient
	fromAddress string
}

func NewAuthMailer(
	client AuthEmailClient,
	fromAddress string,
) *AuthMailer {
	return &AuthMailer{
		client:      client,
		fromAddress: fromAddress,
	}
}

func (m *AuthMailer) SendVerificationEmail(
	ctx context.Context,
	toEmail string,
	verifyURL string,
) error {
	if m == nil {
		return fmt.Errorf("auth mailer is nil")
	}

	if m.client == nil {
		return fmt.Errorf("auth email client is not configured")
	}

	fromAddress := m.fromAddress
	if fromAddress == "" {
		return fmt.Errorf("from address is empty")
	}

	if toEmail == "" {
		return fmt.Errorf("to email is empty")
	}

	if verifyURL == "" {
		return fmt.Errorf("verify URL is empty")
	}

	subject := "【AMOL】メールアドレス確認のお願い"

	body := fmt.Sprintf(
		`AMOLへのご登録ありがとうございます。

メールアドレスの確認を完了するには、下記のリンクを開いてください。

確認リンク:
%s

このメールに心当たりがない場合は、このメッセージは破棄してください。

--
AMOL`,
		verifyURL,
	)

	if err := m.client.Send(
		ctx,
		fromAddress,
		toEmail,
		subject,
		body,
	); err != nil {
		return fmt.Errorf(
			"send verification email failed: to=%s: %w",
			toEmail,
			err,
		)
	}

	return nil
}

func (m *AuthMailer) SendPasswordResetEmail(
	ctx context.Context,
	toEmail string,
	resetURL string,
) error {
	if m == nil {
		return fmt.Errorf("auth mailer is nil")
	}

	if m.client == nil {
		return fmt.Errorf("auth email client is not configured")
	}

	fromAddress := m.fromAddress
	if fromAddress == "" {
		return fmt.Errorf("from address is empty")
	}

	if toEmail == "" {
		return fmt.Errorf("to email is empty")
	}

	if resetURL == "" {
		return fmt.Errorf("reset URL is empty")
	}

	subject := "【AMOL】パスワード再設定のご案内"

	body := fmt.Sprintf(
		`AMOLのパスワード再設定リクエストを受け付けました。

下記のリンクを開き、新しいパスワードを設定してください。

再設定リンク:
%s

このメールに心当たりがない場合は、このメッセージは破棄してください。

--
AMOL`,
		resetURL,
	)

	if err := m.client.Send(
		ctx,
		fromAddress,
		toEmail,
		subject,
		body,
	); err != nil {
		return fmt.Errorf(
			"send password reset email failed: to=%s: %w",
			toEmail,
			err,
		)
	}

	return nil
}
