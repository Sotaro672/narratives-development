// frontend/mall/src/features/cart/presentation/components/CartContent.tsx

import Button from "../../../../components/ui/Button";
import type { CartDisplayItem } from "../../../shared/types/cart";
import CartItemCard from "./CartItemCard";
import CartSummary from "./CartSummary";

type CartContentProps = {
  items: CartDisplayItem[];
  totalAmount: number;
  removingItemKey: string;
  isPurchaseDisabled: boolean;
  onRemoveItem: (item: CartDisplayItem) => void | Promise<void>;
  onOpenItem: (path: string) => void;
  onPurchase: () => void;
};

export default function CartContent({
  items,
  totalAmount,
  removingItemKey,
  isPurchaseDisabled,
  onRemoveItem,
  onOpenItem,
  onPurchase,
}: CartContentProps) {
  const removalDisabled = removingItemKey !== "";

  return (
    <div className="cart-page-content">
      <div className="cart-page-list">
        {items.map((item) => {
          const isRemoving = removingItemKey === item.itemKey;

          return (
            <CartItemCard
              key={item.itemKey}
              item={item}
              removing={isRemoving}
              removalDisabled={removalDisabled}
              onRemove={onRemoveItem}
              onOpen={onOpenItem}
            />
          );
        })}
      </div>

      <div className="cart-page-summary-column">
        <CartSummary itemCount={items.length} totalAmount={totalAmount} />

        <Button
          type="button"
          size="lg"
          className="cart-page-purchase-button"
          disabled={isPurchaseDisabled}
          onClick={onPurchase}
        >
          購入する
        </Button>
      </div>
    </div>
  );
}