// backend/internal/adapters/out/mail/resend_wire.go
package mail

import (
	"log"
	"os"
)

// 環境変数名（Cloud Run / ローカル共通）
const (
	envResendAPIKey   = "RESEND_API_KEY"
	envResendFrom     = "RESEND_FROM"      // 例: no-reply@amol.jp
	envConsoleBaseURL = "CONSOLE_BASE_URL" // 例: https://amol.jp
)

// NewInvitationMailerWithResend は、Resend を使った InvitationMailer を生成します。
//
// - RESEND_API_KEY  : Resend の API キー
// - RESEND_FROM     : 送信元メールアドレス
// - CONSOLE_BASE_URL: https://amol.jp
//
// companyResolver には CompanyID→会社名、brandResolver には BrandID→ブランド名を返す
// ドメインサービス（company.Service / brand.Service など）を渡してください。
func NewInvitationMailerWithResend(
	companyResolver CompanyNameResolver,
	brandResolver BrandNameResolver,
) *InvitationMailer {
	apiKey := os.Getenv(envResendAPIKey)
	fromAddr := os.Getenv(envResendFrom)
	consoleBaseURL := os.Getenv(envConsoleBaseURL)

	if apiKey == "" {
		log.Printf("[mail] WARN: RESEND_API_KEY is empty. InvitationMailer will fail to send mail.")
	}
	if fromAddr == "" {
		log.Printf("[mail] WARN: RESEND_FROM is empty. InvitationMailer will fail to send mail.")
	}
	if consoleBaseURL == "" {
		consoleBaseURL = "https://amol.jp"
		log.Printf("[mail] INFO: CONSOLE_BASE_URL is empty. default=%s", consoleBaseURL)
	}

	// ResendClient を EmailClient として利用
	client := NewResendClient(apiKey)

	// InvitationMailer は InvitationMailerPort の実装で、
	// usecase.InvitationMailerPort とシグネチャ互換なのでそのまま渡せる。
	mailer := NewInvitationMailer(
		client,
		fromAddr,
		consoleBaseURL,
		companyResolver,
		brandResolver,
	)

	log.Printf("[mail] InvitationMailerWithResend initialized. from=%s baseURL=%s",
		fromAddr, consoleBaseURL)

	return mailer
}
