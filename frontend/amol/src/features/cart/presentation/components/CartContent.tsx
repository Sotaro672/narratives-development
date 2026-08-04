// frontend/amol/src/features/cart/presentation/components/CartContent.tsx

import type {
  CartDisplayItem,
} from "../../types/cart";

import CartItemCard from "./CartItemCard";
import CartSummary from "./CartSummary";

type CartContentProps = {
  items: CartDisplayItem[];
  totalAmount: number;
  removingItemKey: string;

  onRemoveItem: (
    item: CartDisplayItem,
  ) => void | Promise<void>;

  onOpenItem: (
    path: string,
  ) => void;
};

export default function CartContent({
  items,
  totalAmount,
  removingItemKey,
  onRemoveItem,
  onOpenItem,
}: CartContentProps) {
  const removalDisabled =
    removingItemKey !== "";

  return (
    <div className="cart-page-content">
      <div className="cart-page-list">
        {items.map((item) => {
          const isRemoving =
            removingItemKey ===
            item.itemKey;

          return (
            <CartItemCard
              key={item.itemKey}
              item={item}
              removing={
                isRemoving
              }
              removalDisabled={
                removalDisabled
              }
              onRemove={
                onRemoveItem
              }
              onOpen={
                onOpenItem
              }
            />
          );
        })}
      </div>

      <CartSummary
        itemCount={items.length}
        totalAmount={totalAmount}
      />
    </div>
  );
}