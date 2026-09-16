// frontend/amol/src/features/trade/presentation/components/TradeOrderActionPrompt.tsx

export type TradeOrderAction =
  | "cancel"
  | "dispatch"
  | "return"
  | "receive-return";

type TradeOrderActionPromptProps = {
  action: TradeOrderAction;
  processing: boolean;
  error?: string | null;
  onAction: () => void;
};

function getPromptText(action: TradeOrderAction): string {
  switch (action) {
    case "cancel":
      return "注文をキャンセルしますか？";
    case "dispatch":
      return "商品を発送しますか？";
    case "return":
      return "商品の返品を申請しますか？";
    case "receive-return":
      return "返品商品を受領しましたか？";
  }
}

function getActionLabel(
  action: TradeOrderAction,
  processing: boolean,
): string {
  if (processing) {
    switch (action) {
      case "cancel":
        return "キャンセル中...";
      case "dispatch":
        return "発送処理中...";
      case "return":
        return "返品申請中...";
      case "receive-return":
        return "返品処理中...";
    }
  }

  switch (action) {
    case "cancel":
      return "注文をキャンセル";
    case "dispatch":
      return "発送する";
    case "return":
      return "返品を申請";
    case "receive-return":
      return "返品受領・返金";
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
          <span className="chat-detail-page__sender">システム</span>
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