// frontend/admin/shell/src/features/report/presentation/components/ReportCaseInfoSection.tsx

import { useNavigate } from "react-router-dom";

import type { ReportCase } from "../../../../shared/type/report";
import TextLink from "../../../../shared/ui/TextLink/TextLink";
import { formatDateTime } from "../../../../shared/util/dateFormat";
import { getTargetAuthorLabel } from "../model/reportLabels";

type ReportCaseInfoSectionProps = {
  reportCase: ReportCase;
};

function buildTargetTokenDetailPath(reportCase: ReportCase): string | null {
  const companyId = reportCase.targetCompanyId?.trim() || "";
  const tokenBlueprintId = reportCase.targetTokenBlueprintId?.trim() || "";

  if (reportCase.targetType !== "RESALE" || !companyId || !tokenBlueprintId) {
    return null;
  }

  return `/contracts/${encodeURIComponent(companyId)}/token-blueprints/${encodeURIComponent(tokenBlueprintId)}`;
}

function buildResaleDetailPath(reportCase: ReportCase): string | null {
  const avatarId = reportCase.targetAuthorId?.trim() || "";
  const resaleId = reportCase.targetId?.trim() || "";

  if (reportCase.targetType !== "RESALE" || !avatarId || !resaleId) {
    return null;
  }

  return `/avatars/${encodeURIComponent(avatarId)}/resales/${encodeURIComponent(resaleId)}`;
}

function buildTargetAuthorDetailPath(reportCase: ReportCase): string | null {
  const companyId = reportCase.targetCompanyId?.trim() || "";
  const targetAuthorId = reportCase.targetAuthorId?.trim() || "";

  if (!targetAuthorId) {
    return null;
  }

  if (reportCase.targetAuthorType === "BRAND" && companyId) {
    return `/contracts/${encodeURIComponent(companyId)}/brands/${encodeURIComponent(targetAuthorId)}`;
  }

  if (
    reportCase.targetType === "RESALE" &&
    reportCase.targetAuthorType === "AVATAR"
  ) {
    return `/avatars/${encodeURIComponent(targetAuthorId)}`;
  }

  return null;
}

export default function ReportCaseInfoSection({
  reportCase,
}: ReportCaseInfoSectionProps) {
  const navigate = useNavigate();

  const targetTokenLabel =
    reportCase.targetTokenName ||
    reportCase.targetTokenBlueprintId ||
    "-";

  const targetAuthorLabel =
    reportCase.targetAuthorName ||
    reportCase.targetAuthorId ||
    "-";

  const targetTokenDetailPath = buildTargetTokenDetailPath(reportCase);
  const resaleDetailPath = buildResaleDetailPath(reportCase);
  const targetAuthorDetailPath = buildTargetAuthorDetailPath(reportCase);

  return (
    <section className="ui-detail-section">
      <dl className="ui-detail-definition-list ui-detail-definition-list--meta">
        {reportCase.targetType === "RESALE" ? (
          <>
            <dt>対象トークン</dt>
            <dd>
              {targetTokenDetailPath ? (
                <TextLink
                  tone="inherit"
                  onClick={() => navigate(targetTokenDetailPath)}
                >
                  {targetTokenLabel}
                </TextLink>
              ) : (
                targetTokenLabel
              )}
            </dd>

            <dt>再販ID</dt>
            <dd>
              {resaleDetailPath ? (
                <TextLink
                  tone="inherit"
                  onClick={() => navigate(resaleDetailPath)}
                >
                  {reportCase.targetId}
                </TextLink>
              ) : (
                reportCase.targetId || "-"
              )}
            </dd>
          </>
        ) : null}

        <dt>{getTargetAuthorLabel(reportCase.targetType)}</dt>
        <dd>
          {targetAuthorDetailPath ? (
            <TextLink
              tone="inherit"
              onClick={() => navigate(targetAuthorDetailPath)}
            >
              {targetAuthorLabel}
            </TextLink>
          ) : (
            targetAuthorLabel
          )}
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