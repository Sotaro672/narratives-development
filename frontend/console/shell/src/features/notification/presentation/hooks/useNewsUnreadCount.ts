// frontend/console/shell/src/features/notification/presentation/hooks/useNewsUnreadCount.ts

import { useCallback, useEffect, useState } from "react";

import { getNewsUnreadCountApi } from "../../infrastructure/newsApi";
import { subscribeNewsNotificationChanged } from "../notificationEvent";

export type UseNewsUnreadCountParams = {
  enabled?: boolean;
};

export type UseNewsUnreadCountResult = {
  unreadCount: number;
  loading: boolean;
  error: string | null;
  reload: () => Promise<void>;
};

function normalizeUnreadCount(value: unknown): number {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return 0;
  }

  return Math.max(0, Math.floor(value));
}

function resolveErrorMessage(error: unknown): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }

  return "システム通知の未読件数取得に失敗しました。";
}

export function useNewsUnreadCount(
  params: UseNewsUnreadCountParams = {},
): UseNewsUnreadCountResult {
  const enabled = params.enabled ?? true;

  const [unreadCount, setUnreadCount] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!enabled) {
      setUnreadCount(0);
      setError(null);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const response = await getNewsUnreadCountApi();
      setUnreadCount(normalizeUnreadCount(response.unreadCount));
    } catch (loadError) {
      setUnreadCount(0);
      setError(resolveErrorMessage(loadError));
    } finally {
      setLoading(false);
    }
  }, [enabled]);

  useEffect(() => {
    void load();

    if (!enabled) {
      return;
    }

    return subscribeNewsNotificationChanged(() => {
      void load();
    });
  }, [enabled, load]);

  return {
    unreadCount,
    loading,
    error,
    reload: load,
  };
}

export default useNewsUnreadCount;