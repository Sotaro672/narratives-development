// frontend/admin/shell/src/features/report/presentation/components/ReportCaseInfoSection.tsx

import type { ReportCase } from "../../../../shared/type/report";
import { formatDateTime } from "../../../../shared/util/dateFormat";
import {
  getActorTypeLabel,
  getTargetAuthorLabel,
  getTargetAuthorTypeLabel,
  getTargetParentLabel,
} from "../model/reportLabels";
import ReportDetailField from "./ReportDetailField";

type ReportCaseInfoSectionProps = {
  reportCase: ReportCase;
};

export default function ReportCaseInfoSection({
  reportCase,
}: ReportCaseInfoSectionProps) {
  return (
    <section className="report-detail-page__section">
      <h2 className="report-detail-page__section-title">
        ケース情報
      </h2>

      <dl className="report-detail-page__fields report-detail-page__fields--compact">
        <ReportDetailField
          label={getTargetParentLabel(reportCase.targetType)}
          value={
            reportCase.targetParentName ||
            reportCase.targetParentId ||
            "-"
          }
        />

        <ReportDetailField
          label={getTargetAuthorTypeLabel(reportCase.targetType)}
          value={getActorTypeLabel(reportCase.targetAuthorType)}
        />

        <ReportDetailField
          label={getTargetAuthorLabel(reportCase.targetType)}
          value={
            reportCase.targetAuthorName ||
            reportCase.targetAuthorId ||
            "-"
          }
        />

        <ReportDetailField
          label="初回通報"
          value={formatDateTime(reportCase.createdAt)}
        />

        <ReportDetailField
          label="最終更新"
          value={formatDateTime(reportCase.updatedAt)}
        />
      </dl>
    </section>
  );
}