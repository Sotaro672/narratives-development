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
  useMarkNewsReadMutation,
  useNewsQuery,
} from "../features/news/hooks/useNewsQuery";
import NewsDetail from "../features/notification/presentation/components/NewsDetail";
import {
  useMarkReportDecisionNotificationReadMutation,
  useReportDecisionNotificationQuery,
} from "../features/notification/presentation/hooks/useReportDecisionNotificationsQuery";
import type { ReportDecisionNotification } from "../features/notification/infrastructure/reportDecisionNotificationApi";
import type { AnnouncementListItem } from "../features/shared/types/announcements";
import type { News } from "../features/shared/types/news";
import {
  getReportReasonLabel,
  type ReportTargetType,
} from "../features/shared/types/report";

import "../styles/page-layout.css";
import "../styles/announcement-page.css";

type AnnouncementDetailLocationState = {
  announcement?: AnnouncementListItem;
  reportDecisionNotification?: ReportDecisionNotification;
  news?: News;
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

function getDecisionCardTitle(
  notification: ReportDecisionNotification,
): string {
  if (notification.notificationKind === "TARGET_ENFORCEMENT") {
    return "運営による措置のお知らせ";
  }

  return "通報内容の確認が完了しました";
}

function getDecisionBody(
  notification: ReportDecisionNotification,
): string {
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
    newsId = "",
  } = useParams<{
    announcementId?: string;
    notificationId?: string;
    newsId?: string;
  }>();

  const location = useLocation();
  const locationState =
    location.state as AnnouncementDetailLocationState | null;

  const stateAnnouncement = locationState?.announcement;
  const stateDecisionNotification =
    locationState?.reportDecisionNotification;
  const stateNews = locationState?.news;

  const effectiveNewsId = useMemo(
    () => newsId || stateNews?.id || "",
    [newsId, stateNews?.id],
  );

  const isNewsDetail = Boolean(effectiveNewsId);

  const effectiveNotificationId = useMemo(
    () =>
      isNewsDetail
        ? ""
        : notificationId.trim() ||
          stateDecisionNotification?.id?.trim() ||
          "",
    [
      isNewsDetail,
      notificationId,
      stateDecisionNotification?.id,
    ],
  );

  const isReportDecisionDetail =
    !isNewsDetail &&
    Boolean(effectiveNotificationId);

  const effectiveAnnouncementId = useMemo(() => {
    if (
      isNewsDetail ||
      isReportDecisionDetail
    ) {
      return "";
    }

    return (
      announcementId.trim() ||
      stateAnnouncement?.id?.trim() ||
      ""
    );
  }, [
    announcementId,
    isNewsDetail,
    isReportDecisionDetail,
    stateAnnouncement?.id,
  ]);

  const initialAnnouncement = useMemo(() => {
    if (
      !stateAnnouncement ||
      stateAnnouncement.id !== effectiveAnnouncementId
    ) {
      return null;
    }

    return stateAnnouncement;
  }, [
    effectiveAnnouncementId,
    stateAnnouncement,
  ]);

  const initialDecisionNotification = useMemo(() => {
    if (
      !stateDecisionNotification ||
      stateDecisionNotification.id !== effectiveNotificationId
    ) {
      return null;
    }

    return stateDecisionNotification;
  }, [
    effectiveNotificationId,
    stateDecisionNotification,
  ]);

  const initialNews = useMemo(() => {
    if (
      !stateNews ||
      stateNews.id !== effectiveNewsId
    ) {
      return null;
    }

    return stateNews;
  }, [
    effectiveNewsId,
    stateNews,
  ]);

  const announcementsQuery = useAnnouncementsQuery({
    page: 1,
    perPage: 100,
    enabled:
      !isNewsDetail &&
      !isReportDecisionDetail &&
      Boolean(effectiveAnnouncementId),
  });

  const decisionNotificationQuery =
    useReportDecisionNotificationQuery(
      effectiveNotificationId,
      {
        enabled:
          isReportDecisionDetail &&
          Boolean(effectiveNotificationId),
      },
    );

  const newsQuery = useNewsQuery({
    page: 1,
    perPage: 100,
    enabled:
      isNewsDetail &&
      Boolean(effectiveNewsId),
  });

  const markAnnouncementReadMutation =
    useMarkAnnouncementReadMutation();

  const markDecisionReadMutation =
    useMarkReportDecisionNotificationReadMutation();

  const markNewsReadMutation =
    useMarkNewsReadMutation();

  const announcementFromQuery = useMemo(() => {
    if (!announcementsQuery.data) {
      return null;
    }

    return (
      announcementsQuery.data.items.find(
        (item) =>
          item.id === effectiveAnnouncementId,
      ) ?? null
    );
  }, [
    announcementsQuery.data,
    effectiveAnnouncementId,
  ]);

  const newsFromQuery = useMemo(() => {
    if (!newsQuery.data) {
      return null;
    }

    return (
      newsQuery.data.items.find(
        (item) =>
          item.id === effectiveNewsId,
      ) ?? null
    );
  }, [
    effectiveNewsId,
    newsQuery.data,
  ]);

  const announcement =
    !isNewsDetail &&
    !isReportDecisionDetail &&
    announcementsQuery.data !== undefined
      ? announcementFromQuery
      : initialAnnouncement;

  const decisionNotification =
    isReportDecisionDetail &&
    decisionNotificationQuery.data !== undefined
      ? decisionNotificationQuery.data
      : initialDecisionNotification;

  const news =
    isNewsDetail &&
    newsQuery.data !== undefined
      ? newsFromQuery
      : initialNews;

  const markedAnnouncementReadRef =
    useRef<string>("");

  const markedDecisionReadRef =
    useRef<string>("");

  const markedNewsReadRef =
    useRef<string>("");

  useEffect(() => {
    if (
      isNewsDetail ||
      isReportDecisionDetail ||
      !effectiveAnnouncementId ||
      !announcement
    ) {
      return;
    }

    if (announcement.isRead === true) {
      markedAnnouncementReadRef.current =
        effectiveAnnouncementId;
      return;
    }

    if (
      markedAnnouncementReadRef.current ===
      effectiveAnnouncementId
    ) {
      return;
    }

    markedAnnouncementReadRef.current =
      effectiveAnnouncementId;

    markAnnouncementReadMutation.mutate(
      effectiveAnnouncementId,
    );
  }, [
    announcement,
    effectiveAnnouncementId,
    isNewsDetail,
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
      markedDecisionReadRef.current =
        effectiveNotificationId;
      return;
    }

    if (
      markedDecisionReadRef.current ===
      effectiveNotificationId
    ) {
      return;
    }

    markedDecisionReadRef.current =
      effectiveNotificationId;

    markDecisionReadMutation.mutate(
      effectiveNotificationId,
    );
  }, [
    decisionNotification,
    effectiveNotificationId,
    isReportDecisionDetail,
    markDecisionReadMutation,
  ]);

  useEffect(() => {
    if (
      !isNewsDetail ||
      !effectiveNewsId ||
      !news
    ) {
      return;
    }

    if (news.isRead === true) {
      markedNewsReadRef.current =
        effectiveNewsId;
      return;
    }

    if (
      markedNewsReadRef.current ===
      effectiveNewsId
    ) {
      return;
    }

    markedNewsReadRef.current =
      effectiveNewsId;

    markNewsReadMutation.mutate(
      effectiveNewsId,
    );
  }, [
    effectiveNewsId,
    isNewsDetail,
    markNewsReadMutation,
    news,
  ]);

  const announcementQueryError =
    !isNewsDetail &&
    !isReportDecisionDetail &&
    announcementsQuery.error instanceof Error
      ? announcementsQuery.error.message
      : !isNewsDetail &&
          !isReportDecisionDetail &&
          announcementsQuery.error
        ? "お知らせの取得に失敗しました"
        : "";

  const decisionQueryError =
    isReportDecisionDetail &&
    decisionNotificationQuery.error instanceof Error
      ? decisionNotificationQuery.error.message
      : isReportDecisionDetail &&
          decisionNotificationQuery.error
        ? "通報結果通知の取得に失敗しました"
        : "";

  const newsQueryError =
    isNewsDetail &&
    newsQuery.error instanceof Error
      ? newsQuery.error.message
      : isNewsDetail &&
          newsQuery.error
        ? "システム通知の取得に失敗しました"
        : "";

  const announcementMutationError =
    !isNewsDetail &&
    !isReportDecisionDetail &&
    markAnnouncementReadMutation.error instanceof Error
      ? markAnnouncementReadMutation.error.message
      : !isNewsDetail &&
          !isReportDecisionDetail &&
          markAnnouncementReadMutation.error
        ? "お知らせの既読化に失敗しました"
        : "";

  const decisionMutationError =
    isReportDecisionDetail &&
    markDecisionReadMutation.error instanceof Error
      ? markDecisionReadMutation.error.message
      : isReportDecisionDetail &&
          markDecisionReadMutation.error
        ? "通報結果通知の既読化に失敗しました"
        : "";

  const newsMutationError =
    isNewsDetail &&
    markNewsReadMutation.error instanceof Error
      ? markNewsReadMutation.error.message
      : isNewsDetail &&
          markNewsReadMutation.error
        ? "システム通知の既読化に失敗しました"
        : "";

  const loading = isNewsDetail
    ? Boolean(effectiveNewsId) &&
      newsQuery.isPending &&
      !initialNews
    : isReportDecisionDetail
      ? Boolean(effectiveNotificationId) &&
        decisionNotificationQuery.isPending &&
        !initialDecisionNotification
      : Boolean(effectiveAnnouncementId) &&
        announcementsQuery.isPending &&
        !initialAnnouncement;

  const notFoundError = isNewsDetail
    ? !effectiveNewsId
      ? "システム通知が見つかりません。"
      : newsQuery.isSuccess && !news
        ? "システム通知が見つかりません。"
        : ""
    : isReportDecisionDetail
      ? !effectiveNotificationId
        ? "通報結果通知が見つかりません。"
        : decisionNotificationQuery.isSuccess &&
            !decisionNotification
          ? "通報結果通知が見つかりません。"
          : ""
      : !effectiveAnnouncementId
        ? "お知らせが見つかりません。"
        : announcementsQuery.isSuccess &&
            !announcement
          ? "お知らせが見つかりません。"
          : "";

  const error =
    newsMutationError ||
    decisionMutationError ||
    announcementMutationError ||
    newsQueryError ||
    decisionQueryError ||
    announcementQueryError ||
    notFoundError;

  const tokenLabel =
    announcement?.tokenName ||
    announcement?.targetToken ||
    "対象トークン";

  const publishedAtLabel =
    formatDateTime(
      announcement?.publishedAt,
    );

  const attachmentFiles =
    Array.isArray(
      announcement?.attachmentFiles,
    )
      ? announcement.attachmentFiles
      : [];

  const decisionTargetLabel =
    decisionNotification
      ? getReportTargetLabel(
          decisionNotification.targetType,
        )
      : "";

  const decisionCardLabel =
    decisionNotification
      ? getDecisionCardLabel(
          decisionNotification,
          decisionTargetLabel,
        )
      : "";

  const decisionTitle =
    decisionNotification
      ? getDecisionCardTitle(
          decisionNotification,
        )
      : "";

  const decisionBody =
    decisionNotification
      ? getDecisionBody(
          decisionNotification,
        )
      : "";

  const decisionStatusLabel =
    decisionNotification
      ? getDecisionStatusLabel(
          decisionNotification,
        )
      : "";

  const decisionOccurredAt =
    decisionNotification
      ? decisionNotification.decidedAt ||
        decisionNotification.createdAt
      : "";

  const decisionOccurredAtLabel =
    formatDateTime(
      decisionOccurredAt,
    );

  const isReporterDecision =
    decisionNotification?.notificationKind ===
    "REPORTER_DECISION";

  const reportReasonLabel =
    isReporterDecision &&
    decisionNotification
      ? getReportReasonLabel(
          decisionNotification.reportReason,
        )
      : "";

  return (
    <Layout
      title={
        isNewsDetail
          ? "システム通知"
          : "お知らせ"
      }
      showBackButton
      backTo="/announcements"
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

        {!loading &&
        isNewsDetail &&
        !news &&
        !newsQueryError ? (
          <div className="announcement-page__empty">
            システム通知が見つかりません。
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
        !isNewsDetail &&
        !isReportDecisionDetail &&
        !announcement &&
        !announcementQueryError ? (
          <div className="announcement-page__empty">
            お知らせが見つかりません。
          </div>
        ) : null}

        {!loading &&
        isNewsDetail &&
        news ? (
          <NewsDetail news={news} />
        ) : null}

        {!loading &&
        isReportDecisionDetail &&
        decisionNotification ? (
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
                  dateTime={
                    decisionOccurredAt ||
                    undefined
                  }
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

        {!loading &&
        !isNewsDetail &&
        !isReportDecisionDetail &&
        announcement ? (
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
                  dateTime={
                    announcement.publishedAt ??
                    undefined
                  }
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
                  {attachmentFiles.map(
                    (file, index) => {
                      const fileName =
                        file.fileName ||
                        file.id ||
                        `添付ファイル ${index + 1}`;

                      const fileUrl =
                        file.fileUrl ||
                        "";

                      const mimeType =
                        file.mimeType ||
                        "";

                      const isImage =
                        mimeType.startsWith(
                          "image/",
                        );

                      const attachmentKey =
                        `${file.id || fileName}-${index}`;

                      if (
                        isImage &&
                        fileUrl
                      ) {
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
                    },
                  )}
                </div>
              </div>
            ) : null}
          </article>
        ) : null}
      </section>
    </Layout>
  );
}