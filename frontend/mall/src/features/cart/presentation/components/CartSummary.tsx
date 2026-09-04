// frontend/amol/src/features/cart/presentation/components/CartSummary.tsx

import {
  formatPrice,
} from "../../../../components/utils/price";

type CartSummaryProps = {
  itemCount: number;
  totalAmount: number;
};

export default function CartSummary({
  itemCount,
  totalAmount,
}: CartSummaryProps) {
  const normalizedItemCount =
    Number.isFinite(itemCount) &&
    itemCount > 0
      ? Math.floor(itemCount)
      : 0;

  const normalizedTotalAmount =
    Number.isFinite(totalAmount) &&
    totalAmount >= 0
      ? totalAmount
      : 0;

  return (
    <aside
      className="cart-page-summary"
      aria-label="注文内容"
    >
      <h2 className="cart-page-summary__title">
        注文内容
      </h2>

      <dl className="cart-page-summary__list">
        <div>
          <dt>商品数</dt>

          <dd>
            {normalizedItemCount}
          </dd>
        </div>

        <div>
          <dt>合計</dt>

          <dd>
            {formatPrice(
              normalizedTotalAmount,
            )}
          </dd>
        </div>
      </dl>
    </aside>
  );
}