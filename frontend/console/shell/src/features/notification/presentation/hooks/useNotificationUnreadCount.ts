// frontend/console/shell/src/features/notification/presentation/hooks/useNotificationUnreadCount.ts

import { useCallback } from "react";

import { useNewsUnreadCount } from "./useNewsUnreadCount";
import { useReportDecisionNotificationUnreadCount } from "./useReportDecisionNotificationUnreadCount";

export type UseNotificationUnreadCountParams = {
  enabled?: boolean;
};

export type UseNotificationUnreadCountResult = {
  unreadCount: number;
  reportDecisionUnreadCount: number;
  newsUnreadCount: number;
  loading: boolean;
  error: string | null;
  reload: () => Promise<void>;
};

function normalizeCount(value: unknown): number {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return 0;
  }

  return Math.max(0, Math.floor(value));
}

export function useNotificationUnreadCount(
  params: UseNotificationUnreadCountParams = {},
): UseNotificationUnreadCountResult {
  const enabled = params.enabled ?? true;

  const reportDecisionNotification =
    useReportDecisionNotificationUnreadCount({
      enabled,
    });

  const news = useNewsUnreadCount({
    enabled,
  });

  const reportDecisionUnreadCount = enabled
    ? normalizeCount(reportDecisionNotification.unreadCount)
    : 0;

  const newsUnreadCount = enabled
    ? normalizeCount(news.unreadCount)
    : 0;

  const unreadCount =
    reportDecisionUnreadCount +
    newsUnreadCount;

  const loading =
    enabled &&
    (reportDecisionNotification.loading || news.loading);

  const error = enabled
    ? reportDecisionNotification.error ??
      news.error ??
      null
    : null;

  const reload = useCallback(async () => {
    if (!enabled) {
      return;
    }

    await Promise.all([
      reportDecisionNotification.reload(),
      news.reload(),
    ]);
  }, [
    enabled,
    news.reload,
    reportDecisionNotification.reload,
  ]);

  return {
    unreadCount,
    reportDecisionUnreadCount,
    newsUnreadCount,
    loading,
    error,
    reload,
  };
}

export default useNotificationUnreadCount;