import { useMemo } from "react";
import { useLocation, useNavigate } from "react-router-dom";

import type {
  OrderConfirmedLocationState,
  OrderConfirmedViewModel,
} from "../types";
import { formatPaymentStatus, getShippingAddressLines } from "../utils/format";
import { toOrderConfirmedItemViewModels } from "../utils/item";

export function useOrderConfirmedPage(): OrderConfirmedViewModel & {
  handleGoToWallet: () => void;
  handleGoToLists: () => void;
} {
  const navigate = useNavigate();
  const location = useLocation();

  const state = (location.state ?? {}) as OrderConfirmedLocationState;

  const payment = state.payment ?? null;
  const cartItems = Array.isArray(state.cartItems) ? state.cartItems : [];
  const shippingAddress = state.shippingAddress ?? null;

  const invoiceId = payment?.invoiceId ?? state.invoiceId ?? "";
  const paymentId = payment?.id ?? payment?.paymentId ?? state.paymentId ?? "";
  const amount = payment?.amount ?? state.amount ?? 0;
  const status = payment?.status ?? "SUCCEEDED";

  const items = useMemo(() => {
    return toOrderConfirmedItemViewModels(cartItems);
  }, [cartItems]);

  const shippingAddressLines = useMemo(() => {
    return getShippingAddressLines(shippingAddress);
  }, [shippingAddress]);

  const statusLabel = useMemo(() => {
    return formatPaymentStatus(status);
  }, [status]);

  const handleGoToWallet = () => {
    navigate("/wallet");
  };

  const handleGoToLists = () => {
    navigate("/lists");
  };

  return {
    invoiceId,
    paymentId,
    amount,
    statusLabel,
    items,
    shippingAddressLines,
    handleGoToWallet,
    handleGoToLists,
  };
}