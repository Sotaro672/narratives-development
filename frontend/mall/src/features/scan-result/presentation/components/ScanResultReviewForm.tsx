// frontend/mall/src/features/scan-result/presentation/components/ScanResultReviewForm.tsx

import Button from "../../../../components/ui/Button";
import RatingSelect from "../../../../components/ui/RatingSelect";
import SectionCard from "../../../../components/ui/SectionCard";
import Textbox from "../../../../components/ui/Textbox";

type ScanResultReviewFormProps = {
  reviewBody: string;
  reviewRating: number;
  postingReview: boolean;
  postReviewError: string | null;
  onReviewBodyChange: (value: string) => void;
  onReviewRatingChange: (rating: number) => void;
  onSubmit: () => void | Promise<void>;
};

export default function ScanResultReviewForm(props: ScanResultReviewFormProps) {
  const {
    reviewBody,
    reviewRating,
    postingReview,
    postReviewError,
    onReviewBodyChange,
    onReviewRatingChange,
    onSubmit,
  } = props;

  return (
    <SectionCard>
      <h2>口コミを投稿</h2>

      <label className="scan-result-label">
        評価
        <RatingSelect
          value={reviewRating}
          onChange={onReviewRatingChange}
        />
      </label>

      <Textbox
        label="本文"
        value={reviewBody}
        disabled={postingReview}
        error={postReviewError ?? undefined}
        placeholder="口コミを入力してください"
        onChange={(event) => onReviewBodyChange(event.target.value)}
      />

      <Button
        type="button"
        disabled={postingReview || !reviewBody.trim()}
        onClick={onSubmit}
      >
        {postingReview ? "投稿中..." : "投稿する"}
      </Button>
    </SectionCard>
  );
}