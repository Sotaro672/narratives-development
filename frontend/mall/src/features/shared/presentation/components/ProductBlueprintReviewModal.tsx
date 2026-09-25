// frontend/mall/src/features/shared/presentation/components/ProductBlueprintReviewModal.tsx

import RatingSelect from "../../../../components/ui/RatingSelect";
import ChatComposerModal from "./ChatComposerModal";

import "../../styles/product-blueprint-review-modal.css";

export type ProductBlueprintReviewModalProps = {
  open: boolean;
  body: string;
  rating: number;
  submitting: boolean;
  error?: string | null;
  onBodyChange: (value: string) => void;
  onRatingChange: (rating: number) => void;
  onCancel: () => void;
  onSubmit: () => void;
};

export default function ProductBlueprintReviewModal({
  open,
  body,
  rating,
  submitting,
  error,
  onBodyChange,
  onRatingChange,
  onCancel,
  onSubmit,
}: ProductBlueprintReviewModalProps) {
  const canSubmit =
    !submitting &&
    Boolean(body.trim()) &&
    Number.isInteger(rating) &&
    rating >= 1 &&
    rating <= 5;

  return (
    <ChatComposerModal
      open={open}
      title="レビューを投稿"
      content={body}
      placeholder="レビューを入力してください"
      error={error}
      submitting={submitting}
      canSubmit={canSubmit}
      submitLabel="投稿する"
      submittingLabel="投稿中..."
      onContentChange={onBodyChange}
      onCancel={onCancel}
      onSubmit={onSubmit}
      inputAriaLabel="レビュー本文"
      rows={5}
      beforeInput={
        <fieldset
          className="product-blueprint-review-modal__rating-field"
          disabled={submitting}
        >
          <label className="product-blueprint-review-modal__rating-label">
            <span>評価</span>
            <RatingSelect
              value={rating}
              onChange={onRatingChange}
            />
          </label>
        </fieldset>
      }
    />
  );
}