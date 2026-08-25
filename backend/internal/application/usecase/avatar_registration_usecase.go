// backend/internal/application/usecase/avatar_registration_usecase.go
package usecase

import (
	"context"
	"errors"

	avatardom "narratives/internal/domain/avatar"
)

var (
	ErrAvatarRegistrationUsecaseNotConfigured = errors.New(
		"avatar registration: usecase not configured",
	)
	ErrAvatarRegistrationPaymentMethodUsecaseNotConfigured = errors.New(
		"avatar registration: payment method usecase not configured",
	)
)

type AvatarRegistrationUsecase struct {
	avatarUC *AvatarUsecase

	paymentMethodUC *PaymentMethodUsecase

	autoCreateDevelopmentPaymentMethod bool
}

func NewAvatarRegistrationUsecase(
	avatarUC *AvatarUsecase,
	paymentMethodUC *PaymentMethodUsecase,
	autoCreateDevelopmentPaymentMethod bool,
) *AvatarRegistrationUsecase {
	return &AvatarRegistrationUsecase{
		avatarUC:                           avatarUC,
		paymentMethodUC:                    paymentMethodUC,
		autoCreateDevelopmentPaymentMethod: autoCreateDevelopmentPaymentMethod,
	}
}

func (u *AvatarRegistrationUsecase) Create(
	ctx context.Context,
	in CreateAvatarInput,
) (avatardom.Avatar, error) {
	if u == nil || u.avatarUC == nil {
		return avatardom.Avatar{},
			ErrAvatarRegistrationUsecaseNotConfigured
	}

	if u.autoCreateDevelopmentPaymentMethod {
		if u.paymentMethodUC == nil {
			return avatardom.Avatar{},
				ErrAvatarRegistrationPaymentMethodUsecaseNotConfigured
		}

		_, err :=
			u.paymentMethodUC.
				EnsureDevelopmentDefaultPaymentMethod(
					ctx,
					in.UserUID,
					in.AvatarName,
				)
		if err != nil {
			return avatardom.Avatar{}, err
		}
	}

	return u.avatarUC.Create(
		ctx,
		in,
	)
}
