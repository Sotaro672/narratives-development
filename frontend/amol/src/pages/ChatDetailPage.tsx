// frontend/amol/src/pages/ChatDetailPage.tsx
import {
  type ChangeEvent,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useLocation, useParams } from "react-router-dom";

import Layout from "../components/layout/Layout";
import {
  getInquiry,
  listInquiryReplies,
  markInquiryAsRead,
  replyInquiry,
  uploadReplyImage,
  type Inquiry,
  type InquiryImage,
  type InquiryReply,
} from "../features/inquiry/api/inquiryApi";

import "../styles/page-layout.css";
import "../styles/chat-detail-page.css";

type ChatDetailLocationState = {
  inquiry?: Inquiry | null;
  replies?: InquiryReply[] | null;
};

export default function ChatDetailPage() {
  const { inquiryId } = useParams<{ inquiryId: string }>();
  const location = useLocation();

  const state = location.state as ChatDetailLocationState | null;

  const [inquiry, setInquiry] = useState<Inquiry | null>(
    state?.inquiry ?? null,
  );
  const [replies, setReplies] = useState<InquiryReply[]>(
    Array.isArray(state?.replies) ? state.replies : [],
  );
  const [loading, setLoading] = useState<boolean>(!state?.inquiry);
  const [error, setError] = useState<string>("");

  const [isReplyModalOpen, setIsReplyModalOpen] = useState(false);
  const [replyContent, setReplyContent] = useState("");
  const [replyFiles, setReplyFiles] = useState<File[]>([]);
  const [replyError, setReplyError] = useState("");
  const [postingReply, setPostingReply] = useState(false);

  const canSubmitReply = replyContent.trim() !== "" || replyFiles.length > 0;

  const sortedReplies = useMemo(() => {
    return [...replies].sort((a, b) => {
      const aTime = getComparableTime(a.createdAt ?? a.updatedAt);
      const bTime = getComparableTime(b.createdAt ?? b.updatedAt);

      return aTime - bTime;
    });
  }, [replies]);

  const loadThread = useCallback(async () => {
    if (!inquiryId) {
      setInquiry(null);
      setReplies([]);
      setError("問い合わせIDが見つかりません。");
      setLoading(false);
      return;
    }

    setLoading(true);
    setError("");

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
          : "チャット内容の取得に失敗しました",
      );
    } finally {
      setLoading(false);
    }
  }, [inquiryId]);

  useEffect(() => {
    void loadThread();
  }, [loadThread]);

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
    if (!inquiryId || postingReply) {
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
        caught instanceof Error ? caught.message : "返信の送信に失敗しました",
      );
    } finally {
      setPostingReply(false);
    }
  }, [inquiryId, postingReply, replyContent, replyFiles]);

  const title = getInquiryTitle(inquiry);
  const replyActionDisabled = !inquiryId || loading || !inquiry || postingReply;

  return (
    <Layout
      title={title}
      showBackButton
      showFooter
      mode="mypage"
      mainClassName="chat-detail-page-layout"
      actionButtonLabel="返信"
      onActionButtonClick={openReplyModal}
      actionButtonDisabled={replyActionDisabled}
      footerProps={{
        variant: "default",
        centerActionLabel: "返信",
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

        {!loading && !inquiry ? (
          <div className="chat-detail-page__empty">
            問い合わせが見つかりません。
          </div>
        ) : null}

        {!loading && inquiry ? (
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
                <p className="chat-detail-page__content">{inquiry.content}</p>
              ) : null}

              <ImageGrid images={inquiry.images} />
            </article>

            <div className="chat-detail-page__reply-section">
              <h3 className="chat-detail-page__section-title">返信一覧</h3>

              {sortedReplies.length === 0 ? (
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
                </div>
              )}
            </div>
          </div>
        ) : null}
      </section>

      {isReplyModalOpen ? (
        <div className="chat-detail-page__modal-backdrop">
          <div
            className="chat-detail-page__modal"
            role="dialog"
            aria-modal="true"
            aria-labelledby="chat-detail-reply-modal-title"
          >
            <div className="chat-detail-page__modal-header">
              <h2 id="chat-detail-reply-modal-title">返信する</h2>
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
              placeholder="返信内容を入力"
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
        </div>
      ) : null}
    </Layout>
  );
}

function ImageGrid({ images }: { images?: InquiryImage[] | null }) {
  if (!Array.isArray(images) || images.length === 0) {
    return null;
  }

  return (
    <div className="chat-detail-page__images">
      {images.map((image, index) => {
        const src = image.fileUrl;
        if (!src) {
          return null;
        }

        return (
          <a
            key={`${image.objectPath || image.fileName || src}-${index}`}
            className="chat-detail-page__image-link"
            href={src}
            target="_blank"
            rel="noreferrer"
          >
            <img
              className="chat-detail-page__image"
              src={src}
              alt={image.fileName || `添付画像${index + 1}`}
              loading="lazy"
            />
          </a>
        );
      })}
    </div>
  );
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