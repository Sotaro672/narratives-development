// frontend/amol/src/features/inquiry/presentation/components/InquiryListItem.tsx

import Badge from "../../../../components/ui/Badge";
import { ListRow } from "../../../../components/ui/List";
import { getInquiryTypeLabel } from "../../../shared/types/inquiryTypes";
import type { InquiryChatListItem } from "../hooks/useInquiryListPage";

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
      dateValue={item.latestActivityAt}
      preview={preview}
      meta={
        <>
          {countLabel ? (
            <Badge
              variant="neutral"
              size="sm"
              className="chat-list-page__reply-count"
            >
              {countLabel}
            </Badge>
          ) : null}

          <Badge
            variant="info"
            size="sm"
            className="chat-list-page__status"
          >
            {statusLabel}
          </Badge>

          {hasAttention ? (
            <Badge
              variant="info"
              size="sm"
              className="chat-list-page__badge-count"
              aria-label={`要確認 ${badgeCount} 件`}
            >
              {badgeCount > 99 ? "99+" : badgeCount}
            </Badge>
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