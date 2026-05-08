// frontend/amol/src/features/token-commnet/hooks/useTokenCommentCard.ts

import { useCallback, useEffect, useMemo, useState } from "react";

import {
  dislikeTokenComment,
  fetchTokenComments,
  likeTokenComment,
  postTokenComment,
  postTokenCommentReply,
} from "../api/tokenCommentApi";
import type { TokenComment } from "../types/tokenCommentTypes";
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
  replyingCommentId: string | null;
  replyBody: string;
  replyPosting: boolean;
  setCommentBody: (value: string) => void;
  setReplyBody: (value: string) => void;
  refreshComments: () => Promise<void>;
  postComment: () => Promise<void>;
  toggleExpanded: (commentId: string) => void;
  likeComment: (commentId: string) => Promise<void>;
  dislikeComment: (commentId: string) => Promise<void>;
  startReply: (commentId: string) => void;
  cancelReply: () => void;
  submitReply: (parentCommentId: string) => Promise<void>;
};

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

  const [replyingCommentId, setReplyingCommentId] = useState<string | null>(
    null
  );
  const [replyBody, setReplyBody] = useState("");
  const [replyPosting, setReplyPosting] = useState(false);

  const commentTree = useMemo(() => buildTokenCommentTree(comments), [comments]);

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
        getErrorMessage(error, "コメントの取得に失敗しました。")
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
        getErrorMessage(error, "コメントの投稿に失敗しました。")
      );
    } finally {
      setPosting(false);
    }
  }, [commentBody, posting, refreshComments, tokenBlueprintId]);

  const toggleExpanded = useCallback((commentId: string) => {
    setExpandedIds((current) => {
      const next = new Set(current);

      if (next.has(commentId)) {
        next.delete(commentId);
      } else {
        next.add(commentId);
      }

      return next;
    });
  }, []);

  const likeComment = useCallback(
    async (commentId: string) => {
      if (!tokenBlueprintId || !commentId) {
        return;
      }

      setCommentsError("");

      try {
        await likeTokenComment({
          tokenBlueprintId,
          commentId,
        });
        await refreshComments();
      } catch (error) {
        setCommentsError(
          getErrorMessage(error, "コメントのいいねに失敗しました。")
        );
      }
    },
    [refreshComments, tokenBlueprintId]
  );

  const dislikeComment = useCallback(
    async (commentId: string) => {
      if (!tokenBlueprintId || !commentId) {
        return;
      }

      setCommentsError("");

      try {
        await dislikeTokenComment({
          tokenBlueprintId,
          commentId,
        });
        await refreshComments();
      } catch (error) {
        setCommentsError(
          getErrorMessage(error, "コメントのよくないねに失敗しました。")
        );
      }
    },
    [refreshComments, tokenBlueprintId]
  );

  const startReply = useCallback((commentId: string) => {
    setReplyingCommentId(commentId);
    setReplyBody("");
    setExpandedIds((current) => {
      const next = new Set(current);
      next.add(commentId);
      return next;
    });
  }, []);

  const cancelReply = useCallback(() => {
    setReplyingCommentId(null);
    setReplyBody("");
  }, []);

  const submitReply = useCallback(
    async (parentCommentId: string) => {
      const body = replyBody.trim();

      if (!tokenBlueprintId || !parentCommentId || !body || replyPosting) {
        return;
      }

      setReplyPosting(true);
      setCommentsError("");

      try {
        await postTokenCommentReply({
          tokenBlueprintId,
          parentCommentId,
          body,
        });

        setReplyingCommentId(null);
        setReplyBody("");
        setExpandedIds((current) => {
          const next = new Set(current);
          next.add(parentCommentId);
          return next;
        });

        await refreshComments();
      } catch (error) {
        setCommentsError(
          getErrorMessage(error, "返信コメントの投稿に失敗しました。")
        );
      } finally {
        setReplyPosting(false);
      }
    },
    [refreshComments, replyBody, replyPosting, tokenBlueprintId]
  );

  useEffect(() => {
    if (!autoFetch) {
      return;
    }

    void refreshComments();
  }, [autoFetch, refreshComments]);

  useEffect(() => {
    setComments([]);
    setCommentsError("");
    setPosting(false);
    setCommentBody("");
    setExpandedIds(new Set());
    setReplyingCommentId(null);
    setReplyBody("");
    setReplyPosting(false);
  }, [tokenBlueprintId]);

  return {
    comments,
    commentTree,
    commentsLoading,
    commentsError,
    posting,
    commentBody,
    expandedIds,
    replyingCommentId,
    replyBody,
    replyPosting,
    setCommentBody,
    setReplyBody,
    refreshComments,
    postComment,
    toggleExpanded,
    likeComment,
    dislikeComment,
    startReply,
    cancelReply,
    submitReply,
  };
}