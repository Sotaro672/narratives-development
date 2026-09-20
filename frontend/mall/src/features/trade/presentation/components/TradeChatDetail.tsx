// frontend/mall/src/features/trade/presentation/components/TradeChatDetail.tsx

import { useNavigate } from "react-router-dom";

import Layout from "../../../../components/layout/Layout";
import Alert from "../../../../components/ui/Alert";
import TextState from "../../../../components/ui/TextState";
import ReturnRequestModal from "../../../order/components/ReturnRequestModal";
import ReportModal from "../../../report/components/ReportModal";
import ChatComposerModal from "../../../shared/presentation/components/ChatComposerModal";
import "../../../shared/styles/trade-chat-detail.css";

import useTradeCancel from "../hooks/useTradeCancel";
import useTradeMessageReport from "../hooks/useTradeMessageReport";
import useTradeReply from "../hooks/useTradeReply";
import useTradeReturn from "../hooks/useTradeReturn";
import useTradeReturnReceipt from "../hooks/useTradeReturnReceipt";
import useTradeThread from "../hooks/useTradeThread";
import { getTradeOrderAction, getTradeTitle } from "../util/tradeChatDetail";

import TradeMessageCard from "./TradeMessageCard";
import TradeOrderActionPrompt from "./TradeOrderActionPrompt";
import TradeReturnReceiptModal from "./TradeReturnReceiptModal";
import TradeThreadHeader from "./TradeThreadHeader";

import "../../../../styles/order-detail-page.css";

type TradeChatDetailProps = {
  tradeId: string;
  onBack: () => void;
};

export default function TradeChatDetail({
  tradeId,
  onBack,
}: TradeChatDetailProps) {
  const navigate = useNavigate();
  const thread = useTradeThread(tradeId);

  const cancelFlow = useTradeCancel({
    tradeId: thread.tradeId,
    trade: thread.trade,
    reload: thread.reload,
  });

  const returnFlow = useTradeReturn({
    tradeId: thread.tradeId,
    trade: thread.trade,
    reload: thread.reload,
    blocked: cancelFlow.cancelling,
  });

  const returnReceiptFlow = useTradeReturnReceipt({
    tradeId: thread.tradeId,
    trade: thread.trade,
    reload: thread.reload,
    blocked: cancelFlow.cancelling || returnFlow.returning,
  });

  const reply = useTradeReply({
    tradeId: thread.tradeId,
    trade: thread.trade,
    setTrade: thread.setTrade,
    loading: thread.loading,
    blocked:
      cancelFlow.cancelling ||
      returnFlow.returning ||
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
      returnFlow.returning ||
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
        if (trade.viewerSide !== "seller" || trade.isDispatched) {
          return;
        }

        navigate(`/dispatch/trades/${encodeURIComponent(thread.tradeId)}`);
        return;

      case "return":
        returnFlow.openModal();
        return;

      case "receive-return":
        if (
          trade.viewerSide !== "seller" ||
          !trade.isDispatched ||
          !trade.isReturnRequested ||
          trade.isReturnCompleted
        ) {
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
    returnFlow.returning ||
    returnReceiptFlow.submitting;

  return (
    <>
      <Layout
        title={title}
        showBackButton
        onBackButtonClick={onBack}
        showFooter={
          !reply.open &&
          !cancelFlow.open &&
          !returnFlow.open &&
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
            <Alert variant="error" className="chat-detail-page__error">
              {thread.error}
            </Alert>
          ) : null}

          {thread.loading ? (
            <TextState
              variant="loading"
              className="chat-detail-page__state"
            >
              読み込み中...
            </TextState>
          ) : null}

          {!thread.loading && !thread.trade ? (
            <TextState
              variant="empty"
              className="chat-detail-page__empty"
            >
              取引が見つかりません。
            </TextState>
          ) : null}

          {!thread.loading && thread.trade ? (
            <div className="chat-detail-page__thread">
              <TradeThreadHeader trade={thread.trade} />

              <div className="chat-detail-page__reply-section">
                <h3 className="chat-detail-page__section-title">
                  メッセージ一覧
                </h3>

                {thread.messages.length === 0 && !shouldShowOrderAction ? (
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
                        error={
                          orderAction === "receive-return"
                            ? returnReceiptFlow.error
                            : undefined
                        }
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

      <ReturnRequestModal
        open={returnFlow.open}
        packageState={returnFlow.packageState}
        reason={returnFlow.reason}
        error={returnFlow.error}
        submitting={returnFlow.returning}
        onPackageStateChange={returnFlow.setPackageState}
        onReasonChange={returnFlow.setReason}
        onCancel={returnFlow.closeModal}
        onSubmit={() => {
          void returnFlow.submit();
        }}
      />

      <TradeReturnReceiptModal
        open={returnReceiptFlow.open}
        merchandiseRefundAmount={returnReceiptFlow.merchandiseRefundAmount}
        merchandiseRefundMaxAmount={returnReceiptFlow.merchandiseRefundMaxAmount}
        refundOutboundShipping={returnReceiptFlow.refundOutboundShipping}
        coverReturnShipping={returnReceiptFlow.coverReturnShipping}
        error={returnReceiptFlow.error}
        submitting={returnReceiptFlow.submitting}
        selectionLocked={returnReceiptFlow.selectionLocked}
        canSubmit={returnReceiptFlow.canSubmit}
        onMerchandiseRefundAmountChange={
          returnReceiptFlow.setMerchandiseRefundAmount
        }
        onRefundOutboundShippingChange={
          returnReceiptFlow.setRefundOutboundShipping
        }
        onCoverReturnShippingChange={
          returnReceiptFlow.setCoverReturnShipping
        }
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