// frontend/amol/src/features/order-confirmed/hooks/useOrderConfirmedPage.ts

import { useMemo } from "react";
import {
  useLocation,
  useNavigate,
} from "react-router-dom";

import type {
  OrderConfirmedLocationState,
  OrderConfirmedViewModel,
} from "../../shared/types/orderConfirmed";
import {
  formatPaymentStatus,
  getShippingAddressLines,
} from "../utils/format";
import {
  toOrderConfirmedItemViewModels,
} from "../utils/item";

export function useOrderConfirmedPage(): OrderConfirmedViewModel & {
  handleGoToWallet: () => void;
  handleGoToLists: () => void;
} {
  const navigate = useNavigate();
  const location = useLocation();

  const state =
    (location.state ?? {}) as OrderConfirmedLocationState;

  const payment =
    state.payment ?? null;

  const cartItems =
    Array.isArray(state.cartItems)
      ? state.cartItems
      : [];

  const shippingAddress =
    state.shippingAddress ?? null;

  const orderId =
    state.orderId ?? "";

  const amount =
    payment?.amount ?? 0;

  const status =
    payment?.status ?? "SUCCEEDED";

  const items = useMemo(
    () =>
      toOrderConfirmedItemViewModels(
        cartItems,
      ),
    [cartItems],
  );

  const shippingAddressLines = useMemo(
    () =>
      getShippingAddressLines(
        shippingAddress,
      ),
    [shippingAddress],
  );

  const statusLabel = useMemo(
    () =>
      formatPaymentStatus(
        status,
      ),
    [status],
  );

  const handleGoToWallet = () => {
    navigate("/wallet");
  };

  const handleGoToLists = () => {
    navigate("/lists");
  };

  return {
    orderId,
    amount,
    statusLabel,
    items,
    shippingAddressLines,
    handleGoToWallet,
    handleGoToLists,
  };
}