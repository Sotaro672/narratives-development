// frontend/mall/src/features/order-confirmed/components/OrderConfirmedHero.tsx

import Card from "../../../components/ui/Card";

export function OrderConfirmedHero() {
  return (
    <Card
      as="section"
      variant="panel"
      className="order-confirmed-page__hero"
    >
      <div className="order-confirmed-page__check" aria-hidden="true">
        ✓
      </div>

      <h1 className="order-confirmed-page__title">
        注文受付が完了しました
      </h1>

      <p className="order-confirmed-page__description">
        ご注文ありがとうございます。お支払いは商品の発送時に行われます。
      </p>
    </Card>
  );
}