// frontend/mall/src/features/trade/presentation/components/TradeChatDetail.tsx

import { useNavigate } from "react-router-dom";

import Layout from "../../../../components/layout/Layout";
import Alert from "../../../../components/ui/Alert";
import StatePanel from "../../../../components/ui/StatePanel";
import TextState from "../../../../components/ui/TextState";
import ReportModal from "../../../report/components/ReportModal";
import ChatComposerModal from "../../../shared/presentation/components/ChatComposerModal";
import "../../../shared/styles/trade-chat-detail.css";

import useTradeCancel from "../hooks/useTradeCancel";
import useTradeMessageReport from "../hooks/useTradeMessageReport";
import useTradeReply from "../hooks/useTradeReply";
import useTradeReturnAgreement from "../hooks/useTradeReturnAgreement";
import useTradeReturnConsultation from "../hooks/useTradeReturnConsultation";
import useTradeReturnProposal from "../hooks/useTradeReturnProposal";
import useTradeReturnReceipt from "../hooks/useTradeReturnReceipt";
import useTradeReturnShipment from "../hooks/useTradeReturnShipment";
import useTradeThread from "../hooks/useTradeThread";
import { getTradeOrderAction, getTradeTitle } from "../util/tradeChatDetail";

import TradeMessageCard from "./TradeMessageCard";
import TradeOrderActionPrompt from "./TradeOrderActionPrompt";
import TradeReturnAgreementModal from "./TradeReturnAgreementModal";
import TradeReturnConsultationModal from "./TradeReturnConsultationModal";
import TradeReturnProposalModal from "./TradeReturnProposalModal";
import TradeReturnReceiptModal from "./TradeReturnReceiptModal";
import TradeReturnShipmentModal from "./TradeReturnShipmentModal";
import TradeThreadHeader from "./TradeThreadHeader";

type TradeChatDetailProps = {
  tradeId: string;
};

