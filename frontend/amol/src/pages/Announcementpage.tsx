import { useCallback, useEffect, useMemo, useState } from "react";

import {
  fetchMeAnnouncements,
  markMeAnnouncementRead,
} from "../features/announcement/api/announcementApi";
import type { AnnouncementListItem } from "../features/announcement/types";

import "../styles/page-layout.css";
import "../styles/announcement-page.css";

export default function Announcementpage() {
  const [items, setItems] = useState<AnnouncementListItem[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [markingReadId, setMarkingReadId] = useState<string | null>(null);
  const [error, setError] = useState<string>("");

  const unreadCount = useMemo(() => {
    return items.filter((item) => item.isRead === false).length;
  }, [items]);

  const loadAnnouncements = useCallback(async (signal?: AbortSignal) => {
    setLoading(true);
    setError("");

    try {
      const result = await fetchMeAnnouncements({
        page: 1,
        perPage: 100,
        signal,
      });

      setItems(result.items);
    } catch (caught) {
      if (signal?.aborted) {
        return;
      }

      setItems([]);
      setError(
        caught instanceof Error
          ? caught.message
          : "お知らせの取得に失敗しました",
      );
    } finally {
      if (!signal?.aborted) {
        setLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    const controller = new AbortController();

    void loadAnnouncements(controller.signal);

    return () => {
      controller.abort();
    };
  }, [loadAnnouncements]);

  const handleMarkRead = useCallback(
    async (announcementId: string) => {
      if (!announcementId || markingReadId) {
        return;
      }

      setMarkingReadId(announcementId);
      setError("");

      try {
        await markMeAnnouncementRead(announcementId);

        setItems((current) =>
          current.map((item) =>
            item.id === announcementId
              ? {
                  ...item,
                  isRead: true,
                  readAt: item.readAt ?? new Date().toISOString(),
                }
              : item,
          ),
        );
      } catch (caught) {
        setError(
          caught instanceof Error
            ? caught.message
            : "お知らせの既読化に失敗しました",
        );
      } finally {
        setMarkingReadId(null);
      }
    },
    [markingReadId],
  );

  return (
    <section className="page-section content-page-section announcement-page">
      <div className="announcement-page__header">
        <p className="announcement-page__eyebrow">Announcement</p>

        <h1 className="page-title announcement-page__title">お知らせ</h1>

        <p className="page-description content-page-description announcement-page__description">
          あなた宛てのお知らせを確認できます。
        </p>

        <div className="announcement-page__summary" aria-label="未読件数">
          未読 {unreadCount} 件
        </div>
      </div>

      {error ? (
        <div className="announcement-page__error" role="alert">
          {error}
        </div>
      ) : null}

      {loading ? (
        <div className="announcement-page__state">読み込み中...</div>
      ) : null}

      {!loading && items.length === 0 ? (
        <div className="announcement-page__empty">
          現在、お知らせはありません。
        </div>
      ) : null}

      {!loading && items.length > 0 ? (
        <div className="announcement-page__list">
          {items.map((item) => {
            const isUnread = item.isRead === false;
            const tokenLabel = item.tokenName || item.targetToken || "対象トークン";
            const publishedAtLabel = formatDateTime(item.publishedAt);
            const isMarking = markingReadId === item.id;

            return (
              <article
                key={item.id}
                className={
                  isUnread
                    ? "announcement-page__card announcement-page__card--unread"
                    : "announcement-page__card"
                }
              >
                <div className="announcement-page__card-head">
                  <div className="announcement-page__card-meta">
                    <span className="announcement-page__token">
                      {tokenLabel}
                    </span>

                    {publishedAtLabel ? (
                      <time
                        className="announcement-page__date"
                        dateTime={item.publishedAt ?? undefined}
                      >
                        {publishedAtLabel}
                      </time>
                    ) : null}
                  </div>

                  {isUnread ? (
                    <span className="announcement-page__unread-badge">
                      未読
                    </span>
                  ) : (
                    <span className="announcement-page__read-badge">
                      既読
                    </span>
                  )}
                </div>

                <h2 className="announcement-page__card-title">{item.title}</h2>

                <p className="announcement-page__content">{item.content}</p>

                {Array.isArray(item.attachments) && item.attachments.length > 0 ? (
                  <div className="announcement-page__attachments">
                    添付 {item.attachments.length} 件
                  </div>
                ) : null}

                {isUnread ? (
                  <div className="announcement-page__card-actions">
                    <button
                      type="button"
                      className="announcement-page__read-button"
                      onClick={() => void handleMarkRead(item.id)}
                      disabled={isMarking}
                    >
                      {isMarking ? "更新中..." : "既読にする"}
                    </button>
                  </div>
                ) : null}
              </article>
            );
          })}
        </div>
      ) : null}
    </section>
  );
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