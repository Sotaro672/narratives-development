// frontend/amol/src/features/cart/presentation/components/CartItemMeta.tsx

import type {
  CartDisplayItem,
} from "../../types/cart";

import {
  getModelVariation,
} from "../../utils/cartUtils";

import {
  formatAlcoholVolume,
} from "../utils/cartItemDisplay";

type CartItemMetaProps = {
  item: CartDisplayItem;
};

export default function CartItemMeta({
  item,
}: CartItemMetaProps) {
  const model =
    getModelVariation(
      item.catalog,
      item.modelId ?? "",
    );

  const modelKind =
    item.modelKind ??
    model?.kind ??
    "unknown";

  const isAlcohol =
    modelKind === "alcohol";

  const modelNumber =
    model?.modelNumber?.trim() ||
    item.modelNumber?.trim() ||
    "-";

  const colorName =
    model?.colorName?.trim() ||
    item.colorName?.trim() ||
    item.color?.trim() ||
    "-";

  const size =
    model?.size?.trim() ||
    item.size?.trim() ||
    "-";

  return (
    <dl className="cart-page-item__meta">
      {isAlcohol ? (
        <>
          <div>
            <dt>品番</dt>

            <dd>
              {modelNumber}
            </dd>
          </div>

          <div>
            <dt>容量</dt>

            <dd>
              {formatAlcoholVolume(
                item,
                model,
              )}
            </dd>
          </div>
        </>
      ) : (
        <>
          <div>
            <dt>カラー</dt>

            <dd>
              {colorName}
            </dd>
          </div>

          <div>
            <dt>サイズ</dt>

            <dd>
              {size}
            </dd>
          </div>
        </>
      )}

      <div>
        <dt>数量</dt>

        <dd>
          {item.qty}
        </dd>
      </div>
    </dl>
  );
}