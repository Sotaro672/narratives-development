// frontend/amol/src/pages/AnnouncementDetailPage.tsx
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useLocation, useParams } from "react-router-dom";

import Layout from "../components/layout/Layout";
import {
  fetchMeAnnouncements,
  markMeAnnouncementRead,
} from "../features/announcement/api/announcementApi";
import type { AnnouncementListItem } from "../features/announcement/types";

import "../styles/page-layout.css";
import "../styles/announcement-page.css";

type AnnouncementDetailLocationState = {
  announcement?: AnnouncementListItem;
};

export default function AnnouncementDetailPage() {
  const { announcementId = "" } = useParams<{ announcementId: string }>();
  const location = useLocation();

  const locationState = location.state as AnnouncementDetailLocationState | null;
  const initialAnnouncement = locationState?.announcement;

  const [announcement, setAnnouncement] =
    useState<AnnouncementListItem | null>(initialAnnouncement ?? null);
  const [loading, setLoading] = useState<boolean>(!initialAnnouncement);
  const [error, setError] = useState<string>("");

  const markedReadRef = useRef<string>("");

  const effectiveAnnouncementId = useMemo(() => {
    return announcementId || announcement?.id || "";
  }, [announcementId, announcement?.id]);

  const loadAnnouncement = useCallback(
    async (signal?: AbortSignal) => {
      if (!effectiveAnnouncementId) {
        setLoading(false);
        setError("お知らせが見つかりません。");
        return;
      }

      if (announcement) {
        setLoading(false);
        return;
      }

      setLoading(true);
      setError("");

      try {
        const result = await fetchMeAnnouncements({
          page: 1,
          perPage: 100,
          signal,
        });

        const found =
          result.items.find((item) => item.id === effectiveAnnouncementId) ??
          null;

        setAnnouncement(found);

        if (!found) {
          setError("お知らせが見つかりません。");
        }
      } catch (caught) {
        if (signal?.aborted) {
          return;
        }

        setAnnouncement(null);
        setError(
          caught instanceof Error
            ? caught.message
            : "お知らせの取得に失敗しました",
        );
      } finally {
        if (!signal?.aborted) {
          setLoading(false);
        }
      }
    },
    [announcement, effectiveAnnouncementId],
  );

  useEffect(() => {
    const controller = new AbortController();

    void loadAnnouncement(controller.signal);

    return () => {
      controller.abort();
    };
  }, [loadAnnouncement]);

  useEffect(() => {
    if (!effectiveAnnouncementId) {
      return;
    }

    if (markedReadRef.current === effectiveAnnouncementId) {
      return;
    }

    markedReadRef.current = effectiveAnnouncementId;

    void markMeAnnouncementRead(effectiveAnnouncementId)
      .then(() => {
        setAnnouncement((current) => {
          if (!current) {
            return current;
          }

          return {
            ...current,
            isRead: true,
            readAt: current.readAt ?? new Date().toISOString(),
          };
        });
      })
      .catch((caught) => {
        setError(
          caught instanceof Error
            ? caught.message
            : "お知らせの既読化に失敗しました",
        );
      });
  }, [effectiveAnnouncementId]);

  const tokenLabel =
    announcement?.tokenName || announcement?.targetToken || "対象トークン";
  const publishedAtLabel = formatDateTime(announcement?.publishedAt);
  const attachmentFiles = Array.isArray(announcement?.attachmentFiles)
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
          <div className="announcement-page__error" role="alert">
            {error}
          </div>
        ) : null}

        {loading ? (
          <div className="announcement-page__state">読み込み中...</div>
        ) : null}

        {!loading && !announcement ? (
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
                <span className="announcement-page__token">{tokenLabel}</span>

                {publishedAtLabel ? (
                  <time
                    className="announcement-page__date"
                    dateTime={announcement.publishedAt ?? undefined}
                  >
                    {publishedAtLabel}
                  </time>
                ) : null}
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
                      file.fileName || file.id || `添付ファイル ${index + 1}`;
                    const fileUrl = file.fileUrl || "";
                    const mimeType = file.mimeType || "";
                    const isImage = mimeType.startsWith("image/");

                    if (isImage && fileUrl) {
                      return (
                        <a
                          key={`${file.id ?? fileName}-${index}`}
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
                          key={`${file.id ?? fileName}-${index}`}
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
                        key={`${file.id ?? fileName}-${index}`}
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

function formatDateTime(value?: string | null): string {
  if (!value) {
    return "";
  }

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return "";
  }

  return new Intl.DateTimeFormat("ja-JP", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}