export default function TradeChatDetail({
  tradeId,
}: TradeChatDetailProps) {
  const navigate = useNavigate();
  const thread = useTradeThread(tradeId);

  const cancelFlow = useTradeCancel({
    tradeId: thread.tradeId,
    trade: thread.trade,
    reload: thread.reload,
  });

  const returnConsultationFlow = useTradeReturnConsultation({
    tradeId: thread.tradeId,
    trade: thread.trade,
    reload: thread.reload,
    blocked: cancelFlow.cancelling,
  });

  const returnProposalFlow = useTradeReturnProposal({
    tradeId: thread.tradeId,
    trade: thread.trade,
    reload: thread.reload,
    blocked:
      cancelFlow.cancelling ||
      returnConsultationFlow.submitting,
  });

  const returnAgreementFlow = useTradeReturnAgreement({
    tradeId: thread.tradeId,
    trade: thread.trade,
    reload: thread.reload,
    blocked:
      cancelFlow.cancelling ||
      returnConsultationFlow.submitting ||
      returnProposalFlow.submitting,
  });

  const returnShipmentFlow = useTradeReturnShipment({
    tradeId: thread.tradeId,
    trade: thread.trade,
    reload: thread.reload,
    blocked:
      cancelFlow.cancelling ||
      returnConsultationFlow.submitting ||
      returnProposalFlow.submitting ||
      returnAgreementFlow.submitting,
  });

  const returnReceiptFlow = useTradeReturnReceipt({
    tradeId: thread.tradeId,
    trade: thread.trade,
    reload: thread.reload,
    blocked:
      cancelFlow.cancelling ||
      returnConsultationFlow.submitting ||
      returnProposalFlow.submitting ||
      returnAgreementFlow.submitting ||
      returnShipmentFlow.loading,
  });

  const reply = useTradeReply({
    tradeId: thread.tradeId,
    trade: thread.trade,
    setTrade: thread.setTrade,
    loading: thread.loading,
    blocked:
      cancelFlow.cancelling ||
      returnConsultationFlow.submitting ||
      returnProposalFlow.submitting ||
      returnAgreementFlow.submitting ||
      returnShipmentFlow.loading ||
      returnReceiptFlow.submitting,
  });

  const report = useTradeMessageReport({
    tradeId: thread.tradeId,
    trade: thread.trade,
  });

  const title = getTradeTitle(thread.trade?.productName);
  const orderAction = getTradeOrderAction(thread.trade);
  const shouldShowOrderAction = orderAction !== null;

  const handleOrderAction = (): void => {
    const trade = thread.trade;

    if (
      cancelFlow.cancelling ||
      returnConsultationFlow.submitting ||
      returnProposalFlow.submitting ||
      returnAgreementFlow.submitting ||
      returnShipmentFlow.loading ||
      returnReceiptFlow.submitting ||
      !trade ||
      !orderAction ||
      trade.status !== "active" ||
      trade.isCancelled ||
      !thread.tradeId
    ) {
      return;
    }

    switch (orderAction) {
      case "dispatch":
        if (
          trade.viewerSide !== "seller" ||
          trade.isDispatched
        ) {
          return;
        }

        navigate(
          `/dispatch/trades/${encodeURIComponent(thread.tradeId)}`,
        );
        return;

      case "start-return-consultation":
        if (
          trade.viewerSide !== "buyer" ||
          !trade.isDispatched ||
          trade.transferred ||
          trade.returnStatus !== "none"
        ) {
          return;
        }

        returnConsultationFlow.openModal();
        return;

      case "respond-return-consultation":
        if (
          trade.viewerSide !== "seller" ||
          !trade.isDispatched ||
          trade.returnStatus !== "discussing"
        ) {
          return;
        }

        returnProposalFlow.openModal();
        return;

      case "review-return-proposal":
        if (
          trade.viewerSide !== "buyer" ||
          !trade.isDispatched ||
          trade.transferred ||
          trade.returnStatus !== "proposed" ||
          !trade.returnProposal
        ) {
          return;
        }

        returnAgreementFlow.openModal();
        return;

      case "prepare-return-shipment": {
        const proposal = trade.returnProposal;

        if (
          trade.viewerSide !== "buyer" ||
          !trade.isDispatched ||
          trade.transferred ||
          trade.returnStatus !== "agreed" ||
          !proposal ||
          proposal.agreement !== "agree" ||
          proposal.rejectedAt ||
          proposal.returnRequirement !== "required"
        ) {
          return;
        }

        void returnShipmentFlow.openModal();
        return;
      }

      case "receive-return":
        if (!returnReceiptFlow.canSubmit) {
          return;
        }

        returnReceiptFlow.openModal();
        return;

      case "cancel":
        cancelFlow.openModal();
        return;
    }
  };

  const orderActionProcessing =
    cancelFlow.cancelling ||
    returnConsultationFlow.submitting ||
    returnProposalFlow.submitting ||
    returnAgreementFlow.submitting ||
    returnShipmentFlow.loading ||
    returnReceiptFlow.submitting;

  const orderActionError =
    orderAction === "prepare-return-shipment"
      ? returnShipmentFlow.error
      : orderAction === "receive-return"
        ? returnReceiptFlow.error
        : orderAction === "review-return-proposal"
          ? returnAgreementFlow.error
          : orderAction === "respond-return-consultation"
            ? returnProposalFlow.error
            : orderAction === "start-return-consultation"
              ? returnConsultationFlow.error
              : undefined;

  return (
    <>
      <Layout
        title={title}
        showFooter={
          !reply.open &&
          !cancelFlow.open &&
          !returnConsultationFlow.open &&
          !returnProposalFlow.open &&
          !returnAgreementFlow.open &&
          !returnShipmentFlow.open &&
          !returnReceiptFlow.open &&
          !report.isOpen
        }
        mode="mypage"
        mainClassName="chat-detail-page-layout"
        actionButtonLabel="返信"
        onActionButtonClick={reply.openModal}
        actionButtonDisabled={reply.actionDisabled}
        footerProps={{
          variant: "default",
          centerActionLabel: "返信",
          centerActionDisabled: reply.actionDisabled,
          onCenterActionClick: reply.openModal,
        }}
      >
        <section className="page-section content-page-section chat-detail-page">
          {thread.error ? (
            <Alert
              variant="error"
              className="chat-detail-page__error"
            >
              {thread.error}
            </Alert>
          ) : null}

          {thread.loading ? (
            <StatePanel
              variant="loading"
              title="読み込み中..."
            />
          ) : null}

          {!thread.loading && !thread.trade ? (
            <StatePanel
              variant="empty"
              title="取引が見つかりません。"
            />
          ) : null}

          {!thread.loading && thread.trade ? (
            <div className="chat-detail-page__thread">
              <TradeThreadHeader trade={thread.trade} />

              <div className="chat-detail-page__reply-section">
                <h3 className="chat-detail-page__section-title">
                  メッセージ一覧
                </h3>

                {thread.messages.length === 0 &&
                !shouldShowOrderAction ? (
                  <TextState
                    variant="empty"
                    className="chat-detail-page__no-replies"
                  >
                    まだメッセージはありません。
                  </TextState>
                ) : (
                  <div className="chat-detail-page__replies">
                    {thread.messages.map((message) => (
                      <TradeMessageCard
                        key={message.id}
                        message={message}
                        trade={thread.trade!}
                        onReport={report.openMessageReport}
                      />
                    ))}

                    {orderAction ? (
                      <TradeOrderActionPrompt
                        action={orderAction}
                        processing={orderActionProcessing}
                        error={orderActionError}
                        onAction={handleOrderAction}
                      />
                    ) : null}
                  </div>
                )}
              </div>
            </div>
          ) : null}
        </section>
      </Layout>

      <ChatComposerModal
        open={reply.open}
        title="返信する"
        content={reply.content}
        placeholder="メッセージを入力"
        error={reply.error}
        submitting={reply.submitting}
        canSubmit={reply.canSubmit}
        submitLabel="送信"
        submittingLabel="送信中..."
        onContentChange={reply.setContent}
        onCancel={reply.closeModal}
        onSubmit={() => {
          void reply.submit();
        }}
      />

      <ChatComposerModal
        open={cancelFlow.open}
        title="注文をキャンセルする"
        content={cancelFlow.message}
        placeholder="キャンセル理由を入力してください"
        error={cancelFlow.error}
        submitting={cancelFlow.cancelling}
        canSubmit={cancelFlow.canSubmit}
        submitLabel="メッセージを送信してキャンセル"
        submittingLabel="キャンセル中..."
        cancelLabel="戻る"
        description="出品者へ送るキャンセル理由のメッセージを入力してください。"
        onContentChange={cancelFlow.setMessage}
        onCancel={cancelFlow.closeModal}
        onSubmit={() => {
          void cancelFlow.submit();
        }}
      />

      <TradeReturnConsultationModal
        open={returnConsultationFlow.open}
        reason={returnConsultationFlow.reason}
        detail={returnConsultationFlow.detail}
        error={returnConsultationFlow.error}
        submitting={returnConsultationFlow.submitting}
        onReasonChange={returnConsultationFlow.setReason}
        onDetailChange={returnConsultationFlow.setDetail}
        onCancel={returnConsultationFlow.closeModal}
        onSubmit={() => {
          void returnConsultationFlow.submit();
        }}
      />

      <TradeReturnProposalModal
        open={returnProposalFlow.open}
        agreement={returnProposalFlow.agreement}
        returnRequirement={returnProposalFlow.returnRequirement}
        refundAmount={returnProposalFlow.refundAmount}
        refundAmountMax={returnProposalFlow.refundAmountMax}
        error={returnProposalFlow.error}
        submitting={returnProposalFlow.submitting}
        onAgreementChange={returnProposalFlow.setAgreement}
        onReturnRequirementChange={
          returnProposalFlow.setReturnRequirement
        }
        onRefundAmountChange={
          returnProposalFlow.setRefundAmount
        }
        onCancel={returnProposalFlow.closeModal}
        onSubmit={() => {
          void returnProposalFlow.submit();
        }}
      />

      <TradeReturnAgreementModal
        open={returnAgreementFlow.open}
        proposal={returnAgreementFlow.proposal}
        error={returnAgreementFlow.error}
        submitting={returnAgreementFlow.submitting}
        accepting={returnAgreementFlow.accepting}
        rejecting={returnAgreementFlow.rejecting}
        onCancel={returnAgreementFlow.closeModal}
        onAccept={() => {
          void returnAgreementFlow.accept();
        }}
        onReject={() => {
          void returnAgreementFlow.reject();
        }}
      />

      <TradeReturnShipmentModal
        open={returnShipmentFlow.open}
        shipment={returnShipmentFlow.shipment}
        qrCodePayload={returnShipmentFlow.qrCodePayload}
        error={returnShipmentFlow.error}
        loading={returnShipmentFlow.loading}
        onCancel={returnShipmentFlow.closeModal}
        onRetryPreparation={() => {
          void returnShipmentFlow.retryPreparation();
        }}
      />

      <TradeReturnReceiptModal
        open={returnReceiptFlow.open}
        proposal={returnReceiptFlow.proposal}
        refundAmount={returnReceiptFlow.refundAmount}
        error={returnReceiptFlow.error}
        submitting={returnReceiptFlow.submitting}
        canSubmit={returnReceiptFlow.canSubmit}
        onCancel={returnReceiptFlow.closeModal}
        onSubmit={() => {
          void returnReceiptFlow.submit();
        }}
      />

      <ReportModal
        open={report.isOpen}
        targetType={report.target?.type}
        reason={report.reason}
        detail={report.detail}
        submitting={report.submitting}
        error={report.error}
        result={report.result}
        canSubmit={report.canSubmit}
        onReasonChange={report.setReason}
        onDetailChange={report.setDetail}
        onSubmit={report.submit}
        onClose={report.close}
      />
    </>
  );
}