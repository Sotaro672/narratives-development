// frontend/console/tokenBlueprintReview/src/presentation/hook/use_tokenBlueprintReviewDetail.tsx
import { useEffect, useMemo, useState, useCallback } from "react";
import { useNavigate, useParams } from "react-router-dom";

import type { TokenBlueprint } from "../../../../tokenBlueprint/src/domain/entity/tokenBlueprint";

import {
  fetchTokenBlueprintReviewDetail,
  fetchTokenBlueprintAggregateForDetail,
  fetchTokenBlueprintCommentsForDetail,
  postBrandComment,
  postBrandReply,
  removeBrandComment,
  reactBrandToComment,
} from "../../application/tokenBlueprintReviewDetailService";

import type { GCSTokenContent } from "../../../../shell/src/shared/types/tokenContents";

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

  tokenContents: GCSTokenContent[];

  reviewAggregate: TokenBlueprintReviewAggregate | null;
  comments: Comment[];

  loading: boolean;
  submitting: boolean;
};

type UseTokenBlueprintReviewDetailHandlers = {
  onBack: () => void;
  reload: () => Promise<void>;
  createComment: (body: string, options?: { commentId?: string; parentCommentId?: string }) => Promise<Comment>;
  createReply: (parentCommentId: string, body: string, options?: { commentId?: string }) => Promise<Comment>;
  deleteComment: (commentId: string) => Promise<void>;
  reactToComment: (commentId: string, type: ReactionType) => Promise<Comment>;
};

export type UseTokenBlueprintReviewDetailResult = {
  vm: UseTokenBlueprintReviewDetailVM;
  handlers: UseTokenBlueprintReviewDetailHandlers;
};

function cacheBuster(url: string, t?: Date | number | string): string {
  const u = String(url || "");
  if (!u) return "";

  const lower = u.toLowerCase();
  const isSignedUrl =
    lower.includes("x-goog-signature=") ||
    lower.includes("x-goog-credential=") ||
    lower.includes("x-goog-algorithm=") ||
    lower.includes("x-goog-date=") ||
    lower.includes("x-amz-signature=") ||
    lower.includes("x-amz-credential=") ||
    lower.includes("x-amz-algorithm=") ||
    lower.includes("x-amz-date=") ||
    lower.includes("signature=") ||
    lower.includes("googleaccessid=");

  if (isSignedUrl) return u;

  try {
    const parsed = new URL(
      u,
      typeof window !== "undefined" ? window.location.origin : "http://local",
    );
    if (parsed.searchParams.has("v")) return u;
  } catch {
    // noop
  }

  let ts: number | null = null;

  if (t instanceof Date) ts = t.getTime();
  else if (typeof t === "number") ts = t;
  else if (typeof t === "string") {
    const d = Date.parse(t);
    if (!Number.isNaN(d)) ts = d;
  }

  if (!ts) return u;

  const sep = u.includes("?") ? "&" : "?";
  return `${u}${sep}v=${ts}`;
}

function toTokenContents(
  contents: unknown,
  contentsBaseUrl?: string,
  blueprintVer?: unknown,
): GCSTokenContent[] {
  if (!Array.isArray(contents)) return [];

  const base = String(contentsBaseUrl || "").replace(/\/+$/, "");
  const out: GCSTokenContent[] = [];

  for (let i = 0; i < contents.length; i++) {
    const x = contents[i];

    if (typeof x === "string") {
      const url = x;
      if (!url) continue;

      out.push({
        id: `legacy_${i + 1}`,
        name: `legacy_${i + 1}`,
        type: "document",
        url,
        size: 0,
      });
      continue;
    }

    if (x && typeof x === "object") {
      const record = x as {
        id?: unknown;
        name?: unknown;
        type?: unknown;
        size?: unknown;
        url?: unknown;
        updatedAt?: unknown;
        createdAt?: unknown;
      };

      const id = String(record.id ?? "") || `content_${i + 1}`;
      const name = String(record.name ?? "") || id;
      const type = String(record.type ?? "");
      const size = Number(record.size ?? 0) || 0;

      let url = String(record.url ?? "");
      if (!url && base && id) {
        url = `${base}/${encodeURIComponent(id)}`;
      }
      if (!url) continue;

      const normalizedType: GCSTokenContent["type"] =
        type === "image" || type === "video" || type === "pdf" || type === "document"
          ? type
          : "document";

      const ver = record.updatedAt ?? record.createdAt ?? blueprintVer;

      out.push({
        id,
        name,
        type: normalizedType,
        url: cacheBuster(url, ver as Date | number | string | undefined),
        size,
      });
    }
  }

  return out;
}

