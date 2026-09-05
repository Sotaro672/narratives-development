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

function getStatusLabel(status: ReviewReportCaseStatus): string {
  switch (status) {
    case "PENDING":
      return "未対応";
    case "KEPT":
      return "維持";
    case "REMOVED":
      return "削除";
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

  return value.length > 80
    ? `${value.slice(0, 80)}…`
    : value;
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
        render: (reportCase) =>
          formatDateTime(reportCase.updatedAt),
        sortValue: (reportCase) =>
          new Date(reportCase.updatedAt).getTime(),
        nowrap: true,
      },
      {
        key: "status",
        header: "対応状況",
        render: (reportCase) =>
          getStatusLabel(reportCase.status),
        sortValue: (reportCase) =>
          reportCase.status,
        nowrap: true,
      },
      {
        key: "targetType",
        header: "対象",
        render: (reportCase) =>
          getTargetTypeLabel(reportCase.targetType),
        sortValue: (reportCase) =>
          reportCase.targetType,
        nowrap: true,
      },
      {
        key: "snapshot",
        header: "内容",
        render: (reportCase) =>
          getSnapshotSummary(reportCase),
        minWidth: "280px",
      },
      {
        key: "targetAuthorType",
        header: "投稿者種別",
        render: (reportCase) =>
          getAuthorTypeLabel(
            reportCase.targetAuthorType,
          ),
        nowrap: true,
      },
      {
        key: "reportCount",
        header: "通報件数",
        render: (reportCase) =>
          `${reportCase.reportCount}件`,
        sortValue: (reportCase) =>
          reportCase.reportCount,
        align: "right",
        nowrap: true,
      },
      {
        key: "createdAt",
        header: "初回通報",
        render: (reportCase) =>
          formatDateTime(reportCase.createdAt),
        sortValue: (reportCase) =>
          new Date(reportCase.createdAt).getTime(),
        nowrap: true,
      },
    ],
    [],
  );

  const handleRowClick = (
    reportCase: ReviewReportCase,
  ) => {
    navigate(
      `/reports/${encodeURIComponent(
        reportCase.id,
      )}`,
      {
        state: { reportCase },
      },
    );
  };

  return (
    <Page>
      <PageHeader
        title="通報"
        actions={
          <div className="reports-page__header-actions">
            <select
              className="reports-page__select"
              value={status ?? ""}
              aria-label="対応状況で絞り込む"
              disabled={loading}
              onChange={(event) =>
                setStatus(
                  event.target.value
                    ? event.target.value as ReviewReportCaseStatus
                    : undefined,
                )
              }
            >
              <option value="">
                すべての対応状況
              </option>
              <option value="PENDING">
                未対応
              </option>
              <option value="KEPT">
                維持
              </option>
              <option value="REMOVED">
                削除
              </option>
            </select>

            <select
              className="reports-page__select"
              value={targetType ?? ""}
              aria-label="通報対象で絞り込む"
              disabled={loading}
              onChange={(event) =>
                setTargetType(
                  event.target.value
                    ? event.target.value as ReviewReportTargetType
                    : undefined,
                )
              }
            >
              <option value="">
                すべての対象
              </option>
              <option value="PRODUCT_BLUEPRINT_REVIEW">
                商品レビュー
              </option>
              <option value="TOKEN_BLUEPRINT_COMMENT">
                トークンコメント
              </option>
            </select>

            <RefreshButton
              onClick={reload}
              loading={loading}
              title="リフレッシュ"
              ariaLabel="通報一覧をリフレッシュ"
            />
          </div>
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
            getRowKey={(reportCase) =>
              reportCase.id
            }
            emptyMessage="通報はありません。"
            filteredEmptyMessage="条件に一致する通報はありません。"
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
                disabled={
                  !hasPreviousPage || loading
                }
                onClick={() =>
                  setPage(page - 1)
                }
              >
                前へ
              </button>

              <span className="reports-page__pagination-label">
                {page} / {totalPages}
              </span>

              <button
                type="button"
                className="reports-page__pagination-button"
                disabled={
                  !hasNextPage || loading
                }
                onClick={() =>
                  setPage(page + 1)
                }
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