// frontend/console/shell/src/features/notification/presentation/hooks/useReviewReportDecisionNotificationUnreadCount.ts

import {
  useCallback,
  useEffect,
  useState,
} from "react";

import {
  listReviewReportDecisionNotificationsApi,
} from "../../infrastructure/reviewReportDecisionNotificationApi";
import {
  subscribeReviewReportDecisionNotificationChanged,
} from "../notificationEvent";

const UNREAD_COUNT_PAGE = 1;
const UNREAD_COUNT_PER_PAGE = 1;

export type UseReviewReportDecisionNotificationUnreadCountParams = {
  enabled?: boolean;
};

export type UseReviewReportDecisionNotificationUnreadCountResult = {
  unreadCount: number;
  loading: boolean;
  error: string | null;
  reload: () => Promise<void>;
};

function resolveErrorMessage(
  error: unknown,
): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }

  return "未読通知件数の取得に失敗しました。";
}

export function useReviewReportDecisionNotificationUnreadCount(
  params: UseReviewReportDecisionNotificationUnreadCountParams = {},
): UseReviewReportDecisionNotificationUnreadCountResult {
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
      const result =
        await listReviewReportDecisionNotificationsApi({
          isRead: false,
          page: UNREAD_COUNT_PAGE,
          perPage: UNREAD_COUNT_PER_PAGE,
        });

      setUnreadCount(
        Math.max(
          0,
          result.totalCount,
        ),
      );
    } catch (loadError) {
      setUnreadCount(0);
      setError(
        resolveErrorMessage(loadError),
      );
    } finally {
      setLoading(false);
    }
  }, [enabled]);

  useEffect(() => {
    void load();

    return subscribeReviewReportDecisionNotificationChanged(() => {
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

export default useReviewReportDecisionNotificationUnreadCount;