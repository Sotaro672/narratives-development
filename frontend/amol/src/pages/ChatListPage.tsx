// frontend/amol/src/pages/ChatListPage.tsx

import Layout from "../components/layout/Layout";

import InquiryListItem from "../features/inquiry/presentation/components/InquiryListItem";
import {
  useInquiryListPage,
  type ResaleChatListItem,
} from "../features/inquiry/presentation/hooks/useInquiryListPage";

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
      showBackButton
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
          <div
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
            })}
          </div>
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
  const dateLabel = formatChatDate(item.latestActivityAt);
  const statusLabel = getResaleStatusLabel(item.status);
  const countLabel = item.commentCount > 0
    ? `コメント ${item.commentCount} 件`
    : "";
  const imageUrl = item.imageUrl || item.tokenIcon;
  const initial = getInitial(item.productName || item.tokenName || item.brandName);

  const handleOpen = () => {
    if (navigating) {
      return;
    }

    onOpen();
  };

  return (
    <article
      className={
        hasAttention
          ? "chat-list-page__row chat-list-page__row--attention"
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
        {imageUrl ? (
          <img
            src={imageUrl}
            alt=""
            className="chat-list-page__avatar-image"
          />
        ) : (
          <span>{initial}</span>
        )}
      </div>

      <div className="chat-list-page__body">
        <div className="chat-list-page__head">
          <div className="chat-list-page__title-wrap">
            <h2 className="chat-list-page__title">
              {title}
            </h2>

            {item.brandName ? (
              <span className="chat-list-page__sub-label">
                {item.brandName}
              </span>
            ) : null}
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
          </div>
        </div>
      </div>
    </article>
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

function getInitial(value: string): string {
  return Array.from(value)[0] ?? "？";
}

function formatChatDate(value: string): string {
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