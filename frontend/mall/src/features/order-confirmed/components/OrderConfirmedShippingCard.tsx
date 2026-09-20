// frontend/mall/src/features/order-confirmed/components/OrderConfirmedShippingCard.tsx

import Card from "../../../components/ui/Card";
import TextState from "../../../components/ui/TextState";

type OrderConfirmedShippingCardProps = {
  lines: string[];
};

export function OrderConfirmedShippingCard({
  lines,
}: OrderConfirmedShippingCardProps) {
  return (
    <Card as="section" variant="panel">
      <h2 className="order-confirmed-page__card-title">
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
    </Card>
  );
}