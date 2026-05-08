// backend/internal/platform/di/mall/wiring_policy.go
package mall

import (
	"errors"

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
//   and builds lightweight optional deps.

var (
	errWiringNilInfra   = errors.New("di.mall: wiring policy infra is nil")
	errWiringNilPayment = errors.New("di.mall: payment usecase is nil")
)

// buildPaymentFlowUsecase wires PaymentFlowUsecase conditionally.
//
// Legacy webhook trigger has been removed.
// PaymentFlowUsecase now requires a StripePaymentIntentGateway.
// If the gateway is not available at this wiring layer, the payment flow is disabled.
func buildPaymentFlowUsecase(
	infra *shared.Infra,
	paymentUC *usecase.PaymentUsecase,
) (*usecase.PaymentFlowUsecase, bool, error) {
	if infra == nil {
		return nil, false, errWiringNilInfra
	}
	if paymentUC == nil {
		return nil, false, errWiringNilPayment
	}

	paymentIntentGateway := buildStripePaymentIntentGateway(infra)
	if paymentIntentGateway == nil {
		return nil, false, nil
	}

	return usecase.NewPaymentFlowUsecase(paymentUC, paymentIntentGateway), true, nil
}

func buildStripePaymentIntentGateway(infra *shared.Infra) usecase.StripePaymentIntentGateway {
	if infra == nil {
		return nil
	}

	return infra.PaymentMethodGateway
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

// buildWalletSecretProvider wires WalletSecretProvider / AvatarSecretProvider conditionally.
// Policy:
// - Requires SecretManager client and ProjectID.
// - BrandWalletSecretPrefix is required for brand transfer.
// - AvatarWalletSecretPrefix is optional for boot itself, but required to enable share transfer.
// - If SecretManager is missing, return nil (feature disabled).
//
// NOTE:
// - TransferUsecase construction is intentionally NOT done here.
// - This provider is returned to container.go, where usecases are built when enabled.
func buildWalletSecretProvider(infra *shared.Infra) (usecase.WalletSecretProvider, error) {
	if infra == nil {
		return nil, errWiringNilInfra
	}

	if infra.SecretManager == nil {
		return nil, nil
	}

	projectID := infra.ProjectID
	if projectID == "" {
		return nil, errors.New("di.mall: ProjectID is empty (cannot wire WalletSecretProvider)")
	}

	brandPrefix := infra.BrandWalletSecretPrefix
	if brandPrefix == "" {
		return nil, errors.New("di.mall: BrandWalletSecretPrefix is empty (cannot wire WalletSecretProvider)")
	}

	// Share transfer で avatar signer を使うための prefix。
	// shared.Infra 側に AvatarWalletSecretPrefix が未追加の場合でも、
	// brand transfer だけは壊さないよう空文字を許容して保持する。
	avatarPrefix := infra.AvatarWalletSecretPrefix

	return &walletSecretProviderSM{
		sm:                 infra.SecretManager,
		projectID:          projectID,
		brandSecretPrefix:  brandPrefix,
		avatarSecretPrefix: avatarPrefix,
		version:            "latest",
	}, nil
}
