// frontend/amol/src/features/token-commnet/components/TokenReviewAggregateCard.tsx

import { useTokenReviewAggregateCard } from "../hooks/useTokenReviewAggregateCard";

type TokenReviewAggregateCardProps = {
  tokenBlueprintId: string;
  productId: string;
  currentAvatarId: string;
  shareTitle?: string;
  shareText?: string;
  shareUrl?: string;
  onResaleClick?: () => void;
};

export default function TokenReviewAggregateCard({
  tokenBlueprintId,
  productId,
  currentAvatarId,
  shareTitle = "トークン詳細",
  shareText = "",
  shareUrl,
  onResaleClick,
}: TokenReviewAggregateCardProps) {
  const {
    likeCount,
    dislikeCount,
    commentCount,
    loading,
    enabled,
    handleLike,
    handleDislike,
  } = useTokenReviewAggregateCard({
    tokenBlueprintId,
    shareTitle,
    shareText,
    shareUrl,
  });

  const canTap = enabled && !loading;

  const canOpenResalePage =
    canTap &&
    Boolean(productId.trim()) &&
    Boolean(tokenBlueprintId.trim()) &&
    typeof onResaleClick === "function";

  const handleOpenResalePage = () => {
    if (!canOpenResalePage) {
      return;
    }

    onResaleClick();
  };

  return (
    <div className="token-review-aggregate" aria-label="トークンレビュー集計">
      <button
        type="button"
        className="token-review-aggregate__pill token-review-aggregate__pill--button"
        disabled={!canTap}
        onClick={() => void handleLike()}
      >
        <span className="token-review-aggregate__icon" aria-hidden="true">
          👍
        </span>
        <span className="token-review-aggregate__label">{likeCount}</span>
      </button>

      <button
        type="button"
        className="token-review-aggregate__pill token-review-aggregate__pill--button"
        disabled={!canTap}
        onClick={() => void handleDislike()}
      >
        <span className="token-review-aggregate__icon" aria-hidden="true">
          👎
        </span>
        <span className="token-review-aggregate__label">{dislikeCount}</span>
      </button>

      <button
        type="button"
        className="token-review-aggregate__pill token-review-aggregate__pill--button"
        disabled={!canOpenResalePage}
        onClick={handleOpenResalePage}
      >
        <span className="token-review-aggregate__icon" aria-hidden="true">
          ↗
        </span>
        <span className="token-review-aggregate__label">
          {currentAvatarId.trim() ? "出品" : "出品"}
        </span>
      </button>

      <span className="token-review-aggregate__spacer" />

      <div
        className="token-review-aggregate__pill"
        aria-label={`コメント ${commentCount} 件`}
      >
        <span className="token-review-aggregate__icon" aria-hidden="true">
          💬
        </span>
        <span className="token-review-aggregate__label">{commentCount}</span>
      </div>
    </div>
  );
}