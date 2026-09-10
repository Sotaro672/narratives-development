// frontend/console/shell/src/pages/notificationPage.tsx

import { useMemo, type KeyboardEvent } from "react";
import { useNavigate } from "react-router-dom";

import { useNewsNotifications } from "../features/notification/presentation/hooks/useNewsNotifications";
import { useReportDecisionNotifications } from "../features/notification/presentation/hooks/useReportDecisionNotifications";
import {
  toReportDecisionNotificationViewModels,
  type ReportDecisionNotificationViewModel,
} from "../features/notification/presentation/model/reportDecisionNotification";
import List from "../layout/List/List";
import type { News } from "../shared/types/news";

import "../styles/notification.css";

type ReportDecisionFeedItem = {
  kind: "reportDecision";
  key: string;
  occurredAt: string;
  item: ReportDecisionNotificationViewModel;
};

type NewsFeedItem = {
  kind: "news";
  key: string;
  occurredAt: string;
  item: News;
};

type NotificationFeedItem = ReportDecisionFeedItem | NewsFeedItem;

function toTimestamp(value: string | null | undefined): number {
  if (!value) {
    return 0;
  }

  const timestamp = Date.parse(value);
  return Number.isFinite(timestamp) ? timestamp : 0;
}

export default function NotificationPage() {
  const navigate = useNavigate();

  const {
    notifications: reportDecisionNotifications,
    loading: reportDecisionLoading,
    error: reportDecisionError,
    reload: reloadReportDecisionNotifications,
  } = useReportDecisionNotifications({
    page: 1,
    perPage: 100,
  });

  const {
    notifications: newsNotifications,
    loading: newsLoading,
    error: newsError,
    reload: reloadNewsNotifications,
  } = useNewsNotifications({
    page: 1,
    perPage: 100,
  });

  const reportDecisionItems = useMemo(
    () => toReportDecisionNotificationViewModels(reportDecisionNotifications),
    [reportDecisionNotifications],
  );

  const items = useMemo<NotificationFeedItem[]>(() => {
    const decisionItems: ReportDecisionFeedItem[] = reportDecisionItems.map(
      (item) => ({
        kind: "reportDecision",
        key: `reportDecision:${item.id}`,
        occurredAt: item.occurredAt,
        item,
      }),
    );

    const newsItems: NewsFeedItem[] = newsNotifications.map((item) => ({
      kind: "news",
      key: `news:${item.id}`,
      occurredAt: item.publishedAt || item.createdAt,
      item,
    }));

    return [...decisionItems, ...newsItems].sort(
      (left, right) =>
        toTimestamp(right.occurredAt) - toTimestamp(left.occurredAt),
    );
  }, [newsNotifications, reportDecisionItems]);

  const loading = reportDecisionLoading || newsLoading;
  const error = reportDecisionError ?? newsError ?? null;

  const openNotification = (feedItem: NotificationFeedItem) => {
    if (feedItem.kind === "news") {
      navigate(`/notifications/news/${encodeURIComponent(feedItem.item.id)}`);
      return;
    }

    navigate(
      `/notifications/report-decision/${encodeURIComponent(feedItem.item.id)}`,
    );
  };

  const handleNotificationKeyDown = (
    event: KeyboardEvent<HTMLTableRowElement>,
    feedItem: NotificationFeedItem,
  ) => {
    if (event.key !== "Enter" && event.key !== " ") {
      return;
    }

    event.preventDefault();
    openNotification(feedItem);
  };

  const reload = async () => {
    await Promise.all([
      reloadReportDecisionNotifications(),
      reloadNewsNotifications(),
    ]);
  };

  return (
    <div className="notification-page">
      {error ? (
        <div className="notification-page__error" role="alert">
          {error}
        </div>
      ) : null}

      <List
        title="通知"
        headerCells={["タイトル"]}
        showResetButton
        isResetting={loading}
        onReset={() => {
          void reload();
        }}
      >
        {items.map((feedItem) => {
          const title = feedItem.item.title;
          const isUnread = !feedItem.item.isRead;

          return (
            <tr
              key={feedItem.key}
              role="button"
              tabIndex={0}
              aria-label={`${title}を開く`}
              className={[
                "notification-page__row",
                isUnread ? "notification-page__row--unread" : "",
              ]
                .filter(Boolean)
                .join(" ")}
              onClick={() => {
                openNotification(feedItem);
              }}
              onKeyDown={(event) =>
                handleNotificationKeyDown(event, feedItem)
              }
            >
              <td className="notification-page__title-cell">
                <span
                  className={[
                    "notification-page__title",
                    isUnread ? "notification-page__title--unread" : "",
                  ]
                    .filter(Boolean)
                    .join(" ")}
                >
                  {title}
                </span>
              </td>
            </tr>
          );
        })}
      </List>
    </div>
  );
}