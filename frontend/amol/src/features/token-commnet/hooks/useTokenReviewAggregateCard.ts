// frontend/amol/src/features/token-commnet/hooks/useTokenReviewAggregateCard.ts

import { useCallback, useEffect, useMemo, useState } from "react";

import {
  fetchTokenBlueprintReviewAggregate,
  upsertTokenBlueprintReaction,
} from "../api/tokenCommentApi";
import type { TokenBlueprintReviewAggregate } from "../types/tokenCommentTypes";

type UseTokenReviewAggregateCardOptions = {
  tokenBlueprintId: string;
  shareTitle?: string;
  shareText?: string;
  shareUrl?: string;
  autoFetch?: boolean;
};

type UseTokenReviewAggregateCardReturn = {
  aggregate: TokenBlueprintReviewAggregate | null;
  likeCount: number;
  dislikeCount: number;
  commentCount: number;
  loading: boolean;
  errorMessage: string;
  enabled: boolean;
  shareLoading: boolean;
  refreshAggregate: () => Promise<void>;
  handleLike: () => Promise<void>;
  handleDislike: () => Promise<void>;
  handleShare: () => Promise<void>;
};

function getErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }

  return fallback;
}

function getCurrentShareUrl(shareUrl?: string): string {
  if (shareUrl) {
    return shareUrl;
  }

  if (typeof window === "undefined") {
    return "";
  }

  return window.location.href;
}

async function copyTextToClipboard(value: string): Promise<void> {
  if (!value) {
    return;
  }

  if (
    typeof navigator !== "undefined" &&
    navigator.clipboard &&
    typeof navigator.clipboard.writeText === "function"
  ) {
    await navigator.clipboard.writeText(value);
    return;
  }

  if (typeof document === "undefined") {
    return;
  }

  const textarea = document.createElement("textarea");
  textarea.value = value;
  textarea.setAttribute("readonly", "true");
  textarea.style.position = "fixed";
  textarea.style.top = "-9999px";
  textarea.style.left = "-9999px";

  document.body.appendChild(textarea);
  textarea.select();

  try {
    document.execCommand("copy");
  } finally {
    document.body.removeChild(textarea);
  }
}

export function useTokenReviewAggregateCard({
  tokenBlueprintId,
  shareTitle = "トークン詳細",
  shareText = "",
  shareUrl,
  autoFetch = true,
}: UseTokenReviewAggregateCardOptions): UseTokenReviewAggregateCardReturn {
  const [aggregate, setAggregate] =
    useState<TokenBlueprintReviewAggregate | null>(null);
  const [loading, setLoading] = useState(false);
  const [shareLoading, setShareLoading] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");

  const enabled = Boolean(tokenBlueprintId);

  const likeCount = aggregate?.likeCount ?? 0;
  const dislikeCount = aggregate?.dislikeCount ?? 0;
  const commentCount = aggregate?.totalCommentCount ?? 0;

  const resolvedShareUrl = useMemo(
    () => getCurrentShareUrl(shareUrl),
    [shareUrl]
  );

  const refreshAggregate = useCallback(async () => {
    if (!tokenBlueprintId) {
      setAggregate(null);
      setErrorMessage("");
      return;
    }

    setLoading(true);
    setErrorMessage("");

    try {
      const result = await fetchTokenBlueprintReviewAggregate(tokenBlueprintId);
      setAggregate(result);
    } catch (error) {
      setAggregate(null);
      setErrorMessage(
        getErrorMessage(error, "レビュー集計の取得に失敗しました。")
      );
    } finally {
      setLoading(false);
    }
  }, [tokenBlueprintId]);

  const handleLike = useCallback(async () => {
    if (!tokenBlueprintId || loading) {
      return;
    }

    setLoading(true);
    setErrorMessage("");

    try {
      await upsertTokenBlueprintReaction({
        tokenBlueprintId,
        type: "like",
      });

      const result = await fetchTokenBlueprintReviewAggregate(tokenBlueprintId);
      setAggregate(result);
    } catch (error) {
      setErrorMessage(
        getErrorMessage(error, "いいねの更新に失敗しました。")
      );
    } finally {
      setLoading(false);
    }
  }, [loading, tokenBlueprintId]);

  const handleDislike = useCallback(async () => {
    if (!tokenBlueprintId || loading) {
      return;
    }

    setLoading(true);
    setErrorMessage("");

    try {
      await upsertTokenBlueprintReaction({
        tokenBlueprintId,
        type: "dislike",
      });

      const result = await fetchTokenBlueprintReviewAggregate(tokenBlueprintId);
      setAggregate(result);
    } catch (error) {
      setErrorMessage(
        getErrorMessage(error, "よくないねの更新に失敗しました。")
      );
    } finally {
      setLoading(false);
    }
  }, [loading, tokenBlueprintId]);

  const handleShare = useCallback(async () => {
    if (shareLoading) {
      return;
    }

    setShareLoading(true);
    setErrorMessage("");

    try {
      const url = resolvedShareUrl;
      const text = [shareText, url].filter(Boolean).join("\n");

      if (
        typeof navigator !== "undefined" &&
        typeof navigator.share === "function"
      ) {
        await navigator.share({
          title: shareTitle,
          text: shareText || undefined,
          url: url || undefined,
        });
        return;
      }

      await copyTextToClipboard(text || shareTitle);
    } catch (error) {
      const isShareCanceled =
        error instanceof DOMException && error.name === "AbortError";

      if (!isShareCanceled) {
        setErrorMessage(
          getErrorMessage(error, "共有リンクの作成に失敗しました。")
        );
      }
    } finally {
      setShareLoading(false);
    }
  }, [resolvedShareUrl, shareLoading, shareText, shareTitle]);

  useEffect(() => {
    if (!autoFetch) {
      return;
    }

    void refreshAggregate();
  }, [autoFetch, refreshAggregate]);

  useEffect(() => {
    setAggregate(null);
    setErrorMessage("");
    setLoading(false);
    setShareLoading(false);
  }, [tokenBlueprintId]);

  return {
    aggregate,
    likeCount,
    dislikeCount,
    commentCount,
    loading,
    errorMessage,
    enabled,
    shareLoading,
    refreshAggregate,
    handleLike,
    handleDislike,
    handleShare,
  };
}