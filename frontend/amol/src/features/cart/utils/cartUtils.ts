// frontend/amol/src/features/cart/utils/cartUtils.ts

import type { CartDisplayItem } from "../../shared/types/cart";

export function getCartItemPrice(
  item: CartDisplayItem,
): number | null {
  return item.price ?? null;
}

export function calculateCartTotalAmount(
  items: CartDisplayItem[],
): number {
  return items.reduce((total, item) => {
    if (item.price === undefined) {
      return total;
    }

    return total + item.price * item.qty;
  }, 0);
}

export function formatYen(
  amount: number,
): string {
  return new Intl.NumberFormat("ja-JP", {
    style: "currency",
    currency: "JPY",
    maximumFractionDigits: 0,
  }).format(amount);
}

export function formatPrice(
  amount: number,
): string {
  return formatYen(amount);
}