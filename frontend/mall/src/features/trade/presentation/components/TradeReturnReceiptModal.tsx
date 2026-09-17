// frontend/mall/src/features/trade/presentation/components/TradeReturnReceiptModal.tsx

import { createPortal } from "react-dom";

import Button from "../../../../components/ui/Button";

export type TradeReturnReceiptModalProps = {
  open: boolean;
  merchandiseRefundAmount: number | "";
  merchandiseRefundMaxAmount: number;
  refundOutboundShipping: boolean;
  coverReturnShipping: boolean;
  error?: string | null;
  submitting: boolean;
  selectionLocked: boolean;
  canSubmit: boolean;
  onMerchandiseRefundAmountChange: (value: string | number) => void;
  onRefundOutboundShippingChange: (value: boolean) => void;
  onCoverReturnShippingChange: (value: boolean) => void;
  onCancel: () => void;
  onSubmit: () => void;
};

function formatCurrency(value: number): string {
  if (!Number.isFinite(value) || value < 0) {
    return "-";
  }

  return `${value.toLocaleString("ja-JP")}円`;
}

export default function TradeReturnReceiptModal({
  open,
  merchandiseRefundAmount,
  merchandiseRefundMaxAmount,
  refundOutboundShipping,
  coverReturnShipping,
  error,
  submitting,
  selectionLocked,
  canSubmit,
  onMerchandiseRefundAmountChange,
  onRefundOutboundShippingChange,
  onCoverReturnShippingChange,
  onCancel,
  onSubmit,
}: TradeReturnReceiptModalProps) {
  if (!open || typeof document === "undefined") {
    return null;
  }

  const inputDisabled = submitting || selectionLocked;

  const handleSubmit = (): void => {
    if (!canSubmit || submitting) {
      return;
    }

    onSubmit();
  };

  return createPortal(
    <div
      className="order-detail-page__return-modal-backdrop"
      role="presentation"
    >
      <div
        className="order-detail-page__return-modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby="trade-return-receipt-modal-title"
      >
        <div className="order-detail-page__return-modal-header">
          <h2 id="trade-return-receipt-modal-title">
            返品受領・返金
          </h2>

          <button
            type="button"
            className="order-detail-page__return-modal-close"
            onClick={onCancel}
            disabled={submitting}
            aria-label="閉じる"
          >
            ×
          </button>
        </div>

        <div className="order-detail-page__return-modal-notice">
          <h3 className="order-detail-page__return-modal-notice-title">
            返金内容を確認してください
          </h3>

          <p>
            返品商品を受領したことを確認したうえで、購入者へ返金する商品代金と送料条件を指定してください。
          </p>
        </div>

        <label
          className="order-detail-page__return-modal-field"
          htmlFor="trade-return-receipt-refund-amount"
        >
          <span className="order-detail-page__return-modal-label">
            商品代金の返金額（税込）

            <span
              className="order-detail-page__return-modal-required"
              aria-hidden="true"
            >
              *
            </span>
          </span>

          <div className="order-detail-page__return-modal-refund-amount-row">
            <input
              id="trade-return-receipt-refund-amount"
              type="number"
              inputMode="numeric"
              min={1}
              max={
                merchandiseRefundMaxAmount > 0
                  ? merchandiseRefundMaxAmount
                  : undefined
              }
              step={1}
              value={merchandiseRefundAmount}
              onChange={(event) => {
                onMerchandiseRefundAmountChange(event.target.value);
              }}
              disabled={inputDisabled}
              className="order-detail-page__return-modal-input"
              aria-describedby="trade-return-receipt-refund-limit"
            />

            <span className="order-detail-page__return-modal-refund-amount-unit">
              円
            </span>
          </div>

          <span
            id="trade-return-receipt-refund-limit"
            className="order-detail-page__return-modal-help"
          >
            返金上限: {formatCurrency(merchandiseRefundMaxAmount)}
          </span>

          <span className="order-detail-page__return-modal-help">
            1円以上、商品代金（税込）の返金上限以内で指定してください。
          </span>
        </label>

        <fieldset
          className="order-detail-page__return-modal-package-options"
          disabled={inputDisabled}
        >
          <legend className="order-detail-page__return-modal-label">
            送料の返金・負担
          </legend>

          <label className="order-detail-page__return-modal-package-option">
            <input
              type="checkbox"
              checked={refundOutboundShipping}
              onChange={(event) => {
                onRefundOutboundShippingChange(event.target.checked);
              }}
              disabled={inputDisabled}
              className="order-detail-page__return-modal-agreement-checkbox"
            />

            <span>
              <strong>
                購入時の配送料も返金する
              </strong>

              <span className="order-detail-page__return-modal-option-description">
                購入者が支払った往路の配送料とその消費税を、購入者への返金額に含めます。
              </span>
            </span>
          </label>

          <label className="order-detail-page__return-modal-package-option">
            <input
              type="checkbox"
              checked={coverReturnShipping}
              onChange={(event) => {
                onCoverReturnShippingChange(event.target.checked);
              }}
              disabled={inputDisabled}
              className="order-detail-page__return-modal-agreement-checkbox"
            />

            <span>
              <strong>
                返品時の配送料を出品者側が負担する
              </strong>

              <span className="order-detail-page__return-modal-option-description">
                復路の配送料を出品者側の負担として計上します。購入者へのStripe返金額には加算されません。
              </span>
            </span>
          </label>
        </fieldset>

        {selectionLocked ? (
          <div className="order-detail-page__return-modal-notice">
            <p>
              返金処理を開始済みのため、返金額と送料条件は変更できません。金融処理が未完了の場合は、同じ条件で再実行できます。
            </p>
          </div>
        ) : null}

        {error ? (
          <div
            className="order-detail-page__return-modal-error"
            role="alert"
          >
            {error}
          </div>
        ) : null}

        <div className="order-detail-page__return-modal-actions">
          <Button
            variant="primary"
            size="md"
            className="order-detail-page__return-modal-action"
            onClick={handleSubmit}
            disabled={!canSubmit || submitting}
          >
            {submitting
              ? "返品処理中..."
              : selectionLocked
                ? "同じ条件で再実行する"
                : "返品受領・返金"}
          </Button>
        </div>
      </div>
    </div>,
    document.body,
  );
}