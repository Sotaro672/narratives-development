// backend/internal/platform/di/mall/wiring_policy.go
package mall

import (
	"errors"
	"strings"

	httpout "narratives/internal/adapters/out/http"
	mallquery "narratives/internal/application/query/mall"
	usecase "narratives/internal/application/usecase"
	shared "narratives/internal/platform/di/shared"
)

// wiring_policy.go defines wiring-time policies (conditional dependency assembly)
// for the mall DI container.
//
// Policy scope (IMPORTANT):
// - This file MUST NOT construct TransferUsecase.
// - This file MUST NOT call WithInventoryRepo.
// - This file only decides whether optional features are "enabled" based on runtime settings
//   and builds lightweight optional deps (e.g., webhook trigger client, secret provider).

var (
	errWiringNilInfra   = errors.New("di.mall: wiring policy infra is nil")
	errWiringNilPayment = errors.New("di.mall: payment usecase is nil")
)

// buildPaymentFlowUsecase wires PaymentFlowUsecase conditionally.
// Policy:
// - If SelfBaseURL is configured, enable webhook trigger client.
// - Otherwise, build PaymentFlowUsecase with nil trigger (disabled).
func buildPaymentFlowUsecase(infra *shared.Infra, paymentUC *usecase.PaymentUsecase) (*usecase.PaymentFlowUsecase, bool, error) {
	if infra == nil {
		return nil, false, errWiringNilInfra
	}
	if paymentUC == nil {
		return nil, false, errWiringNilPayment
	}

	base := strings.TrimSpace(infra.SelfBaseURL)
	if base == "" {
		return usecase.NewPaymentFlowUsecase(paymentUC, nil), false, nil
	}

	trigger := httpout.NewStripeWebhookClient(base)
	return usecase.NewPaymentFlowUsecase(paymentUC, trigger), true, nil
}

// buildScanVerifier wires ScanVerifier conditionally.
// Policy:
// - If OrderScanVerifyQ exists, expose it as usecase.ScanVerifier via adapter.
// - Otherwise, return nil (feature disabled).
func buildScanVerifier(orderScanVerifyQ *mallquery.OrderScanVerifyQuery) usecase.ScanVerifier {
	if orderScanVerifyQ == nil {
		return nil
	}
	return mallquery.NewScanVerifierAdapter(orderScanVerifyQ)
}

// buildWalletSecretProvider wires WalletSecretProvider conditionally.
// Policy:
// - Requires SecretManager client and ProjectID.
// - BrandWalletSecretPrefix must be present (expected to be normalized+validated in shared.Infra).
// - If any prerequisite is missing, return nil (feature disabled).
//
// NOTE:
// - TransferUsecase construction is intentionally NOT done here.
// - This provider is returned to container.go, where TransferUsecase is built when enabled.
func buildWalletSecretProvider(infra *shared.Infra) (usecase.WalletSecretProvider, error) {
	if infra == nil {
		return nil, errWiringNilInfra
	}

	if infra.SecretManager == nil {
		return nil, nil
	}

	projectID := strings.TrimSpace(infra.ProjectID)
	if projectID == "" {
		// ProjectID should be required by shared.Infra, but keep as defensive check.
		return nil, errors.New("di.mall: ProjectID is empty (cannot wire WalletSecretProvider)")
	}

	prefix := strings.TrimSpace(infra.BrandWalletSecretPrefix)
	if prefix == "" {
		// Should be normalized+validated in shared.Infra; treat as wiring error.
		return nil, errors.New("di.mall: BrandWalletSecretPrefix is empty (cannot wire WalletSecretProvider)")
	}

	return &brandWalletSecretProviderSM{
		sm:           infra.SecretManager,
		projectID:    projectID,
		secretPrefix: prefix,
		version:      "latest",
	}, nil
}
