// frontend/mall/src/features/token-commnet/hooks/useTokenCommentCard.ts

import { useCallback, useEffect, useMemo, useState } from "react";

import {
  deleteTokenComment,
  dislikeTokenComment,
  fetchTokenComments,
  likeTokenComment,
  postTokenComment,
  postTokenCommentReply,
  updateTokenComment,
} from "../api/tokenCommentApi";

import type { TokenComment } from "../../shared/types/tokenCommentTypes";
import { buildTokenCommentTree } from "../utils/commentTree";

export type UseTokenCommentCardOptions = {
  tokenBlueprintId: string;
  autoFetch?: boolean;
};

export type UseTokenCommentCardReturn = {
  comments: TokenComment[];
  commentTree: ReturnType<typeof buildTokenCommentTree>;
  commentsLoading: boolean;
  commentsError: string;
  posting: boolean;
  commentBody: string;
  expandedIds: Set<string>;
  replyThreadRootCommentId: string | null;
  replyingCommentId: string | null;
  replyBody: string;
  replyPosting: boolean;
  editingCommentId: string | null;
  editBody: string;
  editSaving: boolean;
  deletingCommentId: string | null;
  setCommentBody: (value: string) => void;
  setReplyBody: (value: string) => void;
  setEditBody: (value: string) => void;
  refreshComments: () => Promise<void>;
  postComment: () => Promise<void>;
  toggleExpanded: (commentId: string) => void;
  likeComment: (commentId: string) => Promise<void>;
  dislikeComment: (commentId: string) => Promise<void>;
  startReply: (commentId: string) => void;
  cancelReply: () => void;
  submitReply: (parentCommentId: string) => Promise<void>;
  startEdit: (commentId: string) => void;
  cancelEdit: () => void;
  submitEdit: () => Promise<void>;
  deleteComment: (commentId: string) => Promise<void>;
};

type CommentReactionType = "like" | "dislike";

function getErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }

  return fallback;
}

