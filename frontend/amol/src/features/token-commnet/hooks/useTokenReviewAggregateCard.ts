// frontend/amol/src/features/token-commnet/hooks/useTokenReviewAggregateCard.ts

import {
  useCallback,
  useEffect,
  useState,
} from "react";

import {
  fetchTokenBlueprintReviewAggregate,
  upsertTokenBlueprintReaction,
} from "../api/tokenCommentApi";

import type {
  TokenBlueprintReviewAggregate,
} from "../types/tokenCommentTypes";

type UseTokenReviewAggregateCardOptions = {
  tokenBlueprintId: string;
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
  refreshAggregate: () => Promise<void>;
  handleLike: () => Promise<void>;
  handleDislike: () => Promise<void>;
};

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

export function useTokenReviewAggregateCard({
  tokenBlueprintId,
  autoFetch = true,
}: UseTokenReviewAggregateCardOptions): UseTokenReviewAggregateCardReturn {
  const [
    aggregate,
    setAggregate,
  ] =
    useState<TokenBlueprintReviewAggregate | null>(
      null,
    );

  const [
    loading,
    setLoading,
  ] = useState(false);

  const [
    errorMessage,
    setErrorMessage,
  ] = useState("");

  const enabled =
    Boolean(tokenBlueprintId);

  const likeCount =
    aggregate?.likeCount ?? 0;

  const dislikeCount =
    aggregate?.dislikeCount ?? 0;

  const commentCount =
    aggregate?.totalCommentCount ?? 0;

  const refreshAggregate =
    useCallback(async () => {
      if (!tokenBlueprintId) {
        setAggregate(null);
        setErrorMessage("");

        return;
      }

      setLoading(true);
      setErrorMessage("");

      try {
        const result =
          await fetchTokenBlueprintReviewAggregate(
            tokenBlueprintId,
          );

        setAggregate(result);
      } catch (error) {
        setAggregate(null);

        setErrorMessage(
          getErrorMessage(
            error,
            "レビュー集計の取得に失敗しました。",
          ),
        );
      } finally {
        setLoading(false);
      }
    }, [tokenBlueprintId]);

  const handleLike =
    useCallback(async () => {
      if (
        !tokenBlueprintId ||
        loading
      ) {
        return;
      }

      setLoading(true);
      setErrorMessage("");

      try {
        await upsertTokenBlueprintReaction({
          tokenBlueprintId,
          type: "like",
        });

        const result =
          await fetchTokenBlueprintReviewAggregate(
            tokenBlueprintId,
          );

        setAggregate(result);
      } catch (error) {
        setErrorMessage(
          getErrorMessage(
            error,
            "いいねの更新に失敗しました。",
          ),
        );
      } finally {
        setLoading(false);
      }
    }, [
      loading,
      tokenBlueprintId,
    ]);

  const handleDislike =
    useCallback(async () => {
      if (
        !tokenBlueprintId ||
        loading
      ) {
        return;
      }

      setLoading(true);
      setErrorMessage("");

      try {
        await upsertTokenBlueprintReaction({
          tokenBlueprintId,
          type: "dislike",
        });

        const result =
          await fetchTokenBlueprintReviewAggregate(
            tokenBlueprintId,
          );

        setAggregate(result);
      } catch (error) {
        setErrorMessage(
          getErrorMessage(
            error,
            "よくないねの更新に失敗しました。",
          ),
        );
      } finally {
        setLoading(false);
      }
    }, [
      loading,
      tokenBlueprintId,
    ]);

  useEffect(() => {
    if (!autoFetch) {
      return;
    }

    void refreshAggregate();
  }, [
    autoFetch,
    refreshAggregate,
  ]);

  useEffect(() => {
    setAggregate(null);
    setErrorMessage("");
    setLoading(false);
  }, [tokenBlueprintId]);

  return {
    aggregate,
    likeCount,
    dislikeCount,
    commentCount,
    loading,
    errorMessage,
    enabled,
    refreshAggregate,
    handleLike,
    handleDislike,
  };
}