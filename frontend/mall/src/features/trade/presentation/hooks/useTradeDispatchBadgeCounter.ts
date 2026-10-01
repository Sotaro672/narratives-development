// frontend/mall/src/features/trade/presentation/hooks/useTradeDispatchBadgeCounter.ts

import { useCallback, useEffect, useState } from "react";

import { fetchMyTradeChats } from "../../infrastructure/tradeApi";

type UseTradeDispatchBadgeCounterParams = {
  enabled?: boolean;
};

type UseTradeDispatchBadgeCounterResult = {
  badgeCount: number;
  loading: boolean;
  error: Error | null;
  loadBadgeCount: () => Promise<void>;
  clearBadgeCount: () => void;
};

function toError(caught: unknown): Error {
  return caught instanceof Error
    ? caught
    : new Error("failed to fetch trade dispatch badge count");
}

export function useTradeDispatchBadgeCounter(
  params: UseTradeDispatchBadgeCounterParams = {},
): UseTradeDispatchBadgeCounterResult {
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
      const result = await fetchMyTradeChats();

      const dispatchWaitingCount = result.items.filter(
        (trade) =>
          trade.viewerSide === "seller" &&
          trade.status === "active" &&
          !trade.isCancelled &&
          !trade.isDispatched,
      ).length;

      const unreadMessageCount = result.items.reduce(
        (total, trade) => {
          const unreadCount =
            Number.isFinite(trade.unreadMessageCount)
              ? Math.max(
                  0,
                  Math.floor(trade.unreadMessageCount),
                )
              : 0;

          return total + unreadCount;
        },
        0,
      );

      setBadgeCount(
        dispatchWaitingCount +
          unreadMessageCount,
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

  return {
    badgeCount,
    loading,
    error,
    loadBadgeCount,
    clearBadgeCount,
  };
}