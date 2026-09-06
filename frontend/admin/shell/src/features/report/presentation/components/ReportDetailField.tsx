// frontend/admin/shell/src/features/report/presentation/components/ReportDetailField.tsx

import type { ReactNode } from "react";

type ReportDetailFieldProps = {
  label: string;
  value: ReactNode;
};

export default function ReportDetailField({
  label,
  value,
}: ReportDetailFieldProps) {
  return (
    <div className="report-detail-page__field">
      <dt className="report-detail-page__field-label">{label}</dt>
      <dd className="report-detail-page__field-value">{value}</dd>
    </div>
  );
}