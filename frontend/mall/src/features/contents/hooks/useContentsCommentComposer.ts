// frontend/mall/src/features/contents/hooks/useContentsCommentComposer.ts

import { useCallback } from "react";

import type { MobileComposerFooterProps } from "../../../components/layout/MobileComposerFooter";
import type { TokenCommentCardController } from "../../shared/types/contents";
import { getTokenCommentDisplayName } from "../../shared/types/tokenCommentTypes";

type UseContentsCommentComposerParams = {
  commentCard: TokenCommentCardController;
  loading: boolean;
  tokenBlueprintId: string;
};

type ComposerProps = Pick<
  MobileComposerFooterProps,
  | "content"
  | "placeholder"
  | "error"
  | "submitting"
  | "canSubmit"
  | "disabled"
  | "submitLabel"
  | "submittingLabel"
  | "beforeInput"
  | "onContentChange"
  | "onSubmit"
>;

export function useContentsCommentComposer({
  commentCard,
  loading,
  tokenBlueprintId,
}: UseContentsCommentComposerParams) {
  const isReplying = Boolean(commentCard.replyingCommentId);

  const replyingCommentId = commentCard.replyingCommentId?.trim() ?? "";

  const replyingComment = replyingCommentId
    ? commentCard.comments.find(
        (comment) => comment.commentId?.trim() === replyingCommentId,
      ) ?? null
    : null;

  const isNestedReply =
    Boolean(replyingComment) &&
    (replyingComment?.depth ?? 0) >= 1;

  const replyingToName =
    isNestedReply && replyingComment
      ? getTokenCommentDisplayName(replyingComment).trim()
      : "";

  const commentActionDisabled =
    commentCard.posting ||
    loading ||
    !tokenBlueprintId;

  const replyActionDisabled =
    commentCard.replyPosting ||
    loading ||
    !tokenBlueprintId;

  const canSubmitComment =
    !commentActionDisabled &&
    commentCard.commentBody.trim().length > 0;

  const canSubmitReply =
    !replyActionDisabled &&
    Boolean(replyingCommentId) &&
    commentCard.replyBody.trim().length > 0;

  const handleSubmitComment = useCallback(() => {
    if (!canSubmitComment) {
      return;
    }

    void commentCard.postComment();
  }, [
    canSubmitComment,
    commentCard,
  ]);

  const handleSubmitReply = useCallback(() => {
    const targetCommentId = commentCard.replyingCommentId?.trim() ?? "";

    if (
      !targetCommentId ||
      !canSubmitReply
    ) {
      return;
    }

    void commentCard.submitReply(
      targetCommentId,
    );
  }, [
    canSubmitReply,
    commentCard,
  ]);

  const composerProps: ComposerProps = {
    content: isReplying
      ? commentCard.replyBody
      : commentCard.commentBody,
    placeholder:
      isReplying && replyingToName
        ? `@${replyingToName} 返信を書く…`
        : isReplying
          ? "返信を書く…"
          : "コメントを書く…",
    error:
      commentCard.commentsError ||
      undefined,
    submitting: isReplying
      ? commentCard.replyPosting
      : commentCard.posting,
    canSubmit: isReplying
      ? canSubmitReply
      : canSubmitComment,
    disabled: isReplying
      ? replyActionDisabled
      : commentActionDisabled,
    submitLabel: isReplying
      ? "返信"
      : "投稿",
    submittingLabel: "投稿中...",
    beforeInput:
      isReplying && replyingToName
        ? `${replyingToName}に返信しています`
        : undefined,
    onContentChange: isReplying
      ? commentCard.setReplyBody
      : commentCard.setCommentBody,
    onSubmit: isReplying
      ? handleSubmitReply
      : handleSubmitComment,
  };

  return {
    isReplying,
    replyingComment,
    isNestedReply,
    replyingToName,
    composerProps,
  };
}