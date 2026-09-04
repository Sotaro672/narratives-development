//frontend\amol\src\features\order-confirmed\components\OrderConfirmedShippingCard.tsx
type OrderConfirmedShippingCardProps = {
  lines: string[];
};

export function OrderConfirmedShippingCard({
  lines,
}: OrderConfirmedShippingCardProps) {
  return (
    <section className="order-confirmed-page__card">
      <h2 className="order-confirmed-page__card-title">配送先情報</h2>

      {lines.length > 0 ? (
        <div className="order-confirmed-page__shipping-address">
          {lines.map((line) => (
            <p className="order-confirmed-page__shipping-address-line" key={line}>
              {line}
            </p>
          ))}
        </div>
      ) : (
        <p className="order-confirmed-page__empty">
          配送先情報を取得できませんでした。
        </p>
      )}
    </section>
  );
}