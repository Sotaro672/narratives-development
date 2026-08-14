// frontend/amol/src/features/cart/mappers/cartMapper.ts

import type { CartDisplayItem, CartDTO } from "../types/cart";

export function cartDTOToDisplayItems(cart: CartDTO): CartDisplayItem[] {
  return Object.entries(cart.items).map(([itemKey, item]) => ({
    ...item,
    itemKey,
    avatarId: cart.avatarId,
    catalog: null,
  }));
}