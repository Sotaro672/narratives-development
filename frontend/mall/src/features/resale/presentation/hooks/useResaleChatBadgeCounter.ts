// frontend/amol/src/features/resale/presentation/hooks/useResaleChatBadgeCounter.ts

import { useCallback, useEffect, useState } from "react";

import { getMyResaleChatBadgeCount } from "../../api/resaleReviewApi";
import {
  subscribeResaleChatBadgeDelta,
  subscribeResaleChatBadgeRefresh,
} from "../resaleChatBadgeEvents";

type UseResaleChatBadgeCounterParams = {
  enabled?: boolean;
};

type UseResaleChatBadgeCounterResult = {
  badgeCount: number;
  loading: boolean;
  error: Error | null;
  loadBadgeCount: () => Promise<void>;
  clearBadgeCount: () => void;
};

function toError(caught: unknown): Error {
  return caught instanceof Error
    ? caught
    : new Error("failed to fetch resale chat badge count");
}

export function useResaleChatBadgeCounter(
  params: UseResaleChatBadgeCounterParams = {},
): UseResaleChatBadgeCounterResult {
  const enabled = params.enabled ?? true;

  const [badgeCount, setBadgeCount] = useState(0);
  const [loading, setLoading] = useState(enabled);
  const [error, setError] = useState<Error | null>(null);

  const clearBadgeCount = useCallback(() => {
    setBadgeCount(0);
    setLoading(false);
    setError(null);
  }, []);

  const loadBadgeCount = useCallback(async () => {
    if (!enabled) {
      clearBadgeCount();
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const result = await getMyResaleChatBadgeCount();

      setBadgeCount(
        Number.isFinite(result.unreadCommentCount)
          ? Math.max(0, Math.floor(result.unreadCommentCount))
          : 0,
      );
    } catch (caught) {
      setBadgeCount(0);
      setError(toError(caught));
    } finally {
      setLoading(false);
    }
  }, [clearBadgeCount, enabled]);

  useEffect(() => {
    void loadBadgeCount();
  }, [loadBadgeCount]);

  useEffect(() => {
    if (!enabled) {
      return;
    }

    return subscribeResaleChatBadgeDelta((delta) => {
      setBadgeCount((currentCount) =>
        Math.max(0, currentCount + delta),
      );
    });
  }, [enabled]);

  useEffect(() => {
    if (!enabled) {
      return;
    }

    return subscribeResaleChatBadgeRefresh(() => {
      void loadBadgeCount();
    });
  }, [enabled, loadBadgeCount]);

  return {
    badgeCount,
    loading,
    error,
    loadBadgeCount,
    clearBadgeCount,
  };
}