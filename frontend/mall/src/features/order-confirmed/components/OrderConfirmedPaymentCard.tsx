// frontend/mall/src/features/order-confirmed/components/OrderConfirmedPaymentCard.tsx

import InfoList from "../../../components/ui/InfoList";
import { formatPrice } from "../../../components/utils/price";

type OrderConfirmedPaymentCardProps = {
  statusLabel: string;
  subtotalAmount: number;
  shippingAmount: number;
  taxAmount: number;
  amount: number;
  orderId: string;
};

export function OrderConfirmedPaymentCard({
  statusLabel,
  subtotalAmount,
  shippingAmount,
  taxAmount,
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
      key: "subtotal",
      label: "商品小計",
      value: formatPrice(subtotalAmount),
    },
    {
      key: "shipping",
      label: "配送料",
      value: formatPrice(shippingAmount),
    },
    {
      key: "tax",
      label: "消費税",
      value: formatPrice(taxAmount),
    },
    {
      key: "amount",
      label: "合計金額",
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
    <section className="order-confirmed-page__content-section">
      <h2 className="order-confirmed-page__section-title">
        決済情報
      </h2>

      <InfoList rows={rows} />
    </section>
  );
}