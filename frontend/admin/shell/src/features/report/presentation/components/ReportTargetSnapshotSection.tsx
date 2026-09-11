// frontend/admin/shell/src/features/report/presentation/components/ReportTargetSnapshotSection.tsx

import type { ReportCase } from "../../../../shared/type/report";
import {
  getSnapshotBodyLabel,
  getSnapshotTitleLabel,
} from "../model/reportLabels";

type ReportTargetSnapshotSectionProps = {
  reportCase: ReportCase;
};

export default function ReportTargetSnapshotSection({
  reportCase,
}: ReportTargetSnapshotSectionProps) {
  return (
    <section className="ui-detail-section">
      <dl className="ui-detail-definition-list ui-detail-definition-list--rows">
        {reportCase.snapshotRating !== null ? (
          <div>
            <dt>評価</dt>
            <dd>{reportCase.snapshotRating} / 5</dd>
          </div>
        ) : null}

        {reportCase.snapshotTitle && reportCase.targetType !== "RESALE" ? (
          <div>
            <dt>{getSnapshotTitleLabel(reportCase.targetType)}</dt>
            <dd>{reportCase.snapshotTitle}</dd>
          </div>
        ) : null}

        <div>
          <dt>{getSnapshotBodyLabel(reportCase.targetType)}</dt>
          <dd className="report-detail-page__body-text">
            {reportCase.snapshotBody || "-"}
          </dd>
        </div>
      </dl>
    </section>
  );
}