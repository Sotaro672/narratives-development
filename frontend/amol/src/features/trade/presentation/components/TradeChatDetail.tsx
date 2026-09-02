// frontend/amol/src/features/trade/presentation/components/TradeChatDetail.tsx

import { useCallback, useEffect, useMemo, useState } from "react";
import { Copy } from "lucide-react";
import { useNavigate } from "react-router-dom";

import Layout from "../../../../components/layout/Layout";
import { formatDateTime } from "../../../../components/utils/date";
import { getApiBaseUrl } from "../../../../lib/apiBaseUrl";
import { getFirebaseIdToken } from "../../../../lib/authToken";

import { returnOrderItem } from "../../../order/api/orderDetailApi";
import ReturnRequestModal, { type ReturnPackageState } from "../../../order/components/ReturnRequestModal";
import ChatComposerModal from "../../../shared/presentation/components/ChatComposerModal";
import ChatImageGrid from "../../../shared/presentation/components/ChatImageGrid";
import ChatMessageBubble from "../../../shared/presentation/components/ChatMessageBubble";
import ChatMessageHeader from "../../../shared/presentation/components/ChatMessageHeader";
import ChatMetaSection, { type ChatMetaItem } from "../../../shared/presentation/components/ChatMetaSection";
import ChatThreadCard from "../../../shared/presentation/components/ChatThreadCard";
import { createProductModelDisplay } from "../../../shared/presentation/utils/productModelDisplay";
import "../../../shared/styles/trade-chat-detail.css";
import type { TradeDetail, TradeMessage } from "../../../shared/types/trade";

import {
  cancelTradeOrderItem,
  createTradeMessage,
  fetchTradeById,
  markTradeMessagesAsRead,
} from "../../infrastructure/tradeApi";

import TradeOrderActionPrompt, { type TradeOrderAction } from "./TradeOrderActionPrompt";
import { getTradeStatusLabel } from "../util/tradeStatus";

import "../../../../styles/order-detail-page.css";

const MESSAGE_LIMIT = 100;

type TradeChatDetailProps = {
  tradeId: string;
  onBack: () => void;
};

type TradeSenderDisplay = {
  name: string;
  icon: string;
};

function getErrorMessage(caught: unknown, fallbackMessage: string): string {
  if (caught instanceof Error && caught.message) return caught.message;
  return fallbackMessage;
}

function getTradeTitle(productName?: string): string {
  return productName ? `${productName}/取引` : "取引";
}

function sortMessages(messages: TradeMessage[]): TradeMessage[] {
  return [...messages].sort(
    (firstMessage, secondMessage) =>
      new Date(firstMessage.createdAt).getTime() -
      new Date(secondMessage.createdAt).getTime(),
  );
}

function getTradeOrderAction(trade: TradeDetail | null): TradeOrderAction | null {
  if (!trade || trade.status !== "active" || trade.isCancelled) return null;

  if (trade.viewerSide === "seller") {
    return trade.isDispatched ? null : "dispatch";
  }

  if (!trade.isDispatched) return "cancel";

  if (
    !trade.transferred &&
    !trade.isReturnRequested &&
    !trade.isReturnCompleted
  ) {
    return "return";
  }

  return null;
}

function getTradeMessageSenderDisplay(
  message: TradeMessage,
  trade: TradeDetail,
): TradeSenderDisplay {
  switch (message.senderSide) {
    case "buyer":
      return {
        name: trade.buyerAvatarName || "購入者",
        icon: trade.buyerAvatarIcon || "",
      };

    case "seller":
      return {
        name: trade.sellerAvatarName || "出品者",
        icon: trade.sellerAvatarIcon || "",
      };

    case "system":
      return {
        name: "AMOL",
        icon: "",
      };
  }
}

