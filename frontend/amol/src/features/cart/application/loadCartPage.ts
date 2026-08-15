// frontend/amol/src/features/cart/application/loadCartPage.ts

import { fetchCart } from "../api/cartApi";
import type { CartDisplayItem } from "../../shared/types/cart";

export type LoadCartPageResult = {
  items: CartDisplayItem[];
};

export async function loadCartPage(): Promise<LoadCartPageResult> {
  const cart = await fetchCart();

  const items: CartDisplayItem[] = Object.entries(cart.items).map(([itemKey, item]) => ({
    ...item,
    itemKey,
    avatarId: cart.avatarId,
  }));

  return { items };
}