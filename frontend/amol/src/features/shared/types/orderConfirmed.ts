// frontend\amol\src\features\shared\types\orderConfirmed.ts

import type {
  CartDisplayItem,
} from "./cart";
import type {
  CreatedPayment,
} from "./payment";
import type {
  ShippingAddress,
} from "./shippingAddress";

export type OrderConfirmedLocationState = {
  payment?: CreatedPayment;
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
  paymentId: string;
  amount: number;
  statusLabel: string;
  items: OrderConfirmedItemViewModel[];
  shippingAddressLines: string[];
};