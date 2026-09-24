// frontend/amol/src/features/cart/presentation/components/CartPageLoading.tsx

import StatePanel from "../../../../components/ui/StatePanel";

export default function CartPageLoading() {
  return (
    <StatePanel
      variant="loading"
      icon="🛒"
      title="カートを読み込んでいます"
      description="追加済みのアイテムを確認しています。"
      className="cart-page-empty"
    />
  );
}