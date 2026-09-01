// frontend/amol/src/features/trade/presentation/components/TradeChatDetail.tsx

import { useCallback, useEffect, useMemo, useState } from "react";
import { createPortal } from "react-dom";
import { useNavigate } from "react-router-dom";

import Layout from "../../../../components/layout/Layout";
import { formatDateTime } from "../../../../components/utils/date";

import {
  cancelTradeOrderItem,
  createTradeMessage,
  fetchTradeById,
  markTradeMessagesAsRead,
} from "../../infrastructure/tradeApi";

import type {
  TradeDetail,
  TradeMessage,
} from "../../../shared/types/trade";

import TradeOrderActionPrompt from "./TradeOrderActionPrompt";

const MESSAGE_LIMIT = 100;

type TradeChatDetailProps = {
  tradeId: string;
  onBack: () => void;
};

function getErrorMessage(
  caught: unknown,
  fallbackMessage: string,
): string {
  if (caught instanceof Error && caught.message) {
    return caught.message;
  }

  return fallbackMessage;
}

function sortMessages(
  messages: TradeMessage[],
): TradeMessage[] {
  return [...messages].sort(
    (firstMessage, secondMessage) =>
      new Date(firstMessage.createdAt).getTime() -
      new Date(secondMessage.createdAt).getTime(),
  );
}

function getTradeStatusLabel(
  status: TradeDetail["status"],
): string {
  switch (status) {
    case "active":
      return "取引中";
    case "closed":
      return "取引終了";
    default:
      return "";
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
          "取引チャットの取得に失敗しました。",
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
    if (!isReplyModalOpen) {
      return;
    }

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

  const canSubmitReply = /\S/u.test(replyContent);

  const shouldShowOrderAction =
    !!trade &&
    trade.status === "active" &&
    !trade.isCancelled &&
    !trade.isDispatched;

  const replyActionDisabled =
    loading ||
    !trade ||
    trade.status !== "active" ||
    postingReply ||
    orderActionProcessing;

  const openReplyModal = useCallback(() => {
    if (replyActionDisabled) {
      return;
    }

    setReplyError("");
    setIsReplyModalOpen(true);
  }, [replyActionDisabled]);

  const closeReplyModal = useCallback(() => {
    if (postingReply) {
      return;
    }

    setIsReplyModalOpen(false);
    setReplyContent("");
    setReplyError("");
  }, [postingReply]);

  const submitReply = useCallback(async (): Promise<void> => {
    if (
      postingReply ||
      orderActionProcessing ||
      !trade ||
      trade.status !== "active" ||
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
        if (!currentTrade) {
          return currentTrade;
        }

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
    trade,
  ]);

  const handleOrderAction = useCallback(async (): Promise<void> => {
    if (
      orderActionProcessing ||
      !trade ||
      trade.status !== "active" ||
      trade.isCancelled ||
      trade.isDispatched ||
      !normalizedTradeId
    ) {
      return;
    }

    setOrderActionError("");

    if (trade.viewerSide === "seller") {
      navigate(
        `/dispatch/trades/${encodeURIComponent(normalizedTradeId)}`,
      );
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
  }, [
    loadThread,
    navigate,
    normalizedTradeId,
    orderActionProcessing,
    trade,
  ]);

  return (
    <>
      <Layout
        title="取引チャット"
        showBackButton
        onBackButtonClick={onBack}
        showFooter={!isReplyModalOpen}
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
              取引チャットが見つかりません。
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
                        viewerSide={trade.viewerSide}
                      />
                    ))}

                    {shouldShowOrderAction ? (
                      <TradeOrderActionPrompt
                        viewerSide={trade.viewerSide}
                        processing={orderActionProcessing}
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

  const counterpartAvatarId =
    trade.viewerSide === "buyer"
      ? trade.sellerAvatarId
      : trade.buyerAvatarId;

  return (
    <article className="chat-detail-page__inquiry">
      <div className="chat-detail-page__message-head">
        <div className="chat-detail-page__sender-profile">
          <div
            className="chat-detail-page__sender-icon"
            aria-hidden="true"
          >
            <span>{counterpartLabel.slice(0, 1)}</span>
          </div>

          <div>
            <span className="chat-detail-page__sender">
              {counterpartLabel}
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
          {getTradeStatusLabel(trade.status)}
        </span>
      </div>

      <h2 className="chat-detail-page__subject">
        取引チャット
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
            <dd>{counterpartAvatarId}</dd>
          </div>
        </dl>
      </section>
    </article>
  );
}

function TradeMessageCard({
  message,
  viewerSide,
}: {
  message: TradeMessage;
  viewerSide: TradeDetail["viewerSide"];
}) {
  const isSystem = message.senderSide === "system";
  const isMine =
    !isSystem &&
    message.senderSide === viewerSide;

  const senderLabel = isSystem
    ? "システム"
    : isMine
      ? "あなた"
      : "相手";

  const className = isMine
    ? "chat-detail-page__reply chat-detail-page__reply--avatar"
    : "chat-detail-page__reply";

  return (
    <article className={className}>
      <div className="chat-detail-page__message-head">
        <div className="chat-detail-page__sender-profile">
          <div
            className="chat-detail-page__sender-icon"
            aria-hidden="true"
          >
            <span>{senderLabel.slice(0, 1)}</span>
          </div>

          <div>
            <span className="chat-detail-page__sender">
              {senderLabel}
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
  if (
    !open ||
    typeof document === "undefined"
  ) {
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
            disabled={
              !canSubmit ||
              submitting
            }
          >
            {submitting
              ? "送信中..."
              : "送信"}
          </button>
        </div>
      </div>
    </div>,
    document.body,
  );
}