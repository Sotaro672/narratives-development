// frontend/amol/src/features/inquiry/presentation/components/InquiryListItem.tsx

import type { InquiryChatListItem } from "../hooks/useInquiryListPage";
import {
  getInquiryTypeLabel,
} from "../../../shared/types/inquiryTypes";

type InquiryListItemProps = {
  item: InquiryChatListItem;
  navigating: boolean;
  onOpen: () => void;
};

export default function InquiryListItem({
  item,
  navigating,
  onOpen,
}: InquiryListItemProps) {
  const unreadCount = item.unreadReplyCount;
  const isUnread = unreadCount > 0;
  const title = getInquiryTitle(item);
  const preview = getInquiryPreview(item);
  const dateLabel = formatInquiryDate(item.latestActivityAt);
  const statusLabel = getInquiryStatusLabel(item.status);
  const countLabel = getReplyCountLabel(item);
  const avatarInitial = getInitial(title);

  const handleOpen = () => {
    if (navigating) {
      return;
    }

    onOpen();
  };

  return (
    <article
      className={
        isUnread
          ? "chat-list-page__row chat-list-page__row--unread"
          : "chat-list-page__row"
      }
      role="button"
      tabIndex={0}
      aria-label={`${title} のチャットを開く`}
      aria-busy={navigating}
      onClick={handleOpen}
      onKeyDown={(event) => {
        if (
          navigating ||
          (event.key !== "Enter" && event.key !== " ")
        ) {
          return;
        }

        event.preventDefault();
        onOpen();
      }}
    >
      <div
        className="chat-list-page__avatar"
        aria-hidden="true"
      >
        <span>{avatarInitial}</span>
      </div>

      <div className="chat-list-page__body">
        <div className="chat-list-page__head">
          <div className="chat-list-page__title-wrap">
            <h2 className="chat-list-page__title">
              {title}
            </h2>

            <span className="chat-list-page__sub-label">
              {item.productId}
            </span>
          </div>

          {dateLabel ? (
            <time
              className="chat-list-page__date"
              dateTime={item.latestActivityAt}
            >
              {dateLabel}
            </time>
          ) : null}
        </div>

        <div className="chat-list-page__content">
          <p className="chat-list-page__preview">
            {preview}
          </p>

          <div className="chat-list-page__meta">
            {countLabel ? (
              <span className="chat-list-page__reply-count">
                {countLabel}
              </span>
            ) : null}

            <span className="chat-list-page__status">
              {statusLabel}
            </span>

            {isUnread ? (
              <span
                className="chat-list-page__unread-count"
                aria-label={`未読 ${unreadCount} 件`}
              >
                {unreadCount > 99
                  ? "99+"
                  : unreadCount}
              </span>
            ) : null}
          </div>
        </div>
      </div>
    </article>
  );
}

function getInquiryTitle(item: InquiryChatListItem): string {
  if (item.inquiryType === "product") {
    return item.subject || getInquiryTypeLabel(item.inquiryType);
  }

  return getInquiryTypeLabel(item.inquiryType);
}

function getInquiryPreview(item: InquiryChatListItem): string {
  if (item.latestReply) {
    if (item.latestReply.content) {
      return item.latestReply.content;
    }

    if (item.latestReply.images?.length) {
      return `画像 ${item.latestReply.images.length} 件`;
    }
  }

  if (item.content) {
    return item.content;
  }

  if (item.images?.length) {
    return `画像 ${item.images.length} 件`;
  }

  return "メッセージはありません";
}

function getInquiryStatusLabel(
  status: InquiryChatListItem["status"],
): string {
  switch (status) {
    case "open":
      return "未対応";

    case "in_progress":
      return "対応中";

    case "resolved":
      return "解決済み";

    case "closed":
      return "クローズ";
  }
}

function getReplyCountLabel(
  item: InquiryChatListItem,
): string {
  return item.replyCount > 0
    ? `返信 ${item.replyCount} 件`
    : "";
}

function getInitial(value: string): string {
  return Array.from(value)[0] ?? "？";
}

function formatInquiryDate(value: string): string {
  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return "";
  }

  const now = new Date();
  const isToday =
    date.getFullYear() === now.getFullYear() &&
    date.getMonth() === now.getMonth() &&
    date.getDate() === now.getDate();

  if (isToday) {
    return new Intl.DateTimeFormat("ja-JP", {
      hour: "2-digit",
      minute: "2-digit",
    }).format(date);
  }

  if (date.getFullYear() === now.getFullYear()) {
    return new Intl.DateTimeFormat("ja-JP", {
      month: "2-digit",
      day: "2-digit",
    }).format(date);
  }

  return new Intl.DateTimeFormat("ja-JP", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(date);
}