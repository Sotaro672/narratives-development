// backend/internal/platform/di/mall/container_config.go
package mall

import (
	"os"
	"strings"
)

const (
	mallResendAPIKeyEnv                      = "RESEND_API_KEY"
	mallResendFromEnv                        = "RESEND_FROM"
	mallAutoCreateStripeTestPaymentMethodEnv = "MALL_AUTO_CREATE_STRIPE_TEST_PAYMENT_METHOD"
	mallFrontendBaseURLEnv                   = "MALL_FRONTEND_BASE_URL"
	mallAuthActionBaseURLEnv                 = "AUTH_ACTION_BASE_URL"
	mallStripeWebhookSecretEnv               = "STRIPE_WEBHOOK_SECRET"
)

type mallConfig struct {
	ResendAPIKey string
	ResendFrom   string

	AutoCreateStripeTestPaymentMethod bool
	FrontendBaseURL                   string

	AuthActionBaseURL   string
	StripeWebhookSecret string
}

func loadMallConfigFromEnv() mallConfig {
	return mallConfig{
		ResendAPIKey: strings.TrimSpace(
			os.Getenv(mallResendAPIKeyEnv),
		),
		ResendFrom: strings.TrimSpace(
			os.Getenv(mallResendFromEnv),
		),
		AutoCreateStripeTestPaymentMethod: envBool(
			mallAutoCreateStripeTestPaymentMethodEnv,
		),
		FrontendBaseURL: strings.TrimSpace(
			os.Getenv(mallFrontendBaseURLEnv),
		),
		AuthActionBaseURL: strings.TrimSpace(
			os.Getenv(mallAuthActionBaseURLEnv),
		),
		StripeWebhookSecret: strings.TrimSpace(
			os.Getenv(mallStripeWebhookSecretEnv),
		),
	}
}

func envBool(name string) bool {
	return strings.EqualFold(
		strings.TrimSpace(os.Getenv(name)),
		"true",
	)
}
