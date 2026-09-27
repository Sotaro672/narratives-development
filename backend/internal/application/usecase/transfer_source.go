// backend/internal/application/usecase/transfer_source.go
package usecase

import (
	"context"
	"fmt"

	applicationport "narratives/internal/application/port"
	orderdom "narratives/internal/domain/order"
)

type transferExecutionSource struct {
	FromAvatarID string
	FromBrandID  string

	FromWallet string
}

func (u *TransferUsecase) resolveTransferSource(
	ctx context.Context,
	target applicationport.TransferTargetItem,
	brandID string,
	buyerAvatarID string,
) (transferExecutionSource, error) {
	switch target.ItemType {
	case orderdom.OrderItemTypeList:
		return u.resolveListTransferSource(
			ctx,
			brandID,
		)

	case orderdom.OrderItemTypeResale:
		return u.resolveResaleTransferSource(
			ctx,
			target,
			buyerAvatarID,
		)

	default:
		return transferExecutionSource{},
			ErrTransferNoEligibleOrder
	}
}

func (u *TransferUsecase) resolveListTransferSource(
	ctx context.Context,
	brandID string,
) (transferExecutionSource, error) {
	if brandID == "" {
		return transferExecutionSource{},
			ErrTransferBrandIDEmpty
	}

	fromWallet, err :=
		u.brandWallet.ResolveBrandWalletAddress(
			ctx,
			brandID,
		)
	if err != nil {
		return transferExecutionSource{},
			fmt.Errorf(
				"transfer_uc: resolve brand wallet failed brandId=%s: %w",
				brandID,
				err,
			)
	}
	if fromWallet == "" {
		return transferExecutionSource{},
			ErrTransferFromWalletEmpty
	}

	return transferExecutionSource{
		FromBrandID: brandID,
		FromWallet:  fromWallet,
	}, nil
}

func (u *TransferUsecase) resolveResaleTransferSource(
	ctx context.Context,
	target applicationport.TransferTargetItem,
	buyerAvatarID string,
) (transferExecutionSource, error) {
	if u.resaleRepo == nil {
		return transferExecutionSource{},
			ErrTransferResaleNotConfigured
	}

	resaleID := target.ResaleID
	if resaleID == "" {
		return transferExecutionSource{},
			ErrTransferResaleIDEmpty
	}

	resale, err := u.resaleRepo.GetByID(
		ctx,
		resaleID,
	)
	if err != nil {
		return transferExecutionSource{},
			fmt.Errorf(
				"transfer_uc: resolve resale failed resaleId=%s: %w",
				resaleID,
				err,
			)
	}

	fromAvatarID := resale.AvatarID
	if fromAvatarID == "" {
		return transferExecutionSource{},
			ErrTransferResaleSellerAvatarIDEmpty
	}
	if fromAvatarID == buyerAvatarID {
		return transferExecutionSource{},
			ErrTransferSameAvatar
	}

	fromWallet, err :=
		u.avatarWallet.ResolveAvatarWalletAddress(
			ctx,
			fromAvatarID,
		)
	if err != nil {
		return transferExecutionSource{},
			fmt.Errorf(
				"transfer_uc: resolve seller avatar wallet failed avatarId=%s: %w",
				fromAvatarID,
				err,
			)
	}
	if fromWallet == "" {
		return transferExecutionSource{},
			ErrTransferFromWalletEmpty
	}

	return transferExecutionSource{
		FromAvatarID: fromAvatarID,
		FromWallet:   fromWallet,
	}, nil
}
