// frontend/mall/src/pages/AnnouncementDetailPage.tsx

import { useEffect, useMemo, useRef, useState } from "react";
import { useLocation, useParams } from "react-router-dom";

import Layout from "../components/layout/Layout";
import Alert from "../components/ui/Alert";
import MediaGallery, {
  type MediaGalleryItem,
} from "../components/ui/MediaGallery";
import SectionHeader from "../components/ui/SectionHeader";
import TextState from "../components/ui/TextState";
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
import ReportModal from "../features/report/components/ReportModal";
import { useReport } from "../features/report/hooks/useReport";
import ReportFlagButton from "../features/shared/presentation/components/ReportFlagButton";
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
  const locationState = location.state as AnnouncementDetailLocationState | null;
  const stateAnnouncement = locationState?.announcement;
  const stateDecisionNotification = locationState?.reportDecisionNotification;
  const stateNews = locationState?.news;
  const report = useReport();

  const [attachmentActiveIndex, setAttachmentActiveIndex] = useState(0);

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
    !isNewsDetail && Boolean(effectiveNotificationId);

  const {
    announcement,
    loading: announcementLoading,
    error: announcementError,
    notFound: announcementNotFound,
  } = useAnnouncementDetail({
    announcementId,
    initialAnnouncement: stateAnnouncement,
    enabled: !isNewsDetail && !isReportDecisionDetail,
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
    if (!stateNews || stateNews.id !== effectiveNewsId) {
      return null;
    }

    return stateNews;
  }, [effectiveNewsId, stateNews]);

  const newsQuery = useNewsQuery({
    page: 1,
    perPage: 100,
    enabled: isNewsDetail && Boolean(effectiveNewsId),
  });

  const markNewsReadMutation = useMarkNewsReadMutation();

  const newsFromQuery = useMemo(() => {
    if (!newsQuery.data) {
      return null;
    }

    return (
      newsQuery.data.items.find((item) => item.id === effectiveNewsId) ?? null
    );
  }, [effectiveNewsId, newsQuery.data]);

  const news =
    isNewsDetail && newsQuery.data !== undefined
      ? newsFromQuery
      : initialNews;

  const markedNewsReadRef = useRef<string>("");

  useEffect(() => {
    if (!isNewsDetail || !effectiveNewsId || !news) {
      return;
    }

    if (news.isRead === true) {
      markedNewsReadRef.current = effectiveNewsId;
      return;
    }

    if (markedNewsReadRef.current === effectiveNewsId) {
      return;
    }

    markedNewsReadRef.current = effectiveNewsId;
    markNewsReadMutation.mutate(effectiveNewsId);
  }, [
    effectiveNewsId,
    isNewsDetail,
    markNewsReadMutation,
    news,
  ]);

  const newsQueryError =
    isNewsDetail && newsQuery.error instanceof Error
      ? newsQuery.error.message
      : isNewsDetail && newsQuery.error
        ? "システム通知の取得に失敗しました"
        : "";

  const newsMutationError =
    isNewsDetail && markNewsReadMutation.error instanceof Error
      ? markNewsReadMutation.error.message
      : isNewsDetail && markNewsReadMutation.error
        ? "システム通知の既読化に失敗しました"
        : "";

  const loading = isNewsDetail
    ? Boolean(effectiveNewsId) && newsQuery.isPending && !initialNews
    : isReportDecisionDetail
      ? decisionLoading
      : announcementLoading;

  const newsNotFound =
    isNewsDetail &&
    (!effectiveNewsId || (newsQuery.isSuccess && !news));

  const error =
    newsMutationError ||
    newsQueryError ||
    (isReportDecisionDetail && !decisionNotFound ? decisionError : "") ||
    (!isNewsDetail &&
    !isReportDecisionDetail &&
    !announcementNotFound
      ? announcementError
      : "");

  const tokenLabel =
    announcement?.tokenName ||
    announcement?.targetToken ||
    "対象トークン";

  const publishedAtLabel = formatDateTime(announcement?.publishedAt);

  const attachmentFiles = Array.isArray(announcement?.attachmentFiles)
    ? announcement.attachmentFiles
    : [];

  const attachmentMediaItems = useMemo<MediaGalleryItem[]>(
    () =>
      attachmentFiles.flatMap((file, index) => {
        const fileUrl = file.fileUrl?.trim() || "";
        const mimeType = file.mimeType?.trim() || "";
        const isMedia =
          mimeType.startsWith("image/") ||
          mimeType.startsWith("video/");

        if (!fileUrl || !isMedia) {
          return [];
        }

        const fileName =
          file.fileName ||
          file.id ||
          `添付メディア ${index + 1}`;

        return [
          {
            id: `${file.id || fileName}-${index}`,
            url: fileUrl,
            fileName,
            type: mimeType,
          },
        ];
      }),
    [attachmentFiles],
  );

  const attachmentOtherFiles = useMemo(
    () =>
      attachmentFiles.flatMap((file, index) => {
        const fileUrl = file.fileUrl?.trim() || "";
        const mimeType = file.mimeType?.trim() || "";
        const isMedia =
          mimeType.startsWith("image/") ||
          mimeType.startsWith("video/");

        if (fileUrl && isMedia) {
          return [];
        }

        return [
          {
            file,
            index,
            fileUrl,
            mimeType,
            fileName:
              file.fileName ||
              file.id ||
              `添付ファイル ${index + 1}`,
          },
        ];
      }),
    [attachmentFiles],
  );

  useEffect(() => {
    setAttachmentActiveIndex((currentIndex) => {
      if (attachmentMediaItems.length === 0) {
        return 0;
      }

      return Math.min(
        Math.max(currentIndex, 0),
        attachmentMediaItems.length - 1,
      );
    });
  }, [attachmentMediaItems.length]);

  const handleAttachmentPrev = () => {
    if (attachmentMediaItems.length <= 1) {
      return;
    }

    setAttachmentActiveIndex((currentIndex) =>
      currentIndex <= 0
        ? attachmentMediaItems.length - 1
        : currentIndex - 1,
    );
  };

  const handleAttachmentNext = () => {
    if (attachmentMediaItems.length <= 1) {
      return;
    }

    setAttachmentActiveIndex((currentIndex) =>
      currentIndex >= attachmentMediaItems.length - 1
        ? 0
        : currentIndex + 1,
    );
  };

  const handleAttachmentSelect = (index: number) => {
    if (
      index < 0 ||
      index >= attachmentMediaItems.length
    ) {
      return;
    }

    setAttachmentActiveIndex(index);
  };

  const handleOpenAnnouncementReport = () => {
    if (!announcement?.id) {
      return;
    }

    report.openAnnouncementReport({
      announcementId: announcement.id,
    });
  };

  return (
    <>
      <Layout
        title={isNewsDetail ? "システム通知" : "お知らせ"}
        showFooter
        mode="mypage"
        mainClassName="announcement-page-layout"
      >
        <section className="page-section content-page-section announcement-page">
          {error ? (
            <Alert
              variant="error"
              className="announcement-page__error"
            >
              {error}
            </Alert>
          ) : null}

          {loading ? (
            <TextState
              variant="loading"
              className="announcement-page__state-text"
            >
              読み込み中...
            </TextState>
          ) : null}

          {!loading &&
          isNewsDetail &&
          newsNotFound &&
          !newsQueryError ? (
            <TextState
              variant="empty"
              className="announcement-page__state-text"
            >
              システム通知が見つかりません。
            </TextState>
          ) : null}

          {!loading &&
          isReportDecisionDetail &&
          decisionNotFound ? (
            <TextState
              variant="empty"
              className="announcement-page__state-text"
            >
              通報結果通知が見つかりません。
            </TextState>
          ) : null}

          {!loading &&
          !isNewsDetail &&
          !isReportDecisionDetail &&
          announcementNotFound ? (
            <TextState
              variant="empty"
              className="announcement-page__state-text"
            >
              お知らせが見つかりません。
            </TextState>
          ) : null}

          {!loading && isNewsDetail && news ? (
            <NewsDetail news={news} />
          ) : null}

          {!loading &&
          isReportDecisionDetail &&
          decisionNotification ? (
            <ReportDecisionDetail notification={decisionNotification} />
          ) : null}

          {!loading &&
          !isNewsDetail &&
          !isReportDecisionDetail &&
          announcement ? (
            <article className="announcement-page__detail">
              <h1 className="announcement-page__detail-title">
                {announcement.title}
              </h1>

              <SectionHeader
                right={
                  <ReportFlagButton
                    label="お知らせを通報"
                    disabled={!announcement.id || report.submitting}
                    onClick={handleOpenAnnouncementReport}
                  />
                }
              >
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
              </SectionHeader>

              <div className="announcement-page__detail-content">
                {announcement.content}
              </div>

              {attachmentFiles.length > 0 ? (
                <div className="announcement-page__detail-attachments">
                  {attachmentMediaItems.length > 0 ? (
                    <MediaGallery
                      items={attachmentMediaItems}
                      activeIndex={attachmentActiveIndex}
                      altFallback="お知らせ添付メディア"
                      placeholderText="メディアがありません"
                      className="announcement-page__attachment-gallery"
                      onPrev={handleAttachmentPrev}
                      onNext={handleAttachmentNext}
                      onSelect={handleAttachmentSelect}
                    />
                  ) : null}

                  {attachmentOtherFiles.length > 0 ? (
                    <div className="announcement-page__attachment-list">
                      {attachmentOtherFiles.map(
                        ({
                          file,
                          index,
                          fileUrl,
                          mimeType,
                          fileName,
                        }) => {
                          const attachmentKey =
                            `${file.id || fileName}-${index}`;

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
                  ) : null}
                </div>
              ) : null}
            </article>
          ) : null}
        </section>
      </Layout>

      <ReportModal
        open={report.isOpen}
        targetType={report.target?.type}
        reason={report.reason}
        detail={report.detail}
        submitting={report.submitting}
        error={report.error}
        result={report.result}
        canSubmit={report.canSubmit}
        onReasonChange={report.setReason}
        onDetailChange={report.setDetail}
        onSubmit={report.submit}
        onClose={report.close}
      />
    </>
  );
}