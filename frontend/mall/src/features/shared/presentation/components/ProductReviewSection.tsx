// frontend/mall/src/features/shared/presentation/components/ProductReviewSection.tsx

import { useState } from "react";

import Alert from "../../../../components/ui/Alert";
import Card from "../../../../components/ui/Card";
import DateDisplay from "../../../../components/ui/Date";
import MediaIcon from "../../../../components/ui/MediaIcon";
import SectionHeader from "../../../../components/ui/SectionHeader";
import TextButton from "../../../../components/ui/TextButton";
import TextState from "../../../../components/ui/TextState";

import ReportModal from "../../../report/components/ReportModal";
import { useReport } from "../../../report/hooks/useReport";
import ReportFlagButton from "./ReportFlagButton";

import "../../styles/product-review.css";

export type ProductReviewItem = {
  id: string;
  avatarId?: string | null;
  avatarName?: string | null;
  avatarIcon?: string | null;
  rating?: number | null;
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
      <Card
        as="section"
        variant="panel"
        padding="md"
        className={["product-review", className].filter(Boolean).join(" ")}
      >
        <SectionHeader
          title="レビュー"
          titleAs="h2"
          className="ui-section-header--title-sm"
          right={
            loading ? (
              <TextState variant="loading">
                読み込み中...
              </TextState>
            ) : undefined
          }
        />

        {hasSummary ? (
          <div className="product-review__summary">
            {Number.isFinite(averageRating) ? (
              <strong className="product-review__average">
                {Number(averageRating).toFixed(1)}
              </strong>
            ) : null}

            {Number.isFinite(totalCount) ? (
              <span className="product-review__count">
                {Number(totalCount)}件
              </span>
            ) : null}
          </div>
        ) : null}

        {safeErrorMessage ? (
          <Alert variant="error">
            {safeErrorMessage}
          </Alert>
        ) : null}

        {!loading && !safeErrorMessage && safeItems.length === 0 ? (
          <TextState variant="empty">
            {emptyText}
          </TextState>
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
              <TextButton
                className="product-review__toggle"
                aria-expanded={reviewsExpanded}
                onClick={() => setReviewsExpanded((current) => !current)}
              >
                {reviewsExpanded ? "閉じる" : "詳しく見る"}
              </TextButton>
            ) : null}
          </>
        ) : null}
      </Card>

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
  const reviewBody = review.body?.trim() || "";
  const reviewedAt = review.reviewedAt?.trim() || "";
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
      <MediaIcon
        src={avatarIcon}
        alt={avatarIcon ? avatarName : ""}
        fallback={avatarName.slice(0, 1)}
        size="sm"
        shape="circle"
      />

      <div className="product-review__author-body">
        <span className="product-review__author-name">
          {avatarName}
        </span>

        <span className="product-review__meta">
          {renderRatingStars(review.rating)}
          {reviewedAt ? (
            <>
              {"・"}
              <DateDisplay
                value={reviewedAt}
                variant="dateTime"
                className="product-review__date"
              />
            </>
          ) : null}
        </span>
      </div>
    </>
  );

  return (
    <article className="product-review__item">
      <div className="product-review__item-header">
        {canOpenAvatar ? (
          <TextButton
            className="product-review__author product-review__author--button"
            onClick={() => onAvatarClick?.(avatarId)}
          >
            {avatarContent}
          </TextButton>
        ) : (
          <div className="product-review__author">
            {avatarContent}
          </div>
        )}

        {canReport ? (
          <ReportFlagButton
            label={`${avatarName}のレビューを通報`}
            onClick={() => onReport?.(review)}
          />
        ) : null}
      </div>

      {reviewBody ? (
        <p className="product-review__body">
          {reviewBody}
        </p>
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