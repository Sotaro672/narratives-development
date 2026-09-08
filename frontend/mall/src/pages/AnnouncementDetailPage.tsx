// frontend/mall/src/pages/AnnouncementDetailPage.tsx

import { useEffect, useMemo, useRef } from "react";
import { useLocation, useParams } from "react-router-dom";

import Layout from "../components/layout/Layout";
import { formatDateTime } from "../components/utils/date";

import { useAnnouncementDetail } from "../features/announcement/hooks/useAnnouncementDetail";
import {
  useMarkNewsReadMutation,
  useNewsQuery,
} from "../features/news/hooks/useNewsQuery";
import NewsDetail from "../features/notification/presentation/components/NewsDetail";
import ReportDecisionDetail from "../features/notification/presentation/components/ReportDecisionDetail";
import { useReportDecisionDetail } from "../features/notification/presentation/hooks/useReportDecisionDetail";
import type { ReportDecisionNotification } from "../features/notification/infrastructure/reportDecisionNotificationApi";
import type { AnnouncementListItem } from "../features/shared/types/announcements";
import type { News } from "../features/shared/types/news";

import "../styles/page-layout.css";
import "../styles/announcement-page.css";

type AnnouncementDetailLocationState = {
  announcement?: AnnouncementListItem;
  reportDecisionNotification?: ReportDecisionNotification;
  news?: News;
};

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
    [isNewsDetail, notificationId, stateDecisionNotification?.id],
  );

  const isReportDecisionDetail =
    !isNewsDetail &&
    Boolean(effectiveNotificationId);

  const {
    announcement,
    loading: announcementLoading,
    error: announcementError,
    notFound: announcementNotFound,
  } = useAnnouncementDetail({
    announcementId,
    initialAnnouncement: stateAnnouncement,
    enabled:
      !isNewsDetail &&
      !isReportDecisionDetail,
  });

  const {
    notification: decisionNotification,
    loading: decisionLoading,
    error: decisionError,
    notFound: decisionNotFound,
  } = useReportDecisionDetail({
    notificationId: effectiveNotificationId,
    initialNotification: stateDecisionNotification,
    enabled: isReportDecisionDetail,
  });

  const initialNews = useMemo(() => {
    if (
      !stateNews ||
      stateNews.id !== effectiveNewsId
    ) {
      return null;
    }

    return stateNews;
  }, [effectiveNewsId, stateNews]);

  const newsQuery = useNewsQuery({
    page: 1,
    perPage: 100,
    enabled:
      isNewsDetail &&
      Boolean(effectiveNewsId),
  });

  const markNewsReadMutation =
    useMarkNewsReadMutation();

  const newsFromQuery = useMemo(() => {
    if (!newsQuery.data) {
      return null;
    }

    return (
      newsQuery.data.items.find(
        (item) => item.id === effectiveNewsId,
      ) ?? null
    );
  }, [effectiveNewsId, newsQuery.data]);

  const news =
    isNewsDetail &&
    newsQuery.data !== undefined
      ? newsFromQuery
      : initialNews;

  const markedNewsReadRef = useRef<string>("");

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

  const newsQueryError =
    isNewsDetail &&
    newsQuery.error instanceof Error
      ? newsQuery.error.message
      : isNewsDetail &&
          newsQuery.error
        ? "システム通知の取得に失敗しました"
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
      ? decisionLoading
      : announcementLoading;

  const newsNotFound =
    isNewsDetail &&
    (
      !effectiveNewsId ||
      (
        newsQuery.isSuccess &&
        !news
      )
    );

  const error =
    newsMutationError ||
    newsQueryError ||
    (
      isReportDecisionDetail &&
      !decisionNotFound
        ? decisionError
        : ""
    ) ||
    (
      !isNewsDetail &&
      !isReportDecisionDetail &&
      !announcementNotFound
        ? announcementError
        : ""
    );

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
        newsNotFound &&
        !newsQueryError ? (
          <div className="announcement-page__empty">
            システム通知が見つかりません。
          </div>
        ) : null}

        {!loading &&
        isReportDecisionDetail &&
        decisionNotFound ? (
          <div className="announcement-page__empty">
            通報結果通知が見つかりません。
          </div>
        ) : null}

        {!loading &&
        !isNewsDetail &&
        !isReportDecisionDetail &&
        announcementNotFound ? (
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
          <ReportDecisionDetail
            notification={decisionNotification}
          />
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
                  {attachmentFiles.map((file, index) => {
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