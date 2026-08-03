// frontend/amol/src/pages/AnnouncementDetailPage.tsx

import { useEffect, useMemo, useRef } from "react";
import {
  useLocation,
  useParams,
} from "react-router-dom";

import Layout from "../components/layout/Layout";
import { formatDateTime } from "../components/utils/date";

import {
  useAnnouncementsQuery,
  useMarkAnnouncementReadMutation,
} from "../features/announcement/hooks/useAnnouncementsQuery";

import type { AnnouncementListItem } from "../features/shared/types/announcements";

import "../styles/page-layout.css";
import "../styles/announcement-page.css";

type AnnouncementDetailLocationState = {
  announcement?: AnnouncementListItem;
};

export default function AnnouncementDetailPage() {
  const { announcementId = "" } = useParams<{
    announcementId: string;
  }>();

  const location = useLocation();

  const locationState =
    location.state as AnnouncementDetailLocationState | null;

  const stateAnnouncement =
    locationState?.announcement;

  const effectiveAnnouncementId = useMemo(() => {
    return (
      announcementId.trim() ||
      stateAnnouncement?.id?.trim() ||
      ""
    );
  }, [
    announcementId,
    stateAnnouncement?.id,
  ]);

  const initialAnnouncement = useMemo(() => {
    if (
      !stateAnnouncement ||
      stateAnnouncement.id !==
        effectiveAnnouncementId
    ) {
      return null;
    }

    return stateAnnouncement;
  }, [
    effectiveAnnouncementId,
    stateAnnouncement,
  ]);

  const announcementsQuery =
    useAnnouncementsQuery({
      page: 1,
      perPage: 100,
      enabled: Boolean(
        effectiveAnnouncementId,
      ),
    });

  const markAnnouncementReadMutation =
    useMarkAnnouncementReadMutation();

  const announcementFromQuery = useMemo(() => {
    if (!announcementsQuery.data) {
      return null;
    }

    return (
      announcementsQuery.data.items.find(
        (item) =>
          item.id ===
          effectiveAnnouncementId,
      ) ?? null
    );
  }, [
    announcementsQuery.data,
    effectiveAnnouncementId,
  ]);

  const announcement =
    announcementsQuery.data !== undefined
      ? announcementFromQuery
      : initialAnnouncement;

  const loading =
    Boolean(effectiveAnnouncementId) &&
    announcementsQuery.isPending &&
    !initialAnnouncement;

  const markedReadRef = useRef<string>("");

  useEffect(() => {
    if (
      !effectiveAnnouncementId ||
      !announcement
    ) {
      return;
    }

    if (announcement.isRead === true) {
      markedReadRef.current =
        effectiveAnnouncementId;

      return;
    }

    if (
      markedReadRef.current ===
      effectiveAnnouncementId
    ) {
      return;
    }

    markedReadRef.current =
      effectiveAnnouncementId;

    markAnnouncementReadMutation.mutate(
      effectiveAnnouncementId,
    );
  }, [
    announcement,
    effectiveAnnouncementId,
    markAnnouncementReadMutation,
  ]);

  const queryError =
    announcementsQuery.error instanceof Error
      ? announcementsQuery.error.message
      : announcementsQuery.error
        ? "お知らせの取得に失敗しました"
        : "";

  const mutationError =
    markAnnouncementReadMutation.error instanceof
    Error
      ? markAnnouncementReadMutation.error.message
      : markAnnouncementReadMutation.error
        ? "お知らせの既読化に失敗しました"
        : "";

  const notFoundError =
    !effectiveAnnouncementId
      ? "お知らせが見つかりません。"
      : announcementsQuery.isSuccess &&
          !announcement
        ? "お知らせが見つかりません。"
        : "";

  const error =
    mutationError ||
    queryError ||
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
        !announcement &&
        !queryError ? (
          <div className="announcement-page__empty">
            お知らせが見つかりません。
          </div>
        ) : null}

        {!loading && announcement ? (
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
                        `添付ファイル ${
                          index + 1
                        }`;

                      const fileUrl =
                        file.fileUrl || "";

                      const mimeType =
                        file.mimeType || "";

                      const isImage =
                        mimeType.startsWith(
                          "image/",
                        );

                      const attachmentKey =
                        `${
                          file.id ||
                          fileName
                        }-${index}`;

                      if (
                        isImage &&
                        fileUrl
                      ) {
                        return (
                          <a
                            key={
                              attachmentKey
                            }
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
                            key={
                              attachmentKey
                            }
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
                          key={
                            attachmentKey
                          }
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