// frontend/amol/src/pages/ChatListPage.tsx

import Layout from "../components/layout/Layout";
import List, { ListRow } from "../components/ui/List";

import InquiryListItem from "../features/inquiry/presentation/components/InquiryListItem";
import {
  useInquiryListPage,
  type ResaleChatListItem,
  type TradeChatListItem,
} from "../features/inquiry/presentation/hooks/useInquiryListPage";
import { getTradeStatusLabel } from "../features/trade/presentation/util/tradeStatus";

import "../styles/page-layout.css";
import "../features/inquiry/presentation/styles/inquiry-list-page.css";

export default function ChatListPage() {
  const {
    sortedItems,
    loading,
    navigatingId,
    error,
    handleOpenChat,
  } = useInquiryListPage();

  return (
    <Layout
      title="チャット"
      showFooter
      mode="mypage"
      mainClassName="chat-list-page-layout"
    >
      <section className="page-section content-page-section chat-list-page">
        {error ? (
          <div className="chat-list-page__error" role="alert">
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
                  <InquiryListItem
                    key={`inquiry:${item.id}`}
                    item={item}
                    navigating={navigatingId === item.id}
                    onOpen={() => {
                      void handleOpenChat(item);
                    }}
                  />
                );
              }

              if (item.chatKind === "resale") {
                return (
                  <ResaleListItem
                    key={`resale:${item.resaleId}`}
                    item={item}
                    navigating={navigatingId === item.resaleId}
                    onOpen={() => {
                      void handleOpenChat(item);
                    }}
                  />
                );
              }

              return (
                <TradeListItem
                  key={`trade:${item.id}`}
                  item={item}
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
    </Layout>
  );
}

type ResaleListItemProps = {
  item: ResaleChatListItem;
  navigating: boolean;
  onOpen: () => void;
};

function ResaleListItem({
  item,
  navigating,
  onOpen,
}: ResaleListItemProps) {
  const hasAttention = item.unreadCommentCount > 0;
  const title = getResaleTitle(item);
  const preview = getResalePreview(item);
  const statusLabel = getResaleStatusLabel(item.status);
  const countLabel =
    item.commentCount > 0
      ? `コメント ${item.commentCount} 件`
      : "";
  const imageUrl = item.imageUrl || item.tokenIcon;
  const initial = getInitial(
    item.productName || item.tokenName || item.brandName,
  );

  return (
    <ListRow
      attention={hasAttention}
      busy={navigating}
      ariaLabel={`${title} のチャットを開く`}
      onClick={onOpen}
      leading={
        imageUrl ? (
          <img src={imageUrl} alt="" />
        ) : (
          <span>{initial}</span>
        )
      }
      title={title}
      subLabel={item.brandName || undefined}
      dateValue={item.latestActivityAt}
      preview={preview}
      meta={
        <>
          {countLabel ? (
            <span className="chat-list-page__reply-count">
              {countLabel}
            </span>
          ) : null}

          {statusLabel ? (
            <span className="chat-list-page__status">
              {statusLabel}
            </span>
          ) : null}

          {hasAttention ? (
            <span
              className="chat-list-page__badge-count"
              aria-label={`未読 ${item.unreadCommentCount} 件`}
            >
              {item.unreadCommentCount > 99
                ? "99+"
                : item.unreadCommentCount}
            </span>
          ) : null}
        </>
      }
    />
  );
}

type TradeListItemProps = {
  item: TradeChatListItem;
  navigating: boolean;
  onOpen: () => void;
};

function TradeListItem({
  item,
  navigating,
  onOpen,
}: TradeListItemProps) {
  const hasAttention = item.unreadMessageCount > 0;
  const title = getTradeTitle(item);
  const preview = getTradePreview(item);
  const statusLabel = getTradeStatusLabel(item);
  const initial = getInitial(
    item.counterpartAvatarName || item.productName || "取引",
  );

  return (
    <ListRow
      attention={hasAttention}
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
      subLabel={item.counterpartAvatarName || undefined}
      dateValue={item.latestActivityAt}
      preview={preview}
      meta={
        <>
          {statusLabel ? (
            <span className="chat-list-page__status">
              {statusLabel}
            </span>
          ) : null}

          {hasAttention ? (
            <span
              className="chat-list-page__badge-count"
              aria-label={`未読 ${item.unreadMessageCount} 件`}
            >
              {item.unreadMessageCount > 99
                ? "99+"
                : item.unreadMessageCount}
            </span>
          ) : null}
        </>
      }
    />
  );
}

function getResaleTitle(item: ResaleChatListItem): string {
  if (item.productName) {
    return `${item.productName}/再出品`;
  }

  if (item.tokenName) {
    return `${item.tokenName}/再出品`;
  }

  return "再出品";
}

function getTradeTitle(item: TradeChatListItem): string {
  if (item.productName) {
    return `${item.productName}/取引`;
  }

  return "取引";
}

function getResalePreview(item: ResaleChatListItem): string {
  const body = item.latestComment?.body?.trim();

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

function getTradePreview(item: TradeChatListItem): string {
  const content = item.latestMessage?.content?.trim();

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