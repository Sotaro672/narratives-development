// frontend/admin/shell/src/features/report/presentation/components/ReportItemsSection.tsx

import { useMemo } from "react";

import type { ReportItem } from "../../../../shared/type/report";
import Pagination from "../../../../shared/ui/Pagination/Pagination";
import Table, { type TableColumn } from "../../../../shared/ui/Table/Table";
import { formatDateTime } from "../../../../shared/util/dateFormat";
import { getReasonLabel } from "../model/reportLabels";

type ReportItemsSectionProps = {
  reports: ReportItem[];
  loading: boolean;
  page: number;
  totalPages: number;
  onPageChange: (page: number) => void;
};

export default function ReportItemsSection({
  reports,
  loading,
  page,
  totalPages,
  onPageChange,
}: ReportItemsSectionProps) {
  const columns = useMemo<TableColumn<ReportItem>[]>(
    () => [
      {
        key: "createdAt",
        header: "通報日時",
        render: (report) => formatDateTime(report.createdAt),
        sortValue: (report) => new Date(report.createdAt).getTime(),
        nowrap: true,
      },
      {
        key: "reporterId",
        header: "通報者",
        render: (report) => report.reporterName || report.reporterId || "-",
        minWidth: "180px",
      },
      {
        key: "reason",
        header: "理由",
        render: (report) => getReasonLabel(report.reason),
        filter: {
          getValue: (report) => getReasonLabel(report.reason),
          options: [
            { value: "スパム", label: "スパム" },
            { value: "嫌がらせ", label: "嫌がらせ" },
            { value: "不適切な内容", label: "不適切な内容" },
            { value: "虚偽情報", label: "虚偽情報" },
            { value: "その他", label: "その他" },
          ],
        },
        nowrap: true,
      },
      {
        key: "detail",
        header: "詳細",
        render: (report) => report.detail || "-",
        minWidth: "260px",
      },
      {
        key: "companyId",
        header: "会社",
        render: (report) => report.companyName || report.companyId || "-",
        minWidth: "160px",
      },
    ],
    [],
  );

  return (
    <section className="ui-detail-section">
      <div className="report-detail-page__reports-header">
        <h2 className="ui-detail-section__title">通報内容</h2>

        {loading ? (
          <span className="report-detail-page__updating" aria-live="polite">
            更新中...
          </span>
        ) : null}
      </div>

      <Table
        columns={columns}
        rows={reports}
        getRowKey={(report) => report.id}
        emptyMessage="通報内容はありません。"
        filteredEmptyMessage="条件に一致する通報はありません。"
      />

      <Pagination
        page={page}
        totalPages={totalPages}
        onPageChange={onPageChange}
        disabled={loading}
        ariaLabel="通報内容のページ送り"
      />
    </section>
  );
}