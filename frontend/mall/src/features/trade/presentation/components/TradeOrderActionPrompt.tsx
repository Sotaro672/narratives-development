// frontend/mall/src/features/trade/presentation/components/TradeOrderActionPrompt.tsx

import type { TradeOrderActionKind } from "../util/tradeChatDetail";

type TradeOrderActionPromptProps = {
  action: TradeOrderActionKind;
  processing: boolean;
  error?: string | null;
  onAction: () => void;
};

function getPromptText(action: TradeOrderActionKind): string {
  switch (action) {
    case "cancel":
      return "注文をキャンセルしますか？";

    case "dispatch":
      return "商品を発送しますか？";

    case "start-return-consultation":
      return "返品について相談しますか？";

    case "respond-return-consultation":
      return "購入者から返品についての相談が届いています。";

    case "review-return-proposal":
      return "出品者から返品条件が提示されています。";

    case "prepare-return-shipment":
      return "返品用QRを表示しますか？";

    case "receive-return":
      return "返品商品の受領確認と返金処理を進めますか？";
  }
}

function getActionLabel(
  action: TradeOrderActionKind,
  processing: boolean,
): string {
  if (processing) {
    switch (action) {
      case "cancel":
        return "キャンセル中...";

      case "dispatch":
        return "発送処理中...";

      case "start-return-consultation":
        return "送信中...";

      case "respond-return-consultation":
        return "回答中...";

      case "review-return-proposal":
        return "処理中...";

      case "prepare-return-shipment":
        return "QR準備中...";

      case "receive-return":
        return "受領・返金処理中...";
    }
  }

  switch (action) {
    case "cancel":
      return "注文をキャンセル";

    case "dispatch":
      return "発送する";

    case "start-return-consultation":
      return "返品について相談する";

    case "respond-return-consultation":
      return "返品相談に回答する";

    case "review-return-proposal":
      return "返品条件を確認する";

    case "prepare-return-shipment":
      return "返品用QRを表示する";

    case "receive-return":
      return "返品受領・返金を進める";
  }
}

export default function TradeOrderActionPrompt({
  action,
  processing,
  error,
  onAction,
}: TradeOrderActionPromptProps) {
  return (
    <article className="chat-detail-page__reply chat-detail-page__reply--system">
      <div className="chat-detail-page__message-head">
        <div>
          <span className="chat-detail-page__sender">
            システム
          </span>
        </div>
      </div>

      <p className="chat-detail-page__content">
        {getPromptText(action)}
      </p>

      {error ? (
        <div
          className="chat-detail-page__modal-error"
          role="alert"
        >
          {error}
        </div>
      ) : null}

      <div className="chat-detail-page__close-prompt-actions">
        <button
          type="button"
          onClick={onAction}
          disabled={processing}
        >
          {getActionLabel(action, processing)}
        </button>
      </div>
    </article>
  );
}