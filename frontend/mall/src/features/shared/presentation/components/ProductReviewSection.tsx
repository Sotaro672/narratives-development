// frontend/mall/src/features/shared/presentation/components/ProductReviewSection.tsx

import { useState } from "react";

import { formatDateTime } from "../../../../components/utils/date";
import ReportModal from "../../../report/components/ReportModal";
import { useReport } from "../../../report/hooks/useReport";

import "../../styles/product-review.css";

export type ProductReviewItem = {
  id: string;
  avatarId?: string | null;
  avatarName?: string | null;
  avatarIcon?: string | null;
  rating?: number | null;
  title?: string | null;
  body?: string | null;
  reviewedAt?: string | null;
  helpfulVotes?: number | null;
  totalVotes?: number | null;
};

export type ProductReviewSectionProps = {
  items: ProductReviewItem[];
  productBlueprintId?: string | null;
  currentAvatarId?: string | null;
  averageRating?: number | null;
  totalCount?: number | null;
  loading?: boolean;
  errorMessage?: string | null;
  emptyText?: string;
  showHelpfulVotes?: boolean;
  onAvatarClick?: (avatarId: string) => void;
  className?: string;
};

function joinClassNames(...classNames: Array<string | undefined | false>): string {
  return classNames.filter(Boolean).join(" ");
}

function renderRatingStars(value?: number | null): string {
  const rating = Math.max(0, Math.min(5, Math.trunc(Number(value ?? 0))));
  return rating <= 0 ? "評価なし" : "★".repeat(rating) + "☆".repeat(5 - rating);
}

export default function ProductReviewSection({
  items,
  productBlueprintId,
  currentAvatarId,
  averageRating,
  totalCount,
  loading = false,
  errorMessage,
  emptyText = "まだレビューはありません。",
  showHelpfulVotes = false,
  onAvatarClick,
  className,
}: ProductReviewSectionProps) {
  const [reviewsExpanded, setReviewsExpanded] = useState(false);

  const {
    target,
    isOpen,
    reason,
    detail,
    submitting,
    error: reportError,
    result,
    canSubmit,
    openProductBlueprintReviewReport,
    close: closeReport,
    setReason,
    setDetail,
    submit,
  } = useReport();

  const safeItems = Array.isArray(items) ? items : [];
  const safeErrorMessage = errorMessage?.trim() || "";
  const normalizedProductBlueprintId = productBlueprintId?.trim() || "";
  const normalizedCurrentAvatarId = currentAvatarId?.trim() || "";
  const hasSummary = Number.isFinite(averageRating) || Number.isFinite(totalCount);
  const hasMoreReviews = safeItems.length > 1;
  const visibleItems = reviewsExpanded ? safeItems : safeItems.slice(0, 1);

  const handleReport = (review: ProductReviewItem) => {
    const reviewId = review.id?.trim() || "";
    const reviewAvatarId = review.avatarId?.trim() || "";

    if (!normalizedProductBlueprintId || !normalizedCurrentAvatarId || !reviewId) {
      return;
    }

    if (reviewAvatarId && reviewAvatarId === normalizedCurrentAvatarId) {
      return;
    }

    openProductBlueprintReviewReport({
      productBlueprintId: normalizedProductBlueprintId,
      reviewId,
    });
  };

  return (
    <>
      <section className={joinClassNames("product-review", className)}>
        <div className="product-review__header">
          <h2 className="product-review__heading">レビュー</h2>
          {loading ? <span className="product-review__status">読み込み中...</span> : null}
        </div>

        {hasSummary ? (
          <div className="product-review__summary">
            {Number.isFinite(averageRating) ? (
              <strong className="product-review__average">
                {Number(averageRating).toFixed(1)}
              </strong>
            ) : null}
            {Number.isFinite(totalCount) ? (
              <span className="product-review__count">{Number(totalCount)}件</span>
            ) : null}
          </div>
        ) : null}

        {safeErrorMessage ? (
          <p className="product-review__error" role="alert">
            {safeErrorMessage}
          </p>
        ) : null}

        {!loading && !safeErrorMessage && safeItems.length === 0 ? (
          <p className="product-review__empty">{emptyText}</p>
        ) : null}

        {!safeErrorMessage && safeItems.length > 0 ? (
          <>
            <div className="product-review__list">
              {visibleItems.map((review) => (
                <ProductReviewItemView
                  key={review.id}
                  review={review}
                  productBlueprintId={normalizedProductBlueprintId}
                  currentAvatarId={normalizedCurrentAvatarId}
                  showHelpfulVotes={showHelpfulVotes}
                  onAvatarClick={onAvatarClick}
                  onReport={handleReport}
                />
              ))}
            </div>

            {hasMoreReviews ? (
              <button
                type="button"
                className="product-review__toggle"
                aria-expanded={reviewsExpanded}
                onClick={() => setReviewsExpanded((current) => !current)}
              >
                {reviewsExpanded ? "閉じる" : "詳しく見る"}
              </button>
            ) : null}
          </>
        ) : null}
      </section>

      <ReportModal
        open={isOpen}
        targetType={target?.type}
        reason={reason}
        detail={detail}
        submitting={submitting}
        error={reportError}
        result={result}
        canSubmit={canSubmit}
        onReasonChange={setReason}
        onDetailChange={setDetail}
        onSubmit={submit}
        onClose={closeReport}
      />
    </>
  );
}

