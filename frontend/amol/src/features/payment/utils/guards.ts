// frontend/amol/src/features/payment/utils/guards.ts

import type {
  CartDisplayItem,
} from "../../shared/types/cart";
import type {
  ShippingAddress,
} from "../../shared/types/types";
import type {
  CanonicalShippingAddress,
  CreatedPayment,
} from "../../shared/types/payment";

export function isPaymentSucceeded(
  payment: CreatedPayment,
): boolean {
  const normalizedStatus =
    payment.status?.trim().toLowerCase();

  return normalizedStatus === "succeeded";
}

export function isPaymentRequiresAction(
  payment: CreatedPayment,
): boolean {
  const normalizedStatus =
    payment.status?.trim().toLowerCase();

  return (
    payment.requiresAction === true ||
    normalizedStatus === "requires_action" ||
    normalizedStatus === "requires_source_action"
  );
}

export function normalizeCartItems(
  items: CartDisplayItem[],
): CartDisplayItem[] {
  return items.map((item) =>
    normalizeCartItem(item),
  );
}

export function normalizeShippingAddress(
  address: ShippingAddress | null,
): CanonicalShippingAddress | null {
  if (!address) {
    return null;
  }

  return address as CanonicalShippingAddress;
}

function normalizeCartItem(
  item: CartDisplayItem,
): CartDisplayItem {
  const type =
    item.type === "resale" ||
    item.resaleId ||
    item.productId
      ? "resale"
      : "list";

  if (type === "resale") {
    return {
      ...item,
      type: "resale",
      qty: 1,
    };
  }

  return {
    ...item,
    type: "list",
    qty: normalizeQty(item.qty),
  };
}

function normalizeQty(
  qty: number | undefined,
): number {
  if (
    typeof qty !== "number" ||
    !Number.isFinite(qty) ||
    qty <= 0
  ) {
    return 1;
  }

  return qty;
}