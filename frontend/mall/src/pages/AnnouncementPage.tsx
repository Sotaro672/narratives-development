// frontend/mall/src/pages/AnnouncementPage.tsx

import { useCallback, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import Layout from "../components/layout/Layout";
import { formatDateTime } from "../components/utils/date";

import {
  useAnnouncementsQuery,
  useMarkAnnouncementReadMutation,
} from "../features/announcement/hooks/useAnnouncementsQuery";
import {
  useMarkReportDecisionNotificationReadMutation,
  useReportDecisionNotificationsQuery,
} from "../features/notification/hooks/useReportDecisionNotificationsQuery";
import type { ReportDecisionNotification } from "../features/notification/infrastructure/reportDecisionNotificationApi";
import type { AnnouncementListItem } from "../features/shared/types/announcements";
import {
  getReportReasonLabel,
  type ReportTargetType,
} from "../features/shared/types/report";

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

type NotificationFeedItem =
  | AnnouncementFeedItem
  | ReportDecisionFeedItem;

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
    case "TOKEN_BLUEPRINT_COMMENT":
      return "トークンコメント";
    case "AVATAR":
      return "アバター";
    default:
      return "投稿内容";
  }
}

function getDecisionBody(
  notification: ReportDecisionNotification,
): string {
  if (notification.notificationKind === "TARGET_ENFORCEMENT") {
    switch (notification.targetType) {
      case "PRODUCT_BLUEPRINT_REVIEW":
        return "運営の裁定により、あなたの商品レビューを削除しました。";
      case "AVATAR":
        return "運営の裁定により、再販サービスの利用を停止しました。";
      case "TOKEN_BLUEPRINT_COMMENT":
        return "運営の裁定により、あなたのトークンコメントを削除しました。";
      default:
        return "運営の裁定により、対象コンテンツに措置を行いました。";
    }
  }

  if (notification.targetType === "AVATAR") {
    switch (notification.decisionStatus) {
      case "REMOVED":
        return "通報いただいた内容を確認し、対象アバターの再販サービス利用を停止しました。";
      case "KEPT":
        return "通報いただいた内容を確認しました。審査の結果、対象アバターへの変更は行いませんでした。";
      default:
        return "通報いただいた内容の確認が完了しました。";
    }
  }

  switch (notification.decisionStatus) {
    case "REMOVED":
      return "通報いただいた内容を確認し、対象コンテンツを削除しました。";
    case "KEPT":
      return "通報いただいた内容を確認しました。審査の結果、対象コンテンツを維持します。";
    default:
      return "通報いただいた内容の確認が完了しました。";
  }
}

