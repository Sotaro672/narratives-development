// frontend/amol/src/features/inquiry/presentation/hooks/useInquiryBadgeCounter.tsx

import { useCallback, useEffect, useState } from "react";

import { getInquiryBadgeCount } from "../../api/inquiryApi";

type UseInquiryBadgeCounterParams = {
  enabled?: boolean;
};

type UseInquiryBadgeCounterResult = {
  badgeCount: number;
  loading: boolean;
  error: Error | null;
  loadBadgeCount: () => Promise<void>;
  clearBadgeCount: () => void;
};

function toError(caught: unknown): Error {
  return caught instanceof Error
    ? caught
    : new Error("failed to fetch inquiry badge count");
}

export function useInquiryBadgeCounter(
  params: UseInquiryBadgeCounterParams = {},
): UseInquiryBadgeCounterResult {
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
      const result = await getInquiryBadgeCount();
      setBadgeCount(result.totalCount);
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