// frontend/amol/src/features/cart/utils/cartUtils.ts

import type { CartDisplayItem } from "../../shared/types/cart";

export function getCartItemPrice(item: CartDisplayItem): number | null {
  return item.price ?? null;
}

export function calculateCartTotalAmount(items: CartDisplayItem[]): number {
  return items.reduce((total, item) => {
    if (item.price === undefined) {
      return total;
    }

    return total + item.price * item.qty;
  }, 0);
}

export function formatAlcoholVolume(item: CartDisplayItem): string {
  if (item.volumeValue !== undefined && item.volumeUnit) {
    return `${item.volumeValue}${item.volumeUnit}`;
  }

  return item.modelLabel || "-";
}

export function getCartItemBrandName(item: CartDisplayItem): string {
  return item.brandName || "ブランド未設定";
}

export function getCartItemProductName(item: CartDisplayItem): string {
  return item.productName || "商品名未設定";
}

export function getCartItemListTitle(item: CartDisplayItem): string {
  if (item.title && item.title !== item.productName) {
    return item.title;
  }

  return "";
}

export function getCartItemImageUrl(item: CartDisplayItem): string {
  return item.imageUrl || item.listImage || "";
}

export function getCartItemNavigationPath(item: CartDisplayItem): string {
  if (item.listId) {
    return `/lists/${encodeURIComponent(item.listId)}`;
  }

  if (item.resaleId) {
    return `/market/${encodeURIComponent(item.resaleId)}`;
  }

  return "";
}

export function formatYen(amount: number): string {
  return new Intl.NumberFormat("ja-JP", {
    style: "currency",
    currency: "JPY",
    maximumFractionDigits: 0,
  }).format(amount);
}

export function formatPrice(amount: number): string {
  return formatYen(amount);
}