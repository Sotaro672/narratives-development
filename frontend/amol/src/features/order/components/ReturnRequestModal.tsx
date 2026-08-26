// frontend/amol/src/features/order/components/ReturnRequestModal.tsx

import {
  useEffect,
  useState,
} from "react";
import { createPortal } from "react-dom";

import Button from "../../../components/ui/Button";

export type ReturnRequestModalProps = {
  open: boolean;
  reason: string;
  error?: string | null;
  submitting: boolean;
  onReasonChange: (value: string) => void;
  onCancel: () => void;
  onSubmit: () => void;
};

export default function ReturnRequestModal({
  open,
  reason,
  error,
  submitting,
  onReasonChange,
  onCancel,
  onSubmit,
}: ReturnRequestModalProps) {
  const [
    agreedToReturnConditions,
    setAgreedToReturnConditions,
  ] = useState(false);

  useEffect(() => {
    if (!open) {
      setAgreedToReturnConditions(false);
    }
  }, [
    open,
  ]);

  if (
    !open ||
    typeof document === "undefined"
  ) {
    return null;
  }

  const normalizedReason =
    reason.trim();

  const canSubmit =
    normalizedReason.length > 0 &&
    agreedToReturnConditions &&
    !submitting;

  const handleSubmit = () => {
    if (!canSubmit) {
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
        aria-labelledby="order-detail-return-modal-title"
        aria-describedby="order-detail-return-modal-conditions"
      >
        <div className="order-detail-page__return-modal-header">
          <h2 id="order-detail-return-modal-title">
            返品を申請する
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

        <div
          id="order-detail-return-modal-conditions"
          className="order-detail-page__return-modal-notice"
        >
          <h3 className="order-detail-page__return-modal-notice-title">
            返品条件
          </h3>

          <ol className="order-detail-page__return-modal-condition-list">
            <li>
              返品が承認された場合、返金対象は商品代金（税込）のみです。
            </li>

            <li>
              商品代金（税込）には、商品本体価格とその商品にかかる消費税が含まれます。
            </li>

            <li>
              ご購入時の配送料および配送料にかかる消費税は返金対象外です。
            </li>

            <li>
              返品商品の返送にかかる配送料はお客様のご負担となります。
            </li>

            <li>
              返品手続き中は、商品が入っている配送用梱包材を開けないでください。
            </li>
          </ol>
        </div>

        <label
          className="order-detail-page__return-modal-field"
          htmlFor="order-detail-return-reason"
        >
          <span className="order-detail-page__return-modal-label">
            返品理由

            <span
              className="order-detail-page__return-modal-required"
              aria-hidden="true"
            >
              *
            </span>
          </span>

          <textarea
            id="order-detail-return-reason"
            className="order-detail-page__return-modal-textarea"
            value={reason}
            onChange={(event) => {
              onReasonChange(
                event.target.value,
              );
            }}
            placeholder="返品理由を入力してください"
            rows={6}
            required
            disabled={submitting}
          />
        </label>

        <label className="order-detail-page__return-modal-agreement">
          <input
            type="checkbox"
            className="order-detail-page__return-modal-agreement-checkbox"
            checked={agreedToReturnConditions}
            onChange={(event) => {
              setAgreedToReturnConditions(
                event.target.checked,
              );
            }}
            disabled={submitting}
          />

          <span>
            返品条件に合意する
          </span>
        </label>

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
            variant="secondary"
            size="md"
            className="order-detail-page__return-modal-action order-detail-page__return-modal-cancel-action"
            onClick={onCancel}
            disabled={submitting}
          >
            キャンセル
          </Button>

          <Button
            variant="primary"
            size="md"
            className="order-detail-page__return-modal-action"
            onClick={handleSubmit}
            disabled={!canSubmit}
          >
            {submitting
              ? "申請中..."
              : "返品を申請する"}
          </Button>
        </div>
      </div>
    </div>,
    document.body,
  );
}