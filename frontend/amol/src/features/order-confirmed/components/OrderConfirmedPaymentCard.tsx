//frontend\amol\src\features\order-confirmed\components\OrderConfirmedPaymentCard.tsx
import { formatYen } from "../../cart/utils/cartUtils";

type OrderConfirmedPaymentCardProps = {
  statusLabel: string;
  amount: number;
  invoiceId: string;
  paymentId: string;
};

export function OrderConfirmedPaymentCard({
  statusLabel,
  amount,
  invoiceId,
  paymentId,
}: OrderConfirmedPaymentCardProps) {
  return (
    <section className="order-confirmed-page__card">
      <h2 className="order-confirmed-page__card-title">決済情報</h2>

      <dl className="order-confirmed-page__details">
        <div className="order-confirmed-page__detail-row">
          <dt>ステータス</dt>
          <dd>{statusLabel}</dd>
        </div>

        <div className="order-confirmed-page__detail-row">
          <dt>金額</dt>
          <dd>{formatYen(amount)}</dd>
        </div>

        {invoiceId ? (
          <div className="order-confirmed-page__detail-row">
            <dt>請求ID</dt>
            <dd>{invoiceId}</dd>
          </div>
        ) : null}

        {paymentId ? (
          <div className="order-confirmed-page__detail-row">
            <dt>決済ID</dt>
            <dd>{paymentId}</dd>
          </div>
        ) : null}
      </dl>
    </section>
  );
}