export function useTokenCommentCard({
  tokenBlueprintId,
  autoFetch = true,
}: UseTokenCommentCardOptions): UseTokenCommentCardReturn {
  const [comments, setComments] = useState<TokenComment[]>([]);
  const [commentsLoading, setCommentsLoading] = useState(false);
  const [commentsError, setCommentsError] = useState("");
  const [posting, setPosting] = useState(false);
  const [commentBody, setCommentBody] = useState("");
  const [expandedIds, setExpandedIds] = useState<Set<string>>(() => new Set());
  const [replyThreadRootCommentId, setReplyThreadRootCommentId] = useState<string | null>(null);
  const [replyingCommentId, setReplyingCommentId] = useState<string | null>(null);
  const [replyBody, setReplyBody] = useState("");
  const [replyPosting, setReplyPosting] = useState(false);
  const [editingCommentId, setEditingCommentId] = useState<string | null>(null);
  const [editBody, setEditBody] = useState("");
  const [editSaving, setEditSaving] = useState(false);
  const [deletingCommentId, setDeletingCommentId] = useState<string | null>(null);

  const commentTree = useMemo(
    () => buildTokenCommentTree(comments),
    [comments],
  );

  const clearReply = useCallback(() => {
    setReplyThreadRootCommentId(null);
    setReplyingCommentId(null);
    setReplyBody("");
  }, []);

  const clearEdit = useCallback(() => {
    setEditingCommentId(null);
    setEditBody("");
  }, []);

  const expandComment = useCallback((commentId: string) => {
    const normalizedCommentId = commentId.trim();

    if (!normalizedCommentId) {
      return;
    }

    setExpandedIds((current) => {
      if (current.has(normalizedCommentId)) {
        return current;
      }

      const next = new Set(current);
      next.add(normalizedCommentId);
      return next;
    });
  }, []);

  const refreshComments = useCallback(async () => {
    if (!tokenBlueprintId) {
      setComments([]);
      setCommentsError("");
      return;
    }

    setCommentsLoading(true);
    setCommentsError("");

    try {
      const response = await fetchTokenComments(tokenBlueprintId);
      setComments(response.items);
    } catch (error) {
      setComments([]);
      setCommentsError(
        getErrorMessage(error, "コメントの取得に失敗しました。"),
      );
    } finally {
      setCommentsLoading(false);
    }
  }, [tokenBlueprintId]);

  const postComment = useCallback(async () => {
    const body = commentBody.trim();

    if (!tokenBlueprintId || !body || posting) {
      return;
    }

    setPosting(true);
    setCommentsError("");

    try {
      await postTokenComment({
        tokenBlueprintId,
        body,
      });

      setCommentBody("");
      await refreshComments();
    } catch (error) {
      setCommentsError(
        getErrorMessage(error, "コメントの投稿に失敗しました。"),
      );
    } finally {
      setPosting(false);
    }
  }, [
    commentBody,
    posting,
    refreshComments,
    tokenBlueprintId,
  ]);

  const toggleExpanded = useCallback((commentId: string) => {
    const normalizedCommentId = commentId.trim();

    if (!normalizedCommentId) {
      return;
    }

    setExpandedIds((current) => {
      const next = new Set(current);

      if (next.has(normalizedCommentId)) {
        next.delete(normalizedCommentId);
      } else {
        next.add(normalizedCommentId);
      }

      return next;
    });
  }, []);

  const reactToComment = useCallback(
    async (
      commentId: string,
      reactionType: CommentReactionType,
    ) => {
      const normalizedCommentId = commentId.trim();

      if (!tokenBlueprintId || !normalizedCommentId) {
        return;
      }

      setCommentsError("");

      try {
        const input = {
          tokenBlueprintId,
          commentId: normalizedCommentId,
        };

        if (reactionType === "like") {
          await likeTokenComment(input);
        } else {
          await dislikeTokenComment(input);
        }

        await refreshComments();
      } catch (error) {
        const fallback =
          reactionType === "like"
            ? "コメントのいいねに失敗しました。"
            : "コメントのよくないねに失敗しました。";

        setCommentsError(
          getErrorMessage(error, fallback),
        );
      }
    },
    [
      refreshComments,
      tokenBlueprintId,
    ],
  );

  const likeComment = useCallback(
    async (commentId: string) => {
      await reactToComment(commentId, "like");
    },
    [reactToComment],
  );

  const dislikeComment = useCallback(
    async (commentId: string) => {
      await reactToComment(commentId, "dislike");
    },
    [reactToComment],
  );

  const startReply = useCallback(
    (commentId: string) => {
      const normalizedCommentId = commentId.trim();

      if (!normalizedCommentId || editSaving) {
        return;
      }

      const targetComment = comments.find(
        (comment) =>
          comment.commentId?.trim() === normalizedCommentId,
      );

      if (!targetComment || targetComment.deleted) {
        return;
      }

      const rootCommentId =
        targetComment.rootCommentId?.trim() ||
        normalizedCommentId;

      clearEdit();
      setReplyThreadRootCommentId(rootCommentId);
      setReplyingCommentId(normalizedCommentId);
      setReplyBody("");
      expandComment(normalizedCommentId);
    },
    [
      clearEdit,
      comments,
      editSaving,
      expandComment,
    ],
  );

  const cancelReply = useCallback(() => {
    clearReply();
  }, [clearReply]);

  const submitReply = useCallback(
    async (parentCommentId: string) => {
      const normalizedParentCommentId = parentCommentId.trim();
      const body = replyBody.trim();

      if (
        !tokenBlueprintId ||
        !normalizedParentCommentId ||
        !body ||
        replyPosting
      ) {
        return;
      }

      setReplyPosting(true);
      setCommentsError("");

      try {
        await postTokenCommentReply({
          tokenBlueprintId,
          parentCommentId: normalizedParentCommentId,
          body,
        });

        clearReply();
        expandComment(normalizedParentCommentId);
        await refreshComments();
      } catch (error) {
        setCommentsError(
          getErrorMessage(
            error,
            "返信コメントの投稿に失敗しました。",
          ),
        );
      } finally {
        setReplyPosting(false);
      }
    },
    [
      clearReply,
      expandComment,
      refreshComments,
      replyBody,
      replyPosting,
      tokenBlueprintId,
    ],
  );

  const startEdit = useCallback(
    (commentId: string) => {
      const normalizedCommentId = commentId.trim();

      if (
        !normalizedCommentId ||
        editSaving ||
        deletingCommentId === normalizedCommentId
      ) {
        return;
      }

      const targetComment = comments.find(
        (comment) =>
          comment.commentId?.trim() === normalizedCommentId,
      );

      if (!targetComment || targetComment.deleted) {
        return;
      }

      setCommentsError("");
      setEditingCommentId(normalizedCommentId);
      setEditBody(targetComment.body);
    },
    [
      comments,
      deletingCommentId,
      editSaving,
    ],
  );

  const cancelEdit = useCallback(() => {
    if (editSaving) {
      return;
    }

    clearEdit();
  }, [
    clearEdit,
    editSaving,
  ]);

  const submitEdit = useCallback(async () => {
    const normalizedCommentId = editingCommentId?.trim() ?? "";
    const body = editBody.trim();

    if (
      !tokenBlueprintId ||
      !normalizedCommentId ||
      !body ||
      editSaving ||
      deletingCommentId === normalizedCommentId
    ) {
      return;
    }

    setEditSaving(true);
    setCommentsError("");

    try {
      await updateTokenComment({
        tokenBlueprintId,
        commentId: normalizedCommentId,
        body,
      });

      clearEdit();
      await refreshComments();
    } catch (error) {
      setCommentsError(
        getErrorMessage(
          error,
          "コメントの編集に失敗しました。",
        ),
      );
    } finally {
      setEditSaving(false);
    }
  }, [
    clearEdit,
    deletingCommentId,
    editBody,
    editSaving,
    editingCommentId,
    refreshComments,
    tokenBlueprintId,
  ]);

  const deleteComment = useCallback(
    async (commentId: string) => {
      const normalizedCommentId = commentId.trim();

      if (
        !tokenBlueprintId ||
        !normalizedCommentId ||
        deletingCommentId ||
        (
          editSaving &&
          editingCommentId === normalizedCommentId
        )
      ) {
        return;
      }

      setDeletingCommentId(normalizedCommentId);
      setCommentsError("");

      try {
        await deleteTokenComment({
          tokenBlueprintId,
          commentId: normalizedCommentId,
        });

        if (editingCommentId === normalizedCommentId) {
          clearEdit();
        }

        if (replyingCommentId === normalizedCommentId) {
          setReplyingCommentId(null);
          setReplyBody("");
        }

        await refreshComments();
      } catch (error) {
        setCommentsError(
          getErrorMessage(
            error,
            "コメントの削除に失敗しました。",
          ),
        );
      } finally {
        setDeletingCommentId(null);
      }
    },
    [
      clearEdit,
      deletingCommentId,
      editSaving,
      editingCommentId,
      refreshComments,
      replyingCommentId,
      tokenBlueprintId,
    ],
  );

  useEffect(() => {
    if (!autoFetch) {
      return;
    }

    void refreshComments();
  }, [
    autoFetch,
    refreshComments,
  ]);

  useEffect(() => {
    setComments([]);
    setCommentsError("");
    setPosting(false);
    setCommentBody("");
    setExpandedIds(new Set());
    setReplyThreadRootCommentId(null);
    setReplyingCommentId(null);
    setReplyBody("");
    setReplyPosting(false);
    setEditingCommentId(null);
    setEditBody("");
    setEditSaving(false);
    setDeletingCommentId(null);
  }, [tokenBlueprintId]);

  return {
    comments,
    commentTree,
    commentsLoading,
    commentsError,
    posting,
    commentBody,
    expandedIds,
    replyThreadRootCommentId,
    replyingCommentId,
    replyBody,
    replyPosting,
    editingCommentId,
    editBody,
    editSaving,
    deletingCommentId,
    setCommentBody,
    setReplyBody,
    setEditBody,
    refreshComments,
    postComment,
    toggleExpanded,
    likeComment,
    dislikeComment,
    startReply,
    cancelReply,
    submitReply,
    startEdit,
    cancelEdit,
    submitEdit,
    deleteComment,
  };
}