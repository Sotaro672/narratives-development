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

type ChatListItem = Inquiry & {
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

  const loadInquiries = useCallback(async (signal?: AbortSignal) => {
    setLoading(true);
    setError("");

    try {
      const result = await listMeInquiries({
        page: 1,
        perPage: 100,
        signal,
      });

      if (signal?.aborted) {
        return;
      }

      const itemsWithReplies = await Promise.all(
        result.items.map(async (item): Promise<ChatListItem> => {
          if (!item.id) {
            return {
              ...item,
              replies: [],
            };
          }

          try {
            const replies = await listInquiryReplies(item.id);

            return {
              ...item,
              replies,
            };
          } catch {
            return {
              ...item,
              replies: [],
            };
          }
        }),
      );

      if (signal?.aborted) {
        return;
      }

      setItems(itemsWithReplies);
    } catch (caught) {
      if (signal?.aborted) {
        return;
      }

      setItems([]);
      setError(
        caught instanceof Error
          ? caught.message
          : "問い合わせ一覧の取得に失敗しました",
      );
    } finally {
      if (!signal?.aborted) {
        setLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    const controller = new AbortController();

    void loadInquiries(controller.signal);

    return () => {
      controller.abort();
    };
  }, [loadInquiries]);

  const handleOpenChat = useCallback(
    async (item: ChatListItem) => {
      if (!item.id || navigatingId) {
        return;
      }

      setNavigatingId(item.id);
      setError("");

      const now = new Date().toISOString();

      let nextItem: ChatListItem = item;

      try {
        if (item.isRead === false) {
          const updated = await markInquiryAsRead(item.id);

          nextItem = {
            ...item,
            ...(updated ?? {}),
            isRead: true,
            readAt: item.readAt ?? now,
            replies: item.replies,
          };

          setItems((current) =>
            current.map((currentItem) =>
              currentItem.id === item.id ? nextItem : currentItem,
            ),
          );
        }
      } catch (caught) {
        setError(
          caught instanceof Error
            ? caught.message
            : "問い合わせの既読化に失敗しました",
        );
      } finally {
        setNavigatingId(null);

        navigate(`/chats/${item.id}`, {
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
            現在、問い合わせはありません。
          </div>
        ) : null}

        {!loading && sortedItems.length > 0 ? (
          <div className="chat-list-page__list" aria-label="チャット一覧">
            {sortedItems.map((item) => {
              const isUnread = item.isRead === false;
              const isNavigating = navigatingId === item.id;

              const title = getInquiryTitle(item);
              const preview = getInquiryPreview(item);
              const dateLabel = formatChatDate(getLatestActivityAt(item));
              const subLabel = getSubLabel(item);
              const statusLabel = getStatusLabel(item.status);
              const replyCount = item.replies.length;

              return (
                <article
                  key={item.id}
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
                    {getInitial(title)}
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
                        {replyCount > 0 ? (
                          <span className="chat-list-page__reply-count">
                            返信 {replyCount} 件
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

function getInquiryTitle(item: ChatListItem): string {
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

function getInquiryPreview(item: ChatListItem): string {
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

function getSubLabel(item: ChatListItem): string {
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
    return "問";
  }

  return Array.from(trimmed)[0] ?? "問";
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