// frontend/console/shell/src/features/notification/presentation/notificationEvent.ts

export const REPORT_DECISION_NOTIFICATION_CHANGED =
  "report-decision-notification-changed";

export function emitReportDecisionNotificationChanged(): void {
  if (typeof window === "undefined") {
    return;
  }

  window.dispatchEvent(
    new Event(REPORT_DECISION_NOTIFICATION_CHANGED),
  );
}

export function subscribeReportDecisionNotificationChanged(
  listener: () => void,
): () => void {
  if (typeof window === "undefined") {
    return () => {};
  }

  const handleChanged = () => {
    listener();
  };

  window.addEventListener(
    REPORT_DECISION_NOTIFICATION_CHANGED,
    handleChanged,
  );

  return () => {
    window.removeEventListener(
      REPORT_DECISION_NOTIFICATION_CHANGED,
      handleChanged,
    );
  };
}