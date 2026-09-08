// frontend/mall/src/features/notification/presentation/components/NewsDetail.tsx

import { formatDateTime } from "../../../../components/utils/date";
import type { News } from "../../../shared/types/news";

type NewsDetailProps = {
  news: News;
};

export default function NewsDetail({
  news,
}: NewsDetailProps) {
  const occurredAt =
    news.publishedAt ||
    news.createdAt;

  const occurredAtLabel =
    formatDateTime(occurredAt);

  return (
    <article className="announcement-page__detail">
      <h1 className="announcement-page__detail-title">
        {news.title}
      </h1>

      <div className="announcement-page__card-head">
        <div className="announcement-page__card-meta">
          <span className="announcement-page__token">
            システム通知
          </span>

          <time
            className="announcement-page__date"
            dateTime={occurredAt || undefined}
          >
            {occurredAtLabel}
          </time>
        </div>
      </div>

      {news.image ? (
        <div className="announcement-page__detail-news-image-wrap">
          <img
            className="announcement-page__detail-news-image"
            src={news.image.fileUrl}
            alt={news.image.alt || news.title}
          />
        </div>
      ) : null}

      <div className="announcement-page__detail-content">
        {news.body}
      </div>
    </article>
  );
}