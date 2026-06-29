//frontend\amol\src\pages\ChatDetailPage.tsx
import {
  type ChangeEvent,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { createPortal } from "react-dom";
import { useLocation, useParams } from "react-router-dom";

import Layout from "../components/layout/Layout";
import {
  closeInquiry,
  getInquiry,
  listInquiryReplies,
  markInquiryAsRead,
  replyInquiry,
  uploadReplyImage,
  type Inquiry,
  type InquiryImage,
  type InquiryReply,
} from "../features/inquiry/api/inquiryApi";
import {
  listMessageThread,
  markMessageAsRead,
  sendMessage,
  uploadMessageImage,
  type Message,
  type MessageImageAttachment,
} from "../features/message/api/messageApi";

import "../styles/page-layout.css";
import "../styles/chat-detail-page.css";

type ChatDetailLocationState = {
  inquiry?: Inquiry | null;
  replies?: InquiryReply[] | null;
  messageThread?: {
    peerAvatarId: string;
    peerAvatarName?: string | null;
    peerAvatarIcon?: string | null;
    messages: Message[];
  } | null;
};

type ChatRouteParams = {
  inquiryId?: string;
  peerAvatarId?: string;
};

type PeerAvatarInfo = {
  name: string;
  icon: string;
};

export default function ChatDetailPage() {
  const { inquiryId, peerAvatarId } = useParams<ChatRouteParams>();
  const location = useLocation();

  const state = location.state as ChatDetailLocationState | null;
  const isMessageMode = location.pathname.startsWith("/chats/messages/");
  const messagePeerAvatarId =
    peerAvatarId || (isMessageMode ? getLastPathSegment(location.pathname) : "");

  const initialMessageThread = isMessageMode
    ? state?.messageThread ?? null
    : null;
  const initialMessageThreadMessages = Array.isArray(
    initialMessageThread?.messages,
  )
    ? initialMessageThread.messages
    : [];
  const initialPeerAvatarInfo = resolvePeerAvatarFromMessages(
    initialMessageThreadMessages,
    messagePeerAvatarId,
  );

  const [inquiry, setInquiry] = useState<Inquiry | null>(
    isMessageMode ? null : state?.inquiry ?? null,
  );
  const [replies, setReplies] = useState<InquiryReply[]>(
    !isMessageMode && Array.isArray(state?.replies) ? state.replies : [],
  );
  const [messages, setMessages] = useState<Message[]>(
    isMessageMode && Array.isArray(state?.messageThread?.messages)
      ? state.messageThread.messages
      : [],
  );
  const [peerAvatarName, setPeerAvatarName] = useState<string>(
    textOrEmpty(initialMessageThread?.peerAvatarName) ||
      initialPeerAvatarInfo.name,
  );
  const [peerAvatarIcon, setPeerAvatarIcon] = useState<string>(
    textOrEmpty(initialMessageThread?.peerAvatarIcon) ||
      initialPeerAvatarInfo.icon,
  );

  const [loading, setLoading] = useState<boolean>(
    isMessageMode
      ? !Array.isArray(state?.messageThread?.messages)
      : !state?.inquiry,
  );
  const [error, setError] = useState<string>("");

  const [isReplyModalOpen, setIsReplyModalOpen] = useState(false);
  const [replyContent, setReplyContent] = useState("");
  const [replyFiles, setReplyFiles] = useState<File[]>([]);
  const [replyError, setReplyError] = useState("");
  const [postingReply, setPostingReply] = useState(false);

  const [closingInquiry, setClosingInquiry] = useState(false);
  const [closeError, setCloseError] = useState("");

  const canSubmitReply = replyContent.trim() !== "" || replyFiles.length > 0;

  const sortedReplies = useMemo(() => {
    return [...replies].sort((a, b) => {
      const aTime = getComparableTime(a.createdAt ?? a.updatedAt);
      const bTime = getComparableTime(b.createdAt ?? b.updatedAt);

      return aTime - bTime;
    });
  }, [replies]);

  const sortedMessages = useMemo(() => {
    return [...messages].sort((a, b) => {
      const aTime = getComparableTime(a.createdAt ?? a.updatedAt);
      const bTime = getComparableTime(b.createdAt ?? b.updatedAt);

      return aTime - bTime;
    });
  }, [messages]);

  const loadThread = useCallback(async () => {
    if (isMessageMode) {
      if (!messagePeerAvatarId) {
        setMessages([]);
        setPeerAvatarName("");
        setPeerAvatarIcon("");
        setError("相手のアバターIDを取得できませんでした。");
        setLoading(false);
        return;
      }

      setLoading(true);
      setError("");
      setCloseError("");

      try {
        const result = await listMessageThread(messagePeerAvatarId, {
          limit: 100,
        });

        const nextMessages = result.messages ?? [];
        setMessages(nextMessages);

        const resolvedPeer = resolvePeerAvatarFromMessages(
          nextMessages,
          messagePeerAvatarId,
        );

        setPeerAvatarName((current) => resolvedPeer.name || current);
        setPeerAvatarIcon((current) => resolvedPeer.icon || current);

        const unreadIds = nextMessages
          .filter((message) => message.isRead === false)
          .map((message) => message.id)
          .filter(Boolean);

        if (unreadIds.length > 0) {
          await Promise.allSettled(
            unreadIds.map((messageId) => markMessageAsRead(messageId)),
          );

          setMessages((current) =>
            current.map((message) =>
              unreadIds.includes(message.id)
                ? {
                    ...message,
                    isRead: true,
                    readAt: message.readAt ?? new Date().toISOString(),
                  }
                : message,
            ),
          );
        }
      } catch (caught) {
        setMessages([]);
        setError(
          caught instanceof Error
            ? caught.message
            : "メッセージの取得に失敗しました。",
        );
      } finally {
        setLoading(false);
      }

      return;
    }

    if (!inquiryId) {
      setInquiry(null);
      setReplies([]);
      setError("問い合わせIDが見つかりません。");
      setLoading(false);
      return;
    }

    setLoading(true);
    setError("");
    setCloseError("");

    try {
      const [nextInquiry, nextReplies, updatedInquiry] = await Promise.all([
        getInquiry(inquiryId),
        listInquiryReplies(inquiryId),
        markInquiryAsRead(inquiryId),
      ]);

      setInquiry(updatedInquiry ?? nextInquiry);
      setReplies(nextReplies);
    } catch (caught) {
      setInquiry(null);
      setReplies([]);
      setError(
        caught instanceof Error
          ? caught.message
          : "チャット内容の取得に失敗しました。",
      );
    } finally {
      setLoading(false);
    }
  }, [inquiryId, isMessageMode, messagePeerAvatarId]);

  useEffect(() => {
    void loadThread();
  }, [loadThread]);

  useEffect(() => {
    if (!isReplyModalOpen) {
      return;
    }

    const previousOverflow = document.body.style.overflow;
    const previousTouchAction = document.body.style.touchAction;

    document.body.style.overflow = "hidden";
    document.body.style.touchAction = "none";

    return () => {
      document.body.style.overflow = previousOverflow;
      document.body.style.touchAction = previousTouchAction;
    };
  }, [isReplyModalOpen]);

  const openReplyModal = useCallback(() => {
    setReplyError("");
    setIsReplyModalOpen(true);
  }, []);

  const closeReplyModal = useCallback(() => {
    if (postingReply) {
      return;
    }

    setIsReplyModalOpen(false);
    setReplyContent("");
    setReplyFiles([]);
    setReplyError("");
  }, [postingReply]);

  const handleReplyFilesChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      const files = Array.from(event.target.files ?? []);

      if (files.length > 0) {
        setReplyFiles((current) => [...current, ...files]);
      }

      event.target.value = "";
    },
    [],
  );

  const removeReplyFile = useCallback((index: number) => {
    setReplyFiles((current) =>
      current.filter((_, currentIndex) => currentIndex !== index),
    );
  }, []);

  const submitReply = useCallback(async () => {
    if (postingReply) {
      return;
    }

    const content = replyContent.trim();

    if (!content && replyFiles.length === 0) {
      setReplyError("本文または画像を入力してください。");
      return;
    }

    setPostingReply(true);
    setReplyError("");

    try {
      if (isMessageMode) {
        if (!messagePeerAvatarId) {
          setReplyError("相手のアバターIDを取得できませんでした。");
          return;
        }

        const images = await Promise.all(
          replyFiles.map((file) =>
            uploadMessageImage({
              receiverAvatarId: messagePeerAvatarId,
              file,
            }),
          ),
        );

        const createdMessage = await sendMessage({
          receiverAvatarId: messagePeerAvatarId,
          body: content,
          images,
        });

        if (createdMessage) {
          setMessages((current) => [...current, createdMessage]);

          const resolvedPeer = resolvePeerAvatarFromMessages(
            [createdMessage],
            messagePeerAvatarId,
          );

          setPeerAvatarName((current) => resolvedPeer.name || current);
          setPeerAvatarIcon((current) => resolvedPeer.icon || current);
        } else {
          const result = await listMessageThread(messagePeerAvatarId, {
            limit: 100,
          });
          const nextMessages = result.messages ?? [];

          setMessages(nextMessages);

          const resolvedPeer = resolvePeerAvatarFromMessages(
            nextMessages,
            messagePeerAvatarId,
          );

          setPeerAvatarName((current) => resolvedPeer.name || current);
          setPeerAvatarIcon((current) => resolvedPeer.icon || current);
        }

        setIsReplyModalOpen(false);
        setReplyContent("");
        setReplyFiles([]);
        return;
      }

      if (!inquiryId) {
        return;
      }

      const images = await Promise.all(
        replyFiles.map((file) =>
          uploadReplyImage({
            inquiryId,
            file,
          }),
        ),
      );

      const createdReply = await replyInquiry(inquiryId, {
        content,
        images,
      });

      if (createdReply) {
        setReplies((current) => [...current, createdReply]);
      } else {
        const nextReplies = await listInquiryReplies(inquiryId);
        setReplies(nextReplies);
      }

      setIsReplyModalOpen(false);
      setReplyContent("");
      setReplyFiles([]);
    } catch (caught) {
      setReplyError(
        caught instanceof Error ? caught.message : "送信に失敗しました。",
      );
    } finally {
      setPostingReply(false);
    }
  }, [
    inquiryId,
    isMessageMode,
    messagePeerAvatarId,
    postingReply,
    replyContent,
    replyFiles,
  ]);

  const handleCloseInquiry = useCallback(async () => {
    if (!inquiryId || closingInquiry || isMessageMode) {
      return;
    }

    setClosingInquiry(true);
    setCloseError("");

    try {
      const closedInquiry = await closeInquiry(inquiryId);

      setInquiry((current) =>
        closedInquiry ??
        (current
          ? {
              ...current,
              status: "closed",
            }
          : current),
      );
    } catch (caught) {
      setCloseError(
        caught instanceof Error ? caught.message : "クローズに失敗しました。",
      );
    } finally {
      setClosingInquiry(false);
    }
  }, [inquiryId, closingInquiry, isMessageMode]);

  const messageTitle = textOrEmpty(peerAvatarName) || "メッセージ";
  const title = isMessageMode ? messageTitle : getInquiryTitle(inquiry);
  const shouldShowClosePrompt = !isMessageMode && inquiry?.status === "resolved";
  const replyActionDisabled = isMessageMode
    ? !messagePeerAvatarId || loading || postingReply
    : !inquiryId ||
      loading ||
      !inquiry ||
      postingReply ||
      inquiry.status === "closed";

  return (
    <>
      <Layout
        title={title}
        showBackButton
        showFooter={!isReplyModalOpen}
        mode="mypage"
        mainClassName="chat-detail-page-layout"
        actionButtonLabel={isMessageMode ? "送信" : "返信"}
        onActionButtonClick={openReplyModal}
        actionButtonDisabled={replyActionDisabled}
        footerProps={{
          variant: "default",
          centerActionLabel: isMessageMode ? "送信" : "返信",
          centerActionDisabled: replyActionDisabled,
          onCenterActionClick: openReplyModal,
        }}
      >
        <section className="page-section content-page-section chat-detail-page">
          {error ? (
            <div className="chat-detail-page__error" role="alert">
              {error}
            </div>
          ) : null}

          {loading ? (
            <div className="chat-detail-page__state">読み込み中...</div>
          ) : null}

          {!loading && isMessageMode ? (
            <MessageThread
              messages={sortedMessages}
              peerAvatarId={messagePeerAvatarId}
              peerAvatarName={peerAvatarName}
              peerAvatarIcon={peerAvatarIcon}
            />
          ) : null}

          {!loading && !isMessageMode && !inquiry ? (
            <div className="chat-detail-page__empty">
              問い合わせが見つかりません。
            </div>
          ) : null}

          {!loading && !isMessageMode && inquiry ? (
            <div className="chat-detail-page__thread">
              <article className="chat-detail-page__inquiry">
                <div className="chat-detail-page__message-head">
                  <div>
                    <span className="chat-detail-page__sender">
                      あなたの問い合わせ
                    </span>

                    {inquiry.createdAt ? (
                      <time
                        className="chat-detail-page__date"
                        dateTime={inquiry.createdAt}
                      >
                        {formatDateTime(inquiry.createdAt)}
                      </time>
                    ) : null}
                  </div>

                  {inquiry.status ? (
                    <span className="chat-detail-page__status">
                      {getStatusLabel(inquiry.status)}
                    </span>
                  ) : null}
                </div>

                {inquiry.subject ? (
                  <h2 className="chat-detail-page__subject">
                    {inquiry.subject}
                  </h2>
                ) : null}

                {inquiry.content ? (
                  <p className="chat-detail-page__content">
                    {inquiry.content}
                  </p>
                ) : null}

                <ImageGrid images={inquiry.images} />
              </article>

              <div className="chat-detail-page__reply-section">
                <h3 className="chat-detail-page__section-title">返信一覧</h3>

                {sortedReplies.length === 0 && !shouldShowClosePrompt ? (
                  <div className="chat-detail-page__no-replies">
                    まだ返信はありません。
                  </div>
                ) : (
                  <div className="chat-detail-page__replies">
                    {sortedReplies.map((reply, index) => {
                      const isAvatarReply = reply.senderType === "avatar";

                      return (
                        <article
                          key={reply.id || `${reply.inquiryId}-${index}`}
                          className={
                            isAvatarReply
                              ? "chat-detail-page__reply chat-detail-page__reply--avatar"
                              : "chat-detail-page__reply"
                          }
                        >
                          <div className="chat-detail-page__message-head">
                            <div>
                              <span className="chat-detail-page__sender">
                                {isAvatarReply ? "あなた" : "テナント"}
                              </span>

                              {reply.createdAt ? (
                                <time
                                  className="chat-detail-page__date"
                                  dateTime={reply.createdAt}
                                >
                                  {formatDateTime(reply.createdAt)}
                                </time>
                              ) : null}
                            </div>
                          </div>

                          {reply.content ? (
                            <p className="chat-detail-page__content">
                              {reply.content}
                            </p>
                          ) : null}

                          <ImageGrid images={reply.images} />
                        </article>
                      );
                    })}

                    {shouldShowClosePrompt ? (
                      <article className="chat-detail-page__reply chat-detail-page__reply--system">
                        <div className="chat-detail-page__message-head">
                          <div>
                            <span className="chat-detail-page__sender">
                              テナント
                            </span>
                          </div>
                        </div>

                        <p className="chat-detail-page__content">
                          クローズしますか？
                        </p>

                        {closeError ? (
                          <div
                            className="chat-detail-page__modal-error"
                            role="alert"
                          >
                            {closeError}
                          </div>
                        ) : null}

                        <div className="chat-detail-page__close-prompt-actions">
                          <button
                            type="button"
                            onClick={handleCloseInquiry}
                            disabled={closingInquiry}
                          >
                            {closingInquiry ? "クローズ中..." : "クローズする"}
                          </button>
                        </div>
                      </article>
                    ) : null}
                  </div>
                )}
              </div>
            </div>
          ) : null}
        </section>
      </Layout>

      {isReplyModalOpen
        ? createPortal(
            <div className="chat-detail-page__modal-backdrop">
              <div
                className="chat-detail-page__modal"
                role="dialog"
                aria-modal="true"
                aria-labelledby="chat-detail-reply-modal-title"
              >
                <div className="chat-detail-page__modal-header">
                  <h2 id="chat-detail-reply-modal-title">
                    {isMessageMode ? "メッセージを送る" : "返信する"}
                  </h2>
                  <button
                    type="button"
                    className="chat-detail-page__modal-close"
                    onClick={closeReplyModal}
                    disabled={postingReply}
                    aria-label="閉じる"
                  >
                    ×
                  </button>
                </div>

                <textarea
                  className="chat-detail-page__reply-input"
                  value={replyContent}
                  onChange={(event) => setReplyContent(event.target.value)}
                  placeholder={isMessageMode ? "メッセージを入力" : "返信内容を入力"}
                  rows={6}
                  disabled={postingReply}
                />

                <label className="chat-detail-page__file-picker">
                  <span>画像を追加</span>
                  <input
                    type="file"
                    accept="image/*"
                    multiple
                    onChange={handleReplyFilesChange}
                    disabled={postingReply}
                  />
                </label>

                {replyFiles.length > 0 ? (
                  <div className="chat-detail-page__selected-files">
                    {replyFiles.map((file, index) => (
                      <div
                        key={`${file.name}-${file.lastModified}-${index}`}
                        className="chat-detail-page__selected-file"
                      >
                        <span>{file.name}</span>
                        <button
                          type="button"
                          onClick={() => removeReplyFile(index)}
                          disabled={postingReply}
                        >
                          削除
                        </button>
                      </div>
                    ))}
                  </div>
                ) : null}

                {replyError ? (
                  <div className="chat-detail-page__modal-error" role="alert">
                    {replyError}
                  </div>
                ) : null}

                <div className="chat-detail-page__modal-actions">
                  <button
                    type="button"
                    onClick={closeReplyModal}
                    disabled={postingReply}
                  >
                    キャンセル
                  </button>
                  <button
                    type="button"
                    onClick={submitReply}
                    disabled={!canSubmitReply || postingReply}
                  >
                    {postingReply ? "送信中..." : "送信"}
                  </button>
                </div>
              </div>
            </div>,
            document.body,
          )
        : null}
    </>
  );
}

