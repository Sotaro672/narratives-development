// frontend/console/shell/src/features/order/presentation/components/orderItemList.tsx

import Text from "../../../../shared/ui/text";
import type { OrderDetailItemDTO } from "../hooks/useOrderDetail";
import OrderItemCard from "./orderItemCard";

export type OrderItemListProps = {
  items: OrderDetailItemDTO[];
};

export default function OrderItemList({
  items,
}: OrderItemListProps) {
  return (
    <div>
      <Text
        as="div"
        weight="semibold"
        className="order-detail__section-title"
      >
        アイテム
      </Text>

      {items.length === 0 ? (
        <Text
          as="div"
          tone="muted"
          className="order-detail__message"
        >
          アイテムがありません。
        </Text>
      ) : (
        <div className="order-detail__items">
          {items.map((item, index) => (
            <OrderItemCard
              key={`${item.modelNumber ?? "item"}-${index}`}
              item={item}
              index={index}
            />
          ))}
        </div>
      )}
    </div>
  );
}