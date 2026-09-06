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
  useMarkReviewReportDecisionNotificationReadMutation,
  useReviewReportDecisionNotificationsQuery,
} from "../features/notification/hooks/useReviewReportDecisionNotificationsQuery";
import type { ReviewReportDecisionNotification } from "../features/notification/infrastructure/reviewReportDecisionNotificationApi";
import type { AnnouncementListItem } from "../features/shared/types/announcements";
import {
  getReviewReportReasonLabel,
  type ReviewReportTargetType,
} from "../features/shared/types/reviewReport";

import "../styles/page-layout.css";
import "../styles/announcement-page.css";

type AnnouncementFeedItem = {
  kind: "announcement";
  key: string;
  occurredAt: string;
  announcement: AnnouncementListItem;
};

type ReviewReportDecisionFeedItem = {
  kind: "reviewReportDecision";
  key: string;
  occurredAt: string;
  notification: ReviewReportDecisionNotification;
};

type NotificationFeedItem =
  | AnnouncementFeedItem
  | ReviewReportDecisionFeedItem;

function toTimestamp(value: string | null | undefined): number {
  if (!value) {
    return 0;
  }

  const timestamp = Date.parse(value);
  return Number.isFinite(timestamp) ? timestamp : 0;
}

function getReviewReportTargetLabel(
  targetType: ReviewReportTargetType,
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
  notification: ReviewReportDecisionNotification,
): string {
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
      return "通報いただいた内容を確認し、対象コンテンツを非表示にしました。";
    case "KEPT":
      return "通報いただいた内容を確認しました。審査の結果、掲載を継続します。";
    default:
      return "通報いただいた内容の確認が完了しました。";
  }
}

function getDecisionStatusLabel(
  notification: ReviewReportDecisionNotification,
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
      return "非表示";
    case "KEPT":
      return "掲載継続";
    default:
      return notification.decisionStatus;
  }
}

export default function AnnouncementPage() {
  const navigate = useNavigate();

  const [navigatingId, setNavigatingId] = useState<string | null>(null);
  const [markingDecisionId, setMarkingDecisionId] = useState<string | null>(
    null,
  );
  const [actionError, setActionError] = useState<string>("");

  const announcementsQuery = useAnnouncementsQuery({
    page: 1,
    perPage: 100,
  });

  const decisionNotificationsQuery =
    useReviewReportDecisionNotificationsQuery({
      page: 1,
      perPage: 100,
    });

  const markAnnouncementReadMutation =
    useMarkAnnouncementReadMutation();

  const markDecisionReadMutation =
    useMarkReviewReportDecisionNotificationReadMutation();

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

    const decisionItems: ReviewReportDecisionFeedItem[] =
      decisionNotifications.map((notification) => ({
        kind: "reviewReportDecision",
        key: `reviewReportDecision:${notification.id}`,
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
      markingDecisionId,
      navigate,
      navigatingId,
    ],
  );

  const handleOpenDecisionNotification = useCallback(
    async (
      notification: ReviewReportDecisionNotification,
    ) => {
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
        await markDecisionReadMutation.mutateAsync(
          notification.id,
        );
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

                const isNavigating =
                  navigatingId === announcement.id;

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
                    onClick={() =>
                      void handleOpenAnnouncement(
                        announcement,
                      )
                    }
                    onKeyDown={(event) => {
                      if (
                        event.key === "Enter" ||
                        event.key === " "
                      ) {
                        event.preventDefault();
                        void handleOpenAnnouncement(
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
                          announcement
                            .attachments.length
                        }{" "}
                        件
                      </div>
                    ) : null}
                  </article>
                );
              }

              const notification =
                item.notification;

              const isUnread =
                notification.isRead === false;

              const isMarkingRead =
                markingDecisionId ===
                notification.id;

              const targetLabel =
                getReviewReportTargetLabel(
                  notification.targetType,
                );

              const reportReasonLabel =
                getReviewReportReasonLabel(
                  notification.reportReason,
                );

              const decisionStatusLabel =
                getDecisionStatusLabel(
                  notification,
                );

              const occurredAtLabel =
                formatDateTime(
                  item.occurredAt,
                );

              const body =
                getDecisionBody(
                  notification,
                );

              return (
                <article
                  key={item.key}
                  className={
                    isUnread
                      ? "announcement-page__card announcement-page__card--unread"
                      : "announcement-page__card"
                  }
                  role={
                    isUnread
                      ? "button"
                      : undefined
                  }
                  tabIndex={
                    isUnread ? 0 : undefined
                  }
                  aria-label={
                    isUnread
                      ? "通報結果通知を既読にする"
                      : undefined
                  }
                  aria-busy={isMarkingRead}
                  onClick={() => {
                    if (isUnread) {
                      void handleOpenDecisionNotification(
                        notification,
                      );
                    }
                  }}
                  onKeyDown={(event) => {
                    if (
                      !isUnread ||
                      (
                        event.key !== "Enter" &&
                        event.key !== " "
                      )
                    ) {
                      return;
                    }

                    event.preventDefault();
                    void handleOpenDecisionNotification(
                      notification,
                    );
                  }}
                >
                  <div className="announcement-page__card-head">
                    <div className="announcement-page__card-meta">
                      <span className="announcement-page__token">
                        通報結果・{targetLabel}
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
                        {isMarkingRead
                          ? "既読処理中"
                          : "未読"}
                      </span>
                    ) : (
                      <span className="announcement-page__read-badge">
                        既読
                      </span>
                    )}
                  </div>

                  <h2 className="announcement-page__card-title">
                    通報内容の確認が完了しました
                  </h2>

                  <div className="announcement-page__detail-content">
                    {body}
                  </div>

                  <div className="announcement-page__attachments">
                    通報理由: {reportReasonLabel}
                  </div>

                  {notification.reportDetail ? (
                    <div className="announcement-page__attachments">
                      通報詳細:{" "}
                      {notification.reportDetail}
                    </div>
                  ) : null}

                  <div className="announcement-page__attachments">
                    審査結果:{" "}
                    {decisionStatusLabel}
                  </div>

                  {notification.decisionReason ? (
                    <div className="announcement-page__attachments">
                      審査理由:{" "}
                      {notification.decisionReason}
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