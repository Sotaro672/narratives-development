// frontend/amol/src/features/payment/components/PaymentItemsCard.tsx

import { formatPrice } from "../../../components/utils/price";
import { getModelPrice, getModelVariation } from "../../cart/utils/cartUtils";
import type { CartDisplayItem } from "../../shared/types/cart";

type PaymentItemsCardProps = {
  amount: number;
  cartItems: CartDisplayItem[];
};

function getItemTitle(item: CartDisplayItem): string {
  const catalog = item.catalog;

  return (
    item.productName ||
    item.title ||
    catalog?.productBlueprint.productName ||
    catalog?.list.title ||
    "商品名未設定"
  );
}

function getItemPrice(item: CartDisplayItem): number | null {
  if (typeof item.price === "number") {
    return item.price;
  }

  return getModelPrice(item.catalog, item.modelId ?? "");
}

function getAlcoholModelLabel(item: CartDisplayItem): string {
  if (item.modelLabel) {
    return item.modelLabel;
  }

  const volumeLabel =
    typeof item.volumeValue === "number" && item.volumeUnit
      ? `${item.volumeValue}${item.volumeUnit}`
      : "";

  return [item.modelNumber, volumeLabel].filter(Boolean).join(" / ");
}

function getApparelModelLabel(item: CartDisplayItem): string {
  const model = getModelVariation(item.catalog, item.modelId ?? "");
  const colorName = item.colorName ?? model?.colorName ?? "";
  const size = item.size ?? model?.size ?? "";

  return [
    colorName ? `カラー: ${colorName}` : "",
    size ? `サイズ: ${size}` : "",
  ]
    .filter(Boolean)
    .join(" / ");
}

function getItemModelLabel(item: CartDisplayItem): string {
  if (item.modelKind === "alcohol") {
    return getAlcoholModelLabel(item);
  }

  return getApparelModelLabel(item);
}

export function PaymentItemsCard({
  amount,
  cartItems,
}: PaymentItemsCardProps) {
  return (
    <section className="payment-page__card">
      <h2 className="payment-page__section-title">注文内容</h2>

      {cartItems.length > 0 ? (
        <ul className="payment-page__items">
          {cartItems.map((item) => {
            const price = getItemPrice(item);
            const lineAmount = price === null ? null : price * item.qty;
            const title = getItemTitle(item);
            const modelLabel = getItemModelLabel(item);

            return (
              <li className="payment-page__item" key={item.itemKey}>
                <div>
                  <p className="payment-page__item-title">{title}</p>

                  {modelLabel ? (
                    <p className="payment-page__item-meta">{modelLabel}</p>
                  ) : null}

                  <p className="payment-page__item-meta">
                    数量: {item.qty}
                  </p>
                </div>

                <p className="payment-page__item-price">
                  {formatPrice(lineAmount)}
                </p>
              </li>
            );
          })}
        </ul>
      ) : (
        <p className="payment-page__empty">
          カート情報がありません。
        </p>
      )}

      <div className="payment-page__total">
        <span>合計</span>
        <strong>{formatPrice(amount)}</strong>
      </div>
    </section>
  );
}