// frontend/amol/src/pages/ChatListPage.tsx
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import Layout from "../components/layout/Layout";
import {
  listInquiryReplies,
  listMeInquiries,
  markInquiryAsRead,
  type Inquiry,
  type InquiryReply,
} from "../features/inquiry/api/inquiryApi";

import "../styles/page-layout.css";
import "../styles/chat-list-page.css";

type InquiryChatListItem = Inquiry & {
  chatKind: "inquiry";
  readAt?: string | null;

  productName?: string | null;
  tokenName?: string | null;
  brandName?: string | null;
  avatarName?: string | null;
  senderName?: string | null;

  latestMessage?: string | null;
  latestReplyContent?: string | null;

  replies: InquiryReply[];
};

type ChatListItem = InquiryChatListItem;

export default function ChatListPage() {
  const navigate = useNavigate();

  const [items, setItems] = useState<ChatListItem[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [navigatingId, setNavigatingId] = useState<string | null>(null);
  const [error, setError] = useState<string>("");

  const sortedItems = useMemo(() => {
    return [...items].sort((a, b) => {
      const aTime = getComparableTime(getLatestActivityAt(a));
      const bTime = getComparableTime(getLatestActivityAt(b));

      return bTime - aTime;
    });
  }, [items]);

  const loadChats = useCallback(async (signal?: AbortSignal) => {
    setLoading(true);
    setError("");

    try {
      const inquiries = await loadInquiryItems(signal);

      if (signal?.aborted) {
        return;
      }

      setItems(inquiries);
    } catch (caught) {
      if (signal?.aborted) {
        return;
      }

      setItems([]);
      setError(
        caught instanceof Error
          ? caught.message
          : "チャット一覧の取得に失敗しました",
      );
    } finally {
      if (!signal?.aborted) {
        setLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    const controller = new AbortController();

    void loadChats(controller.signal);

    return () => {
      controller.abort();
    };
  }, [loadChats]);

  const handleOpenChat = useCallback(
    async (item: ChatListItem) => {
      if (!item.id || navigatingId) {
        return;
      }

      setNavigatingId(item.id);
      setError("");

      try {
        await openInquiryChat(item, setItems, navigate);
      } catch (caught) {
        setError(
          caught instanceof Error
            ? caught.message
            : "チャットを開く処理に失敗しました",
        );
      } finally {
        setNavigatingId(null);
      }
    },
    [navigate, navigatingId],
  );

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
          <div className="chat-list-page__state">読み込み中...</div>
        ) : null}

        {!loading && sortedItems.length === 0 ? (
          <div className="chat-list-page__empty">
            現在、チャットはありません。
          </div>
        ) : null}

        {!loading && sortedItems.length > 0 ? (
          <div className="chat-list-page__list" aria-label="チャット一覧">
            {sortedItems.map((item) => {
              const isUnread = item.isRead === false;
              const isNavigating = navigatingId === item.id;

              const title = getChatTitle(item);
              const preview = getChatPreview(item);
              const dateLabel = formatChatDate(getLatestActivityAt(item));
              const subLabel = getChatSubLabel(item);
              const statusLabel = getChatStatusLabel(item);
              const countLabel = getChatCountLabel(item);
              const avatarIcon = getChatAvatarIcon(item);
              const avatarInitial = getInitial(title);

              return (
                <article
                  key={`${item.chatKind}:${item.id}`}
                  className={
                    isUnread
                      ? "chat-list-page__row chat-list-page__row--unread"
                      : "chat-list-page__row"
                  }
                  role="button"
                  tabIndex={0}
                  aria-label={`${title} のチャットを開く`}
                  aria-busy={isNavigating}
                  onClick={() => void handleOpenChat(item)}
                  onKeyDown={(event) => {
                    if (event.key === "Enter" || event.key === " ") {
                      event.preventDefault();
                      void handleOpenChat(item);
                    }
                  }}
                >
                  <div className="chat-list-page__avatar" aria-hidden="true">
                    {avatarIcon ? (
                      <img
                        src={avatarIcon}
                        alt=""
                        loading="lazy"
                        referrerPolicy="no-referrer"
                        style={{
                          width: "100%",
                          height: "100%",
                          display: "block",
                          objectFit: "cover",
                          borderRadius: "inherit",
                        }}
                        onError={(event) => {
                          event.currentTarget.style.display = "none";

                          const fallback =
                            event.currentTarget.nextElementSibling;
                          if (fallback instanceof HTMLElement) {
                            fallback.style.display = "inline";
                          }
                        }}
                      />
                    ) : null}

                    <span style={avatarIcon ? { display: "none" } : undefined}>
                      {avatarInitial}
                    </span>
                  </div>

                  <div className="chat-list-page__body">
                    <div className="chat-list-page__head">
                      <div className="chat-list-page__title-wrap">
                        <h2 className="chat-list-page__title">{title}</h2>

                        {subLabel ? (
                          <span className="chat-list-page__sub-label">
                            {subLabel}
                          </span>
                        ) : null}
                      </div>

                      {dateLabel ? (
                        <time
                          className="chat-list-page__date"
                          dateTime={getLatestActivityAt(item) ?? undefined}
                        >
                          {dateLabel}
                        </time>
                      ) : null}
                    </div>

                    <div className="chat-list-page__content">
                      <p className="chat-list-page__preview">{preview}</p>

                      <div className="chat-list-page__meta">
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

                        {isUnread ? (
                          <span
                            className="chat-list-page__unread-dot"
                            aria-label="未読"
                          />
                        ) : null}
                      </div>
                    </div>
                  </div>
                </article>
              );
            })}
          </div>
        ) : null}
      </section>
    </Layout>
  );
}

