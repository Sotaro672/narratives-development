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
  reportBrandTokenBlueprintComment,
} from "../../application/tokenBlueprintReviewDetailService";

import type {
  TokenBlueprintReviewAggregate,
  Comment,
  ReactionType,
} from "../../../../shared/types/tokenBlueprintReview";
import type {
  ReviewReportReason,
  ReviewReportResponse,
} from "../../../../shared/types/report";

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
  reportComment: (
    commentId: string,
    reason: ReviewReportReason,
    detail?: string,
  ) => Promise<ReviewReportResponse>;
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

  const normalizedTokenBlueprintReviewId =
    tokenBlueprintReviewId?.trim() ?? "";

  const [blueprint, setBlueprint] =
    useState<TokenBlueprint | null>(null);
  const [reviewAggregate, setReviewAggregate] =
    useState<TokenBlueprintReviewAggregate | null>(null);
  const [comments, setComments] =
    useState<Comment[]>([]);
  const [loading, setLoading] =
    useState(false);
  const [submitting, setSubmitting] =
    useState(false);

  const reload = useCallback(async (): Promise<void> => {
    if (!normalizedTokenBlueprintReviewId) {
      return;
    }

    setLoading(true);

    try {
      const tb = await fetchTokenBlueprintReviewDetail(
        normalizedTokenBlueprintReviewId,
      );
      setBlueprint(tb);

      try {
        const aggregate =
          await fetchTokenBlueprintAggregateForDetail(
            normalizedTokenBlueprintReviewId,
          );
        setReviewAggregate(aggregate);
      } catch {
        setReviewAggregate(null);
      }

      try {
        const result =
          await fetchTokenBlueprintCommentsForDetail(
            normalizedTokenBlueprintReviewId,
          );
        setComments(result.items);
      } catch {
        setComments([]);
      }
    } catch {
      navigate("/tokenBlueprintReview", {
        replace: true,
      });
    } finally {
      setLoading(false);
    }
  }, [
    normalizedTokenBlueprintReviewId,
    navigate,
  ]);

  useEffect(() => {
    void reload();
  }, [reload]);

  const tokenContents = useMemo<ContentFile[]>(() => {
    return toTokenContents(
      blueprint?.contentFiles ?? [],
    );
  }, [blueprint]);

  const handleBack = useCallback(() => {
    navigate("/tokenBlueprintReview", {
      replace: true,
    });
  }, [navigate]);

  const handleCreateComment = useCallback(
    async (
      body: string,
      options?: {
        commentId?: string;
        parentCommentId?: string;
      },
    ): Promise<Comment> => {
      if (!normalizedTokenBlueprintReviewId) {
        throw new Error(
          "tokenBlueprintReviewId is required",
        );
      }

      setSubmitting(true);

      try {
        const created = await postBrandComment(
          normalizedTokenBlueprintReviewId,
          body,
          options,
        );

        await reload();
        return created;
      } finally {
        setSubmitting(false);
      }
    },
    [
      normalizedTokenBlueprintReviewId,
      reload,
    ],
  );

  const handleCreateReply = useCallback(
    async (
      parentCommentId: string,
      body: string,
      options?: {
        commentId?: string;
      },
    ): Promise<Comment> => {
      if (!normalizedTokenBlueprintReviewId) {
        throw new Error(
          "tokenBlueprintReviewId is required",
        );
      }

      setSubmitting(true);

      try {
        const created = await postBrandReply(
          normalizedTokenBlueprintReviewId,
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
    [
      normalizedTokenBlueprintReviewId,
      reload,
    ],
  );

  const handleDeleteComment = useCallback(
    async (
      commentId: string,
    ): Promise<void> => {
      if (!normalizedTokenBlueprintReviewId) {
        throw new Error(
          "tokenBlueprintReviewId is required",
        );
      }

      const normalizedCommentId =
        commentId.trim();

      if (!normalizedCommentId) {
        throw new Error(
          "commentId is required",
        );
      }

      setSubmitting(true);

      try {
        await removeBrandComment(
          normalizedTokenBlueprintReviewId,
          normalizedCommentId,
        );

        await reload();
      } finally {
        setSubmitting(false);
      }
    },
    [
      normalizedTokenBlueprintReviewId,
      reload,
    ],
  );

  const handleReactToComment = useCallback(
    async (
      commentId: string,
      type: ReactionType,
    ): Promise<Comment> => {
      if (!normalizedTokenBlueprintReviewId) {
        throw new Error(
          "tokenBlueprintReviewId is required",
        );
      }

      const normalizedCommentId =
        commentId.trim();

      if (!normalizedCommentId) {
        throw new Error(
          "commentId is required",
        );
      }

      setSubmitting(true);

      try {
        const updated =
          await reactBrandToComment(
            normalizedTokenBlueprintReviewId,
            normalizedCommentId,
            type,
          );

        setComments((current) =>
          current.map((comment) =>
            comment.commentId ===
            updated.commentId
              ? updated
              : comment,
          ),
        );

        return updated;
      } finally {
        setSubmitting(false);
      }
    },
    [normalizedTokenBlueprintReviewId],
  );

  const handleReportComment = useCallback(
    async (
      commentId: string,
      reason: ReviewReportReason,
      detail?: string,
    ): Promise<ReviewReportResponse> => {
      if (!normalizedTokenBlueprintReviewId) {
        throw new Error(
          "tokenBlueprintReviewId is required",
        );
      }

      const normalizedCommentId =
        commentId.trim();

      if (!normalizedCommentId) {
        throw new Error(
          "commentId is required",
        );
      }

      const normalizedDetail =
        detail?.trim() ?? "";

      return reportBrandTokenBlueprintComment(
        normalizedTokenBlueprintReviewId,
        normalizedCommentId,
        reason,
        normalizedDetail || undefined,
      );
    },
    [normalizedTokenBlueprintReviewId],
  );

  const vm: UseTokenBlueprintReviewDetailVM = {
    blueprint,
    title: "トークン設計レビュー",
    assigneeName:
      blueprint?.assigneeName ?? "",
    createdByName:
      blueprint?.createdByName ?? "",
    createdAt:
      blueprint?.createdAt ?? "",
    updatedByName:
      blueprint?.updatedByName ?? "",
    updatedAt:
      blueprint?.updatedAt ?? "",
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
    reportComment: handleReportComment,
  };

  return {
    vm,
    handlers,
  };
}