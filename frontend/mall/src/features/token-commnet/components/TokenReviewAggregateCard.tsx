// frontend/mall/src/features/token-commnet/components/TokenReviewAggregateCard.tsx

import { useEffect, useId, useState } from "react";

import Badge from "../../../components/ui/Badge";
import Chip from "../../../components/ui/Chip";
import TextButton from "../../../components/ui/TextButton";
import { getMyAvatar } from "../../avatar/api/avatarApi";
import ReportModal from "../../report/components/ReportModal";
import { useReport } from "../../report/hooks/useReport";
import { useAuthState } from "../../shared/hooks/useAuthState";
import ReportFlagButton from "../../shared/presentation/components/ReportFlagButton";
import { useTokenReviewAggregateCard } from "../hooks/useTokenReviewAggregateCard";

const DESCRIPTION_COLLAPSE_THRESHOLD = 80;

type TokenReviewAggregateCardProps = {
  tokenBlueprintId: string;
  productId: string;
  tokenDescription?: string | null;
  resaleDisabled?: boolean;
  resaleLabel?: string;
  onResaleClick?: () => void;
};

export default function TokenReviewAggregateCard({
  tokenBlueprintId,
  productId,
  tokenDescription,
  resaleDisabled = false,
  resaleLabel = "出品",
  onResaleClick,
}: TokenReviewAggregateCardProps) {
  const { authResolved, isLoggedIn } = useAuthState();
  const [currentAvatarId, setCurrentAvatarId] = useState("");
  const [descriptionExpanded, setDescriptionExpanded] = useState(false);
  const descriptionId = useId();

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
  const normalizedTokenDescription = tokenDescription?.trim() ?? "";
  const canTap = enabled && !loading;
  const isDescriptionExpandable =
    normalizedTokenDescription.length > DESCRIPTION_COLLAPSE_THRESHOLD;

  const displayedDescription =
    isDescriptionExpandable && !descriptionExpanded
      ? `${normalizedTokenDescription.slice(0, DESCRIPTION_COLLAPSE_THRESHOLD).trimEnd()}…`
      : normalizedTokenDescription;

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

  useEffect(() => {
    setDescriptionExpanded(false);
  }, [tokenDescription]);

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

        <Badge
          variant="neutral"
          size="md"
          aria-label={`コメント ${commentCount} 件`}
        >
          <span aria-hidden="true">💬</span>
          <span>{commentCount}</span>
        </Badge>

        <Chip
          variant="neutral"
          size="md"
          disabled={!canOpenResalePage}
          onClick={handleOpenResalePage}
        >
          <span aria-hidden="true">↗</span>
          <span>{resaleLabel}</span>
        </Chip>

        <span className="token-review-aggregate__spacer" />

        <ReportFlagButton
          label="トークンを通報"
          disabled={!canReport}
          onClick={handleOpenReport}
        />
      </div>

      {normalizedTokenDescription ? (
        <div className="token-review-description">
          <p
            id={descriptionId}
            className="token-review-description__text"
          >
            {displayedDescription}
          </p>

          {isDescriptionExpandable ? (
            <TextButton
              className="token-review-description__toggle"
              aria-expanded={descriptionExpanded}
              aria-controls={descriptionId}
              onClick={() => setDescriptionExpanded((current) => !current)}
            >
              {descriptionExpanded ? "閉じる" : "詳しく見る"}
            </TextButton>
          ) : null}
        </div>
      ) : null}

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