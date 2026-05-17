//frontend\amol\src\features\order-confirmed\components\OrderConfirmedHero.tsx
export function OrderConfirmedHero() {
  return (
    <div className="order-confirmed-page__hero">
      <div className="order-confirmed-page__check" aria-hidden="true">
        ✓
      </div>

      <h1 className="order-confirmed-page__title">注文が完了しました</h1>

      <p className="order-confirmed-page__description">
        ご注文ありがとうございます。決済が正常に完了しました。
      </p>
    </div>
  );
}