// frontend/mall/src/features/announcement/presentation/components/AnnouncementListPane.tsx

import { useCallback, useMemo } from "react";
import {
  Bell,
  Newspaper,
  ShieldCheck,
  TriangleAlert,
} from "lucide-react";
import {
  useLocation,
  useNavigate,
} from "react-router-dom";

import Badge from "../../../../components/ui/Badge";
import List, {
  ListRow,
} from "../../../../components/ui/List";
import StatePanel from "../../../../components/ui/StatePanel";

import { useAnnouncementsQuery } from "../hooks/useAnnouncementsQuery";
import { useNewsQuery } from "../../../news/hooks/useNewsQuery";
import type { ReportDecisionNotification } from "../../../notification/infrastructure/reportDecisionNotificationApi";
import { useReportDecisionNotificationsQuery } from "../../../notification/presentation/hooks/useReportDecisionNotificationsQuery";
import type { AnnouncementListItem } from "../../../shared/types/announcements";
import type { News } from "../../../shared/types/news";
import type { ReportTargetType } from "../../../shared/types/report";

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

type SelectedNotification =
  | {
      kind: "announcement";
      id: string;
    }
  | {
      kind: "reportDecision";
      id: string;
    }
  | {
      kind: "news";
      id: string;
    }
  | null;

function toTimestamp(
  value: string | null | undefined,
): number {
  if (!value) {
    return 0;
  }

  const timestamp = Date.parse(value);
  return Number.isFinite(timestamp)
    ? timestamp
    : 0;
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
  if (
    notification.notificationKind ===
    "TARGET_ENFORCEMENT"
  ) {
    return `運営からのお知らせ・${targetLabel}`;
  }

  return `通報結果・${targetLabel}`;
}

function getDecisionCardTitle(
  notification: ReportDecisionNotification,
): string {
  if (
    notification.notificationKind ===
    "TARGET_ENFORCEMENT"
  ) {
    return "運営による措置のお知らせ";
  }

  return "通報内容の確認が完了しました";
}

function getAnnouncementPreview(
  announcement: AnnouncementListItem,
): string {
  const content =
    announcement.content?.trim();

  if (content) {
    return content;
  }

  const attachmentCount =
    Array.isArray(
      announcement.attachmentFiles,
    ) &&
    announcement.attachmentFiles.length > 0
      ? announcement.attachmentFiles.length
      : Array.isArray(
            announcement.attachments,
          )
        ? announcement.attachments.length
        : 0;

  if (attachmentCount > 0) {
    return `添付 ${attachmentCount} 件`;
  }

  return "お知らせ";
}

