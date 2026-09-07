// frontend/console/shell/src/pages/notificationPage.tsx

import { useMemo, type KeyboardEvent } from "react";

import { useReportDecisionNotifications } from "../features/notification/presentation/hooks/useReportDecisionNotifications";
import {
  toReportDecisionNotificationViewModels,
  type ReportDecisionNotificationViewModel,
} from "../features/notification/presentation/model/reportDecisionNotification";
import List from "../layout/List/List";
import { safeDateTimeLabelJa } from "../shared/util/dateJa";

import "../styles/notification.css";

export default function NotificationPage() {
  const {
    notifications,
    loading,
    error,
    markingReadId,
    reload,
    markRead,
  } = useReportDecisionNotifications({
    page: 1,
    perPage: 100,
  });

  const items = useMemo(
    () => toReportDecisionNotificationViewModels(notifications),
    [notifications],
  );

  const handleNotificationClick = async (
    item: ReportDecisionNotificationViewModel,
  ) => {
    if (item.isRead || markingReadId !== null) {
      return;
    }

    await markRead(item.id);
  };

  const handleNotificationKeyDown = (
    event: KeyboardEvent<HTMLTableRowElement>,
    item: ReportDecisionNotificationViewModel,
  ) => {
    if (item.isRead || markingReadId !== null) {
      return;
    }

    if (event.key !== "Enter" && event.key !== " ") {
      return;
    }

    event.preventDefault();
    void handleNotificationClick(item);
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
        {items.map((item) => {
          const isUnread = !item.isRead;
          const isMarkingRead = markingReadId === item.id;

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

          return (
            <tr
              key={item.id}
              role={isUnread ? "button" : undefined}
              tabIndex={isUnread ? 0 : undefined}
              aria-label={
                isUnread
                  ? `${item.title}を既読にする`
                  : undefined
              }
              aria-busy={isMarkingRead ? true : undefined}
              className={rowClassName}
              onClick={() => {
                if (isUnread) {
                  void handleNotificationClick(item);
                }
              }}
              onKeyDown={(event) =>
                handleNotificationKeyDown(event, item)
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

              <td className="notification-page__content-cell">
                <div className="notification-page__content">
                  <span className="notification-page__category">
                    {item.category}
                  </span>

                  <strong className="notification-page__title">
                    {item.title}
                  </strong>

                  <span className="notification-page__body">
                    {item.body}
                  </span>

                  {item.decisionReason ? (
                    <span className="notification-page__detail">
                      審査理由: {item.decisionReason}
                    </span>
                  ) : null}
                </div>
              </td>

              <td className="notification-page__target">
                {item.targetLabel}
              </td>

              <td className="notification-page__reason-cell">
                <div className="notification-page__reason">
                  <span>{item.reportReasonLabel}</span>

                  {item.reportDetail ? (
                    <span className="notification-page__detail">
                      {item.reportDetail}
                    </span>
                  ) : null}
                </div>
              </td>

              <td className="notification-page__decision">
                {item.decisionStatusLabel}
              </td>

              <td className="notification-page__date">
                {safeDateTimeLabelJa(item.occurredAt, "")}
              </td>
            </tr>
          );
        })}
      </List>
    </div>
  );
}