// frontend/admin/shell/src/pages/ReportDetailPage.tsx

import { useNavigate, useParams } from "react-router-dom";

import ReportCaseInfoSection from "../features/report/presentation/components/ReportCaseInfoSection";
import ReportDecisionModal from "../features/report/presentation/components/ReportDecisionModal";
import ReportItemsSection from "../features/report/presentation/components/ReportItemsSection";
import ReportTargetSnapshotSection from "../features/report/presentation/components/ReportTargetSnapshotSection";
import { useReportDecisionModal } from "../features/report/presentation/hooks/useReportDecisionModal";
import { useReportDetail } from "../features/report/presentation/hooks/useReportDetail";
import {
  getStatusLabel,
  getStatusTone,
  getTargetTypeLabel,
} from "../features/report/presentation/model/reportLabels";
import Button from "../shared/ui/Button/Button";
import Page, { DetailPageBody, PageHeader } from "../shared/ui/Page/Page";
import RefreshButton from "../shared/ui/RefreshButton/RefreshButton";
import Tab from "../shared/ui/Tab/Tab";

import "./ReportDetailPage.css";

export default function ReportDetailPage() {
  const navigate = useNavigate();
  const { reportId } = useParams();

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

  const {
    decisionReason,
    decisionModalOpen,
    decisionAttempted,
    setDecisionReason,
    openDecisionModal,
    closeDecisionModal,
    handleKeep,
    handleRemove,
  } = useReportDecisionModal({
    canDecide,
    deciding,
    keep,
    remove,
  });

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
                  aria-label={`対応状況 ${getStatusLabel(reportCase.status, reportCase.targetType)}`}
                >
                  {getStatusLabel(reportCase.status, reportCase.targetType)}
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
                <ReportTargetSnapshotSection reportCase={reportCase} />

                <ReportItemsSection
                  reports={reports}
                  loading={loading}
                  page={page}
                  totalPages={totalPages}
                  hasPreviousPage={hasPreviousPage}
                  hasNextPage={hasNextPage}
                  onPageChange={setPage}
                />
              </div>
            }
            aside={
              <div className="report-detail-page__aside">
                <ReportCaseInfoSection reportCase={reportCase} />
              </div>
            }
          />
        ) : null}
      </Page>

      {reportCase ? (
        <ReportDecisionModal
          open={decisionModalOpen}
          status={reportCase.status}
          targetType={reportCase.targetType}
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