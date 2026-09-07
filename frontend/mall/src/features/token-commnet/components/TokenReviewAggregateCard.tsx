// frontend/mall/src/features/token-commnet/components/TokenReviewAggregateCard.tsx

import { useEffect, useState } from "react";

import { getMyAvatar } from "../../avatar/api/avatarApi";
import ReportModal from "../../report/components/ReportModal";
import { useReport } from "../../report/hooks/useReport";
import { useAuthState } from "../../shared/hooks/useAuthState";
import { useTokenReviewAggregateCard } from "../hooks/useTokenReviewAggregateCard";

type TokenReviewAggregateCardProps = {
  tokenBlueprintId: string;
  productId: string;
  resaleDisabled?: boolean;
  resaleLabel?: string;
  onResaleClick?: () => void;
};

export default function TokenReviewAggregateCard({
  tokenBlueprintId,
  productId,
  resaleDisabled = false,
  resaleLabel = "出品",
  onResaleClick,
}: TokenReviewAggregateCardProps) {
  const { authResolved, isLoggedIn } = useAuthState();
  const [currentAvatarId, setCurrentAvatarId] = useState("");

  const {
    target,
    isOpen,
    reason,
    detail,
    submitting,
    error: reportError,
    result,
    canSubmit,
    openTokenBlueprintReport,
    close: closeReport,
    setReason,
    setDetail,
    submit,
  } = useReport();

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
  });

  const normalizedTokenBlueprintId = tokenBlueprintId.trim();
  const canTap = enabled && !loading;

  const canOpenResalePage =
    canTap &&
    !resaleDisabled &&
    Boolean(productId.trim()) &&
    Boolean(normalizedTokenBlueprintId) &&
    typeof onResaleClick === "function";

  const canReport =
    authResolved &&
    isLoggedIn &&
    Boolean(currentAvatarId) &&
    Boolean(normalizedTokenBlueprintId) &&
    !submitting;

  useEffect(() => {
    let cancelled = false;

    async function loadCurrentAvatar() {
      if (!authResolved || !isLoggedIn) {
        setCurrentAvatarId("");
        return;
      }

      try {
        const avatar = await getMyAvatar();

        if (cancelled) {
          return;
        }

        setCurrentAvatarId(avatar?.avatarId?.trim() ?? "");
      } catch {
        if (!cancelled) {
          setCurrentAvatarId("");
        }
      }
    }

    void loadCurrentAvatar();

    return () => {
      cancelled = true;
    };
  }, [authResolved, isLoggedIn]);

  const handleOpenResalePage = () => {
    if (!canOpenResalePage) {
      return;
    }

    onResaleClick();
  };

  const handleOpenReport = () => {
    if (!canReport) {
      return;
    }

    openTokenBlueprintReport({
      tokenBlueprintId: normalizedTokenBlueprintId,
    });
  };

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
          disabled={!canOpenResalePage}
          onClick={handleOpenResalePage}
        >
          <span className="token-review-aggregate__icon" aria-hidden="true">
            ↗
          </span>
          <span className="token-review-aggregate__label">{resaleLabel}</span>
        </button>

        <button
          type="button"
          className="token-review-aggregate__pill token-review-aggregate__pill--button"
          aria-label="トークンを通報"
          disabled={!canReport}
          onClick={handleOpenReport}
        >
          <span className="token-review-aggregate__icon" aria-hidden="true">
            ⚑
          </span>
          <span className="token-review-aggregate__label">通報</span>
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