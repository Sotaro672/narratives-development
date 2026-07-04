// frontend/amol/src/features/payment/utils/guards.ts
import type { CartDisplayItem } from "../../cart/types";
import type { ShippingAddress } from "../../shipping-address/types";
import type {
  CanonicalCartDisplayItem,
  CanonicalShippingAddress,
  CreatedPayment,
} from "../types";

export function isPaymentSucceeded(payment: CreatedPayment): boolean {
  const normalizedStatus = payment.status?.trim().toLowerCase();

  return normalizedStatus === "succeeded";
}

export function isPaymentRequiresAction(payment: CreatedPayment): boolean {
  const normalizedStatus = payment.status?.trim().toLowerCase();

  return (
    payment.requiresAction === true ||
    normalizedStatus === "requires_action" ||
    normalizedStatus === "requires_source_action"
  );
}

export function normalizeCartItems(
  items: CartDisplayItem[],
): CanonicalCartDisplayItem[] {
  return items.map((item) => normalizeCartItem(item));
}

export function normalizeShippingAddress(
  address: ShippingAddress | null,
): CanonicalShippingAddress | null {
  if (!address) {
    return null;
  }

  return address as CanonicalShippingAddress;
}

function normalizeCartItem(item: CartDisplayItem): CanonicalCartDisplayItem {
  const raw = item as CanonicalCartDisplayItem;

  const type =
    raw.type === "resale" || raw.resaleId || raw.productId ? "resale" : "list";

  if (type === "resale") {
    return {
      ...raw,
      type: "resale",
      qty: 1,
    };
  }

  return {
    ...raw,
    type: "list",
    qty: normalizeQty(raw.qty),
  };
}

function normalizeQty(qty: number | undefined): number {
  if (typeof qty !== "number" || !Number.isFinite(qty) || qty <= 0) {
    return 1;
  }

  return qty;
}