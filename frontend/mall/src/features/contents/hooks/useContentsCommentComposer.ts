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
  const editingCommentId = commentCard.editingCommentId?.trim() ?? "";
  const replyingCommentId = commentCard.replyingCommentId?.trim() ?? "";
  const isEditing = Boolean(editingCommentId);
  const isReplying = Boolean(replyingCommentId);

  const editingComment = editingCommentId
    ? commentCard.comments.find(
        (comment) => comment.commentId?.trim() === editingCommentId,
      ) ?? null
    : null;

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

  const editActionDisabled =
    commentCard.editSaving ||
    commentCard.deletingCommentId === editingCommentId ||
    loading ||
    !tokenBlueprintId;

  const canSubmitComment =
    !commentActionDisabled &&
    commentCard.commentBody.trim().length > 0;

  const canSubmitReply =
    !replyActionDisabled &&
    Boolean(replyingCommentId) &&
    commentCard.replyBody.trim().length > 0;

  const canSubmitEdit =
    !editActionDisabled &&
    Boolean(editingCommentId) &&
    Boolean(editingComment) &&
    commentCard.editBody.trim().length > 0;

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

    void commentCard.submitReply(targetCommentId);
  }, [
    canSubmitReply,
    commentCard,
  ]);

  const handleSubmitEdit = useCallback(() => {
    if (!canSubmitEdit) {
      return;
    }

    void commentCard.submitEdit();
  }, [
    canSubmitEdit,
    commentCard,
  ]);

  const composerProps: ComposerProps = {
    content: isEditing
      ? commentCard.editBody
      : isReplying
        ? commentCard.replyBody
        : commentCard.commentBody,
    placeholder: isEditing
      ? "コメントを編集…"
      : isReplying && replyingToName
        ? `@${replyingToName} 返信を書く…`
        : isReplying
          ? "返信を書く…"
          : "コメントを書く…",
    error:
      commentCard.commentsError ||
      undefined,
    submitting: isEditing
      ? commentCard.editSaving
      : isReplying
        ? commentCard.replyPosting
        : commentCard.posting,
    canSubmit: isEditing
      ? canSubmitEdit
      : isReplying
        ? canSubmitReply
        : canSubmitComment,
    disabled: isEditing
      ? editActionDisabled
      : isReplying
        ? replyActionDisabled
        : commentActionDisabled,
    submitLabel: isEditing
      ? "保存"
      : isReplying
        ? "返信"
        : "投稿",
    submittingLabel: isEditing
      ? "保存中..."
      : "投稿中...",
    beforeInput: isEditing
      ? "コメントを編集中"
      : isReplying && replyingToName
        ? `${replyingToName}に返信しています`
        : undefined,
    onContentChange: isEditing
      ? commentCard.setEditBody
      : isReplying
        ? commentCard.setReplyBody
        : commentCard.setCommentBody,
    onSubmit: isEditing
      ? handleSubmitEdit
      : isReplying
        ? handleSubmitReply
        : handleSubmitComment,
  };

  return {
    isEditing,
    editingComment,
    isReplying,
    replyingComment,
    isNestedReply,
    replyingToName,
    composerProps,
  };
}