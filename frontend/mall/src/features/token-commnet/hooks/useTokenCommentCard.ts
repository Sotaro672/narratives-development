// frontend/mall/src/features/token-commnet/hooks/useTokenCommentCard.ts

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";

import {
  dislikeTokenComment,
  fetchTokenComments,
  likeTokenComment,
  postTokenComment,
  postTokenCommentReply,
} from "../api/tokenCommentApi";

import type {
  TokenComment,
} from "../../shared/types/tokenCommentTypes";

import {
  buildTokenCommentTree,
} from "../utils/commentTree";

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

type CommentReactionType =
  | "like"
  | "dislike";

function getErrorMessage(
  error: unknown,
  fallback: string,
): string {
  if (
    error instanceof Error &&
    error.message
  ) {
    return error.message;
  }

  return fallback;
}

export function useTokenCommentCard({
  tokenBlueprintId,
  autoFetch = true,
}: UseTokenCommentCardOptions): UseTokenCommentCardReturn {
  const [
    comments,
    setComments,
  ] = useState<TokenComment[]>([]);

  const [
    commentsLoading,
    setCommentsLoading,
  ] = useState(false);

  const [
    commentsError,
    setCommentsError,
  ] = useState("");

  const [
    posting,
    setPosting,
  ] = useState(false);

  const [
    commentBody,
    setCommentBody,
  ] = useState("");

  const [
    expandedIds,
    setExpandedIds,
  ] = useState<Set<string>>(
    () => new Set(),
  );

  const [
    replyThreadRootCommentId,
    setReplyThreadRootCommentId,
  ] = useState<string | null>(
    null,
  );

  const [
    replyingCommentId,
    setReplyingCommentId,
  ] = useState<string | null>(
    null,
  );

  const [
    replyBody,
    setReplyBody,
  ] = useState("");

  const [
    replyPosting,
    setReplyPosting,
  ] = useState(false);

  const commentTree = useMemo(
    () =>
      buildTokenCommentTree(
        comments,
      ),
    [comments],
  );

  const clearReply = useCallback(() => {
    setReplyThreadRootCommentId(null);
    setReplyingCommentId(null);
    setReplyBody("");
  }, []);

  const expandComment = useCallback(
    (commentId: string) => {
      const normalizedCommentId =
        commentId.trim();

      if (!normalizedCommentId) {
        return;
      }

      setExpandedIds((current) => {
        if (
          current.has(
            normalizedCommentId,
          )
        ) {
          return current;
        }

        const next =
          new Set(current);

        next.add(
          normalizedCommentId,
        );

        return next;
      });
    },
    [],
  );

  const refreshComments = useCallback(async () => {
    if (!tokenBlueprintId) {
      setComments([]);
      setCommentsError("");
      return;
    }

    setCommentsLoading(true);
    setCommentsError("");

    try {
      const response =
        await fetchTokenComments(
          tokenBlueprintId,
        );

      setComments(
        response.items,
      );
    } catch (error) {
      setComments([]);

      setCommentsError(
        getErrorMessage(
          error,
          "コメントの取得に失敗しました。",
        ),
      );
    } finally {
      setCommentsLoading(false);
    }
  }, [tokenBlueprintId]);

  const postComment = useCallback(async () => {
    const body =
      commentBody.trim();

    if (
      !tokenBlueprintId ||
      !body ||
      posting
    ) {
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
        getErrorMessage(
          error,
          "コメントの投稿に失敗しました。",
        ),
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

  const toggleExpanded = useCallback(
    (commentId: string) => {
      const normalizedCommentId =
        commentId.trim();

      if (!normalizedCommentId) {
        return;
      }

      setExpandedIds((current) => {
        const next =
          new Set(current);

        if (
          next.has(
            normalizedCommentId,
          )
        ) {
          next.delete(
            normalizedCommentId,
          );
        } else {
          next.add(
            normalizedCommentId,
          );
        }

        return next;
      });
    },
    [],
  );

  const reactToComment = useCallback(
    async (
      commentId: string,
      reactionType: CommentReactionType,
    ) => {
      const normalizedCommentId =
        commentId.trim();

      if (
        !tokenBlueprintId ||
        !normalizedCommentId
      ) {
        return;
      }

      setCommentsError("");

      try {
        const input = {
          tokenBlueprintId,
          commentId: normalizedCommentId,
        };

        if (
          reactionType ===
          "like"
        ) {
          await likeTokenComment(
            input,
          );
        } else {
          await dislikeTokenComment(
            input,
          );
        }

        await refreshComments();
      } catch (error) {
        const fallback =
          reactionType ===
          "like"
            ? "コメントのいいねに失敗しました。"
            : "コメントのよくないねに失敗しました。";

        setCommentsError(
          getErrorMessage(
            error,
            fallback,
          ),
        );
      }
    },
    [
      refreshComments,
      tokenBlueprintId,
    ],
  );

  const likeComment = useCallback(
    async (
      commentId: string,
    ) => {
      await reactToComment(
        commentId,
        "like",
      );
    },
    [reactToComment],
  );

  const dislikeComment = useCallback(
    async (
      commentId: string,
    ) => {
      await reactToComment(
        commentId,
        "dislike",
      );
    },
    [reactToComment],
  );

  const startReply = useCallback(
    (commentId: string) => {
      const normalizedCommentId =
        commentId.trim();

      if (!normalizedCommentId) {
        return;
      }

      const targetComment =
        comments.find(
          (comment) =>
            comment.commentId?.trim() ===
            normalizedCommentId,
        );

      const rootCommentId =
        targetComment?.rootCommentId?.trim() ||
        normalizedCommentId;

      setReplyThreadRootCommentId(
        rootCommentId,
      );

      setReplyingCommentId(
        normalizedCommentId,
      );

      setReplyBody("");

      expandComment(
        normalizedCommentId,
      );
    },
    [
      comments,
      expandComment,
    ],
  );

  const cancelReply = useCallback(() => {
    clearReply();
  }, [clearReply]);

  const submitReply = useCallback(
    async (
      parentCommentId: string,
    ) => {
      const normalizedParentCommentId =
        parentCommentId.trim();

      const body =
        replyBody.trim();

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
          parentCommentId:
            normalizedParentCommentId,
          body,
        });

        clearReply();

        expandComment(
          normalizedParentCommentId,
        );

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
    setExpandedIds(
      new Set(),
    );
    setReplyThreadRootCommentId(
      null,
    );
    setReplyingCommentId(
      null,
    );
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
    replyThreadRootCommentId,
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