function getDecisionStatusLabel(
  notification: ReportDecisionNotification,
): string {
  if (notification.targetType === "AVATAR") {
    switch (notification.decisionStatus) {
      case "REMOVED":
        return "再販利用停止";
      case "KEPT":
        return "変化なし";
      default:
        return notification.decisionStatus;
    }
  }

  switch (notification.decisionStatus) {
    case "REMOVED":
      return "削除";
    case "KEPT":
      return "維持";
    default:
      return notification.decisionStatus;
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

  const [navigatingId, setNavigatingId] = useState<string | null>(null);
  const [markingDecisionId, setMarkingDecisionId] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string>("");

  const announcementsQuery = useAnnouncementsQuery({
    page: 1,
    perPage: 100,
  });

  const decisionNotificationsQuery = useReportDecisionNotificationsQuery({
    page: 1,
    perPage: 100,
  });

  const markAnnouncementReadMutation = useMarkAnnouncementReadMutation();
  const markDecisionReadMutation = useMarkReportDecisionNotificationReadMutation();

  const announcements = useMemo(
    () => announcementsQuery.data?.items ?? [],
    [announcementsQuery.data?.items],
  );

  const decisionNotifications = useMemo(
    () => decisionNotificationsQuery.data?.items ?? [],
    [decisionNotificationsQuery.data?.items],
  );

  const items = useMemo<NotificationFeedItem[]>(() => {
    const announcementItems: AnnouncementFeedItem[] = announcements.map(
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
      decisionNotifications.map((notification) => ({
        kind: "reportDecision",
        key: `reportDecision:${notification.id}`,
        occurredAt:
          notification.decidedAt ||
          notification.createdAt,
        notification,
      }));

    return [...announcementItems, ...decisionItems].sort(
      (left, right) =>
        toTimestamp(right.occurredAt) -
        toTimestamp(left.occurredAt),
    );
  }, [announcements, decisionNotifications]);

  const loading =
    announcementsQuery.isPending ||
    decisionNotificationsQuery.isPending;

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

  const error =
    actionError ||
    announcementQueryError ||
    decisionQueryError;

  const handleOpenAnnouncement = useCallback(
    async (item: AnnouncementListItem) => {
      if (!item.id || navigatingId || markingDecisionId) {
        return;
      }

      setNavigatingId(item.id);
      setActionError("");

      let announcementForNavigation = item;

      try {
        if (item.isRead === false) {
          const readAt = item.readAt ?? new Date().toISOString();

          await markAnnouncementReadMutation.mutateAsync(item.id);

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
      markingDecisionId,
      navigate,
      navigatingId,
    ],
  );

  const handleOpenDecisionNotification = useCallback(
    async (notification: ReportDecisionNotification) => {
      if (
        !notification.id ||
        notification.isRead ||
        markingDecisionId ||
        navigatingId
      ) {
        return;
      }

      setMarkingDecisionId(notification.id);
      setActionError("");

      try {
        await markDecisionReadMutation.mutateAsync(notification.id);
      } catch (caught) {
        setActionError(
          caught instanceof Error
            ? caught.message
            : "通報結果通知の既読化に失敗しました",
        );
      } finally {
        setMarkingDecisionId(null);
      }
    },
    [
      markDecisionReadMutation,
      markingDecisionId,
      navigatingId,
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
          <div className="announcement-page__error" role="alert">
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
                const isUnread = announcement.isRead === false;

                const tokenLabel =
                  announcement.tokenName ||
                  announcement.targetToken ||
                  "お知らせ";

                const occurredAtLabel = formatDateTime(item.occurredAt);
                const isNavigating = navigatingId === announcement.id;

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
                    aria-busy={isNavigating}
                    onClick={() => void handleOpenAnnouncement(announcement)}
                    onKeyDown={(event) => {
                      if (event.key === "Enter" || event.key === " ") {
                        event.preventDefault();
                        void handleOpenAnnouncement(announcement);
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
                          dateTime={item.occurredAt || undefined}
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

                    {Array.isArray(announcement.attachmentFiles) &&
                    announcement.attachmentFiles.length > 0 ? (
                      <div className="announcement-page__attachments">
                        添付 {announcement.attachmentFiles.length} 件
                      </div>
                    ) : Array.isArray(announcement.attachments) &&
                      announcement.attachments.length > 0 ? (
                      <div className="announcement-page__attachments">
                        添付 {announcement.attachments.length} 件
                      </div>
                    ) : null}
                  </article>
                );
              }

              const notification = item.notification;
              const isUnread = notification.isRead === false;
              const isMarkingRead = markingDecisionId === notification.id;
              const targetLabel = getReportTargetLabel(notification.targetType);
              const decisionStatusLabel = getDecisionStatusLabel(notification);
              const occurredAtLabel = formatDateTime(item.occurredAt);
              const body = getDecisionBody(notification);
              const cardLabel = getDecisionCardLabel(
                notification,
                targetLabel,
              );
              const cardTitle = getDecisionCardTitle(notification);

              const isReporterDecision =
                notification.notificationKind === "REPORTER_DECISION";

              const reportReasonLabel = isReporterDecision
                ? getReportReasonLabel(notification.reportReason)
                : "";

              return (
                <article
                  key={item.key}
                  className={
                    isUnread
                      ? "announcement-page__card announcement-page__card--unread"
                      : "announcement-page__card"
                  }
                  role={isUnread ? "button" : undefined}
                  tabIndex={isUnread ? 0 : undefined}
                  aria-label={
                    isUnread
                      ? notification.notificationKind === "TARGET_ENFORCEMENT"
                        ? "運営からの措置通知を既読にする"
                        : "通報結果通知を既読にする"
                      : undefined
                  }
                  aria-busy={isMarkingRead}
                  onClick={() => {
                    if (isUnread) {
                      void handleOpenDecisionNotification(notification);
                    }
                  }}
                  onKeyDown={(event) => {
                    if (
                      !isUnread ||
                      (event.key !== "Enter" && event.key !== " ")
                    ) {
                      return;
                    }

                    event.preventDefault();
                    void handleOpenDecisionNotification(notification);
                  }}
                >
                  <div className="announcement-page__card-head">
                    <div className="announcement-page__card-meta">
                      <span className="announcement-page__token">
                        {cardLabel}
                      </span>

                      <time
                        className="announcement-page__date"
                        dateTime={item.occurredAt || undefined}
                      >
                        {occurredAtLabel}
                      </time>
                    </div>

                    {isUnread ? (
                      <span className="announcement-page__unread-badge">
                        {isMarkingRead ? "既読処理中" : "未読"}
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

                  <div className="announcement-page__detail-content">
                    {body}
                  </div>

                  {isReporterDecision ? (
                    <>
                      <div className="announcement-page__attachments">
                        通報理由: {reportReasonLabel}
                      </div>

                      {notification.reportDetail ? (
                        <div className="announcement-page__attachments">
                          通報詳細: {notification.reportDetail}
                        </div>
                      ) : null}
                    </>
                  ) : null}

                  <div className="announcement-page__attachments">
                    審査結果: {decisionStatusLabel}
                  </div>

                  {notification.decisionReason ? (
                    <div className="announcement-page__attachments">
                      審査理由: {notification.decisionReason}
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