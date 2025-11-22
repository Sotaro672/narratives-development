// backend/internal/adapters/out/mail/sendgrid_wire.go
package mail

import (
	"log"
	"os"
)

// 環境変数名（Cloud Run / ローカル共通）
const (
	envSendGridAPIKey = "SENDGRID_API_KEY"
	envSendGridFrom   = "SENDGRID_FROM"    // 例: no-reply@narratives.jp
	envConsoleBaseURL = "CONSOLE_BASE_URL" // 例: https://narratives.jp
)

// NewInvitationMailerWithSendGrid は、SendGrid を使った InvitationMailer を生成します。
//
// - SENDGRID_API_KEY : SendGrid の API キー
// - SENDGRID_FROM   : 送信元メールアドレス
// - CONSOLE_BASE_URL: https://narratives.jp
//
// companyResolver には CompanyID→会社名、brandResolver には BrandID→ブランド名を返す
// ドメインサービス（company.Service / brand.Service など）を渡してください。
func NewInvitationMailerWithSendGrid(
	companyResolver CompanyNameResolver,
	brandResolver BrandNameResolver,
) *InvitationMailer {
	apiKey := os.Getenv(envSendGridAPIKey)
	fromAddr := os.Getenv(envSendGridFrom)
	consoleBaseURL := os.Getenv(envConsoleBaseURL)

	if apiKey == "" {
		log.Printf("[mail] WARN: SENDGRID_API_KEY is empty. InvitationMailer will fail to send mail.")
	}
	if fromAddr == "" {
		log.Printf("[mail] WARN: SENDGRID_FROM is empty. InvitationMailer will fail to send mail.")
	}
	if consoleBaseURL == "" {
		consoleBaseURL = "https://narratives.jp"
		log.Printf("[mail] INFO: CONSOLE_BASE_URL is empty. default=%s", consoleBaseURL)
	}

	// SendGridClient を EmailClient として利用
	client := NewSendGridClient(apiKey)

	// InvitationMailer は InvitationMailerPort の実装で、
	// usecase.InvitationMailerPort とシグネチャ互換なのでそのまま渡せる。
	mailer := NewInvitationMailer(
		client,
		fromAddr,
		consoleBaseURL,
		companyResolver,
		brandResolver,
	)

	log.Printf("[mail] InvitationMailerWithSendGrid initialized. from=%s baseURL=%s",
		fromAddr, consoleBaseURL)

	return mailer
}
