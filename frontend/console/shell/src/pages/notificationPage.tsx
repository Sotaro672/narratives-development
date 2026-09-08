// frontend/console/shell/src/pages/notificationPage.tsx

import { useMemo, type KeyboardEvent } from "react";

import { useNewsNotifications } from "../features/notification/presentation/hooks/useNewsNotifications";
import { useReportDecisionNotifications } from "../features/notification/presentation/hooks/useReportDecisionNotifications";
import {
  toReportDecisionNotificationViewModels,
  type ReportDecisionNotificationViewModel,
} from "../features/notification/presentation/model/reportDecisionNotification";
import List from "../layout/List/List";
import type { News } from "../shared/types/news";
import { safeDateTimeLabelJa } from "../shared/util/dateJa";

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
  const {
    notifications: reportDecisionNotifications,
    loading: reportDecisionLoading,
    error: reportDecisionError,
    markingReadId: reportDecisionMarkingReadId,
    reload: reloadReportDecisionNotifications,
    markRead: markReportDecisionRead,
  } = useReportDecisionNotifications({
    page: 1,
    perPage: 100,
  });

  const {
    notifications: newsNotifications,
    loading: newsLoading,
    error: newsError,
    markingReadId: newsMarkingReadId,
    reload: reloadNewsNotifications,
    markRead: markNewsRead,
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

  const handleNotificationClick = async (feedItem: NotificationFeedItem) => {
    if (feedItem.kind === "news") {
      if (feedItem.item.isRead || newsMarkingReadId !== null) {
        return;
      }

      await markNewsRead(feedItem.item.id);
      return;
    }

    if (
      feedItem.item.isRead ||
      reportDecisionMarkingReadId !== null
    ) {
      return;
    }

    await markReportDecisionRead(feedItem.item.id);
  };

  const handleNotificationKeyDown = (
    event: KeyboardEvent<HTMLTableRowElement>,
    feedItem: NotificationFeedItem,
  ) => {
    if (feedItem.item.isRead) {
      return;
    }

    if (
      feedItem.kind === "news"
        ? newsMarkingReadId !== null
        : reportDecisionMarkingReadId !== null
    ) {
      return;
    }

    if (event.key !== "Enter" && event.key !== " ") {
      return;
    }

    event.preventDefault();
    void handleNotificationClick(feedItem);
  };

  const reload = async () => {
    await Promise.all([
      reloadReportDecisionNotifications(),
      reloadNewsNotifications(),
    ]);
  };

  const headers = [
    "状態",
    "通知内容",
    "対象",
    "通報理由",
    "審査結果",
    "通知日時",
  ];

  return (
    <div className="notification-page">
      {error ? (
        <div className="notification-page__error" role="alert">
          {error}
        </div>
      ) : null}

      <List
        title="通知"
        headerCells={headers}
        showResetButton
        isResetting={loading}
        onReset={() => {
          void reload();
        }}
      >
        {items.map((feedItem) => {
          const isNews = feedItem.kind === "news";
          const item = feedItem.item;
          const isUnread = !item.isRead;
          const isMarkingRead = isNews
            ? newsMarkingReadId === item.id
            : reportDecisionMarkingReadId === item.id;

          const rowClassName = [
            "notification-page__row",
            isUnread ? "notification-page__row--unread" : "",
            isMarkingRead ? "notification-page__row--busy" : "",
          ]
            .filter(Boolean)
            .join(" ");

          const statusClassName = [
            "notification-page__status",
            isUnread
              ? "notification-page__status--unread"
              : "notification-page__status--read",
          ]
            .filter(Boolean)
            .join(" ");

          const title = feedItem.item.title;

          return (
            <tr
              key={feedItem.key}
              role={isUnread ? "button" : undefined}
              tabIndex={isUnread ? 0 : undefined}
              aria-label={
                isUnread ? `${title}を既読にする` : undefined
              }
              aria-busy={isMarkingRead ? true : undefined}
              className={rowClassName}
              onClick={() => {
                if (isUnread) {
                  void handleNotificationClick(feedItem);
                }
              }}
              onKeyDown={(event) =>
                handleNotificationKeyDown(event, feedItem)
              }
            >
              <td className="notification-page__status-cell">
                <span className={statusClassName}>
                  {isMarkingRead
                    ? "既読処理中"
                    : isUnread
                      ? "未読"
                      : "既読"}
                </span>
              </td>

              {feedItem.kind === "news" ? (
                <>
                  <td className="notification-page__content-cell">
                    <div className="notification-page__content">
                      <span className="notification-page__category">
                        システム通知
                      </span>

                      {feedItem.item.image ? (
                        <div className="notification-page__news-image-wrap">
                          <img
                            className="notification-page__news-image"
                            src={feedItem.item.image.fileUrl}
                            alt={
                              feedItem.item.image.alt ||
                              feedItem.item.title
                            }
                            loading="lazy"
                          />
                        </div>
                      ) : null}

                      <strong className="notification-page__title">
                        {feedItem.item.title}
                      </strong>

                      <span className="notification-page__body">
                        {feedItem.item.body}
                      </span>
                    </div>
                  </td>

                  <td className="notification-page__target">
                    システム
                  </td>

                  <td className="notification-page__reason-cell">
                    -
                  </td>

                  <td className="notification-page__decision">
                    -
                  </td>

                  <td className="notification-page__date">
                    {safeDateTimeLabelJa(feedItem.occurredAt, "")}
                  </td>
                </>
              ) : (
                <>
                  <td className="notification-page__content-cell">
                    <div className="notification-page__content">
                      <span className="notification-page__category">
                        {feedItem.item.category}
                      </span>

                      <strong className="notification-page__title">
                        {feedItem.item.title}
                      </strong>

                      <span className="notification-page__body">
                        {feedItem.item.body}
                      </span>

                      {feedItem.item.decisionReason ? (
                        <span className="notification-page__detail">
                          審査理由: {feedItem.item.decisionReason}
                        </span>
                      ) : null}
                    </div>
                  </td>

                  <td className="notification-page__target">
                    {feedItem.item.targetLabel}
                  </td>

                  <td className="notification-page__reason-cell">
                    <div className="notification-page__reason">
                      <span>{feedItem.item.reportReasonLabel}</span>

                      {feedItem.item.reportDetail ? (
                        <span className="notification-page__detail">
                          {feedItem.item.reportDetail}
                        </span>
                      ) : null}
                    </div>
                  </td>

                  <td className="notification-page__decision">
                    {feedItem.item.decisionStatusLabel}
                  </td>

                  <td className="notification-page__date">
                    {safeDateTimeLabelJa(feedItem.occurredAt, "")}
                  </td>
                </>
              )}
            </tr>
          );
        })}
      </List>
    </div>
  );
}