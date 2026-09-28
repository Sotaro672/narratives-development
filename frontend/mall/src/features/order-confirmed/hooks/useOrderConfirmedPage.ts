// frontend/amol/src/features/order-confirmed/hooks/useOrderConfirmedPage.ts

import { useMemo } from "react";
import { useLocation, useNavigate } from "react-router-dom";

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

export function useOrderConfirmedPage(): OrderConfirmedViewModel & {
  hasResaleItem: boolean;
  handleGoToTrade: () => void;
} {
  const navigate = useNavigate();
  const location = useLocation();

  const state = (location.state ?? {}) as OrderConfirmedLocationState;

  const cartItems = Array.isArray(state.cartItems) ? state.cartItems : [];
  const shippingAddress = state.shippingAddress ?? null;
  const orderId = state.orderId ?? "";

  const amount = normalizeAmount(state.amount);
  const subtotalAmount = normalizeAmount(state.subtotalAmount);
  const shippingAmount = normalizeAmount(state.shippingAmount);
  const taxAmount = normalizeAmount(state.taxAmount);

  const items = useMemo(
    () => toOrderConfirmedItemViewModels(cartItems),
    [cartItems],
  );

  const shippingAddressLines = useMemo(
    () => getShippingAddressLines(shippingAddress),
    [shippingAddress],
  );

  const hasResaleItem = useMemo(
    () => cartItems.some((item) => item.type === "resale"),
    [cartItems],
  );

  const statusLabel = "発送時に決済";

  const handleGoToTrade = () => {
    const normalizedOrderId = orderId.trim();

    if (!normalizedOrderId || !hasResaleItem) {
      return;
    }

    navigate(
      `/orders/${encodeURIComponent(normalizedOrderId)}/trade`,
    );
  };

  return {
    orderId,
    amount,
    subtotalAmount,
    shippingAmount,
    taxAmount,
    statusLabel,
    items,
    shippingAddressLines,
    hasResaleItem,
    handleGoToTrade,
  };
}