async function loadInquiryItems(
  signal?: AbortSignal,
): Promise<InquiryChatListItem[]> {
  const result = await listMeInquiries({
    page: 1,
    perPage: 100,
    signal,
  });

  if (signal?.aborted) {
    return [];
  }

  return Promise.all(
    result.items.map(async (item): Promise<InquiryChatListItem> => {
      if (!item.id) {
        return {
          ...item,
          chatKind: "inquiry",
          replies: [],
        };
      }

      try {
        const replies = await listInquiryReplies(item.id);

        return {
          ...item,
          chatKind: "inquiry",
          replies,
        };
      } catch {
        return {
          ...item,
          chatKind: "inquiry",
          replies: [],
        };
      }
    }),
  );
}

async function openInquiryChat(
  item: InquiryChatListItem,
  setItems: React.Dispatch<React.SetStateAction<ChatListItem[]>>,
  navigate: ReturnType<typeof useNavigate>,
): Promise<void> {
  const inquiryId = item.id;

  if (!inquiryId) {
    return;
  }

  const now = new Date().toISOString();

  let nextItem: InquiryChatListItem = item;

  if (item.isRead === false) {
    const updated = await markInquiryAsRead(inquiryId);

    nextItem = {
      ...item,
      ...(updated ?? {}),
      chatKind: "inquiry",
      isRead: true,
      readAt: item.readAt ?? now,
      replies: item.replies,
    };

    setItems((current) =>
      current.map((currentItem) =>
        currentItem.id === inquiryId ? nextItem : currentItem,
      ),
    );
  }

  navigate(`/chats/${inquiryId}`, {
    state: {
      inquiry: {
        ...nextItem,
        isRead: true,
        readAt: nextItem.readAt ?? now,
      },
      replies: nextItem.replies,
    },
  });
}

function getChatTitle(item: ChatListItem): string {
  return getInquiryTitle(item);
}

function getChatPreview(item: ChatListItem): string {
  return getInquiryPreview(item);
}

function getChatSubLabel(item: ChatListItem): string {
  return getInquirySubLabel(item);
}

function getChatStatusLabel(item: ChatListItem): string {
  return getStatusLabel(item.status);
}

function getChatCountLabel(item: ChatListItem): string {
  return item.replies.length > 0 ? `返信 ${item.replies.length} 件` : "";
}

function getInquiryTitle(item: InquiryChatListItem): string {
  const subject = textOrEmpty(item.subject);
  if (subject) {
    return subject;
  }

  const productName = textOrEmpty(item.productName);
  if (productName) {
    return productName;
  }

  const tokenName = textOrEmpty(item.tokenName);
  if (tokenName) {
    return tokenName;
  }

  return "問い合わせ";
}

function getInquiryPreview(item: InquiryChatListItem): string {
  const latestReply = getLatestReply(item.replies);
  const latestReplyContent = textOrEmpty(latestReply?.content);

  if (latestReplyContent) {
    return latestReplyContent;
  }

  const content = textOrEmpty(item.content);
  if (content) {
    return content;
  }

  if (Array.isArray(item.images) && item.images.length > 0) {
    return `画像 ${item.images.length} 件`;
  }

  return "メッセージはありません";
}

function getChatAvatarIcon(_item: ChatListItem): string {
  return "";
}

function getInquirySubLabel(item: InquiryChatListItem): string {
  return (
    textOrEmpty(item.brandName) ||
    textOrEmpty(item.avatarName) ||
    textOrEmpty(item.senderName) ||
    textOrEmpty(item.productId) ||
    ""
  );
}

function getStatusLabel(status?: string | null): string {
  switch (status) {
    case "open":
      return "未対応";
    case "resolved":
      return "解決済み";
    case "closed":
      return "クローズ";
    default:
      return "";
  }
}

function getInitial(value: string): string {
  const trimmed = textOrEmpty(value);

  if (!trimmed) {
    return "？";
  }

  return Array.from(trimmed)[0] ?? "？";
}

function getLatestActivityAt(item: ChatListItem): string | null | undefined {
  const latestReply = getLatestReply(item.replies);

  return (
    latestReply?.updatedAt ||
    latestReply?.createdAt ||
    item.updatedAt ||
    item.createdAt
  );
}

function getLatestReply(replies: InquiryReply[]): InquiryReply | null {
  if (!Array.isArray(replies) || replies.length === 0) {
    return null;
  }

  return [...replies].sort((a, b) => {
    const aTime = getComparableTime(a.updatedAt ?? a.createdAt);
    const bTime = getComparableTime(b.updatedAt ?? b.createdAt);

    return bTime - aTime;
  })[0] ?? null;
}

function formatChatDate(value?: string | null): string {
  if (!value) {
    return "";
  }

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return "";
  }

  const now = new Date();
  const sameYear = date.getFullYear() === now.getFullYear();
  const sameMonth = date.getMonth() === now.getMonth();
  const sameDate = date.getDate() === now.getDate();

  if (sameYear && sameMonth && sameDate) {
    return new Intl.DateTimeFormat("ja-JP", {
      hour: "2-digit",
      minute: "2-digit",
    }).format(date);
  }

  if (sameYear) {
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

function getComparableTime(value?: string | null): number {
  if (!value) {
    return 0;
  }

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return 0;
  }

  return date.getTime();
}

function textOrEmpty(value: unknown): string {
  return String(value ?? "").trim();
}