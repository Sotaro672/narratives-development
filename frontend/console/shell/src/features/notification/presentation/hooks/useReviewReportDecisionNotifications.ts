// frontend/console/shell/src/features/notification/presentation/hooks/useReportDecisionNotifications.ts

import { useCallback, useEffect, useState } from "react";

import {
  listReportDecisionNotificationsApi,
  markReportDecisionNotificationReadApi,
  type ReportDecisionNotification,
  type ReportDecisionNotificationPage,
} from "../../infrastructure/reportDecisionNotificationApi";
import { emitReportDecisionNotificationChanged } from "../notificationEvent";

const DEFAULT_PAGE = 1;
const DEFAULT_PER_PAGE = 20;

export type UseReportDecisionNotificationsParams = {
  page?: number;
  perPage?: number;
  isRead?: boolean;
  enabled?: boolean;
};

export type UseReportDecisionNotificationsResult = {
  notifications: ReportDecisionNotification[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
  loading: boolean;
  error: string | null;
  markingReadId: string | null;
  reload: () => Promise<void>;
  markRead: (notificationId: string) => Promise<ReportDecisionNotification | null>;
};

function createEmptyResult(
  page: number,
  perPage: number,
): ReportDecisionNotificationPage {
  return {
    items: [],
    totalCount: 0,
    totalPages: 0,
    page,
    perPage,
  };
}

function resolveErrorMessage(error: unknown): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }

  return "裁定結果通知の取得に失敗しました。";
}

export function useReportDecisionNotifications(
  params: UseReportDecisionNotificationsParams = {},
): UseReportDecisionNotificationsResult {
  const page = params.page ?? DEFAULT_PAGE;
  const perPage = params.perPage ?? DEFAULT_PER_PAGE;
  const enabled = params.enabled ?? true;
  const isRead = params.isRead;

  const [result, setResult] = useState<ReportDecisionNotificationPage>(
    () => createEmptyResult(page, perPage),
  );
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [markingReadId, setMarkingReadId] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!enabled) {
      setResult(createEmptyResult(page, perPage));
      setError(null);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const response = await listReportDecisionNotificationsApi({
        page,
        perPage,
        ...(isRead !== undefined ? { isRead } : {}),
      });

      setResult(response);
    } catch (loadError) {
      setResult(createEmptyResult(page, perPage));
      setError(resolveErrorMessage(loadError));
    } finally {
      setLoading(false);
    }
  }, [enabled, isRead, page, perPage]);

  useEffect(() => {
    void load();
  }, [load]);

  const markRead = useCallback(
    async (
      notificationId: string,
    ): Promise<ReportDecisionNotification | null> => {
      if (!notificationId || markingReadId !== null) {
        return null;
      }

      setMarkingReadId(notificationId);
      setError(null);

      try {
        const updated = await markReportDecisionNotificationReadApi(
          notificationId,
        );

        setResult((current) => ({
          ...current,
          items: current.items.map((notification) =>
            notification.id === updated.id ? updated : notification,
          ),
        }));

        await load();
        emitReportDecisionNotificationChanged();

        return updated;
      } catch (markReadError) {
        setError(resolveErrorMessage(markReadError));
        return null;
      } finally {
        setMarkingReadId(null);
      }
    },
    [load, markingReadId],
  );

  return {
    notifications: result.items,
    totalCount: result.totalCount,
    totalPages: result.totalPages,
    page: result.page,
    perPage: result.perPage,
    loading,
    error,
    markingReadId,
    reload: load,
    markRead,
  };
}

export default useReportDecisionNotifications;