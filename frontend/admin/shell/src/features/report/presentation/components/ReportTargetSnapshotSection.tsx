// frontend/admin/shell/src/features/report/presentation/components/ReportTargetSnapshotSection.tsx

import type { ReviewReportCase } from "../../../../shared/type/reviewReport";
import {
  getSnapshotBodyLabel,
  getSnapshotTitleLabel,
} from "../model/reportLabels";
import ReportDetailField from "./ReportDetailField";

type ReportTargetSnapshotSectionProps = {
  reportCase: ReviewReportCase;
};

export default function ReportTargetSnapshotSection({
  reportCase,
}: ReportTargetSnapshotSectionProps) {
  return (
    <section className="report-detail-page__section">
      <dl className="report-detail-page__fields">
        {reportCase.snapshotRating !== null ? (
          <ReportDetailField
            label="評価"
            value={`${reportCase.snapshotRating} / 5`}
          />
        ) : null}

        {reportCase.snapshotTitle ? (
          <ReportDetailField
            label={getSnapshotTitleLabel(reportCase.targetType)}
            value={reportCase.snapshotTitle}
          />
        ) : null}

        <ReportDetailField
          label={getSnapshotBodyLabel(reportCase.targetType)}
          value={
            <div className="report-detail-page__body-text">
              {reportCase.snapshotBody || "-"}
            </div>
          }
        />
      </dl>
    </section>
  );
}