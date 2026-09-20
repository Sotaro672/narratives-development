// frontend/mall/src/features/payment-method/components/PaymentMethodStatusCard.tsx

import Card from "../../../components/ui/Card";
import InfoList from "../../../components/ui/InfoList";
import TextState from "../../../components/ui/TextState";
import type { CardPaymentMethod } from "../../shared/types/paymentMethods";
import { cardBrandLabel } from "../utils/paymentMethodUtils";

type PaymentMethodStatusCardProps = {
  isLoading: boolean;
  paymentMethod: CardPaymentMethod | null;
};

export default function PaymentMethodStatusCard({
  isLoading,
  paymentMethod,
}: PaymentMethodStatusCardProps) {
  return (
    <Card variant="panel">
      <h2 className="payment-method-page-card__title">登録状況</h2>

      {isLoading ? (
        <TextState variant="loading">読み込み中...</TextState>
      ) : paymentMethod ? (
        <InfoList
          rows={[
            {
              key: "brand",
              label: "ブランド",
              value: cardBrandLabel(paymentMethod.brand),
            },
            {
              key: "last4",
              label: "下4桁",
              value: paymentMethod.last4 || "-",
            },
            {
              key: "expiry",
              label: "有効期限",
              value: `${paymentMethod.expMonth}/${paymentMethod.expYear}`,
            },
            {
              key: "cardholderName",
              label: "カード名義人",
              value: paymentMethod.cardholderName || "-",
            },
            {
              key: "default",
              label: "既定カード",
              value: paymentMethod.isDefault ? "はい" : "いいえ",
            },
          ]}
        />
      ) : (
        <TextState variant="empty">
          登録済みの支払方法はありません。
        </TextState>
      )}
    </Card>
  );
}