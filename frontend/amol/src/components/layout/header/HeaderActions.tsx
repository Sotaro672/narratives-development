// frontend/src/components/layout/header/HeaderActions.tsx
import { Link } from "react-router-dom";

import type { HeaderActionState } from "./types";

type HeaderActionsProps = {
  actions: HeaderActionState;
};

export default function HeaderActions({ actions }: HeaderActionsProps) {
  const {
    hasActionButton,
    actionButtonLabel,
    onActionButtonClick,
    actionButtonDisabled,

    hasSecondaryActionButton,
    secondaryActionButtonLabel,
    onSecondaryActionButtonClick,
    secondaryActionButtonDisabled,

    shouldShowLoginButton,
    shouldShowSettingsButton,

    shouldShowCartButton,
    cartButtonLabel,
    onCartButtonClick,
    cartButtonDisabled,
    cartItemCount,

    toggleSettings,
  } = actions;

  const safeCartItemCount =
    typeof cartItemCount === "number" && Number.isFinite(cartItemCount)
      ? Math.max(0, Math.floor(cartItemCount))
      : 0;

  const cartBadgeLabel =
    safeCartItemCount > 99 ? "99+" : String(safeCartItemCount);

  return (
    <div className="header__right">
      {hasActionButton ? (
        <button
          type="button"
          className="header__settings-link header__action-button header__add-to-cart-button"
          aria-label={actionButtonLabel}
          title={actionButtonLabel}
          onClick={onActionButtonClick}
          disabled={actionButtonDisabled}
        >
          {actionButtonLabel}
        </button>
      ) : null}

      {hasSecondaryActionButton ? (
        <button
          type="button"
          className="header__settings-link header__action-button header__secondary-action-button header__buy-button"
          aria-label={secondaryActionButtonLabel}
          title={secondaryActionButtonLabel}
          onClick={onSecondaryActionButtonClick}
          disabled={secondaryActionButtonDisabled}
        >
          {secondaryActionButtonLabel}
        </button>
      ) : null}

      {shouldShowLoginButton ? (
        <Link to="/signin/select" className="header__login-link">
          ログイン
        </Link>
      ) : null}

      {shouldShowCartButton ? (
        <button
          type="button"
          className="header__settings-link header__cart-link"
          aria-label={`${cartButtonLabel || "カート"} ${safeCartItemCount}件`}
          title={cartButtonLabel || "カート"}
          onClick={onCartButtonClick}
          disabled={cartButtonDisabled}
        >
          <span className="header__cart-icon" aria-hidden="true">
            🛒
          </span>

          {safeCartItemCount > 0 ? (
            <span className="header__cart-badge" aria-hidden="true">
              {cartBadgeLabel}
            </span>
          ) : null}
        </button>
      ) : null}

      {shouldShowSettingsButton ? (
        <button
          type="button"
          className="header__settings-link"
          aria-label="設定"
          title="設定"
          onClick={toggleSettings}
        >
          ⚙
        </button>
      ) : null}
    </div>
  );
}