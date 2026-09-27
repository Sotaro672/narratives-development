// backend/internal/application/usecase/resale_access.go
package usecase

import (
	"context"
	"errors"

	resaledom "narratives/internal/domain/resale"
	tokendom "narratives/internal/domain/token"
)

var (
	// ErrResaleServiceSuspended は、対象アバターが運営裁定により
	// 再販サービスの利用を停止されていることを表す。
	//
	// 停止対象:
	// - 新規Resale出品
	// - suspended状態からlisting状態への再公開
	// - 他ユーザーのResale商品の新規購入
	//
	// 停止対象外:
	// - Avatar自体の利用
	// - Wallet
	// - 新品商品の購入
	// - 既に成立済みのTrade
	// - 発送、返品、返金、入金など既存Tradeの継続処理
	ErrResaleServiceSuspended = errors.New(
		"resale: service suspended",
	)

	// ErrResaleAccessDenied は、指定されたAvatarが対象Resaleの所有者ではなく、
	// 所有者専用操作を実行できないことを表す。
	ErrResaleAccessDenied = errors.New(
		"resale: access denied",
	)
)

// AvatarResaleAccessChecker は、アバターが再販サービスを利用可能かを
// 判定するためのread-side port。
//
// 永続化方式はUsecaseから隠蔽する。
// 現在の想定実装では ReportCases の
// AVATAR + REMOVED を再販利用停止状態として扱う。
type AvatarResaleAccessChecker interface {
	IsAvatarResaleSuspended(
		ctx context.Context,
		avatarID string,
	) (bool, error)
}

// ResaleAssetOwnershipResolver は、新規Resale出品時に
// assetの現在所有権とtoken identityを検証するためのread-side port。
//
// 所有権判定ではFirestore wallet.assetIdsを正とせず、
// on-chain上の現在所有者を正とする。
//
// 実装は以下を提供する:
// - avatarがassetIdを現在所有していることの確認
// - assetIdからcanonical token情報の解決
// - tokenBlueprintIdに属するassetId一覧の解決
//
// WalletUsecaseがこのinterfaceを実装する。
type ResaleAssetOwnershipResolver interface {
	EnsureAvatarOwnsAssetID(
		ctx context.Context,
		avatarID string,
		assetID string,
	) error

	ResolveTokenByAssetID(
		ctx context.Context,
		assetID string,
	) (tokendom.ResolveTokenByAssetIDResult, error)

	ListAssetIDsByTokenBlueprintID(
		ctx context.Context,
		tokenBlueprintID string,
	) (tokendom.ListAssetIDsByTokenBlueprintIDResult, error)
}

// GetOwned は、対象Resaleを取得し、指定Avatarが所有者であることを検証する。
//
// HTTP adapterから所有権判定を分離するための共通Usecase。
// 認証済みAvatarID自体の解決はHTTP middleware / handler側で行い、
// Usecaseには解決済みのAvatarIDのみを渡す。
func (uc *ResaleUsecase) GetOwned(
	ctx context.Context,
	resaleID string,
	avatarID string,
) (resaledom.Resale, error) {
	if uc == nil || uc.resaleRepo == nil {
		return resaledom.Resale{}, ErrNotSupported(
			"Resale.GetOwned",
		)
	}
	if resaleID == "" {
		return resaledom.Resale{}, resaledom.ErrInvalidID
	}
	if avatarID == "" {
		return resaledom.Resale{}, resaledom.ErrInvalidAvatarID
	}

	item, err := uc.resaleRepo.GetByID(ctx, resaleID)
	if err != nil {
		return resaledom.Resale{}, err
	}
	if item.ID != resaleID {
		return resaledom.Resale{}, resaledom.ErrInvalidID
	}
	if item.AvatarID != avatarID {
		return resaledom.Resale{}, ErrResaleAccessDenied
	}

	return item, nil
}

// checkAvatarResaleAccess は、再販サービスを開始する操作の直前に
// 対象アバターの利用可否を検証する共通ヘルパー。
//
// checkerが未設定の場合にアクセスを許可してしまうと、DI漏れによって
// 利用停止を回避できるためfail-closedとする。
func checkAvatarResaleAccess(
	ctx context.Context,
	checker AvatarResaleAccessChecker,
	avatarID string,
) error {
	if avatarID == "" {
		return resaledom.ErrInvalidAvatarID
	}
	if checker == nil {
		return ErrNotSupported(
			"Resale.AvatarResaleAccessChecker",
		)
	}

	suspended, err := checker.IsAvatarResaleSuspended(
		ctx,
		avatarID,
	)
	if err != nil {
		return err
	}
	if suspended {
		return ErrResaleServiceSuspended
	}

	return nil
}

// checkResaleAssetOwnership は、新規Resale出品対象assetについて
// 現在所有権とcanonical token identityを検証する。
//
// 検証順:
// 1. avatarがassetIdをon-chain上で現在所有していること
// 2. assetIdから解決したcanonical productIdが入力値と一致すること
// 3. assetIdが入力されたtokenBlueprintIdに属すること
//
// resolver未設定時はDI漏れを許可しないためfail-closedとする。
func checkResaleAssetOwnership(
	ctx context.Context,
	resolver ResaleAssetOwnershipResolver,
	avatarID string,
	assetID string,
	productID string,
	tokenBlueprintID string,
) error {
	if avatarID == "" {
		return resaledom.ErrInvalidAvatarID
	}
	if assetID == "" {
		return resaledom.ErrInvalidAssetID
	}
	if productID == "" {
		return resaledom.ErrInvalidProductID
	}
	if tokenBlueprintID == "" {
		return resaledom.ErrInvalidTokenBlueprintID
	}
	if resolver == nil {
		return ErrNotSupported(
			"Resale.ResaleAssetOwnershipResolver",
		)
	}

	if err := resolver.EnsureAvatarOwnsAssetID(
		ctx,
		avatarID,
		assetID,
	); err != nil {
		if errors.Is(err, ErrWalletAssetIDNotOwned) ||
			errors.Is(err, ErrResaleAccessDenied) {
			return ErrResaleAccessDenied
		}

		return err
	}

	resolvedToken, err := resolver.ResolveTokenByAssetID(
		ctx,
		assetID,
	)
	if err != nil {
		return err
	}

	if resolvedToken.AssetID == "" ||
		resolvedToken.AssetID != assetID {
		return resaledom.ErrInvalidAssetID
	}
	if resolvedToken.ProductID == "" ||
		resolvedToken.ProductID != productID {
		return resaledom.ErrInvalidProductID
	}

	blueprintAssets, err := resolver.ListAssetIDsByTokenBlueprintID(
		ctx,
		tokenBlueprintID,
	)
	if err != nil {
		return err
	}
	if blueprintAssets.TokenBlueprintID != "" &&
		blueprintAssets.TokenBlueprintID != tokenBlueprintID {
		return resaledom.ErrInvalidTokenBlueprintID
	}

	for _, candidateAssetID := range blueprintAssets.AssetIDs {
		if candidateAssetID == assetID {
			return nil
		}
	}

	return resaledom.ErrInvalidTokenBlueprintID
}

// IsResaleServiceSuspended は、HTTP adapter等で
// ErrResaleServiceSuspendedを判定するためのヘルパー。
func IsResaleServiceSuspended(err error) bool {
	return errors.Is(
		err,
		ErrResaleServiceSuspended,
	)
}

// IsResaleAccessDenied は、HTTP adapter等で
// ErrResaleAccessDeniedを判定するためのヘルパー。
func IsResaleAccessDenied(err error) bool {
	return errors.Is(
		err,
		ErrResaleAccessDenied,
	)
}
