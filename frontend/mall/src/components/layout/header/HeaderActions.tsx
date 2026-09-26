// frontend/mall/src/components/layout/header/HeaderActions.tsx

import { Bell, MessageCircle, Settings, ShoppingCart } from "lucide-react";
import { Link, useLocation } from "react-router-dom";

import { useInquiryBadgeCounter } from "../../../features/inquiry/presentation/hooks/useInquiryBadgeCounter";
import { useNotificationUnreadCount } from "../../../features/notification/presentation/hooks/useNotificationUnreadCount";
import { useResaleChatBadgeCounter } from "../../../features/resale/presentation/hooks/useResaleChatBadgeCounter";
import { useTradeDispatchBadgeCounter } from "../../../features/trade/presentation/hooks/useTradeDispatchBadgeCounter";

import Badge from "../../ui/Badge";
import Button from "../../ui/Button";
import IconButton from "../../ui/IconButton";
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

  const { badgeCount: tradeDispatchBadgeCount } = useTradeDispatchBadgeCounter({
    enabled: shouldShowAnnouncementButton,
  });

  const safeCartItemCount = normalizeCount(cartItemCount);
  const safeNotificationUnreadCount = normalizeCount(notificationUnreadCount);
  const safeInquiryBadgeCount = normalizeCount(inquiryBadgeCount);
  const safeResaleChatBadgeCount = normalizeCount(resaleChatBadgeCount);
  const safeTradeDispatchBadgeCount = normalizeCount(tradeDispatchBadgeCount);

  const safeChatBadgeCount =
    safeInquiryBadgeCount +
    safeResaleChatBadgeCount +
    safeTradeDispatchBadgeCount;

  const cartBadgeLabel = formatBadgeLabel(safeCartItemCount);
  const notificationUnreadBadgeLabel = formatBadgeLabel(
    safeNotificationUnreadCount,
  );
  const chatBadgeLabel = formatBadgeLabel(safeChatBadgeCount);

  const shouldShowResaleDetailActions = isResaleDetailPagePath(
    location.pathname,
  );

  const primaryActionClassName = [
    "header__action-button",
    "header__add-to-cart-button",
    shouldShowResaleDetailActions ? "header__resale-detail-action-button" : "",
  ]
    .filter(Boolean)
    .join(" ");

  const secondaryActionClassName = [
    "header__action-button",
    "header__secondary-action-button",
    "header__buy-button",
    shouldShowResaleDetailActions ? "header__resale-detail-action-button" : "",
  ]
    .filter(Boolean)
    .join(" ");

  const tertiaryActionClassName = [
    "header__action-button",
    "header__tertiary-action-button",
    shouldShowResaleDetailActions ? "header__resale-detail-action-button" : "",
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div className="header__right">
      {hasActionButton ? (
        <Button
          variant="secondary"
          size="sm"
          className={primaryActionClassName}
          aria-label={actionButtonLabel}
          title={actionButtonLabel}
          onClick={onActionButtonClick}
          disabled={actionButtonDisabled}
        >
          {actionButtonLabel}
        </Button>
      ) : null}

      {hasSecondaryActionButton ? (
        <Button
          variant="secondary"
          size="sm"
          className={secondaryActionClassName}
          aria-label={secondaryActionButtonLabel}
          title={secondaryActionButtonLabel}
          onClick={onSecondaryActionButtonClick}
          disabled={secondaryActionButtonDisabled}
        >
          {secondaryActionButtonLabel}
        </Button>
      ) : null}

      {hasTertiaryActionButton ? (
        <Button
          variant="secondary"
          size="sm"
          className={tertiaryActionClassName}
          aria-label={tertiaryActionButtonLabel}
          title={tertiaryActionButtonLabel}
          onClick={onTertiaryActionButtonClick}
          disabled={tertiaryActionButtonDisabled}
        >
          {tertiaryActionButtonLabel}
        </Button>
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
            <Bell size={20} strokeWidth={1.8} />
          </span>

          {safeNotificationUnreadCount > 0 ? (
            <Badge
              variant="danger"
              size="sm"
              className="header__cart-badge"
              aria-hidden="true"
            >
              {notificationUnreadBadgeLabel}
            </Badge>
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
            <MessageCircle size={20} strokeWidth={1.8} />
          </span>

          {safeChatBadgeCount > 0 ? (
            <Badge
              variant="danger"
              size="sm"
              className="header__cart-badge"
              aria-hidden="true"
            >
              {chatBadgeLabel}
            </Badge>
          ) : null}
        </Link>
      ) : null}

      {shouldShowCartButton ? (
        <IconButton
          variant="ghost"
          size="md"
          className="header__cart-link"
          aria-label={`${cartButtonLabel || "カート"} ${safeCartItemCount}件`}
          title={cartButtonLabel || "カート"}
          onClick={onCartButtonClick}
          disabled={cartButtonDisabled}
        >
          <ShoppingCart size={20} strokeWidth={1.8} aria-hidden="true" />

          {safeCartItemCount > 0 ? (
            <Badge
              variant="danger"
              size="sm"
              className="header__cart-badge"
              aria-hidden="true"
            >
              {cartBadgeLabel}
            </Badge>
          ) : null}
        </IconButton>
      ) : null}

      {shouldShowSettingsButton ? (
        <IconButton
          variant="ghost"
          size="md"
          aria-label="設定"
          title="設定"
          onClick={toggleSettings}
        >
          <Settings size={20} strokeWidth={1.8} aria-hidden="true" />
        </IconButton>
      ) : null}
    </div>
  );
}