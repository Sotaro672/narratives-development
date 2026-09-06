// frontend/admin/shell/src/features/report/presentation/components/ReportItemsSection.tsx

import { useMemo } from "react";

import type { ReviewReportItem } from "../../../../shared/type/reviewReport";
import Table, {
  type TableColumn,
} from "../../../../shared/ui/Table/Table";
import { formatDateTime } from "../../../../shared/util/dateFormat";
import { getReasonLabel } from "../model/reportLabels";

type ReportItemsSectionProps = {
  reports: ReviewReportItem[];
  loading: boolean;
  page: number;
  totalPages: number;
  hasPreviousPage: boolean;
  hasNextPage: boolean;
  onPageChange: (page: number) => void;
};

export default function ReportItemsSection({
  reports,
  loading,
  page,
  totalPages,
  hasPreviousPage,
  hasNextPage,
  onPageChange,
}: ReportItemsSectionProps) {
  const columns = useMemo<TableColumn<ReviewReportItem>[]>(
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
        render: (report) =>
          report.reporterName ||
          report.reporterId ||
          "-",
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
        render: (report) =>
          report.companyName ||
          report.companyId ||
          "-",
        minWidth: "160px",
      },
    ],
    [],
  );

  return (
    <section className="report-detail-page__section">
      <div className="report-detail-page__reports-header">
        <h2 className="report-detail-page__section-title">
          通報内容
        </h2>

        {loading ? (
          <span
            className="report-detail-page__updating"
            aria-live="polite"
          >
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

      {totalPages > 1 ? (
        <nav
          className="report-detail-page__pagination"
          aria-label="通報内容のページ送り"
        >
          <button
            type="button"
            className="report-detail-page__pagination-button"
            disabled={!hasPreviousPage || loading}
            onClick={() => onPageChange(page - 1)}
          >
            前へ
          </button>

          <span className="report-detail-page__pagination-label">
            {page} / {totalPages}
          </span>

          <button
            type="button"
            className="report-detail-page__pagination-button"
            disabled={!hasNextPage || loading}
            onClick={() => onPageChange(page + 1)}
          >
            次へ
          </button>
        </nav>
      ) : null}
    </section>
  );
}