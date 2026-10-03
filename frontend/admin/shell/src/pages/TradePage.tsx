// frontend/admin/shell/src/pages/TradePage.tsx

import { useCallback } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { useAvatars } from "../features/avatar/presentation/hooks/useAvatars";
import ReportDecisionModal from "../features/report/presentation/components/ReportDecisionModal";
import { useReportDecisionModal } from "../features/report/presentation/hooks/useReportDecisionModal";
import { useReportDetail } from "../features/report/presentation/hooks/useReportDetail";
import ResaleDetailAside from "../features/resale/presentation/components/ResaleDetailAside";
import { useResaleDetail } from "../features/resale/presentation/hooks/useResaleDetail";
import TradeMessageTable from "../features/trade/presentation/components/TradeMessageTable";
import { useResaleTrades } from "../features/trade/presentation/hooks/useResaleTrades";
import { useTradeReturnReport } from "../features/trade/presentation/hooks/useTradeReturnReport";
import Button from "../shared/ui/Button/Button";
import Page, { DetailPageBody, PageHeader } from "../shared/ui/Page/Page";

export default function TradePage() {
  const navigate = useNavigate();
  const { avatarId = "", resaleId = "", tradeId = "" } = useParams<{
    avatarId?: string;
    resaleId?: string;
    tradeId?: string;
  }>();

  const { avatars } = useAvatars();
  const { trades } = useResaleTrades(resaleId);
  const { resale, loading, error, reload } = useResaleDetail(
    avatarId,
    resaleId,
  );

  const {
    reportCase: returnReportCase,
    report: returnReport,
    loading: returnReportLoading,
    error: returnReportError,
    reload: reloadReturnReport,
  } = useTradeReturnReport(tradeId);

  const {
    deciding,
    decisionError,
    canDecide,
    canKeep,
    canRemove,
    keep,
    remove,
  } = useReportDetail(returnReportCase?.id);

  const keepReturnTrade = useCallback(
    async (
      decisionReason: string,
      continueTrade?: boolean,
    ) => {
      const result = await keep(
        decisionReason,
        continueTrade,
      );

      if (result) {
        await reloadReturnReport();
      }

      return result;
    },
    [keep, reloadReturnReport],
  );

  const {
    decisionReason,
    continueTrade,
    decisionModalOpen,
    decisionAttempted,
    setDecisionReason,
    setContinueTrade,
    openDecisionModal,
    closeDecisionModal,
    handleKeep,
    handleRemove,
  } = useReportDecisionModal({
    canDecide,
    deciding,
    keep: keepReturnTrade,
    remove,
  });

  const sellerAvatarName =
    avatars.find((avatar) => avatar.id === avatarId)?.avatarName ?? "";

  const buyerAvatarName =
    trades?.items.find((trade) => trade.id === tradeId)?.buyerAvatarName ?? "";

  const canOpenReturnDecision =
    returnReportCase?.targetType === "TRADE" &&
    returnReportCase.status === "PENDING";

  const tradeMessageTableKey = [
    tradeId,
    returnReportCase?.updatedAt ?? "",
  ].join(":");

  return (
    <>
      <Page>
        <PageHeader
          title="取引詳細"
          leading={
            <button
              type="button"
              className="ui-page-header__back"
              aria-label="戻る"
              onClick={() =>
                navigate(
                  `/avatars/${encodeURIComponent(avatarId)}/resales/${encodeURIComponent(resaleId)}`,
                )
              }
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
            canOpenReturnDecision ? (
              <Button
                variant="secondary"
                size="sm"
                disabled={
                  returnReportLoading ||
                  deciding ||
                  !canDecide
                }
                onClick={openDecisionModal}
              >
                裁定
              </Button>
            ) : undefined
          }
        />

        <DetailPageBody
          main={
            <TradeMessageTable
              key={tradeMessageTableKey}
              tradeId={tradeId}
              buyerAvatarName={buyerAvatarName}
              sellerAvatarName={sellerAvatarName}
              returnReportCase={returnReportCase}
              returnReport={returnReport}
            />
          }
          aside={
            <ResaleDetailAside
              resale={resale}
              loading={loading}
              error={error}
              onReload={reload}
              returnReportCaseId={returnReportCase?.id}
              returnReportLoading={returnReportLoading}
              returnReportError={returnReportError}
            />
          }
        />
      </Page>

      {returnReportCase?.targetType === "TRADE" ? (
        <ReportDecisionModal
          open={decisionModalOpen}
          status={returnReportCase.status}
          targetType={returnReportCase.targetType}
          decisionReason={decisionReason}
          continueTrade={continueTrade}
          deciding={deciding}
          decisionError={decisionAttempted ? decisionError : null}
          canKeep={canKeep}
          canRemove={canRemove}
          onChangeDecisionReason={setDecisionReason}
          onChangeContinueTrade={setContinueTrade}
          onClose={closeDecisionModal}
          onKeep={handleKeep}
          onRemove={handleRemove}
        />
      ) : null}
    </>
  );
}