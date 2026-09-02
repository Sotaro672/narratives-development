// frontend/amol/src/features/trade/presentation/components/TradeChatDetail.tsx

import { useCallback, useEffect, useMemo, useState } from "react";
import { createPortal } from "react-dom";
import { useNavigate } from "react-router-dom";

import Layout from "../../../../components/layout/Layout";
import { formatDateTime } from "../../../../components/utils/date";
import { getApiBaseUrl } from "../../../../lib/apiBaseUrl";
import { getFirebaseIdToken } from "../../../../lib/authToken";

import { returnOrderItem } from "../../../order/api/orderDetailApi";
import ReturnRequestModal, { type ReturnPackageState } from "../../../order/components/ReturnRequestModal";
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

function getInitial(value: string): string {
  return Array.from(value)[0] ?? "？";
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

  const [orderActionProcessing, setOrderActionProcessing] = useState(false);
  const [orderActionError, setOrderActionError] = useState("");

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

  useEffect(() => {
    if (!isReplyModalOpen) return;

    const previousOverflow = document.body.style.overflow;
    const previousTouchAction = document.body.style.touchAction;

    document.body.style.overflow = "hidden";
    document.body.style.touchAction = "none";

    return () => {
      document.body.style.overflow = previousOverflow;
      document.body.style.touchAction = previousTouchAction;
    };
  }, [isReplyModalOpen]);

  const sortedMessages = useMemo(
    () => sortMessages(trade?.messages ?? []),
    [trade?.messages],
  );

  const title = getTradeTitle(trade?.productName);
  const canSubmitReply = /\S/u.test(replyContent);
  const orderAction = getTradeOrderAction(trade);
  const shouldShowOrderAction = orderAction !== null;

  const replyActionDisabled =
    loading ||
    !trade ||
    trade.status !== "active" ||
    trade.isCancelled ||
    postingReply ||
    orderActionProcessing ||
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
      orderActionProcessing ||
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
    normalizedTradeId,
    orderActionProcessing,
    postingReply,
    replyContent,
    returning,
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
      trade.isReturnCompleted
    ) {
      return;
    }

    setReturnPackageState(null);
    setReturnReason("");
    setReturnError("");
    setIsReturnModalOpen(true);
  }, [trade]);

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
      !trade ||
      trade.viewerSide !== "buyer" ||
      trade.status !== "active" ||
      trade.isCancelled ||
      !trade.isDispatched ||
      trade.transferred ||
      trade.isReturnRequested ||
      trade.isReturnCompleted
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
    loadThread,
    returnPackageState,
    returnReason,
    returning,
    trade,
  ]);

  const handleOrderAction = useCallback(async (): Promise<void> => {
    if (
      orderActionProcessing ||
      returning ||
      !trade ||
      !orderAction ||
      trade.status !== "active" ||
      trade.isCancelled ||
      !normalizedTradeId
    ) {
      return;
    }

    setOrderActionError("");

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
        if (
          trade.viewerSide !== "buyer" ||
          trade.isDispatched
        ) {
          return;
        }

        setOrderActionProcessing(true);

        try {
          await cancelTradeOrderItem({
            orderId: trade.orderId,
            orderItemIndex: trade.orderItemIndex,
          });

          await loadThread();
        } catch (caught) {
          setOrderActionError(
            getErrorMessage(
              caught,
              "注文のキャンセルに失敗しました。",
            ),
          );
        } finally {
          setOrderActionProcessing(false);
        }

        return;
    }
  }, [
    loadThread,
    navigate,
    normalizedTradeId,
    openReturnModal,
    orderAction,
    orderActionProcessing,
    returning,
    trade,
  ]);

  return (
    <>
      <Layout
        title={title}
        showBackButton
        onBackButtonClick={onBack}
        showFooter={!isReplyModalOpen && !isReturnModalOpen}
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
                        processing={orderActionProcessing || returning}
                        error={orderActionError}
                        onAction={() => {
                          void handleOrderAction();
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

      <TradeMessageModal
        open={isReplyModalOpen}
        content={replyContent}
        error={replyError}
        submitting={postingReply}
        canSubmit={canSubmitReply}
        onContentChange={setReplyContent}
        onCancel={closeReplyModal}
        onSubmit={() => {
          void submitReply();
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
  const initial = getInitial(displayName);

  return (
    <article className="chat-detail-page__inquiry">
      <div className="chat-detail-page__message-head">
        <div className="chat-detail-page__sender-profile">
          <div
            className="chat-detail-page__sender-icon"
            aria-hidden="true"
          >
            {counterpartAvatarIcon ? (
              <img
                src={counterpartAvatarIcon}
                alt=""
                className="chat-detail-page__sender-icon-image"
              />
            ) : (
              <span>{initial}</span>
            )}
          </div>

          <div>
            <span className="chat-detail-page__sender">
              {displayName}
            </span>

            {trade.createdAt ? (
              <time
                className="chat-detail-page__date"
                dateTime={trade.createdAt}
              >
                {formatDateTime(trade.createdAt)}
              </time>
            ) : null}
          </div>
        </div>

        <span className="chat-detail-page__status">
          {getTradeStatusLabel(trade)}
        </span>
      </div>

      <h2 className="chat-detail-page__subject">
        {title}
      </h2>

      <section className="chat-detail-page__product-meta">
        <h3 className="chat-detail-page__product-meta-title">
          取引情報
        </h3>

        <dl className="chat-detail-page__product-meta-list">
          <div className="chat-detail-page__product-meta-row">
            <dt>注文ID</dt>
            <dd>{trade.orderId}</dd>
          </div>

          <div className="chat-detail-page__product-meta-row">
            <dt>商品番号</dt>
            <dd>{trade.orderItemIndex + 1}</dd>
          </div>

          <div className="chat-detail-page__product-meta-row">
            <dt>{counterpartLabel}</dt>
            <dd>{displayName}</dd>
          </div>

          {trade.returnRequestedAt ? (
            <div className="chat-detail-page__product-meta-row">
              <dt>返品申請日時</dt>
              <dd>{formatDateTime(trade.returnRequestedAt)}</dd>
            </div>
          ) : null}

          {trade.returnCompletedAt ? (
            <div className="chat-detail-page__product-meta-row">
              <dt>返品完了日時</dt>
              <dd>{formatDateTime(trade.returnCompletedAt)}</dd>
            </div>
          ) : null}

          {trade.transferredAt ? (
            <div className="chat-detail-page__product-meta-row">
              <dt>受取日時</dt>
              <dd>{formatDateTime(trade.transferredAt)}</dd>
            </div>
          ) : null}
        </dl>
      </section>
    </article>
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

  const senderInitial = getInitial(sender.name);

  const className = isMine
    ? "chat-detail-page__reply chat-detail-page__reply--avatar"
    : isSystem
      ? "chat-detail-page__reply chat-detail-page__reply--system"
      : "chat-detail-page__reply";

  return (
    <article className={className}>
      <div className="chat-detail-page__message-head">
        <div className="chat-detail-page__sender-profile">
          {!isSystem ? (
            <div
              className="chat-detail-page__sender-icon"
              aria-hidden="true"
            >
              {sender.icon ? (
                <img
                  src={sender.icon}
                  alt=""
                  className="chat-detail-page__sender-icon-image"
                />
              ) : (
                <span>{senderInitial}</span>
              )}
            </div>
          ) : null}

          <div>
            <span className="chat-detail-page__sender">
              {sender.name}
            </span>

            {message.createdAt ? (
              <time
                className="chat-detail-page__date"
                dateTime={message.createdAt}
              >
                {formatDateTime(message.createdAt)}
              </time>
            ) : null}
          </div>
        </div>
      </div>

      {message.content ? (
        <p className="chat-detail-page__content">
          {message.content}
        </p>
      ) : null}
    </article>
  );
}

function TradeMessageModal({
  open,
  content,
  error,
  submitting,
  canSubmit,
  onContentChange,
  onCancel,
  onSubmit,
}: {
  open: boolean;
  content: string;
  error: string;
  submitting: boolean;
  canSubmit: boolean;
  onContentChange: (value: string) => void;
  onCancel: () => void;
  onSubmit: () => void;
}) {
  if (!open || typeof document === "undefined") {
    return null;
  }

  return createPortal(
    <div className="chat-detail-page__modal-backdrop">
      <div
        className="chat-detail-page__modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby="trade-chat-message-modal-title"
      >
        <div className="chat-detail-page__modal-header">
          <h2 id="trade-chat-message-modal-title">
            返信する
          </h2>

          <button
            type="button"
            className="chat-detail-page__modal-close"
            onClick={onCancel}
            disabled={submitting}
            aria-label="閉じる"
          >
            ×
          </button>
        </div>

        <textarea
          className="chat-detail-page__reply-input"
          value={content}
          onChange={(event) => {
            onContentChange(event.target.value);
          }}
          placeholder="メッセージを入力"
          rows={6}
          maxLength={500}
          disabled={submitting}
        />

        {error ? (
          <div
            className="chat-detail-page__modal-error"
            role="alert"
          >
            {error}
          </div>
        ) : null}

        <div className="chat-detail-page__modal-actions">
          <button
            type="button"
            onClick={onCancel}
            disabled={submitting}
          >
            キャンセル
          </button>

          <button
            type="button"
            onClick={onSubmit}
            disabled={!canSubmit || submitting}
          >
            {submitting ? "送信中..." : "送信"}
          </button>
        </div>
      </div>
    </div>,
    document.body,
  );
}