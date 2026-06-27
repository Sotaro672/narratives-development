// frontend/amol/src/pages/ChatDetailPage.tsx
import { useCallback, useEffect, useMemo, useState } from "react";
import { useLocation, useParams } from "react-router-dom";

import Layout from "../components/layout/Layout";
import {
  getInquiry,
  listInquiryReplies,
  markInquiryAsRead,
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
      const [nextInquiry, nextReplies] = await Promise.all([
        getInquiry(inquiryId),
        listInquiryReplies(inquiryId),
      ]);

      setInquiry(nextInquiry);
      setReplies(nextReplies);

      if (nextInquiry?.isRead === false) {
        await markInquiryAsRead(inquiryId);

        setInquiry((current) =>
          current
            ? {
                ...current,
                isRead: true,
              }
            : current,
        );
      }
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

  const title = getInquiryTitle(inquiry);

  return (
    <Layout
      title={title}
      showBackButton
      showFooter
      mode="mypage"
      mainClassName="chat-detail-page-layout"
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
              <h3 className="chat-detail-page__section-title">
                返信一覧
              </h3>

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
              alt={image.fileName || `添付画像 ${index + 1}`}
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