function MessageThread({
  messages,
  peerAvatarId,
  peerAvatarName,
  peerAvatarIcon,
}: {
  messages: Message[];
  peerAvatarId: string;
  peerAvatarName?: string;
  peerAvatarIcon?: string;
}) {
  const peerDisplayName = textOrEmpty(peerAvatarName) || "相手";

  if (!Array.isArray(messages) || messages.length === 0) {
    return (
      <div className="chat-detail-page__empty">
        まだメッセージはありません。
      </div>
    );
  }

  return (
    <div className="chat-detail-page__thread">
      <div className="chat-detail-page__reply-section">
        <h3 className="chat-detail-page__section-title">メッセージ</h3>

        <div className="chat-detail-page__replies">
          {messages.map((message, index) => {
            const isOwnMessage = message.senderAvatarId !== peerAvatarId;
            const senderName = isOwnMessage ? "あなた" : peerDisplayName;
            const senderIcon = isOwnMessage
              ? ""
              : getPeerIconForMessage(message, peerAvatarId, peerAvatarIcon);
            const initialSource = senderName || peerAvatarId || "相";

            return (
              <article
                key={message.id || `${message.createdAt}-${index}`}
                className={
                  isOwnMessage
                    ? "chat-detail-page__reply chat-detail-page__reply--avatar chat-detail-page__message-card chat-detail-page__message-card--own"
                    : "chat-detail-page__reply chat-detail-page__message-card chat-detail-page__message-card--peer"
                }
              >
                <div
                  className={
                    isOwnMessage
                      ? "chat-detail-page__message-line chat-detail-page__message-line--own"
                      : "chat-detail-page__message-line"
                  }
                >
                  {!isOwnMessage ? (
                    <MessageAvatarIcon
                      icon={senderIcon}
                      fallbackText={initialSource}
                    />
                  ) : null}

                  <div className="chat-detail-page__message-main">
                    <div className="chat-detail-page__message-head">
                      <div>
                        <span className="chat-detail-page__sender">
                          {senderName}
                        </span>

                        {message.createdAt ? (
                          <time
                            className="chat-detail-page__date"
                            dateTime={message.createdAt}
                          >
                            {formatDateTime(message.createdAt)}
                          </time>
                        ) : null}
                      </div>
                    </div>

                    {message.body ? (
                      <p className="chat-detail-page__content">
                        {message.body}
                      </p>
                    ) : null}

                    <ImageGrid images={message.images} />
                  </div>
                </div>
              </article>
            );
          })}
        </div>
      </div>
    </div>
  );
}

