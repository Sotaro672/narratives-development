// frontend/amol/src/features/trade/presentation/components/TradeOrderActionPrompt.tsx

type TradeOrderActionPromptProps = {
  viewerSide: "buyer" | "seller";
  processing: boolean;
  error?: string | null;
  onAction: () => void;
};

export default function TradeOrderActionPrompt({
  viewerSide,
  processing,
  error,
  onAction,
}: TradeOrderActionPromptProps) {
  const isBuyer = viewerSide === "buyer";

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
        {isBuyer
          ? "注文をキャンセルしますか？"
          : "商品を発送しますか？"}
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
          {processing
            ? isBuyer
              ? "キャンセル中..."
              : "発送処理中..."
            : isBuyer
              ? "注文をキャンセル"
              : "発送する"}
        </button>
      </div>
    </article>
  );
}