// frontend/amol/src/features/catalog/presentation/components/ReviewSection.tsx
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
  onAvatarClick?: (avatarId: string) => void;
};

export default function ReviewSection({
  reviewSummary,
  reviewItems,
  isLoadingReviews,
  reviewErrorMessage,
  onAvatarClick,
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
            <ReviewItem
              key={review.id}
              review={review}
              onAvatarClick={onAvatarClick}
            />
          ))}
        </div>
      ) : null}
    </section>
  );
}

function ReviewItem({
  review,
  onAvatarClick,
}: {
  review: CatalogProductBlueprintReview;
  onAvatarClick?: (avatarId: string) => void;
}) {
  const reviewedAt = formatDateTime(review.reviewedAt);
  const avatarName = review.avatarName || "匿名ユーザー";
  const canOpenAvatar = Boolean(review.avatarId && onAvatarClick);

  const avatarContent = (
    <>
      {review.avatarIcon ? (
        <img
          src={review.avatarIcon}
          alt={avatarName}
          className="catalog-page-review-avatar"
        />
      ) : (
        <div className="catalog-page-review-avatar-placeholder">
          {avatarName.slice(0, 1)}
        </div>
      )}

      <div className="catalog-page-review-avatar-body">
        <p className="catalog-page-review-avatar-name">{avatarName}</p>
        <p className="catalog-page-review-meta">
          {renderRatingStars(review.rating)}
          {reviewedAt !== "-" ? `・${reviewedAt}` : ""}
        </p>
      </div>
    </>
  );

  return (
    <article className="catalog-page-review-item">
      <div className="catalog-page-review-header">
        {canOpenAvatar ? (
          <button
            type="button"
            className="catalog-page-review-avatar-button"
            onClick={() => onAvatarClick?.(review.avatarId)}
          >
            {avatarContent}
          </button>
        ) : (
          <div className="catalog-page-review-avatar-content">
            {avatarContent}
          </div>
        )}
      </div>

      {review.body ? (
        <p className="catalog-page-review-body">{review.body}</p>
      ) : null}

      <p className="catalog-page-review-votes">
        参考になった: {review.helpfulVotes} / {review.totalVotes}
      </p>
    </article>
  );
}