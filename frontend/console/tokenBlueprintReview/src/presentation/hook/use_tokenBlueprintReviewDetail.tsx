// frontend/console/tokenBlueprintReview/src/presentation/hook/use_tokenBlueprintReviewDetail.tsx
import { useEffect, useMemo, useState, useCallback } from "react";
import { useNavigate, useParams } from "react-router-dom";

import type {
  TokenBlueprint,
  ContentFile,
} from "../../../../tokenBlueprint/src/domain/entity/tokenBlueprint";

import {
  fetchTokenBlueprintReviewDetail,
  fetchTokenBlueprintAggregateForDetail,
  fetchTokenBlueprintCommentsForDetail,
  postBrandComment,
  postBrandReply,
  removeBrandComment,
  reactBrandToComment,
} from "../../application/tokenBlueprintReviewDetailService";

import type { FirebaseStorageTokenContent } from "../../../../shell/src/shared/types/tokenContents";

import type {
  TokenBlueprintReviewAggregate,
  Comment,
  ReactionType,
} from "../../domain/entity";

type UseTokenBlueprintReviewDetailVM = {
  blueprint: TokenBlueprint | null;
  title: string;
  assigneeName: string;

  createdByName: string;
  createdAt: string;
  updatedByName: string;
  updatedAt: string;

  tokenContents: FirebaseStorageTokenContent[];

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
  reactToComment: (commentId: string, type: ReactionType) => Promise<Comment>;
};

export type UseTokenBlueprintReviewDetailResult = {
  vm: UseTokenBlueprintReviewDetailVM;
  handlers: UseTokenBlueprintReviewDetailHandlers;
};

function toTokenContents(
  contentFiles: ContentFile[],
): FirebaseStorageTokenContent[] {
  return contentFiles
    .filter((file) => Boolean(file.url))
    .map((file) => ({
      id: file.id,
      name: file.name,
      type: file.type,
      contentType: file.contentType,
      size: file.size,
      objectPath: file.objectPath,
      url: file.url as string,
    }));
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
  const [loading, setLoading] = useState<boolean>(false);
  const [submitting, setSubmitting] = useState<boolean>(false);
  const [assignee, setAssignee] = useState<string>("");

  const reload = useCallback(async () => {
    const id = tokenBlueprintReviewId ?? "";
    if (!id) return;

    setLoading(true);

    try {
      const tb = await fetchTokenBlueprintReviewDetail(id);
      setBlueprint(tb);

      const assigneeName = tb.assigneeName || tb.assigneeId || "";
      setAssignee((prev) => prev || assigneeName);

      if (tb.companyId) {
        try {
          const agg = await fetchTokenBlueprintAggregateForDetail(
            tb.companyId,
            id,
          );
          setReviewAggregate(agg);
        } catch {
          setReviewAggregate(null);
        }
      } else {
        setReviewAggregate(null);
      }

      try {
        const res = await fetchTokenBlueprintCommentsForDetail(id);
        setComments(res.items);
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

  const createdByName = useMemo(() => {
    return blueprint?.createdByName || blueprint?.createdBy || "";
  }, [blueprint]);

  const updatedByName = useMemo(() => {
    return blueprint?.updatedByName || blueprint?.updatedBy || "";
  }, [blueprint]);

  const createdAt = useMemo(() => {
    return blueprint?.createdAt || "";
  }, [blueprint]);

  const updatedAt = useMemo(() => {
    return blueprint?.updatedAt || "";
  }, [blueprint]);

  const tokenContents: FirebaseStorageTokenContent[] = useMemo(() => {
    return toTokenContents(blueprint?.contentFiles ?? []);
  }, [blueprint]);

  const handleBack = useCallback(() => {
    navigate("/tokenBlueprintReview", { replace: true });
  }, [navigate]);

  const handleCreateComment = useCallback(
    async (
      body: string,
      options?: { commentId?: string; parentCommentId?: string },
    ) => {
      const id = tokenBlueprintReviewId ?? "";
      if (!id) {
        throw new Error("tokenBlueprintReviewId is empty");
      }

      setSubmitting(true);

      try {
        const created = await postBrandComment(id, body, options);
        setComments((prev) => [created, ...prev]);
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
    ) => {
      const id = tokenBlueprintReviewId ?? "";
      if (!id) {
        throw new Error("tokenBlueprintReviewId is empty");
      }

      setSubmitting(true);

      try {
        const created = await postBrandReply(id, parentCommentId, body, options);
        await reload();
        return created;
      } finally {
        setSubmitting(false);
      }
    },
    [tokenBlueprintReviewId, reload],
  );

  const handleDeleteComment = useCallback(
    async (commentId: string) => {
      const id = tokenBlueprintReviewId ?? "";
      if (!id) {
        throw new Error("tokenBlueprintReviewId is empty");
      }

      setSubmitting(true);

      try {
        await removeBrandComment(id, commentId);
        await reload();
      } finally {
        setSubmitting(false);
      }
    },
    [tokenBlueprintReviewId, reload],
  );

  const handleReactToComment = useCallback(
    async (commentId: string, type: ReactionType) => {
      const id = tokenBlueprintReviewId ?? "";
      if (!id) {
        throw new Error("tokenBlueprintReviewId is empty");
      }

      setSubmitting(true);

      try {
        const updated = await reactBrandToComment(id, commentId, type);
        setComments((prev) =>
          prev.map((c) => (c.commentId === updated.commentId ? updated : c)),
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
    assigneeName: assignee || blueprint?.assigneeName || blueprint?.assigneeId || "",

    createdByName,
    createdAt,
    updatedByName,
    updatedAt,

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