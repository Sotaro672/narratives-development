// frontend/mall/src/pages/AnnouncementPage.tsx

import { useCallback, useMemo } from "react";
import { useNavigate } from "react-router-dom";

import Layout from "../components/layout/Layout";
import { formatDateTime } from "../components/utils/date";

import { useAnnouncementsQuery } from "../features/announcement/hooks/useAnnouncementsQuery";
import {
  useMarkNewsReadMutation,
  useNewsQuery,
} from "../features/news/hooks/useNewsQuery";
import { useReportDecisionNotificationsQuery } from "../features/notification/hooks/useReportDecisionNotificationsQuery";
import type { ReportDecisionNotification } from "../features/notification/infrastructure/reportDecisionNotificationApi";
import type { AnnouncementListItem } from "../features/shared/types/announcements";
import type { News } from "../features/shared/types/news";
import type { ReportTargetType } from "../features/shared/types/report";

import "../styles/page-layout.css";
import "../styles/announcement-page.css";

type AnnouncementFeedItem = {
  kind: "announcement";
  key: string;
  occurredAt: string;
  announcement: AnnouncementListItem;
};

type ReportDecisionFeedItem = {
  kind: "reportDecision";
  key: string;
  occurredAt: string;
  notification: ReportDecisionNotification;
};

type NewsFeedItem = {
  kind: "news";
  key: string;
  occurredAt: string;
  news: News;
};

type NotificationFeedItem =
  | AnnouncementFeedItem
  | ReportDecisionFeedItem
  | NewsFeedItem;

function toTimestamp(value: string | null | undefined): number {
  if (!value) {
    return 0;
  }

  const timestamp = Date.parse(value);
  return Number.isFinite(timestamp) ? timestamp : 0;
}

function getReportTargetLabel(
  targetType: ReportTargetType,
): string {
  switch (targetType) {
    case "PRODUCT_BLUEPRINT_REVIEW":
      return "商品レビュー";
    case "LIST":
      return "出品";
    case "TOKEN_BLUEPRINT":
      return "トークン";
    case "TOKEN_BLUEPRINT_COMMENT":
      return "トークンコメント";
    case "AVATAR":
      return "アバター";
    case "RESALE":
      return "再販出品";
    default:
      return "投稿内容";
  }
}

function getDecisionCardLabel(
  notification: ReportDecisionNotification,
  targetLabel: string,
): string {
  if (notification.notificationKind === "TARGET_ENFORCEMENT") {
    return `運営からのお知らせ・${targetLabel}`;
  }

  return `通報結果・${targetLabel}`;
}

function getDecisionCardTitle(
  notification: ReportDecisionNotification,
): string {
  if (notification.notificationKind === "TARGET_ENFORCEMENT") {
    return "運営による措置のお知らせ";
  }

  return "通報内容の確認が完了しました";
}

