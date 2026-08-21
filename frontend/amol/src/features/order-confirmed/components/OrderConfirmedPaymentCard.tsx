// frontend/amol/src/features/order-confirmed/components/OrderConfirmedPaymentCard.tsx

import {
  formatPrice,
} from "../../../components/utils/price";

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
  return (
    <section className="order-confirmed-page__card">
      <h2 className="order-confirmed-page__card-title">
        決済情報
      </h2>

      <dl className="order-confirmed-page__details">
        <div className="order-confirmed-page__detail-row">
          <dt>ステータス</dt>
          <dd>{statusLabel}</dd>
        </div>

        <div className="order-confirmed-page__detail-row">
          <dt>金額</dt>
          <dd>{formatPrice(amount)}</dd>
        </div>

        {orderId ? (
          <div className="order-confirmed-page__detail-row">
            <dt>注文ID</dt>
            <dd>{orderId}</dd>
          </div>
        ) : null}
      </dl>
    </section>
  );
}