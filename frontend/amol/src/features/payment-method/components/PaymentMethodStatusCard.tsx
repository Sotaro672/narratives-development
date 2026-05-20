//frontend\amol\src\features\payment-method\components\PaymentMethodStatusCard.tsx
import type { CardPaymentMethod } from "../types";
import { cardBrandLabel } from "../utils/paymentMethodUtils";

type PaymentMethodStatusCardProps = {
  isLoading: boolean;
  paymentMethod: CardPaymentMethod | null;
};

export default function PaymentMethodStatusCard(
  props: PaymentMethodStatusCardProps,
) {
  const { isLoading, paymentMethod } = props;

  return (
    <div className="payment-method-page-card">
      <h2 className="payment-method-page-card__title">登録状況</h2>

      {isLoading ? (
        <p className="payment-method-page-card__text">読み込み中...</p>
      ) : paymentMethod ? (
        <div className="payment-method-page-card__details">
          <p className="payment-method-page-card__text">
            <strong>ブランド:</strong> {cardBrandLabel(paymentMethod.brand)}
          </p>
          <p className="payment-method-page-card__text">
            <strong>下4桁:</strong> {paymentMethod.last4 || "-"}
          </p>
          <p className="payment-method-page-card__text">
            <strong>有効期限:</strong> {paymentMethod.expMonth}/
            {paymentMethod.expYear}
          </p>
          <p className="payment-method-page-card__text">
            <strong>カード名義人:</strong>{" "}
            {paymentMethod.cardholderName || "-"}
          </p>
          <p className="payment-method-page-card__text">
            <strong>既定カード:</strong>{" "}
            {paymentMethod.isDefault ? "はい" : "いいえ"}
          </p>
        </div>
      ) : (
        <div className="payment-method-page-card__details">
          <p className="payment-method-page-card__text">
            登録済みの支払方法はありません。
          </p>
        </div>
      )}
    </div>
  );
}