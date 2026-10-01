// frontend/mall/src/features/trade/presentation/components/TradeChatDetail.tsx

import {
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useNavigate } from "react-router-dom";

import { useMobilePortrait } from "../../../../components/hooks/useMobilePortrait";
import Alert from "../../../../components/ui/Alert";
import Preview from "../../../../components/ui/Preview";
import StatePanel from "../../../../components/ui/StatePanel";
import useIdentityVerification from "../../../identityVerification/hooks/useIdentityVerification";
import ReportModal from "../../../report/components/ReportModal";
import ChatComposerModal from "../../../shared/presentation/components/ChatComposerModal";
import ChatInlineComposer from "../../../shared/presentation/components/ChatInlineComposer";
import ChatMessageBubble from "../../../shared/presentation/components/ChatMessageBubble";
import { useChatWorkspace } from "../../../shared/presentation/context/ChatWorkspaceContext";
import "../../../shared/styles/trade-chat-detail.css";

import useTradeCancel from "../hooks/useTradeCancel";
import useTradeMessageReport from "../hooks/useTradeMessageReport";
import useTradeReply from "../hooks/useTradeReply";
import useTradeReturnAgreement from "../hooks/useTradeReturnAgreement";
import useTradeReturnConsultation from "../hooks/useTradeReturnConsultation";
import useTradeReturnDispute from "../hooks/useTradeReturnDispute";
import useTradeReturnProposal from "../hooks/useTradeReturnProposal";
import useTradeReturnReceipt from "../hooks/useTradeReturnReceipt";
import useTradeReturnShipment from "../hooks/useTradeReturnShipment";
import useTradeThread from "../hooks/useTradeThread";
import {
  getTradeOrderAction,
  getTradeTitle,
} from "../util/tradeChatDetail";
import { createTradeDispatchQrPayload } from "../util/tradeDispatchQr";

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
  const isMobilePortrait = useMobilePortrait();

  const {
    registerComposer,
    setHeaderTitle,
    clearHeaderTitle,
  } = useChatWorkspace();

  const unregisterComposerRef = useRef<(() => void) | null>(null);
  const thread = useTradeThread(tradeId);
  const [dispatchQrPreviewOpen, setDispatchQrPreviewOpen] = useState(false);

  const identityVerification = useIdentityVerification({
    enabled: thread.trade?.viewerSide === "buyer",
  });

  const dispatchQrPayload = useMemo(
    () => createTradeDispatchQrPayload(thread.tradeId),
    [thread.tradeId],
  );

  const cancelFlow = useTradeCancel({
    tradeId: thread.tradeId,
    trade: thread.trade,
    reload: thread.reload,
  });

  const returnConsultationFlow = useTradeReturnConsultation({
    tradeId: thread.tradeId,
    trade: thread.trade,
    reload: thread.reload,
    blocked:
      cancelFlow.cancelling ||
      identityVerification.isLoading ||
      !identityVerification.isVerified,
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

  const returnDisputeFlow = useTradeReturnDispute({
    tradeId: thread.tradeId,
    trade: thread.trade,
    reload: thread.reload,
    blocked:
      cancelFlow.cancelling ||
      returnConsultationFlow.submitting ||
      returnProposalFlow.submitting ||
      returnAgreementFlow.submitting ||
      returnShipmentFlow.loading ||
      returnReceiptFlow.submitting,
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
      returnReceiptFlow.submitting ||
      returnDisputeFlow.submitting,
  });

  const report = useTradeMessageReport({
    tradeId: thread.tradeId,
    trade: thread.trade,
  });

  useEffect(() => {
    unregisterComposerRef.current = registerComposer({
      content: reply.content,
      placeholder: "メッセージを入力",
      files: reply.files,
      error: reply.error,
      submitting: reply.submitting,
      canSubmit: reply.canSubmit,
      disabled: reply.actionDisabled,
      maxFiles: 10,
      accept: "image/*",
      onContentChange: reply.setContent,
      onFilesAdd: reply.addFiles,
      onRemoveFile: reply.removeFile,
      onSubmit: reply.submit,
    });
  }, [
    registerComposer,
    reply.actionDisabled,
    reply.addFiles,
    reply.canSubmit,
    reply.content,
    reply.error,
    reply.files,
    reply.removeFile,
    reply.setContent,
    reply.submit,
    reply.submitting,
  ]);

  useEffect(() => {
    return () => {
      unregisterComposerRef.current?.();
      unregisterComposerRef.current = null;
    };
  }, []);

  useEffect(() => {
    if (!thread.trade) {
      setHeaderTitle("取引");
      return;
    }

    setHeaderTitle(
      getTradeTitle(
        thread.trade.productName,
      ),
    );
  }, [
    setHeaderTitle,
    thread.trade,
  ]);

  useEffect(() => {
    return () => {
      clearHeaderTitle();
    };
  }, [clearHeaderTitle]);

  const orderAction = getTradeOrderAction(thread.trade);

  const handleOrderAction = (): void => {
    const trade = thread.trade;

    if (
      cancelFlow.cancelling ||
      returnConsultationFlow.submitting ||
      returnProposalFlow.submitting ||
      returnAgreementFlow.submitting ||
      returnShipmentFlow.loading ||
      returnReceiptFlow.submitting ||
      returnDisputeFlow.submitting ||
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

        if (identityVerification.isLoading) {
          return;
        }

        if (!identityVerification.isVerified) {
          navigate("/settings/identity-verification");
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

      case "report-return-dispute":
        if (
          trade.viewerSide !== "buyer" ||
          !trade.isDispatched ||
          trade.transferred ||
          trade.returnStatus !== "discussing" ||
          trade.returnProposal?.agreement !== "disagree"
        ) {
          return;
        }

        returnDisputeFlow.openModal();
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
    returnReceiptFlow.submitting ||
    returnDisputeFlow.submitting ||
    (
      orderAction === "start-return-consultation" &&
      identityVerification.isLoading
    );

  const orderActionError =
    orderAction === "report-return-dispute"
      ? returnDisputeFlow.error
      : orderAction === "prepare-return-shipment"
        ? returnShipmentFlow.error
        : orderAction === "receive-return"
          ? returnReceiptFlow.error
          : orderAction === "review-return-proposal"
            ? returnAgreementFlow.error
            : orderAction === "respond-return-consultation"
              ? returnProposalFlow.error
              : orderAction === "start-return-consultation"
                ? identityVerification.errorMessage ||
                  returnConsultationFlow.error
                : undefined;

  return (
    <>
      <div
        className="chat-detail-page-layout chat-detail-page-layout--inquiry"
        data-chat-detail
      >
        <section className="product-detail-page-layout chat-detail-page">
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
            <div className="chat-detail-page__split">
              <div className="chat-detail-page__left">
                <TradeThreadHeader
                  trade={thread.trade}
                />
              </div>

              <div className="chat-detail-page__right">
                <div className="chat-detail-page__reply-section">
                  <div className="chat-detail-page__replies">
                    <ChatMessageBubble
                      senderName={
                        thread.trade.buyerAvatarName ||
                        "購入者"
                      }
                      senderIcon={
                        thread.trade.buyerAvatarIcon
                      }
                      createdAt={
                        thread.trade.createdAt
                      }
                      content={`${
                        thread.trade.buyerAvatarName ||
                        "購入者"
                      }さんが${
                        thread.trade.productName ||
                        "商品"
                      }を購入しました。`}
                      isMine={
                        thread.trade.viewerSide ===
                        "buyer"
                      }
                    />

                    {thread.messages.map((message) => (
                      <TradeMessageCard
                        key={message.id}
                        message={message}
                        trade={thread.trade!}
                        onReport={
                          report.openMessageReport
                        }
                        onOpenDispatchQr={
                          thread.trade!.viewerSide ===
                          "seller"
                            ? () => {
                                setDispatchQrPreviewOpen(
                                  true,
                                );
                              }
                            : undefined
                        }
                        onOpenReturnShipmentQr={
                          thread.trade!.viewerSide ===
                          "buyer"
                            ? () => {
                                void returnShipmentFlow.openPersistedShipment();
                              }
                            : undefined
                        }
                      />
                    ))}

                    {thread.trade.transferredAt ? (
                      <ChatMessageBubble
                        senderName={
                          thread.trade
                            .buyerAvatarName ||
                          "購入者"
                        }
                        senderIcon={
                          thread.trade
                            .buyerAvatarIcon
                        }
                        createdAt={
                          thread.trade
                            .transferredAt
                        }
                        content={`${
                          thread.trade
                            .buyerAvatarName ||
                          "購入者"
                        }さんがトークンを受け取りました。`}
                        isMine={
                          thread.trade
                            .viewerSide ===
                          "buyer"
                        }
                      />
                    ) : null}

                    {orderAction ? (
                      <TradeOrderActionPrompt
                        action={orderAction}
                        processing={
                          orderActionProcessing
                        }
                        error={
                          orderActionError
                        }
                        onAction={
                          handleOrderAction
                        }
                      />
                    ) : null}
                  </div>
                </div>

                <ChatInlineComposer
                  content={reply.content}
                  placeholder="メッセージを入力"
                  files={reply.files}
                  error={reply.error}
                  submitting={
                    reply.submitting
                  }
                  canSubmit={
                    reply.canSubmit
                  }
                  disabled={
                    reply.actionDisabled
                  }
                  onContentChange={
                    reply.setContent
                  }
                  onFilesAdd={
                    reply.addFiles
                  }
                  onRemoveFile={
                    reply.removeFile
                  }
                  onSubmit={
                    reply.submit
                  }
                />
              </div>
            </div>
          ) : null}
        </section>
      </div>

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
        description="出品者へ送るキャンセル理由のメッセージを入力してください。"
        rows={isMobilePortrait ? 1 : 6}
        onContentChange={cancelFlow.setMessage}
        onCancel={cancelFlow.closeModal}
        onSubmit={() => {
          void cancelFlow.submit();
        }}
      />

      <ChatComposerModal
        open={returnDisputeFlow.open}
        title="運営へ報告する"
        content={returnDisputeFlow.content}
        placeholder="取引の状況や解決してほしい内容を入力してください"
        error={returnDisputeFlow.error}
        submitting={returnDisputeFlow.submitting}
        canSubmit={returnDisputeFlow.canSubmit}
        submitLabel="運営へ報告する"
        submittingLabel="報告中..."
        description="出品者が返品に合意しなかったため、この取引を運営へ報告できます。取引の状況や確認してほしい内容を入力してください。"
        maxLength={5000}
        onContentChange={returnDisputeFlow.setContent}
        onCancel={returnDisputeFlow.closeModal}
        onSubmit={() => {
          void returnDisputeFlow.submit();
        }}
      />

      <TradeReturnConsultationModal
        open={returnConsultationFlow.open}
        reason={returnConsultationFlow.reason}
        detail={returnConsultationFlow.detail}
        error={returnConsultationFlow.error}
        submitting={
          returnConsultationFlow.submitting
        }
        onReasonChange={
          returnConsultationFlow.setReason
        }
        onDetailChange={
          returnConsultationFlow.setDetail
        }
        onCancel={
          returnConsultationFlow.closeModal
        }
        onSubmit={() => {
          void returnConsultationFlow.submit();
        }}
      />

      <TradeReturnProposalModal
        open={returnProposalFlow.open}
        agreement={returnProposalFlow.agreement}
        returnRequirement={
          returnProposalFlow.returnRequirement
        }
        refundAmount={
          returnProposalFlow.refundAmount
        }
        refundAmountMax={
          returnProposalFlow.refundAmountMax
        }
        error={returnProposalFlow.error}
        submitting={
          returnProposalFlow.submitting
        }
        onAgreementChange={
          returnProposalFlow.setAgreement
        }
        onReturnRequirementChange={
          returnProposalFlow.setReturnRequirement
        }
        onRefundAmountChange={
          returnProposalFlow.setRefundAmount
        }
        onCancel={
          returnProposalFlow.closeModal
        }
        onSubmit={() => {
          void returnProposalFlow.submit();
        }}
      />

      <TradeReturnAgreementModal
        open={returnAgreementFlow.open}
        proposal={returnAgreementFlow.proposal}
        error={returnAgreementFlow.error}
        submitting={
          returnAgreementFlow.submitting
        }
        accepting={
          returnAgreementFlow.accepting
        }
        rejecting={
          returnAgreementFlow.rejecting
        }
        onCancel={
          returnAgreementFlow.closeModal
        }
        onAccept={() => {
          void returnAgreementFlow.accept();
        }}
        onReject={() => {
          void returnAgreementFlow.reject();
        }}
      />

      <TradeReturnShipmentModal
        open={returnShipmentFlow.open}
        shipment={
          returnShipmentFlow.shipment
        }
        qrCodePayload={
          returnShipmentFlow.qrCodePayload
        }
        error={returnShipmentFlow.error}
        loading={
          returnShipmentFlow.loading
        }
        onCancel={
          returnShipmentFlow.closeModal
        }
        onRetryPreparation={() => {
          void returnShipmentFlow.retryPreparation();
        }}
      />

      <TradeReturnReceiptModal
        open={returnReceiptFlow.open}
        proposal={
          returnReceiptFlow.proposal
        }
        refundAmount={
          returnReceiptFlow.refundAmount
        }
        error={returnReceiptFlow.error}
        submitting={
          returnReceiptFlow.submitting
        }
        canSubmit={
          returnReceiptFlow.canSubmit
        }
        onCancel={
          returnReceiptFlow.closeModal
        }
        onSubmit={() => {
          void returnReceiptFlow.submit();
        }}
      />

      <Preview
        open={
          thread.trade?.viewerSide === "seller" &&
          dispatchQrPreviewOpen &&
          Boolean(dispatchQrPayload)
        }
        type="qr"
        qrValue={dispatchQrPayload}
        alt="PUDO発送用QRコード"
        onClose={() => {
          setDispatchQrPreviewOpen(false);
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