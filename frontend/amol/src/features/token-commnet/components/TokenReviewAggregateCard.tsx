// frontend/amol/src/features/token-commnet/components/TokenReviewAggregateCard.tsx

import { useTokenReviewAggregateCard } from "../hooks/useTokenReviewAggregateCard";
import { useTokenTransferSheet } from "../hooks/useTokenTransferSheet";
import TokenTransferSheet from "./TokenTransferSheet";

type TokenReviewAggregateCardProps = {
  tokenBlueprintId: string;
  productId: string;
  currentAvatarId: string;
  shareTitle?: string;
  shareText?: string;
  shareUrl?: string;
};

export default function TokenReviewAggregateCard({
  tokenBlueprintId,
  productId,
  currentAvatarId,
  shareTitle = "トークン詳細",
  shareText = "",
  shareUrl,
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

  const {
    state: transferSheetState,
    openSheet,
    closeSheet,
    changeTab,
    refresh,
    selectTarget,
    submit,
  } = useTokenTransferSheet({
    productId,
    currentAvatarId,
  });

  const canTap = enabled && !loading;
  const canOpenTransferSheet =
    canTap &&
    !transferSheetState.loading &&
    !transferSheetState.submitting &&
    Boolean(productId.trim()) &&
    Boolean(currentAvatarId.trim());

  return (
    <>
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
          disabled={!canOpenTransferSheet}
          onClick={() => void openSheet()}
        >
          <span className="token-review-aggregate__icon" aria-hidden="true">
            ↗
          </span>
          <span className="token-review-aggregate__label">
            {transferSheetState.loading || transferSheetState.refreshing
              ? "読込中"
              : "渡す"}
          </span>

          {transferSheetState.loading || transferSheetState.refreshing ? (
            <span
              className="token-review-aggregate__loading"
              aria-label="共有準備中"
            />
          ) : null}
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

      <TokenTransferSheet
        open={transferSheetState.open}
        activeTab={transferSheetState.activeTab}
        followState={transferSheetState.followState}
        loading={transferSheetState.loading}
        refreshing={transferSheetState.refreshing}
        submitting={transferSheetState.submitting}
        errorMessage={transferSheetState.errorMessage}
        selectedTargetAvatarId={transferSheetState.selectedTargetAvatarId}
        onClose={closeSheet}
        onChangeTab={changeTab}
        onRefresh={refresh}
        onSelectTarget={selectTarget}
        onSubmit={submit}
      />
    </>
  );
}