// frontend/mall/src/features/token-commnet/components/TokenReviewAggregateCard.tsx

import { useEffect, useState } from "react";

import Badge from "../../../components/ui/Badge";
import Chip from "../../../components/ui/Chip";
import { getMyAvatar } from "../../avatar/api/avatarApi";
import ReportModal from "../../report/components/ReportModal";
import { useReport } from "../../report/hooks/useReport";
import { useAuthState } from "../../shared/hooks/useAuthState";
import ReportFlagButton from "../../shared/presentation/components/ReportFlagButton";
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
        <Chip
          variant="neutral"
          size="md"
          disabled={!canTap}
          onClick={() => void handleLike()}
        >
          <span aria-hidden="true">👍</span>
          <span>{likeCount}</span>
        </Chip>

        <Chip
          variant="neutral"
          size="md"
          disabled={!canTap}
          onClick={() => void handleDislike()}
        >
          <span aria-hidden="true">👎</span>
          <span>{dislikeCount}</span>
        </Chip>

        <Chip
          variant="neutral"
          size="md"
          disabled={!canOpenResalePage}
          onClick={handleOpenResalePage}
        >
          <span aria-hidden="true">↗</span>
          <span>{resaleLabel}</span>
        </Chip>

        <ReportFlagButton
          label="トークンを通報"
          disabled={!canReport}
          onClick={handleOpenReport}
        />

        <span className="token-review-aggregate__spacer" />

        <Badge
          variant="neutral"
          size="md"
          aria-label={`コメント ${commentCount} 件`}
        >
          <span aria-hidden="true">💬</span>
          <span>{commentCount}</span>
        </Badge>
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