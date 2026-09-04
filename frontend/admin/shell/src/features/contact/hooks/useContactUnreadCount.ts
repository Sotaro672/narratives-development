// frontend/admin/shell/src/features/contact/hooks/useContactUnreadCount.ts
import { useCallback, useEffect, useRef, useState } from "react";

import { listContacts } from "../infrastructure/contactApi";

export function useContactUnreadCount() {
  const requestIdRef = useRef(0);
  const [unreadCount, setUnreadCount] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refreshUnreadCount = useCallback(async () => {
    const requestId = ++requestIdRef.current;

    setLoading(true);
    setError(null);

    try {
      const result = await listContacts({
        page: 1,
        perPage: 1,
        isRead: false,
      });

      if (requestId !== requestIdRef.current) {
        return;
      }

      setUnreadCount(result.totalCount);
    } catch (cause) {
      if (requestId !== requestIdRef.current) {
        return;
      }

      setError(
        cause instanceof Error
          ? cause.message
          : "未読問い合わせ件数の取得に失敗しました。",
      );
    } finally {
      if (requestId === requestIdRef.current) {
        setLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    void refreshUnreadCount();

    return () => {
      requestIdRef.current += 1;
    };
  }, [refreshUnreadCount]);

  return {
    unreadCount,
    loading,
    error,
    refreshUnreadCount,
  };
}