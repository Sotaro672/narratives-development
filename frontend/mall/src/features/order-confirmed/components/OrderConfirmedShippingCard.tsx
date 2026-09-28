// frontend/mall/src/features/order-confirmed/components/OrderConfirmedShippingCard.tsx

import TextState from "../../../components/ui/TextState";

type OrderConfirmedShippingCardProps = {
  lines: string[];
};

export function OrderConfirmedShippingCard({
  lines,
}: OrderConfirmedShippingCardProps) {
  return (
    <section className="order-confirmed-page__content-section">
      <h2 className="order-confirmed-page__section-title">
        配送先情報
      </h2>

      {lines.length > 0 ? (
        <div className="order-confirmed-page__shipping-address">
          {lines.map((line, index) => (
            <p
              key={`${line}-${index}`}
              className="order-confirmed-page__shipping-address-line"
            >
              {line}
            </p>
          ))}
        </div>
      ) : (
        <TextState variant="empty">
          配送先情報を取得できませんでした。
        </TextState>
      )}
    </section>
  );
}