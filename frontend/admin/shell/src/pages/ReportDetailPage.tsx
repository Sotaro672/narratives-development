// frontend/admin/shell/src/pages/ReportDetailPage.tsx

import { useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { useReportDetail } from "../features/report/hooks/useReportDetail";
import type {
  ReviewReportActorType,
  ReviewReportCaseStatus,
  ReviewReportItem,
  ReviewReportReason,
  ReviewReportTargetType,
} from "../shared/type/reviewReport";
import Page, { DetailPageBody, PageHeader } from "../shared/ui/Page/Page";
import RefreshButton from "../shared/ui/RefreshButton/RefreshButton";
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

  const {
    reportCase,
    reports,
    loading,
    error,
    page,
    totalCount,
    totalPages,
    hasPreviousPage,
    hasNextPage,
    deciding,
    decisionError,
    canDecide,
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
        key: "reporterType",
        header: "通報者種別",
        render: (report) => getActorTypeLabel(report.reporterType),
        filter: {
          getValue: (report) => getActorTypeLabel(report.reporterType),
          options: [
            { value: "ユーザー", label: "ユーザー" },
            { value: "ブランド", label: "ブランド" },
          ],
        },
        nowrap: true,
      },
      {
        key: "reporterId",
        header: "通報者ID",
        render: (report) => report.reporterId || "-",
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
        header: "会社ID",
        render: (report) => report.companyId || "-",
        minWidth: "160px",
      },
    ],
    [],
  );

  const handleKeep = async () => {
    const result = await keep(decisionReason);

    if (result) {
      setDecisionReason("");
    }
  };

  const handleRemove = async () => {
    const result = await remove(decisionReason);

    if (result) {
      setDecisionReason("");
    }
  };

  const pageTitle = reportCase
    ? getTargetTypeLabel(reportCase.targetType)
    : "通報詳細";

  return (
    <Page>
      <PageHeader
        title={pageTitle}
        meta={reportCase ? `通報 ${reportCase.reportCount}件` : undefined}
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
          <RefreshButton
            onClick={reload}
            loading={loading}
            title="リフレッシュ"
            ariaLabel="通報詳細をリフレッシュ"
          />
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
        <p role="alert">
          通報情報を取得できませんでした。
        </p>
      ) : null}

      {reportCase ? (
        <DetailPageBody
          main={
            <div className="report-detail-page__main">
              <section className="report-detail-page__section">
                <div>
                  <h2 className="report-detail-page__section-title">
                    通報対象
                  </h2>
                  <p className="report-detail-page__description">
                    通報が最初に作成された時点の投稿内容です。
                  </p>
                </div>

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
                  <div>
                    <h2 className="report-detail-page__section-title">
                      通報内容
                    </h2>
                    <p className="report-detail-page__description">
                      {totalCount}件の通報があります。
                    </p>
                  </div>

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
                    label="対応状況"
                    value={getStatusLabel(reportCase.status)}
                  />
                  <DetailField
                    label="対象種別"
                    value={getTargetTypeLabel(reportCase.targetType)}
                  />
                  <DetailField
                    label="ケースID"
                    value={reportCase.id}
                  />
                  <DetailField
                    label="対象ID"
                    value={reportCase.targetId}
                  />
                  <DetailField
                    label="親ID"
                    value={reportCase.targetParentId || "-"}
                  />
                  <DetailField
                    label="投稿者種別"
                    value={getActorTypeLabel(reportCase.targetAuthorType)}
                  />
                  <DetailField
                    label="投稿者ID"
                    value={reportCase.targetAuthorId}
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

              {reportCase.status !== "PENDING" ? (
                <section className="report-detail-page__section">
                  <h2 className="report-detail-page__section-title">
                    裁定結果
                  </h2>

                  <dl className="report-detail-page__fields report-detail-page__fields--compact">
                    <DetailField
                      label="結果"
                      value={getStatusLabel(reportCase.status)}
                    />
                    <DetailField
                      label="裁定日時"
                      value={
                        reportCase.decidedAt
                          ? formatDateTime(reportCase.decidedAt)
                          : "-"
                      }
                    />
                    <DetailField
                      label="管理者"
                      value={reportCase.decidedBy || "-"}
                    />
                    <DetailField
                      label="裁定理由"
                      value={
                        <div className="report-detail-page__pre-wrap">
                          {reportCase.decisionReason || "-"}
                        </div>
                      }
                    />
                  </dl>
                </section>
              ) : (
                <section className="report-detail-page__decision">
                  <div>
                    <h2 className="report-detail-page__section-title">
                      裁定
                    </h2>
                    <p className="report-detail-page__description">
                      投稿を維持するか、削除するかを決定します。
                    </p>
                  </div>

                  <label className="report-detail-page__decision-field">
                    <span className="report-detail-page__decision-label">
                      裁定理由
                    </span>
                    <textarea
                      className="report-detail-page__decision-textarea"
                      value={decisionReason}
                      rows={5}
                      maxLength={2000}
                      disabled={deciding}
                      placeholder="裁定の根拠を入力してください。"
                      onChange={(event) =>
                        setDecisionReason(event.target.value)
                      }
                    />
                  </label>

                  {decisionError ? (
                    <p
                      className="report-detail-page__decision-error"
                      role="alert"
                    >
                      {decisionError}
                    </p>
                  ) : null}

                  <div className="report-detail-page__decision-actions">
                    <button
                      type="button"
                      className="report-detail-page__decision-button"
                      disabled={!canDecide || !decisionReason.trim()}
                      onClick={() => void handleKeep()}
                    >
                      {deciding ? "処理中..." : "維持する"}
                    </button>

                    <button
                      type="button"
                      className="report-detail-page__decision-button report-detail-page__decision-button--danger"
                      disabled={!canDecide || !decisionReason.trim()}
                      onClick={() => void handleRemove()}
                    >
                      {deciding ? "処理中..." : "削除する"}
                    </button>
                  </div>

                  <p className="report-detail-page__decision-note">
                    「削除する」を選択すると、対象コンテンツの削除に成功した後でケースが削除済みとして確定します。
                  </p>
                </section>
              )}
            </div>
          }
        />
      ) : null}
    </Page>
  );
}