// frontend/console/shell/src/features/notification/presentation/hooks/useReportDecisionNotificationUnreadCount.ts

import { useCallback, useEffect, useState } from "react";

import { listReportDecisionNotificationsApi } from "../../infrastructure/reportDecisionNotificationApi";
import { subscribeReportDecisionNotificationChanged } from "../notificationEvent";

const UNREAD_COUNT_PAGE = 1;
const UNREAD_COUNT_PER_PAGE = 1;

export type UseReportDecisionNotificationUnreadCountParams = {
  enabled?: boolean;
};

export type UseReportDecisionNotificationUnreadCountResult = {
  unreadCount: number;
  loading: boolean;
  error: string | null;
  reload: () => Promise<void>;
};

function resolveErrorMessage(error: unknown): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }

  return "未読通知件数の取得に失敗しました。";
}

export function useReportDecisionNotificationUnreadCount(
  params: UseReportDecisionNotificationUnreadCountParams = {},
): UseReportDecisionNotificationUnreadCountResult {
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
      const result = await listReportDecisionNotificationsApi({
        isRead: false,
        page: UNREAD_COUNT_PAGE,
        perPage: UNREAD_COUNT_PER_PAGE,
      });

      setUnreadCount(Math.max(0, result.totalCount));
    } catch (loadError) {
      setUnreadCount(0);
      setError(resolveErrorMessage(loadError));
    } finally {
      setLoading(false);
    }
  }, [enabled]);

  useEffect(() => {
    void load();

    return subscribeReportDecisionNotificationChanged(() => {
      void load();
    });
  }, [load]);

  return {
    unreadCount,
    loading,
    error,
    reload: load,
  };
}

export default useReportDecisionNotificationUnreadCount;