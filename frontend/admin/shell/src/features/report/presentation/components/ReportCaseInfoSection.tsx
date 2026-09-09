// frontend/admin/shell/src/features/report/presentation/components/ReportCaseInfoSection.tsx

import { Link } from "react-router-dom";

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

function buildTargetDetailPath(reportCase: ReportCase): string | null {
  const companyId = reportCase.targetCompanyId?.trim() || "";
  const targetParentId = reportCase.targetParentId?.trim() || "";

  if (!companyId || !targetParentId) {
    return null;
  }

  const encodedCompanyId = encodeURIComponent(companyId);
  const encodedTargetParentId = encodeURIComponent(targetParentId);

  switch (reportCase.targetType) {
    case "LIST":
      return `/contracts/${encodedCompanyId}/lists/${encodedTargetParentId}`;

    case "TOKEN_BLUEPRINT":
    case "TOKEN_BLUEPRINT_COMMENT":
      return `/contracts/${encodedCompanyId}/token-blueprints/${encodedTargetParentId}`;

    case "PRODUCT_BLUEPRINT_REVIEW":
      return `/contracts/${encodedCompanyId}/product-blueprints/${encodedTargetParentId}`;

    default:
      return null;
  }
}

export default function ReportCaseInfoSection({
  reportCase,
}: ReportCaseInfoSectionProps) {
  const targetParentLabel =
    reportCase.targetParentName ||
    reportCase.targetParentId ||
    "-";

  const targetDetailPath = buildTargetDetailPath(reportCase);

  return (
    <section className="report-detail-page__section">
      <h2 className="report-detail-page__section-title">
        ケース情報
      </h2>

      <dl className="report-detail-page__fields report-detail-page__fields--compact">
        <ReportDetailField
          label={getTargetParentLabel(reportCase.targetType)}
          value={
            targetDetailPath ? (
              <Link
                to={targetDetailPath}
                className="report-detail-page__target-link"
              >
                {targetParentLabel}
              </Link>
            ) : (
              targetParentLabel
            )
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