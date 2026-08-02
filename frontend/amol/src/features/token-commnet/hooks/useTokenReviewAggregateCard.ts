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
};

type UseTokenReviewAggregateCardReturn = {
  likeCount: number;
  dislikeCount: number;
  commentCount: number;
  loading: boolean;
  enabled: boolean;
  handleLike: () => Promise<void>;
  handleDislike: () => Promise<void>;
};

export function useTokenReviewAggregateCard({
  tokenBlueprintId,
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
        return;
      }

      setLoading(true);

      try {
        const result =
          await fetchTokenBlueprintReviewAggregate(
            tokenBlueprintId,
          );

        setAggregate(result);
      } catch {
        setAggregate(null);
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
      } catch {
        return;
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
      } catch {
        return;
      } finally {
        setLoading(false);
      }
    }, [
      loading,
      tokenBlueprintId,
    ]);

  useEffect(() => {
    void refreshAggregate();
  }, [refreshAggregate]);

  useEffect(() => {
    setAggregate(null);
    setLoading(false);
  }, [tokenBlueprintId]);

  return {
    likeCount,
    dislikeCount,
    commentCount,
    loading,
    enabled,
    handleLike,
    handleDislike,
  };
}