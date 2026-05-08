// frontend/amol/src/features/scan-result/components/ScanResultReviewList.tsx
import Pager from "../../../components/ui/Pager";
import SectionCard from "../../../components/ui/SectionCard";
import SectionHeader from "../../../components/ui/SectionHeader";
import TextState from "../../../components/ui/TextState";
import { formatDateTime } from "../../../components/utils/date";
import type { CatalogReviewPage } from "../types";

type ScanResultReviewListProps = {
  reviews: CatalogReviewPage | null;
  reviewsError: string | null;
  busyReviews: boolean;
  reviewPage: number;
  onPrevReviewsPage: () => void;
  onNextReviewsPage: () => void;
};

export default function ScanResultReviewList(props: ScanResultReviewListProps) {
  const {
    reviews,
    reviewsError,
    busyReviews,
    reviewPage,
    onPrevReviewsPage,
    onNextReviewsPage,
  } = props;

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
          {reviews.items.map((review) => (
            <article className="scan-result-review" key={review.id}>
              <div className="scan-result-review__header">
                <strong>{review.avatarName || review.avatarId || "匿名"}</strong>
                <span>
                  {"★".repeat(Math.max(1, Math.min(5, review.rating)))}
                </span>
              </div>

              <h3>{review.title || "Review"}</h3>
              <p>{review.body}</p>
              <small>{formatDateTime(review.reviewedAt)}</small>
            </article>
          ))}
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