export function useTokenBlueprintReviewDetail(): UseTokenBlueprintReviewDetailResult {
  const navigate = useNavigate();
  const { tokenBlueprintReviewId } = useParams<{ tokenBlueprintReviewId: string }>();

  const [blueprint, setBlueprint] = useState<TokenBlueprint | null>(null);
  const [reviewAggregate, setReviewAggregate] =
    useState<TokenBlueprintReviewAggregate | null>(null);
  const [comments, setComments] = useState<Comment[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [submitting, setSubmitting] = useState<boolean>(false);
  const [assignee, setAssignee] = useState<string>("");

  const reload = useCallback(async () => {
    const id = String(tokenBlueprintReviewId || "");
    if (!id) return;

    setLoading(true);
    try {
      const tb = await fetchTokenBlueprintReviewDetail(id);
      setBlueprint(tb);

      const assigneeName = (() => {
        const maybe = tb as TokenBlueprint & { assigneeName?: string };
        return maybe.assigneeName || tb.assigneeId || "";
      })();
      setAssignee((prev) => prev || assigneeName);

      const companyId = (() => {
        const maybe = tb as TokenBlueprint & { companyId?: string };
        return String(maybe.companyId ?? "");
      })();

      if (companyId) {
        try {
          const agg = await fetchTokenBlueprintAggregateForDetail(companyId, id);
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
    if (!blueprint) return "";
    const maybe = blueprint as TokenBlueprint & {
      createdByName?: string;
      createdById?: string;
    };
    return String(maybe.createdByName ?? "") || String(maybe.createdById ?? "");
  }, [blueprint]);

  const updatedByName = useMemo(() => {
    if (!blueprint) return "";
    const maybe = blueprint as TokenBlueprint & {
      updatedByName?: string;
      updatedById?: string;
    };
    return String(maybe.updatedByName ?? "") || String(maybe.updatedById ?? "");
  }, [blueprint]);

  const createdAt = useMemo(() => {
    if (!blueprint) return "";
    const maybe = blueprint as TokenBlueprint & { createdAt?: string };
    return String(maybe.createdAt ?? "");
  }, [blueprint]);

  const updatedAt = useMemo(() => {
    if (!blueprint) return "";
    const maybe = blueprint as TokenBlueprint & { updatedAt?: string };
    return String(maybe.updatedAt ?? "");
  }, [blueprint]);

  const contentsBaseUrl = useMemo(() => {
    if (!blueprint) return undefined;
    const maybe = blueprint as TokenBlueprint & { contentsUrl?: string };
    const url = String(maybe.contentsUrl ?? "");
    return url || undefined;
  }, [blueprint]);

  const blueprintVer = useMemo(() => {
    if (!blueprint) return undefined;
    const maybe = blueprint as TokenBlueprint & {
      updatedAt?: string;
      createdAt?: string;
    };
    return maybe.updatedAt ?? maybe.createdAt;
  }, [blueprint]);

  const tokenContents: GCSTokenContent[] = useMemo(() => {
    if (!blueprint) return [];
    const maybe = blueprint as TokenBlueprint & { contentFiles?: unknown };
    return toTokenContents(maybe.contentFiles, contentsBaseUrl, blueprintVer);
  }, [blueprint, contentsBaseUrl, blueprintVer]);

  const handleBack = useCallback(() => {
    navigate("/tokenBlueprintReview", { replace: true });
  }, [navigate]);

  const handleCreateComment = useCallback(
    async (body: string, options?: { commentId?: string; parentCommentId?: string }) => {
      const id = String(tokenBlueprintReviewId || "");
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
    async (parentCommentId: string, body: string, options?: { commentId?: string }) => {
      const id = String(tokenBlueprintReviewId || "");
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
      const id = String(tokenBlueprintReviewId || "");
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
      const id = String(tokenBlueprintReviewId || "");
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
    assigneeName:
      assignee ||
      (() => {
        if (!blueprint) return "";
        const maybe = blueprint as TokenBlueprint & { assigneeName?: string };
        return maybe.assigneeName || blueprint.assigneeId || "";
      })(),

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