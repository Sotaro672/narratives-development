// frontend/mall/src/features/order-confirmed/components/OrderConfirmedPaymentCard.tsx

import Card from "../../../components/ui/Card";
import InfoList from "../../../components/ui/InfoList";
import { formatPrice } from "../../../components/utils/price";

type OrderConfirmedPaymentCardProps = {
  statusLabel: string;
  amount: number;
  orderId: string;
};

export function OrderConfirmedPaymentCard({
  statusLabel,
  amount,
  orderId,
}: OrderConfirmedPaymentCardProps) {
  const rows = [
    {
      key: "status",
      label: "ステータス",
      value: statusLabel,
    },
    {
      key: "amount",
      label: "金額",
      value: formatPrice(amount),
    },
    ...(orderId
      ? [
          {
            key: "orderId",
            label: "注文ID",
            value: orderId,
          },
        ]
      : []),
  ];

  return (
    <Card as="section" variant="panel">
      <h2 className="order-confirmed-page__card-title">
        決済情報
      </h2>

      <InfoList rows={rows} />
    </Card>
  );
}