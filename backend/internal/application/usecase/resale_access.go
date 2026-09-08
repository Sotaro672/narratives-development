// backend/internal/application/usecase/resale_access.go
package usecase

import (
	"context"
	"errors"

	resaledom "narratives/internal/domain/resale"
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