function MessageAvatarIcon({
  icon,
  fallbackText,
}: {
  icon?: string;
  fallbackText: string;
}) {
  return (
    <div className="chat-detail-page__message-avatar" aria-hidden="true">
      {icon ? (
        <img
          className="chat-detail-page__message-avatar-image"
          src={icon}
          alt=""
          loading="lazy"
          referrerPolicy="no-referrer"
          onError={(event) => {
            event.currentTarget.classList.add("is-hidden");

            const fallback = event.currentTarget.nextElementSibling;
            if (fallback instanceof HTMLElement) {
              fallback.classList.remove("is-hidden");
            }
          }}
        />
      ) : null}

      <span
        className={
          icon
            ? "chat-detail-page__message-avatar-fallback is-hidden"
            : "chat-detail-page__message-avatar-fallback"
        }
      >
        {getInitial(fallbackText)}
      </span>
    </div>
  );
}

function ImageGrid({
  images,
}: {
  images?: Array<InquiryImage | MessageImageAttachment> | null;
}) {
  if (!Array.isArray(images) || images.length === 0) {
    return null;
  }

  return (
    <div className="chat-detail-page__images">
      {images.map((image, index) => {
        const src = getImageSrc(image);
        if (!src) {
          return null;
        }

        const label = getImageLabel(image, index);

        return (
          <a
            key={`${getImageKey(image, src)}-${index}`}
            className="chat-detail-page__image-link"
            href={src}
            target="_blank"
            rel="noreferrer"
          >
            <img
              className="chat-detail-page__image"
              src={src}
              alt={label}
              loading="lazy"
            />
          </a>
        );
      })}
    </div>
  );
}

