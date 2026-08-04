// frontend/amol/src/features/market/presentation/components/MarketReviewAvatar.tsx

import {
  textOrEmpty,
} from "../../../../components/utils/textOrEmpty";

import type {
  ProductBlueprintReview,
} from "../../../shared/types/review";

type MarketReviewAvatarProps = {
  review:
    ProductBlueprintReview;
};

export default function MarketReviewAvatar({
  review,
}: MarketReviewAvatarProps) {
  const avatarName =
    textOrEmpty(
      review.avatarName,
    );

  const avatarIcon =
    textOrEmpty(
      review.avatarIcon,
    );

  const avatarId =
    textOrEmpty(
      review.avatarId,
    );

  return (
    <div className="market-detail-page__review-author">
      {avatarIcon ? (
        <img
          src={avatarIcon}
          alt={
            avatarName ||
            avatarId ||
            "レビュー投稿者"
          }
          className="market-detail-page__review-author-icon"
        />
      ) : (
        <span
          className="market-detail-page__review-author-icon market-detail-page__review-author-icon--placeholder"
          aria-hidden="true"
        >
          ◎
        </span>
      )}

      <span className="market-detail-page__review-author-name">
        {avatarName ||
          avatarId ||
          "匿名"}
      </span>
    </div>
  );
}