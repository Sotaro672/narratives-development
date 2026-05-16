// frontend/amol/src/features/catalog/components/ReviewSection.tsx
import { formatDateTime } from "../../../../components/utils/date";
import type {
  CatalogProductBlueprintReview,
  CatalogProductReviewSummary,
} from "../../types";
import { renderRatingStars } from "../../utils/format";

type ReviewSectionProps = {
  reviewSummary: CatalogProductReviewSummary | undefined;
  reviewItems: CatalogProductBlueprintReview[];
  isLoadingReviews: boolean;
  reviewErrorMessage: string;
};

export default function ReviewSection({
  reviewSummary,
  reviewItems,
  isLoadingReviews,
  reviewErrorMessage,
}: ReviewSectionProps) {
  return (
    <section className="catalog-page-card">
      <h2 className="catalog-page-card-title">レビュー</h2>

      {reviewSummary ? (
        <div className="catalog-page-review-summary">
          <strong>{reviewSummary.averageRating.toFixed(1)}</strong>
          <span>{reviewSummary.totalCount}件</span>
        </div>
      ) : null}

      {isLoadingReviews ? (
        <p className="catalog-page-model-help">
          レビューを読み込んでいます。
        </p>
      ) : null}

      {!isLoadingReviews && reviewErrorMessage ? (
        <p className="catalog-page-error" role="alert">
          {reviewErrorMessage}
        </p>
      ) : null}

      {!isLoadingReviews &&
      !reviewErrorMessage &&
      reviewItems.length === 0 ? (
        <p className="catalog-page-model-help">
          まだレビューはありません。
        </p>
      ) : null}

      {!isLoadingReviews &&
      !reviewErrorMessage &&
      reviewItems.length > 0 ? (
        <div className="catalog-page-review-list">
          {reviewItems.map((review) => (
            <ReviewItem key={review.id} review={review} />
          ))}
        </div>
      ) : null}
    </section>
  );
}

function ReviewItem({
  review,
}: {
  review: CatalogProductBlueprintReview;
}) {
  const reviewedAt = formatDateTime(review.reviewedAt);

  return (
    <article className="catalog-page-review-item">
      <div className="catalog-page-review-header">
        {review.avatarIcon ? (
          <img
            src={review.avatarIcon}
            alt={review.avatarName || "avatar"}
            className="catalog-page-review-avatar"
          />
        ) : (
          <div className="catalog-page-review-avatar-placeholder">
            {review.avatarName?.slice(0, 1) || "?"}
          </div>
        )}

        <div>
          <p className="catalog-page-review-avatar-name">
            {review.avatarName || "匿名ユーザー"}
          </p>
          <p className="catalog-page-review-meta">
            {renderRatingStars(review.rating)}
            {reviewedAt !== "-" ? `・${reviewedAt}` : ""}
          </p>
        </div>
      </div>

      {review.title ? (
        <h3 className="catalog-page-review-title">{review.title}</h3>
      ) : null}

      {review.body ? (
        <p className="catalog-page-review-body">{review.body}</p>
      ) : null}

      <p className="catalog-page-review-votes">
        参考になった: {review.helpfulVotes} / {review.totalVotes}
      </p>
    </article>
  );
}