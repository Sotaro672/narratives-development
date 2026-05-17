//frontend\amol\src\features\payment\components\PaymentMethodsCard.tsx
import type { PaymentMethod } from "../types";
import {
  formatCardBrand,
  formatCardExpiry,
  formatCardholderName,
  formatCardLast4,
} from "../utils/format";

type PaymentMethodsCardProps = {
  paymentMethods: PaymentMethod[];
  selectedPaymentMethodId: string;
  onSelectPaymentMethod: (paymentMethodId: string) => void;
  onGoToPaymentMethod: () => void;
};

export function PaymentMethodsCard({
  paymentMethods,
  selectedPaymentMethodId,
  onSelectPaymentMethod,
  onGoToPaymentMethod,
}: PaymentMethodsCardProps) {
  return (
    <section className="payment-page__card">
      <div className="payment-page__section-header">
        <h2 className="payment-page__section-title">支払い方法</h2>
        <button
          type="button"
          className="payment-page__text-button"
          onClick={onGoToPaymentMethod}
        >
          カードを管理
        </button>
      </div>

      {paymentMethods.length > 0 ? (
        <div className="payment-page__payment-methods">
          {paymentMethods.map((method) => {
            return (
              <label className="payment-page__payment-method" key={method.id}>
                <input
                  type="radio"
                  name="paymentMethod"
                  value={method.id}
                  checked={selectedPaymentMethodId === method.id}
                  onChange={() => onSelectPaymentMethod(method.id)}
                />

                <span className="payment-page__payment-method-body">
                  <span className="payment-page__payment-method-row">
                    <span className="payment-page__payment-method-label">
                      ブランド
                    </span>
                    <span className="payment-page__payment-method-value">
                      {formatCardBrand(method.brand)}
                    </span>
                  </span>

                  <span className="payment-page__payment-method-row">
                    <span className="payment-page__payment-method-label">
                      口座名義
                    </span>
                    <span className="payment-page__payment-method-value">
                      {formatCardholderName(method)}
                    </span>
                  </span>

                  <span className="payment-page__payment-method-row">
                    <span className="payment-page__payment-method-label">
                      番号下4桁
                    </span>
                    <span className="payment-page__payment-method-value">
                      {formatCardLast4(method)}
                    </span>
                  </span>

                  <span className="payment-page__payment-method-row">
                    <span className="payment-page__payment-method-label">
                      有効期限
                    </span>
                    <span className="payment-page__payment-method-value">
                      {formatCardExpiry(method)}
                    </span>
                  </span>
                </span>
              </label>
            );
          })}
        </div>
      ) : (
        <div className="payment-page__empty-block">
          <p>登録済みの支払い方法がありません。</p>
          <button
            type="button"
            className="payment-page__primary-button"
            onClick={onGoToPaymentMethod}
          >
            支払い方法を登録する
          </button>
        </div>
      )}
    </section>
  );
}