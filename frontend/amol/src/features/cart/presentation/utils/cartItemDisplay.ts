// frontend/amol/src/features/cart/presentation/utils/cartItemDisplay.ts

import type { CartDisplayItem } from "../../types/cart";

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