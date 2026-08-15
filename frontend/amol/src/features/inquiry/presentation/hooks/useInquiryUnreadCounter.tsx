// frontend/amol/src/features/inquiry/presentation/hooks/useInquiryUnreadCounter.tsx

import { useCallback, useEffect, useState } from "react";

import { getUnreadInquiryCount } from "../../api/inquiryApi";

type UseInquiryUnreadCounterParams = {
  enabled?: boolean;
};

type UseInquiryUnreadCounterResult = {
  unreadCount: number;
  loading: boolean;
  error: Error | null;
  loadUnreadCount: () => Promise<void>;
  clearUnreadCount: () => void;
};

function toError(caught: unknown): Error {
  return caught instanceof Error
    ? caught
    : new Error("failed to fetch inquiry unread count");
}

export function useInquiryUnreadCounter(
  params: UseInquiryUnreadCounterParams = {},
): UseInquiryUnreadCounterResult {
  const enabled = params.enabled ?? true;

  const [unreadCount, setUnreadCount] = useState(0);
  const [loading, setLoading] = useState(enabled);
  const [error, setError] = useState<Error | null>(null);

  const clearUnreadCount = useCallback(() => {
    setUnreadCount(0);
    setLoading(false);
    setError(null);
  }, []);

  const loadUnreadCount = useCallback(async () => {
    if (!enabled) {
      clearUnreadCount();
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const count = await getUnreadInquiryCount();
      setUnreadCount(count);
    } catch (caught) {
      setUnreadCount(0);
      setError(toError(caught));
    } finally {
      setLoading(false);
    }
  }, [clearUnreadCount, enabled]);

  useEffect(() => {
    void loadUnreadCount();
  }, [loadUnreadCount]);

  return {
    unreadCount,
    loading,
    error,
    loadUnreadCount,
    clearUnreadCount,
  };
}