// frontend/admin/shell/src/pages/ReportDetailPage.tsx

import { useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import ReportDecisionModal from "../features/report/components/ReportDecisionModal";
import { useReportDetail } from "../features/report/hooks/useReportDetail";
import type {
  ReviewReportActorType,
  ReviewReportCaseStatus,
  ReviewReportItem,
  ReviewReportReason,
  ReviewReportTargetType,
} from "../shared/type/reviewReport";
import Button from "../shared/ui/Button/Button";
import Page, { DetailPageBody, PageHeader } from "../shared/ui/Page/Page";
import RefreshButton from "../shared/ui/RefreshButton/RefreshButton";
import Tab, { type TabTone } from "../shared/ui/Tab/Tab";
import Table, { type TableColumn } from "../shared/ui/Table/Table";
import { formatDateTime } from "../shared/util/dateFormat";

import "./ReportDetailPage.css";

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

function getStatusTone(status: ReviewReportCaseStatus): TabTone {
  switch (status) {
    case "PENDING":
      return "warning";
    case "KEPT":
      return "success";
    case "REMOVED":
      return "danger";
    default:
      return "neutral";
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

function getActorTypeLabel(actorType: ReviewReportActorType): string {
  switch (actorType) {
    case "AVATAR":
      return "ユーザー";
    case "BRAND":
      return "ブランド";
    default:
      return actorType;
  }
}

function getReasonLabel(reason: ReviewReportReason): string {
  switch (reason) {
    case "SPAM":
      return "スパム";
    case "HARASSMENT":
      return "嫌がらせ";
    case "INAPPROPRIATE":
      return "不適切な内容";
    case "FALSE_INFORMATION":
      return "虚偽情報";
    case "OTHER":
      return "その他";
    default:
      return reason;
  }
}

function DetailField({
  label,
  value,
}: {
  label: string;
  value: React.ReactNode;
}) {
  return (
    <div className="report-detail-page__field">
      <dt className="report-detail-page__field-label">{label}</dt>
      <dd className="report-detail-page__field-value">{value}</dd>
    </div>
  );
}

export default function ReportDetailPage() {
  const navigate = useNavigate();
  const { reportId } = useParams();
  const [decisionReason, setDecisionReason] = useState("");
  const [decisionModalOpen, setDecisionModalOpen] = useState(false);
  const [decisionAttempted, setDecisionAttempted] = useState(false);

  const {
    reportCase,
    reports,
    loading,
    error,
    page,
    totalPages,
    hasPreviousPage,
    hasNextPage,
    deciding,
    decisionError,
    canDecide,
    canKeep,
    canRemove,
    setPage,
    reload,
    keep,
    remove,
  } = useReportDetail(reportId);

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

  const openDecisionModal = () => {
    if (!canDecide) return;
    setDecisionReason("");
    setDecisionAttempted(false);
    setDecisionModalOpen(true);
  };

  const closeDecisionModal = () => {
    if (deciding) return;
    setDecisionModalOpen(false);
    setDecisionReason("");
    setDecisionAttempted(false);
  };

  const handleKeep = async () => {
    setDecisionAttempted(true);
    const result = await keep(decisionReason);

    if (result) {
      setDecisionModalOpen(false);
      setDecisionReason("");
      setDecisionAttempted(false);
    }
  };

  const handleRemove = async () => {
    setDecisionAttempted(true);
    const result = await remove(decisionReason);

    if (result) {
      setDecisionModalOpen(false);
      setDecisionReason("");
      setDecisionAttempted(false);
    }
  };

  const pageTitle = reportCase
    ? getTargetTypeLabel(reportCase.targetType)
    : "通報詳細";

  return (
    <>
      <Page>
        <PageHeader
          title={pageTitle}
          meta={
            reportCase ? (
              <>
                <span>通報 {reportCase.reportCount}件</span>
                <Tab
                  tone={getStatusTone(reportCase.status)}
                  aria-label={`対応状況 ${getStatusLabel(reportCase.status)}`}
                >
                  {getStatusLabel(reportCase.status)}
                </Tab>
              </>
            ) : undefined
          }
          leading={
            <button
              type="button"
              className="ui-page-header__back"
              aria-label="戻る"
              onClick={() => navigate("/reports")}
            >
              <svg
                width="18"
                height="18"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                aria-hidden="true"
              >
                <path d="M19 12H5" />
                <path d="M12 19l-7-7 7-7" />
              </svg>
            </button>
          }
          actions={
            <>
              <Button
                variant="secondary"
                size="sm"
                disabled={!canDecide || loading}
                onClick={openDecisionModal}
              >
                裁定
              </Button>

              <RefreshButton
                onClick={reload}
                loading={loading}
                title="リフレッシュ"
                ariaLabel="通報詳細をリフレッシュ"
              />
            </>
          }
        />

        {loading && !reportCase ? (
          <p>通報情報を読み込んでいます。</p>
        ) : null}

        {!loading && error ? (
          <p role="alert">
            通報情報を取得できませんでした。{error}
          </p>
        ) : null}

        {!loading && !error && !reportCase ? (
          <p role="alert">通報情報を取得できませんでした。</p>
        ) : null}

        {reportCase ? (
          <DetailPageBody
            main={
              <div className="report-detail-page__main">
                <section className="report-detail-page__section">
                  <dl className="report-detail-page__fields">
                    {reportCase.snapshotRating !== null ? (
                      <DetailField
                        label="評価"
                        value={`${reportCase.snapshotRating} / 5`}
                      />
                    ) : null}

                    {reportCase.snapshotTitle ? (
                      <DetailField
                        label="タイトル"
                        value={reportCase.snapshotTitle}
                      />
                    ) : null}

                    <DetailField
                      label="本文"
                      value={
                        <div className="report-detail-page__body-text">
                          {reportCase.snapshotBody || "-"}
                        </div>
                      }
                    />
                  </dl>
                </section>

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
                        onClick={() => setPage(page - 1)}
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
                        onClick={() => setPage(page + 1)}
                      >
                        次へ
                      </button>
                    </nav>
                  ) : null}
                </section>
              </div>
            }
            aside={
              <div className="report-detail-page__aside">
                <section className="report-detail-page__section">
                  <h2 className="report-detail-page__section-title">
                    ケース情報
                  </h2>

                  <dl className="report-detail-page__fields report-detail-page__fields--compact">
                    <DetailField
                      label="親"
                      value={
                        reportCase.targetParentName ||
                        reportCase.targetParentId ||
                        "-"
                      }
                    />
                    <DetailField
                      label="投稿者種別"
                      value={getActorTypeLabel(reportCase.targetAuthorType)}
                    />
                    <DetailField
                      label="投稿者"
                      value={
                        reportCase.targetAuthorName ||
                        reportCase.targetAuthorId ||
                        "-"
                      }
                    />
                    <DetailField
                      label="初回通報"
                      value={formatDateTime(reportCase.createdAt)}
                    />
                    <DetailField
                      label="最終更新"
                      value={formatDateTime(reportCase.updatedAt)}
                    />
                  </dl>
                </section>
              </div>
            }
          />
        ) : null}
      </Page>

      {reportCase ? (
        <ReportDecisionModal
          open={decisionModalOpen}
          status={reportCase.status}
          decisionReason={decisionReason}
          deciding={deciding}
          decisionError={decisionAttempted ? decisionError : null}
          canKeep={canKeep}
          canRemove={canRemove}
          onChangeDecisionReason={setDecisionReason}
          onClose={closeDecisionModal}
          onKeep={handleKeep}
          onRemove={handleRemove}
        />
      ) : null}
    </>
  );
}