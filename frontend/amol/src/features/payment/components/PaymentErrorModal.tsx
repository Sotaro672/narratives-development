//frontend\amol\src\features\payment\components\PaymentErrorModal.tsx
type PaymentErrorModalProps = {
  message: string;
  onClose: () => void;
};

export function PaymentErrorModal({ message, onClose }: PaymentErrorModalProps) {
  if (!message) {
    return null;
  }

  return (
    <div
      className="payment-page__modal-backdrop"
      role="presentation"
      onClick={onClose}
    >
      <div
        className="payment-page__modal"
        role="alertdialog"
        aria-modal="true"
        aria-labelledby="payment-error-modal-title"
        onClick={(event) => event.stopPropagation()}
      >
        <h2 id="payment-error-modal-title" className="payment-page__modal-title">
          注文または決済処理に失敗しました
        </h2>

        <p className="payment-page__modal-message">{message}</p>

        <button
          type="button"
          className="payment-page__primary-button"
          onClick={onClose}
        >
          閉じる
        </button>
      </div>
    </div>
  );
}