export default function TradeChatDetail({
  tradeId,
  onBack,
}: TradeChatDetailProps) {
  const navigate = useNavigate();
  const normalizedTradeId = tradeId.trim();

  const [trade, setTrade] = useState<TradeDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const [isReplyModalOpen, setIsReplyModalOpen] = useState(false);
  const [replyContent, setReplyContent] = useState("");
  const [replyError, setReplyError] = useState("");
  const [postingReply, setPostingReply] = useState(false);

  const [isCancelModalOpen, setIsCancelModalOpen] = useState(false);
  const [cancelMessage, setCancelMessage] = useState("");
  const [cancelError, setCancelError] = useState("");
  const [cancelling, setCancelling] = useState(false);

  const [isReturnModalOpen, setIsReturnModalOpen] = useState(false);
  const [returnPackageState, setReturnPackageState] = useState<ReturnPackageState | null>(null);
  const [returnReason, setReturnReason] = useState("");
  const [returnError, setReturnError] = useState("");
  const [returning, setReturning] = useState(false);

  const loadThread = useCallback(async (): Promise<void> => {
    if (!normalizedTradeId) {
      setTrade(null);
      setError("取引IDが見つかりません。");
      setLoading(false);
      return;
    }

    setLoading(true);
    setError("");
    setReplyError("");

    try {
      const loadedTrade = await fetchTradeById({
        tradeId: normalizedTradeId,
        limit: MESSAGE_LIMIT,
      });

      setTrade({
        ...loadedTrade,
        messages: sortMessages(loadedTrade.messages),
      });

      try {
        await markTradeMessagesAsRead({
          tradeId: normalizedTradeId,
        });
      } catch (caught) {
        setError(
          getErrorMessage(
            caught,
            "メッセージの既読状態の更新に失敗しました。",
          ),
        );
      }
    } catch (caught) {
      setTrade(null);
      setError(
        getErrorMessage(
          caught,
          "取引の取得に失敗しました。",
        ),
      );
    } finally {
      setLoading(false);
    }
  }, [normalizedTradeId]);

  useEffect(() => {
    void loadThread();
  }, [loadThread]);

  const sortedMessages = useMemo(
    () => sortMessages(trade?.messages ?? []),
    [trade?.messages],
  );

  const title = getTradeTitle(trade?.productName);
  const canSubmitReply = /\S/u.test(replyContent);
  const canSubmitCancel = /\S/u.test(cancelMessage);
  const orderAction = getTradeOrderAction(trade);
  const shouldShowOrderAction = orderAction !== null;

  const replyActionDisabled =
    loading ||
    !trade ||
    trade.status !== "active" ||
    trade.isCancelled ||
    postingReply ||
    cancelling ||
    returning;

  const openReplyModal = useCallback(() => {
    if (replyActionDisabled) return;

    setReplyError("");
    setIsReplyModalOpen(true);
  }, [replyActionDisabled]);

  const closeReplyModal = useCallback(() => {
    if (postingReply) return;

    setIsReplyModalOpen(false);
    setReplyContent("");
    setReplyError("");
  }, [postingReply]);

  const submitReply = useCallback(async (): Promise<void> => {
    if (
      postingReply ||
      cancelling ||
      returning ||
      !trade ||
      trade.status !== "active" ||
      trade.isCancelled ||
      !normalizedTradeId
    ) {
      return;
    }

    if (!/\S/u.test(replyContent)) {
      setReplyError("メッセージを入力してください。");
      return;
    }

    setPostingReply(true);
    setReplyError("");

    try {
      const created = await createTradeMessage({
        tradeId: normalizedTradeId,
        content: replyContent,
      });

      setTrade((currentTrade) => {
        if (!currentTrade) return currentTrade;

        return {
          ...currentTrade,
          messages: sortMessages([
            ...currentTrade.messages.filter(
              (message) => message.id !== created.id,
            ),
            created,
          ]),
          lastMessageAt: created.createdAt,
          updatedAt: created.createdAt,
        };
      });

      setIsReplyModalOpen(false);
      setReplyContent("");
      setReplyError("");
    } catch (caught) {
      setReplyError(
        getErrorMessage(
          caught,
          "メッセージの送信に失敗しました。",
        ),
      );
    } finally {
      setPostingReply(false);
    }
  }, [
    cancelling,
    normalizedTradeId,
    postingReply,
    replyContent,
    returning,
    trade,
  ]);

  const openCancelModal = useCallback(() => {
    if (
      !trade ||
      trade.viewerSide !== "buyer" ||
      trade.status !== "active" ||
      trade.isCancelled ||
      trade.isDispatched ||
      trade.transferred
    ) {
      return;
    }

    setCancelMessage("");
    setCancelError("");
    setIsCancelModalOpen(true);
  }, [trade]);

  const closeCancelModal = useCallback(() => {
    if (cancelling) return;

    setIsCancelModalOpen(false);
    setCancelMessage("");
    setCancelError("");
  }, [cancelling]);

  const submitCancel = useCallback(async (): Promise<void> => {
    if (
      cancelling ||
      !trade ||
      trade.viewerSide !== "buyer" ||
      trade.status !== "active" ||
      trade.isCancelled ||
      trade.isDispatched ||
      trade.transferred ||
      !normalizedTradeId
    ) {
      return;
    }

    const normalizedMessage = cancelMessage.trim();
    if (!normalizedMessage) {
      setCancelError("キャンセル理由のメッセージを入力してください。");
      return;
    }

    setCancelling(true);
    setCancelError("");

    try {
      await createTradeMessage({
        tradeId: normalizedTradeId,
        content: normalizedMessage,
      });

      await cancelTradeOrderItem({
        orderId: trade.orderId,
        orderItemIndex: trade.orderItemIndex,
      });

      setIsCancelModalOpen(false);
      setCancelMessage("");
      setCancelError("");

      await loadThread();
    } catch (caught) {
      setCancelError(
        getErrorMessage(
          caught,
          "注文のキャンセルに失敗しました。",
        ),
      );
    } finally {
      setCancelling(false);
    }
  }, [
    cancelMessage,
    cancelling,
    loadThread,
    normalizedTradeId,
    trade,
  ]);

  const openReturnModal = useCallback(() => {
    if (
      !trade ||
      trade.viewerSide !== "buyer" ||
      trade.status !== "active" ||
      trade.isCancelled ||
      !trade.isDispatched ||
      trade.transferred ||
      trade.isReturnRequested ||
      trade.isReturnCompleted ||
      !normalizedTradeId
    ) {
      return;
    }

    setReturnPackageState(null);
    setReturnReason("");
    setReturnError("");
    setIsReturnModalOpen(true);
  }, [normalizedTradeId, trade]);

  const closeReturnModal = useCallback(() => {
    if (returning) return;

    setIsReturnModalOpen(false);
    setReturnPackageState(null);
    setReturnReason("");
    setReturnError("");
  }, [returning]);

  const submitReturn = useCallback(async (): Promise<void> => {
    if (
      returning ||
      cancelling ||
      !trade ||
      trade.viewerSide !== "buyer" ||
      trade.status !== "active" ||
      trade.isCancelled ||
      !trade.isDispatched ||
      trade.transferred ||
      trade.isReturnRequested ||
      trade.isReturnCompleted ||
      !normalizedTradeId
    ) {
      return;
    }

    if (
      returnPackageState !== "unopened" &&
      returnPackageState !== "opened"
    ) {
      setReturnError("商品の開封状態を選択してください。");
      return;
    }

    const normalizedReason = returnReason.trim();
    if (!normalizedReason) {
      setReturnError("返品理由を入力してください。");
      return;
    }

    setReturning(true);
    setReturnError("");

    try {
      const backendUrl = getApiBaseUrl();
      if (!backendUrl) {
        throw new Error("VITE_API_BASE_URLが設定されていません。");
      }

      const idToken = await getFirebaseIdToken();

      await createTradeMessage({
        tradeId: normalizedTradeId,
        content: normalizedReason,
      });

      await returnOrderItem({
        backendUrl,
        idToken,
        orderId: trade.orderId,
        itemIndex: trade.orderItemIndex,
        packageState: returnPackageState,
        reason: normalizedReason,
      });

      setIsReturnModalOpen(false);
      setReturnPackageState(null);
      setReturnReason("");
      setReturnError("");

      await loadThread();
    } catch (caught) {
      setReturnError(
        getErrorMessage(
          caught,
          "商品の返品受付に失敗しました。",
        ),
      );
    } finally {
      setReturning(false);
    }
  }, [
    cancelling,
    loadThread,
    normalizedTradeId,
    returnPackageState,
    returnReason,
    returning,
    trade,
  ]);

  const handleOrderAction = useCallback((): void => {
    if (
      cancelling ||
      returning ||
      !trade ||
      !orderAction ||
      trade.status !== "active" ||
      trade.isCancelled ||
      !normalizedTradeId
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
          `/dispatch/trades/${encodeURIComponent(normalizedTradeId)}`,
        );
        return;

      case "return":
        openReturnModal();
        return;

      case "cancel":
        openCancelModal();
        return;
    }
  }, [
    cancelling,
    navigate,
    normalizedTradeId,
    openCancelModal,
    openReturnModal,
    orderAction,
    returning,
    trade,
  ]);

  return (
    <>
      <Layout
        title={title}
        showBackButton
        onBackButtonClick={onBack}
        showFooter={
          !isReplyModalOpen &&
          !isCancelModalOpen &&
          !isReturnModalOpen
        }
        mode="mypage"
        mainClassName="chat-detail-page-layout"
        actionButtonLabel="返信"
        onActionButtonClick={openReplyModal}
        actionButtonDisabled={replyActionDisabled}
        footerProps={{
          variant: "default",
          centerActionLabel: "返信",
          centerActionDisabled: replyActionDisabled,
          onCenterActionClick: openReplyModal,
        }}
      >
        <section className="page-section content-page-section chat-detail-page">
          {error ? (
            <div className="chat-detail-page__error" role="alert">
              {error}
            </div>
          ) : null}

          {loading ? (
            <div className="chat-detail-page__state">
              読み込み中...
            </div>
          ) : null}

          {!loading && !trade ? (
            <div className="chat-detail-page__empty">
              取引が見つかりません。
            </div>
          ) : null}

          {!loading && trade ? (
            <div className="chat-detail-page__thread">
              <TradeThreadHeader trade={trade} />

              <div className="chat-detail-page__reply-section">
                <h3 className="chat-detail-page__section-title">
                  メッセージ一覧
                </h3>

                {sortedMessages.length === 0 && !shouldShowOrderAction ? (
                  <div className="chat-detail-page__no-replies">
                    まだメッセージはありません。
                  </div>
                ) : (
                  <div className="chat-detail-page__replies">
                    {sortedMessages.map((message) => (
                      <TradeMessageCard
                        key={message.id}
                        message={message}
                        trade={trade}
                      />
                    ))}

                    {orderAction ? (
                      <TradeOrderActionPrompt
                        action={orderAction}
                        processing={
                          cancelling ||
                          returning
                        }
                        onAction={() => {
                          handleOrderAction();
                        }}
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
        open={isReplyModalOpen}
        title="返信する"
        content={replyContent}
        placeholder="メッセージを入力"
        error={replyError}
        submitting={postingReply}
        canSubmit={canSubmitReply}
        submitLabel="送信"
        submittingLabel="送信中..."
        onContentChange={setReplyContent}
        onCancel={closeReplyModal}
        onSubmit={() => {
          void submitReply();
        }}
      />

      <ChatComposerModal
        open={isCancelModalOpen}
        title="注文をキャンセルする"
        content={cancelMessage}
        placeholder="キャンセル理由を入力してください"
        error={cancelError}
        submitting={cancelling}
        canSubmit={canSubmitCancel}
        submitLabel="メッセージを送信してキャンセル"
        submittingLabel="キャンセル中..."
        cancelLabel="戻る"
        description="出品者へ送るキャンセル理由のメッセージを入力してください。"
        onContentChange={(value) => {
          setCancelMessage(value);
          setCancelError("");
        }}
        onCancel={closeCancelModal}
        onSubmit={() => {
          void submitCancel();
        }}
      />

      <ReturnRequestModal
        open={isReturnModalOpen}
        packageState={returnPackageState}
        reason={returnReason}
        error={returnError}
        submitting={returning}
        onPackageStateChange={(value) => {
          setReturnPackageState(value);
          setReturnError("");
        }}
        onReasonChange={(value) => {
          setReturnReason(value);
          setReturnError("");
        }}
        onCancel={closeReturnModal}
        onSubmit={() => {
          void submitReturn();
        }}
      />
    </>
  );
}

function TradeThreadHeader({
  trade,
}: {
  trade: TradeDetail;
}) {
  const counterpartLabel =
    trade.viewerSide === "buyer"
      ? "出品者"
      : "購入者";

  const counterpartAvatarName =
    trade.viewerSide === "buyer"
      ? trade.sellerAvatarName
      : trade.buyerAvatarName;

  const counterpartAvatarIcon =
    trade.viewerSide === "buyer"
      ? trade.sellerAvatarIcon
      : trade.buyerAvatarIcon;

  const displayName =
    counterpartAvatarName || counterpartLabel;

  const title = getTradeTitle(trade.productName);
  const model = createProductModelDisplay(trade.resale);
  const [orderIdCopied, setOrderIdCopied] = useState(false);

  useEffect(() => {
    if (!orderIdCopied) return;

    const timeoutId = window.setTimeout(() => {
      setOrderIdCopied(false);
    }, 2000);

    return () => {
      window.clearTimeout(timeoutId);
    };
  }, [orderIdCopied]);

  const handleCopyOrderId = async (): Promise<void> => {
    try {
      await navigator.clipboard.writeText(trade.orderId);
      setOrderIdCopied(true);
    } catch {
      // Clipboard API が利用できない環境では何もしない。
    }
  };

  const transactionMetaItems: ChatMetaItem[] = [
    {
      label: "注文ID",
      value: (
        <span className="trade-chat-detail__order-id">
          <span>{trade.orderId}</span>

          <button
            type="button"
            className="trade-chat-detail__copy-button"
            onClick={() => {
              void handleCopyOrderId();
            }}
            aria-label={orderIdCopied ? "コピーしました" : "注文IDをコピー"}
            title={orderIdCopied ? "コピーしました" : "注文IDをコピー"}
          >
            {orderIdCopied ? (
              <span>コピーしました</span>
            ) : (
              <Copy
                size={15}
                strokeWidth={2}
                aria-hidden="true"
              />
            )}
          </button>
        </span>
      ),
    },
  ];

  if (trade.returnRequestedAt) {
    transactionMetaItems.push({
      label: "返品申請日時",
      value: formatDateTime(trade.returnRequestedAt),
    });
  }

  if (trade.returnCompletedAt) {
    transactionMetaItems.push({
      label: "返品完了日時",
      value: formatDateTime(trade.returnCompletedAt),
    });
  }

  if (trade.transferredAt) {
    transactionMetaItems.push({
      label: "受取日時",
      value: formatDateTime(trade.transferredAt),
    });
  }

  const productMetaItems: ChatMetaItem[] = [
    {
      label: "商品の状態",
      value: trade.resale.condition,
    },
  ];

  if (model.modelNumber) {
    productMetaItems.push({
      label: "モデル番号",
      value: model.modelNumber,
    });
  }

  if (model.kindLabel && model.kindLabel !== "アパレル") {
    productMetaItems.push({
      label: "種別",
      value: model.kindLabel,
    });
  }

  if (model.size) {
    productMetaItems.push({
      label: "サイズ",
      value: model.size,
    });
  }

  if (model.colorLabel || model.colorCssValue) {
    productMetaItems.push({
      label: "カラー",
      value: model.colorLabel || model.colorCssValue,
    });
  }

  if (model.measurementsLabel !== "-") {
    productMetaItems.push({
      label: "採寸",
      value: model.measurementsLabel,
    });
  }

  if (model.volumeLabel !== "-") {
    productMetaItems.push({
      label: "容量",
      value: model.volumeLabel,
    });
  }

  return (
    <ChatThreadCard variant="trade">
      <ChatMessageHeader
        name={displayName}
        icon={counterpartAvatarIcon}
        createdAt={trade.createdAt}
        action={
          <span className="chat-detail-page__status">
            {getTradeStatusLabel(trade)}
          </span>
        }
      />

      <h2 className="chat-detail-page__subject">
        {title}
      </h2>

      <ChatMetaSection
        title="取引情報"
        items={transactionMetaItems}
      />

      <ChatMetaSection
        title="商品情報"
        items={productMetaItems}
      />

      {trade.resale.description ? (
        <section className="chat-detail-page__product-meta">
          <h3 className="chat-detail-page__product-meta-title">
            商品説明
          </h3>

          <p className="chat-detail-page__content">
            {trade.resale.description}
          </p>
        </section>
      ) : null}

      <ChatImageGrid
        images={trade.resale.images.map((image) => ({
          key: image.id,
          url: image.url,
          alt: "商品状態",
        }))}
      />
    </ChatThreadCard>
  );
}

function TradeMessageCard({
  message,
  trade,
}: {
  message: TradeMessage;
  trade: TradeDetail;
}) {
  const isSystem = message.senderSide === "system";
  const isMine =
    !isSystem &&
    message.senderSide === trade.viewerSide;

  const sender = getTradeMessageSenderDisplay(
    message,
    trade,
  );

  return (
    <ChatMessageBubble
      senderName={sender.name}
      senderIcon={sender.icon}
      createdAt={message.createdAt}
      content={message.content}
      isMine={isMine}
      isSystem={isSystem}
    />
  );
}