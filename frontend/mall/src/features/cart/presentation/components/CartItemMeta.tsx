// frontend/amol/src/features/cart/presentation/components/CartItemMeta.tsx

import type { CartDisplayItem } from "../../../shared/types/cart";
import { formatAlcoholVolume } from "../../utils/cartUtils";

type CartItemMetaProps = {
  item: CartDisplayItem;
};

export default function CartItemMeta({
  item,
}: CartItemMetaProps) {
  const isAlcohol = item.modelKind === "alcohol";
  const modelNumber = item.modelNumber || "-";
  const color = item.color || "-";
  const size = item.size || "-";

  return (
    <dl className="cart-page-item__meta">
      {isAlcohol ? (
        <>
          <div>
            <dt>品番</dt>
            <dd>{modelNumber}</dd>
          </div>

          <div>
            <dt>容量</dt>
            <dd>{formatAlcoholVolume(item)}</dd>
          </div>
        </>
      ) : (
        <>
          <div>
            <dt>カラー</dt>
            <dd>{color}</dd>
          </div>

          <div>
            <dt>サイズ</dt>
            <dd>{size}</dd>
          </div>
        </>
      )}

      <div>
        <dt>数量</dt>
        <dd>{item.qty}</dd>
      </div>
    </dl>
  );
}