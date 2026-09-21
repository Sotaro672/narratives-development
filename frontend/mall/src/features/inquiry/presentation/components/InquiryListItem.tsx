// frontend/amol/src/features/inquiry/presentation/components/InquiryListItem.tsx

import { ListRow } from "../../../../components/ui/List";
import type { InquiryChatListItem } from "../hooks/useInquiryListPage";
import { getInquiryTypeLabel } from "../../../shared/types/inquiryTypes";

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
  const closePendingCount = item.status === "resolved" ? 1 : 0;
  const badgeCount = item.unreadReplyCount + closePendingCount;
  const hasAttention = badgeCount > 0;
  const title = getInquiryTitle(item);
  const preview = getInquiryPreview(item);
  const dateLabel = formatInquiryDate(item.latestActivityAt);
  const statusLabel = getInquiryStatusLabel(item.status);
  const countLabel = getReplyCountLabel(item);
  const brandInitial = getInitial(item.brandName || item.productName);

  return (
    <ListRow
      attention={hasAttention}
      busy={navigating}
      ariaLabel={`${title} のチャットを開く`}
      onClick={onOpen}
      leading={
        item.brandIcon ? (
          <img src={item.brandIcon} alt="" />
        ) : (
          <span>{brandInitial}</span>
        )
      }
      title={title}
      subLabel={item.brandName || undefined}
      dateLabel={dateLabel || undefined}
      dateTime={item.latestActivityAt}
      preview={preview}
      meta={
        <>
          {countLabel ? (
            <span className="chat-list-page__reply-count">
              {countLabel}
            </span>
          ) : null}

          <span className="chat-list-page__status">
            {statusLabel}
          </span>

          {hasAttention ? (
            <span
              className="chat-list-page__badge-count"
              aria-label={`要確認 ${badgeCount} 件`}
            >
              {badgeCount > 99 ? "99+" : badgeCount}
            </span>
          ) : null}
        </>
      }
    />
  );
}

function getInquiryTitle(item: InquiryChatListItem): string {
  const typeLabel = getInquiryTypeLabel(item.inquiryType);

  if (!item.productName) {
    return typeLabel;
  }

  return `${item.productName}/${typeLabel}`;
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

function getReplyCountLabel(item: InquiryChatListItem): string {
  return item.replyCount > 0 ? `返信 ${item.replyCount} 件` : "";
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