// frontend/admin/shell/src/pages/NewsDetailPage.tsx

import { useLocation, useNavigate, useParams } from "react-router-dom";

import type { News } from "../shared/type/news";
import Page, { PageHeader } from "../shared/ui/Page/Page";
import { formatDateTime } from "../shared/util/dateFormat";

import "./NewsDetailPage.css";

type NewsDetailLocationState = {
  news?: News;
};

function formatFileSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return "-";
  }

  if (bytes < 1024) {
    return `${bytes} B`;
  }

  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} KB`;
  }

  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export default function NewsDetailPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const { newsId = "" } = useParams<{ newsId?: string }>();

  const locationState = location.state as NewsDetailLocationState | null;
  const stateNews = locationState?.news;

  const news =
    stateNews &&
    stateNews.id === newsId
      ? stateNews
      : null;

  return (
    <Page>
      <PageHeader
        title="通知詳細"
        meta={news ? formatDateTime(news.publishedAt) : undefined}
        leading={
          <button
            type="button"
            className="ui-page-header__back"
            aria-label="戻る"
            onClick={() => navigate("/news")}
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

      {!news ? (
        <p className="news-detail-page__error" role="alert">
          通知情報を取得できませんでした。通知一覧から開き直してください。
        </p>
      ) : (
        <div className="news-detail-page">
          <section className="news-detail-page__section">
            <h2 className="news-detail-page__title">{news.title}</h2>

            <dl className="news-detail-page__meta">
              <div className="news-detail-page__meta-row">
                <dt>配信日時</dt>
                <dd>{formatDateTime(news.publishedAt)}</dd>
              </div>

              <div className="news-detail-page__meta-row">
                <dt>作成日時</dt>
                <dd>{formatDateTime(news.createdAt)}</dd>
              </div>

              <div className="news-detail-page__meta-row">
                <dt>作成者</dt>
                <dd>{news.createdBy || "-"}</dd>
              </div>
            </dl>
          </section>

          <section className="news-detail-page__section">
            <h2 className="news-detail-page__section-title">本文</h2>
            <div className="news-detail-page__body">{news.body}</div>
          </section>

          {news.image ? (
            <section className="news-detail-page__section">
              <h2 className="news-detail-page__section-title">添付画像</h2>

              <a
                className="news-detail-page__image-link"
                href={news.image.fileUrl}
                target="_blank"
                rel="noreferrer"
                aria-label={`${news.image.fileName || "添付画像"}を開く`}
              >
                <img
                  className="news-detail-page__image"
                  src={news.image.fileUrl}
                  alt={news.image.alt || news.title}
                />
              </a>

              <dl className="news-detail-page__meta news-detail-page__image-meta">
                <div className="news-detail-page__meta-row">
                  <dt>ファイル名</dt>
                  <dd>{news.image.fileName || "-"}</dd>
                </div>

                <div className="news-detail-page__meta-row">
                  <dt>MIME Type</dt>
                  <dd>{news.image.mimeType || "-"}</dd>
                </div>

                <div className="news-detail-page__meta-row">
                  <dt>ファイルサイズ</dt>
                  <dd>{formatFileSize(news.image.fileSize)}</dd>
                </div>

                {news.image.alt !== undefined ? (
                  <div className="news-detail-page__meta-row">
                    <dt>代替テキスト</dt>
                    <dd>{news.image.alt || "-"}</dd>
                  </div>
                ) : null}
              </dl>
            </section>
          ) : null}
        </div>
      )}
    </Page>
  );
}