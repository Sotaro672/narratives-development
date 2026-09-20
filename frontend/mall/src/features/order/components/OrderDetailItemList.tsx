// frontend/mall/src/features/order/components/OrderDetailItemList.tsx

import Card from "../../../components/ui/Card";
import SectionHeader from "../../../components/ui/SectionHeader";
import type { OrderDetail } from "../../shared/types/orderDetailTypes";
import OrderDetailItem from "./OrderDetailItem";

type OrderDetailItemListProps = {
  order: OrderDetail;
  cancellingItemIndex: number | null;
  returningItemIndex: number | null;
  tradeNavigatingIndex: number | null;
  onCancelItem: (itemIndex: number) => void | Promise<void>;
  onReturnItem: (itemIndex: number) => void;
  onOpenTrade: (orderId: string, itemIndex: number) => void | Promise<void>;
  onOpenBrand: (brandId?: string) => void;
};

export default function OrderDetailItemList({
  order,
  cancellingItemIndex,
  returningItemIndex,
  tradeNavigatingIndex,
  onCancelItem,
  onReturnItem,
  onOpenTrade,
  onOpenBrand,
}: OrderDetailItemListProps) {
  return (
    <Card as="section">
      <SectionHeader title="商品" titleAs="h2" />

      <ul className="order-detail-page__items">
        {order.items.map((item, index) => (
          <OrderDetailItem
            key={`${order.id}-${item.inventoryId}-${item.modelId}-${index}`}
            orderId={order.id}
            item={item}
            index={index}
            cancellingItemIndex={cancellingItemIndex}
            returningItemIndex={returningItemIndex}
            tradeNavigatingIndex={tradeNavigatingIndex}
            onCancelItem={onCancelItem}
            onReturnItem={onReturnItem}
            onOpenTrade={onOpenTrade}
            onOpenBrand={onOpenBrand}
          />
        ))}
      </ul>
    </Card>
  );
}