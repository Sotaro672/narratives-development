// frontend/mall/src/components/layout/header/HeaderActions.tsx

import { Link, useLocation } from "react-router-dom";

import { useInquiryBadgeCounter } from "../../../features/inquiry/presentation/hooks/useInquiryBadgeCounter";
import { useNotificationUnreadCount } from "../../../features/notification/hooks/useNotificationUnreadCount";
import { useResaleChatBadgeCounter } from "../../../features/resale/presentation/hooks/useResaleChatBadgeCounter";

import type { HeaderActionState } from "./types";

type HeaderActionsProps = {
  actions: HeaderActionState;
};

function normalizeCount(value: unknown): number {
  return typeof value === "number" && Number.isFinite(value)
    ? Math.max(0, Math.floor(value))
    : 0;
}

function formatBadgeLabel(count: number): string {
  return count > 99 ? "99+" : String(count);
}

function isResalePagePath(pathname: string): boolean {
  return pathname === "/resale" || pathname === "/resale/";
}

function isResaleDetailPagePath(pathname: string): boolean {
  return /^\/resales\/[^/]+\/?$/.test(pathname);
}

export default function HeaderActions({ actions }: HeaderActionsProps) {
  const location = useLocation();

  const {
    hasActionButton,
    actionButtonLabel,
    onActionButtonClick,
    actionButtonDisabled,
    hasSecondaryActionButton,
    secondaryActionButtonLabel,
    onSecondaryActionButtonClick,
    secondaryActionButtonDisabled,
    hasTertiaryActionButton,
    tertiaryActionButtonLabel,
    onTertiaryActionButtonClick,
    tertiaryActionButtonDisabled,
    shouldShowLoginButton,
    shouldShowAnnouncementButton,
    shouldShowSettingsButton,
    shouldShowCartButton,
    cartButtonLabel,
    onCartButtonClick,
    cartButtonDisabled,
    cartItemCount,
    toggleSettings,
  } = actions;

  const { unreadCount: notificationUnreadCount } = useNotificationUnreadCount({
    enabled: shouldShowAnnouncementButton,
  });

  const { badgeCount: inquiryBadgeCount } = useInquiryBadgeCounter({
    enabled: shouldShowAnnouncementButton,
  });

  const { badgeCount: resaleChatBadgeCount } = useResaleChatBadgeCounter({
    enabled: shouldShowAnnouncementButton,
  });

  const safeCartItemCount = normalizeCount(cartItemCount);
  const safeNotificationUnreadCount = normalizeCount(notificationUnreadCount);
  const safeInquiryBadgeCount = normalizeCount(inquiryBadgeCount);
  const safeResaleChatBadgeCount = normalizeCount(resaleChatBadgeCount);
  const safeChatBadgeCount = safeInquiryBadgeCount + safeResaleChatBadgeCount;

  const cartBadgeLabel = formatBadgeLabel(safeCartItemCount);
  const notificationUnreadBadgeLabel = formatBadgeLabel(
    safeNotificationUnreadCount,
  );
  const chatBadgeLabel = formatBadgeLabel(safeChatBadgeCount);

  const shouldShowResaleButton = isResalePagePath(location.pathname);
  const shouldShowResaleDetailActions = isResaleDetailPagePath(
    location.pathname,
  );

  const resaleButtonLabel = actionButtonLabel || "出品";
  const resaleButtonDisabled = !onActionButtonClick || actionButtonDisabled;

  const primaryActionClassName = [
    "header__settings-link",
    "header__action-button",
    "header__add-to-cart-button",
    shouldShowResaleDetailActions
      ? "header__resale-detail-action-button"
      : "",
  ]
    .filter(Boolean)
    .join(" ");

  const secondaryActionClassName = [
    "header__settings-link",
    "header__action-button",
    "header__secondary-action-button",
    "header__buy-button",
    shouldShowResaleDetailActions
      ? "header__resale-detail-action-button"
      : "",
  ]
    .filter(Boolean)
    .join(" ");

  const tertiaryActionClassName = [
    "header__settings-link",
    "header__action-button",
    "header__tertiary-action-button",
    shouldShowResaleDetailActions
      ? "header__resale-detail-action-button"
      : "",
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div className="header__right">
      {shouldShowResaleButton ? (
        <button
          type="button"
          className="header__settings-link header__resale-button"
          aria-label={resaleButtonLabel}
          title={resaleButtonLabel}
          onClick={onActionButtonClick}
          disabled={resaleButtonDisabled}
        >
          {resaleButtonLabel}
        </button>
      ) : null}

      {hasActionButton && !shouldShowResaleButton ? (
        <button
          type="button"
          className={primaryActionClassName}
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
          className={secondaryActionClassName}
          aria-label={secondaryActionButtonLabel}
          title={secondaryActionButtonLabel}
          onClick={onSecondaryActionButtonClick}
          disabled={secondaryActionButtonDisabled}
        >
          {secondaryActionButtonLabel}
        </button>
      ) : null}

      {hasTertiaryActionButton ? (
        <button
          type="button"
          className={tertiaryActionClassName}
          aria-label={tertiaryActionButtonLabel}
          title={tertiaryActionButtonLabel}
          onClick={onTertiaryActionButtonClick}
          disabled={tertiaryActionButtonDisabled}
        >
          {tertiaryActionButtonLabel}
        </button>
      ) : null}

      {shouldShowLoginButton ? (
        <Link to="/signin/select" className="header__login-link">
          ログイン
        </Link>
      ) : null}

      {shouldShowAnnouncementButton ? (
        <Link
          to="/announcements"
          className="header__settings-link header__cart-link"
          aria-label={`通知 ${safeNotificationUnreadCount}件`}
          title="通知"
        >
          <span className="header__cart-icon" aria-hidden="true">
            🔔
          </span>

          {safeNotificationUnreadCount > 0 ? (
            <span className="header__cart-badge" aria-hidden="true">
              {notificationUnreadBadgeLabel}
            </span>
          ) : null}
        </Link>
      ) : null}

      {shouldShowAnnouncementButton ? (
        <Link
          to="/chats"
          className="header__settings-link header__cart-link"
          aria-label={`メッセージ ${safeChatBadgeCount}件`}
          title="メッセージ"
        >
          <span className="header__cart-icon" aria-hidden="true">
            💬
          </span>

          {safeChatBadgeCount > 0 ? (
            <span className="header__cart-badge" aria-hidden="true">
              {chatBadgeLabel}
            </span>
          ) : null}
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