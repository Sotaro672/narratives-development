// backend/internal/platform/di/mall/wiring_policy.go
package mall

import (
	"errors"

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
	errWiringNilInfra       = errors.New("di.mall: wiring policy infra is nil")
	errWiringNilPayment     = errors.New("di.mall: payment usecase is nil")
	errWiringNilOrderReader = errors.New("di.mall: payment flow order reader is nil")
)

// buildPaymentFlowUsecase wires PaymentFlowUsecase conditionally.
//
// PaymentFlowUsecase requires:
// - PaymentUsecase
// - OrderReaderForPaymentFlow
// - StripePaymentIntentGateway
//
// If the Stripe gateway is unavailable at this wiring layer,
// the payment flow is disabled.
func buildPaymentFlowUsecase(
	infra *shared.Infra,
	paymentUC *usecase.PaymentUsecase,
	orderReader usecase.OrderReaderForPaymentFlow,
) (*usecase.PaymentFlowUsecase, bool, error) {
	if infra == nil {
		return nil, false, errWiringNilInfra
	}
	if paymentUC == nil {
		return nil, false, errWiringNilPayment
	}
	if orderReader == nil {
		return nil, false, errWiringNilOrderReader
	}

	paymentIntentGateway := buildStripePaymentIntentGateway(infra)
	if paymentIntentGateway == nil {
		return nil, false, nil
	}

	return usecase.NewPaymentFlowUsecase(
		paymentUC,
		orderReader,
		paymentIntentGateway,
	), true, nil
}

func buildStripePaymentIntentGateway(
	infra *shared.Infra,
) usecase.StripePaymentIntentGateway {
	if infra == nil {
		return nil
	}

	return infra.PaymentMethodGateway
}

// buildScanVerifier wires ScanVerifier conditionally.
// Policy:
// - If a verifier exists, expose it directly as usecase.ScanVerifier.
// - Otherwise, return nil (feature disabled).
//
// NOTE:
// - PreviewQuery now implements VerifyMatch and can be used as usecase.ScanVerifier.
func buildScanVerifier(
	verifier usecase.ScanVerifier,
) usecase.ScanVerifier {
	if verifier == nil {
		return nil
	}

	return verifier
}
