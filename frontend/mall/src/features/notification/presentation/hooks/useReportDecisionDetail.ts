// frontend/mall/src/features/notification/presentation/hooks/useReportDecisionDetail.ts

import { useEffect, useMemo, useRef } from "react";

import type { ReportDecisionNotification } from "../../infrastructure/reportDecisionNotificationApi";
import {
  useMarkReportDecisionNotificationReadMutation,
  useReportDecisionNotificationQuery,
} from "./useReportDecisionNotificationsQuery";

type UseReportDecisionDetailParams = {
  notificationId?: string;
  initialNotification?: ReportDecisionNotification | null;
  enabled?: boolean;
};

export type UseReportDecisionDetailResult = {
  notificationId: string;
  notification: ReportDecisionNotification | null;
  loading: boolean;
  error: string;
  notFound: boolean;
};

export function useReportDecisionDetail({
  notificationId = "",
  initialNotification = null,
  enabled = true,
}: UseReportDecisionDetailParams): UseReportDecisionDetailResult {
  const effectiveNotificationId = useMemo(
    () =>
      notificationId.trim() ||
      initialNotification?.id?.trim() ||
      "",
    [initialNotification?.id, notificationId],
  );

  const validInitialNotification = useMemo(() => {
    if (
      !initialNotification ||
      initialNotification.id !== effectiveNotificationId
    ) {
      return null;
    }

    return initialNotification;
  }, [effectiveNotificationId, initialNotification]);

  const notificationQuery = useReportDecisionNotificationQuery(
    effectiveNotificationId,
    {
      enabled:
        enabled &&
        Boolean(effectiveNotificationId),
    },
  );

  const markReadMutation =
    useMarkReportDecisionNotificationReadMutation();

  const notification =
    notificationQuery.data !== undefined
      ? notificationQuery.data
      : validInitialNotification;

  const markedReadRef = useRef<string>("");

  useEffect(() => {
    if (
      !enabled ||
      !effectiveNotificationId ||
      !notification
    ) {
      return;
    }

    if (notification.isRead === true) {
      markedReadRef.current =
        effectiveNotificationId;
      return;
    }

    if (
      markedReadRef.current ===
      effectiveNotificationId
    ) {
      return;
    }

    markedReadRef.current =
      effectiveNotificationId;

    markReadMutation.mutate(
      effectiveNotificationId,
    );
  }, [
    effectiveNotificationId,
    enabled,
    markReadMutation,
    notification,
  ]);

  const queryError =
    enabled &&
    notificationQuery.error instanceof Error
      ? notificationQuery.error.message
      : enabled &&
          notificationQuery.error
        ? "通報結果通知の取得に失敗しました"
        : "";

  const mutationError =
    enabled &&
    markReadMutation.error instanceof Error
      ? markReadMutation.error.message
      : enabled &&
          markReadMutation.error
        ? "通報結果通知の既読化に失敗しました"
        : "";

  const loading =
    enabled &&
    Boolean(effectiveNotificationId) &&
    notificationQuery.isPending &&
    !validInitialNotification;

  const notFound =
    enabled &&
    (
      !effectiveNotificationId ||
      (
        notificationQuery.isSuccess &&
        !notification
      )
    );

  const error =
    mutationError ||
    queryError ||
    (
      notFound
        ? "通報結果通知が見つかりません。"
        : ""
    );

  return {
    notificationId: effectiveNotificationId,
    notification,
    loading,
    error,
    notFound,
  };
}

export default useReportDecisionDetail;