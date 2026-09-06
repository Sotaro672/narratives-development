// frontend/console/shell/src/features/notification/presentation/notificationEvent.ts

export const REVIEW_REPORT_DECISION_NOTIFICATION_CHANGED =
  "review-report-decision-notification-changed";

export function emitReviewReportDecisionNotificationChanged(): void {
  if (typeof window === "undefined") {
    return;
  }

  window.dispatchEvent(
    new Event(
      REVIEW_REPORT_DECISION_NOTIFICATION_CHANGED,
    ),
  );
}

export function subscribeReviewReportDecisionNotificationChanged(
  listener: () => void,
): () => void {
  if (typeof window === "undefined") {
    return () => {};
  }

  const handleChanged = () => {
    listener();
  };

  window.addEventListener(
    REVIEW_REPORT_DECISION_NOTIFICATION_CHANGED,
    handleChanged,
  );

  return () => {
    window.removeEventListener(
      REVIEW_REPORT_DECISION_NOTIFICATION_CHANGED,
      handleChanged,
    );
  };
}