function getImageSrc(image: InquiryImage | MessageImageAttachment): string {
  if ("fileUrl" in image) {
    return image.fileUrl || "";
  }

  return image.downloadUrl || "";
}

function getImageLabel(
  image: InquiryImage | MessageImageAttachment,
  index: number,
): string {
  if ("fileName" in image && image.fileName) {
    return image.fileName;
  }

  return `添付画像 ${index + 1}`;
}

function getImageKey(
  image: InquiryImage | MessageImageAttachment,
  src: string,
): string {
  if ("objectPath" in image && image.objectPath) {
    return image.objectPath;
  }

  if ("storagePath" in image && image.storagePath) {
    return image.storagePath;
  }

  return src;
}

function getInquiryTitle(inquiry: Inquiry | null): string {
  const subject = textOrEmpty(inquiry?.subject);

  if (subject) {
    return subject;
  }

  return "チャット詳細";
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

function resolvePeerAvatarFromMessages(
  messages: Message[] | null | undefined,
  peerAvatarId: string,
): PeerAvatarInfo {
  let name = "";
  let icon = "";

  for (const message of messages ?? []) {
    if (!name) {
      name = textOrEmpty(message.peerAvatarName);
    }

    if (!icon) {
      icon = textOrEmpty(message.peerAvatarIcon);
    }

    if (name && icon) {
      return { name, icon };
    }
  }

  for (const message of messages ?? []) {
    const senderAvatarId = textOrEmpty(message.senderAvatarId);
    const receiverAvatarId = textOrEmpty(message.receiverAvatarId);

    if (senderAvatarId === peerAvatarId) {
      if (!name) {
        name = textOrEmpty(message.senderAvatarName);
      }

      if (!icon) {
        icon = textOrEmpty(message.senderAvatarIcon);
      }
    }

    if (receiverAvatarId === peerAvatarId) {
      if (!name) {
        name = textOrEmpty(message.receiverAvatarName);
      }

      if (!icon) {
        icon = textOrEmpty(message.receiverAvatarIcon);
      }
    }

    if (name && icon) {
      return { name, icon };
    }
  }

  return { name, icon };
}

function getPeerIconForMessage(
  message: Message,
  peerAvatarId: string,
  fallbackIcon?: string,
): string {
  const peerIcon = textOrEmpty(message.peerAvatarIcon);
  if (peerIcon) {
    return peerIcon;
  }

  if (textOrEmpty(message.senderAvatarId) === peerAvatarId) {
    return textOrEmpty(message.senderAvatarIcon) || textOrEmpty(fallbackIcon);
  }

  if (textOrEmpty(message.receiverAvatarId) === peerAvatarId) {
    return textOrEmpty(message.receiverAvatarIcon) || textOrEmpty(fallbackIcon);
  }

  return textOrEmpty(fallbackIcon);
}

function getLastPathSegment(pathname: string): string {
  const parts = pathname.split("/").filter(Boolean);
  return decodeURIComponent(parts[parts.length - 1] ?? "");
}

function formatDateTime(value?: string | null): string {
  if (!value) {
    return "";
  }

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return "";
  }

  return new Intl.DateTimeFormat("ja-JP", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

function getInitial(value: string): string {
  const trimmed = textOrEmpty(value);

  if (!trimmed) {
    return "？";
  }

  return Array.from(trimmed)[0] ?? "？";
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