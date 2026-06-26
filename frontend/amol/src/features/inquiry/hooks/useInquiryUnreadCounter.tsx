// frontend/amol/src/features/inquiry/hooks/useInquiryUnreadCounter.tsx
import { useCallback, useEffect, useState } from "react";

import { getUnreadInquiryCount } from "../api/inquiryApi";

type UseInquiryUnreadCounterParams = {
  companyId?: string;
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
  const companyId = String(params.companyId ?? "").trim();
  const enabled = params.enabled ?? true;

  const [unreadCount, setUnreadCount] = useState<number>(0);
  const [loading, setLoading] = useState<boolean>(enabled && Boolean(companyId));
  const [error, setError] = useState<Error | null>(null);

  const clearUnreadCount = useCallback(() => {
    setUnreadCount(0);
    setLoading(false);
    setError(null);
  }, []);

  const loadUnreadCount = useCallback(async () => {
    if (!enabled || !companyId) {
      clearUnreadCount();
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const count = await getUnreadInquiryCount({
        companyId,
      });

      setUnreadCount(
        typeof count === "number" && Number.isFinite(count)
          ? Math.max(0, Math.floor(count))
          : 0,
      );
    } catch (caught) {
      setError(toError(caught));
      setUnreadCount(0);
    } finally {
      setLoading(false);
    }
  }, [clearUnreadCount, companyId, enabled]);

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