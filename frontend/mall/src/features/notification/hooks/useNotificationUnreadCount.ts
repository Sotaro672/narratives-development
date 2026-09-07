// frontend/mall/src/features/notification/hooks/useNotificationUnreadCount.ts

import { useCallback } from "react";

import { useAnnouncementsQuery } from "../../announcement/hooks/useAnnouncementsQuery";
import { useReportDecisionNotificationUnreadCountQuery } from "./useReportDecisionNotificationsQuery";

const ANNOUNCEMENT_UNREAD_COUNT_PAGE = 1;
const ANNOUNCEMENT_UNREAD_COUNT_PER_PAGE = 100;

export type UseNotificationUnreadCountParams = {
  enabled?: boolean;
};

export type UseNotificationUnreadCountResult = {
  unreadCount: number;
  announcementUnreadCount: number;
  reportDecisionUnreadCount: number;
  loading: boolean;
  fetching: boolean;
  error: Error | null;
  reload: () => Promise<void>;
};

function normalizeCount(value: unknown): number {
  return typeof value === "number" && Number.isFinite(value)
    ? Math.max(0, Math.floor(value))
    : 0;
}

export function useNotificationUnreadCount(
  params: UseNotificationUnreadCountParams = {},
): UseNotificationUnreadCountResult {
  const enabled = params.enabled ?? true;

  const announcementsQuery = useAnnouncementsQuery({
    page: ANNOUNCEMENT_UNREAD_COUNT_PAGE,
    perPage: ANNOUNCEMENT_UNREAD_COUNT_PER_PAGE,
    enabled,
  });

  const reportDecisionQuery =
    useReportDecisionNotificationUnreadCountQuery({
      enabled,
    });

  const announcementUnreadCount = enabled
    ? normalizeCount(
        announcementsQuery.data?.items.filter(
          (item) => item.isRead === false,
        ).length,
      )
    : 0;

  const reportDecisionUnreadCount = enabled
    ? normalizeCount(reportDecisionQuery.unreadCount)
    : 0;

  const unreadCount =
    announcementUnreadCount +
    reportDecisionUnreadCount;

  const loading =
    enabled &&
    (announcementsQuery.isPending ||
      reportDecisionQuery.isPending);

  const fetching =
    enabled &&
    (announcementsQuery.isFetching ||
      reportDecisionQuery.isFetching);

  const error = enabled
    ? announcementsQuery.error ??
      reportDecisionQuery.error ??
      null
    : null;

  const reload = useCallback(async () => {
    if (!enabled) {
      return;
    }

    await Promise.all([
      announcementsQuery.refetch(),
      reportDecisionQuery.refetch(),
    ]);
  }, [
    announcementsQuery.refetch,
    enabled,
    reportDecisionQuery.refetch,
  ]);

  return {
    unreadCount,
    announcementUnreadCount,
    reportDecisionUnreadCount,
    loading,
    fetching,
    error,
    reload,
  };
}

export default useNotificationUnreadCount;