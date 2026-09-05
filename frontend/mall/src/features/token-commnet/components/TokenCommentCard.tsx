// frontend/mall/src/features/token-commnet/components/TokenCommentCard.tsx

import { useEffect, useState } from "react";

import { getMyAvatar } from "../../avatar/api/avatarApi";
import ReviewReportModal from "../../review-report/components/ReviewReportModal";
import { useReviewReport } from "../../review-report/hooks/useReviewReport";
import { useAuthState } from "../../shared/hooks/useAuthState";
import type { TokenCommentTreeNode } from "../../shared/types/tokenCommentTypes";

import TokenCommentForm from "./TokenCommentForm";
import TokenCommentList from "./TokenCommentList";

type TokenCommentCardProps = {
  tokenBlueprintId: string;
  loading?: boolean;
  hideCommentForm?: boolean;
  commentTree: TokenCommentTreeNode[];
  commentsLoading: boolean;
  commentsError: string;
  posting: boolean;
  commentBody: string;
  expandedIds: Set<string>;
  replyingCommentId: string | null;
  replyBody: string;
  replyPosting: boolean;
  onCommentBodyChange: (value: string) => void;
  onReplyBodyChange: (value: string) => void;
  onPostComment: () => Promise<void>;
  onToggleExpanded: (commentId: string) => void;
  onLikeComment: (commentId: string) => Promise<void>;
  onDislikeComment: (commentId: string) => Promise<void>;
  onStartReply: (commentId: string) => void;
  onCancelReply: () => void;
  onSubmitReply: (parentCommentId: string) => Promise<void>;
};

export default function TokenCommentCard({
  tokenBlueprintId,
  loading = false,
  hideCommentForm = false,
  commentTree,
  commentsLoading,
  commentsError,
  posting,
  commentBody,
  expandedIds,
  replyingCommentId,
  replyBody,
  replyPosting,
  onCommentBodyChange,
  onReplyBodyChange,
  onPostComment,
  onToggleExpanded,
  onLikeComment,
  onDislikeComment,
  onStartReply,
  onCancelReply,
  onSubmitReply,
}: TokenCommentCardProps) {
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
    openTokenBlueprintCommentReport,
    close: closeReport,
    setReason,
    setDetail,
    submit,
  } = useReviewReport();

  const normalizedTokenBlueprintId = tokenBlueprintId.trim();

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

  const handleReportComment = (commentId: string) => {
    const normalizedCommentId = commentId.trim();

    if (
      !isLoggedIn ||
      !currentAvatarId ||
      !normalizedTokenBlueprintId ||
      !normalizedCommentId
    ) {
      return;
    }

    openTokenBlueprintCommentReport({
      tokenBlueprintId: normalizedTokenBlueprintId,
      commentId: normalizedCommentId,
    });
  };

  return (
    <>
      <section
        className={[
          "token-comment-card",
          hideCommentForm ? "token-comment-card--hide-form" : "",
        ]
          .filter(Boolean)
          .join(" ")}
      >
        <div className="token-comment-card__header">
          <div className="token-comment-card__title-wrap">
            <span className="token-comment-card__icon">💬</span>
            <h2 className="token-comment-card__title">コメント</h2>
          </div>
        </div>

        {!normalizedTokenBlueprintId ? (
          <p className="token-comment-card__message">
            tokenBlueprintId 未取得のためコメントを表示できません。
          </p>
        ) : (
          <>
            {!hideCommentForm ? (
              <TokenCommentForm
                value={commentBody}
                posting={posting}
                loading={loading}
                rows={1}
                onChange={onCommentBodyChange}
                onSubmit={onPostComment}
              />
            ) : null}

            {commentsError ? (
              <p className="token-comment-card__error" role="alert">
                {commentsError}
              </p>
            ) : null}

            <TokenCommentList
              tokenBlueprintId={normalizedTokenBlueprintId}
              currentAvatarId={currentAvatarId}
              commentTree={commentTree}
              commentsLoading={commentsLoading}
              expandedIds={expandedIds}
              replyingCommentId={replyingCommentId}
              replyBody={replyBody}
              replyPosting={replyPosting}
              onToggleExpanded={onToggleExpanded}
              onLike={onLikeComment}
              onDislike={onDislikeComment}
              onStartReply={onStartReply}
              onCancelReply={onCancelReply}
              onReplyBodyChange={onReplyBodyChange}
              onSubmitReply={onSubmitReply}
              onReport={handleReportComment}
            />
          </>
        )}
      </section>

      <ReviewReportModal
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