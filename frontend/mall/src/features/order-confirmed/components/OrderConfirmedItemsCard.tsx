// frontend/mall/src/features/order-confirmed/components/OrderConfirmedItemsCard.tsx

import Card from "../../../components/ui/Card";
import TextState from "../../../components/ui/TextState";
import { formatPrice } from "../../../components/utils/price";
import type {
  OrderConfirmedItemViewModel,
} from "../../shared/types/orderConfirmed";

type OrderConfirmedItemsCardProps = {
  items: OrderConfirmedItemViewModel[];
};

export function OrderConfirmedItemsCard({
  items,
}: OrderConfirmedItemsCardProps) {
  return (
    <Card as="section" variant="panel">
      <h2 className="order-confirmed-page__card-title">
        注文内容
      </h2>

      {items.length > 0 ? (
        <ul className="order-confirmed-page__items">
          {items.map((item) => (
            <li
              key={item.itemKey}
              className="order-confirmed-page__item"
            >
              <div>
                <p className="order-confirmed-page__item-title">
                  {item.title}
                </p>

                {item.modelLabel ? (
                  <p className="order-confirmed-page__item-meta">
                    {item.modelLabel}
                  </p>
                ) : null}

                <p className="order-confirmed-page__item-meta">
                  数量: {item.qty}
                </p>
              </div>

              <p className="order-confirmed-page__item-price">
                {formatPrice(item.lineAmount)}
              </p>
            </li>
          ))}
        </ul>
      ) : (
        <TextState variant="empty">
          注文内容を取得できませんでした。
        </TextState>
      )}
    </Card>
  );
}