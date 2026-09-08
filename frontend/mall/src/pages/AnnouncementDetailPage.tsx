// frontend/mall/src/pages/AnnouncementDetailPage.tsx

import { useEffect, useMemo, useRef } from "react";
import { useLocation, useParams } from "react-router-dom";

import Layout from "../components/layout/Layout";
import { formatDateTime } from "../components/utils/date";

import {
  useAnnouncementsQuery,
  useMarkAnnouncementReadMutation,
} from "../features/announcement/hooks/useAnnouncementsQuery";
import {
  useMarkReportDecisionNotificationReadMutation,
  useReportDecisionNotificationQuery,
} from "../features/notification/hooks/useReportDecisionNotificationsQuery";
import type { ReportDecisionNotification } from "../features/notification/infrastructure/reportDecisionNotificationApi";
import type { AnnouncementListItem } from "../features/shared/types/announcements";
import {
  getReportReasonLabel,
  type ReportTargetType,
} from "../features/shared/types/report";

import "../styles/page-layout.css";
import "../styles/announcement-page.css";

type AnnouncementDetailLocationState = {
  announcement?: AnnouncementListItem;
  reportDecisionNotification?: ReportDecisionNotification;
};

function getReportTargetLabel(targetType: ReportTargetType): string {
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

function getDecisionCardTitle(notification: ReportDecisionNotification): string {
  if (notification.notificationKind === "TARGET_ENFORCEMENT") {
    return "運営による措置のお知らせ";
  }
  return "通報内容の確認が完了しました";
}

function getDecisionBody(notification: ReportDecisionNotification): string {
  if (notification.notificationKind === "TARGET_ENFORCEMENT") {
    switch (notification.targetType) {
      case "PRODUCT_BLUEPRINT_REVIEW":
        return "運営の裁定により、あなたの商品レビューを削除しました。";
      case "LIST":
        return "運営の裁定により、対象出品を停止しました。Listおよび登録済み画像は削除されていません。";
      case "TOKEN_BLUEPRINT":
        return "運営の裁定により、対象トークンをAMOL上で非表示にしました。オンチェーン上のトークンやメタデータは削除されていません。";
      case "AVATAR":
        return "運営の裁定により、再販サービスの利用を停止しました。";
      case "RESALE":
        return "運営の裁定により、対象の再販出品を停止しました。Resaleおよび登録済み画像は削除されていません。";
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

  if (notification.targetType === "LIST") {
    switch (notification.decisionStatus) {
      case "REMOVED":
        return "通報いただいた内容を確認し、対象出品を停止しました。";
      case "KEPT":
        return "通報いただいた内容を確認しました。審査の結果、対象出品の掲載を継続します。";
      default:
        return "通報いただいた内容の確認が完了しました。";
    }
  }

  if (notification.targetType === "RESALE") {
    switch (notification.decisionStatus) {
      case "REMOVED":
        return "通報いただいた内容を確認し、対象の再販出品を停止しました。";
      case "KEPT":
        return "通報いただいた内容を確認しました。審査の結果、対象の再販出品の掲載を継続します。";
      default:
        return "通報いただいた内容の確認が完了しました。";
    }
  }

  if (notification.targetType === "TOKEN_BLUEPRINT") {
    switch (notification.decisionStatus) {
      case "REMOVED":
        return "通報いただいた内容を確認し、対象トークンをAMOL上で非表示にしました。オンチェーン上のトークンやメタデータは削除されていません。";
      case "KEPT":
        return "通報いただいた内容を確認しました。審査の結果、対象トークンを維持します。";
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

function getDecisionStatusLabel(notification: ReportDecisionNotification): string {
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

  if (
    notification.targetType === "LIST" ||
    notification.targetType === "RESALE"
  ) {
    switch (notification.decisionStatus) {
      case "REMOVED":
        return "出品停止";
      case "KEPT":
        return "維持";
      default:
        return notification.decisionStatus;
    }
  }

  if (notification.targetType === "TOKEN_BLUEPRINT") {
    switch (notification.decisionStatus) {
      case "REMOVED":
        return "非表示";
      case "KEPT":
        return "維持";
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

export default function AnnouncementDetailPage() {
  const {
    announcementId = "",
    notificationId = "",
  } = useParams<{
    announcementId?: string;
    notificationId?: string;
  }>();

  const location = useLocation();
  const locationState = location.state as AnnouncementDetailLocationState | null;
  const stateAnnouncement = locationState?.announcement;
  const stateDecisionNotification = locationState?.reportDecisionNotification;

  const effectiveNotificationId = useMemo(
    () => notificationId.trim() || stateDecisionNotification?.id?.trim() || "",
    [notificationId, stateDecisionNotification?.id],
  );

  const isReportDecisionDetail = Boolean(effectiveNotificationId);

  const effectiveAnnouncementId = useMemo(() => {
    if (isReportDecisionDetail) {
      return "";
    }
    return announcementId.trim() || stateAnnouncement?.id?.trim() || "";
  }, [announcementId, isReportDecisionDetail, stateAnnouncement?.id]);

  const initialAnnouncement = useMemo(() => {
    if (
      !stateAnnouncement ||
      stateAnnouncement.id !== effectiveAnnouncementId
    ) {
      return null;
    }
    return stateAnnouncement;
  }, [effectiveAnnouncementId, stateAnnouncement]);

  const initialDecisionNotification = useMemo(() => {
    if (
      !stateDecisionNotification ||
      stateDecisionNotification.id !== effectiveNotificationId
    ) {
      return null;
    }
    return stateDecisionNotification;
  }, [effectiveNotificationId, stateDecisionNotification]);

  const announcementsQuery = useAnnouncementsQuery({
    page: 1,
    perPage: 100,
    enabled: !isReportDecisionDetail && Boolean(effectiveAnnouncementId),
  });

  const decisionNotificationQuery = useReportDecisionNotificationQuery(
    effectiveNotificationId,
    {
      enabled: isReportDecisionDetail && Boolean(effectiveNotificationId),
    },
  );

  const markAnnouncementReadMutation = useMarkAnnouncementReadMutation();
  const markDecisionReadMutation =
    useMarkReportDecisionNotificationReadMutation();

  const announcementFromQuery = useMemo(() => {
    if (!announcementsQuery.data) {
      return null;
    }
    return (
      announcementsQuery.data.items.find(
        (item) => item.id === effectiveAnnouncementId,
      ) ?? null
    );
  }, [announcementsQuery.data, effectiveAnnouncementId]);

  const announcement =
    !isReportDecisionDetail && announcementsQuery.data !== undefined
      ? announcementFromQuery
      : initialAnnouncement;

  const decisionNotification =
    isReportDecisionDetail && decisionNotificationQuery.data !== undefined
      ? decisionNotificationQuery.data
      : initialDecisionNotification;

  const markedAnnouncementReadRef = useRef<string>("");
  const markedDecisionReadRef = useRef<string>("");

  useEffect(() => {
    if (
      isReportDecisionDetail ||
      !effectiveAnnouncementId ||
      !announcement
    ) {
      return;
    }

    if (announcement.isRead === true) {
      markedAnnouncementReadRef.current = effectiveAnnouncementId;
      return;
    }

    if (markedAnnouncementReadRef.current === effectiveAnnouncementId) {
      return;
    }

    markedAnnouncementReadRef.current = effectiveAnnouncementId;
    markAnnouncementReadMutation.mutate(effectiveAnnouncementId);
  }, [
    announcement,
    effectiveAnnouncementId,
    isReportDecisionDetail,
    markAnnouncementReadMutation,
  ]);

  useEffect(() => {
    if (
      !isReportDecisionDetail ||
      !effectiveNotificationId ||
      !decisionNotification
    ) {
      return;
    }

    if (decisionNotification.isRead === true) {
      markedDecisionReadRef.current = effectiveNotificationId;
      return;
    }

    if (markedDecisionReadRef.current === effectiveNotificationId) {
      return;
    }

    markedDecisionReadRef.current = effectiveNotificationId;
    markDecisionReadMutation.mutate(effectiveNotificationId);
  }, [
    decisionNotification,
    effectiveNotificationId,
    isReportDecisionDetail,
    markDecisionReadMutation,
  ]);

  const announcementQueryError =
    !isReportDecisionDetail && announcementsQuery.error instanceof Error
      ? announcementsQuery.error.message
      : !isReportDecisionDetail && announcementsQuery.error
        ? "お知らせの取得に失敗しました"
        : "";

  const decisionQueryError =
    isReportDecisionDetail && decisionNotificationQuery.error instanceof Error
      ? decisionNotificationQuery.error.message
      : isReportDecisionDetail && decisionNotificationQuery.error
        ? "通報結果通知の取得に失敗しました"
        : "";

  const announcementMutationError =
    !isReportDecisionDetail &&
    markAnnouncementReadMutation.error instanceof Error
      ? markAnnouncementReadMutation.error.message
      : !isReportDecisionDetail && markAnnouncementReadMutation.error
        ? "お知らせの既読化に失敗しました"
        : "";

  const decisionMutationError =
    isReportDecisionDetail &&
    markDecisionReadMutation.error instanceof Error
      ? markDecisionReadMutation.error.message
      : isReportDecisionDetail && markDecisionReadMutation.error
        ? "通報結果通知の既読化に失敗しました"
        : "";

  const loading = isReportDecisionDetail
    ? Boolean(effectiveNotificationId) &&
      decisionNotificationQuery.isPending &&
      !initialDecisionNotification
    : Boolean(effectiveAnnouncementId) &&
      announcementsQuery.isPending &&
      !initialAnnouncement;

  const notFoundError = isReportDecisionDetail
    ? !effectiveNotificationId
      ? "通報結果通知が見つかりません。"
      : decisionNotificationQuery.isSuccess && !decisionNotification
        ? "通報結果通知が見つかりません。"
        : ""
    : !effectiveAnnouncementId
      ? "お知らせが見つかりません。"
      : announcementsQuery.isSuccess && !announcement
        ? "お知らせが見つかりません。"
        : "";

  const error =
    decisionMutationError ||
    announcementMutationError ||
    decisionQueryError ||
    announcementQueryError ||
    notFoundError;

  const tokenLabel =
    announcement?.tokenName ||
    announcement?.targetToken ||
    "対象トークン";

  const publishedAtLabel = formatDateTime(announcement?.publishedAt);

  const attachmentFiles = Array.isArray(announcement?.attachmentFiles)
    ? announcement.attachmentFiles
    : [];

  const decisionTargetLabel = decisionNotification
    ? getReportTargetLabel(decisionNotification.targetType)
    : "";

  const decisionCardLabel = decisionNotification
    ? getDecisionCardLabel(decisionNotification, decisionTargetLabel)
    : "";

  const decisionTitle = decisionNotification
    ? getDecisionCardTitle(decisionNotification)
    : "";

  const decisionBody = decisionNotification
    ? getDecisionBody(decisionNotification)
    : "";

  const decisionStatusLabel = decisionNotification
    ? getDecisionStatusLabel(decisionNotification)
    : "";

  const decisionOccurredAt = decisionNotification
    ? decisionNotification.decidedAt || decisionNotification.createdAt
    : "";

  const decisionOccurredAtLabel = formatDateTime(decisionOccurredAt);

  const isReporterDecision =
    decisionNotification?.notificationKind === "REPORTER_DECISION";

  const reportReasonLabel =
    isReporterDecision && decisionNotification
      ? getReportReasonLabel(decisionNotification.reportReason)
      : "";

  return (
    <Layout
      title="お知らせ"
      showBackButton
      backTo="/announcements"
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

        {!loading &&
        isReportDecisionDetail &&
        !decisionNotification &&
        !decisionQueryError ? (
          <div className="announcement-page__empty">
            通報結果通知が見つかりません。
          </div>
        ) : null}

        {!loading &&
        !isReportDecisionDetail &&
        !announcement &&
        !announcementQueryError ? (
          <div className="announcement-page__empty">
            お知らせが見つかりません。
          </div>
        ) : null}

        {!loading && isReportDecisionDetail && decisionNotification ? (
          <article className="announcement-page__detail">
            <h1 className="announcement-page__detail-title">
              {decisionTitle}
            </h1>

            <div className="announcement-page__card-head">
              <div className="announcement-page__card-meta">
                <span className="announcement-page__token">
                  {decisionCardLabel}
                </span>

                <time
                  className="announcement-page__date"
                  dateTime={decisionOccurredAt || undefined}
                >
                  {decisionOccurredAtLabel}
                </time>
              </div>
            </div>

            <div className="announcement-page__detail-content">
              {decisionBody}
            </div>

            {isReporterDecision ? (
              <>
                <div className="announcement-page__attachments">
                  通報理由: {reportReasonLabel}
                </div>

                {decisionNotification.reportDetail ? (
                  <div className="announcement-page__attachments">
                    通報詳細: {decisionNotification.reportDetail}
                  </div>
                ) : null}
              </>
            ) : null}

            <div className="announcement-page__attachments">
              審査結果: {decisionStatusLabel}
            </div>

            {decisionNotification.decisionReason ? (
              <div className="announcement-page__attachments">
                審査理由: {decisionNotification.decisionReason}
              </div>
            ) : null}
          </article>
        ) : null}

        {!loading && !isReportDecisionDetail && announcement ? (
          <article className="announcement-page__detail">
            <h1 className="announcement-page__detail-title">
              {announcement.title}
            </h1>

            <div className="announcement-page__card-head">
              <div className="announcement-page__card-meta">
                <span className="announcement-page__token">
                  {tokenLabel}
                </span>

                <time
                  className="announcement-page__date"
                  dateTime={announcement.publishedAt ?? undefined}
                >
                  {publishedAtLabel}
                </time>
              </div>
            </div>

            <div className="announcement-page__detail-content">
              {announcement.content}
            </div>

            {attachmentFiles.length > 0 ? (
              <div className="announcement-page__detail-attachments">
                <div className="announcement-page__attachment-list">
                  {attachmentFiles.map((file, index) => {
                    const fileName =
                      file.fileName ||
                      file.id ||
                      `添付ファイル ${index + 1}`;

                    const fileUrl = file.fileUrl || "";
                    const mimeType = file.mimeType || "";
                    const isImage = mimeType.startsWith("image/");
                    const attachmentKey = `${file.id || fileName}-${index}`;

                    if (isImage && fileUrl) {
                      return (
                        <a
                          key={attachmentKey}
                          className="announcement-page__image-attachment"
                          href={fileUrl}
                          target="_blank"
                          rel="noreferrer"
                          aria-label={`${fileName} を開く`}
                        >
                          <img
                            className="announcement-page__attachment-image"
                            src={fileUrl}
                            alt={fileName}
                            loading="lazy"
                          />
                        </a>
                      );
                    }

                    if (fileUrl) {
                      return (
                        <a
                          key={attachmentKey}
                          className="announcement-page__attachment-item announcement-page__attachment-link"
                          href={fileUrl}
                          target="_blank"
                          rel="noreferrer"
                        >
                          <span className="announcement-page__attachment-name">
                            {fileName}
                          </span>

                          {mimeType ? (
                            <span className="announcement-page__attachment-meta">
                              {mimeType}
                            </span>
                          ) : null}
                        </a>
                      );
                    }

                    return (
                      <div
                        key={attachmentKey}
                        className="announcement-page__attachment-item"
                      >
                        <span className="announcement-page__attachment-name">
                          {fileName}
                        </span>

                        {mimeType ? (
                          <span className="announcement-page__attachment-meta">
                            {mimeType}
                          </span>
                        ) : null}
                      </div>
                    );
                  })}
                </div>
              </div>
            ) : null}
          </article>
        ) : null}
      </section>
    </Layout>
  );
}