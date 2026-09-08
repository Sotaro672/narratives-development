// frontend/mall/src/features/notification/presentation/components/ReportDecisionDetail.tsx

import { formatDateTime } from "../../../../components/utils/date";

import type { ReportDecisionNotification } from "../../infrastructure/reportDecisionNotificationApi";
import { createReportDecisionPresentation } from "../model/reportDecisionPresentation";

type ReportDecisionDetailProps = {
  notification: ReportDecisionNotification;
};

export default function ReportDecisionDetail({
  notification,
}: ReportDecisionDetailProps) {
  const presentation =
    createReportDecisionPresentation(notification);

  const occurredAtLabel =
    formatDateTime(presentation.occurredAt);

  return (
    <article className="announcement-page__detail">
      <h1 className="announcement-page__detail-title">
        {presentation.title}
      </h1>

      <div className="announcement-page__card-head">
        <div className="announcement-page__card-meta">
          <span className="announcement-page__token">
            {presentation.cardLabel}
          </span>

          <time
            className="announcement-page__date"
            dateTime={presentation.occurredAt || undefined}
          >
            {occurredAtLabel}
          </time>
        </div>
      </div>

      <div className="announcement-page__detail-content">
        {presentation.body}
      </div>

      {presentation.isReporterDecision ? (
        <>
          <div className="announcement-page__attachments">
            通報理由: {presentation.reportReasonLabel}
          </div>

          {notification.reportDetail ? (
            <div className="announcement-page__attachments">
              通報詳細: {notification.reportDetail}
            </div>
          ) : null}
        </>
      ) : null}

      <div className="announcement-page__attachments">
        審査結果: {presentation.statusLabel}
      </div>

      {notification.decisionReason ? (
        <div className="announcement-page__attachments">
          審査理由: {notification.decisionReason}
        </div>
      ) : null}
    </article>
  );
}