export default function AnnouncementListPane() {
  const location = useLocation();
  const navigate = useNavigate();

  const announcementsQuery =
    useAnnouncementsQuery({
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

  const announcements = useMemo(
    () =>
      announcementsQuery.data?.items ?? [],
    [announcementsQuery.data?.items],
  );

  const decisionNotifications = useMemo(
    () =>
      decisionNotificationsQuery.data?.items ??
      [],
    [
      decisionNotificationsQuery.data?.items,
    ],
  );

  const news = useMemo(
    () => newsQuery.data?.items ?? [],
    [newsQuery.data?.items],
  );

  const items =
    useMemo<NotificationFeedItem[]>(() => {
      const announcementItems: AnnouncementFeedItem[] =
        announcements.map(
          (announcement) => ({
            kind: "announcement",
            key: `announcement:${announcement.id}`,
            occurredAt:
              announcement.publishedAt ??
              announcement.createdAt ??
              "",
            announcement,
          }),
        );

      const decisionItems: ReportDecisionFeedItem[] =
        decisionNotifications.map(
          (notification) => ({
            kind: "reportDecision",
            key: `reportDecision:${notification.id}`,
            occurredAt:
              notification.decidedAt ||
              notification.createdAt,
            notification,
          }),
        );

      const newsItems: NewsFeedItem[] =
        news.map((newsItem) => ({
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
    decisionNotificationsQuery.error instanceof
    Error
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

  const error =
    announcementQueryError ||
    decisionQueryError ||
    newsQueryError;

  const selectedNotification =
    getSelectedNotification(
      location.pathname,
    );

  const handleOpenAnnouncement =
    useCallback(
      (item: AnnouncementListItem) => {
        const announcementId =
          item.id?.trim() ?? "";

        if (!announcementId) {
          return;
        }

        navigate(
          `/announcements/${encodeURIComponent(
            announcementId,
          )}`,
          {
            state: {
              announcement: item,
            },
          },
        );
      },
      [navigate],
    );

  const handleOpenDecisionNotification =
    useCallback(
      (
        notification: ReportDecisionNotification,
      ) => {
        const notificationId =
          notification.id?.trim() ?? "";

        if (!notificationId) {
          return;
        }

        navigate(
          `/announcements/report-decisions/${encodeURIComponent(
            notificationId,
          )}`,
          {
            state: {
              reportDecisionNotification:
                notification,
            },
          },
        );
      },
      [navigate],
    );

  const handleOpenNews = useCallback(
    (newsItem: News) => {
      const newsId =
        newsItem.id?.trim() ?? "";

      if (!newsId) {
        return;
      }

      navigate(
        `/announcements/news/${encodeURIComponent(
          newsId,
        )}`,
        {
          state: {
            news: newsItem,
          },
        },
      );
    },
    [navigate],
  );

  return (
    <section className="page-section content-page-section announcement-page announcement-page--list announcement-page--workspace">
      {error ? (
        <StatePanel
          variant="error"
          icon={
            <TriangleAlert
              size={28}
              strokeWidth={1.8}
            />
          }
          title="通知を取得できませんでした"
          description={error}
        />
      ) : null}

      {!error && loading ? (
        <StatePanel
          variant="loading"
          title="通知を読み込んでいます"
          description="最新のお知らせを確認しています。"
        />
      ) : null}

      {!error &&
      !loading &&
      items.length === 0 ? (
        <StatePanel
          variant="empty"
          icon={
            <Bell
              size={28}
              strokeWidth={1.8}
            />
          }
          title="現在、通知はありません"
          description="新しいお知らせが届くと、ここに表示されます。"
        />
      ) : null}

      {!error &&
      !loading &&
      items.length > 0 ? (
        <List
          className="announcement-page__list"
          aria-label="通知一覧"
        >
          {items.map((item) => {
            if (
              item.kind === "announcement"
            ) {
              const announcement =
                item.announcement;
              const isUnread =
                announcement.isRead === false;
              const tokenLabel =
                announcement.tokenName ||
                announcement.targetToken ||
                "お知らせ";

              return (
                <ListRow
                  key={item.key}
                  attention={isUnread}
                  selected={isSelectedNotification(
                    selectedNotification,
                    "announcement",
                    announcement.id,
                  )}
                  ariaLabel={`${announcement.title} の詳細を開く`}
                  onClick={() => {
                    handleOpenAnnouncement(
                      announcement,
                    );
                  }}
                  leading={
                    announcement.tokenIcon ? (
                      <img
                        src={
                          announcement.tokenIcon
                        }
                        alt=""
                      />
                    ) : (
                      <Bell
                        size={20}
                        strokeWidth={1.8}
                      />
                    )
                  }
                  title={announcement.title}
                  subLabel={tokenLabel}
                  dateValue={item.occurredAt}
                  preview={getAnnouncementPreview(
                    announcement,
                  )}
                  meta={
                    <Badge
                      variant={
                        isUnread
                          ? "info"
                          : "neutral"
                      }
                      size="sm"
                    >
                      {isUnread
                        ? "未読"
                        : "既読"}
                    </Badge>
                  }
                />
              );
            }

            if (item.kind === "news") {
              const newsItem = item.news;
              const isUnread =
                newsItem.isRead === false;

              return (
                <ListRow
                  key={item.key}
                  attention={isUnread}
                  selected={isSelectedNotification(
                    selectedNotification,
                    "news",
                    newsItem.id,
                  )}
                  ariaLabel={`${newsItem.title} の詳細を開く`}
                  onClick={() => {
                    handleOpenNews(newsItem);
                  }}
                  leading={
                    <Newspaper
                      size={20}
                      strokeWidth={1.8}
                    />
                  }
                  title={newsItem.title}
                  subLabel="システム通知"
                  dateValue={item.occurredAt}
                  preview={
                    newsItem.body?.trim() ||
                    "システム通知"
                  }
                  meta={
                    <Badge
                      variant={
                        isUnread
                          ? "info"
                          : "neutral"
                      }
                      size="sm"
                    >
                      {isUnread
                        ? "未読"
                        : "既読"}
                    </Badge>
                  }
                />
              );
            }

            const notification =
              item.notification;
            const isUnread =
              notification.isRead === false;
            const targetLabel =
              getReportTargetLabel(
                notification.targetType,
              );
            const cardLabel =
              getDecisionCardLabel(
                notification,
                targetLabel,
              );
            const cardTitle =
              getDecisionCardTitle(
                notification,
              );

            return (
              <ListRow
                key={item.key}
                attention={isUnread}
                selected={isSelectedNotification(
                  selectedNotification,
                  "reportDecision",
                  notification.id,
                )}
                ariaLabel={`${cardTitle} の詳細を開く`}
                onClick={() => {
                  handleOpenDecisionNotification(
                    notification,
                  );
                }}
                leading={
                  <ShieldCheck
                    size={20}
                    strokeWidth={1.8}
                  />
                }
                title={cardTitle}
                subLabel={cardLabel}
                dateValue={item.occurredAt}
                preview={
                  notification.decisionReason?.trim() ||
                  targetLabel
                }
                meta={
                  <Badge
                    variant={
                      isUnread
                        ? "info"
                        : "neutral"
                    }
                    size="sm"
                  >
                    {isUnread
                      ? "未読"
                      : "既読"}
                  </Badge>
                }
              />
            );
          })}
        </List>
      ) : null}
    </section>
  );
}

function getSelectedNotification(
  pathname: string,
): SelectedNotification {
  const segments = pathname
    .split("/")
    .filter(Boolean);

  if (segments[0] !== "announcements") {
    return null;
  }

  if (
    segments[1] === "news" &&
    segments[2]
  ) {
    return {
      kind: "news",
      id: decodePathSegment(segments[2]),
    };
  }

  if (
    segments[1] === "report-decisions" &&
    segments[2]
  ) {
    return {
      kind: "reportDecision",
      id: decodePathSegment(segments[2]),
    };
  }

  if (
    segments.length === 2 &&
    segments[1]
  ) {
    return {
      kind: "announcement",
      id: decodePathSegment(segments[1]),
    };
  }

  return null;
}

function isSelectedNotification(
  selectedNotification: SelectedNotification,
  kind:
    | "announcement"
    | "reportDecision"
    | "news",
  id: string,
): boolean {
  return (
    selectedNotification?.kind === kind &&
    selectedNotification.id === id
  );
}

function decodePathSegment(
  value: string,
): string {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}