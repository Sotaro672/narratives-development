//frontend\amol\src\features\order-confirmed\components\OrderConfirmedItemsCard.tsx
import { formatYen } from "../../cart/utils/cartUtils";
import type { OrderConfirmedItemViewModel } from "../types";

type OrderConfirmedItemsCardProps = {
  items: OrderConfirmedItemViewModel[];
};

export function OrderConfirmedItemsCard({
  items,
}: OrderConfirmedItemsCardProps) {
  return (
    <section className="order-confirmed-page__card">
      <h2 className="order-confirmed-page__card-title">注文内容</h2>

      {items.length > 0 ? (
        <ul className="order-confirmed-page__items">
          {items.map((item) => {
            return (
              <li className="order-confirmed-page__item" key={item.itemKey}>
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
                  {item.lineAmount === null
                    ? "価格未設定"
                    : formatYen(item.lineAmount)}
                </p>
              </li>
            );
          })}
        </ul>
      ) : (
        <p className="order-confirmed-page__empty">
          注文内容を取得できませんでした。
        </p>
      )}
    </section>
  );
}