function ProductReviewItemView({
  review,
  productBlueprintId,
  currentAvatarId,
  showHelpfulVotes,
  onAvatarClick,
  onReport,
}: {
  review: ProductReviewItem;
  productBlueprintId: string;
  currentAvatarId: string;
  showHelpfulVotes: boolean;
  onAvatarClick?: (avatarId: string) => void;
  onReport?: (review: ProductReviewItem) => void;
}) {
  const reviewId = review.id?.trim() || "";
  const avatarId = review.avatarId?.trim() || "";
  const avatarName = review.avatarName?.trim() || "匿名ユーザー";
  const avatarIcon = review.avatarIcon?.trim() || "";
  const reviewTitle = review.title?.trim() || "";
  const reviewBody = review.body?.trim() || "";
  const reviewedAt = review.reviewedAt?.trim() || "";
  const reviewedAtLabel = reviewedAt ? formatDateTime(reviewedAt) : "-";
  const canOpenAvatar = Boolean(avatarId && onAvatarClick);
  const isOwnReview = Boolean(
    currentAvatarId &&
    avatarId &&
    currentAvatarId === avatarId,
  );
  const canReport = Boolean(
    productBlueprintId &&
    currentAvatarId &&
    reviewId &&
    !isOwnReview &&
    onReport,
  );

  const avatarContent = (
    <>
      {avatarIcon ? (
        <img
          src={avatarIcon}
          alt={avatarName}
          className="product-review__avatar"
          loading="lazy"
        />
      ) : (
        <span className="product-review__avatar-placeholder" aria-hidden="true">
          {avatarName.slice(0, 1)}
        </span>
      )}

      <div className="product-review__author-body">
        <span className="product-review__author-name">{avatarName}</span>
        <span className="product-review__meta">
          {renderRatingStars(review.rating)}
          {reviewedAtLabel !== "-" ? `・${reviewedAtLabel}` : ""}
        </span>
      </div>
    </>
  );

  return (
    <article className="product-review__item">
      <div className="product-review__item-header">
        {canOpenAvatar ? (
          <button
            type="button"
            className="product-review__author product-review__author--button"
            onClick={() => onAvatarClick?.(avatarId)}
          >
            {avatarContent}
          </button>
        ) : (
          <div className="product-review__author">{avatarContent}</div>
        )}

        {canReport ? (
          <button
            type="button"
            className="product-review__report"
            aria-label={`${avatarName}のレビューを通報`}
            onClick={() => onReport?.(review)}
          >
            通報
          </button>
        ) : null}
      </div>

      {reviewTitle ? (
        <h3 className="product-review__title">{reviewTitle}</h3>
      ) : null}

      {reviewBody ? (
        <p className="product-review__body">{reviewBody}</p>
      ) : null}

      {showHelpfulVotes &&
      Number.isFinite(review.helpfulVotes) &&
      Number.isFinite(review.totalVotes) ? (
        <p className="product-review__votes">
          参考になった: {Number(review.helpfulVotes)} / {Number(review.totalVotes)}
        </p>
      ) : null}
    </article>
  );
}