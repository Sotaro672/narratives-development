// frontend/admin/shell/src/pages/NewsDetailPage.tsx

import { useLocation, useNavigate, useParams } from "react-router-dom";

import type { News } from "../shared/type/news";
import Page, { DetailPageBody, PageHeader } from "../shared/ui/Page/Page";
import { formatDateTime } from "../shared/util/dateFormat";
import { formatFileSize } from "../shared/util/fileSizeFormat";

import "./NewsDetailPage.css";

type NewsDetailLocationState = {
  news?: News;
};

export default function NewsDetailPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const { newsId = "" } = useParams<{ newsId?: string }>();

  const locationState = location.state as NewsDetailLocationState | null;
  const stateNews = locationState?.news;
  const news = stateNews && stateNews.id === newsId ? stateNews : null;

  return (
    <Page>
      <PageHeader
        title="通知詳細"
        meta={news ? formatDateTime(news.publishedAt) : undefined}
        leading={
          <button type="button" className="ui-page-header__back" aria-label="戻る" onClick={() => navigate("/news")}>
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <path d="M19 12H5" />
              <path d="M12 19l-7-7 7-7" />
            </svg>
          </button>
        }
      />

      {!news ? (
        <p role="alert">通知情報を取得できませんでした。通知一覧から開き直してください。</p>
      ) : (
        <DetailPageBody
          main={
            <div className="news-detail-page__main">
              <section className="news-detail-page__section">
                <h2 className="news-detail-page__section-title">通知内容</h2>

                <dl className="news-detail-page__fields">
                  <div className="news-detail-page__field">
                    <dt className="news-detail-page__field-label">タイトル</dt>
                    <dd className="news-detail-page__field-value news-detail-page__title">{news.title}</dd>
                  </div>

                  <div className="news-detail-page__field">
                    <dt className="news-detail-page__field-label">本文</dt>
                    <dd className="news-detail-page__field-value news-detail-page__body-text">{news.body}</dd>
                  </div>
                </dl>
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
                </section>
              ) : null}
            </div>
          }
          aside={
            <div className="news-detail-page__aside">
              <section className="news-detail-page__section">
                <h2 className="news-detail-page__section-title">通知情報</h2>

                <dl className="news-detail-page__fields news-detail-page__fields--compact">
                  <div className="news-detail-page__field">
                    <dt className="news-detail-page__field-label">配信日時</dt>
                    <dd className="news-detail-page__field-value">{formatDateTime(news.publishedAt)}</dd>
                  </div>

                  <div className="news-detail-page__field">
                    <dt className="news-detail-page__field-label">作成日時</dt>
                    <dd className="news-detail-page__field-value">{formatDateTime(news.createdAt)}</dd>
                  </div>

                  <div className="news-detail-page__field">
                    <dt className="news-detail-page__field-label">作成者</dt>
                    <dd className="news-detail-page__field-value">{news.createdByName || "-"}</dd>
                  </div>
                </dl>
              </section>

              {news.image ? (
                <section className="news-detail-page__section">
                  <h2 className="news-detail-page__section-title">画像情報</h2>

                  <dl className="news-detail-page__fields news-detail-page__fields--compact">
                    <div className="news-detail-page__field">
                      <dt className="news-detail-page__field-label">ファイル名</dt>
                      <dd className="news-detail-page__field-value">{news.image.fileName || "-"}</dd>
                    </div>

                    <div className="news-detail-page__field">
                      <dt className="news-detail-page__field-label">ファイルサイズ</dt>
                      <dd className="news-detail-page__field-value">{formatFileSize(news.image.fileSize)}</dd>
                    </div>
                  </dl>
                </section>
              ) : null}
            </div>
          }
        />
      )}
    </Page>
  );
}