//frontend\amol\src\features\order-confirmed\types.ts
import type { CartDisplayItem } from "../cart/types";
import type { ShippingAddress } from "../shipping-address/types";

export type ConfirmedPayment = {
  id?: string;
  paymentId?: string;
  invoiceId?: string;
  paymentMethodId?: string;
  amount?: number;
  status?: string;
  createdAt?: string;
};

export type OrderConfirmedLocationState = {
  payment?: ConfirmedPayment;
  invoiceId?: string;
  paymentId?: string;
  paymentMethodId?: string;
  amount?: number;
  cartItems?: CartDisplayItem[];
  shippingAddress?: ShippingAddress | null;
};

export type OrderConfirmedItemViewModel = {
  itemKey: string;
  title: string;
  modelLabel: string;
  qty: number;
  lineAmount: number | null;
};

export type OrderConfirmedViewModel = {
  invoiceId: string;
  paymentId: string;
  amount: number;
  statusLabel: string;
  items: OrderConfirmedItemViewModel[];
  shippingAddressLines: string[];
};