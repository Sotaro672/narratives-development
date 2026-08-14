// frontend/amol/src/features/order-confirmed/utils/item.ts

import { getCartItemPrice } from "../../cart/utils/cartUtils";
import type { CartDisplayItem } from "../../shared/types/cart";
import type { OrderConfirmedItemViewModel } from "../../shared/types/orderConfirmed";

function getItemTitle(item: CartDisplayItem): string {
  return item.productName || item.title || "商品名未設定";
}

function getAlcoholModelLabel(item: CartDisplayItem): string {
  if (item.modelLabel) {
    return item.modelLabel;
  }

  const volumeLabel =
    item.volumeValue !== undefined && item.volumeUnit
      ? `${item.volumeValue}${item.volumeUnit}`
      : "";

  return [item.modelNumber, volumeLabel].filter(Boolean).join(" / ");
}

function getApparelModelLabel(item: CartDisplayItem): string {
  return [
    item.color ? `カラー: ${item.color}` : "",
    item.size ? `サイズ: ${item.size}` : "",
  ]
    .filter(Boolean)
    .join(" / ");
}

function getItemModelLabel(item: CartDisplayItem): string {
  return item.modelKind === "alcohol"
    ? getAlcoholModelLabel(item)
    : getApparelModelLabel(item);
}

export function toOrderConfirmedItemViewModel(
  item: CartDisplayItem,
): OrderConfirmedItemViewModel {
  const price = getCartItemPrice(item);
  const lineAmount = price === null ? null : price * item.qty;

  return {
    itemKey: item.itemKey,
    title: getItemTitle(item),
    modelLabel: getItemModelLabel(item),
    qty: item.qty,
    lineAmount,
  };
}

export function toOrderConfirmedItemViewModels(
  items: CartDisplayItem[],
): OrderConfirmedItemViewModel[] {
  return items.map(toOrderConfirmedItemViewModel);
}