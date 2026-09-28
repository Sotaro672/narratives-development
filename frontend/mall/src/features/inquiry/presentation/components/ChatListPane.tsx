// frontend/mall/src/features/inquiry/presentation/components/ChatListPane.tsx

import { useLocation } from "react-router-dom";

import Badge from "../../../../components/ui/Badge";
import List, { ListRow } from "../../../../components/ui/List";
import { getInquiryTypeLabel } from "../../../shared/types/inquiryTypes";
import { getTradeStatusLabel } from "../../../trade/presentation/util/tradeStatus";
import {
  useInquiryListPage,
  type InquiryChatListItem,
  type ResaleChatListItem,
  type TradeChatListItem,
} from "../hooks/useInquiryListPage";

type SelectedChat =
  | {
      kind: "inquiry";
      id: string;
    }
  | {
      kind: "resale";
      id: string;
    }
  | {
      kind: "trade";
      id: string;
    }
  | null;

export default function ChatListPane() {
  const location = useLocation();
  const {
    sortedItems,
    loading,
    navigatingId,
    error,
    handleOpenChat,
  } = useInquiryListPage();

  const selectedChat = getSelectedChat(location.pathname);

  return (
    <section className="page-section content-page-section chat-list-page chat-list-page--workspace">
      {error ? (
        <div
          className="chat-list-page__error"
          role="alert"
        >
          {error}
        </div>
      ) : null}

      {loading ? (
        <div className="chat-list-page__state">
          読み込み中...
        </div>
      ) : null}

      {!loading && sortedItems.length === 0 ? (
        <div className="chat-list-page__empty">
          現在、チャットはありません。
        </div>
      ) : null}

      {!loading && sortedItems.length > 0 ? (
        <List
          className="chat-list-page__list"
          aria-label="チャット一覧"
        >
          {sortedItems.map((item) => {
            if (item.chatKind === "inquiry") {
              return (
                <InquiryChatListRow
                  key={`inquiry:${item.id}`}
                  item={item}
                  selected={isSelectedChat(
                    selectedChat,
                    "inquiry",
                    item.id,
                  )}
                  navigating={navigatingId === item.id}
                  onOpen={() => {
                    void handleOpenChat(item);
                  }}
                />
              );
            }

            if (item.chatKind === "resale") {
              return (
                <ResaleChatListRow
                  key={`resale:${item.resaleId}`}
                  item={item}
                  selected={isSelectedChat(
                    selectedChat,
                    "resale",
                    item.resaleId,
                  )}
                  navigating={navigatingId === item.resaleId}
                  onOpen={() => {
                    void handleOpenChat(item);
                  }}
                />
              );
            }

            return (
              <TradeChatListRow
                key={`trade:${item.id}`}
                item={item}
                selected={isSelectedChat(
                  selectedChat,
                  "trade",
                  item.id,
                )}
                navigating={navigatingId === item.id}
                onOpen={() => {
                  void handleOpenChat(item);
                }}
              />
            );
          })}
        </List>
      ) : null}
    </section>
  );
}

type InquiryChatListRowProps = {
  item: InquiryChatListItem;
  selected: boolean;
  navigating: boolean;
  onOpen: () => void;
};

