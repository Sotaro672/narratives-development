// frontend/mall/src/features/payment/components/PaymentMethodsCard.tsx

import Button from "../../../components/ui/Button";
import Card from "../../../components/ui/Card";
import SectionHeader from "../../../components/ui/SectionHeader";
import TextButton from "../../../components/ui/TextButton";
import TextState from "../../../components/ui/TextState";
import type { CardPaymentMethod } from "../../shared/types/paymentMethods";
import {
  formatCardBrand,
  formatCardExpiry,
  formatCardholderName,
  formatCardLast4,
} from "../utils/format";

type PaymentMethodsCardProps = {
  paymentMethods: CardPaymentMethod[];
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
    <Card as="section" variant="panel" className="payment-page__card">
      <SectionHeader
        title="支払い方法"
        titleAs="h2"
        className="payment-page__section-header"
        right={
          <TextButton type="button" onClick={onGoToPaymentMethod}>
            カードを管理
          </TextButton>
        }
      />

      {paymentMethods.length > 0 ? (
        <div className="payment-page__payment-methods">
          {paymentMethods.map((method) => (
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
          ))}
        </div>
      ) : (
        <div className="payment-page__empty-block">
          <TextState variant="empty">
            登録済みの支払い方法がありません。
          </TextState>

          <Button
            type="button"
            variant="primary"
            fullWidth
            onClick={onGoToPaymentMethod}
          >
            支払い方法を登録する
          </Button>
        </div>
      )}
    </Card>
  );
}