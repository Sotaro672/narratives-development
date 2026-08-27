// backend/internal/adapters/out/mail/resend_wire.go
package mail

import (
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
	apiKey := strings.TrimSpace(os.Getenv(envResendAPIKey))
	fromAddress := strings.TrimSpace(os.Getenv(envResendFrom))

	client := NewResendClient(apiKey)

	return NewInvitationMailer(
		client,
		fromAddress,
		companyRepo,
		brandResolver,
	)
}

// NewOrderDispatchNotificationMailerWithResendは、
// Resendを使ったOrderDispatchNotificationMailerを生成します。
//
// - RESEND_API_KEY: ResendのAPIキー
// - RESEND_FROM  : 送信元メールアドレス
func NewOrderDispatchNotificationMailerWithResend() *OrderDispatchNotificationMailer {
	apiKey := strings.TrimSpace(os.Getenv(envResendAPIKey))
	fromAddress := strings.TrimSpace(os.Getenv(envResendFrom))

	client := NewResendClient(apiKey)

	return NewOrderDispatchNotificationMailer(
		client,
		fromAddress,
	)
}

// NewRefundCompletionNotificationMailerWithResendは、
// Resendを使ったRefundCompletionNotificationMailerを生成します。
//
// - RESEND_API_KEY: ResendのAPIキー
// - RESEND_FROM  : 送信元メールアドレス
func NewRefundCompletionNotificationMailerWithResend() *RefundCompletionNotificationMailer {
	apiKey := strings.TrimSpace(os.Getenv(envResendAPIKey))
	fromAddress := strings.TrimSpace(os.Getenv(envResendFrom))

	client := NewResendClient(apiKey)

	return NewRefundCompletionNotificationMailer(
		client,
		fromAddress,
	)
}

// NewAuthMailerWithResendは、Resendを使ったAuthMailerを生成します。
//
// - RESEND_API_KEY: ResendのAPIキー
// - RESEND_FROM  : 送信元メールアドレス
//
// Firebase Authの標準メール送信ではなく、Backend側で生成した認証リンクを
// Resend経由で送信するために使用します。
func NewAuthMailerWithResend() *AuthMailer {
	apiKey := strings.TrimSpace(os.Getenv(envResendAPIKey))
	fromAddress := strings.TrimSpace(os.Getenv(envResendFrom))

	client := NewResendClient(apiKey)

	return NewAuthMailer(
		client,
		fromAddress,
	)
}
