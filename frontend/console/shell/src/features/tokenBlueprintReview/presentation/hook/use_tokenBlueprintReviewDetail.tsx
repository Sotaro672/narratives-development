// frontend/console/shell/src/features/tokenBlueprintReview/presentation/hook/use_tokenBlueprintReviewDetail.tsx

import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import type {
  TokenBlueprint,
  ContentFile,
} from "../../../../shared/types/tokenBlueprint";

import {
  fetchTokenBlueprintReviewDetail,
  fetchTokenBlueprintAggregateForDetail,
  fetchTokenBlueprintCommentsForDetail,
  postBrandComment,
  postBrandReply,
  removeBrandComment,
  reactBrandToComment,
} from "../../application/tokenBlueprintReviewDetailService";

import type {
  TokenBlueprintReviewAggregate,
  Comment,
  ReactionType,
} from "../../../../shared/types/tokenBlueprintReview";

type UseTokenBlueprintReviewDetailVM = {
  blueprint: TokenBlueprint | null;
  title: string;
  assigneeName: string;
  createdByName: string;
  createdAt: string;
  updatedByName: string;
  updatedAt: string;
  tokenContents: ContentFile[];
  reviewAggregate: TokenBlueprintReviewAggregate | null;
  comments: Comment[];
  loading: boolean;
  submitting: boolean;
};

type UseTokenBlueprintReviewDetailHandlers = {
  onBack: () => void;
  reload: () => Promise<void>;
  createComment: (
    body: string,
    options?: { commentId?: string; parentCommentId?: string },
  ) => Promise<Comment>;
  createReply: (
    parentCommentId: string,
    body: string,
    options?: { commentId?: string },
  ) => Promise<Comment>;
  deleteComment: (commentId: string) => Promise<void>;
  reactToComment: (
    commentId: string,
    type: ReactionType,
  ) => Promise<Comment>;
};

export type UseTokenBlueprintReviewDetailResult = {
  vm: UseTokenBlueprintReviewDetailVM;
  handlers: UseTokenBlueprintReviewDetailHandlers;
};

function toTokenContents(contentFiles: ContentFile[]): ContentFile[] {
  return contentFiles.filter((file) => Boolean(file.url));
}

export function useTokenBlueprintReviewDetail(): UseTokenBlueprintReviewDetailResult {
  const navigate = useNavigate();
  const { tokenBlueprintReviewId } = useParams<{
    tokenBlueprintReviewId: string;
  }>();

  const [blueprint, setBlueprint] = useState<TokenBlueprint | null>(null);
  const [reviewAggregate, setReviewAggregate] =
    useState<TokenBlueprintReviewAggregate | null>(null);
  const [comments, setComments] = useState<Comment[]>([]);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const reload = useCallback(async (): Promise<void> => {
    if (!tokenBlueprintReviewId) {
      return;
    }

    setLoading(true);

    try {
      const tb = await fetchTokenBlueprintReviewDetail(tokenBlueprintReviewId);
      setBlueprint(tb);

      try {
        const aggregate = await fetchTokenBlueprintAggregateForDetail(
          tokenBlueprintReviewId,
        );
        setReviewAggregate(aggregate);
      } catch {
        setReviewAggregate(null);
      }

      try {
        const result = await fetchTokenBlueprintCommentsForDetail(
          tokenBlueprintReviewId,
        );
        setComments(result.items);
      } catch {
        setComments([]);
      }
    } catch {
      navigate("/tokenBlueprintReview", { replace: true });
    } finally {
      setLoading(false);
    }
  }, [tokenBlueprintReviewId, navigate]);

  useEffect(() => {
    void reload();
  }, [reload]);

  const tokenContents = useMemo<ContentFile[]>(() => {
    return toTokenContents(blueprint?.contentFiles ?? []);
  }, [blueprint]);

  const handleBack = useCallback(() => {
    navigate("/tokenBlueprintReview", { replace: true });
  }, [navigate]);

  const handleCreateComment = useCallback(
    async (
      body: string,
      options?: { commentId?: string; parentCommentId?: string },
    ): Promise<Comment> => {
      if (!tokenBlueprintReviewId) {
        throw new Error("tokenBlueprintReviewId is required");
      }

      setSubmitting(true);

      try {
        const created = await postBrandComment(
          tokenBlueprintReviewId,
          body,
          options,
        );
        await reload();
        return created;
      } finally {
        setSubmitting(false);
      }
    },
    [tokenBlueprintReviewId, reload],
  );

  const handleCreateReply = useCallback(
    async (
      parentCommentId: string,
      body: string,
      options?: { commentId?: string },
    ): Promise<Comment> => {
      if (!tokenBlueprintReviewId) {
        throw new Error("tokenBlueprintReviewId is required");
      }

      setSubmitting(true);

      try {
        const created = await postBrandReply(
          tokenBlueprintReviewId,
          parentCommentId,
          body,
          options,
        );
        await reload();
        return created;
      } finally {
        setSubmitting(false);
      }
    },
    [tokenBlueprintReviewId, reload],
  );

  const handleDeleteComment = useCallback(
    async (commentId: string): Promise<void> => {
      if (!tokenBlueprintReviewId) {
        throw new Error("tokenBlueprintReviewId is required");
      }

      setSubmitting(true);

      try {
        await removeBrandComment(tokenBlueprintReviewId, commentId);
        await reload();
      } finally {
        setSubmitting(false);
      }
    },
    [tokenBlueprintReviewId, reload],
  );

  const handleReactToComment = useCallback(
    async (
      commentId: string,
      type: ReactionType,
    ): Promise<Comment> => {
      if (!tokenBlueprintReviewId) {
        throw new Error("tokenBlueprintReviewId is required");
      }

      setSubmitting(true);

      try {
        const updated = await reactBrandToComment(
          tokenBlueprintReviewId,
          commentId,
          type,
        );

        setComments((current) =>
          current.map((comment) =>
            comment.commentId === updated.commentId ? updated : comment,
          ),
        );

        return updated;
      } finally {
        setSubmitting(false);
      }
    },
    [tokenBlueprintReviewId],
  );

  const vm: UseTokenBlueprintReviewDetailVM = {
    blueprint,
    title: "トークン設計レビュー",
    assigneeName: blueprint?.assigneeName ?? "",
    createdByName: blueprint?.createdByName ?? "",
    createdAt: blueprint?.createdAt ?? "",
    updatedByName: blueprint?.updatedByName ?? "",
    updatedAt: blueprint?.updatedAt ?? "",
    tokenContents,
    reviewAggregate,
    comments,
    loading,
    submitting,
  };

  const handlers: UseTokenBlueprintReviewDetailHandlers = {
    onBack: handleBack,
    reload,
    createComment: handleCreateComment,
    createReply: handleCreateReply,
    deleteComment: handleDeleteComment,
    reactToComment: handleReactToComment,
  };

  return { vm, handlers };
}