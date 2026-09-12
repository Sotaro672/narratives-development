// frontend/admin/shell/src/pages/AnnouncementPage.tsx

import { useMemo } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { useAnnouncementDetail } from "../features/announcement/presentation/hooks/useAnnouncementDetail";
import MediaGallery, {
  type MediaGalleryItem,
} from "../shared/ui/MediaGallery/MediaGallery";
import Page, {
  DetailPageBody,
  PageHeader,
} from "../shared/ui/Page/Page";
import Tab from "../shared/ui/Tab/Tab";
import { formatDateTime } from "../shared/util/dateFormat";

export default function AnnouncementPage() {
  const navigate = useNavigate();
  const { companyId = "", announcementId = "" } = useParams<{
    companyId?: string;
    announcementId?: string;
  }>();

  const {
    announcement,
    loading,
    error,
    reload,
  } = useAnnouncementDetail(companyId, announcementId);

  const galleryItems = useMemo<MediaGalleryItem[]>(
    () =>
      (announcement?.attachmentFiles ?? [])
        .filter(
          (file) =>
            Boolean(file.fileUrl) &&
            (!file.mimeType || file.mimeType.startsWith("image/")),
        )
        .map((file) => ({
          id: file.id || file.objectPath,
          url: file.fileUrl,
          fileName: file.fileName || undefined,
        })),
    [announcement?.attachmentFiles],
  );

  const renderMain = () => {
    if (loading && !announcement) {
      return <p>告知詳細を取得しています...</p>;
    }

    if (error && !announcement) {
      return (
        <div role="alert">
          <p>告知詳細を取得できませんでした。</p>
          <p>{error}</p>
          <button type="button" onClick={() => void reload()}>
            再読み込み
          </button>
        </div>
      );
    }

    if (!announcement) {
      return <p role="alert">告知情報を取得できませんでした。</p>;
    }

    return (
      <div>
        <section className="ui-detail-section">
          <h2>配信タイトル</h2>
          <p>{announcement.title || "-"}</p>
        </section>

        <section className="ui-detail-section">
          <h2>本文</h2>
          <div style={{ whiteSpace: "pre-wrap" }}>
            {announcement.content || "-"}
          </div>
        </section>

        {galleryItems.length > 0 ? (
          <section className="ui-detail-section">
            <h2>添付画像</h2>
            <MediaGallery
              items={galleryItems}
              altFallback={announcement.title || "告知添付画像"}
              placeholderText="添付画像はありません。"
            />
          </section>
        ) : null}
      </div>
    );
  };

  return (
    <Page>
      <PageHeader
        title={announcement?.title || "告知詳細"}
        meta={
          announcement ? (
            <Tab
              tone={announcement.published ? "success" : "warning"}
              aria-label={`告知状態 ${
                announcement.published ? "公開済み" : "下書き"
              }`}
            >
              {announcement.published ? "公開済み" : "下書き"}
            </Tab>
          ) : undefined
        }
        leading={
          <button
            type="button"
            className="ui-page-header__back"
            aria-label="戻る"
            onClick={() => navigate(-1)}
          >
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              aria-hidden="true"
            >
              <path d="M19 12H5" />
              <path d="M12 19l-7-7 7-7" />
            </svg>
          </button>
        }
      />

      {announcement ? (
        <DetailPageBody
          main={renderMain()}
          aside={
            <section className="ui-detail-section">
              <dl className="ui-detail-definition-list ui-detail-definition-list--meta">
                <dt>告知ID</dt>
                <dd>{announcement.id || "-"}</dd>

                <dt>対象トークンID</dt>
                <dd>{announcement.targetToken || "-"}</dd>

                <dt>送信対象数</dt>
                <dd>
                  {(announcement.targetAvatars ?? []).length.toLocaleString()}
                </dd>

                <dt>作成者</dt>
                <dd>{announcement.createdByName || "-"}</dd>

                <dt>作成日時</dt>
                <dd>
                  {announcement.createdAt
                    ? formatDateTime(announcement.createdAt)
                    : "-"}
                </dd>

                <dt>更新者</dt>
                <dd>{announcement.updatedByName || "-"}</dd>

                <dt>最終更新日時</dt>
                <dd>
                  {announcement.updatedAt
                    ? formatDateTime(announcement.updatedAt)
                    : "-"}
                </dd>

                <dt>配信日時</dt>
                <dd>
                  {announcement.publishedAt
                    ? formatDateTime(announcement.publishedAt)
                    : "-"}
                </dd>
              </dl>
            </section>
          }
        />
      ) : (
        renderMain()
      )}
    </Page>
  );
}