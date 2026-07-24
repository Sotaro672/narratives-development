// backend/internal/adapters/out/mail/resend_wire.go
package mail

import (
	"log"
	"os"
	"strings"

	companydom "narratives/internal/domain/company"
)

// 環境変数名（Cloud Run / ローカル共通）
const (
	envResendAPIKey = "RESEND_API_KEY"
	envResendFrom   = "RESEND_FROM"
)

// NewInvitationMailerWithResendは、Resendを使ったInvitationMailerを生成します。
//
// - RESEND_API_KEY: ResendのAPIキー
// - RESEND_FROM  : 送信元メールアドレス
//
// companyRepoにはCompanyIDからCompanyを取得するcompany.Repository、
// brandResolverにはBrandIDからBrandを取得するRepositoryまたはResolverを渡します。
func NewInvitationMailerWithResend(
	companyRepo companydom.Repository,
	brandResolver BrandNameResolver,
) *InvitationMailer {
	apiKey := strings.TrimSpace(
		os.Getenv(envResendAPIKey),
	)
	fromAddress := strings.TrimSpace(
		os.Getenv(envResendFrom),
	)

	if apiKey == "" {
		log.Printf(
			"[mail] WARN: RESEND_API_KEY is empty. InvitationMailer will fail to send mail.",
		)
	}

	if fromAddress == "" {
		log.Printf(
			"[mail] WARN: RESEND_FROM is empty. InvitationMailer will fail to send mail.",
		)
	}

	client := NewResendClient(apiKey)

	mailer := NewInvitationMailer(
		client,
		fromAddress,
		companyRepo,
		brandResolver,
	)

	log.Printf(
		"[mail] InvitationMailerWithResend initialized. from=%s",
		fromAddress,
	)

	return mailer
}

// NewAuthMailerWithResendは、Resendを使ったAuthMailerを生成します。
//
// - RESEND_API_KEY: ResendのAPIキー
// - RESEND_FROM  : 送信元メールアドレス
//
// Firebase Authの標準メール送信ではなく、Backend側で生成した認証リンクを
// Resend経由で送信するために使用します。
func NewAuthMailerWithResend() *AuthMailer {
	apiKey := strings.TrimSpace(
		os.Getenv(envResendAPIKey),
	)
	fromAddress := strings.TrimSpace(
		os.Getenv(envResendFrom),
	)

	if apiKey == "" {
		log.Printf(
			"[mail] WARN: RESEND_API_KEY is empty. AuthMailer will fail to send mail.",
		)
	}

	if fromAddress == "" {
		log.Printf(
			"[mail] WARN: RESEND_FROM is empty. AuthMailer will fail to send mail.",
		)
	}

	client := NewResendClient(apiKey)

	mailer := NewAuthMailer(
		client,
		fromAddress,
	)

	log.Printf(
		"[mail] AuthMailerWithResend initialized. from=%s",
		fromAddress,
	)

	return mailer
}
