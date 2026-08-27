// backend/internal/platform/di/shared/settlement.go
package shared

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	stripeadapter "narratives/internal/adapters/out/stripe"
	uc "narratives/internal/application/usecase"
	settlementdom "narratives/internal/domain/settlement"
)

const (
	settlementPlatformFeeRateEnv = "SETTLEMENT_PLATFORM_FEE_RATE"
	settlementPlatformFeeBaseEnv = "SETTLEMENT_PLATFORM_FEE_BASE"
)

// SettlementDependencies contains the shared runtime dependencies required by
// SettlementUsecase, RefundUsecase, and ItemRefundUsecase.
//
// Stripe credentials and settlement fee policy are infrastructure concerns.
// Console and Mall DI should consume this already-constructed dependency set
// instead of loading secrets or environment variables independently.
type SettlementDependencies struct {
	Calculator settlementdom.PlatformFeeCalculator

	SettlementCalculator uc.SettlementCalculator

	StripeTransferGateway         uc.StripeSettlementTransferGateway
	StripeRefundGateway           uc.StripeRefundGateway
	StripeTransferReversalGateway uc.StripeTransferReversalGateway
}

// BuildSettlementDependencies builds the common settlement/refund
// infrastructure dependencies.
//
// It performs the following initialization once:
//
//   - loads stripe-secret-key from Secret Manager
//   - validates the Stripe secret key
//   - loads SETTLEMENT_PLATFORM_FEE_RATE
//   - loads SETTLEMENT_PLATFORM_FEE_BASE
//   - builds the platform fee calculator
//   - builds the settlement calculator
//   - builds the Stripe transfer gateway
//   - builds the Stripe refund gateway
//   - builds the Stripe transfer reversal gateway
func BuildSettlementDependencies(
	ctx context.Context,
	infra *Infra,
) (*SettlementDependencies, error) {
	if infra == nil {
		return nil, errors.New(
			"shared.settlement: infra is nil",
		)
	}

	stripeSecretKey, err :=
		infra.AccessSecretVersion(
			ctx,
			stripeSecretKeySecretID,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"shared.settlement: load Stripe secret: %w",
			err,
		)
	}

	stripeSecretKey = strings.TrimSpace(
		stripeSecretKey,
	)
	if stripeSecretKey == "" ||
		!strings.HasPrefix(
			stripeSecretKey,
			"sk_",
		) {
		return nil, errors.New(
			"shared.settlement: Stripe secret is invalid",
		)
	}

	platformFeeCalculator, err :=
		buildSettlementPlatformFeeCalculator()
	if err != nil {
		return nil, err
	}

	settlementCalculator :=
		settlementdom.NewCalculator(
			platformFeeCalculator,
		)
	if settlementCalculator == nil {
		return nil, errors.New(
			"shared.settlement: settlement calculator is nil",
		)
	}

	stripeTransferGateway :=
		stripeadapter.NewTransferGateway(
			stripeSecretKey,
		)
	if stripeTransferGateway == nil {
		return nil, errors.New(
			"shared.settlement: Stripe transfer gateway is nil",
		)
	}

	stripeRefundGateway :=
		stripeadapter.NewRefundGateway(
			stripeSecretKey,
		)
	if stripeRefundGateway == nil {
		return nil, errors.New(
			"shared.settlement: Stripe refund gateway is nil",
		)
	}

	stripeTransferReversalGateway :=
		stripeadapter.NewTransferReversalGateway(
			stripeSecretKey,
		)
	if stripeTransferReversalGateway == nil {
		return nil, errors.New(
			"shared.settlement: Stripe transfer reversal gateway is nil",
		)
	}

	return &SettlementDependencies{
		Calculator: platformFeeCalculator,

		SettlementCalculator: settlementCalculator,

		StripeTransferGateway:         stripeTransferGateway,
		StripeRefundGateway:           stripeRefundGateway,
		StripeTransferReversalGateway: stripeTransferReversalGateway,
	}, nil
}

func buildSettlementPlatformFeeCalculator() (
	settlementdom.PlatformFeeCalculator,
	error,
) {
	platformFeeRateText := strings.TrimSpace(
		os.Getenv(
			settlementPlatformFeeRateEnv,
		),
	)
	if platformFeeRateText == "" {
		return nil, fmt.Errorf(
			"shared.settlement: %s is empty",
			settlementPlatformFeeRateEnv,
		)
	}

	platformFeeRate, err :=
		strconv.Atoi(
			platformFeeRateText,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"shared.settlement: invalid %s: %w",
			settlementPlatformFeeRateEnv,
			err,
		)
	}

	platformFeeBaseText := strings.TrimSpace(
		os.Getenv(
			settlementPlatformFeeBaseEnv,
		),
	)
	if platformFeeBaseText == "" {
		return nil, fmt.Errorf(
			"shared.settlement: %s is empty",
			settlementPlatformFeeBaseEnv,
		)
	}

	platformFeeCalculator, err :=
		settlementdom.NewPercentagePlatformFeeCalculator(
			platformFeeRate,
			settlementdom.PlatformFeeBase(
				platformFeeBaseText,
			),
		)
	if err != nil {
		return nil, fmt.Errorf(
			"shared.settlement: build platform fee calculator: %w",
			err,
		)
	}

	if platformFeeCalculator == nil {
		return nil, errors.New(
			"shared.settlement: platform fee calculator is nil",
		)
	}

	return platformFeeCalculator, nil
}
