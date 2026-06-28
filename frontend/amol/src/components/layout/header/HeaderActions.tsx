// frontend/amol/src/components/layout/header/HeaderActions.tsx
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";

import { useAnnouncementUnreadCount } from "../../../features/announcement/hooks/useAnnouncementUnreadCount";
import { useInquiryUnreadCounter } from "../../../features/inquiry/hooks/useInquiryUnreadCounter";
import { countUnreadReceivedMessages } from "../../../features/message/api/messageApi";
import type { HeaderActionState } from "./types";

type HeaderActionsProps = {
  actions: HeaderActionState;
};

const MESSAGE_UNREAD_FETCH_LIMIT = 100;

function normalizeCount(value: unknown): number {
  return typeof value === "number" && Number.isFinite(value)
    ? Math.max(0, Math.floor(value))
    : 0;
}

function formatBadgeLabel(count: number): string {
  return count > 99 ? "99+" : String(count);
}

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
    shouldShowAnnouncementButton,
    shouldShowSettingsButton,

    shouldShowCartButton,
    cartButtonLabel,
    onCartButtonClick,
    cartButtonDisabled,
    cartItemCount,

    toggleSettings,
  } = actions;

  const [messageUnreadCount, setMessageUnreadCount] = useState(0);

  const { unreadCount: announcementUnreadCount } = useAnnouncementUnreadCount({
    enabled: shouldShowAnnouncementButton,
  });

  const { unreadCount: inquiryUnreadCount } = useInquiryUnreadCounter({
    enabled: shouldShowAnnouncementButton,
  });

  useEffect(() => {
    let ignore = false;

    async function loadMessageUnreadCount() {
      if (!shouldShowAnnouncementButton) {
        setMessageUnreadCount(0);
        return;
      }

      try {
        const unreadCount = await countUnreadReceivedMessages({
          limit: MESSAGE_UNREAD_FETCH_LIMIT,
        });

        if (ignore) {
          return;
        }

        setMessageUnreadCount(unreadCount);
      } catch (error) {
        console.error(error);

        if (!ignore) {
          setMessageUnreadCount(0);
        }
      }
    }

    void loadMessageUnreadCount();

    return () => {
      ignore = true;
    };
  }, [shouldShowAnnouncementButton]);

  const safeCartItemCount = normalizeCount(cartItemCount);
  const safeAnnouncementUnreadCount = normalizeCount(announcementUnreadCount);
  const safeInquiryUnreadCount = normalizeCount(inquiryUnreadCount);
  const safeMessageUnreadCount = normalizeCount(messageUnreadCount);

  const safeChatUnreadCount =
    safeInquiryUnreadCount + safeMessageUnreadCount;

  const cartBadgeLabel = formatBadgeLabel(safeCartItemCount);
  const announcementUnreadBadgeLabel = formatBadgeLabel(
    safeAnnouncementUnreadCount,
  );
  const chatUnreadBadgeLabel = formatBadgeLabel(safeChatUnreadCount);

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

      {shouldShowAnnouncementButton ? (
        <Link
          to="/announcements"
          className="header__settings-link header__cart-link"
          aria-label={`お知らせ ${safeAnnouncementUnreadCount}件`}
          title="お知らせ"
        >
          <span className="header__cart-icon" aria-hidden="true">
            🔔
          </span>

          {safeAnnouncementUnreadCount > 0 ? (
            <span className="header__cart-badge" aria-hidden="true">
              {announcementUnreadBadgeLabel}
            </span>
          ) : null}
        </Link>
      ) : null}

      {shouldShowAnnouncementButton ? (
        <Link
          to="/chats"
          className="header__settings-link header__cart-link"
          aria-label={`メッセージ ${safeChatUnreadCount}件`}
          title="メッセージ"
        >
          <span className="header__cart-icon" aria-hidden="true">
            💬
          </span>

          {safeChatUnreadCount > 0 ? (
            <span className="header__cart-badge" aria-hidden="true">
              {chatUnreadBadgeLabel}
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