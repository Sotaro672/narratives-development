//frontend\amol\src\pages\ChatListPage.tsx
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
import {
  listReceivedMessages,
  listSentMessages,
  markMessageAsRead,
  type Message,
} from "../features/message/api/messageApi";

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

type MessageThreadListItem = {
  chatKind: "message";
  id: string;
  peerAvatarId: string;
  messages: Message[];
  unreadMessageIds: string[];
  latestMessage?: Message;
  isRead: boolean;
  createdAt?: string;
  updatedAt?: string;
};

type ChatListItem = InquiryChatListItem | MessageThreadListItem;

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
      const [inquiries, messageThreads] = await Promise.all([
        loadInquiryItems(signal),
        loadMessageThreadItems(),
      ]);

      if (signal?.aborted) {
        return;
      }

      setItems([...inquiries, ...messageThreads]);
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
        if (item.chatKind === "inquiry") {
          await openInquiryChat(item, setItems, navigate);
          return;
        }

        await openMessageThread(item, setItems, navigate);
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

async function loadMessageThreadItems(): Promise<MessageThreadListItem[]> {
  const [received, sent] = await Promise.all([
    listReceivedMessages({ limit: 100 }),
    listSentMessages({ limit: 100 }),
  ]);

  return buildMessageThreads(received.messages, sent.messages);
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
        currentItem.chatKind === "inquiry" && currentItem.id === inquiryId
          ? nextItem
          : currentItem,
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

async function openMessageThread(
  item: MessageThreadListItem,
  setItems: React.Dispatch<React.SetStateAction<ChatListItem[]>>,
  navigate: ReturnType<typeof useNavigate>,
): Promise<void> {
  const now = new Date().toISOString();

  if (item.unreadMessageIds.length > 0) {
    await Promise.all(item.unreadMessageIds.map((id) => markMessageAsRead(id)));
  }

  const nextItem: MessageThreadListItem = {
    ...item,
    isRead: true,
    unreadMessageIds: [],
    messages: item.messages.map((message) =>
      item.unreadMessageIds.includes(message.id)
        ? {
            ...message,
            isRead: true,
            readAt: message.readAt ?? now,
            updatedAt: message.updatedAt ?? now,
          }
        : message,
    ),
  };

  setItems((current) =>
    current.map((currentItem) =>
      currentItem.chatKind === "message" && currentItem.id === item.id
        ? nextItem
        : currentItem,
    ),
  );

  navigate(`/chats/messages/${item.peerAvatarId}`, {
    state: {
      messageThread: {
        peerAvatarId: item.peerAvatarId,
        messages: nextItem.messages,
      },
    },
  });
}

function buildMessageThreads(
  received: Message[],
  sent: Message[],
): MessageThreadListItem[] {
  const threads = new Map<
    string,
    {
      peerAvatarId: string;
      messages: Message[];
      unreadMessageIds: string[];
    }
  >();

  for (const message of received) {
    const peerAvatarId = textOrEmpty(message.senderAvatarId);
    if (!peerAvatarId) {
      continue;
    }

    const thread = getOrCreateThread(threads, peerAvatarId);
    thread.messages.push(message);

    if (message.isRead === false && message.id) {
      thread.unreadMessageIds.push(message.id);
    }
  }

  for (const message of sent) {
    const peerAvatarId = textOrEmpty(message.receiverAvatarId);
    if (!peerAvatarId) {
      continue;
    }

    const thread = getOrCreateThread(threads, peerAvatarId);
    thread.messages.push(message);
  }

  return Array.from(threads.values()).map((thread) => {
    const messages = sortMessagesDesc(dedupeMessages(thread.messages));
    const latestMessage = messages[0];

    return {
      chatKind: "message",
      id: `message:${thread.peerAvatarId}`,
      peerAvatarId: thread.peerAvatarId,
      messages,
      unreadMessageIds: Array.from(new Set(thread.unreadMessageIds)),
      latestMessage,
      isRead: thread.unreadMessageIds.length === 0,
      createdAt: latestMessage?.createdAt,
      updatedAt: latestMessage?.updatedAt,
    };
  });
}

function getOrCreateThread(
  threads: Map<
    string,
    {
      peerAvatarId: string;
      messages: Message[];
      unreadMessageIds: string[];
    }
  >,
  peerAvatarId: string,
) {
  const current = threads.get(peerAvatarId);
  if (current) {
    return current;
  }

  const next = {
    peerAvatarId,
    messages: [],
    unreadMessageIds: [],
  };

  threads.set(peerAvatarId, next);
  return next;
}

function dedupeMessages(messages: Message[]): Message[] {
  const byID = new Map<string, Message>();

  for (const message of messages) {
    const key =
      message.id ||
      `${message.senderAvatarId}:${message.receiverAvatarId}:${message.createdAt}`;

    if (!byID.has(key)) {
      byID.set(key, message);
    }
  }

  return Array.from(byID.values());
}

function sortMessagesDesc(messages: Message[]): Message[] {
  return [...messages].sort((a, b) => {
    const aTime = getComparableTime(a.updatedAt || a.createdAt);
    const bTime = getComparableTime(b.updatedAt || b.createdAt);

    return bTime - aTime;
  });
}

function getChatTitle(item: ChatListItem): string {
  if (item.chatKind === "message") {
    return getMessageThreadTitle(item);
  }

  return getInquiryTitle(item);
}

function getChatPreview(item: ChatListItem): string {
  if (item.chatKind === "message") {
    return getMessagePreview(item);
  }

  return getInquiryPreview(item);
}

function getChatSubLabel(item: ChatListItem): string {
  if (item.chatKind === "message") {
    return "メッセージ";
  }

  return getInquirySubLabel(item);
}

function getChatStatusLabel(item: ChatListItem): string {
  if (item.chatKind === "message") {
    return "";
  }

  return getStatusLabel(item.status);
}

function getChatCountLabel(item: ChatListItem): string {
  if (item.chatKind === "message") {
    return item.messages.length > 0 ? `メッセージ ${item.messages.length} 件` : "";
  }

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

function getMessageThreadTitle(item: MessageThreadListItem): string {
  const peerAvatarId = textOrEmpty(item.peerAvatarId);

  if (!peerAvatarId) {
    return "メッセージ";
  }

  return peerAvatarId;
}

function getMessagePreview(item: MessageThreadListItem): string {
  const latest = item.latestMessage ?? item.messages[0];

  const body = textOrEmpty(latest?.body);
  if (body) {
    return body;
  }

  if (Array.isArray(latest?.images) && latest.images.length > 0) {
    return `画像 ${latest.images.length} 件`;
  }

  return "メッセージはありません";
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
  if (item.chatKind === "message") {
    return (
      item.latestMessage?.updatedAt ||
      item.latestMessage?.createdAt ||
      item.updatedAt ||
      item.createdAt
    );
  }

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