export default function AnnouncementPage() {
  const navigate = useNavigate();

  const announcementsQuery = useAnnouncementsQuery({
    page: 1,
    perPage: 100,
  });

  const decisionNotificationsQuery =
    useReportDecisionNotificationsQuery({
      page: 1,
      perPage: 100,
    });

  const newsQuery = useNewsQuery({
    page: 1,
    perPage: 100,
  });

  const {
    mutate: markNewsRead,
    isPending: isMarkingNewsRead,
    variables: markingNewsId,
    error: markNewsReadError,
  } = useMarkNewsReadMutation();

  const announcements = useMemo(
    () => announcementsQuery.data?.items ?? [],
    [announcementsQuery.data?.items],
  );

  const decisionNotifications = useMemo(
    () => decisionNotificationsQuery.data?.items ?? [],
    [decisionNotificationsQuery.data?.items],
  );

  const news = useMemo(
    () => newsQuery.data?.items ?? [],
    [newsQuery.data?.items],
  );

  const items = useMemo<NotificationFeedItem[]>(() => {
    const announcementItems: AnnouncementFeedItem[] =
      announcements.map((announcement) => ({
        kind: "announcement",
        key: `announcement:${announcement.id}`,
        occurredAt:
          announcement.publishedAt ??
          announcement.createdAt ??
          "",
        announcement,
      }));

    const decisionItems: ReportDecisionFeedItem[] =
      decisionNotifications.map((notification) => ({
        kind: "reportDecision",
        key: `reportDecision:${notification.id}`,
        occurredAt:
          notification.decidedAt ||
          notification.createdAt,
        notification,
      }));

    const newsItems: NewsFeedItem[] = news.map((newsItem) => ({
      kind: "news",
      key: `news:${newsItem.id}`,
      occurredAt:
        newsItem.publishedAt ||
        newsItem.createdAt ||
        "",
      news: newsItem,
    }));

    return [
      ...announcementItems,
      ...decisionItems,
      ...newsItems,
    ].sort(
      (left, right) =>
        toTimestamp(right.occurredAt) -
        toTimestamp(left.occurredAt),
    );
  }, [
    announcements,
    decisionNotifications,
    news,
  ]);

  const loading =
    announcementsQuery.isPending ||
    decisionNotificationsQuery.isPending ||
    newsQuery.isPending;

  const announcementQueryError =
    announcementsQuery.error instanceof Error
      ? announcementsQuery.error.message
      : announcementsQuery.error
        ? "お知らせの取得に失敗しました"
        : "";

  const decisionQueryError =
    decisionNotificationsQuery.error instanceof Error
      ? decisionNotificationsQuery.error.message
      : decisionNotificationsQuery.error
        ? "通報結果通知の取得に失敗しました"
        : "";

  const newsQueryError =
    newsQuery.error instanceof Error
      ? newsQuery.error.message
      : newsQuery.error
        ? "システム通知の取得に失敗しました"
        : "";

  const newsReadError =
    markNewsReadError instanceof Error
      ? markNewsReadError.message
      : markNewsReadError
        ? "システム通知を既読にできませんでした"
        : "";

  const error =
    announcementQueryError ||
    decisionQueryError ||
    newsQueryError ||
    newsReadError;

  const handleOpenAnnouncement = useCallback(
    (item: AnnouncementListItem) => {
      if (!item.id) {
        return;
      }

      navigate(
        `/announcements/${encodeURIComponent(item.id)}`,
        {
          state: {
            announcement: item,
          },
        },
      );
    },
    [navigate],
  );

  const handleOpenDecisionNotification = useCallback(
    (notification: ReportDecisionNotification) => {
      if (!notification.id) {
        return;
      }

      navigate(
        `/announcements/report-decisions/${encodeURIComponent(
          notification.id,
        )}`,
        {
          state: {
            reportDecisionNotification: notification,
          },
        },
      );
    },
    [navigate],
  );

  const handleMarkNewsRead = useCallback(
    (newsItem: News) => {
      const newsId = newsItem.id?.trim();

      if (
        !newsId ||
        newsItem.isRead ||
        isMarkingNewsRead
      ) {
        return;
      }

      markNewsRead(newsId);
    },
    [
      isMarkingNewsRead,
      markNewsRead,
    ],
  );

  return (
    <Layout
      title="通知"
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
            現在、通知はありません。
          </div>
        ) : null}

        {!loading && items.length > 0 ? (
          <div className="announcement-page__list">
            {items.map((item) => {
              if (item.kind === "announcement") {
                const announcement = item.announcement;
                const isUnread =
                  announcement.isRead === false;
                const tokenLabel =
                  announcement.tokenName ||
                  announcement.targetToken ||
                  "お知らせ";
                const occurredAtLabel =
                  formatDateTime(item.occurredAt);

                return (
                  <article
                    key={item.key}
                    className={
                      isUnread
                        ? "announcement-page__card announcement-page__card--unread"
                        : "announcement-page__card"
                    }
                    role="button"
                    tabIndex={0}
                    aria-label={`${announcement.title} の詳細を開く`}
                    onClick={() =>
                      handleOpenAnnouncement(
                        announcement,
                      )
                    }
                    onKeyDown={(event) => {
                      if (
                        event.key === "Enter" ||
                        event.key === " "
                      ) {
                        event.preventDefault();
                        handleOpenAnnouncement(
                          announcement,
                        );
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
                            item.occurredAt ||
                            undefined
                          }
                        >
                          {occurredAtLabel}
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
                      {announcement.title}
                    </h2>

                    {Array.isArray(
                      announcement.attachmentFiles,
                    ) &&
                    announcement.attachmentFiles.length >
                      0 ? (
                      <div className="announcement-page__attachments">
                        添付{" "}
                        {
                          announcement
                            .attachmentFiles.length
                        }{" "}
                        件
                      </div>
                    ) : Array.isArray(
                        announcement.attachments,
                      ) &&
                      announcement.attachments.length >
                        0 ? (
                      <div className="announcement-page__attachments">
                        添付{" "}
                        {
                          announcement.attachments
                            .length
                        }{" "}
                        件
                      </div>
                    ) : null}
                  </article>
                );
              }

              if (item.kind === "news") {
                const newsItem = item.news;
                const isUnread =
                  newsItem.isRead === false;
                const isBusy =
                  isMarkingNewsRead &&
                  markingNewsId?.trim() ===
                    newsItem.id;
                const occurredAtLabel =
                  formatDateTime(item.occurredAt);

                return (
                  <article
                    key={item.key}
                    className={[
                      "announcement-page__card",
                      isUnread
                        ? "announcement-page__card--unread"
                        : "",
                      isBusy
                        ? "announcement-page__card--busy"
                        : "",
                    ]
                      .filter(Boolean)
                      .join(" ")}
                    role={isUnread ? "button" : undefined}
                    tabIndex={isUnread ? 0 : undefined}
                    aria-label={
                      isUnread
                        ? `${newsItem.title} を既読にする`
                        : undefined
                    }
                    aria-busy={
                      isBusy ? true : undefined
                    }
                    onClick={() => {
                      if (isUnread) {
                        handleMarkNewsRead(newsItem);
                      }
                    }}
                    onKeyDown={(event) => {
                      if (
                        !isUnread ||
                        (event.key !== "Enter" &&
                          event.key !== " ")
                      ) {
                        return;
                      }

                      event.preventDefault();
                      handleMarkNewsRead(newsItem);
                    }}
                  >
                    <div className="announcement-page__card-head">
                      <div className="announcement-page__card-meta">
                        <span className="announcement-page__token">
                          システム通知
                        </span>

                        <time
                          className="announcement-page__date"
                          dateTime={
                            item.occurredAt ||
                            undefined
                          }
                        >
                          {occurredAtLabel}
                        </time>
                      </div>

                      {isBusy ? (
                        <span className="announcement-page__unread-badge">
                          既読処理中
                        </span>
                      ) : isUnread ? (
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
                      {newsItem.title}
                    </h2>

                    <div className="announcement-page__news-body">
                      {newsItem.body}
                    </div>
                  </article>
                );
              }

              const notification = item.notification;
              const isUnread =
                notification.isRead === false;
              const targetLabel =
                getReportTargetLabel(
                  notification.targetType,
                );
              const occurredAtLabel =
                formatDateTime(item.occurredAt);
              const cardLabel =
                getDecisionCardLabel(
                  notification,
                  targetLabel,
                );
              const cardTitle =
                getDecisionCardTitle(notification);

              return (
                <article
                  key={item.key}
                  className={
                    isUnread
                      ? "announcement-page__card announcement-page__card--unread"
                      : "announcement-page__card"
                  }
                  role="button"
                  tabIndex={0}
                  aria-label={`${cardTitle} の詳細を開く`}
                  onClick={() =>
                    handleOpenDecisionNotification(
                      notification,
                    )
                  }
                  onKeyDown={(event) => {
                    if (
                      event.key === "Enter" ||
                      event.key === " "
                    ) {
                      event.preventDefault();
                      handleOpenDecisionNotification(
                        notification,
                      );
                    }
                  }}
                >
                  <div className="announcement-page__card-head">
                    <div className="announcement-page__card-meta">
                      <span className="announcement-page__token">
                        {cardLabel}
                      </span>

                      <time
                        className="announcement-page__date"
                        dateTime={
                          item.occurredAt ||
                          undefined
                        }
                      >
                        {occurredAtLabel}
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
                    {cardTitle}
                  </h2>
                </article>
              );
            })}
          </div>
        ) : null}
      </section>
    </Layout>
  );
}