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

type ReportCaseInfoSectionProps = {
  reportCase: ReportCase;
};

function buildTargetDetailPath(reportCase: ReportCase): string | null {
  const companyId = reportCase.targetCompanyId?.trim() || "";
  const targetParentId = reportCase.targetParentId?.trim() || "";

  if (!companyId || !targetParentId) return null;

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
    <section className="ui-detail-section">
      <dl className="ui-detail-definition-list ui-detail-definition-list--meta">
        <dt>{getTargetParentLabel(reportCase.targetType)}</dt>
        <dd>
          {targetDetailPath ? (
            <Link to={targetDetailPath} className="report-detail-page__target-link">
              {targetParentLabel}
            </Link>
          ) : (
            targetParentLabel
          )}
        </dd>

        <dt>{getTargetAuthorTypeLabel(reportCase.targetType)}</dt>
        <dd>{getActorTypeLabel(reportCase.targetAuthorType)}</dd>

        <dt>{getTargetAuthorLabel(reportCase.targetType)}</dt>
        <dd>
          {reportCase.targetAuthorName ||
            reportCase.targetAuthorId ||
            "-"}
        </dd>

        <dt>初回通報</dt>
        <dd className="ui-detail-definition-list__nowrap">
          {formatDateTime(reportCase.createdAt)}
        </dd>

        <dt>最終更新</dt>
        <dd className="ui-detail-definition-list__nowrap">
          {formatDateTime(reportCase.updatedAt)}
        </dd>
      </dl>
    </section>
  );
}