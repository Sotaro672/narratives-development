// frontend/amol/src/pages/AnnouncementPage.tsx
import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import Layout from "../components/layout/Layout";
import {
  fetchMeAnnouncements,
  markMeAnnouncementRead,
} from "../features/announcement/api/announcementApi";
import type { AnnouncementListItem } from "../features/announcement/types";

import "../styles/page-layout.css";
import "../styles/announcement-page.css";

export default function AnnouncementPage() {
  const navigate = useNavigate();

  const [items, setItems] = useState<AnnouncementListItem[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [navigatingId, setNavigatingId] = useState<string | null>(null);
  const [error, setError] = useState<string>("");

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

  const handleOpenAnnouncement = useCallback(
    async (item: AnnouncementListItem) => {
      if (!item.id || navigatingId) {
        return;
      }

      setNavigatingId(item.id);
      setError("");

      try {
        if (item.isRead === false) {
          await markMeAnnouncementRead(item.id);

          setItems((current) =>
            current.map((currentItem) =>
              currentItem.id === item.id
                ? {
                    ...currentItem,
                    isRead: true,
                    readAt: currentItem.readAt ?? new Date().toISOString(),
                  }
                : currentItem,
            ),
          );
        }
      } catch (caught) {
        setError(
          caught instanceof Error
            ? caught.message
            : "お知らせの既読化に失敗しました",
        );
      } finally {
        setNavigatingId(null);

        navigate(`/announcements/${item.id}`, {
          state: {
            announcement: {
              ...item,
              isRead: true,
              readAt: item.readAt ?? new Date().toISOString(),
            },
          },
        });
      }
    },
    [navigate, navigatingId],
  );

  return (
    <Layout
      title="お知らせ"
      showBackButton
      showFooter
      mode="mypage"
      mainClassName="announcement-page-layout"
    >
      <section className="page-section content-page-section announcement-page">
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
              const tokenLabel =
                item.tokenName || item.targetToken || "対象トークン";
              const publishedAtLabel = formatDateTime(item.publishedAt);
              const isNavigating = navigatingId === item.id;

              return (
                <article
                  key={item.id}
                  className={
                    isUnread
                      ? "announcement-page__card announcement-page__card--unread"
                      : "announcement-page__card"
                  }
                  role="button"
                  tabIndex={0}
                  aria-label={`${item.title} の詳細を開く`}
                  aria-busy={isNavigating}
                  onClick={() => void handleOpenAnnouncement(item)}
                  onKeyDown={(event) => {
                    if (event.key === "Enter" || event.key === " ") {
                      event.preventDefault();
                      void handleOpenAnnouncement(item);
                    }
                  }}
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

                  <h2 className="announcement-page__card-title">
                    {item.title}
                  </h2>

                  {Array.isArray(item.attachmentFiles) &&
                  item.attachmentFiles.length > 0 ? (
                    <div className="announcement-page__attachments">
                      添付 {item.attachmentFiles.length} 件
                    </div>
                  ) : Array.isArray(item.attachments) &&
                    item.attachments.length > 0 ? (
                    <div className="announcement-page__attachments">
                      添付 {item.attachments.length} 件
                    </div>
                  ) : null}
                </article>
              );
            })}
          </div>
        ) : null}
      </section>
    </Layout>
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