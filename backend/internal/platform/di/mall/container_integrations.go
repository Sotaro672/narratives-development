// backend/internal/platform/di/mall/container_integrations.go
package mall

import (
	"context"
	"errors"
	"fmt"

	cloudtasksadp "narratives/internal/adapters/out/cloudtasks"
	outfirebase "narratives/internal/adapters/out/firebase"
	mailadp "narratives/internal/adapters/out/mail"
	outsolana "narratives/internal/adapters/out/solana"
	stripeadapter "narratives/internal/adapters/out/stripe"

	transportationdom "narratives/internal/domain/transportation"

	solana "narratives/internal/infra/solana"

	shared "narratives/internal/platform/di/shared"
)

type mallIntegrations struct {
	authUserReader *outfirebase.AuthUserReader

	paymentMethodGateway *stripeadapter.PaymentMethodGateway
	accountGateway       *stripeadapter.AccountGateway

	settlementDependencies *shared.SettlementDependencies

	resaleImageStorage *outfirebase.ResaleImageStorage

	resendClient                       *mailadp.ResendClient
	orderMailer                        *mailadp.OrderMailer
	orderCancellationMailer            *mailadp.OrderCancellationMailer
	inquiryMailer                      *mailadp.InquiryMailer
	refundCompletionNotificationMailer *mailadp.RefundCompletionNotificationMailer

	refundCompletionNotificationQueue *cloudtasksadp.RefundCompletionNotificationQueue

	avatarWalletService *solana.AvatarWalletService
	transportationSvc   *transportationdom.Service

	onchainWalletReader   *solana.OnchainWalletReaderImpl
	tokenTransferReader   *solana.TokenTransferReaderSolana
	previewTransferReader *outsolana.PreviewTransferReader
	tokenTransferExecutor *solana.TokenTransferExecutorSolana
}

func buildMallIntegrations(
	ctx context.Context,
	infra *shared.Infra,
	cfg mallConfig,
	r *mallRepositories,
) (*mallIntegrations, error) {
	if infra == nil {
		return nil, errors.New("di.mall: shared infra is nil")
	}
	if infra.FirebaseAuth == nil {
		return nil, errors.New("di.mall: firebase auth client is nil")
	}
	if r == nil {
		return nil, errors.New("di.mall: repositories are nil")
	}
	if r.paymentMethodRepo == nil {
		return nil, errors.New("di.mall: payment method repository is nil")
	}
	if r.transportationRepo == nil {
		return nil, errors.New("di.mall: transportation repository is nil")
	}

	var customerStore stripeadapter.PaymentMethodCustomerStore = r.paymentMethodRepo
	if err := infra.RegisterPaymentMethodGatewayFromSecret(ctx, customerStore); err != nil {
		return nil, fmt.Errorf(
			"di.mall: register Stripe payment method gateway: %w",
			err,
		)
	}
	if infra.PaymentMethodGateway == nil {
		return nil, errors.New("di.mall: Stripe payment method gateway is nil after registration")
	}

	if infra.AccountGateway == nil {
		if err := infra.RegisterAccountGatewayFromSecret(ctx); err != nil {
			return nil, fmt.Errorf(
				"di.mall: register Stripe payout account gateway: %w",
				err,
			)
		}
	}
	if infra.AccountGateway == nil {
		return nil, errors.New("di.mall: Stripe payout account gateway is nil after registration")
	}

	settlementDependencies, err := shared.BuildSettlementDependencies(ctx, infra)
	if err != nil {
		return nil, fmt.Errorf(
			"di.mall: build settlement dependencies: %w",
			err,
		)
	}
	if settlementDependencies == nil {
		return nil, errors.New("di.mall: settlement dependencies are nil")
	}

	resaleImageStorage, err := outfirebase.NewResaleImageStorageFromEnv(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"di.mall: build resale image storage: %w",
			err,
		)
	}

	refundCompletionNotificationQueue, err :=
		cloudtasksadp.NewRefundCompletionNotificationQueueFromEnv(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"di.mall: build refund completion notification queue: %w",
			err,
		)
	}

	authUserReader := outfirebase.NewAuthUserReader(
		infra.FirebaseAuth,
	)

	resendClient := mailadp.NewResendClient(
		cfg.ResendAPIKey,
	)

	orderMailer := mailadp.NewOrderMailer(
		resendClient,
		r.modelRepoFS,
		r.inventoryRepo,
		r.productBlueprintRepoFS,
		r.tokenBlueprintRepo,
		r.brandRepo,
		r.companyRepo,
	)

	orderCancellationMailer := mailadp.NewOrderCancellationMailer(
		resendClient,
		cfg.ResendFrom,
	)

	inquiryMailer := mailadp.NewInquiryMailer(
		resendClient,
	)

	refundCompletionNotificationMailer :=
		mailadp.NewRefundCompletionNotificationMailer(
			resendClient,
			cfg.ResendFrom,
		)

	avatarWalletService := solana.NewAvatarWalletService(
		infra.ProjectID,
	)

	transportationSvc := transportationdom.NewService(
		r.transportationRepo,
	)

	onchainWalletReader := solana.NewOnchainWalletReaderDevnet()

	tokenTransferReader := solana.NewTokenTransferReaderSolana("")

	previewTransferReader := outsolana.NewPreviewTransferReader(
		tokenTransferReader,
	)

	tokenTransferExecutor := solana.NewTokenTransferExecutorSolana("")

	return &mallIntegrations{
		authUserReader: authUserReader,

		paymentMethodGateway: infra.PaymentMethodGateway,
		accountGateway:       infra.AccountGateway,

		settlementDependencies: settlementDependencies,

		resaleImageStorage: resaleImageStorage,

		resendClient:                       resendClient,
		orderMailer:                        orderMailer,
		orderCancellationMailer:            orderCancellationMailer,
		inquiryMailer:                      inquiryMailer,
		refundCompletionNotificationMailer: refundCompletionNotificationMailer,

		refundCompletionNotificationQueue: refundCompletionNotificationQueue,

		avatarWalletService: avatarWalletService,
		transportationSvc:   transportationSvc,

		onchainWalletReader:   onchainWalletReader,
		tokenTransferReader:   tokenTransferReader,
		previewTransferReader: previewTransferReader,
		tokenTransferExecutor: tokenTransferExecutor,
	}, nil
}
