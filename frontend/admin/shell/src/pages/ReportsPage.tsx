// frontend/admin/shell/src/pages/ReportsPage.tsx

import { useMemo } from "react";
import { useNavigate } from "react-router-dom";

import { useReports } from "../features/report/hooks/useReports";
import type {
  ReviewReportCase,
  ReviewReportCaseStatus,
  ReviewReportTargetType,
} from "../shared/type/reviewReport";
import Page, { PageHeader } from "../shared/ui/Page/Page";
import RefreshButton from "../shared/ui/RefreshButton/RefreshButton";
import Table, { type TableColumn } from "../shared/ui/Table/Table";
import { formatDateTime } from "../shared/util/dateFormat";

import "./ReportsPage.css";

function getStatusLabel(
  status: ReviewReportCaseStatus,
  targetType: ReviewReportTargetType,
): string {
  switch (status) {
    case "PENDING":
      return "未対応";
    case "KEPT":
      return targetType === "AVATAR" ? "変化なし" : "維持";
    case "REMOVED":
      return targetType === "AVATAR" ? "再販利用停止" : "削除";
    default:
      return status;
  }
}

function getTargetTypeLabel(targetType: ReviewReportTargetType): string {
  switch (targetType) {
    case "PRODUCT_BLUEPRINT_REVIEW":
      return "商品レビュー";
    case "TOKEN_BLUEPRINT_COMMENT":
      return "トークンコメント";
    case "AVATAR":
      return "アバター";
    default:
      return targetType;
  }
}

function getAuthorTypeLabel(
  authorType: ReviewReportCase["targetAuthorType"],
): string {
  switch (authorType) {
    case "AVATAR":
      return "ユーザー";
    case "BRAND":
      return "ブランド";
    default:
      return authorType;
  }
}

function getSnapshotSummary(reportCase: ReviewReportCase): string {
  const title = reportCase.snapshotTitle.trim();
  const body = reportCase.snapshotBody.trim();
  const value = title || body;

  if (!value) {
    return "-";
  }

  return value.length > 80 ? `${value.slice(0, 80)}…` : value;
}

export default function ReportsPage() {
  const navigate = useNavigate();

  const {
    items,
    loading,
    error,
    status,
    targetType,
    page,
    totalCount,
    totalPages,
    hasPreviousPage,
    hasNextPage,
    setStatus,
    setTargetType,
    setPage,
    reload,
  } = useReports();

  const columns = useMemo<TableColumn<ReviewReportCase>[]>(
    () => [
      {
        key: "updatedAt",
        header: "更新日時",
        render: (reportCase) => formatDateTime(reportCase.updatedAt),
        sortValue: (reportCase) => new Date(reportCase.updatedAt).getTime(),
        nowrap: true,
      },
      {
        key: "status",
        header: "対応状況",
        render: (reportCase) =>
          getStatusLabel(reportCase.status, reportCase.targetType),
        filter: {
          getValue: (reportCase) => reportCase.status,
          options: [
            { value: "PENDING", label: "未対応" },
            { value: "KEPT", label: "維持・変化なし" },
            { value: "REMOVED", label: "削除・再販利用停止" },
          ],
        },
        nowrap: true,
      },
      {
        key: "targetType",
        header: "対象",
        render: (reportCase) => getTargetTypeLabel(reportCase.targetType),
        filter: {
          getValue: (reportCase) => reportCase.targetType,
          options: [
            {
              value: "PRODUCT_BLUEPRINT_REVIEW",
              label: "商品レビュー",
            },
            {
              value: "TOKEN_BLUEPRINT_COMMENT",
              label: "トークンコメント",
            },
            {
              value: "AVATAR",
              label: "アバター",
            },
          ],
        },
        nowrap: true,
      },
      {
        key: "snapshot",
        header: "内容",
        render: (reportCase) => getSnapshotSummary(reportCase),
        minWidth: "280px",
      },
      {
        key: "targetAuthorType",
        header: "対象者種別",
        render: (reportCase) =>
          getAuthorTypeLabel(reportCase.targetAuthorType),
        nowrap: true,
      },
      {
        key: "reportCount",
        header: "通報件数",
        render: (reportCase) => `${reportCase.reportCount}件`,
        sortValue: (reportCase) => reportCase.reportCount,
        align: "right",
        nowrap: true,
      },
      {
        key: "createdAt",
        header: "初回通報",
        render: (reportCase) => formatDateTime(reportCase.createdAt),
        sortValue: (reportCase) => new Date(reportCase.createdAt).getTime(),
        nowrap: true,
      },
    ],
    [],
  );

  const handleRowClick = (reportCase: ReviewReportCase) => {
    navigate(`/reports/${encodeURIComponent(reportCase.id)}`, {
      state: { reportCase },
    });
  };

  const handleFilterChange = (key: string, value: string) => {
    switch (key) {
      case "status":
        setStatus(
          value
            ? (value as ReviewReportCaseStatus)
            : undefined,
        );
        break;
      case "targetType":
        setTargetType(
          value
            ? (value as ReviewReportTargetType)
            : undefined,
        );
        break;
    }
  };

  const hasActiveFilter =
    status !== undefined ||
    targetType !== undefined;

  return (
    <Page>
      <PageHeader
        title="通報"
        actions={
          <RefreshButton
            onClick={reload}
            loading={loading}
            title="リフレッシュ"
            ariaLabel="通報一覧をリフレッシュ"
          />
        }
      />

      {loading && items.length === 0 ? (
        <p>通報を読み込んでいます。</p>
      ) : null}

      {!loading && error ? (
        <p role="alert">
          通報の取得に失敗しました。{error}
        </p>
      ) : null}

      {!error && (items.length > 0 || !loading) ? (
        <>
          <div className="reports-page__summary">
            <p className="reports-page__count">
              {totalCount}件
            </p>

            {loading ? (
              <span
                className="reports-page__updating"
                aria-live="polite"
              >
                更新中...
              </span>
            ) : null}
          </div>

          <Table
            columns={columns}
            rows={items}
            getRowKey={(reportCase) => reportCase.id}
            emptyMessage={
              hasActiveFilter
                ? "条件に一致する通報はありません。"
                : "通報はありません。"
            }
            filteredEmptyMessage="条件に一致する通報はありません。"
            filterValues={{
              status: status ?? "",
              targetType: targetType ?? "",
            }}
            onFilterChange={handleFilterChange}
            onRowClick={handleRowClick}
          />

          {totalPages > 1 ? (
            <nav
              className="reports-page__pagination"
              aria-label="通報一覧のページ送り"
            >
              <button
                type="button"
                className="reports-page__pagination-button"
                disabled={!hasPreviousPage || loading}
                onClick={() => setPage(page - 1)}
              >
                前へ
              </button>

              <span className="reports-page__pagination-label">
                {page} / {totalPages}
              </span>

              <button
                type="button"
                className="reports-page__pagination-button"
                disabled={!hasNextPage || loading}
                onClick={() => setPage(page + 1)}
              >
                次へ
              </button>
            </nav>
          ) : null}
        </>
      ) : null}
    </Page>
  );
}