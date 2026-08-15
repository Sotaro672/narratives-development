// frontend/amol/src/features/cart/application/loadCartPage.ts

import { fetchCart, fetchCatalog } from "../api/cartApi";
import type { CartDisplayItem } from "../../shared/types/cart";

export type LoadCartPageResult = {
  items: CartDisplayItem[];
};

async function attachCatalog(item: CartDisplayItem): Promise<CartDisplayItem> {
  const listId = item.listId?.trim() ?? "";

  if (item.type === "resale" || !listId) {
    return {
      ...item,
      catalog: null,
    };
  }

  try {
    const catalog = await fetchCatalog(listId);

    return {
      ...item,
      catalog,
    };
  } catch {
    return {
      ...item,
      catalog: null,
    };
  }
}

export async function loadCartPage(): Promise<LoadCartPageResult> {
  const cart = await fetchCart();

  const baseItems: CartDisplayItem[] = Object.entries(cart.items).map(
    ([itemKey, item]) => ({
      ...item,
      itemKey,
      avatarId: cart.avatarId,
      catalog: null,
    }),
  );

  if (baseItems.length === 0) {
    return {
      items: [],
    };
  }

  const items = await Promise.all(baseItems.map(attachCatalog));

  return {
    items,
  };
}