function InquiryChatListRow({
  item,
  selected,
  navigating,
  onOpen,
}: InquiryChatListRowProps) {
  const closePendingCount =
    item.status === "resolved" ? 1 : 0;

  const badgeCount =
    item.unreadReplyCount + closePendingCount;

  const hasAttention = badgeCount > 0;
  const title = getInquiryTitle(item);
  const preview = getInquiryPreview(item);
  const statusLabel = getInquiryStatusLabel(item.status);

  const countLabel =
    item.replyCount > 0
      ? `返信 ${item.replyCount} 件`
      : "";

  const brandInitial = getInitial(
    item.brandName || item.productName,
  );

  return (
    <ListRow
      attention={hasAttention}
      selected={selected}
      busy={navigating}
      ariaLabel={`${title} のチャットを開く`}
      onClick={onOpen}
      leading={
        item.brandIcon ? (
          <img
            src={item.brandIcon}
            alt=""
          />
        ) : (
          <span>{brandInitial}</span>
        )
      }
      title={title}
      subLabel={item.brandName || undefined}
      dateLeading={
        <Badge
          variant="info"
          size="sm"
          className="chat-list-page__status"
        >
          {statusLabel}
        </Badge>
      }
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

type ResaleChatListRowProps = {
  item: ResaleChatListItem;
  selected: boolean;
  navigating: boolean;
  onOpen: () => void;
};

function ResaleChatListRow({
  item,
  selected,
  navigating,
  onOpen,
}: ResaleChatListRowProps) {
  const hasAttention =
    item.unreadCommentCount > 0;

  const title = getResaleTitle(item);
  const preview = getResalePreview(item);
  const statusLabel =
    getResaleStatusLabel(item.status);

  const countLabel =
    item.commentCount > 0
      ? `コメント ${item.commentCount} 件`
      : "";

  const imageUrl =
    item.imageUrl || item.tokenIcon;

  const initial = getInitial(
    item.productName ||
      item.tokenName ||
      item.brandName,
  );

  return (
    <ListRow
      attention={hasAttention}
      selected={selected}
      busy={navigating}
      ariaLabel={`${title} のチャットを開く`}
      onClick={onOpen}
      leading={
        imageUrl ? (
          <img
            src={imageUrl}
            alt=""
          />
        ) : (
          <span>{initial}</span>
        )
      }
      title={title}
      subLabel={item.brandName || undefined}
      dateLeading={
        statusLabel ? (
          <Badge
            variant="info"
            size="sm"
            className="chat-list-page__status"
          >
            {statusLabel}
          </Badge>
        ) : null
      }
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

          {hasAttention ? (
            <Badge
              variant="info"
              size="sm"
              className="chat-list-page__badge-count"
              aria-label={`未読 ${item.unreadCommentCount} 件`}
            >
              {item.unreadCommentCount > 99
                ? "99+"
                : item.unreadCommentCount}
            </Badge>
          ) : null}
        </>
      }
    />
  );
}

type TradeChatListRowProps = {
  item: TradeChatListItem;
  selected: boolean;
  navigating: boolean;
  onOpen: () => void;
};

function TradeChatListRow({
  item,
  selected,
  navigating,
  onOpen,
}: TradeChatListRowProps) {
  const hasAttention =
    item.unreadMessageCount > 0;

  const title = getTradeTitle(item);
  const preview = getTradePreview(item);
  const statusLabel = getTradeStatusLabel(item);

  const initial = getInitial(
    item.counterpartAvatarName ||
      item.productName ||
      "取引",
  );

  return (
    <ListRow
      attention={hasAttention}
      selected={selected}
      busy={navigating}
      ariaLabel={`${title}を開く`}
      onClick={onOpen}
      leading={
        item.counterpartAvatarIcon ? (
          <img
            src={item.counterpartAvatarIcon}
            alt=""
          />
        ) : (
          <span>{initial}</span>
        )
      }
      title={title}
      subLabel={
        item.counterpartAvatarName || undefined
      }
      dateLeading={
        statusLabel ? (
          <Badge
            variant="info"
            size="sm"
            className="chat-list-page__status"
          >
            {statusLabel}
          </Badge>
        ) : null
      }
      dateValue={item.latestActivityAt}
      preview={preview}
      meta={
        hasAttention ? (
          <Badge
            variant="info"
            size="sm"
            className="chat-list-page__badge-count"
            aria-label={`未読 ${item.unreadMessageCount} 件`}
          >
            {item.unreadMessageCount > 99
              ? "99+"
              : item.unreadMessageCount}
          </Badge>
        ) : null
      }
    />
  );
}

function getSelectedChat(
  pathname: string,
): SelectedChat {
  const segments = pathname
    .split("/")
    .filter(Boolean);

  if (segments[0] !== "chats") {
    return null;
  }

  if (
    segments[1] === "resales" &&
    segments[2]
  ) {
    return {
      kind: "resale",
      id: decodePathSegment(segments[2]),
    };
  }

  if (
    segments[1] === "trades" &&
    segments[2] &&
    segments[2] !== "order-items"
  ) {
    return {
      kind: "trade",
      id: decodePathSegment(segments[2]),
    };
  }

  if (
    segments.length === 2 &&
    segments[1]
  ) {
    return {
      kind: "inquiry",
      id: decodePathSegment(segments[1]),
    };
  }

  return null;
}

function isSelectedChat(
  selectedChat: SelectedChat,
  kind: "inquiry" | "resale" | "trade",
  id: string,
): boolean {
  return (
    selectedChat?.kind === kind &&
    selectedChat.id === id
  );
}

function decodePathSegment(
  value: string,
): string {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}

function getInquiryTitle(
  item: InquiryChatListItem,
): string {
  const typeLabel =
    getInquiryTypeLabel(item.inquiryType);

  if (!item.productName) {
    return typeLabel;
  }

  return `${item.productName}/${typeLabel}`;
}

function getInquiryPreview(
  item: InquiryChatListItem,
): string {
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

function getResaleTitle(
  item: ResaleChatListItem,
): string {
  if (item.productName) {
    return `${item.productName}/再出品`;
  }

  if (item.tokenName) {
    return `${item.tokenName}/再出品`;
  }

  return "再出品";
}

function getTradeTitle(
  item: TradeChatListItem,
): string {
  if (item.productName) {
    return `${item.productName}/取引`;
  }

  return "取引";
}

function getResalePreview(
  item: ResaleChatListItem,
): string {
  const body =
    item.latestComment?.body?.trim();

  if (body) {
    return body;
  }

  return "メッセージはありません";
}

function getResaleStatusLabel(
  status: ResaleChatListItem["status"],
): string {
  switch (status) {
    case "listing":
      return "出品中";
    case "suspended":
      return "出品停止";
    case "sold":
      return "売却済";
    default:
      return "";
  }
}

function getTradePreview(
  item: TradeChatListItem,
): string {
  const content =
    item.latestMessage?.content?.trim();

  if (content) {
    return content;
  }

  if (item.latestMessage?.images?.length) {
    return "画像が送信されました";
  }

  return "メッセージはありません";
}

function getInitial(value: string): string {
  return Array.from(value)[0] ?? "？";
}