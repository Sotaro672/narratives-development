// frontend/mall/src/features/order-confirmed/hooks/useOrderConfirmedPage.ts

import { useMemo } from "react";
import { useLocation } from "react-router-dom";

import type {
  OrderConfirmedLocationState,
  OrderConfirmedViewModel,
} from "../../shared/types/orderConfirmed";
import { getShippingAddressLines } from "../utils/format";
import { toOrderConfirmedItemViewModels } from "../utils/item";

function normalizeAmount(value: number | undefined): number {
  return Number.isSafeInteger(value) && (value ?? 0) >= 0
    ? value ?? 0
    : 0;
}

export function useOrderConfirmedPage(): OrderConfirmedViewModel {
  const location = useLocation();

  const state = (location.state ?? {}) as OrderConfirmedLocationState;

  const cartItems = Array.isArray(state.cartItems)
    ? state.cartItems
    : [];

  const shippingAddress =
    state.shippingAddress ?? null;

  const orderId = state.orderId ?? "";

  const amount = normalizeAmount(state.amount);
  const subtotalAmount = normalizeAmount(
    state.subtotalAmount,
  );
  const shippingAmount = normalizeAmount(
    state.shippingAmount,
  );
  const taxAmount = normalizeAmount(state.taxAmount);

  const items = useMemo(
    () => toOrderConfirmedItemViewModels(cartItems),
    [cartItems],
  );

  const shippingAddressLines = useMemo(
    () => getShippingAddressLines(shippingAddress),
    [shippingAddress],
  );

  const statusLabel = "発送時に決済";

  return {
    orderId,
    amount,
    subtotalAmount,
    shippingAmount,
    taxAmount,
    statusLabel,
    items,
    shippingAddressLines,
  };
}