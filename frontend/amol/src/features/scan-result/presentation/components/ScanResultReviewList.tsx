// frontend/amol/src/features/scan-result/components/ScanResultReviewList.tsx
import { useNavigate } from "react-router-dom";

import MediaIcon from "../../../../components/ui/MediaIcon";
import Pager from "../../../../components/ui/Pager";
import SectionCard from "../../../../components/ui/SectionCard";
import SectionHeader from "../../../../components/ui/SectionHeader";
import TextState from "../../../../components/ui/TextState";
import { formatDateTime } from "../../../../components/utils/date";
import type { CatalogReviewPage } from "../../types";

type ScanResultReviewListProps = {
  reviews: CatalogReviewPage | null;
  reviewsError: string | null;
  busyReviews: boolean;
  reviewPage: number;
  onPrevReviewsPage: () => void;
  onNextReviewsPage: () => void;
};

function reviewAvatarFallback(avatarName: string, avatarId: string): string {
  const label = avatarName || avatarId || "匿名";
  return label.slice(0, 1);
}

export default function ScanResultReviewList(props: ScanResultReviewListProps) {
  const navigate = useNavigate();

  const {
    reviews,
    reviewsError,
    busyReviews,
    reviewPage,
    onPrevReviewsPage,
    onNextReviewsPage,
  } = props;

  const handleOpenAvatar = (avatarId: string) => {
    if (!avatarId) {
      return;
    }

    navigate(`/avatars/${encodeURIComponent(avatarId)}`);
  };

  return (
    <SectionCard>
      <SectionHeader
        title="口コミ"
        right={busyReviews ? <TextState>読み込み中...</TextState> : null}
      />

      {reviewsError ? (
        <TextState variant="error">{reviewsError}</TextState>
      ) : reviews?.items.length ? (
        <div className="scan-result-review-list">
          {reviews.items.map((review) => {
            const avatarLabel = review.avatarName || review.avatarId || "匿名";
            const hasAvatarLink = Boolean(review.avatarId);

            const avatarContent = (
              <>
                <MediaIcon
                  src={review.avatarIcon}
                  fallback={reviewAvatarFallback(
                    review.avatarName,
                    review.avatarId,
                  )}
                  size="sm"
                  shape="circle"
                  className="scan-result-review__avatar-icon"
                />

                <strong className="scan-result-review__avatar-name">
                  {avatarLabel}
                </strong>
              </>
            );

            return (
              <article className="scan-result-review" key={review.id}>
                <div className="scan-result-review__header">
                  {hasAvatarLink ? (
                    <button
                      type="button"
                      className="scan-result-review__avatar scan-result-review__avatar--button"
                      onClick={() => handleOpenAvatar(review.avatarId)}
                    >
                      {avatarContent}
                    </button>
                  ) : (
                    <div className="scan-result-review__avatar">
                      {avatarContent}
                    </div>
                  )}

                  <span className="scan-result-review__rating">
                    {"★".repeat(Math.max(1, Math.min(5, review.rating)))}
                  </span>
                </div>
                <p>{review.body}</p>
                <small>{formatDateTime(review.reviewedAt)}</small>
              </article>
            );
          })}
        </div>
      ) : (
        <TextState>口コミはまだありません。</TextState>
      )}

      <Pager
        page={reviewPage}
        hasNext={reviews?.hasNext === true}
        busy={busyReviews}
        onPrev={onPrevReviewsPage}
        onNext={onNextReviewsPage}
      />
    </SectionCard>
  );
}