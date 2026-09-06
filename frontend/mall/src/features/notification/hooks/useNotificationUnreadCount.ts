// frontend/mall/src/features/notification/hooks/useNotificationUnreadCount.ts

import { useCallback } from "react";

import { useAnnouncementsQuery } from "../../announcement/hooks/useAnnouncementsQuery";
import { useReviewReportDecisionNotificationUnreadCountQuery } from "./useReviewReportDecisionNotificationsQuery";

const ANNOUNCEMENT_UNREAD_COUNT_PAGE = 1;
const ANNOUNCEMENT_UNREAD_COUNT_PER_PAGE = 100;

export type UseNotificationUnreadCountParams = {
  enabled?: boolean;
};

export type UseNotificationUnreadCountResult = {
  unreadCount: number;
  announcementUnreadCount: number;
  reviewReportDecisionUnreadCount: number;
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

  const reviewReportDecisionQuery =
    useReviewReportDecisionNotificationUnreadCountQuery({
      enabled,
    });

  const announcementUnreadCount = enabled
    ? normalizeCount(
        announcementsQuery.data?.items.filter(
          (item) => item.isRead === false,
        ).length,
      )
    : 0;

  const reviewReportDecisionUnreadCount = enabled
    ? normalizeCount(
        reviewReportDecisionQuery.unreadCount,
      )
    : 0;

  const unreadCount =
    announcementUnreadCount +
    reviewReportDecisionUnreadCount;

  const loading =
    enabled &&
    (announcementsQuery.isPending ||
      reviewReportDecisionQuery.isPending);

  const fetching =
    enabled &&
    (announcementsQuery.isFetching ||
      reviewReportDecisionQuery.isFetching);

  const error = enabled
    ? announcementsQuery.error ??
      reviewReportDecisionQuery.error ??
      null
    : null;

  const reload = useCallback(async () => {
    if (!enabled) {
      return;
    }

    await Promise.all([
      announcementsQuery.refetch(),
      reviewReportDecisionQuery.refetch(),
    ]);
  }, [
    announcementsQuery.refetch,
    enabled,
    reviewReportDecisionQuery.refetch,
  ]);

  return {
    unreadCount,
    announcementUnreadCount,
    reviewReportDecisionUnreadCount,
    loading,
    fetching,
    error,
    reload,
  };
}

export default useNotificationUnreadCount;