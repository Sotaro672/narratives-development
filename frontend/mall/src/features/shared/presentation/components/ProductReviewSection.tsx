// frontend/mall/src/features/shared/presentation/components/ProductReviewSection.tsx

import { useState } from "react";

import Alert from "../../../../components/ui/Alert";
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
  onHelpfulVote?: (
    productBlueprintId: string,
    reviewId: string,
  ) => void | Promise<void>;
  onDeleteOwnReview?: (
    productBlueprintId: string,
    reviewId: string,
  ) => void | Promise<void>;
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
  onHelpfulVote,
  onDeleteOwnReview,
  className,
}: ProductReviewSectionProps) {
  const [reviewsExpanded, setReviewsExpanded] = useState(false);
  const [helpfulVotingReviewIds, setHelpfulVotingReviewIds] = useState<Set<string>>(
    () => new Set(),
  );
  const [deletingReviewIds, setDeletingReviewIds] = useState<Set<string>>(
    () => new Set(),
  );
  const [helpfulVoteError, setHelpfulVoteError] = useState("");
  const [deleteReviewError, setDeleteReviewError] = useState("");

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

  const handleHelpfulVote = async (review: ProductReviewItem) => {
    const reviewId = review.id?.trim() || "";
    const reviewAvatarId = review.avatarId?.trim() || "";

    if (
      !normalizedProductBlueprintId ||
      !normalizedCurrentAvatarId ||
      !reviewId ||
      !onHelpfulVote
    ) {
      return;
    }

    if (reviewAvatarId && reviewAvatarId === normalizedCurrentAvatarId) {
      return;
    }

    if (helpfulVotingReviewIds.has(reviewId)) {
      return;
    }

    setHelpfulVoteError("");
    setHelpfulVotingReviewIds((current) => {
      const next = new Set(current);
      next.add(reviewId);
      return next;
    });

    try {
      await onHelpfulVote(normalizedProductBlueprintId, reviewId);
    } catch (caught) {
      setHelpfulVoteError(
        caught instanceof Error
          ? caught.message
          : "参考になったの投票に失敗しました。",
      );
    } finally {
      setHelpfulVotingReviewIds((current) => {
        const next = new Set(current);
        next.delete(reviewId);
        return next;
      });
    }
  };

  const handleDeleteOwnReview = async (review: ProductReviewItem) => {
    const reviewId = review.id?.trim() || "";
    const reviewAvatarId = review.avatarId?.trim() || "";

    if (
      !normalizedProductBlueprintId ||
      !normalizedCurrentAvatarId ||
      !reviewId ||
      !reviewAvatarId ||
      reviewAvatarId !== normalizedCurrentAvatarId ||
      !onDeleteOwnReview
    ) {
      return;
    }

    if (deletingReviewIds.has(reviewId)) {
      return;
    }

    const confirmed = window.confirm(
      "このレビューを削除します。よろしいですか？",
    );
    if (!confirmed) {
      return;
    }

    setDeleteReviewError("");
    setDeletingReviewIds((current) => {
      const next = new Set(current);
      next.add(reviewId);
      return next;
    });

    try {
      await onDeleteOwnReview(normalizedProductBlueprintId, reviewId);
    } catch (caught) {
      setDeleteReviewError(
        caught instanceof Error
          ? caught.message
          : "レビューの削除に失敗しました。",
      );
    } finally {
      setDeletingReviewIds((current) => {
        const next = new Set(current);
        next.delete(reviewId);
        return next;
      });
    }
  };

  return (
    <>
      <section className={["product-review", className].filter(Boolean).join(" ")}>
        <SectionHeader
          title="レビュー"
          titleAs="h2"
          titleSize="sm"
          right={
            loading ? (
              <TextState variant="loading">読み込み中...</TextState>
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
          <Alert variant="error">{safeErrorMessage}</Alert>
        ) : null}

        {helpfulVoteError ? (
          <Alert variant="error">{helpfulVoteError}</Alert>
        ) : null}

        {deleteReviewError ? (
          <Alert variant="error">{deleteReviewError}</Alert>
        ) : null}

        {!loading && !safeErrorMessage && safeItems.length === 0 ? (
          <TextState variant="empty">{emptyText}</TextState>
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
                  helpfulVoting={helpfulVotingReviewIds.has(review.id)}
                  deleting={deletingReviewIds.has(review.id)}
                  onAvatarClick={onAvatarClick}
                  onHelpfulVote={onHelpfulVote ? handleHelpfulVote : undefined}
                  onDeleteOwnReview={
                    onDeleteOwnReview ? handleDeleteOwnReview : undefined
                  }
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
  helpfulVoting,
  deleting,
  onAvatarClick,
  onHelpfulVote,
  onDeleteOwnReview,
  onReport,
}: {
  review: ProductReviewItem;
  productBlueprintId: string;
  currentAvatarId: string;
  showHelpfulVotes: boolean;
  helpfulVoting: boolean;
  deleting: boolean;
  onAvatarClick?: (avatarId: string) => void;
  onHelpfulVote?: (review: ProductReviewItem) => void | Promise<void>;
  onDeleteOwnReview?: (review: ProductReviewItem) => void | Promise<void>;
  onReport?: (review: ProductReviewItem) => void;
}) {
  const reviewId = review.id?.trim() || "";
  const avatarId = review.avatarId?.trim() || "";
  const avatarName = review.avatarName?.trim() || "匿名ユーザー";
  const avatarIcon = review.avatarIcon?.trim() || "";
  const reviewBody = review.body?.trim() || "";
  const reviewedAt = review.reviewedAt?.trim() || "";
  const helpfulVotes = Number.isFinite(review.helpfulVotes)
    ? Math.max(0, Number(review.helpfulVotes))
    : 0;
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
  const canDeleteOwnReview = Boolean(
    productBlueprintId &&
      currentAvatarId &&
      reviewId &&
      isOwnReview &&
      onDeleteOwnReview,
  );
  const canVoteHelpful = Boolean(
    showHelpfulVotes &&
      productBlueprintId &&
      currentAvatarId &&
      reviewId &&
      !isOwnReview &&
      onHelpfulVote,
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
        <span className="product-review__author-name">{avatarName}</span>

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
          <div className="product-review__author">{avatarContent}</div>
        )}

        {canDeleteOwnReview ? (
          <TextButton
            className="product-review__delete-button"
            disabled={deleting}
            onClick={() => void onDeleteOwnReview?.(review)}
          >
            {deleting ? "削除中..." : "削除"}
          </TextButton>
        ) : canReport ? (
          <ReportFlagButton
            label={`${avatarName}のレビューを通報`}
            onClick={() => onReport?.(review)}
          />
        ) : null}
      </div>

      {reviewBody ? (
        <p className="product-review__body">{reviewBody}</p>
      ) : null}

      {showHelpfulVotes ? (
        <div className="product-review__votes">
          {canVoteHelpful ? (
            <TextButton
              className="product-review__helpful-button"
              disabled={helpfulVoting}
              onClick={() => void onHelpfulVote?.(review)}
            >
              {helpfulVoting ? "投票中..." : "参考になった"}
            </TextButton>
          ) : (
            <span>参考になった</span>
          )}

          <span className="product-review__helpful-count">
            {helpfulVotes}
          </span>
        </div>
      ) : null}
    </article>
  );
}