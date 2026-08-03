// frontend/amol/src/pages/AnnouncementPage.tsx

import { useCallback, useState } from "react";
import { useNavigate } from "react-router-dom";

import Layout from "../components/layout/Layout";
import { formatDateTime } from "../components/utils/date";

import {
  useAnnouncementsQuery,
  useMarkAnnouncementReadMutation,
} from "../features/announcement/hooks/useAnnouncementsQuery";

import type { AnnouncementListItem } from "../features/shared/types/announcements";

import "../styles/page-layout.css";
import "../styles/announcement-page.css";

export default function AnnouncementPage() {
  const navigate = useNavigate();

  const [navigatingId, setNavigatingId] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string>("");

  const announcementsQuery = useAnnouncementsQuery({
    page: 1,
    perPage: 100,
  });

  const markAnnouncementReadMutation =
    useMarkAnnouncementReadMutation();

  const items = announcementsQuery.data?.items ?? [];
  const loading = announcementsQuery.isPending;

  const queryError =
    announcementsQuery.error instanceof Error
      ? announcementsQuery.error.message
      : announcementsQuery.error
        ? "お知らせの取得に失敗しました"
        : "";

  const error = actionError || queryError;

  const handleOpenAnnouncement = useCallback(
    async (item: AnnouncementListItem) => {
      if (!item.id || navigatingId) {
        return;
      }

      setNavigatingId(item.id);
      setActionError("");

      let announcementForNavigation = item;

      try {
        if (item.isRead === false) {
          const readAt =
            item.readAt ?? new Date().toISOString();

          await markAnnouncementReadMutation.mutateAsync(
            item.id,
          );

          announcementForNavigation = {
            ...item,
            isRead: true,
            readAt,
          };
        }
      } catch (caught) {
        setActionError(
          caught instanceof Error
            ? caught.message
            : "お知らせの既読化に失敗しました",
        );
      } finally {
        setNavigatingId(null);

        navigate(`/announcements/${item.id}`, {
          state: {
            announcement: announcementForNavigation,
          },
        });
      }
    },
    [
      markAnnouncementReadMutation,
      navigate,
      navigatingId,
    ],
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
          <div
            className="announcement-page__error"
            role="alert"
          >
            {error}
          </div>
        ) : null}

        {loading ? (
          <div className="announcement-page__state">
            読み込み中...
          </div>
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
                item.tokenName ||
                item.targetToken ||
                "対象トークン";

              const publishedAtLabel = formatDateTime(
                item.publishedAt,
              );

              const isNavigating =
                navigatingId === item.id;

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
                  onClick={() =>
                    void handleOpenAnnouncement(item)
                  }
                  onKeyDown={(event) => {
                    if (
                      event.key === "Enter" ||
                      event.key === " "
                    ) {
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

                      <time
                        className="announcement-page__date"
                        dateTime={
                          item.publishedAt ?? undefined
                        }
                      >
                        {publishedAtLabel}
                      </time>
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

                  {Array.isArray(
                    item.attachmentFiles,
                  ) &&
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