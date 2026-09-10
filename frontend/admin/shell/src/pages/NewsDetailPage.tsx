// frontend/admin/shell/src/pages/NewsDetailPage.tsx

import { useLocation, useNavigate, useParams } from "react-router-dom";

import type { News } from "../shared/type/news";
import MediaGallery from "../shared/ui/MediaGallery/MediaGallery";
import Page, { DetailPageBody, PageHeader } from "../shared/ui/Page/Page";
import { formatDateTime } from "../shared/util/dateFormat";

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
        title={news?.title || "通知詳細"}
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
                <div className="news-detail-page__body-text">{news.body}</div>
              </section>

              {news.image ? (
                <section className="news-detail-page__section">
                  <h2 className="news-detail-page__section-title">添付画像</h2>
                  <MediaGallery
                    items={[
                      {
                        id: news.image.objectPath || news.id,
                        url: news.image.fileUrl,
                        fileName: news.image.fileName,
                      },
                    ]}
                    altFallback={news.image.alt || news.title}
                  />
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
            </div>
          }
        />
      )}
    </Page>
  );
}