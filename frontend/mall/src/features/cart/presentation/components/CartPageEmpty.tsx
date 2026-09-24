// frontend/amol/src/features/cart/presentation/components/CartPageEmpty.tsx

import StatePanel from "../../../../components/ui/StatePanel";

export default function CartPageEmpty() {
  return (
    <StatePanel
      variant="empty"
      icon="🛒"
      title="カートは空です"
      description="応援したいリストやアイテムを追加すると、ここに表示されます。"
      className="cart-page-empty"
    />
  );
}